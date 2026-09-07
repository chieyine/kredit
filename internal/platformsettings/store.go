package platformsettings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service interface {
	GetAll(ctx context.Context, includeSecrets bool) ([]Setting, error)
	Get(ctx context.Context, key string, includeSecret bool) (Setting, error)
	GetBool(ctx context.Context, key string, defaultVal bool) bool
	GetString(ctx context.Context, key string, defaultVal string) string
	GetInt(ctx context.Context, key string, defaultVal int) int
	GetInt64(ctx context.Context, key string, defaultVal int64) int64
	Update(ctx context.Context, actorID string, key string, rawValue json.RawMessage, reason string) (Setting, error)
	RotateSecret(ctx context.Context, actorID string, key string, plaintextSecret string, reason string) (Setting, error)
	GetGovernance(ctx context.Context) (Governance, error)
	SetGovernance(ctx context.Context, actorID string, mode string, reason string) (Governance, error)
	GetHistory(ctx context.Context, key string, limit, offset int) ([]SettingHistory, error)
}

type PostgresStore struct {
	pool       *pgxpool.Pool
	enc        *Encryptor
	cacheMu    sync.RWMutex
	cache      map[string]Setting
	cacheTime  time.Time
	cacheTTL   time.Duration
	invalidate func(string)
}

func NewPostgresStore(pool *pgxpool.Pool, enc *Encryptor, invalidate func(string)) *PostgresStore {
	if enc == nil {
		enc = NewEncryptor("")
	}
	return &PostgresStore{
		pool:       pool,
		enc:        enc,
		cache:      make(map[string]Setting),
		cacheTTL:   30 * time.Second,
		invalidate: invalidate,
	}
}

