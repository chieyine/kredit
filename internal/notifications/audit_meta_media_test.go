package notifications

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type auditMediaTransport func(*http.Request) (*http.Response, error)

func (f auditMediaTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func auditMediaMetadata(data []byte) map[string]any {
	h := sha256.Sum256(data)
	return map[string]any{"id": "123", "messaging_product": "whatsapp", "url": metaMediaOrigin + "/whatsapp_business/attachments/?mid=123", "mime_type": "audio/ogg; codecs=opus", "file_size": len(data), "sha256": hex.EncodeToString(h[:])}
}

func TestAuditMetaMediaOriginAndIdentifier(t *testing.T) {
	for _, raw := range []string{
		"https://example.test/whatsapp_business/attachments/",
		"http://lookaside.fbsbx.com/whatsapp_business/attachments/",
		"https://lookaside.fbsbx.com.example.test/whatsapp_business/attachments/",
		"https://lookaside.fbsbx.com@127.0.0.1/whatsapp_business/attachments/",
		"https://user:pass@lookaside.fbsbx.com/whatsapp_business/attachments/",
		"https://lookaside.fbsbx.com:444/whatsapp_business/attachments/",
		"https://lookaside.fbsbx.com/other/",
		"https://lookaside.fbsbx.com/whatsapp_business/attachments/../other",
		"https://lookaside.fbsbx.com/whatsapp_business/attachments/#fragment",
	} {
		if allowedMetaMediaURL(raw) {
			t.Fatalf("unexpected media origin/path accepted: %s", raw)
		}
	}
	for _, id := range []string{"", "../me", "123?fields=token", "123/other", strings.Repeat("1", 65)} {
		client := &http.Client{Transport: auditMediaTransport(func(*http.Request) (*http.Response, error) {
			t.Error("invalid media ID caused an outbound request")
			return nil, io.EOF
		})}
		if _, _, err := downloadMetaVoice(context.Background(), client, "https://graph.facebook.com/v25.0/100/messages", "100", "synthetic-token", id); err == nil {
			t.Fatal("invalid media ID accepted")
		}
	}
}

func TestAuditMetaMediaChecksBothRequestsAndExactContent(t *testing.T) {
	data := []byte("synthetic voice bytes")
	for _, base64Digest := range []bool{false, true} {
		metadata := auditMediaMetadata(data)
		if base64Digest {
			h := sha256.Sum256(data)
			metadata["sha256"] = base64.StdEncoding.EncodeToString(h[:])
			metadata["file_size"] = "21"
		}
		calls := 0
		client := &http.Client{Transport: auditMediaTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("Authorization") != "Bearer synthetic-token" {
				t.Error("authentication header missing")
			}
			body := string(data)
			if calls == 1 {
				if r.URL.String() != "https://graph.facebook.com/v25.0/123?phone_number_id=100" {
					t.Errorf("wrong version/phone scope: %s", r.URL)
				}
				raw, _ := json.Marshal(metadata)
				body = string(raw)
			} else if !allowedMetaMediaURL(r.URL.String()) {
				t.Error("token sent outside media origin")
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
		})}
		got, mimeType, err := downloadMetaVoice(context.Background(), client, "https://graph.facebook.com/v25.0/100/messages", "100", "synthetic-token", "123")
		if err != nil || string(got) != string(data) || mimeType != "audio/ogg" || calls != 2 {
			t.Fatalf("valid audio rejected: %s %s %d %v", got, mimeType, calls, err)
		}
	}
}

func TestAuditMetaMediaRejectsBadEvidenceBeforeDownload(t *testing.T) {
	for _, mutation := range []struct {
		field string
		value any
	}{
		{"id", "999"}, {"messaging_product", "other"}, {"url", "https://example.test/private"},
		{"file_size", maxMetaVoiceBytes + 1}, {"file_size", 0}, {"file_size", "invalid"},
		{"mime_type", "text/html"}, {"sha256", "invalid"},
	} {
		metadata := auditMediaMetadata([]byte("voice"))
		metadata[mutation.field] = mutation.value
		calls := 0
		client := &http.Client{Transport: auditMediaTransport(func(*http.Request) (*http.Response, error) {
			calls++
			raw, _ := json.Marshal(metadata)
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
		})}
		if _, _, err := downloadMetaVoice(context.Background(), client, "https://graph.facebook.com/v25.0/100/messages", "100", "synthetic-token", "123"); err == nil || calls != 1 {
			t.Fatalf("invalid %s accepted or downloaded: %v calls=%d", mutation.field, err, calls)
		}
	}
}

func TestAuditMetaMediaRejectsTruncationAndAlteration(t *testing.T) {
	for _, body := range []string{"", "other", strings.Repeat("x", maxMetaVoiceBytes+1)} {
		calls := 0
		client := &http.Client{Transport: auditMediaTransport(func(*http.Request) (*http.Response, error) {
			calls++
			result := body
			if calls == 1 {
				raw, _ := json.Marshal(auditMediaMetadata([]byte("voice")))
				result = string(raw)
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(result))}, nil
		})}
		if got, _, err := downloadMetaVoice(context.Background(), client, "https://graph.facebook.com/v25.0/100/messages", "100", "synthetic-token", "123"); err == nil || got != nil {
			t.Fatal("truncated, altered or oversized audio accepted")
		}
	}
}

func TestAuditMetaMediaNeverFollowsRedirects(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: auditMediaTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			raw, _ := json.Marshal(auditMediaMetadata([]byte("voice")))
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: http.Header{}}, nil
		}
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": {"https://example.test/private"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	})}
	if _, _, err := downloadMetaVoice(context.Background(), client, "https://graph.facebook.com/v25.0/100/messages", "100", "synthetic-token", "123"); err == nil || calls != 2 {
		t.Fatalf("redirect followed or accepted: %v calls=%d", err, calls)
	}
}

type auditTrackedMediaBody struct {
	io.Reader
	closed int
}

func (b *auditTrackedMediaBody) Close() error { b.closed++; return nil }

func TestAuditMetaMediaClosesEachBodyAndBoundsMetadata(t *testing.T) {
	for _, oversized := range []bool{false, true} {
		raw, _ := json.Marshal(auditMediaMetadata([]byte("voice")))
		if oversized {
			raw = []byte(strings.Repeat("x", maxMetaMetadataBytes+1))
		}
		metadata := &auditTrackedMediaBody{Reader: strings.NewReader(string(raw))}
		media := &auditTrackedMediaBody{Reader: strings.NewReader("voice")}
		calls := 0
		client := &http.Client{Transport: auditMediaTransport(func(*http.Request) (*http.Response, error) {
			calls++
			body := metadata
			if calls == 2 {
				body = media
			}
			return &http.Response{StatusCode: 200, Body: body}, nil
		})}
		_, _, err := downloadMetaVoice(context.Background(), client, "https://graph.facebook.com/v25.0/100/messages", "100", "synthetic-token", "123")
		if (err != nil) != oversized || metadata.closed != 1 {
			t.Fatalf("metadata lifetime or validation: closed=%d error=%v", metadata.closed, err)
		}
		if oversized && (calls != 1 || media.closed != 0) {
			t.Fatal("oversized metadata reached media request")
		}
		if !oversized && (calls != 2 || media.closed != 1) {
			t.Fatal("media response not closed exactly once")
		}
	}
}
