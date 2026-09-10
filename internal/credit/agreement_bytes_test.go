package credit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestAgreementPreservesAcceptedHashAcrossJSONNormalization(t *testing.T) {
	original, err := json.Marshal(agreementCanonical{SupplierLegalName: "Supplier", BuyerLegalName: "Buyer", BuyerBusinessID: "buyer-business", GoodsDescription: "Cartons", PrincipalKobo: 125000, Currency: "NGN", DueDate: "2026-09-30", TermsVersion: "terms-test", PrivacyVersion: "privacy-test"})
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(original)
	for _, legacy := range []bool{false, true} {
		a := AgreementVersion{ID: "agreement", CanonicalJSON: original, DocumentHash: hex.EncodeToString(hash[:])}
		if !legacy {
			a.CanonicalBytes = original
		}
		encoded, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if err = json.Unmarshal(encoded, &value); err != nil {
			t.Fatal(err)
		}
		normalized, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		var restored AgreementVersion
		if err = json.Unmarshal(normalized, &restored); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(restored.CanonicalJSON, original) {
			t.Fatalf("accepted bytes changed, legacy=%v", legacy)
		}
		if _, err = PrintableAgreement(View{Agreement: restored}); err != nil {
			t.Fatalf("reopened agreement cannot print, legacy=%v: %v", legacy, err)
		}
	}
	corrupted := AgreementVersion{ID: "agreement", CanonicalJSON: original, CanonicalBytes: []byte(`{"principal_kobo":1}`), DocumentHash: hex.EncodeToString(hash[:])}
	encoded, _ := json.Marshal(corrupted)
	var restored AgreementVersion
	if err = json.Unmarshal(encoded, &restored); err == nil {
		t.Fatal("changed accepted bytes passed hash verification")
	}
}
