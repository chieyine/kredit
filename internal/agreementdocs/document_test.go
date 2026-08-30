package agreementdocs

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"kredit/internal/credit"
	"kredit/internal/mandates"
	"kredit/internal/schedules"
	"kredit/internal/tradelines"
)

func TestRenderHTMLIncludesRequiredEvidence(t *testing.T) {
	canonical := []byte(`{"principal_kobo":120000000}`)
	digest := sha256.Sum256(canonical)
	now := time.Date(2026, 8, 29, 9, 0, 0, 0, time.UTC)
	view := credit.View{
		Request:    credit.CreditRequest{ID: "request-1", SupplierLegalName: "ABC Pharmaceuticals Ltd", BuyerLegalName: "Royal Pharmacy Ltd", PrincipalKobo: 120000000, GoodsDescription: "Inventory", DueDate: "2026-09-30", CollectionAt: now, GraceHours: 24, InvoiceDocumentHash: "invoice-sha256"},
		Agreement:  credit.AgreementVersion{Version: 1, CanonicalJSON: canonical, DocumentHash: hex.EncodeToString(digest[:]), TermsVersion: "terms-v1", PrivacyVersion: "privacy-v1"},
		Acceptance: &credit.Acceptance{AcceptanceMethod: "authenticated_web", AuthenticationLevel: "AAL2", AcceptedAt: now},
		Mandate:    &mandates.Mandate{Provider: "approved-provider", Status: mandates.Active, AmountCeiling: 120000000},
		Release:    &credit.GoodsRelease{DeliveryMethod: "supplier_delivery", ReleasedAt: now},
		Obligation: &credit.Obligation{ID: "obligation-1"},
	}
	body, err := RenderHTML(DocumentData{View: view, Items: []schedules.Item{{Sequence: 1, PrincipalDueKobo: 120000000, DueAt: now, CollectionAt: now.Add(24 * time.Hour)}}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, expected := range []string{"ABC Pharmaceuticals Ltd", "Royal Pharmacy Ltd", "Inventory", "approved-provider", "invoice-sha256", hex.EncodeToString(digest[:]), "Print or save as PDF"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("document missing %q", expected)
		}
	}
}

func TestRenderHTMLRejectsTamperedCanonicalAgreement(t *testing.T) {
	view := credit.View{Agreement: credit.AgreementVersion{CanonicalJSON: []byte(`{}`), DocumentHash: "wrong"}, Acceptance: &credit.Acceptance{}, Release: &credit.GoodsRelease{}, Obligation: &credit.Obligation{}}
	if _, err := RenderHTML(DocumentData{View: view}); err == nil {
		t.Fatal("expected hash mismatch")
	}
}

func TestRenderDrawdownHTMLVerifiesExactTerms(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	store := tradelines.NewStore()
	line, err := store.CreateLine(tradelines.CreateLineInput{SupplierOrganizationID: "supplier-org", BuyerUserID: "buyer-user", BuyerBusinessID: "buyer-business", ApprovedLimitKobo: 100000, Cadence: "monthly", StartAt: now.Add(-time.Hour), EndAt: now.AddDate(1, 0, 0), MandateID: "mandate-1", MandateActive: true, MandateVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	drawdown, _, _, err := store.ReserveDrawdown(tradelines.CreateDrawdownInput{LineID: line.ID, PrincipalKobo: 25000, GoodsDescription: "verified stock", DueDate: "2026-09-30", CollectionAt: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC), IdempotencyKey: "agreement-document"})
	if err != nil {
		t.Fatal(err)
	}
	body, err := RenderDrawdownHTML(DrawdownDocumentData{Line: line, Drawdown: drawdown})
	if err != nil || !strings.Contains(string(body), drawdown.AgreementHash) || !strings.Contains(string(body), "verified stock") {
		t.Fatalf("drawdown document missing verified evidence: err=%v", err)
	}
	drawdown.PrincipalKobo++
	if _, err := RenderDrawdownHTML(DrawdownDocumentData{Line: line, Drawdown: drawdown}); err == nil {
		t.Fatal("tampered drawdown terms rendered successfully")
	}
}
