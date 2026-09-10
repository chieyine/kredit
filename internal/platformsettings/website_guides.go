package platformsettings

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/mail"
	"net/url"
	"strings"
	"time"
)

//go:embed guide_catalog.json
var guideCatalog []byte

type WebsiteGuide struct {
	Description string           `json:"description"`
	Category    string           `json:"category"`
	Keyphrase   string           `json:"keyphrase"`
	FAQ         []WebsiteSection `json:"faq"`
	Sources     []WebsiteSource  `json:"sources"`
	Related     []string         `json:"related"`
}
type WebsiteSource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Note string `json:"note"`
}
type WebsiteContact struct {
	SupportEmail string `json:"support_email"`
	PrivacyEmail string `json:"privacy_email"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
}

var GuideCategories = []string{"Credit sales", "Customer checks", "Agreements", "Payments", "Late payment", "Cash flow", "Business records", "Safe payments", "Industry guides", "Business growth"}

func init() {
	var catalog map[string]string
	if err := json.Unmarshal(guideCatalog, &catalog); err != nil {
		panic(err)
	}
	for slug, title := range catalog {
		WebsitePages["guide-"+slug] = "Guide: " + title
	}
	WebsitePages["contact"] = "Contact details"
	for page, label := range WebsitePages {
		KnownSettings[WebsitePrefix+page] = SettingMeta{Category: "website", Description: label, Validate: validateWebsiteContent}
	}
}
func IsGuidePage(page string) bool {
	_, ok := WebsitePages[page]
	return ok && strings.HasPrefix(page, "guide-")
}
func ValidateWebsitePageCopy(page string, copy WebsiteCopy) error {
	if err := ValidateWebsiteCopy(copy); err != nil {
		return err
	}
	if IsGuidePage(page) != (copy.Guide != nil) || (page == "contact") != (copy.Contact != nil) {
		return errors.New("the content does not match this page")
	}
	sections := page == "faq" || IsLegalPage(page) || IsGuidePage(page)
	if sections && len(copy.Sections) == 0 {
		return errors.New("add at least one section")
	}
	if !sections && len(copy.Sections) > 0 {
		return errors.New("this page does not support sections")
	}
	if copy.Guide == nil {
		for _, section := range copy.Sections {
			if len(section.Points) > 0 {
				return errors.New("bullet lists are supported only in guides")
			}
		}
	}
	return nil
}
func validateEditorialExtras(copy WebsiteCopy) error {
	for _, section := range copy.Sections {
		if len(section.Points) > 40 {
			return errors.New("use at most 40 points in each section")
		}
		for _, point := range section.Points {
			if !validateWebsiteText(point, 2000, true) {
				return errors.New("invalid guide bullet point")
			}
		}
	}
	if g := copy.Guide; g != nil {
		validCategory := false
		for _, category := range GuideCategories {
			if g.Category == category {
				validCategory = true
			}
		}
		if !validCategory || !validateWebsiteText(g.Description, 2000, true) || !validateWebsiteText(g.Keyphrase, 200, true) || len(g.FAQ) > 40 || len(g.Sources) > 40 || len(g.Related) > 12 {
			return errors.New("check guide description, category, questions and sources")
		}
		for _, q := range g.FAQ {
			if !validateWebsiteText(q.Heading, 200, true) || !validateWebsiteText(q.Body, 6000, true) || len(q.Points) > 0 {
				return errors.New("invalid guide question or answer")
			}
		}
		for _, source := range g.Sources {
			u, err := url.Parse(source.URL)
			if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || len(source.URL) > 2000 || !validateWebsiteText(source.Name, 200, true) || !validateWebsiteText(source.Note, 2000, false) {
				return errors.New("guide sources need an HTTPS address, name and valid note")
			}
		}
		seen := map[string]bool{}
		for _, slug := range g.Related {
			if !IsGuidePage("guide-"+slug) || seen[slug] {
				return errors.New("choose distinct existing related guides")
			}
			seen[slug] = true
		}
	}
	if c := copy.Contact; c != nil {
		for _, email := range []string{c.SupportEmail, c.PrivacyEmail} {
			a, err := mail.ParseAddress(email)
			if err != nil || a.Address != email || len(email) > 254 {
				return errors.New("enter a valid contact email address")
			}
		}
		if !validateWebsiteText(c.Address, 2000, true) || !validateWebsiteText(c.Phone, 40, false) {
			return errors.New("invalid contact address or phone")
		}
		if c.Phone != "" {
			for _, r := range c.Phone {
				if !strings.ContainsRune("+0123456789 ()-", r) {
					return errors.New("invalid contact phone")
				}
			}
		}
	}
	return nil
}

// PublishedGuides reads only public editorial rows in one database snapshot.
// Drafts, owner history and connector credentials never enter the response.
func (s *PostgresStore) PublishedGuides(ctx context.Context) (map[string]WebsitePublication, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `SELECT key,value FROM app.platform_settings WHERE category='website' AND NOT is_secret AND key LIKE 'website.guide-%' ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]WebsitePublication{}
	for rows.Next() {
		var key string
		var raw []byte
		if err = rows.Scan(&key, &raw); err != nil {
			return nil, err
		}
		page := strings.TrimPrefix(key, WebsitePrefix)
		if !IsGuidePage(page) {
			continue
		}
		var content WebsiteContent
		if err = json.Unmarshal(raw, &content); err != nil {
			return nil, err
		}
		if content.Published == nil {
			continue
		}
		if err = ValidateWebsitePublication(page, *content.Published); err != nil {
			return nil, err
		}
		result[strings.TrimPrefix(page, "guide-")] = *content.Published
	}
	return result, rows.Err()
}
