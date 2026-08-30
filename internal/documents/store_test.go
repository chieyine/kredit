package documents

import (
	"bytes"
	"context"
	"testing"
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
