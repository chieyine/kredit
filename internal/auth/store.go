package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

const (
	AAL0 = "AAL0"
	AAL1 = "AAL1"
	AAL2 = "AAL2"
)

const (
	// A six-digit TOTP with a three-step acceptance window is brute-forceable
	// given an authenticated AAL1 session and a shared per-IP request budget, so
	// verification is throttled per account as well. The lock expires on its own
	// so a legitimate user is never permanently shut out; account recovery
	// remains the path when they cannot wait.
	mfaMaxFailedAttempts = 5
	mfaLockDuration      = 15 * time.Minute

	// Sessions expire on both an absolute and an idle deadline, as README
	// section 21.2 requires. Last-seen is refreshed at most once an interval so
	// an idle deadline does not add a write to every authenticated request.
	sessionAbsoluteLifetime = 30 * 24 * time.Hour
	sessionIdleTimeout      = 14 * 24 * time.Hour
	sessionIdleRefresh      = time.Hour
)

// ErrMFALocked reports that too many incorrect codes were submitted for this
// account and verification is paused.
var ErrMFALocked = errors.New("too many incorrect verification codes; try again later")

type User struct {
	ID                  string    `json:"id"`
	Email               string    `json:"email,omitempty"`
	Phone               string    `json:"phone,omitempty"`
	DisplayName         string    `json:"display_name,omitempty"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	LastAuthenticatedAt time.Time `json:"last_authenticated_at,omitempty"`
}

type OTPChallenge struct {
	ID           string
	TargetType   string
	TargetHash   string
	TargetValue  string
	Purpose      string
	CodeHash     []byte
	AttemptCount int
	ExpiresAt    time.Time
	ConsumedAt   time.Time
	LastSentAt   time.Time
}

type Session struct {
	ID                  string    `json:"id"`
	UserID              string    `json:"user_id"`
	AuthenticationLevel string    `json:"authentication_level"`
	MFAVerifiedAt       time.Time `json:"mfa_verified_at,omitempty"`
	DeviceLabel         string    `json:"device_label,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	ExpiresAt           time.Time `json:"expires_at"`
	LastSeenAt          time.Time `json:"last_seen_at,omitempty"`
	RevokedAt           time.Time `json:"revoked_at,omitempty"`
}

type MFAMethod struct {
	ID              string
	UserID          string
	Type            string
	Secret          string
	VerifiedAt      time.Time
	RevokedAt       time.Time
	FailedAttempts  int
	LockedUntil     time.Time
	LastUsedCounter int64
}

type Store struct {
	mu            sync.RWMutex
	users         map[string]*User
	usersByTarget map[string]string
	challenges    map[string]*OTPChallenge
	sessions      map[string]*Session
	sessionTokens map[string]string
	mfaMethods    map[string]*MFAMethod
	tokenHashKey  []byte
	otpHMACKey    []byte
	now           func() time.Time
}

func NewStore(tokenHashKey string) *Store {
	return NewStoreWithKeys(tokenHashKey, tokenHashKey)
}

func NewStoreWithKeys(tokenHashKey, otpHMACKey string) *Store {
	if tokenHashKey == "" {
		tokenHashKey = "development-only-change-me"
	}
	if otpHMACKey == "" {
		otpHMACKey = "development-only-change-me"
	}
	return &Store{
		users:         make(map[string]*User),
		usersByTarget: make(map[string]string),
		challenges:    make(map[string]*OTPChallenge),
		sessions:      make(map[string]*Session),
		sessionTokens: make(map[string]string),
		mfaMethods:    make(map[string]*MFAMethod),
		tokenHashKey:  []byte(tokenHashKey),
		otpHMACKey:    []byte(otpHMACKey),
		now:           func() time.Time { return time.Now().UTC() },
	}
}

