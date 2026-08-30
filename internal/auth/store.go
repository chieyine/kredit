package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	AAL0 = "AAL0"
	AAL1 = "AAL1"
	AAL2 = "AAL2"
)

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
	RevokedAt           time.Time `json:"revoked_at,omitempty"`
}

type MFAMethod struct {
	ID         string
	UserID     string
	Type       string
	Secret     string
	VerifiedAt time.Time
	RevokedAt  time.Time
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
	now           func() time.Time
}

func NewStore(tokenHashKey string) *Store {
	if tokenHashKey == "" {
		tokenHashKey = "development-only-change-me"
	}
	return &Store{
		users:         make(map[string]*User),
		usersByTarget: make(map[string]string),
		challenges:    make(map[string]*OTPChallenge),
		sessions:      make(map[string]*Session),
		sessionTokens: make(map[string]string),
		mfaMethods:    make(map[string]*MFAMethod),
		tokenHashKey:  []byte(tokenHashKey),
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
		if existing.TargetHash == s.hashTarget(channel, identifier) && existing.Purpose == purpose && existing.ConsumedAt.IsZero() && now.Before(existing.ExpiresAt) && now.Sub(existing.LastSentAt) < 30*time.Second {
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
	if channel == "email" {
		user.Email = identifier
	} else if channel == "phone" {
		user.Phone = identifier
	} else {
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
	session := Session{ID: newID(), UserID: user.ID, AuthenticationLevel: AAL1, DeviceLabel: strings.TrimSpace(deviceLabel), CreatedAt: now, ExpiresAt: now.Add(30 * 24 * time.Hour)}
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	sessionID := s.sessionTokens[s.hashToken(token)]
	session, ok := s.sessions[sessionID]
	if !ok || !session.RevokedAt.IsZero() || !s.now().Before(session.ExpiresAt) {
		return Session{}, User{}, errors.New("session is invalid or expired")
	}
	user, ok := s.users[session.UserID]
	if !ok || user.Status != "active" {
		return Session{}, User{}, errors.New("user is inactive")
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
	found := false
	for _, session := range s.sessions {
		if session.UserID == userID && session.RevokedAt.IsZero() {
			session.RevokedAt = s.now()
			found = true
		}
	}
	if !found {
		return nil
	}
	return nil
}

func (s *Store) BeginTOTPEnrollment(userID string) (MFAMethod, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return MFAMethod{}, errors.New("user not found")
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
	if !validTOTP(method.Secret, strings.TrimSpace(code), s.now()) {
		return errors.New("mfa code is invalid")
	}
	method.VerifiedAt = s.now()
	return nil
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

func NormalizeIdentifier(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if strings.Contains(normalized, "@") {
		return normalized
	}
	for _, separator := range []string{" ", "-", "(", ")"} {
		normalized = strings.ReplaceAll(normalized, separator, "")
	}
	return normalized
}

func (s *Store) hashCode(code string) []byte {
	mac := hmac.New(sha256.New, s.tokenHashKey)
	_, _ = mac.Write([]byte(code))
	return mac.Sum(nil)
}

func (s *Store) hashToken(token string) string {
	mac := hmac.New(sha256.New, s.tokenHashKey)
	_, _ = mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Store) hashTarget(channel, identifier string) string {
	mac := hmac.New(sha256.New, s.tokenHashKey)
	_, _ = mac.Write([]byte(targetKey(channel, identifier)))
	return hex.EncodeToString(mac.Sum(nil))
}

func targetKey(channel, identifier string) string {
	return channel + ":" + NormalizeIdentifier(identifier)
}

func randomOTP() (string, error) {
	var value [4]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	combined := uint32(value[0])<<24 | uint32(value[1])<<16 | uint32(value[2])<<8 | uint32(value[3])
	return fmt.Sprintf("%06d", combined%1000000), nil
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

func validTOTP(secret, code string, now time.Time) bool {
	for delta := int64(-1); delta <= 1; delta++ {
		if totp(secret, now.Unix()/30+delta) == code {
			return true
		}
	}
	return false
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
