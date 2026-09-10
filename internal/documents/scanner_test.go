package documents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
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

type blockingScanner struct {
	started chan struct{}
	finish  chan struct{}
}

func (b blockingScanner) Scan(ctx context.Context, _ Document, _ string) (ScanState, error) {
	close(b.started)
	select {
	case <-b.finish:
		return ScanClean, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
func TestScanClaimsBeforeCallingProvider(t *testing.T) {
	store := NewStore(NewMemoryObjectStore())
	doc, err := store.Add(context.Background(), "org", "actor", "evidence", "proof.pdf", "application/pdf", "dispute", 4, bytes.NewReader([]byte("safe")))
	if err != nil {
		t.Fatal(err)
	}
	scanner := blockingScanner{make(chan struct{}), make(chan struct{})}
	done := make(chan error, 1)
	go func() { _, err := store.Scan(context.Background(), doc.ID, scanner); done <- err }()
	<-scanner.started
	_, duplicateErr := store.Scan(context.Background(), doc.ID, CleanDevelopmentScanner{})
	close(scanner.finish)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if duplicateErr == nil {
		t.Fatal("another worker borrowed the active scan claim")
	}
}

type failingScanner struct{ calls int }

func (s *failingScanner) Scan(context.Context, Document, string) (ScanState, error) {
	s.calls++
	return "", errors.New("temporary scanner failure")
}

func TestExhaustedScansRemainBlockedAfterLeaseExpires(t *testing.T) {
	store := NewStore(NewMemoryObjectStore())
	now := time.Now()
	store.now = func() time.Time { return now }
	doc, err := store.Add(context.Background(), "org", "actor", "evidence", "proof.pdf", "application/pdf", "dispute", 4, strings.NewReader("safe"))
	if err != nil {
		t.Fatal(err)
	}
	scanner := &failingScanner{}
	for range 6 {
		if _, err := store.Scan(context.Background(), doc.ID, scanner); err == nil {
			t.Fatal("failed or exhausted scan reported success")
		}
		now = now.Add(6 * time.Minute)
	}
	if scanner.calls != 5 {
		t.Fatalf("expected bounded scanner attempts, got %d", scanner.calls)
	}
	if _, err := store.SignedDownload(context.Background(), doc.ID, time.Minute); !errors.Is(err, ErrScanNotClean) {
		t.Fatalf("exhausted document became downloadable: %v", err)
	}
}