func (s *Store) RequestOTP(identifier, channel, purpose string) (OTPChallenge, string, error) {
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
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.challenges {
		if existing.TargetHash == s.hashTarget(channel, identifier) && now.Before(existing.ExpiresAt) && now.Sub(existing.LastSentAt) < 30*time.Second {
			return OTPChallenge{}, "", errors.New("otp resend cooldown active")
		}
	}
	code, err := randomOTP()
	if err != nil {
		return OTPChallenge{}, "", err
	}
	challenge := OTPChallenge{
		ID:          newID(),
		TargetType:  channel,
		TargetHash:  s.hashTarget(channel, identifier),
		TargetValue: identifier,
		Purpose:     purpose,
		CodeHash:    s.hashCode(code),
		ExpiresAt:   now.Add(10 * time.Minute),
		LastSentAt:  now,
	}
	s.challenges[challenge.ID] = &challenge
	return challenge, code, nil
}

func (s *Store) VerifyOTP(challengeID, code, deviceLabel string) (User, Session, string, error) {
	return s.verifyOTP(challengeID, code, deviceLabel, "", "")
}

func (s *Store) VerifyOTPForTarget(challengeID, code, deviceLabel, channel, identifier string) (User, Session, string, error) {
	if channel == "" || identifier == "" {
		return User{}, Session{}, "", errors.New("otp target is required")
	}
	return s.verifyOTP(challengeID, code, deviceLabel, channel, identifier)
}

func (s *Store) VerifyAndAttachIdentifier(userID, challengeID, code, channel, identifier string) error {
	identifier = NormalizeIdentifier(identifier)
	channel = strings.ToLower(strings.TrimSpace(channel))
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.users[userID]
	challenge := s.challenges[challengeID]
	if user == nil || challenge == nil || !challenge.ConsumedAt.IsZero() || !now.Before(challenge.ExpiresAt) || challenge.AttemptCount >= 5 || challenge.TargetType != channel || challenge.TargetHash != s.hashTarget(channel, identifier) {
		return errors.New("otp challenge is invalid or expired")
	}
	challenge.AttemptCount++
	if !hmac.Equal(challenge.CodeHash, s.hashCode(strings.TrimSpace(code))) {
		return errors.New("otp code is invalid")
	}
	key := targetKey(channel, identifier)
	if existing := s.usersByTarget[key]; existing != "" && existing != userID {
		return errors.New("contact is already attached to another account")
	}
	switch channel {
	case "email":
		user.Email = identifier
	case "phone":
		user.Phone = identifier
	default:
		return errors.New("contact channel must be email or phone")
	}
	s.usersByTarget[key] = userID
	challenge.ConsumedAt = now
	return nil
}

func (s *Store) verifyOTP(challengeID, code, deviceLabel, expectedChannel, expectedIdentifier string) (User, Session, string, error) {
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	challenge, ok := s.challenges[challengeID]
	if !ok || !challenge.ConsumedAt.IsZero() || !now.Before(challenge.ExpiresAt) {
		return User{}, Session{}, "", errors.New("otp challenge is invalid or expired")
	}
	if expectedChannel != "" && challenge.TargetHash != s.hashTarget(expectedChannel, expectedIdentifier) {
		return User{}, Session{}, "", errors.New("otp challenge target mismatch")
	}
	if challenge.AttemptCount >= 5 {
		return User{}, Session{}, "", errors.New("otp challenge is locked")
	}
	challenge.AttemptCount++
	if !hmac.Equal(challenge.CodeHash, s.hashCode(strings.TrimSpace(code))) {
		return User{}, Session{}, "", errors.New("otp code is invalid")
	}
	challenge.ConsumedAt = now
	identifier := challenge.TargetValue
	userID := s.usersByTarget[targetKey(challenge.TargetType, identifier)]
	if userID == "" {
		userID = newID()
		user := &User{ID: userID, Status: "active", CreatedAt: now}
		if challenge.TargetType == "email" {
			user.Email = identifier
		} else {
			user.Phone = identifier
		}
		s.users[userID] = user
		s.usersByTarget[targetKey(challenge.TargetType, identifier)] = userID
	}
	user := s.users[userID]
	user.LastAuthenticatedAt = now
	token, err := randomToken()
	if err != nil {
		return User{}, Session{}, "", err
	}
	session := Session{ID: newID(), UserID: user.ID, AuthenticationLevel: AAL1, DeviceLabel: strings.TrimSpace(deviceLabel), CreatedAt: now, ExpiresAt: now.Add(sessionAbsoluteLifetime), LastSeenAt: now}
	s.sessions[session.ID] = &session
	s.sessionTokens[s.hashToken(token)] = session.ID
	return cloneUser(*user), session, token, nil
}

