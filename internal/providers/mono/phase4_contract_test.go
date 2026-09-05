package mono

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"kredit/internal/collections"
	"kredit/internal/ledger"
	"kredit/internal/mandates"
)

// These are synthetic adapter contract tests, NOT evidence of Mono certification.
func TestPhase4DebitOutcomeMatrix(t *testing.T) {
	cases := []struct {
		name   string
		data   debitData
		state  string
		amount ledger.Money
	}{
		{"full", debitData{Status: "successful", Amount: 100000, Currency: "NGN"}, collections.ProviderSucceeded, 100000},
		{"partial", debitData{Status: "partial-debit-successful", Amount: 100000, Collected: 35000, Pending: 65000}, collections.ProviderPartial, 35000},
		{"failed", debitData{Status: "failed", Amount: 100000}, collections.ProviderFailed, 0},
		{"failed_with_collection", debitData{Status: "failed", Amount: 100000, Collected: 35000}, collections.ProviderPartial, 35000},
		{"processing", debitData{Status: "processing", Amount: 100000}, collections.ProviderPending, 0},
		{"processing_with_interim_collection", debitData{Status: "processing", Amount: 100000, Collected: 35000}, collections.ProviderPending, 0},
		{"unknown", debitData{Status: "future-provider-state", Amount: 100000}, collections.ProviderPending, 0},
		{"no_amount", debitData{Status: "successful"}, collections.ProviderPending, 0},
		{"partial_without_collected", debitData{Status: "partial-debit-successful", Amount: 100000}, collections.ProviderPending, 0},
		{"overcollection", debitData{Status: "successful", Amount: 100001}, collections.ProviderPending, 0},
		{"partial_overcollection", debitData{Status: "partial-debit-successful", Collected: 100001}, collections.ProviderPending, 0},
		{"negative_requested_field", debitData{Status: "successful", Amount: -1, Collected: 35000}, collections.ProviderPending, 0},
		{"negative_collected", debitData{Status: "successful", Amount: 100000, Collected: -1}, collections.ProviderPending, 0},
		{"negative_pending", debitData{Status: "successful", Amount: 100000, Pending: -1}, collections.ProviderPending, 0},
		{"full_with_pending", debitData{Status: "successful", Amount: 100000, Collected: 35000, Pending: 65000}, collections.ProviderPending, 0},
		{"live_result", debitData{Status: "successful", Amount: 100000, LiveMode: true}, collections.ProviderPending, 0},
		{"wrong_currency", debitData{Status: "successful", Amount: 100000, Currency: "USD"}, collections.ProviderPending, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := debitResponse(tc.data, "mmc_fixture", "kredit-fixture", 100000)
			if out.State != tc.state || out.SucceededAmountKobo != tc.amount || out.Retryable {
				t.Fatalf("outcome=%+v, want state=%s amount=%d and no automatic retry", out, tc.state, tc.amount)
			}
		})
	}
	for _, requested := range []ledger.Money{0, -1} {
		out := debitResponse(debitData{Status: "successful", Amount: 100000}, "mmc_fixture", "kredit-fixture", requested)
		if out.State != collections.ProviderPending || out.SucceededAmountKobo != 0 {
			t.Fatalf("invalid persisted amount produced money: %+v", out)
		}
	}
}

func TestPhase4LookupRejectsUnconfirmedIdentity(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mandate   string
		reference string
		envelope  string
	}{
		{"foreign_mandate", "mmc_other", "kredit-fixture", "successful"},
		{"foreign_reference", "mmc_fixture", "other-reference", "successful"},
		{"missing_mandate", "", "kredit-fixture", "successful"},
		{"missing_reference", "mmc_fixture", "", "successful"},
		{"unsuccessful_envelope", "mmc_fixture", "kredit-fixture", "failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v3/payments/mandates/mmc_fixture/debit/kredit-fixture" {
					t.Errorf("unexpected lookup request: %s %s", r.Method, r.URL.Path)
				}
				out := map[string]any{"status": tc.envelope, "data": map[string]any{"status": "successful", "amount": 100000, "mandate": tc.mandate, "reference": tc.reference}}
				if err := json.NewEncoder(w).Encode(out); err != nil {
					t.Error(err)
				}
			})
			out, err := c.GetByReference(context.Background(), collections.Request{MandateReference: "mmc_fixture", ExternalReference: "kredit-fixture", AmountKobo: 100000, Currency: "NGN"})
			if err == nil || out.SucceededAmountKobo != 0 {
				t.Fatalf("unconfirmed identity accepted: %+v, err=%v", out, err)
			}
		})
	}
}

