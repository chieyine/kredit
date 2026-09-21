package notifications

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type auditResponseTransport func(*http.Request) (*http.Response, error)

func (f auditResponseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type auditResponseBody struct {
	io.Reader
	read   int
	closed bool
}

func (b *auditResponseBody) Read(p []byte) (int, error) {
	n, e := b.Reader.Read(p)
	b.read += n
	return n, e
}
func (b *auditResponseBody) Close() error { b.closed = true; return nil }

type auditReadFailure struct{}

func (auditReadFailure) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

// A reader reaching its byte cap is not proof of a complete provider response.
func TestAuditCompleteProviderResponse(t *testing.T) {
	const limit = 65536
	const prefix = `{"message_id":"synthetic-receipt"}`
	padded := prefix + strings.Repeat(" ", limit-len(prefix))
	for _, tc := range []struct {
		name, body      string
		status          int
		ok, readFailure bool
	}{
		{name: "normal", body: prefix, status: 200, ok: true},
		{name: "exact limit", body: padded, status: 200, ok: true},
		{name: "oversized whitespace", body: padded + " ", status: 200},
		{name: "trailing rejected content", body: padded + "not-json", status: 200},
		{name: "body read failure", body: prefix, status: 200, readFailure: true},
		{name: "provider rejection", body: prefix, status: 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var reader io.Reader = strings.NewReader(tc.body)
			if tc.readFailure {
				reader = io.MultiReader(reader, auditReadFailure{})
			}
			body := &auditResponseBody{Reader: reader}
			response := &http.Response{StatusCode: tc.status, Body: body, Header: make(http.Header), ContentLength: -1}
			provider, err := NewWebhookProvider(ChannelSMS, "https://connector.example/send", "synthetic-connector-token")
			if err != nil {
				t.Fatal(err)
			}
			provider.client.Transport = auditResponseTransport(func(*http.Request) (*http.Response, error) { return response, nil })
			id, err := provider.Send(context.Background(), Message{Channel: ChannelSMS, Destination: "synthetic-recipient", EventID: "event-test"})
			if (err == nil) != tc.ok || (err != nil && id != "") {
				t.Fatalf("id=%q error=%v; want success=%v", id, err, tc.ok)
			}
			if !body.closed {
				t.Fatal("response body was not closed")
			}
			if body.read > limit+1 {
				t.Fatalf("read %d bytes; maximum is %d", body.read, limit+1)
			}
		})
	}
}
