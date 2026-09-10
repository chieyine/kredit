package legalpublication

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"kredit/internal/platformsettings"

	"github.com/jackc/pgx/v5"
)

type publicationSettings struct {
	platformsettings.Service
	content map[string]platformsettings.WebsiteContent
	failure error
}

func (s publicationSettings) Get(_ context.Context, key string, _ bool) (platformsettings.Setting, error) {
	if s.failure != nil {
		return platformsettings.Setting{}, s.failure
	}
	content, ok := s.content[key]
	if !ok {
		return platformsettings.Setting{}, pgx.ErrNoRows
	}
	raw, err := json.Marshal(content)
	return platformsettings.Setting{Value: raw}, err
}
func TestCurrentDocumentsUseVerifiedPublicationOnly(t *testing.T) {
	copy := platformsettings.WebsiteCopy{Title: "Terms", Introduction: "Read the terms", Sections: []platformsettings.WebsiteSection{{Heading: "Scope", Body: "Existing agreements retain their version."}}}
	settings := publicationSettings{content: map[string]platformsettings.WebsiteContent{"website.terms": {Draft: copy}}}
	reader := SettingsReader(settings)
	versions, err := reader()
	if err != nil || versions != Initial() {
		t.Fatalf("draft changed versions: %+v %v", versions, err)
	}
	raw, _ := json.Marshal(copy)
	digest := sha256.Sum256(raw)
	publication := &platformsettings.WebsitePublication{Copy: copy, Version: 2, DocumentVersion: "legal-terms-v2", EffectiveDate: "2026-09-09", PublishedAt: time.Now().UTC().Format(time.RFC3339), ContentHash: hex.EncodeToString(digest[:])}
	settings.content["website.terms"] = platformsettings.WebsiteContent{Draft: copy, Published: publication}
	versions, err = reader()
	if err != nil || versions.Terms != "legal-terms-v2" || versions.Privacy != PrivacyVersion {
		t.Fatalf("publication not selected: %+v %v", versions, err)
	}
	publication.Copy.Title = "Changed without publication"
	if _, err = reader(); err == nil {
		t.Fatal("corrupt publication silently accepted")
	}
	settings.failure = errors.New("database unavailable")
	if _, err = SettingsReader(settings)(); err == nil {
		t.Fatal("outage silently selected initial versions")
	}
}
