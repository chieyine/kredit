package platformsettings

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCryptoEncryptionDecryptionAndMasking(t *testing.T) {
	const settingKey = "integrations.mono.secret_key"
	enc := NewEncryptor("test-secret-encryption-key-for-unit-tests-32")
	secret := "sk_live_1234567890abcdef"

	ciphertext, err := enc.Encrypt(settingKey, secret)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if ciphertext == secret {
		t.Fatal("ciphertext must not match plaintext")
	}

	decrypted, err := enc.Decrypt(settingKey, ciphertext)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if decrypted != secret {
		t.Fatalf("got %q want %q", decrypted, secret)
	}

	// A ciphertext must not open against a different setting: without that
	// binding, a value copied between rows would swap one credential for
	// another and decrypt cleanly.
	if _, err := enc.Decrypt("integrations.paystack.secret_key", ciphertext); err == nil {
		t.Fatal("ciphertext opened against the wrong setting key")
	}

	// A different root key must be reported as such, not as corruption, so a
	// rotation is diagnosable.
	other := NewEncryptor("a-completely-different-root-key-for-tests-32")
	if _, err := other.Decrypt(settingKey, ciphertext); err == nil {
		t.Fatal("ciphertext opened under a different root key")
	}
	if enc.KeyID() == "" || enc.KeyID() == other.KeyID() {
		t.Fatalf("root keys must have distinct non-empty identities: %q vs %q", enc.KeyID(), other.KeyID())
	}

	masked := MaskSecret(secret)
	if masked != "sk_live_••••••••" {
		t.Fatalf("mask must reveal only the vendor prefix, got %s", masked)
	}
	if strings.Contains(masked, "cdef") || strings.Contains(masked, "1234567890") {
		t.Fatalf("mask leaked secret material: %s", masked)
	}

	fingerprint := enc.SecretFingerprint(secret)
	if len(fingerprint) != 16 {
		t.Fatalf("expected 16-hex char fingerprint, got %s", fingerprint)
	}
	if other.SecretFingerprint(secret) == fingerprint {
		t.Fatal("fingerprints must be keyed, so the same secret differs per deployment")
	}
}

func TestRegistryValidationRules(t *testing.T) {
	// The registry is the whole list. A key that is not here cannot be written,
	// and after migration 091 it is not stored either — which is the point: an
	// owner should never be shown a switch that moves nothing.
	if len(KnownSettings) != 13+len(WebsitePages) {
		t.Fatalf("the registry should hold only settings with a consumer, got %d: %v", len(KnownSettings), keys())
	}
	for _, key := range []string{"features.trade_lines", "features.drawdowns", "features.disputes", "features.system_acceptance"} {
		meta, err := ValidateKeyAndValue(key, json.RawMessage(`true`))
		if err != nil {
			t.Fatalf("%s should validate: %v", key, err)
		}
		if meta.Category != CategoryFeatures {
			t.Fatalf("%s category: got %q want %q", key, meta.Category, CategoryFeatures)
		}
		if meta.Description == "" {
			t.Fatalf("%s has no description; the console would show a bare key", key)
		}
		if _, err := ValidateKeyAndValue(key, json.RawMessage(`"yes"`)); err == nil {
			t.Fatalf("%s accepted a string where a boolean is required", key)
		}
		if _, err := ValidateKeyAndValue(key, json.RawMessage(`null`)); err == nil {
			t.Fatalf("%s accepted null", key)
		}
	}

	// Retired keys stay refused. Every one of these was a real row once; a
	// rollback that reseeds them must not make them writable again.
	for _, key := range []string{
		"launch.mode", "launch.banner_enabled", "fees.late_fee_rate_bps",
		"security.session_idle_minutes", "integrations.paystack.secret_key",
		"integrations.termii.api_key", "integrations.resend.api_key",
		"features.notifications_whatsapp", "features.mono_direct_debit",
		"notifications.channels",
	} {
		if _, err := ValidateKeyAndValue(key, json.RawMessage(`true`)); err == nil {
			t.Fatalf("retired key %s is still writable", key)
		}
	}
}