func (s *Store) FindOrCreateUser(identifier, channel string) (User, error) {
	identifier = NormalizeIdentifier(identifier)
	channel = strings.ToLower(strings.TrimSpace(channel))
	if identifier == "" || (channel != "phone" && channel != "email") {
		return User{}, errors.New("valid identifier and channel are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if userID := s.usersByTarget[targetKey(channel, identifier)]; userID != "" {
		return cloneUser(*s.users[userID]), nil
	}
	now := s.now()
	user := &User{ID: newID(), Status: "active", CreatedAt: now}
	if channel == "email" {
		user.Email = identifier
	} else {
		user.Phone = identifier
	}
	s.users[user.ID] = user
	s.usersByTarget[targetKey(channel, identifier)] = user.ID
	return cloneUser(*user), nil
}

func (s *Store) UserByID(userID string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user := s.users[userID]
	if user == nil {
		return User{}, errors.New("user not found")
	}
	return cloneUser(*user), nil
}

func (s *Store) SessionFromToken(token string) (Session, User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessionID := s.sessionTokens[s.hashToken(token)]
	session, ok := s.sessions[sessionID]
	if !ok || !session.RevokedAt.IsZero() {
		return Session{}, User{}, errors.New("session is invalid or expired")
	}
	now := s.now()
	if !now.Before(session.ExpiresAt) {
		return Session{}, User{}, errors.New("session is invalid or expired")
	}
	lastSeen := session.LastSeenAt
	if lastSeen.IsZero() {
		lastSeen = session.CreatedAt
	}
	if now.Sub(lastSeen) >= sessionIdleTimeout {
		return Session{}, User{}, errors.New("session is invalid or expired")
	}
	user, ok := s.users[session.UserID]
	if !ok || user.Status != "active" {
		return Session{}, User{}, errors.New("user is inactive")
	}
	if now.Sub(lastSeen) >= sessionIdleRefresh {
		session.LastSeenAt = now
	}
	return cloneSession(*session), cloneUser(*user), nil
}

func (s *Store) RevokeSession(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessionID := s.sessionTokens[s.hashToken(token)]
	session, ok := s.sessions[sessionID]
	if !ok {
		return errors.New("session not found")
	}
	session.RevokedAt = s.now()
	return nil
}

func (s *Store) RevokeAllSessions(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, session := range s.sessions {
		if session.UserID == userID && session.RevokedAt.IsZero() {
			session.RevokedAt = s.now()
		}
	}
	return nil
}

func (s *Store) BeginTOTPEnrollment(userID string) (MFAMethod, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return MFAMethod{}, errors.New("user not found")
	}
	if existing := s.mfaMethods[userID]; existing != nil && existing.RevokedAt.IsZero() && !existing.VerifiedAt.IsZero() {
		return MFAMethod{}, errors.New("MFA is already enrolled; use account recovery to replace it")
	}
	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		return MFAMethod{}, err
	}
	method := &MFAMethod{ID: newID(), UserID: userID, Type: "totp", Secret: base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)}
	s.mfaMethods[userID] = method
	return cloneMFA(*method), nil
}

func (s *Store) VerifyTOTP(userID, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	method, ok := s.mfaMethods[userID]
	if !ok || !method.RevokedAt.IsZero() {
		return errors.New("mfa method is not enrolled")
	}
	now := s.now()
	if !method.LockedUntil.IsZero() && now.Before(method.LockedUntil) {
		return ErrMFALocked
	}
	counter, matched := matchingTOTPCounter(method.Secret, strings.TrimSpace(code), now)
	if !matched || (method.LastUsedCounter != 0 && counter <= method.LastUsedCounter) {
		method.FailedAttempts++
		if method.FailedAttempts >= mfaMaxFailedAttempts {
			method.FailedAttempts = 0
			method.LockedUntil = now.Add(mfaLockDuration)
			return ErrMFALocked
		}
		return errors.New("mfa code is invalid")
	}
	method.FailedAttempts, method.LockedUntil = 0, time.Time{}
	method.VerifiedAt = now
	method.LastUsedCounter = counter
	return nil
}

