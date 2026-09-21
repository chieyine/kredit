package paystack

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"kredit/internal/mandates"
)

func TestAuditHostedAuthorizationURL(t *testing.T) {
	// The direct-debit guide documents link.paystack.co; the customer API
	// reference also shows checkout.paystack.com. Both are hosted responses.
	for _, tc := range []struct {
		url string
		ok  bool
	}{
		{"https://link.paystack.co/synthetic", true},
		{"https://checkout.paystack.com/synthetic", true},
		{"https://link.paystack.com/synthetic", true},
		{"https://LINK.PAYSTACK.CO:443/synthetic", true},
		{"https://link.paystack.co.untrusted.example/synthetic", false},
		{"https://untrusted.link.paystack.co/synthetic", false},
		{"https://untrustedpaystack.com/synthetic", false},
		{"https://link.paystack.co@untrusted.example/synthetic", false},
		{"https://untrusted@link.paystack.co/synthetic", false},
		{"http://link.paystack.co/synthetic", false},
		{"https://link.paystack.co:8443/synthetic", false},
		{"//link.paystack.co/synthetic", false},
		{"", false},
	} {
		t.Run(tc.url, func(t *testing.T) {
			client, err := New("synthetic", "sk_test_synthetic", "https://kredit.example/return", false, func(context.Context, string) (string, error) {
				return "buyer@example.test", nil
			})
			if err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(map[string]any{"status": true, "data": map[string]string{"reference": "provider-reference", "redirect_url": tc.url}})
			if err != nil {
				t.Fatal(err)
			}
			client.http.Transport = auditJSONTransport(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path != "/customer/authorization/initialize" || req.Method != http.MethodPost {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
			})
			mandate, err := client.CreateAuthorizationSession(context.Background(), mandates.AuthorizationInput{UserID: "buyer", Reference: "local-reference", AmountCeiling: 10000})
			if (err == nil) != tc.ok {
				t.Fatalf("success=%v, want=%v: %v", err == nil, tc.ok, err)
			}
			if tc.ok && (mandate.AuthorizationURL != tc.url || mandate.ProviderID != "provider-reference" || mandate.Reference != "local-reference" || mandate.Status != mandates.Pending) {
				t.Fatalf("authorization identity or state changed: %+v", mandate)
			}
		})
	}
}
