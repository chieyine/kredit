package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service is the authentication boundary consumed by the HTTP layer. Both
// the deterministic development store and the PostgreSQL implementation
// satisfy it, allowing persistence to be introduced without changing API
// handlers.
type Service interface {
	RequestOTP(identifier, channel, purpose string) (OTPChallenge, string, error)
	VerifyOTP(challengeID, code, deviceLabel string) (User, Session, string, error)
	VerifyOTPForTarget(challengeID, code, deviceLabel, channel, identifier string) (User, Session, string, error)
	VerifyAndAttachIdentifier(userID, challengeID, code, channel, identifier string) error
	FindOrCreateUser(identifier, channel string) (User, error)
	UserByID(userID string) (User, error)
	SessionFromToken(token string) (Session, User, error)
	RevokeSession(token string) error
	RevokeAllSessions(userID string) error
	BeginTOTPEnrollment(userID string) (MFAMethod, error)
	VerifyTOTP(userID, code string) error
	StepUpSession(token, code string) (Session, string, error)
	IsMFAEnrolled(userID string) bool
}

func (s *PostgresStore) VerifyAndAttachIdentifier(userID, challengeID, code, channel, identifier string) error {
	identifier = NormalizeIdentifier(identifier)
	channel = strings.ToLower(strings.TrimSpace(channel))
	if userID == "" || identifier == "" || (channel != "email" && channel != "phone") {
		return errors.New("valid user and contact are required")
	}
	now := s.now()
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err = tx.Exec(context.Background(), `SELECT set_config('app.current_user_id',$1,true)`, userID); err != nil {
		return fmt.Errorf("set contact verification context: %w", err)
	}
	var targetType string
	var targetHash, codeHash []byte
	var attempts int
	var expires time.Time
	var consumed *time.Time
	if err = tx.QueryRow(context.Background(), `SELECT target_type,target_hash,code_hmac,attempt_count,expires_at,consumed_at FROM app.otp_challenges WHERE id=$1 FOR UPDATE`, challengeID).Scan(&targetType, &targetHash, &codeHash, &attempts, &expires, &consumed); err != nil {
		return errors.New("otp challenge is invalid or expired")
	}
	if consumed != nil || !now.Before(expires) || attempts >= 5 || targetType != channel || !equalBytes(targetHash, s.hashTargetBytes(channel, identifier)) {
		return errors.New("otp challenge is invalid or expired")
	}
	if _, err = tx.Exec(context.Background(), `UPDATE app.otp_challenges SET attempt_count=attempt_count+1 WHERE id=$1`, challengeID); err != nil {
		return err
	}
	if !hmacEqual(codeHash, s.hashCodeBytes(strings.TrimSpace(code))) {
		_ = tx.Commit(context.Background())
		return errors.New("otp code is invalid")
	}
	column := "normalized_email"
	if channel == "phone" {
		column = "normalized_phone"
	}
	var conflict bool
	if err = tx.QueryRow(context.Background(), `SELECT EXISTS(SELECT 1 FROM app.users WHERE `+column+`=$1 AND id<>$2)`, identifier, userID).Scan(&conflict); err != nil {
		return err
	}
	if conflict {
		return errors.New("contact is already attached to another account")
	}
	if _, err = tx.Exec(context.Background(), `UPDATE app.users SET `+column+`=$2 WHERE id=$1`, userID, identifier); err != nil {
		return fmt.Errorf("attach verified contact: %w", err)
	}
	if _, err = tx.Exec(context.Background(), `UPDATE app.otp_challenges SET consumed_at=$2 WHERE id=$1`, challengeID, now); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

var _ Service = (*Store)(nil)

// PostgresStore persists authentication state. OTP targets and TOTP secrets
// are encrypted with a key derived from the deployment token hash key; only
// their HMACs are used for lookup and uniqueness checks.
type PostgresStore struct {
	pool          *pgxpool.Pool
	tokenKey      []byte
	otpKey        []byte
	encryptionKey []byte
	now           func() time.Time
}

var _ Service = (*PostgresStore)(nil)

func (s *PostgresStore) UserByID(userID string) (User, error) {
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, errors.New("user not found")
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, userID); err != nil {
		return User{}, errors.New("user not found")
	}
	var user User
	var email, phone *string
	err = tx.QueryRow(ctx, `SELECT id::text,normalized_email,normalized_phone,COALESCE(display_name,''),status,created_at,COALESCE(last_authenticated_at,'epoch'::timestamptz) FROM app.users WHERE id=$1::uuid`, userID).Scan(&user.ID, &email, &phone, &user.DisplayName, &user.Status, &user.CreatedAt, &user.LastAuthenticatedAt)
	if err != nil {
		return User{}, errors.New("user not found")
	}
	if email != nil {
		user.Email = *email
	}
	if phone != nil {
		user.Phone = *phone
	}
	if err = tx.Commit(ctx); err != nil {
		return User{}, errors.New("user not found")
	}
	return user, nil
}

