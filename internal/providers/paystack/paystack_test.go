package paystack

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"kredit/internal/collections"
	"kredit/internal/mandates"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fixture(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	c, err := New("paystack-main", "sk_test_synthetic", "https://kredit.ng/buyer/mandates", false, func(context.Context, string) (string, error) { return "buyer@example.test", nil })
	if err != nil {
		t.Fatal(err)
	}
	c.http.Transport = handlerTransport(handler)
	return c
}
func TestHostedAuthorizationAndExactKoboCharge(t *testing.T) {
	var chargeCount int
	c := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk_test_synthetic" {
			t.Error("missing authentication")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/customer/authorization/initialize":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["channel"] != "direct_debit" || body["email"] != "buyer@example.test" || body["callback_url"] != "https://kredit.ng/buyer/mandates" {
				t.Error("incorrect authorization payload")
			}
			w.Write([]byte(`{"status":true,"data":{"reference":"original-auth","redirect_url":"https://link.paystack.com/example"}}`))
		case "/customer/authorization/verify/original-auth":
			w.Write([]byte(`{"status":true,"data":{"authorization_code":"AUTH_secret","channel":"direct_debit","active":true,"customer":{"email":"buyer@example.test"}}}`))
		case "/transaction/charge_authorization":
			chargeCount++
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["amount"] != float64(12345) || body["reference"] != "reserved-ref" || body["authorization_code"] != "AUTH_secret" {
				t.Error("amount/reference/permission changed")
			}
			w.Write([]byte(`{"status":true,"data":{"status":"success"}}`))
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			w.WriteHeader(400)
		}
	})
	m, err := c.CreateAuthorizationSession(context.Background(), mandates.AuthorizationInput{UserID: "buyer", AmountCeiling: 12345})
	if err != nil || m.Status != mandates.Pending {
		t.Fatalf("%+v %v", m, err)
	}
	encoded, _ := json.Marshal(m)
	if strings.Contains(string(encoded), "AUTH_secret") {
		t.Fatal("authorization secret exposed")
	}
	out, err := c.Submit(context.Background(), collections.Request{MandateReference: m.ProviderID, ExternalReference: "reserved-ref", AmountKobo: 12345, Currency: "NGN"})
	if err != nil || out.State != collections.ProviderPending || out.SucceededAmountKobo != 0 || chargeCount != 1 {
		t.Fatalf("unverified money recognized: %+v %v", out, err)
	}
}
func TestVerificationBindsEveryPaymentFact(t *testing.T) {
	for _, field := range []string{"valid", "reference", "amount", "currency", "domain", "authorization", "email"} {
		t.Run(field, func(t *testing.T) {
			c := fixture(t, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/customer/") {
					w.Write([]byte(`{"status":true,"data":{"authorization_code":"AUTH_secret","channel":"direct_debit","active":false,"customer":{"email":"buyer@example.test"}}}`))
					return
				}
				data := map[string]any{"reference": "reserved-ref", "amount": 12345, "currency": "NGN", "domain": "test", "status": "success", "authorization": map[string]string{"authorization_code": "AUTH_secret", "channel": "direct_debit"}, "customer": map[string]string{"email": "buyer@example.test"}}
				switch field {
				case "reference":
					data[field] = "other"
				case "amount":
					data[field] = 12346
				case "currency":
					data[field] = "USD"
				case "domain":
					data[field] = "live"
				case "authorization":
					data[field] = map[string]string{"authorization_code": "another", "channel": "direct_debit"}
				case "email":
					data["customer"] = map[string]string{"email": "other@example.test"}
				}
				json.NewEncoder(w).Encode(map[string]any{"status": true, "data": data})
			})
			out, err := c.GetByReference(context.Background(), collections.Request{MandateReference: "original-auth", ExternalReference: "reserved-ref", AmountKobo: 12345, Currency: "NGN"})
			if field == "valid" {
				if err != nil || out.SucceededAmountKobo != 12345 || out.SettlementState != collections.ProviderSettlementPending {
					t.Fatalf("%+v %v", out, err)
				}
			} else if err == nil {
				t.Fatal("mismatched payment accepted")
			}
		})
	}
}
func TestUnknownOutcomeNeverRetransmits(t *testing.T) {
	calls := 0
	c := fixture(t, func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(503) })
	_, err := c.GetByReference(context.Background(), collections.Request{ExternalReference: "saved", MandateReference: "original", AmountKobo: 100, Currency: "NGN"})
	if err == nil || calls != 1 {
		t.Fatal("ambiguous lookup must remain unresolved")
	}
}
func TestNativeWebhookSignatureAndEnvironment(t *testing.T) {
	c := fixture(t, func(http.ResponseWriter, *http.Request) { t.Fatal("webhook must not contact provider") })
	raw := []byte(`{"event":"charge.success","data":{"reference":"reserved-ref","domain":"test"}}`)
	m := hmac.New(sha512.New, []byte(c.secret))
	m.Write(raw)
	sig := hex.EncodeToString(m.Sum(nil))
	n, err := c.ParseWebhook(sig, raw)
	if err != nil || n.Reference != "reserved-ref" {
		t.Fatal(err)
	}
	if _, err = c.ParseWebhook(sig, append(raw, ' ')); err == nil {
		t.Fatal("altered payload accepted")
	}
	c.live = true
	if _, err = c.ParseWebhook(sig, raw); err == nil {
		t.Fatal("test payment accepted in production")
	}
}

type handlerTransport http.HandlerFunc

func (h handlerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	h(recorder, r)
	return recorder.Result(), nil
}

func TestRecoveryRequiresOriginalBuyer(t *testing.T) {
	for _, email := range []string{"buyer@example.test", "someone-else@example.test"} {
		c := fixture(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]any{"status": true, "data": map[string]any{"authorization_code": "AUTH_secret", "channel": "direct_debit", "active": true, "customer": map[string]string{"email": email}}})
		})
		m, e := c.RecoverAuthorization(context.Background(), mandates.AuthorizationInput{UserID: "buyer", Reference: "saved-local-ref", AmountCeiling: 12345}, "provider-ref")
		if email == "buyer@example.test" {
			if e != nil || m.AmountCeiling != 12345 || m.ProviderID != "provider-ref" {
				t.Fatalf("%+v %v", m, e)
			}
		} else if e == nil {
			t.Fatal("another buyer's authorization was attached")
		}
	}
}
