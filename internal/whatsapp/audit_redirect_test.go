package whatsapp

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestAuditAIRefusesRedirectsWithAPIKey(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			parser := NewAIParser("synthetic-key")
			calls := 0
			parser.client.Transport = auditResponseTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return &http.Response{StatusCode: status, Header: http.Header{"Location": {"https://untrusted.example/redirect"}}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
				}
				if req.Header.Get("x-goog-api-key") != "" {
					t.Error("API key was forwarded to a redirected request")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"parts":[{"text":"{\"intent\":\"help\"}"}]}}]}`)), Request: req}, nil
			})
			result, err := parser.ParseText(context.Background(), "help")
			if err == nil || result.Intent != IntentUnknown || calls != 1 {
				t.Fatalf("redirect accepted: result=%+v err=%v calls=%d", result, err, calls)
			}
		})
	}
}
