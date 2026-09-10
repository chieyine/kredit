package documents

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"kredit/internal/db"
)

func TestPostgresStoreRoundTrip(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" || os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("KREDIT_INTEGRATION=1 and DATABASE_URL are required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var userID, organizationID string
	if err := pool.QueryRow(ctx, `INSERT INTO app.users (normalized_email) VALUES ($1) RETURNING id::text`, fmt.Sprintf("document-test-%d@example.test", time.Now().UnixNano())).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.organizations (legal_name, business_type, business_address, industry) VALUES ('Document Test', 'limited_company', 'test', 'test') RETURNING id::text`).Scan(&organizationID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.documents WHERE uploaded_by = $1::uuid`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.organizations WHERE id = $1::uuid`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.users WHERE id = $1::uuid`, userID)
	}()

	store := NewPostgresStore(pool, NewMemoryObjectStore())
	doc, err := store.Add(ctx, organizationID, userID, "invoice", "invoice.pdf", "application/pdf", "financial", 3, bytes.NewReader([]byte("pdf")))
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := store.Get(doc.ID); !ok || got.ID != doc.ID || got.ScanState != ScanPending {
		t.Fatalf("unexpected durable document: %+v %v", got, ok)
	}
	ids, err := store.PendingScanIDs(ctx, 10)
	found := false
	for _, id := range ids {
		found = found || id == doc.ID
	}
	if err != nil || !found {
		t.Fatalf("pending scan was not recovered: %v %v", err, ids)
	}
	if _, err := store.Scan(ctx, doc.ID, CleanDevelopmentScanner{}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SignedDownload(ctx, doc.ID, 60); err != nil {
		t.Fatal(err)
	}
	app, err := db.OpenAsRole(ctx, os.Getenv("APP_DATABASE_URL"), "kredit_app")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	objects := NewMemoryObjectStore()
	restricted := NewPostgresStore(app.Raw(), objects)
	slot, _, err := restricted.CreateUpload(ctx, organizationID, userID, "invoice", "direct.pdf", "application/pdf", "financial", 3, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err = objects.Put(ctx, slot.ObjectKey, bytes.NewBufferString("pdf"), 3, "application/pdf"); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("pdf"))
	expected := hex.EncodeToString(digest[:])
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			completed, err := restricted.CompleteUploadForTenant(ctx, slot.ID, userID, organizationID)
			if err != nil || completed.SHA256 != expected || completed.ScanState != ScanPending {
				t.Errorf("direct completion failed: %+v %v", completed, err)
			}
		}()
	}
	wg.Wait()
	reloaded, err := NewPostgresStore(app.Raw(), objects).ReadForTenant(ctx, slot.ID, userID, organizationID)
	if err != nil || reloaded.SHA256 != expected {
		t.Fatalf("checksum not durable: %+v %v", reloaded, err)
	}
	closed, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	closed.Close()
	unavailable := NewPostgresStore(closed, NewMemoryObjectStore())
	if _, err = unavailable.ReadForTenant(ctx, doc.ID, userID, organizationID); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("database outage reported as missing document: %v", err)
	}

}
