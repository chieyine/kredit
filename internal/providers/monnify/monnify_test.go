package monnify

import (
	"context"
	"encoding/json"
	"kredit/internal/collections"
	"kredit/internal/mandates"
	"kredit/internal/providers/bankdebit"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type storeFixture struct {
	bankdebit.RecoveryStore
	v bankdebit.Enrollment
}

func (s *storeFixture) Load(context.Context, string, string) (bankdebit.Enrollment, error) {
	return s.v, nil
}

type transport http.HandlerFunc

func (h transport) RoundTrip(r *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	h(w, r)
	return w.Result(), nil
}
func setup(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	store := &storeFixture{v: bankdebit.Enrollment{Provider: "monnify-main", Reference: "local", State: "CONFIRMED", Input: mandates.AuthorizationInput{AmountCeiling: 100000}, Result: bankdebit.Result{Reference: "local"}, Details: bankdebit.Details{Email: "buyer@example.test", AccountNumber: "0123456789", BankCode: "058"}}}
	c, e := New("monnify-main", "synthetic-key", "synthetic-secret", "contract", "https://kredit.ng/buyer/mandates", false, store)
	if e != nil {
		t.Fatal(e)
	}
	c.token = "temporary"
	c.expires = time.Now().Add(time.Hour)
	c.http.Transport = transport(handler)
	return c
}
func mandateJSON(w http.ResponseWriter) {
	json.NewEncoder(w).Encode(map[string]any{"requestSuccessful": true, "responseCode": "0", "responseBody": []any{map[string]any{"mandateReference": "local", "mandateCode": "CODE", "mandateStatus": "ACTIVATED", "mandateAmount": 1000, "contractCode": "contract", "customerEmailAddress": "buyer@example.test", "customerAccountNumber": "0123456789", "customerAccountBankCode": "058", "startDate": time.Now().Add(-time.Hour).Format(time.RFC3339), "endDate": time.Now().Add(time.Hour).Format(time.RFC3339)}}})
}
func TestDebitUsesPublishedSchemaAndExactNaira(t *testing.T) {
	posts := 0
	c := setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			mandateJSON(w)
			return
		}
		posts++
		if r.URL.Path != "/api/v1/direct-debit/mandate/debit" {
			t.Fatal("wrong endpoint")
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["debitAmount"] != 123.45 || body["mandateCode"] != "CODE" || body["customerEmail"] != "buyer@example.test" || body["paymentReference"] != "saved-ref" || body["amount"] != nil {
			t.Fatalf("incorrect debit contract: %v", body)
		}
		w.Write([]byte(`{"requestSuccessful":true,"responseCode":"0","responseBody":{"transactionStatus":"PENDING"}}`))
	})
	out, e := c.Submit(context.Background(), collections.Request{MandateReference: "local", ExternalReference: "saved-ref", AmountKobo: 12345, Currency: "NGN"})
	if e != nil || posts != 1 || out.State != collections.ProviderPending || out.SucceededAmountKobo != 0 {
		t.Fatalf("%+v %v", out, e)
	}
}
func TestStatusLookupBindsOriginalMandate(t *testing.T) {
	for _, field := range []string{"valid", "amount", "reference", "mandate"} {
		t.Run(field, func(t *testing.T) {
			c := setup(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/direct-debit/mandate/" {
					mandateJSON(w)
					return
				}
				if r.URL.Query().Get("paymentReference") != "saved-ref" {
					t.Fatal("wrong reference lookup")
				}
				data := map[string]any{"transactionStatus": "PAID", "paymentReference": "saved-ref", "debitAmount": 123.45, "mandateCode": "CODE"}
				switch field {
				case "amount":
					data["debitAmount"] = 123.46
				case "reference":
					data["paymentReference"] = "other"
				case "mandate":
					data["mandateCode"] = "other"
				}
				json.NewEncoder(w).Encode(map[string]any{"requestSuccessful": true, "responseCode": "0", "responseBody": data})
			})
			out, e := c.GetByReference(context.Background(), collections.Request{MandateReference: "local", ExternalReference: "saved-ref", AmountKobo: 12345, Currency: "NGN"})
			if field == "valid" {
				if e != nil || out.SucceededAmountKobo != 12345 || out.SettlementState != collections.ProviderSettlementPending {
					t.Fatalf("%+v %v", out, e)
				}
			} else if e == nil {
				t.Fatal("mismatched debit accepted")
			}
		})
	}
}