type phase4RoundTripper func(*http.Request) (*http.Response, error)

func (f phase4RoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestPhase4LostSubmissionResponseUsesOriginalReference(t *testing.T) {
	c := testClient(t, func(http.ResponseWriter, *http.Request) {})
	posts, gets := 0, 0
	c.http = &http.Client{Transport: phase4RoundTripper(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodPost {
			posts++
			return nil, errors.New("simulated lost response containing a private-transport-detail")
		}
		gets++
		if r.Method != http.MethodGet || r.URL.Path != "/v3/payments/mandates/mmc_fixture/debit/kredit-fixture" {
			t.Errorf("reconciliation changed the saved identity: %s %s", r.Method, r.URL.Path)
		}
		body := `{"status":"successful","data":{"status":"successful","amount":100000,"mandate":"mmc_fixture","reference":"kredit-fixture","live_mode":false}}`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	in := collections.Request{MandateReference: "mmc_fixture", ExternalReference: "kredit-fixture", AmountKobo: 100000, Currency: "NGN"}
	out, err := c.Submit(context.Background(), in)
	if err == nil || !strings.Contains(err.Error(), "unknown") || strings.Contains(err.Error(), "private-transport-detail") || out.SucceededAmountKobo != 0 {
		t.Fatalf("unsafe ambiguous result: %+v, err=%v", out, err)
	}
	for range 2 {
		out, err = c.GetByReference(context.Background(), in)
		if err != nil || out.State != collections.ProviderSucceeded || out.SucceededAmountKobo != 100000 {
			t.Fatalf("reference recovery failed: %+v, err=%v", out, err)
		}
	}
	if posts != 1 || gets != 2 {
		t.Fatalf("unexpected requests: posts=%d gets=%d", posts, gets)
	}
}

func TestPhase4MandateLifecycleMapping(t *testing.T) {
	for _, tc := range []struct {
		status   string
		approved bool
		ready    bool
		want     mandates.Status
	}{
		{"approved", false, true, mandates.Pending},
		{"approved", true, false, mandates.Pending},
		{"approved", true, true, mandates.Active},
		{"paused", true, true, mandates.Paused},
		{"suspended", true, true, mandates.Paused},
		{"cancelled", true, true, mandates.Cancelled},
		{"expired", true, true, mandates.Expired},
		{"rejected", true, true, mandates.Failed},
		{"failed", true, true, mandates.Failed},
		{"unrecognized", true, true, mandates.Pending},
	} {
		t.Run(tc.status+"_"+string(tc.want), func(t *testing.T) {
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v3/payments/mandates/mmc_fixture" {
					t.Errorf("unexpected mandate request")
				}
				out := map[string]any{"status": "successful", "data": map[string]any{"id": "mmc_fixture", "status": tc.status, "approved": tc.approved, "ready_to_debit": tc.ready, "amount": 100000, "mandate_type": "sweep", "debit_type": "variable"}}
				if err := json.NewEncoder(w).Encode(out); err != nil {
					t.Error(err)
				}
			})
			m, err := c.GetMandate(context.Background(), "mmc_fixture")
			if err != nil || m.Status != tc.want {
				t.Fatalf("mandate=%+v err=%v want=%s", m, err, tc.want)
			}
		})
	}
}

func TestPhase4CancelledMandateRequiresFreshAuthorization(t *testing.T) {
	var calls atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPatch || r.URL.Path != "/v3/payments/mandates/mmc_fixture/cancel" {
			t.Errorf("unexpected cancellation request")
		}
		if _, err := io.WriteString(w, `{"status":"success","data":{}}`); err != nil {
			t.Error(err)
		}
	})
	m, err := c.CancelMandate(context.Background(), "mmc_fixture", "buyer")
	if err != nil || m.Status != mandates.Cancelled {
		t.Fatalf("cancellation=%+v err=%v", m, err)
	}
	if _, err = c.RestoreAuthorization(context.Background(), "mmc_fixture"); err == nil {
		t.Fatal("cancelled mandate restored without fresh authorization")
	}
	if calls.Load() != 1 {
		t.Fatal("restoration unexpectedly contacted provider")
	}
}

