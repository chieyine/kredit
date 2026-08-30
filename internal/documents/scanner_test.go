package documents

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDevelopmentScannerReleasesQuarantinedDocument(t *testing.T) {
	store := NewStore(NewMemoryObjectStore())
	document, err := store.Add(context.Background(), "org", "actor", "evidence", "proof.pdf", "application/pdf", "dispute", 4, bytes.NewReader([]byte("safe")))
	if err != nil {
		t.Fatal(err)
	}
	document, err = store.Scan(context.Background(), document.ID, CleanDevelopmentScanner{})
	if err != nil || document.ScanState != ScanClean {
		t.Fatalf("scan failed: %v %+v", err, document)
	}
	if _, err := store.SignedDownload(context.Background(), document.ID, 60); err != nil {
		t.Fatal(err)
	}
}

func TestWebhookScannerUsesStableIdempotencyKey(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer scanner-token" || r.Header.Get("Idempotency-Key") != "document-scan:doc-1:digest" {
			t.Fatalf("unexpected scanner headers")
		}
		body, _ := json.Marshal(map[string]string{"state": string(ScanClean)})
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header)}, nil
	})
	scanner, err := NewWebhookScanner("https://scanner.example/scan", "scanner-token")
	if err != nil {
		t.Fatal(err)
	}
	scanner.client.Transport = transport
	state, err := scanner.Scan(context.Background(), Document{ID: "doc-1", SHA256: "digest"}, "https://objects.example/document")
	if err != nil || state != ScanClean {
		t.Fatalf("scan failed: %v %s", err, state)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
