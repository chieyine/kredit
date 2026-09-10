package documents

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDocumentRequiresCleanScanBeforeDownload(t *testing.T) {
	objects := NewMemoryObjectStore()
	store := NewStore(objects)
	doc, err := store.Add(context.Background(), "org-1", "user-1", "invoice", "invoice.pdf", "application/pdf", "financial", 3, bytes.NewReader([]byte("pdf")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SignedDownload(context.Background(), doc.ID, 60); err == nil {
		t.Fatal("expected pending scan to block download")
	}
	if _, err := store.CompleteScan(doc.ID, ScanClean); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SignedDownload(context.Background(), doc.ID, 60); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentUploadSlotsRespectPerUserQuota(t *testing.T) {
	store := NewStore(NewMemoryObjectStore())
	var wait sync.WaitGroup
	var mu sync.Mutex
	succeeded := 0
	for i := 0; i < 40; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, _, err := store.CreateUpload(context.Background(), "org-1", "user-1", "evidence", "proof.png", "image/png", "dispute", 1024, time.Hour); err == nil {
				mu.Lock()
				succeeded++
				mu.Unlock()
			}
		}()
	}
	wait.Wait()
	if succeeded != 20 {
		t.Fatalf("concurrent quota admitted %d slots, want 20", succeeded)
	}
}

func TestDirectUploadSlotCreatesQuarantinedMetadata(t *testing.T) {
	store := NewStore(NewMemoryObjectStore())
	doc, url, err := store.CreateUpload(context.Background(), "org-1", "user-1", "evidence", "proof.png", "image/png", "dispute", 1024, 60)
	if err != nil {
		t.Fatal(err)
	}
	if doc.ScanState != ScanPending || doc.ObjectKey == "" || url == "" {
		t.Fatalf("unexpected upload slot: %+v %q", doc, url)
	}
}

func TestDocumentReadsKeepTenantAndOutageOutcomesDistinct(t *testing.T) {
	ctx := context.Background()
	s := NewStore(NewMemoryObjectStore())
	doc, err := s.Add(ctx, "org", "actor", "invoice", "invoice.pdf", "application/pdf", "financial", 3, bytes.NewReader([]byte("pdf")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReadForTenant(ctx, doc.ID, "actor", "other-org"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-organization read: %v", err)
	}
	if _, err = s.ReadForTenant(ctx, doc.ID, "other-actor", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-actor read: %v", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err = s.ReadForTenant(cancelled, doc.ID, "actor", "org"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation hidden: %v", err)
	}
	if _, err = s.SignedDownloadForTenant(ctx, doc.ID, "actor", "org", time.Minute); !errors.Is(err, ErrScanNotClean) {
		t.Fatalf("scan boundary lost: %v", err)
	}
}
func TestDocumentMetadataIsValidatedBeforeCreatingObject(t *testing.T) {
	objects := NewMemoryObjectStore()
	s := NewStore(objects)
	for _, metadata := range []struct{ purpose, name, retention string }{{"../other-organization", "invoice.pdf", "financial"}, {"invoice", "receipt\nInjected.pdf", "financial"}, {"invoice", strings.Repeat("x", 256), "financial"}, {"invoice", "invoice.pdf", ""}} {
		if _, err := s.Add(context.Background(), "org", "actor", metadata.purpose, metadata.name, "application/pdf", metadata.retention, 3, bytes.NewReader([]byte("pdf"))); err == nil {
			t.Fatalf("invalid metadata accepted: %+v", metadata)
		}
	}
	if len(objects.objects) != 0 {
		t.Fatal("invalid metadata wrote an object")
	}
}

func TestDirectUploadRecordsStoredContentDigest(t *testing.T) {
	objects := NewMemoryObjectStore()
	store := NewStore(objects)
	ctx := context.Background()
	doc, _, err := store.CreateUpload(ctx, "org-1", "user-1", "invoice", "direct.pdf", "application/pdf", "financial", 3, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err = objects.Put(ctx, doc.ObjectKey, strings.NewReader("pdf"), 3, "application/pdf"); err != nil {
		t.Fatal(err)
	}
	completed, err := store.CompleteUploadForTenant(ctx, doc.ID, "user-1", "org-1")
	if err != nil || len(completed.SHA256) != 64 || completed.ScanState != ScanPending {
		t.Fatalf("invalid completion: %+v %v", completed, err)
	}
	again, err := store.CompleteUploadForTenant(ctx, doc.ID, "user-1", "org-1")
	if err != nil || again.SHA256 != completed.SHA256 {
		t.Fatalf("retry changed digest: %+v %v", again, err)
	}
}