func keys() []string {
	out := make([]string, 0, len(KnownSettings))
	for key := range KnownSettings {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
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

	enc := NewEncryptor("test-platform-settings-key-32-bytes-minimum")
	store := NewPostgresStore(pool, enc, nil)

	// Test GetAll contains seeded baseline settings
	all, err := store.GetAll(ctx, false)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(all) < 3 || len(all) > len(KnownSettings) {
		t.Fatalf("GetAll returned %d settings, registry has %d: the console must show the registry, not the table", len(all), len(KnownSettings))
	}

	// Test GetBool
	tradeLines := store.GetBool(ctx, "features.trade_lines", true)
	if tradeLines {
		t.Fatalf("expected features.trade_lines default false, got %v", tradeLines)
	}

	current, err := store.Get(ctx, "features.trade_lines", false)
	if err != nil {
		t.Fatal(err)
	}
	// Test Update
	updated, err := store.Update(ctx, "", "features.trade_lines", json.RawMessage(`true`), "Testing feature toggle", current.Version)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if string(updated.Value) != "true" {
		t.Fatalf("got %s, want true", string(updated.Value))
	}
	if !store.GetBool(ctx, "features.trade_lines", false) {
		t.Fatal("expected GetBool to reflect update")
	}

	if _, err := store.Update(ctx, "", "features.trade_lines", json.RawMessage(`false`), "Stale editor rejected", current.Version); err != ErrVersionConflict {
		t.Fatalf("stale version was not rejected: %v", err)
	}
	// Reset features.trade_lines
	_, _ = store.Update(ctx, "", "features.trade_lines", json.RawMessage(`false`), "Reset after test", updated.Version)

	// Connector credentials are encrypted at rest and never returned in history.
	key := "integrations.notifications.email"
	version := 0
	if existing, readErr := store.Get(ctx, key, false); readErr == nil {
		version = existing.Version
	}
	configJSON, _ := json.Marshal(NotificationConnector{Enabled: true, Endpoint: "https://connector.example/send", Token: "private-integration-test-token"})
	raw, _ := json.Marshal(string(configJSON))
	saved, err := store.Update(ctx, "", key, raw, "Configure test connector", version)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(saved.Value), "private-integration") {
		t.Fatal("save response leaked credentials")
	}
	var persisted string
	if err := pool.QueryRow(ctx, `SELECT value::text FROM app.platform_settings WHERE key=$1`, key).Scan(&persisted); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(persisted, "private-integration") || strings.Contains(persisted, "connector.example") {
		t.Fatal("configuration persisted without encryption")
	}
	opened, err := store.Get(ctx, key, true)
	if err != nil || string(opened.Value) != string(raw) {
		t.Fatalf("runtime cannot read saved configuration: %v", err)
	}
	secretHistory, err := store.GetHistory(ctx, key, 10, 0)
	if err != nil || len(secretHistory) == 0 {
		t.Fatalf("missing connector history: %v", err)
	}
	for _, item := range secretHistory {
		if string(item.NewValue) != "null" || item.OldValue != nil {
			t.Fatal("history exposed secret payload")
		}
	}
	disabled, _ := json.Marshal(`{"enabled":false}`)
	if _, err := store.Update(ctx, "", key, disabled, "Disable test connector", saved.Version); err != nil {
		t.Fatal(err)
	}

	history, err := store.GetHistory(ctx, "features.trade_lines", 10, 0)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(history) == 0 {
		t.Fatal("an update must leave a history row")
	}
	if _, err := pool.Exec(ctx, `UPDATE app.platform_settings_history SET reason='rewrite' WHERE key='features.trade_lines'`); err == nil {
		t.Fatal("history mutation allowed")
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
	governanceHistory, err := store.GetHistory(ctx, "governance.mode", 1, 0)
	if err != nil || len(governanceHistory) != 1 {
		t.Fatalf("governance history unavailable: %v", err)
	}
	var beforeMode, afterMode string
	if err := json.Unmarshal(governanceHistory[0].OldValue, &beforeMode); err != nil || beforeMode != gov.Mode {
		t.Fatalf("previous governance mode missing: %q %v", beforeMode, err)
	}
	if err := json.Unmarshal(governanceHistory[0].NewValue, &afterMode); err != nil || afterMode != GovernanceDelegatedTeam {
		t.Fatalf("new governance mode missing: %q %v", afterMode, err)
	}

	// Restore the original approval rule and verify a distinct history version.
	if _, err := store.SetGovernance(ctx, "", gov.Mode, "Restore original approval rule"); err != nil {
		t.Fatal(err)
	}
	restoredHistory, err := store.GetHistory(ctx, "governance.mode", 1, 0)
	if err != nil || len(restoredHistory) != 1 || restoredHistory[0].Version != governanceHistory[0].Version+1 {
		t.Fatalf("governance versions did not advance: %+v %v", restoredHistory, err)
	}
}

func TestValidatedRuntimeUpdateRollsBackOnValidationFailure(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := NewPostgresStore(pool, NewEncryptor("runtime-validation-test-root-0123456789abcdef"), nil)
	key := "integrations.runtime.scanner"
	var before int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app.platform_settings_history WHERE key=$1`, key).Scan(&before); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(`{"DocumentScannerEnabled":false,"DocumentScannerEndpoint":"","DocumentScannerToken":""}`)
	rejected := errors.New("related settings reject this candidate")
	_, err = store.UpdateValidated(ctx, "", key, encoded, "Rejected candidate must not persist", 0, func(snapshot Service, _ json.RawMessage) (json.RawMessage, error) {
		if _, err := snapshot.Get(ctx, "features.trade_lines", false); err != nil {
			t.Fatal(err)
		}
		return nil, rejected
	})
	if !errors.Is(err, rejected) {
		t.Fatalf("expected validation rejection, got %v", err)
	}
	var after int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app.platform_settings_history WHERE key=$1`, key).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("rejected connection changed immutable history")
	}
}
