package platformsettings

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCryptoEncryptionDecryptionAndMasking(t *testing.T) {
	enc := NewEncryptor("test-secret-encryption-key-for-unit-tests-32")
	secret := "sk_live_1234567890abcdef"

	ciphertext, err := enc.Encrypt(secret)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if ciphertext == secret {
		t.Fatal("ciphertext must not match plaintext")
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if decrypted != secret {
		t.Fatalf("got %q want %q", decrypted, secret)
	}

	masked := MaskSecret(secret)
	if !strings.HasPrefix(masked, "sk_live_") || !strings.HasSuffix(masked, "cdef") || strings.Contains(masked, "1234567890") {
		t.Fatalf("unexpected mask: %s", masked)
	}

	fingerprint := SecretFingerprint(secret)
	if len(fingerprint) != 16 {
		t.Fatalf("expected 16-hex char fingerprint, got %s", fingerprint)
	}
}

func TestRegistryValidationRules(t *testing.T) {
	// Valid launch.mode
	meta, err := ValidateKeyAndValue("launch.mode", json.RawMessage(`"public_launch"`))
	if err != nil || meta.Category != CategoryLaunch {
		t.Fatalf("valid launch.mode failed: %v", err)
	}

	// Invalid launch.mode
	_, err = ValidateKeyAndValue("launch.mode", json.RawMessage(`"super_launch"`))
	if err == nil {
		t.Fatal("invalid launch.mode should fail")
	}

	// Valid boolean
	_, err = ValidateKeyAndValue("features.trade_lines", json.RawMessage(`true`))
	if err != nil {
		t.Fatalf("valid bool failed: %v", err)
	}

	// Invalid boolean
	_, err = ValidateKeyAndValue("features.trade_lines", json.RawMessage(`"yes"`))
	if err == nil {
		t.Fatal("string should fail for bool setting")
	}

	// Range checks
	_, err = ValidateKeyAndValue("security.session_idle_minutes", json.RawMessage(`2`))
	if err == nil {
		t.Fatal("session timeout < 5 should fail")
	}
	_, err = ValidateKeyAndValue("security.session_idle_minutes", json.RawMessage(`30`))
	if err != nil {
		t.Fatalf("valid session timeout failed: %v", err)
	}

	// Integration status
	_, err = ValidateKeyAndValue("integrations.mono.status", json.RawMessage(`"configured"`))
	if err != nil {
		t.Fatalf("valid status failed: %v", err)
	}
	_, err = ValidateKeyAndValue("integrations.mono.status", json.RawMessage(`"active"`))
	if err == nil {
		t.Fatal("status 'active' should fail (must be verified, configured, etc.)")
	}
}

func TestPostgresStoreIntegration(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" || os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("KREDIT_INTEGRATION=1 and DATABASE_URL required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	enc := NewEncryptor("test-platform-settings-key")
	store := NewPostgresStore(pool, enc, nil)

	// Test GetAll contains seeded baseline settings
	all, err := store.GetAll(ctx, false)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(all) < 10 {
		t.Fatalf("expected seeded settings, got %d", len(all))
	}

	// Test GetBool
	tradeLines := store.GetBool(ctx, "features.trade_lines", true)
	if tradeLines {
		t.Fatalf("expected features.trade_lines default false, got %v", tradeLines)
	}

	// Test Update
	updated, err := store.Update(ctx, "", "features.trade_lines", json.RawMessage(`true`), "Testing feature toggle")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if string(updated.Value) != "true" {
		t.Fatalf("got %s, want true", string(updated.Value))
	}
	if !store.GetBool(ctx, "features.trade_lines", false) {
		t.Fatal("expected GetBool to reflect update")
	}

	// Reset features.trade_lines
	_, _ = store.Update(ctx, "", "features.trade_lines", json.RawMessage(`false`), "Reset after test")

	// Test Secret Rotation and Masking
	secretVal := "sk_live_verysecretkey1234"
	rot, err := store.RotateSecret(ctx, "", "integrations.mono.secret_key", secretVal, "Rotating Mono secret")
	if err != nil {
		t.Fatalf("RotateSecret failed: %v", err)
	}
	if !strings.Contains(string(rot.Value), "sk_live_") || strings.Contains(string(rot.Value), "verysecretkey") {
		t.Fatalf("secret was not masked on return: %s", string(rot.Value))
	}

	// Get with secret decrypted
	unmasked, err := store.Get(ctx, "integrations.mono.secret_key", true)
	if err != nil {
		t.Fatalf("Get decrypted failed: %v", err)
	}
	var decrypted string
	_ = json.Unmarshal(unmasked.Value, &decrypted)
	if decrypted != secretVal {
		t.Fatalf("got decrypted %q want %q", decrypted, secretVal)
	}

	// Test History
	history, err := store.GetHistory(ctx, "integrations.mono.secret_key", 10, 0)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(history) == 0 || history[0].Action != "secret_rotated" {
		t.Fatalf("unexpected history: %+v", history)
	}

	// Test Governance Mode
	gov, err := store.GetGovernance(ctx)
	if err != nil {
		t.Fatalf("GetGovernance failed: %v", err)
	}
	if gov.Mode != GovernanceSoloOwner && gov.Mode != GovernanceDelegatedTeam {
		t.Fatalf("unexpected governance mode: %s", gov.Mode)
	}

	newGov, err := store.SetGovernance(ctx, "", GovernanceDelegatedTeam, "Switching to team mode for audit")
	if err != nil {
		t.Fatalf("SetGovernance failed: %v", err)
	}
	if newGov.Mode != GovernanceDelegatedTeam {
		t.Fatalf("got %s want delegated_team", newGov.Mode)
	}

	// Restore solo_owner mode
	_, _ = store.SetGovernance(ctx, "", GovernanceSoloOwner, "Reset to solo_owner")
}
