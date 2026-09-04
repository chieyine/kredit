package mono

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kredit/internal/collections"
	"kredit/internal/mandates"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	client, err := New(server.URL, "test_sk_fixture", "fixture-webhook-secret", "https://kredit.example/return", true, func(context.Context, string, string) (string, error) { return "customer-reference", nil })
	if err != nil {
		t.Fatal(err)
	}
	client.http = server.Client()
	return client
}
func TestVariableSweepAuthorizationIsPending(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v2/payments/initiate" || r.Header.Get("mono-sec-key") != "test_sk_fixture" {
			t.Errorf("invalid mandate request")
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Fatal("invalid JSON")
		}
		if body["mandate_type"] != "sweep" || body["debit_type"] != "variable" || body["allow_partial_sweep"] != true || body["amount"] != float64(50000000) {
			t.Errorf("wrong variable sweep body: %v", body)
		}
		if _, ok := body["bvn"]; ok {
			t.Error("mandate should delegate BVN entry to hosted flow")
		}
		if _, err := w.Write([]byte(`{"status":"successful","data":{"mandate_id":"mmc_test","mono_url":"https://authorise.mono.co/test"}}`)); err != nil {
			t.Error(err)
		}
	})
	m, err := c.CreateAuthorizationSession(context.Background(), mandates.AuthorizationInput{UserID: "buyer", BusinessID: "business", AmountCeiling: 50000000, Purpose: "invoice"})
	if err != nil || m.Status != mandates.Pending || m.AuthorizationURL == "" || m.AmountCeiling != 50000000 {
		t.Fatalf("mandate=%+v err=%v", m, err)
	}
}
func TestActivationRequiresReadyFlag(t *testing.T) {
	for _, tc := range []struct {
		status string
		ready  bool
		want   mandates.Status
	}{{"approved", false, mandates.Pending}, {"approved", true, mandates.Active}, {"cancelled", true, mandates.Cancelled}, {"expired", false, mandates.Expired}} {
		t.Run(tc.status+map[bool]string{true: "-ready", false: "-not-ready"}[tc.ready], func(t *testing.T) {
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewEncoder(w).Encode(map[string]any{"status": "successful", "data": map[string]any{"id": "mmc_test", "mandate_type": "sweep", "debit_type": "variable", "status": tc.status, "approved": true, "ready_to_debit": tc.ready, "amount": 50000000}}); err != nil {
					t.Error(err)
				}
			})
			m, err := c.GetMandate(context.Background(), "mmc_test")
			if err != nil || m.Status != tc.want {
				t.Fatalf("got %s err=%v", m.Status, err)
			}
		})
	}
}
func TestPartialSweepUsesCollectedAmountNotRequestedAmount(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/payments/mandates/mmc_test/debit" {
			t.Errorf("wrong path %s", r.URL.Path)
		}
		if _, err := w.Write([]byte(`{"status":"successful","data":{"status":"partial-debit-successful","amount":50000000,"collected_amount":30000000,"pending_amount":20000000}}`)); err != nil {
			t.Error(err)
		}
	})
	out, err := c.Submit(context.Background(), collections.Request{MandateReference: "mmc_test", ExternalReference: "kredit-test", AmountKobo: 50000000, Currency: "NGN"})
	if err != nil || out.State != collections.ProviderPartial || out.SucceededAmountKobo != 30000000 {
		t.Fatalf("out=%+v err=%v", out, err)
	}
}
func TestUnknownOrMalformedOutcomesDoNotCreditMoney(t *testing.T) {
	for _, d := range []debitData{{Status: "successful", Amount: 50000000, Currency: "USD"}, {Status: "successful", Amount: 50000000, LiveMode: true}, {Status: "processing", Amount: 50000000}, {Status: "unrecognized", Amount: 50000000}, {Status: "successful"}, {Status: "partial-debit-successful", Amount: 50000000}, {Status: "successful", Amount: 60000000}} {
		out := debitResponse(d, "mandate", "reference", 50000000)
		if out.State != collections.ProviderPending || out.SucceededAmountKobo != 0 {
			t.Errorf("unsafe outcome %+v", out)
		}
	}
}
func TestWebhookAuthenticationAndSensitiveDataDiscard(t *testing.T) {
	c := testClient(t, func(http.ResponseWriter, *http.Request) {})
	raw := []byte(`{"event_id":"evt-1","event":"events.mandates.debit.successful","data":{"mandate":"mmc_test","reference_number":"kredit-test","amount":50000000,"bvn":"12345678901","account_details":{"account_number":"0123456789"}}}`)
	if _, err := c.ParseWebhook("bad", raw); err == nil {
		t.Fatal("accepted forged webhook")
	}
	event, err := c.ParseWebhook("fixture-webhook-secret", raw)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(event)
	if strings.Contains(string(encoded), "12345678901") || strings.Contains(string(encoded), "0123456789") || strings.Contains(string(encoded), "amount") {
		t.Fatal("sensitive or non-authoritative payload persisted")
	}
	again, _ := c.ParseWebhook("fixture-webhook-secret", raw)
	if again != event {
		t.Fatal("duplicate webhook changed identity")
	}
	signed := collections.Webhook{EventID: "a", ExternalReference: "b", State: collections.ProviderSucceeded, SucceededAmountKobo: 10}
	signed.Signature = c.Sign(signed)
	if !c.VerifyWebhook(signed) {
		t.Fatal("internal signature invalid")
	}
	signed.ProviderCollectionID = "tampered"
	if c.VerifyWebhook(signed) {
		t.Fatal("tampered financial identity verified")
	}
}
func TestReferenceLookupWorksWithoutSubmissionResponse(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v3/payments/mandates/mmc_test/debit/kredit-test" {
			t.Errorf("bad lookup path")
		}
		if _, err := w.Write([]byte(`{"status":"successful","data":{"status":"successful","amount":50000000,"mandate":"mmc_test","reference":"kredit-test"}}`)); err != nil {
			t.Error(err)
		}
	})
	out, err := c.GetByReference(context.Background(), collections.Request{MandateReference: "mmc_test", ExternalReference: "kredit-test", AmountKobo: 50000000})
	if err != nil || out.State != collections.ProviderSucceeded {
		t.Fatalf("out=%+v err=%v", out, err)
	}
}
func TestProviderErrorsDoNotExposeRestrictedValues(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		if _, err := w.Write([]byte(`{"bvn":"12345678901","message":"secret credential"}`)); err != nil {
			t.Error(err)
		}
	})
	_, err := c.GetMandate(context.Background(), "mmc_test")
	if err == nil || strings.Contains(err.Error(), "12345678901") || strings.Contains(err.Error(), "secret credential") {
		t.Fatalf("unsafe error: %v", err)
	}
}
func TestCustomerResponseOnlyReturnsReference(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(`{"status":"successful","data":{"id":"customer-reference","bvn":"12345678901","identification_no":"12345678901"}}`)); err != nil {
			t.Error(err)
		}
	})
	id, err := c.CreateCustomer(context.Background(), CustomerInput{FirstName: "Test", LastName: "Buyer", Email: "buyer@example.test", Phone: "08000000000", Address: "Test", BVN: "12345678901", ConsentVersion: "v1"})
	if err != nil || id != "customer-reference" {
		t.Fatalf("id=%s err=%v", id, err)
	}
}

func TestReconciliationModeRejectsNewMoneyActionsButAllowsLookup(t *testing.T) {
	calls := 0
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if _, err := w.Write([]byte(`{"status":"successful","data":{"id":"mmc_test","mandate_type":"sweep","debit_type":"variable","status":"approved","approved":true,"ready_to_debit":true,"amount":50000000}}`)); err != nil {
			t.Error(err)
		}
	}).ReconciliationOnly()
	if _, err := c.CreateCustomer(context.Background(), CustomerInput{}); err == nil {
		t.Fatal("customer creation permitted")
	}
	if _, err := c.CreateAuthorizationSession(context.Background(), mandates.AuthorizationInput{}); err == nil {
		t.Fatal("new authorization permitted")
	}
	if _, err := c.Submit(context.Background(), collections.Request{}); err == nil {
		t.Fatal("new debit permitted")
	}
	if calls != 0 {
		t.Fatal("provider called for disabled mutation")
	}
	if _, err := c.GetMandate(context.Background(), "mmc_test"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("reconciliation did not reach provider")
	}
}
