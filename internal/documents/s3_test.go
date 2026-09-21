package documents

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSignedUploadBindsWriteOnceAndEncryptionHeaders(t *testing.T) {
	store, err := NewS3ObjectStore(context.Background(), "https://objects.example.test", "us-east-1", "test-access", "test-secret", "documents")
	if err != nil {
		t.Fatal(err)
	}
	signed, err := store.SignedUploadURL(context.Background(), "invoice/original.pdf", 10*time.Minute, "application/pdf", 4096)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(signed)
	if err != nil {
		t.Fatal(err)
	}
	headers := ";" + parsed.Query().Get("X-Amz-SignedHeaders") + ";"
	for _, header := range []string{"content-length", "if-none-match", "x-amz-server-side-encryption"} {
		if !strings.Contains(headers, ";"+header+";") {
			t.Fatalf("upload can omit or change %s: %s", header, headers)
		}
	}
	if parsed.Query().Get("X-Amz-Expires") != "600" {
		t.Fatal("upload expiry was not bounded")
	}
	if _, err := store.SignedUploadURL(context.Background(), "invoice/invalid.pdf", time.Minute, "application/pdf", 0); err == nil {
		t.Fatal("zero-size direct upload was signed")
	}
	if _, err := store.SignedUploadURL(context.Background(), "invoice/oversize.pdf", time.Minute, "application/pdf", (25<<20)+1); err == nil {
		t.Fatal("oversized direct upload was signed")
	}
}