func (s *Store) StepUpSession(token, code string) (Session, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	oldHash := s.hashToken(token)
	oldSessionID := s.sessionTokens[oldHash]
	oldSession := s.sessions[oldSessionID]
	if oldSession == nil || !oldSession.RevokedAt.IsZero() {
		return Session{}, "", errors.New("session not found")
	}
	now := s.now()
	lastSeen := oldSession.LastSeenAt
	if lastSeen.IsZero() {
		lastSeen = oldSession.CreatedAt
	}
	if !now.Before(oldSession.ExpiresAt) || now.Sub(lastSeen) >= sessionIdleTimeout {
		return Session{}, "", errors.New("session not found")
	}
	method := s.mfaMethods[oldSession.UserID]
	if method == nil || !method.RevokedAt.IsZero() {
		return Session{}, "", errors.New("mfa method is not enrolled")
	}
	if !method.LockedUntil.IsZero() && now.Before(method.LockedUntil) {
		return Session{}, "", ErrMFALocked
	}
	counter, matched := matchingTOTPCounter(method.Secret, strings.TrimSpace(code), now)
	if !matched || (method.LastUsedCounter != 0 && counter <= method.LastUsedCounter) {
		method.FailedAttempts++
		if method.FailedAttempts >= mfaMaxFailedAttempts {
			method.FailedAttempts = 0
			method.LockedUntil = now.Add(mfaLockDuration)
			return Session{}, "", ErrMFALocked
		}
		return Session{}, "", errors.New("mfa code is invalid or was already used")
	}
	newToken, err := randomToken()
	if err != nil {
		return Session{}, "", err
	}
	method.FailedAttempts, method.LockedUntil = 0, time.Time{}
	method.VerifiedAt, method.LastUsedCounter = now, counter
	oldSession.RevokedAt = now
	delete(s.sessionTokens, oldHash)
	newSession := Session{ID: newID(), UserID: oldSession.UserID, AuthenticationLevel: AAL2, MFAVerifiedAt: now, DeviceLabel: oldSession.DeviceLabel, CreatedAt: now, ExpiresAt: oldSession.ExpiresAt, LastSeenAt: now}
	s.sessions[newSession.ID] = &newSession
	s.sessionTokens[s.hashToken(newToken)] = newSession.ID
	return cloneSession(newSession), newToken, nil
}

func (s *Store) ElevateSession(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessionID := s.sessionTokens[s.hashToken(token)]
	session, ok := s.sessions[sessionID]
	if !ok {
		return errors.New("session not found")
	}
	method, ok := s.mfaMethods[session.UserID]
	if !ok || method.VerifiedAt.IsZero() || !method.RevokedAt.IsZero() {
		return errors.New("verified mfa is required")
	}
	session.AuthenticationLevel = AAL2
	session.MFAVerifiedAt = s.now()
	return nil
}

func (s *Store) IsMFAEnrolled(userID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	method, ok := s.mfaMethods[userID]
	return ok && !method.VerifiedAt.IsZero() && method.RevokedAt.IsZero()
}

// NormalizeIdentifier produces the single stored form of a login identifier.
// Email addresses are lowercased; telephone numbers are canonicalised to E.164.
func NormalizeIdentifier(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.Contains(trimmed, "@") {
		return strings.ToLower(trimmed)
	}
	return NormalizePhone(trimmed)
}

