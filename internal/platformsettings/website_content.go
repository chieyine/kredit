package platformsettings

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

const WebsitePrefix = "website."

type WebsiteSection struct {
	Heading string   `json:"heading"`
	Body    string   `json:"body"`
	Points  []string `json:"points,omitempty"`
}
type WebsiteCopy struct {
	Guide        *WebsiteGuide    `json:"guide,omitempty"`
	Contact      *WebsiteContact  `json:"contact,omitempty"`
	Title        string           `json:"title"`
	Accent       string           `json:"accent"`
	Introduction string           `json:"introduction"`
	Sections     []WebsiteSection `json:"sections"`
}
type WebsitePublication struct {
	DocumentVersion string      `json:"document_version,omitempty"`
	EffectiveDate   string      `json:"effective_date,omitempty"`
	Copy            WebsiteCopy `json:"copy"`
	Version         int         `json:"version"`
	PublishedAt     string      `json:"published_at"`
	ContentHash     string      `json:"content_hash"`
}
type WebsiteContent struct {
	Draft     WebsiteCopy         `json:"draft"`
	Published *WebsitePublication `json:"published,omitempty"`
}

// These are the exact slots consumed by the public routes. Financial rates
// remain in business policy; editorial text cannot change an accepted price.
var WebsitePages = map[string]string{"home": "Homepage introduction", "faq": "Common questions", "pricing": "Pricing introduction", "terms": "Terms of service", "privacy": "Privacy notice", "complaints": "Complaints policy"}

func init() {
	for page, label := range WebsitePages {
		KnownSettings[WebsitePrefix+page] = SettingMeta{Category: "website", Description: label, Validate: validateWebsiteContent}
	}
}
func validateWebsiteText(value string, maximum int, required bool) bool {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > maximum || (required && strings.TrimSpace(value) == "") {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return false
		}
	}
	return true
}
func ValidateWebsiteCopy(copy WebsiteCopy) error {
	if !validateWebsiteText(copy.Title, 160, true) || !validateWebsiteText(copy.Accent, 160, false) || !validateWebsiteText(copy.Introduction, 2000, true) || len(copy.Sections) > 40 {
		return errors.New("enter a title, introduction and at most 40 sections within the displayed limits")
	}
	if err := validateEditorialExtras(copy); err != nil {
		return err
	}
	for _, section := range copy.Sections {
		if !validateWebsiteText(section.Heading, 200, true) || !validateWebsiteText(section.Body, 6000, true) {
			return errors.New("each section needs a heading and text within the displayed limits")
		}
	}
	return nil
}
func validateWebsiteContent(raw json.RawMessage) error {
	if len(raw) > 300000 {
		return errors.New("website content is too large")
	}
	var content WebsiteContent
	if err := json.Unmarshal(raw, &content); err != nil {
		return errors.New("invalid website content")
	}
	if err := ValidateWebsiteCopy(content.Draft); err != nil {
		return err
	}
	if content.Published != nil {
		return ValidateWebsiteCopy(content.Published.Copy)
	}
	return nil
}

func IsLegalPage(page string) bool {
	return page == "terms" || page == "privacy" || page == "complaints"
}
func LegalDocumentVersion(page string, version int) string {
	return fmt.Sprintf("legal-%s-v%d", page, version)
}

type WebsiteArchive interface {
	PublishedWebsiteVersion(context.Context, string, int) (WebsitePublication, error)
}

func (s *PostgresStore) PublishedWebsiteVersion(ctx context.Context, page string, version int) (WebsitePublication, error) {
	if s == nil || s.pool == nil {
		return WebsitePublication{}, errors.New("website archive unavailable")
	}
	if !IsLegalPage(page) || version < 1 {
		return WebsitePublication{}, pgx.ErrNoRows
	}
	var raw []byte
	if err := s.pool.QueryRow(ctx, `SELECT new_value FROM app.platform_settings_history WHERE key=$1 AND version=$2`, WebsitePrefix+page, version).Scan(&raw); err != nil {
		return WebsitePublication{}, err
	}
	var content WebsiteContent
	if err := json.Unmarshal(raw, &content); err != nil {
		return WebsitePublication{}, err
	}
	if content.Published == nil || content.Published.Version != version || content.Published.DocumentVersion != LegalDocumentVersion(page, version) {
		return WebsitePublication{}, pgx.ErrNoRows
	}
	if err := ValidateWebsitePublication(page, *content.Published); err != nil {
		return WebsitePublication{}, err
	}
	return *content.Published, nil
}

// ValidateWebsitePublication checks the immutable public copy before it is used
// as either a consent reference or a historical document.
func ValidateWebsitePublication(page string, publication WebsitePublication) error {
	if err := ValidateWebsitePageCopy(page, publication.Copy); err != nil {
		return err
	}
	raw, err := json.Marshal(publication.Copy)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(raw)
	if publication.Version < 1 || publication.ContentHash != hex.EncodeToString(digest[:]) {
		return errors.New("published content integrity check failed")
	}
	if _, err := time.Parse(time.RFC3339, publication.PublishedAt); err != nil {
		return errors.New("invalid publication time")
	}
	if IsLegalPage(page) {
		if publication.DocumentVersion != LegalDocumentVersion(page, publication.Version) || len(publication.Copy.Sections) == 0 {
			return errors.New("invalid legal publication")
		}
		if _, err := time.Parse("2006-01-02", publication.EffectiveDate); err != nil {
			return errors.New("invalid effective date")
		}
	}
	return nil
}
