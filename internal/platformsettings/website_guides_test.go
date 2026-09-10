package platformsettings

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestEditorialValidationAndLegacyHash(t *testing.T) {
	legacy := WebsiteCopy{Title: "Title", Accent: "", Introduction: "Introduction", Sections: []WebsiteSection{{Heading: "Heading", Body: "Body"}}}
	raw, _ := json.Marshal(legacy)
	expected := `{"title":"Title","accent":"","introduction":"Introduction","sections":[{"heading":"Heading","body":"Body"}]}`
	if string(raw) != expected {
		t.Fatal("editorial additions changed existing legal copy encoding")
	}
	sum := sha256.Sum256(raw)
	if err := ValidateWebsitePublication("terms", WebsitePublication{Copy: legacy, Version: 1, PublishedAt: "2026-09-09T00:00:00Z", EffectiveDate: "2026-09-09", DocumentVersion: LegalDocumentVersion("terms", 1), ContentHash: hex.EncodeToString(sum[:])}); err != nil {
		t.Fatal(err)
	}
	page := "guide-how-to-sell-goods-on-credit-in-nigeria"
	guide := legacy
	guide.Guide = &WebsiteGuide{Description: "Description", Category: "Credit sales", Keyphrase: "Credit", FAQ: []WebsiteSection{}, Sources: []WebsiteSource{{Name: "Reference", URL: "https://example.com", Note: ""}}, Related: []string{}}
	if err := ValidateWebsitePageCopy(page, guide); err != nil {
		t.Fatal(err)
	}
	guide.Guide.Sources[0].URL = "javascript:alert(1)"
	if err := ValidateWebsitePageCopy(page, guide); err == nil {
		t.Fatal("unsafe source address accepted")
	}
	if err := ValidateWebsitePageCopy(page, legacy); err == nil {
		t.Fatal("guide without metadata accepted")
	}
	contact := WebsiteCopy{Title: "Contact", Introduction: "Get help", Contact: &WebsiteContact{SupportEmail: "help@example.test", PrivacyEmail: "privacy@example.test", Address: "Office"}}
	if err := ValidateWebsitePageCopy("contact", contact); err != nil {
		t.Fatal(err)
	}
	contact.Contact.SupportEmail = "Name <help@example.test>"
	if err := ValidateWebsitePageCopy("contact", contact); err == nil {
		t.Fatal("contact display name accepted as destination")
	}
}
