package auth

import (
	"bytes"
	"testing"
)

func TestPostgresStoreEncryptsRecoverableTargets(t *testing.T) {
	store := NewPostgresStore(nil, "test-key")
	plaintext := []byte("owner@example.test")
	ciphertext, err := store.encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("ciphertext must not contain the plaintext target")
	}
	decoded, err := store.decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, plaintext) {
		t.Fatalf("decrypted target mismatch: got %q", decoded)
	}
	if _, err := store.decrypt(ciphertext[:len(ciphertext)-1]); err == nil {
		t.Fatal("truncated ciphertext should fail authentication")
	}
}

func TestPostgresStoreImplementsService(t *testing.T) {
	var _ Service = NewPostgresStore(nil, "test-key")
}