func NewPostgresStore(pool *pgxpool.Pool, tokenHashKey string) *PostgresStore {
	return NewPostgresStoreWithKeys(pool, tokenHashKey, tokenHashKey, tokenHashKey)
}

func NewPostgresStoreWithKeys(pool *pgxpool.Pool, tokenHashKey, otpHMACKey, fieldEncryptionKey string) *PostgresStore {
	if tokenHashKey == "" {
		tokenHashKey = "development-only-change-me"
	}
	if otpHMACKey == "" {
		otpHMACKey = "development-only-change-me"
	}
	if fieldEncryptionKey == "" {
		fieldEncryptionKey = "development-only-change-me"
	}
	derived := sha256.Sum256([]byte(fieldEncryptionKey))
	return &PostgresStore{pool: pool, tokenKey: []byte(tokenHashKey), otpKey: []byte(otpHMACKey), encryptionKey: derived[:], now: func() time.Time { return time.Now().UTC() }}
}

func (s *PostgresStore) RequestOTP(identifier, channel, purpose string) (OTPChallenge, string, error) {
	identifier = NormalizeIdentifier(identifier)
	channel = strings.ToLower(strings.TrimSpace(channel))
	purpose = strings.ToLower(strings.TrimSpace(purpose))
	if identifier == "" || (channel != "phone" && channel != "email") {
		return OTPChallenge{}, "", errors.New("valid identifier and channel are required")
	}
	if purpose == "" {
		purpose = "login"
	}
	now := s.now()
	targetHash := s.hashTargetBytes(channel, identifier)
	code, err := randomOTP()
	if err != nil {
		return OTPChallenge{}, "", err
	}
	targetCiphertext, err := s.encrypt([]byte(identifier))
	if err != nil {
		return OTPChallenge{}, "", err
	}
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return OTPChallenge{}, "", err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	// Serialize sends for the same target across API replicas so the cooldown
	// cannot be bypassed by concurrent requests.
	if _, err := tx.Exec(context.Background(), `SELECT pg_advisory_xact_lock(hashtextextended(encode($1, 'hex'), 0))`, targetHash); err != nil {
		return OTPChallenge{}, "", fmt.Errorf("lock otp target: %w", err)
	}
	var cooldown bool
	if err := tx.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM app.otp_challenges
			WHERE target_hash = $1 AND expires_at > $2 AND last_sent_at > $3
		)`, targetHash, now, now.Add(-30*time.Second)).Scan(&cooldown); err != nil {
		return OTPChallenge{}, "", fmt.Errorf("check otp cooldown: %w", err)
	}
	if cooldown {
		return OTPChallenge{}, "", errors.New("otp resend cooldown active")
	}
	var challengeID string
	if err := tx.QueryRow(context.Background(), `
		INSERT INTO app.otp_challenges
			(target_type, target_hash, target_ciphertext, purpose, code_hmac, expires_at, last_sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id::text`, channel, targetHash, targetCiphertext, purpose, s.hashCodeBytes(code), now.Add(10*time.Minute), now).Scan(&challengeID); err != nil {
		return OTPChallenge{}, "", fmt.Errorf("create otp challenge: %w", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return OTPChallenge{}, "", fmt.Errorf("commit otp challenge: %w", err)
	}
	return OTPChallenge{ID: challengeID, TargetType: channel, TargetHash: hex.EncodeToString(targetHash), TargetValue: identifier, Purpose: purpose, CodeHash: s.hashCodeBytes(code), ExpiresAt: now.Add(10 * time.Minute), LastSentAt: now}, code, nil
}

func (s *PostgresStore) VerifyOTP(challengeID, code, deviceLabel string) (User, Session, string, error) {
	return s.verifyOTP(challengeID, code, deviceLabel, "", "")
}

func (s *PostgresStore) VerifyOTPForTarget(challengeID, code, deviceLabel, channel, identifier string) (User, Session, string, error) {
	if channel == "" || identifier == "" {
		return User{}, Session{}, "", errors.New("otp target is required")
	}
	return s.verifyOTP(challengeID, code, deviceLabel, channel, identifier)
}

func (s *PostgresStore) verifyOTP(challengeID, code, deviceLabel, expectedChannel, expectedIdentifier string) (User, Session, string, error) {
	now := s.now()
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return User{}, Session{}, "", err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var targetType string
	var targetHash, targetCiphertext, codeHash []byte
	var attempts int
	var expiresAt time.Time
	var consumedAt *time.Time
	if err := tx.QueryRow(context.Background(), `
		SELECT target_type, target_hash, target_ciphertext, code_hmac, attempt_count, expires_at, consumed_at
		FROM app.otp_challenges WHERE id = $1 FOR UPDATE`, challengeID).Scan(&targetType, &targetHash, &targetCiphertext, &codeHash, &attempts, &expiresAt, &consumedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, Session{}, "", errors.New("otp challenge is invalid or expired")
		}
		return User{}, Session{}, "", fmt.Errorf("load otp challenge: %w", err)
	}
	if consumedAt != nil || !now.Before(expiresAt) {
		return User{}, Session{}, "", errors.New("otp challenge is invalid or expired")
	}
	if expectedChannel != "" && !equalBytes(targetHash, s.hashTargetBytes(expectedChannel, expectedIdentifier)) {
		return User{}, Session{}, "", errors.New("otp challenge target mismatch")
	}
	if attempts >= 5 {
		return User{}, Session{}, "", errors.New("otp challenge is locked")
	}
	if _, err := tx.Exec(context.Background(), `UPDATE app.otp_challenges SET attempt_count = attempt_count + 1 WHERE id = $1`, challengeID); err != nil {
		return User{}, Session{}, "", fmt.Errorf("record otp attempt: %w", err)
	}
	if !hmacEqual(codeHash, s.hashCodeBytes(strings.TrimSpace(code))) {
		if err := tx.Commit(context.Background()); err != nil {
			return User{}, Session{}, "", fmt.Errorf("commit otp attempt: %w", err)
		}
		return User{}, Session{}, "", errors.New("otp code is invalid")
	}
	identifierBytes, err := s.decrypt(targetCiphertext)
	if err != nil {
		return User{}, Session{}, "", errors.New("otp challenge target is unavailable")
	}
	identifier := string(identifierBytes)
	user, err := s.findOrCreateUserTx(context.Background(), tx, identifier, targetType, now)
	if err != nil {
		return User{}, Session{}, "", err
	}
	if user.Status != "active" {
		return User{}, Session{}, "", errors.New("user is inactive")
	}
	if _, err := tx.Exec(context.Background(), `UPDATE app.otp_challenges SET consumed_at = $2 WHERE id = $1`, challengeID, now); err != nil {
		return User{}, Session{}, "", fmt.Errorf("consume otp challenge: %w", err)
	}
	if _, err := tx.Exec(context.Background(), `SELECT set_config('app.current_user_id', $1, true)`, user.ID); err != nil {
		return User{}, Session{}, "", fmt.Errorf("set auth context: %w", err)
	}
	if _, err := tx.Exec(context.Background(), `UPDATE app.users SET last_authenticated_at = $2 WHERE id = $1`, user.ID, now); err != nil {
		return User{}, Session{}, "", fmt.Errorf("update authenticated user: %w", err)
	}
	token, err := randomToken()
	if err != nil {
		return User{}, Session{}, "", err
	}
	var session Session
	if err := tx.QueryRow(context.Background(), `
		INSERT INTO app.sessions (user_id, token_hash, device_label, authentication_level, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, created_at, expires_at`, user.ID, s.hashTokenBytes(token), strings.TrimSpace(deviceLabel), AAL1, now, now.Add(30*24*time.Hour)).Scan(&session.ID, &session.CreatedAt, &session.ExpiresAt); err != nil {
		return User{}, Session{}, "", fmt.Errorf("create session: %w", err)
	}
	session.UserID, session.AuthenticationLevel, session.DeviceLabel = user.ID, AAL1, strings.TrimSpace(deviceLabel)
	if err := tx.Commit(context.Background()); err != nil {
		return User{}, Session{}, "", fmt.Errorf("commit otp verification: %w", err)
	}
	return user, session, token, nil
}

func (s *PostgresStore) FindOrCreateUser(identifier, channel string) (User, error) {
	identifier = NormalizeIdentifier(identifier)
	channel = strings.ToLower(strings.TrimSpace(channel))
	if identifier == "" || (channel != "phone" && channel != "email") {
		return User{}, errors.New("valid identifier and channel are required")
	}
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	user, err := s.findOrCreateUserTx(context.Background(), tx, identifier, channel, s.now())
	if err != nil {
		return User{}, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return User{}, fmt.Errorf("commit user: %w", err)
	}
	return user, nil
}

func (s *PostgresStore) findOrCreateUserTx(ctx context.Context, tx pgx.Tx, identifier, channel string, now time.Time) (User, error) {
	var user User
	var email, phone, displayName, status string
	if err := tx.QueryRow(ctx, `SELECT id::text, COALESCE(normalized_email,''), COALESCE(normalized_phone,''), COALESCE(display_name,''), status, created_at, COALESCE(last_authenticated_at, 'epoch') FROM app.find_or_create_user($1, $2, $3)`, channel, identifier, now).Scan(&user.ID, &email, &phone, &displayName, &status, &user.CreatedAt, &user.LastAuthenticatedAt); err != nil {
		return User{}, fmt.Errorf("find or create user: %w", err)
	}
	user.Email, user.Phone, user.DisplayName, user.Status = email, phone, displayName, status
	if user.LastAuthenticatedAt.Equal(time.Unix(0, 0).UTC()) {
		user.LastAuthenticatedAt = time.Time{}
	}
	return user, nil
}

func (s *PostgresStore) SessionFromToken(token string) (Session, User, error) {
	var session Session
	var user User
	var email, phone, displayName, status string
	var revokedAt *time.Time
	if err := s.pool.QueryRow(context.Background(), `
		SELECT session_id::text, user_id::text, authentication_level, COALESCE(device_label,''), session_created_at, session_expires_at, session_last_seen_at, session_revoked_at,
		       COALESCE(email,''), COALESCE(phone,''), COALESCE(display_name,''), user_status, user_created_at, COALESCE(last_authenticated_at, 'epoch')
		FROM app.session_by_token_hash($1)`, s.hashTokenBytes(token)).Scan(&session.ID, &session.UserID, &session.AuthenticationLevel, &session.DeviceLabel, &session.CreatedAt, &session.ExpiresAt, &session.LastSeenAt, &revokedAt, &email, &phone, &displayName, &status, &user.CreatedAt, &user.LastAuthenticatedAt); err != nil {
		return Session{}, User{}, errors.New("session is invalid or expired")
	}
	now := s.now()
	if revokedAt != nil || !now.Before(session.ExpiresAt) || status != "active" {
		return Session{}, User{}, errors.New("session is invalid or expired")
	}
	// Both deadlines apply: the absolute lifetime above and the idle deadline
	// here, as README section 21.2 requires.
	lastSeen := session.LastSeenAt
	if lastSeen.IsZero() {
		lastSeen = session.CreatedAt
	}
	if now.Sub(lastSeen) >= sessionIdleTimeout {
		return Session{}, User{}, errors.New("session is invalid or expired")
	}
	user.ID, user.Email, user.Phone, user.DisplayName, user.Status = session.UserID, email, phone, displayName, status
	tx, txErr := s.beginUserTx(user.ID)
	if txErr == nil {
		var verifiedAt *time.Time
		if queryErr := tx.QueryRow(context.Background(), `SELECT mfa_verified_at FROM app.sessions WHERE id = $1`, session.ID).Scan(&verifiedAt); queryErr == nil && verifiedAt != nil {
			session.MFAVerifiedAt = verifiedAt.UTC()
		}
		// Refresh last-seen at most once per interval so the idle deadline does
		// not add a write to every authenticated request.
		if now.Sub(lastSeen) >= sessionIdleRefresh {
			if _, execErr := tx.Exec(context.Background(), `SELECT app.touch_session($1, $2)`, session.ID, now); execErr == nil {
				session.LastSeenAt = now
			}
		}
		_ = tx.Commit(context.Background())
	}
	if user.LastAuthenticatedAt.Equal(time.Unix(0, 0).UTC()) {
		user.LastAuthenticatedAt = time.Time{}
	}
	return session, user, nil
}

func (s *PostgresStore) RevokeSession(token string) error {
	session, user, err := s.SessionFromToken(token)
	if err != nil {
		return errors.New("session not found")
	}
	tx, err := s.beginUserTx(user.ID)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(context.Background(), `UPDATE app.sessions SET revoked_at = $2 WHERE id = $1`, session.ID, s.now()); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit session revocation: %w", err)
	}
	return nil
}

func (s *PostgresStore) RevokeAllSessions(userID string) error {
	tx, err := s.beginUserTx(userID)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err = tx.Exec(context.Background(), `UPDATE app.sessions SET revoked_at=$2 WHERE user_id=$1 AND revoked_at IS NULL`, userID, s.now()); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}
	return tx.Commit(context.Background())
}

func (s *PostgresStore) BeginTOTPEnrollment(userID string) (MFAMethod, error) {
	if strings.TrimSpace(userID) == "" {
		return MFAMethod{}, errors.New("verified MFA already exists or enrollment is unavailable")
	}
	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		return MFAMethod{}, err
	}
	secret := base32NoPadding(secretBytes)
	ciphertext, err := s.encrypt([]byte(secret))
	if err != nil {
		return MFAMethod{}, err
	}
	tx, err := s.beginUserTx(userID)
	if err != nil {
		return MFAMethod{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var method MFAMethod
	if err := tx.QueryRow(context.Background(), `
		INSERT INTO app.mfa_methods (user_id, method_type, secret_ciphertext)
		VALUES ($1, 'totp', $2)
        ON CONFLICT (user_id,method_type) WHERE revoked_at IS NULL
        DO UPDATE SET secret_ciphertext=EXCLUDED.secret_ciphertext,failed_attempts=0,locked_until=NULL
        WHERE app.mfa_methods.verified_at IS NULL
		RETURNING id::text, user_id::text, method_type`, userID, ciphertext).Scan(&method.ID, &method.UserID, &method.Type); err != nil {
		return MFAMethod{}, errors.New("verified MFA already exists or enrollment is unavailable")
	}
	if err := tx.Commit(context.Background()); err != nil {
		return MFAMethod{}, fmt.Errorf("commit mfa enrollment: %w", err)
	}
	method.Secret = secret
	return method, nil
}

// VerifyTOTP throttles per account. The row is locked for the whole check so
// concurrent guesses cannot each read the same attempt count and slip past the
// limit together.
func (s *PostgresStore) VerifyTOTP(userID, code string) error {
	var methodID string
	var ciphertext []byte
	var revokedAt, lockedUntil *time.Time
	var failedAttempts int
	var lastUsedCounter *int64
	tx, err := s.beginUserTx(userID)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err := tx.QueryRow(context.Background(), `SELECT id::text, secret_ciphertext, revoked_at, failed_attempts, locked_until, last_used_counter FROM app.mfa_methods WHERE user_id = $1 AND method_type = 'totp' ORDER BY created_at DESC LIMIT 1 FOR UPDATE`, userID).Scan(&methodID, &ciphertext, &revokedAt, &failedAttempts, &lockedUntil, &lastUsedCounter); err != nil || revokedAt != nil {
		return errors.New("mfa method is not enrolled")
	}
	now := s.now()
	if lockedUntil != nil && now.Before(*lockedUntil) {
		return ErrMFALocked
	}
	secret, decryptErr := s.decrypt(ciphertext)
	counter, matched := matchingTOTPCounter(string(secret), strings.TrimSpace(code), now)
	if decryptErr != nil || !matched || (lastUsedCounter != nil && counter <= *lastUsedCounter) {
		locked := failedAttempts+1 >= mfaMaxFailedAttempts
		nextAttempts := failedAttempts + 1
		var nextLock any
		if locked {
			nextAttempts = 0
			nextLock = now.Add(mfaLockDuration)
		}
		if _, execErr := tx.Exec(context.Background(), `UPDATE app.mfa_methods SET failed_attempts = $2, locked_until = $3 WHERE id = $1::uuid`, methodID, nextAttempts, nextLock); execErr != nil {
			return fmt.Errorf("record mfa attempt: %w", execErr)
		}
		if commitErr := tx.Commit(context.Background()); commitErr != nil {
			return fmt.Errorf("commit mfa attempt: %w", commitErr)
		}
		if locked {
			return ErrMFALocked
		}
		return errors.New("mfa code is invalid")
	}
	if _, err := tx.Exec(context.Background(), `UPDATE app.mfa_methods SET verified_at = $2, last_used_at = $2, last_used_counter = $3, failed_attempts = 0, locked_until = NULL WHERE user_id = $1 AND method_type = 'totp' AND revoked_at IS NULL`, userID, now, counter); err != nil {
		return fmt.Errorf("verify mfa method: %w", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit mfa verification: %w", err)
	}
	return nil
}

func (s *PostgresStore) StepUpSession(token, code string) (Session, string, error) {
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Session{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var session Session
	var userID string
	if err = tx.QueryRow(ctx, `SELECT session_id::text,user_id::text,authentication_level,COALESCE(device_label,''),session_created_at,session_expires_at,session_last_seen_at FROM app.session_by_token_hash($1)`, s.hashTokenBytes(token)).Scan(&session.ID, &userID, &session.AuthenticationLevel, &session.DeviceLabel, &session.CreatedAt, &session.ExpiresAt, &session.LastSeenAt); err != nil {
		return Session{}, "", errors.New("session not found")
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, userID); err != nil {
		return Session{}, "", err
	}
	now := s.now()
	var revokedAt *time.Time
	if err = tx.QueryRow(ctx, `SELECT revoked_at FROM app.sessions WHERE id=$1::uuid AND token_hash=$2 FOR UPDATE`, session.ID, s.hashTokenBytes(token)).Scan(&revokedAt); err != nil || revokedAt != nil || !now.Before(session.ExpiresAt) || now.Sub(session.LastSeenAt) >= sessionIdleTimeout {
		return Session{}, "", errors.New("session not found")
	}
	var methodID string
	var ciphertext []byte
	var methodRevokedAt, lockedUntil *time.Time
	var lastUsedCounter *int64
	var failedAttempts int
	if err = tx.QueryRow(ctx, `SELECT id::text,secret_ciphertext,revoked_at,failed_attempts,locked_until,last_used_counter FROM app.mfa_methods WHERE user_id=$1::uuid AND method_type='totp' ORDER BY created_at DESC LIMIT 1 FOR UPDATE`, userID).Scan(&methodID, &ciphertext, &methodRevokedAt, &failedAttempts, &lockedUntil, &lastUsedCounter); err != nil || methodRevokedAt != nil {
		return Session{}, "", errors.New("mfa method is not enrolled")
	}
	if lockedUntil != nil && now.Before(*lockedUntil) {
		return Session{}, "", ErrMFALocked
	}
	secret, decryptErr := s.decrypt(ciphertext)
	counter, matched := matchingTOTPCounter(string(secret), strings.TrimSpace(code), now)
	if decryptErr != nil || !matched || (lastUsedCounter != nil && counter <= *lastUsedCounter) {
		locked := failedAttempts+1 >= mfaMaxFailedAttempts
		nextAttempts := failedAttempts + 1
		var nextLock any
		if locked {
			nextAttempts, nextLock = 0, now.Add(mfaLockDuration)
		}
		if _, updateErr := tx.Exec(ctx, `UPDATE app.mfa_methods SET failed_attempts=$2,locked_until=$3 WHERE id=$1::uuid`, methodID, nextAttempts, nextLock); updateErr != nil {
			return Session{}, "", updateErr
		}
		if err = tx.Commit(ctx); err != nil {
			return Session{}, "", err
		}
		if locked {
			return Session{}, "", ErrMFALocked
		}
		return Session{}, "", errors.New("mfa code is invalid or was already used")
	}
	newToken, err := randomToken()
	if err != nil {
		return Session{}, "", err
	}
	newSession := Session{ID: newID(), UserID: userID, AuthenticationLevel: AAL2, MFAVerifiedAt: now, DeviceLabel: session.DeviceLabel, CreatedAt: now, ExpiresAt: session.ExpiresAt, LastSeenAt: now}
	if _, err = tx.Exec(ctx, `UPDATE app.mfa_methods SET verified_at=$2,last_used_at=$2,last_used_counter=$3,failed_attempts=0,locked_until=NULL WHERE id=$1::uuid`, methodID, now, counter); err != nil {
		return Session{}, "", err
	}
	if _, err = tx.Exec(ctx, `UPDATE app.sessions SET revoked_at=$2 WHERE id=$1::uuid`, session.ID, now); err != nil {
		return Session{}, "", err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.sessions(id,user_id,token_hash,device_label,authentication_level,mfa_verified_at,created_at,expires_at,last_seen_at) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,$6,$7,$6)`, newSession.ID, userID, s.hashTokenBytes(newToken), newSession.DeviceLabel, AAL2, now, newSession.ExpiresAt); err != nil {
		return Session{}, "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return Session{}, "", err
	}
	return newSession, newToken, nil
}