// NormalizePhone canonicalises a Nigerian number to E.164. 08012345678,
// 8012345678, 2348012345678 and "+234 801 234 5678" are one person; storing
// them as different identifiers creates duplicate accounts and splits the
// buyer's trade history, which section 35.6 requires normalisation to prevent.
// A number carrying an explicit country code other than 234 is preserved
// as-dialled rather than guessed at. The function is idempotent: normalising an
// already-normalised value returns it unchanged.
func NormalizePhone(value string) string {
	trimmed := strings.TrimSpace(value)
	dialled := strings.HasPrefix(trimmed, "+")
	var builder strings.Builder
	for _, character := range trimmed {
		if character >= '0' && character <= '9' {
			builder.WriteRune(character)
		}
	}
	digits := builder.String()
	switch {
	case digits == "":
		return ""
	case len(digits) == 13 && strings.HasPrefix(digits, nigeriaCallingCode):
		return "+" + digits
	case len(digits) == 11 && strings.HasPrefix(digits, "0"):
		return "+" + nigeriaCallingCode + digits[1:]
	case len(digits) == 10 && !dialled:
		return "+" + nigeriaCallingCode + digits
	case dialled:
		return "+" + digits
	default:
		return digits
	}
}

const nigeriaCallingCode = "234"

func (s *Store) hashCode(code string) []byte {
	mac := hmac.New(sha256.New, s.otpHMACKey)
	_, _ = mac.Write([]byte(code))
	return mac.Sum(nil)
}

func (s *Store) hashToken(token string) string {
	mac := hmac.New(sha256.New, s.tokenHashKey)
	_, _ = mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Store) hashTarget(channel, identifier string) string {
	mac := hmac.New(sha256.New, s.otpHMACKey)
	_, _ = mac.Write([]byte(targetKey(channel, identifier)))
	return hex.EncodeToString(mac.Sum(nil))
}

func targetKey(channel, identifier string) string {
	return channel + ":" + NormalizeIdentifier(identifier)
}

// randomOTP draws a uniform six-digit code. Reducing a raw 32-bit draw modulo
// 1e6 biases the low codes, so rejection sampling is used instead.
func randomOTP() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func randomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func newID() string {
	token, err := randomToken()
	if err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return token[:32]
}

func cloneUser(user User) User             { return user }
func cloneSession(session Session) Session { return session }
func cloneMFA(method MFAMethod) MFAMethod  { return method }

// validTOTP compares in constant time and refuses to match anything when the
// stored secret cannot be decoded. Without the explicit format check a corrupt
// or empty secret makes totp() return "", which an empty submitted code would
// otherwise match - an authentication bypass.
func validTOTP(secret, code string, now time.Time) bool {
	_, matched := matchingTOTPCounter(secret, code, now)
	return matched
}

func matchingTOTPCounter(secret, code string, now time.Time) (int64, bool) {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return 0, false
	}
	for _, character := range code {
		if character < '0' || character > '9' {
			return 0, false
		}
	}
	var matchedCounter int64
	matched := false
	for delta := int64(-1); delta <= 1; delta++ {
		candidate := totp(secret, now.Unix()/30+delta)
		if candidate == "" {
			return 0, false
		}
		if subtle.ConstantTimeCompare([]byte(candidate), []byte(code)) == 1 {
			matched = true
			matchedCounter = now.Unix()/30 + delta
		}
	}
	return matchedCounter, matched
}

// TOTPCode returns the current six-digit code for a secret. It is provided for
// provider/simulator contract tests; callers must never persist or log secrets.
func TOTPCode(secret string, at time.Time) string {
	return totp(secret, at.Unix()/30)
}

func totp(secret string, counter int64) string {
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return ""
	}
	message := make([]byte, 8)
	for index := 7; index >= 0; index-- {
		message[index] = byte(counter)
		counter >>= 8
	}
	mac := hmac.New(sha1.New, decoded)
	_, _ = mac.Write(message)
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 0x0f
	value := (uint32(digest[offset])&0x7f)<<24 | uint32(digest[offset+1])<<16 | uint32(digest[offset+2])<<8 | uint32(digest[offset+3])
	return fmt.Sprintf("%06d", value%1000000)
}

// ResetAfterRecovery is wired only to completion of the independently reviewed,
// two-factor recovery flow. It does not create an elevated session.
func (s *Store) ResetAfterRecovery(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.users[userID] == nil {
		return errors.New("user not found")
	}
	now := s.now()
	if m := s.mfaMethods[userID]; m != nil {
		m.RevokedAt = now
	}
	for _, session := range s.sessions {
		if session.UserID == userID {
			session.RevokedAt = now
		}
	}
	return nil
}
