package mandates_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"kredit/internal/collections"
	"kredit/internal/identity"
	"kredit/internal/mandates"
)

func TestConnectorsRejectRedirects(t *testing.T) {
	var redirected atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirected.Add(1)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	mandate, err := mandates.NewWebhookProvider("connector", server.URL, "test-token")
	if err != nil {
		t.Fatal(err)
	}
	collection, err := collections.NewWebhookProvider("connector", server.URL, "test-token", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	person, err := identity.NewWebhookProvider("connector", server.URL, "test-token", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	for name, call := range map[string]func(context.Context) error{
		"mandate":    func(ctx context.Context) error { _, err := mandate.GetMandate(ctx, "expected"); return err },
		"collection": func(ctx context.Context) error { _, err := collection.Get(ctx, "expected"); return err },
		"identity":   func(ctx context.Context) error { _, err := person.GetVerification(ctx, "expected"); return err },
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(t.Context()); err == nil {
				t.Fatal("connector accepted redirect")
			}
		})
	}
	if redirected.Load() != 0 {
		t.Fatal("connector contacted redirected destination")
	}
}

func TestMandateConnectorValidatesIdentityAndCancellation(t *testing.T) {
	for _, test := range []struct {
		name, body string
		cancel     bool
		wantError  bool
	}{
		{"wrong mandate", `{"provider_id":"other","status":"ACTIVE"}`, false, true},
		{"unknown state", `{"provider_id":"expected","status":"maybe"}`, false, true},
		{"unconfirmed cancellation", `{"provider_id":"expected","status":"ACTIVE"}`, true, true},
		{"confirmed cancellation", `{"provider_id":"expected","status":"cancelled"}`, true, false},
		{"correct active mandate", `{"provider_id":"expected","status":"active"}`, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(test.body)) }))
			defer server.Close()
			provider, err := mandates.NewWebhookProvider("connector", server.URL, "test-token")
			if err != nil {
				t.Fatal(err)
			}
			if test.cancel {
				_, err = provider.CancelMandate(t.Context(), "expected", "buyer requested")
			} else {
				_, err = provider.GetMandate(t.Context(), "expected")
			}
			if (err != nil) != test.wantError {
				t.Fatalf("error = %v, want error = %v", err, test.wantError)
			}
		})
	}
}

func TestCollectionConnectorRejectsMismatchedLookup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ProviderCollectionID":"other","State":"succeeded","SucceededAmountKobo":100}`))
	}))
	defer server.Close()
	provider, err := collections.NewWebhookProvider("connector", server.URL, "test-token", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Get(t.Context(), "expected"); err == nil {
		t.Fatal("lookup accepted a different collection")
	}
	if _, err := provider.Cancel(t.Context(), "expected"); err == nil {
		t.Fatal("cancellation accepted a different collection")
	}
}
