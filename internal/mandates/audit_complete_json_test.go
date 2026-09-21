package mandates

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type auditJSONTransport func(*http.Request) (*http.Response, error)

func (f auditJSONTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type auditJSONBody struct {
	io.Reader
	read   int
	closed bool
}

func (b *auditJSONBody) Read(p []byte) (int, error) {
	n, e := b.Reader.Read(p)
	b.read += n
	return n, e
}
func (b *auditJSONBody) Close() error { b.closed = true; return nil }

type auditJSONReadFailure struct{}

func (auditJSONReadFailure) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestAuditProviderRequiresCompleteJSON(t *testing.T) {
	const limit = 1 << 20
	adapters := []string{"mandate connector"}
	for _, adapter := range adapters {
		t.Run(adapter, func(t *testing.T) {
			const prefix = `{"accepted":true}`
			padded := prefix + strings.Repeat(" ", limit-len(prefix))
			for _, tc := range []struct {
				name, body string
				ok, broken bool
			}{
				{name: "normal", body: prefix, ok: true},
				{name: "exact limit", body: padded, ok: true},
				{name: "second value", body: prefix + `{}`},
				{name: "trailing garbage", body: prefix + `not-json`},
				{name: "oversized", body: padded + " "},
				{name: "truncated successful prefix", body: padded + `not-json`},
				{name: "read error after valid prefix", body: prefix, broken: true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					var reader io.Reader = strings.NewReader(tc.body)
					if tc.broken {
						reader = io.MultiReader(reader, auditJSONReadFailure{})
					}
					body := &auditJSONBody{Reader: reader}
					transport := auditJSONTransport(func(r *http.Request) (*http.Response, error) {
						return &http.Response{StatusCode: 200, Header: make(http.Header), ContentLength: -1, Body: body, Request: r}, nil
					})
					client := &http.Client{Transport: transport}
					var result struct {
						Accepted bool `json:"accepted"`
					}
					provider := &WebhookProvider{endpoint: "https://connector.example", token: "synthetic", client: client}
					err := provider.request(context.Background(), http.MethodGet, "/mandates/synthetic", nil, &result)
					if (err == nil) != tc.ok {
						t.Fatalf("complete JSON accepted=%v error=%v, want success=%v", result.Accepted, err, tc.ok)
					}
					if tc.ok && !result.Accepted {
						t.Fatal("valid response data was lost")
					}
					if !body.closed || body.read > limit+1 {
						t.Fatalf("body closed=%v bytes=%d", body.closed, body.read)
					}
				})
			}
		})
	}
}