func (s *PostgresStore) ElevateSession(token string) error {
	session, user, err := s.SessionFromToken(token)
	if err != nil {
		return errors.New("session not found")
	}
	if !s.IsMFAEnrolled(user.ID) {
		return errors.New("verified mfa is required")
	}
	tx, err := s.beginUserTx(user.ID)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(context.Background(), `UPDATE app.sessions SET authentication_level = $2, mfa_verified_at = $3 WHERE id = $1`, session.ID, AAL2, s.now()); err != nil {
		return fmt.Errorf("elevate session: %w", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit session elevation: %w", err)
	}
	return nil
}

func (s *PostgresStore) IsMFAEnrolled(userID string) bool {
	tx, err := s.beginUserTx(userID)
	if err != nil {
		return false
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var enrolled bool
	if err := tx.QueryRow(context.Background(), `SELECT EXISTS (SELECT 1 FROM app.mfa_methods WHERE user_id = $1 AND method_type = 'totp' AND verified_at IS NOT NULL AND revoked_at IS NULL)`, userID).Scan(&enrolled); err != nil {
		return false
	}
	_ = tx.Commit(context.Background())
	return enrolled
}

func (s *PostgresStore) beginUserTx(userID string) (pgx.Tx, error) {
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(context.Background(), `SELECT set_config('app.current_user_id', $1, true)`, userID); err != nil {
		_ = tx.Rollback(context.Background())
		return nil, fmt.Errorf("set auth context: %w", err)
	}
	return tx, nil
}

func (s *PostgresStore) hashCodeBytes(code string) []byte { return hmacDigest(s.otpKey, []byte(code)) }
func (s *PostgresStore) hashTargetBytes(channel, identifier string) []byte {
	return hmacDigest(s.otpKey, []byte(targetKey(channel, identifier)))
}
func (s *PostgresStore) hashTokenBytes(token string) []byte {
	return hmacDigest(s.tokenKey, []byte(token))
}

func hmacDigest(key, value []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(value)
	return mac.Sum(nil)
}

func hmacEqual(left, right []byte) bool  { return hmac.Equal(left, right) }
func equalBytes(left, right []byte) bool { return hmac.Equal(left, right) }

func (s *PostgresStore) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (s *PostgresStore) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("invalid ciphertext")
	}
	nonce, payload := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, payload, nil)
}

// base32NoPadding is kept local so the adapter never exposes raw key material
// in SQL or logs; the returned value is only sent to the enrollment client.
func base32NoPadding(value []byte) string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(value)
}
