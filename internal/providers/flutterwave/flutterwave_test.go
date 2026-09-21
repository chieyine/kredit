package flutterwave

import (
	"context"
	"encoding/json"
	"kredit/internal/collections"
	"kredit/internal/mandates"
	"kredit/internal/providers/bankdebit"
	"net/http"
	"net/http/httptest"
	"strings"
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
	store := &storeFixture{v: bankdebit.Enrollment{Provider: "flw-main", Reference: "local", State: "CONFIRMED", Input: mandates.AuthorizationInput{AmountCeiling: 100000}, Result: bankdebit.Result{Reference: "vendor-ref"}, Details: bankdebit.Details{Email: "buyer@example.test", AccountNumber: "0123456789", BankCode: "058"}}}
	c, e := New("flw-main", "FLWSECK_TEST-synthetic", strings.Repeat("s", 32), false, store)
	if e != nil {
		t.Fatal(e)
	}
	c.http.Transport = transport(handler)
	return c
}
func TestDebitRequiresActiveTokenAndKeepsExactAmount(t *testing.T) {
	for _, status := range []string{"APPROVED", "ACTIVE"} {
		t.Run(status, func(t *testing.T) {
			posts := 0
			c := setup(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{"reference": "vendor-ref", "narration": "Kredit bank permission local", "status": status, "token": "private-token", "currency": "NGN", "amount": 1000, "start_date": time.Now().Add(-time.Hour), "end_date": time.Now().Add(time.Hour)}})
					return
				}
				posts++
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if body["amount"] != 123.45 || body["tx_ref"] != "saved-ref" || body["type"] != "account" {
					t.Errorf("incorrect debit payload: %v", body)
				}
				_, _ = w.Write([]byte(`{"status":"success","data":{"status":"successful"}}`))
			})
			out, e := c.Submit(context.Background(), collections.Request{ExternalReference: "saved-ref", MandateReference: "local", AmountKobo: 12345, Currency: "NGN"})
			if status == "APPROVED" {
				if e == nil || posts != 0 {
					t.Fatal("debited before activation")
				}
			} else if e != nil || posts != 1 || out.State != collections.ProviderPending || out.SucceededAmountKobo != 0 {
				t.Fatalf("%+v %v", out, e)
			}
		})
	}
}
func TestPaymentVerificationRejectsDifferentBankAndAmounts(t *testing.T) {
	for _, field := range []string{"valid", "amount", "reference", "account", "bank", "currency"} {
		t.Run(field, func(t *testing.T) {
			c := setup(t, func(w http.ResponseWriter, r *http.Request) {
				data := map[string]any{"tx_ref": "saved-ref", "amount": 123.45, "currency": "NGN", "status": "successful", "payment_type": "account", "auth_model": "EMANDATE", "customer": map[string]string{"email": "buyer@example.test"}, "account": map[string]string{"account_number": "0123456789", "bank_code": "058"}}
				switch field {
				case "amount":
					data["amount"] = 123.46
				case "reference":
					data["tx_ref"] = "other"
				case "account":
					data["account"] = map[string]string{"account_number": "9876543210", "bank_code": "058"}
				case "bank":
					data["account"] = map[string]string{"account_number": "0123456789", "bank_code": "044"}
				case "currency":
					data["currency"] = "USD"
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": data})
			})
			out, e := c.GetByReference(context.Background(), collections.Request{ExternalReference: "saved-ref", MandateReference: "local", AmountKobo: 12345, Currency: "NGN"})
			if field == "valid" {
				if e != nil || out.SucceededAmountKobo != 12345 || out.SettlementState != collections.ProviderSettlementPending {
					t.Fatalf("%+v %v", out, e)
				}
			} else if e == nil {
				t.Fatal("mismatched payment accepted")
			}
		})
	}
}
func TestNativeWebhookRejectsWrongSecret(t *testing.T) {
	c := setup(t, nil)
	raw := []byte(`{"event":"charge.completed","data":{"tx_ref":"saved"}}`)
	headers := http.Header{"Verif-Hash": []string{c.webhookSecret}}
	if _, e := c.ParseNotice(headers, raw); e != nil {
		t.Fatal(e)
	}
	headers.Set("verif-hash", "wrong")
	if _, e := c.ParseNotice(headers, raw); e == nil {
		t.Fatal("accepted unauthenticated event")
	}
}