func TestPhase4WebhookSignalMatrix(t *testing.T) {
	c := testClient(t, func(http.ResponseWriter, *http.Request) {})
	events := map[string]mandates.Status{
		"events.mandates.created":                  "",
		"events.mandates.approved":                 "",
		"events.mandates.ready":                    "",
		"events.mandate.action.reinstate":           "",
		"events.mandates.rejected":                 mandates.Failed,
		"events.mandate.action.pause":              mandates.Paused,
		"events.mandate.action.cancel":             mandates.Cancelled,
		"events.mandates.expired":                  mandates.Expired,
		"events.mandates.debit.processing":         "",
		"events.mandates.debit.successful":         "",
		"events.mandates.debit.failed":             "",
		"events.mandates.debit_attempt.successful": "",
	}
	for name, block := range events {
		t.Run(name, func(t *testing.T) {
			raw, err := json.Marshal(map[string]any{"event": name, "data": map[string]any{"mandate": "mmc_fixture", "reference_number": "kredit-fixture", "amount": 100000, "collected_amount": 35000, "bvn": "private-bvn-fixture", "account_details": map[string]string{"account_number": "private-account-fixture"}, "live_mode": false}})
			if err != nil {
				t.Fatal(err)
			}
			notice, err := c.ParseWebhook("fixture-webhook-secret", raw)
			if err != nil || notice.BlockStatus != block || notice.Reference != "kredit-fixture" {
				t.Fatalf("notice=%+v err=%v", notice, err)
			}
			again, err := c.ParseWebhook("fixture-webhook-secret", raw)
			if err != nil || again != notice || !strings.HasPrefix(notice.EventID, "sha256:") {
				t.Fatal("identical callback did not retain its fallback receipt identity")
			}
			encoded, err := json.Marshal(notice)
			if err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{"amount", "private-bvn-fixture", "private-account-fixture", "account_details"} {
				if strings.Contains(string(encoded), forbidden) {
					t.Fatalf("webhook receipt retained restricted or non-authoritative field %q", forbidden)
				}
			}
		})
	}
}

func TestPhase4EveryDebitSignalRequiresReference(t *testing.T) {
	c := testClient(t, func(http.ResponseWriter, *http.Request) {})
	for _, name := range []string{"events.mandates.debit.processing", "events.mandates.debit.successful", "events.mandates.debit.failed", "events.mandates.debit_attempt.successful"} {
		t.Run(name, func(t *testing.T) {
			raw, err := json.Marshal(map[string]any{"event": name, "data": map[string]any{"mandate": "mmc_fixture", "live_mode": false}})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.ParseWebhook("fixture-webhook-secret", raw); err == nil {
				t.Fatal("accepted an uncorrelated debit notice")
			}
		})
	}
}

func TestPhase4WebhookRejectsInvalidEvidence(t *testing.T) {
	c := testClient(t, func(http.ResponseWriter, *http.Request) {})
	for _, raw := range []string{
		`not-json`,
		`{"event":"unknown","data":{"mandate":"mmc_fixture"}}`,
		`{"event":"events.mandates.ready","data":{}}`,
		`{"event":"events.mandates.ready","data":{"id":"mmc_fixture","live_mode":true}}`,
		strings.Repeat(" ", (1<<20)+1),
	} {
		if _, err := c.ParseWebhook("fixture-webhook-secret", []byte(raw)); err == nil {
			t.Fatal("invalid or live provider evidence was accepted")
		}
	}
	valid := []byte(`{"event":"events.mandates.ready","data":{"id":"mmc_fixture","live_mode":false}}`)
	for _, secret := range []string{"", "wrong-secret"} {
		if _, err := c.ParseWebhook(secret, valid); err == nil {
			t.Fatal("unauthenticated provider evidence was accepted")
		}
	}
}