func (s *PostgresStore) GetAll(ctx context.Context, includeSecrets bool) ([]Setting, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT key, category, value, is_secret, COALESCE(secret_fingerprint, ''), description, version, updated_at, COALESCE(updated_by::text, ''), COALESCE(reason, '')
		FROM app.platform_settings
		ORDER BY category, key
	`)
	if err != nil {
		return nil, fmt.Errorf("query settings: %w", err)
	}
	defer rows.Close()

	var settings []Setting
	for rows.Next() {
		var item Setting
		var rawVal []byte
		if err := rows.Scan(
			&item.Key, &item.Category, &rawVal, &item.IsSecret,
			&item.SecretFingerprint, &item.Description, &item.Version,
			&item.UpdatedAt, &item.UpdatedBy, &item.Reason,
		); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		item.Value = json.RawMessage(rawVal)
		if item.IsSecret {
			if includeSecrets {
				var ciphertext string
				_ = json.Unmarshal(item.Value, &ciphertext)
				plain, err := s.enc.Decrypt(ciphertext)
				if err == nil {
					b, _ := json.Marshal(plain)
					item.Value = b
				}
			} else {
				var ciphertext string
				_ = json.Unmarshal(item.Value, &ciphertext)
				plain, err := s.enc.Decrypt(ciphertext)
				masked := MaskSecret(plain)
				if err != nil || plain == "" {
					masked = ""
				}
				b, _ := json.Marshal(masked)
				item.Value = b
			}
		}
		settings = append(settings, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *PostgresStore) Get(ctx context.Context, key string, includeSecret bool) (Setting, error) {
	var item Setting
	var rawVal []byte
	err := s.pool.QueryRow(ctx, `
		SELECT key, category, value, is_secret, COALESCE(secret_fingerprint, ''), description, version, updated_at, COALESCE(updated_by::text, ''), COALESCE(reason, '')
		FROM app.platform_settings
		WHERE key = $1
	`, key).Scan(
		&item.Key, &item.Category, &rawVal, &item.IsSecret,
		&item.SecretFingerprint, &item.Description, &item.Version,
		&item.UpdatedAt, &item.UpdatedBy, &item.Reason,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Setting{}, fmt.Errorf("setting %q not found", key)
		}
		return Setting{}, err
	}
	item.Value = json.RawMessage(rawVal)
	if item.IsSecret {
		var ciphertext string
		_ = json.Unmarshal(item.Value, &ciphertext)
		plain, err := s.enc.Decrypt(ciphertext)
		if includeSecret {
			if err == nil {
				b, _ := json.Marshal(plain)
				item.Value = b
			}
		} else {
			masked := MaskSecret(plain)
			if err != nil || plain == "" {
				masked = ""
			}
			b, _ := json.Marshal(masked)
			item.Value = b
		}
	}
	return item, nil
}

func (s *PostgresStore) GetBool(ctx context.Context, key string, defaultVal bool) bool {
	item, err := s.Get(ctx, key, false)
	if err != nil {
		return defaultVal
	}
	var b bool
	if err := json.Unmarshal(item.Value, &b); err != nil {
		return defaultVal
	}
	return b
}

func (s *PostgresStore) GetString(ctx context.Context, key string, defaultVal string) string {
	item, err := s.Get(ctx, key, false)
	if err != nil {
		return defaultVal
	}
	var str string
	if err := json.Unmarshal(item.Value, &str); err != nil {
		return defaultVal
	}
	return str
}

func (s *PostgresStore) GetInt(ctx context.Context, key string, defaultVal int) int {
	item, err := s.Get(ctx, key, false)
	if err != nil {
		return defaultVal
	}
	var n int
	if err := json.Unmarshal(item.Value, &n); err != nil {
		return defaultVal
	}
	return n
}

func (s *PostgresStore) GetInt64(ctx context.Context, key string, defaultVal int64) int64 {
	item, err := s.Get(ctx, key, false)
	if err != nil {
		return defaultVal
	}
	var n int64
	if err := json.Unmarshal(item.Value, &n); err != nil {
		return defaultVal
	}
	return n
}

func (s *PostgresStore) Update(ctx context.Context, actorID string, key string, rawValue json.RawMessage, reason string) (Setting, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) < 4 {
		return Setting{}, errors.New("a reason of at least 4 characters is required")
	}

	meta, err := ValidateKeyAndValue(key, rawValue)
	if err != nil {
		return Setting{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Setting{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current Setting
	var currentRaw []byte
	var isSecret bool
	var currentFingerprint string
	var currentVersion int
	var currentDesc string

	err = tx.QueryRow(ctx, `
		SELECT key, category, value, is_secret, COALESCE(secret_fingerprint, ''), description, version
		FROM app.platform_settings
		WHERE key = $1
		FOR UPDATE
	`, key).Scan(&current.Key, &current.Category, &currentRaw, &isSecret, &currentFingerprint, &currentDesc, &currentVersion)

	exists := (err == nil)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Setting{}, err
	}

	if exists {
		current.Value = json.RawMessage(currentRaw)
	}

	storedValue := rawValue
	fingerprint := currentFingerprint
	action := "create"
	newVersion := 1

	if exists {
		newVersion = currentVersion + 1
		action = "update"
	}

	if meta.IsSecret || isSecret {
		// If secret is updated via raw value, encrypt it
		var plaintext string
		if err := json.Unmarshal(rawValue, &plaintext); err != nil {
			return Setting{}, errors.New("secret value must be a string")
		}
		cipherText, err := s.enc.Encrypt(plaintext)
		if err != nil {
			return Setting{}, fmt.Errorf("encrypt secret: %w", err)
		}
		b, _ := json.Marshal(cipherText)
		storedValue = b
		fingerprint = SecretFingerprint(plaintext)
		action = "secret_rotated"
	}

	var updated Setting
	var scannedRaw []byte
	err = tx.QueryRow(ctx, `
		INSERT INTO app.platform_settings (key, category, value, is_secret, secret_fingerprint, description, version, updated_at, updated_by, reason)
		VALUES ($1, $2, $3::jsonb, $4, NULLIF($5, ''), $6, $7, clock_timestamp(), NULLIF($8, '')::uuid, $9)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			is_secret = EXCLUDED.is_secret,
			secret_fingerprint = EXCLUDED.secret_fingerprint,
			version = app.platform_settings.version + 1,
			updated_at = clock_timestamp(),
			updated_by = EXCLUDED.updated_by,
			reason = EXCLUDED.reason
		RETURNING key, category, value, is_secret, COALESCE(secret_fingerprint, ''), description, version, updated_at, COALESCE(updated_by::text, ''), COALESCE(reason, '')
	`, key, meta.Category, string(storedValue), meta.IsSecret || isSecret, fingerprint, meta.Description, newVersion, actorID, reason).Scan(
		&updated.Key, &updated.Category, &scannedRaw, &updated.IsSecret,
		&updated.SecretFingerprint, &updated.Description, &updated.Version,
		&updated.UpdatedAt, &updated.UpdatedBy, &updated.Reason,
	)
	if err != nil {
		return Setting{}, fmt.Errorf("save setting: %w", err)
	}
	updated.Value = json.RawMessage(scannedRaw)

	// Record history
	var oldValAny any
	if exists {
		oldValAny = current.Value
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO app.platform_settings_history (key, old_value, new_value, version, action, actor_id, reason)
		VALUES ($1, $2::jsonb, $3::jsonb, $4, $5, NULLIF($6, '')::uuid, $7)
	`, key, oldValAny, updated.Value, updated.Version, action, actorID, reason)
	if err != nil {
		return Setting{}, fmt.Errorf("save history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Setting{}, err
	}

	if s.invalidate != nil {
		s.invalidate(key)
	}

	// Return with masked secret if secret
	if updated.IsSecret {
		var ciphertext string
		_ = json.Unmarshal(updated.Value, &ciphertext)
		plain, _ := s.enc.Decrypt(ciphertext)
		b, _ := json.Marshal(MaskSecret(plain))
		updated.Value = b
	}

	return updated, nil
}

