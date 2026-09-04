package documents

import (
	"bytes"
	"context"
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