func (s *PostgresStore) RotateSecret(ctx context.Context, actorID string, key string, plaintextSecret string, reason string) (Setting, error) {
	b, err := json.Marshal(plaintextSecret)
	if err != nil {
		return Setting{}, err
	}
	return s.Update(ctx, actorID, key, b, reason)
}

func (s *PostgresStore) GetGovernance(ctx context.Context) (Governance, error) {
	var g Governance
	err := s.pool.QueryRow(ctx, `
		SELECT mode, updated_at, COALESCE(updated_by::text, ''), COALESCE(reason, '')
		FROM app.platform_governance
		WHERE id = 'singleton'
	`).Scan(&g.Mode, &g.UpdatedAt, &g.UpdatedBy, &g.Reason)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Governance{Mode: GovernanceSoloOwner, Reason: "Default configuration"}, nil
		}
		return Governance{}, err
	}
	return g, nil
}

func (s *PostgresStore) SetGovernance(ctx context.Context, actorID string, mode string, reason string) (Governance, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) < 4 {
		return Governance{}, errors.New("a reason of at least 4 characters is required")
	}
	if mode != GovernanceSoloOwner && mode != GovernanceDelegatedTeam {
		return Governance{}, fmt.Errorf("invalid governance mode %q (must be solo_owner or delegated_team)", mode)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Governance{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var g Governance
	err = tx.QueryRow(ctx, `
		INSERT INTO app.platform_governance (id, mode, updated_at, updated_by, reason)
		VALUES ('singleton', $1, clock_timestamp(), NULLIF($2, '')::uuid, $3)
		ON CONFLICT (id) DO UPDATE SET
			mode = EXCLUDED.mode,
			updated_at = clock_timestamp(),
			updated_by = EXCLUDED.updated_by,
			reason = EXCLUDED.reason
		RETURNING mode, updated_at, COALESCE(updated_by::text, ''), COALESCE(reason, '')
	`, mode, actorID, reason).Scan(&g.Mode, &g.UpdatedAt, &g.UpdatedBy, &g.Reason)
	if err != nil {
		return Governance{}, fmt.Errorf("update governance: %w", err)
	}

	// Also record in settings history
	newVal, _ := json.Marshal(mode)
	_, err = tx.Exec(ctx, `
		INSERT INTO app.platform_settings_history (key, old_value, new_value, version, action, actor_id, reason)
		VALUES ('governance.mode', NULL, $1::jsonb, 1, 'update', NULLIF($2, '')::uuid, $3)
	`, newVal, actorID, reason)
	if err != nil {
		return Governance{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Governance{}, err
	}

	return g, nil
}

func (s *PostgresStore) GetHistory(ctx context.Context, key string, limit, offset int) ([]SettingHistory, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id::text, key, old_value, new_value, version, action, COALESCE(actor_id::text, ''), reason, recorded_at
		FROM app.platform_settings_history
	`
	var args []any
	if key != "" {
		query += ` WHERE key = $1 ORDER BY recorded_at DESC, version DESC LIMIT $2 OFFSET $3`
		args = append(args, key, limit, offset)
	} else {
		query += ` ORDER BY recorded_at DESC, version DESC LIMIT $1 OFFSET $2`
		args = append(args, limit, offset)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var list []SettingHistory
	for rows.Next() {
		var h SettingHistory
		var oldRaw, newRaw []byte
		if err := rows.Scan(&h.ID, &h.Key, &oldRaw, &newRaw, &h.Version, &h.Action, &h.ActorID, &h.Reason, &h.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		if len(oldRaw) > 0 {
			h.OldValue = json.RawMessage(oldRaw)
		}
		h.NewValue = json.RawMessage(newRaw)
		list = append(list, h)
	}
	return list, rows.Err()
}
