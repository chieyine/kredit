package usercontrol

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	RecoveryPendingVerification = "PENDING_VERIFICATION"
	RecoveryPendingReview       = "PENDING_REVIEW"
	RecoveryCoolingOff          = "COOLING_OFF"
	RecoveryCompleted           = "COMPLETED"
)

type RecoveryRequest struct {
	ID                     string            `json:"id"`
	TargetUserID           string            `json:"target_user_id,omitempty"`
	State                  string            `json:"state"`
	RequestedChannel       string            `json:"requested_channel"`
	RiskFacts              map[string]string `json:"risk_facts,omitempty"`
	IndependentFactorCount int               `json:"independent_factor_count"`
	ReviewerUserID         string            `json:"reviewer_user_id,omitempty"`
	ReviewReason           string            `json:"review_reason,omitempty"`
	CoolingOffUntil        time.Time         `json:"cooling_off_until,omitempty"`
	ExpiresAt              time.Time         `json:"expires_at"`
	Version                int64             `json:"version"`
	CreatedAt              time.Time         `json:"created_at"`
}

type PrivacyRequest struct {
	ID               string    `json:"id"`
	RequesterUserID  string    `json:"requester_user_id"`
	OrganizationID   string    `json:"organization_id,omitempty"`
	RequestType      string    `json:"request_type"`
	State            string    `json:"state"`
	IdentityVerified time.Time `json:"identity_verified_at,omitempty"`
	DueAt            time.Time `json:"due_at"`
	Details          string    `json:"details,omitempty"`
	DecisionReason   string    `json:"decision_reason,omitempty"`
	RetentionOutcome string    `json:"retention_outcome,omitempty"`
	LegalHoldApplies bool      `json:"legal_hold_applies"`
	DecidedBy        string    `json:"decided_by,omitempty"`
	SecondApprovedBy string    `json:"second_approved_by,omitempty"`
	Version          int64     `json:"version"`
	CreatedAt        time.Time `json:"created_at"`
	CompletedAt      time.Time `json:"completed_at,omitempty"`
	ExportReference  string    `json:"export_reference,omitempty"`
	ExportExpiresAt  time.Time `json:"export_expires_at,omitempty"`
}

type Store struct {
	recoveryDelivery func(context.Context, RecoveryRequest, string) error
	recoveryReset    func(string) error
	mu               sync.Mutex
	pool             *pgxpool.Pool
	secret           []byte
	now              func() time.Time
	users            map[string]string
	recoveries       map[string]*RecoveryRequest
	evidence         map[string]map[string]bool
	codes            map[string]map[string]bool
	capabilities     map[string]string
	privacy          map[string]*PrivacyRequest
	restrictions     map[string]bool
	rate             map[string][]time.Time
}

func NewStore(secret string) *Store {
	return &Store{secret: []byte(secret), now: func() time.Time { return time.Now().UTC() }, users: map[string]string{}, recoveries: map[string]*RecoveryRequest{}, evidence: map[string]map[string]bool{}, codes: map[string]map[string]bool{}, capabilities: map[string]string{}, privacy: map[string]*PrivacyRequest{}, restrictions: map[string]bool{}, rate: map[string][]time.Time{}}
}

func NewPostgresStore(pool *pgxpool.Pool, secret string) *Store {
	s := NewStore(secret)
	s.pool = pool
	return s
}

func (s *Store) BindUser(userID, email, phone string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if email != "" {
		s.users[normalize(email)] = userID
	}
	if phone != "" {
		s.users[normalize(phone)] = userID
	}
}

func (s *Store) GenerateRecoveryCodes(ctx context.Context, userID string) ([]string, error) {
	if userID == "" {
		return nil, errors.New("user is required")
	}
	codes := make([]string, 10)
	for i := range codes {
		codes[i] = randomReadableCode()
	}
	if s.pool != nil {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err = tx.Exec(ctx, `UPDATE app.account_recovery_codes SET state='REVOKED' WHERE user_id=$1::uuid AND state='ACTIVE'`, userID); err != nil {
			return nil, err
		}
		for _, code := range codes {
			h := s.digest(code)
			if _, err = tx.Exec(ctx, `INSERT INTO app.account_recovery_codes(user_id,code_hash,code_hint,expires_at) VALUES($1::uuid,$2,$3,$4)`, userID, h, code[len(code)-4:], s.now().Add(365*24*time.Hour)); err != nil {
				return nil, err
			}
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, err
		}
		return codes, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	active := map[string]bool{}
	for _, code := range codes {
		active[hex.EncodeToString(s.digest(code))] = true
	}
	s.codes[userID] = active
	return codes, nil
}

// RequestRecovery deliberately returns an empty ID for an unknown identifier.
// HTTP callers always send the same response, preventing account enumeration.
func (s *Store) RequestRecovery(ctx context.Context, identifier, channel, fingerprint string) (string, error) {
	identifier, channel = normalize(identifier), strings.ToLower(strings.TrimSpace(channel))
	if identifier == "" || (channel != "email" && channel != "phone") {
		return "", errors.New("recovery request is invalid")
	}
	if s.pool != nil {
		var attempts int
		if err := s.pool.QueryRow(ctx, `INSERT INTO app.account_recovery_rate_limits(fingerprint_hash) VALUES($1) ON CONFLICT(fingerprint_hash) DO UPDATE SET attempt_count=CASE WHEN app.account_recovery_rate_limits.window_started_at<now()-interval '1 hour' THEN 1 ELSE app.account_recovery_rate_limits.attempt_count+1 END,window_started_at=CASE WHEN app.account_recovery_rate_limits.window_started_at<now()-interval '1 hour' THEN now() ELSE app.account_recovery_rate_limits.window_started_at END RETURNING attempt_count`, s.digest(fingerprint)).Scan(&attempts); err != nil {
			return "", err
		}
		if attempts > 5 {
			return "", nil
		}
		column := "normalized_email"
		if channel == "phone" {
			column = "normalized_phone"
		}
		var userID string
		if err := s.pool.QueryRow(ctx, `SELECT id::text FROM app.users WHERE `+column+`=$1`, identifier).Scan(&userID); errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		} else if err != nil {
			return "", err
		}
		var id string
		err := s.pool.QueryRow(ctx, `INSERT INTO app.account_recovery_requests(target_user_id,requested_channel,request_fingerprint,risk_facts) VALUES($1::uuid,$2,$3,$4) ON CONFLICT(target_user_id) WHERE state IN ('PENDING_VERIFICATION','PENDING_REVIEW','COOLING_OFF','APPROVED') DO NOTHING RETURNING id::text`, userID, channel, s.digest(fingerprint), jsonBytes(map[string]string{"request_channel": channel})).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		if err == nil {
			_, err = s.pool.Exec(ctx, `INSERT INTO app.account_recovery_events(request_id,event_type,actor_reference) VALUES($1::uuid,'account.recovery_requested','public:self-service')`, id)
		}
		return id, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := hex.EncodeToString(s.digest(fingerprint))
	cutoff := s.now().Add(-time.Hour)
	kept := s.rate[key][:0]
	for _, at := range s.rate[key] {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	kept = append(kept, s.now())
	s.rate[key] = kept
	if len(kept) > 5 {
		return "", nil
	}
	userID := s.users[identifier]
	if userID == "" {
		return "", nil
	}
	for _, r := range s.recoveries {
		if r.TargetUserID == userID && (r.State == RecoveryPendingVerification || r.State == RecoveryPendingReview || r.State == RecoveryCoolingOff) {
			return "", nil
		}
	}
	id := randomID()
	now := s.now()
	s.recoveries[id] = &RecoveryRequest{ID: id, TargetUserID: userID, State: RecoveryPendingVerification, RequestedChannel: channel, RiskFacts: map[string]string{"request_channel": channel}, ExpiresAt: now.Add(7 * 24 * time.Hour), Version: 1, CreatedAt: now}
	s.evidence[id] = map[string]bool{}
	return id, nil
}

func (s *Store) AddRecoveryEvidence(ctx context.Context, requestID, factor, proof string) (RecoveryRequest, error) {
	valid := map[string]bool{"recovery_code": true, "verified_email": true, "verified_phone": true, "existing_mfa": true, "business_evidence": true, "manual_identity": true}
	if !valid[factor] || proof == "" {
		return RecoveryRequest{}, errors.New("recovery verification is incomplete")
	}
	if s.pool != nil {
		return s.addRecoveryEvidencePG(ctx, requestID, factor, proof)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.recoveries[requestID]
	if r == nil || r.State != RecoveryPendingVerification || !s.now().Before(r.ExpiresAt) {
		return RecoveryRequest{}, errors.New("recovery request is invalid")
	}
	if factor == "recovery_code" {
		h := hex.EncodeToString(s.digest(proof))
		if !s.codes[r.TargetUserID][h] {
			return RecoveryRequest{}, errors.New("recovery code is invalid")
		}
		delete(s.codes[r.TargetUserID], h)
	}
	if s.evidence[requestID][factor] {
		return RecoveryRequest{}, errors.New("recovery evidence was already used")
	}
	s.evidence[requestID][factor] = true
	r.IndependentFactorCount = len(s.evidence[requestID])
	r.Version++
	if r.IndependentFactorCount >= 2 && !onlyPhone(s.evidence[requestID]) {
		r.State = RecoveryPendingReview
	}
	return *r, nil
}

func (s *Store) addRecoveryEvidencePG(ctx context.Context, requestID, factor, proof string) (RecoveryRequest, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RecoveryRequest{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	r, err := loadRecovery(ctx, tx, requestID, true)
	if err != nil || r.State != RecoveryPendingVerification || !s.now().Before(r.ExpiresAt) {
		return RecoveryRequest{}, errors.New("recovery request is invalid")
	}
	if factor == "recovery_code" {
		var codeID string
		err = tx.QueryRow(ctx, `SELECT id::text FROM app.account_recovery_codes WHERE user_id=$1::uuid AND code_hash=$2 AND state='ACTIVE' AND expires_at>now() FOR UPDATE`, r.TargetUserID, s.digest(proof)).Scan(&codeID)
		if err != nil {
			return RecoveryRequest{}, errors.New("recovery code is invalid")
		}
		if _, err = tx.Exec(ctx, `UPDATE app.account_recovery_codes SET state='USED',used_at=now() WHERE id=$1::uuid`, codeID); err != nil {
			return RecoveryRequest{}, err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO app.account_recovery_evidence(request_id,factor_type,evidence_hash) VALUES($1::uuid,$2,$3)`, requestID, factor, s.digest(proof))
	if err != nil {
		return RecoveryRequest{}, errors.New("recovery evidence was already used")
	}
	var count int
	var nonPhone bool
	if err = tx.QueryRow(ctx, `SELECT count(*),bool_or(factor_type<>'verified_phone') FROM app.account_recovery_evidence WHERE request_id=$1::uuid`, requestID).Scan(&count, &nonPhone); err != nil {
		return RecoveryRequest{}, err
	}
	state := RecoveryPendingVerification
	if count >= 2 && nonPhone {
		state = RecoveryPendingReview
	}
	if err = tx.QueryRow(ctx, `UPDATE app.account_recovery_requests SET independent_factor_count=$2,state=$3,attempt_count=attempt_count+1,version=version+1,updated_at=now() WHERE id=$1::uuid RETURNING version`, requestID, count, state).Scan(&r.Version); err != nil {
		return RecoveryRequest{}, err
	}
	r.IndependentFactorCount = count
	r.State = state
	if _, err = tx.Exec(ctx, `INSERT INTO app.account_recovery_events(request_id,event_type,actor_reference,metadata) VALUES($1::uuid,'account.recovery_verified','public:self-service',jsonb_build_object('factor_type',$2::text))`, requestID, factor); err != nil {
		return RecoveryRequest{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return RecoveryRequest{}, err
	}
	return r, nil
}

func (s *Store) ReviewRecovery(ctx context.Context, requestID, reviewerID, decision, reason string, expectedVersion int64) (RecoveryRequest, string, error) {
	if len(strings.TrimSpace(reason)) < 8 || (decision != "approve" && decision != "reject") {
		return RecoveryRequest{}, "", errors.New("recovery decision is invalid")
	}
	if s.pool == nil {
		return s.reviewRecoveryMemory(ctx, requestID, reviewerID, decision, reason, expectedVersion)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RecoveryRequest{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	r, err := loadRecovery(ctx, tx, requestID, true)
	if err != nil || r.State != RecoveryPendingReview || r.Version != expectedVersion || r.TargetUserID == reviewerID {
		return RecoveryRequest{}, "", errors.New("recovery conflict")
	}
	state := "REJECTED"
	var until *time.Time
	token := ""
	if decision == "approve" {
		state = RecoveryCoolingOff
		t := s.now().Add(24 * time.Hour)
		until = &t
		token = randomToken()
		if err := s.SendRecoveryInstructions(ctx, r, token); err != nil {
			return RecoveryRequest{}, "", err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE app.account_recovery_requests SET state=$2,reviewer_user_id=$3::uuid,review_reason=$4,reviewed_at=now(),cooling_off_until=$5,completion_token_hash=$6,version=version+1,updated_at=now() WHERE id=$1::uuid`, requestID, state, reviewerID, reason, until, nullableDigest(s, token))
	if err != nil {
		return RecoveryRequest{}, "", err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.account_recovery_events(request_id,event_type,actor_user_id,actor_reference,metadata) VALUES($1::uuid,'account.recovery_reviewed',$2::uuid,'platform:reviewer',jsonb_build_object('decision',$3::text,'reason',$4::text))`, requestID, reviewerID, decision, reason); err != nil {
		return RecoveryRequest{}, "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return RecoveryRequest{}, "", err
	}
	r.State = state
	r.ReviewerUserID = reviewerID
	r.ReviewReason = reason
	r.Version++
	if until != nil {
		r.CoolingOffUntil = *until
	}
	return r, token, nil
}

func (s *Store) reviewRecoveryMemory(ctx context.Context, id, reviewer, decision, reason string, version int64) (RecoveryRequest, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.recoveries[id]
	if r == nil || r.State != RecoveryPendingReview || r.Version != version || r.TargetUserID == reviewer {
		return RecoveryRequest{}, "", errors.New("recovery conflict")
	}
	token := ""
	if decision == "approve" {
		token = randomToken()
		if err := s.SendRecoveryInstructions(ctx, *r, token); err != nil {
			return RecoveryRequest{}, "", err
		}
	}
	r.ReviewerUserID = reviewer
	r.ReviewReason = reason
	r.Version++
	if decision == "approve" {
		r.State = RecoveryCoolingOff
		r.CoolingOffUntil = s.now().Add(24 * time.Hour)
		s.capabilities[id] = hex.EncodeToString(s.digest(token))
	} else {
		r.State = "REJECTED"
	}
	return *r, token, nil
}

func (s *Store) CompleteRecovery(ctx context.Context, requestID, token string) (string, error) {
	if s.pool != nil {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return "", err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		var userID string
		err = tx.QueryRow(ctx, `UPDATE app.account_recovery_requests SET state='COMPLETED',completed_at=now(),version=version+1,updated_at=now() WHERE id=$1::uuid AND state='COOLING_OFF' AND cooling_off_until<=now() AND expires_at>now() AND completion_token_hash=$2 RETURNING target_user_id::text`, requestID, s.digest(token)).Scan(&userID)
		if err != nil {
			return "", errors.New("recovery request is invalid or cooling off")
		}
		if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, userID); err != nil {
			return "", err
		}
		// Revoke sessions and the lost MFA method in the same transaction as
		// consuming the completion token. The next login must enroll a new factor.
		if _, err = tx.Exec(ctx, `UPDATE app.mfa_methods SET revoked_at=now() WHERE user_id=$1::uuid AND revoked_at IS NULL`, userID); err != nil {
			return "", err
		}
		if _, err = tx.Exec(ctx, `UPDATE app.sessions SET revoked_at=now() WHERE user_id=$1::uuid AND revoked_at IS NULL`, userID); err != nil {
			return "", err
		}
		if _, err = tx.Exec(ctx, `UPDATE app.account_recovery_codes SET state='REVOKED' WHERE user_id=$1::uuid AND state='ACTIVE'`, userID); err != nil {
			return "", err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO app.account_recovery_events(request_id,event_type,actor_reference) VALUES($1::uuid,'account.recovery_completed','public:self-service')`, requestID); err != nil {
			return "", err
		}
		if err = tx.Commit(ctx); err != nil {
			return "", err
		}
		return userID, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.recoveries[requestID]
	if r == nil || r.State != RecoveryCoolingOff || s.now().Before(r.CoolingOffUntil) || !s.now().Before(r.ExpiresAt) || s.capabilities[requestID] != hex.EncodeToString(s.digest(token)) {
		return "", errors.New("recovery request is invalid or cooling off")
	}
	if s.recoveryReset != nil {
		if err := s.recoveryReset(r.TargetUserID); err != nil {
			return "", err
		}
	}
	r.State = RecoveryCompleted
	r.Version++
	delete(s.codes, r.TargetUserID)
	return r.TargetUserID, nil
}

func (s *Store) CancelRecovery(ctx context.Context, requestID, userID string) error {
	if s.pool != nil {
		tag, err := s.pool.Exec(ctx, `UPDATE app.account_recovery_requests SET state='CANCELLED',cancelled_at=now(),version=version+1,updated_at=now() WHERE id=$1::uuid AND target_user_id=$2::uuid AND state IN ('PENDING_VERIFICATION','PENDING_REVIEW','COOLING_OFF')`, requestID, userID)
		if err != nil || tag.RowsAffected() != 1 {
			return errors.New("recovery request is invalid")
		}
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.recoveries[requestID]
	if r == nil || r.TargetUserID != userID {
		return errors.New("recovery request is invalid")
	}
	r.State = "CANCELLED"
	r.Version++
	return nil
}

func (s *Store) ListRecoveries(ctx context.Context, state string) ([]RecoveryRequest, error) {
	if s.pool == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		out := []RecoveryRequest{}
		for _, r := range s.recoveries {
			if state == "" || r.State == state {
				out = append(out, *r)
			}
		}
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text,target_user_id::text,state,requested_channel,risk_facts,independent_factor_count,COALESCE(reviewer_user_id::text,''),COALESCE(review_reason,''),cooling_off_until,expires_at,version,created_at FROM app.account_recovery_requests WHERE $1='' OR state=$1 ORDER BY created_at`, state)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RecoveryRequest{}
	for rows.Next() {
		var r RecoveryRequest
		var risk []byte
		var cool *time.Time
		if err = rows.Scan(&r.ID, &r.TargetUserID, &r.State, &r.RequestedChannel, &risk, &r.IndependentFactorCount, &r.ReviewerUserID, &r.ReviewReason, &cool, &r.ExpiresAt, &r.Version, &r.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(risk, &r.RiskFacts)
		if cool != nil {
			r.CoolingOffUntil = *cool
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) Recovery(ctx context.Context, id string) (RecoveryRequest, error) {
	if s.pool != nil {
		return loadRecovery(ctx, s.pool, id, false)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.recoveries[id]
	if r == nil {
		return RecoveryRequest{}, errors.New("recovery request is invalid")
	}
	return *r, nil
}

func (s *Store) SensitiveActionsBlocked(ctx context.Context, userID string) bool {
	if s.pool != nil {
		var blocked bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.account_recovery_requests WHERE target_user_id=$1::uuid AND state='COOLING_OFF' AND cooling_off_until>now())`, userID).Scan(&blocked); err != nil {
			return true
		}
		return blocked
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.recoveries {
		if r.TargetUserID == userID && r.State == RecoveryCoolingOff && s.now().Before(r.CoolingOffUntil) {
			return true
		}
	}
	return false
}

func (s *Store) CreatePrivacyRequest(ctx context.Context, userID, orgID, kind, details string) (PrivacyRequest, error) {
	kind = strings.ToUpper(strings.TrimSpace(kind))
	if !validPrivacyType(kind) || len(details) > 2000 {
		return PrivacyRequest{}, errors.New("privacy request is invalid")
	}
	now := s.now()
	if s.pool != nil {
		var r PrivacyRequest
		err := s.pool.QueryRow(ctx, `INSERT INTO app.privacy_requests(requester_user_id,organization_id,request_type,state,identity_verified_at,details) VALUES($1::uuid,NULLIF($2,'')::uuid,$3,'IN_REVIEW',now(),$4) RETURNING id::text,requester_user_id::text,COALESCE(organization_id::text,''),request_type,state,identity_verified_at,due_at,details,version,created_at`, userID, orgID, kind, details).Scan(&r.ID, &r.RequesterUserID, &r.OrganizationID, &r.RequestType, &r.State, &r.IdentityVerified, &r.DueAt, &r.Details, &r.Version, &r.CreatedAt)
		if err == nil {
			_, err = s.pool.Exec(ctx, `INSERT INTO app.privacy_request_events(request_id,event_type,actor_user_id,actor_reference) VALUES($1::uuid,'privacy.request_received',$2::uuid,'user:self-service')`, r.ID, userID)
		}
		return r, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := randomID()
	r := &PrivacyRequest{ID: id, RequesterUserID: userID, OrganizationID: orgID, RequestType: kind, State: "IN_REVIEW", IdentityVerified: now, DueAt: now.Add(30 * 24 * time.Hour), Details: details, Version: 1, CreatedAt: now}
	s.privacy[id] = r
	return *r, nil
}

func (s *Store) ListPrivacyForUser(ctx context.Context, userID string) ([]PrivacyRequest, error) {
	return s.listPrivacy(ctx, `requester_user_id=$1::uuid`, userID)
}
func (s *Store) ListPrivacyReview(ctx context.Context) ([]PrivacyRequest, error) {
	return s.listPrivacy(ctx, `state IN ('IN_REVIEW','CLARIFICATION_REQUIRED','APPROVED','PARTIALLY_APPROVED','IN_PROGRESS')`, "")
}
func (s *Store) listPrivacy(ctx context.Context, where, arg string) ([]PrivacyRequest, error) {
	if s.pool == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		out := []PrivacyRequest{}
		for _, r := range s.privacy {
			if arg == "" || r.RequesterUserID == arg {
				out = append(out, *r)
			}
		}
		return out, nil
	}
	q := `SELECT p.id::text,p.requester_user_id::text,COALESCE(p.organization_id::text,''),p.request_type,p.state,p.identity_verified_at,p.due_at,p.details,COALESCE(p.decision_reason,''),COALESCE(p.retention_outcome,''),p.legal_hold_applies,COALESCE(p.decided_by::text,''),COALESCE(p.second_approved_by::text,''),p.version,p.created_at,p.completed_at,COALESCE(e.object_reference,''),e.expires_at FROM app.privacy_requests p LEFT JOIN app.privacy_exports e ON e.request_id=p.id WHERE ` + where + ` ORDER BY p.created_at DESC`
	var rows pgx.Rows
	var err error
	if arg == "" {
		rows, err = s.pool.Query(ctx, q)
	} else {
		rows, err = s.pool.Query(ctx, q, arg)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PrivacyRequest{}
	for rows.Next() {
		var r PrivacyRequest
		var ident, completed, exportExpiry *time.Time
		if err = rows.Scan(&r.ID, &r.RequesterUserID, &r.OrganizationID, &r.RequestType, &r.State, &ident, &r.DueAt, &r.Details, &r.DecisionReason, &r.RetentionOutcome, &r.LegalHoldApplies, &r.DecidedBy, &r.SecondApprovedBy, &r.Version, &r.CreatedAt, &completed, &r.ExportReference, &exportExpiry); err != nil {
			return nil, err
		}
		if ident != nil {
			r.IdentityVerified = *ident
		}
		if completed != nil {
			r.CompletedAt = *completed
		}
		if exportExpiry != nil {
			r.ExportExpiresAt = *exportExpiry
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) DecidePrivacy(ctx context.Context, id, reviewer, decision, reason string, version int64) (PrivacyRequest, error) {
	decision = strings.ToUpper(strings.TrimSpace(decision))
	if len(strings.TrimSpace(reason)) < 8 {
		return PrivacyRequest{}, errors.New("privacy decision is invalid")
	}
	if s.pool == nil {
		return s.decidePrivacyMemory(id, reviewer, decision, reason, version)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PrivacyRequest{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var r PrivacyRequest
	var hold bool
	err = tx.QueryRow(ctx, `SELECT requester_user_id::text,request_type,state,version,EXISTS(SELECT 1 FROM app.legal_holds h WHERE h.user_id=p.requester_user_id AND h.active) FROM app.privacy_requests p WHERE id=$1::uuid FOR UPDATE`, id).Scan(&r.RequesterUserID, &r.RequestType, &r.State, &r.Version, &hold)
	if err != nil || r.Version != version || reviewer == r.RequesterUserID {
		return PrivacyRequest{}, errors.New("privacy request conflict")
	}
	state := decision
	retention := ""
	if decision == "APPROVED" && (r.RequestType == "DELETION" || r.RequestType == "RESTRICTION") && hold {
		state = "PARTIALLY_APPROVED"
		retention = "financial and legally held records retained; other processing restricted"
	}
	if state != "APPROVED" && state != "PARTIALLY_APPROVED" && state != "REJECTED" && state != "CLARIFICATION_REQUIRED" {
		return PrivacyRequest{}, errors.New("privacy decision is invalid")
	}
	err = tx.QueryRow(ctx, `UPDATE app.privacy_requests SET state=$2,decision_reason=$3,retention_outcome=NULLIF($4,''),legal_hold_applies=$5,decided_by=$6::uuid,decided_at=now(),version=version+1,updated_at=now() WHERE id=$1::uuid RETURNING version,due_at,created_at`, id, state, reason, retention, hold, reviewer).Scan(&r.Version, &r.DueAt, &r.CreatedAt)
	if err != nil {
		return PrivacyRequest{}, err
	}
	if state == "APPROVED" || state == "PARTIALLY_APPROVED" {
		if r.RequestType == "RESTRICTION" || r.RequestType == "DELETION" {
			if _, err = tx.Exec(ctx, `INSERT INTO app.processing_restrictions(user_id,privacy_request_id,scope,reason) VALUES($1::uuid,$2::uuid,'non-essential-processing',$3)`, r.RequesterUserID, id, reason); err != nil {
				return PrivacyRequest{}, err
			}
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.privacy_request_events(request_id,event_type,actor_user_id,actor_reference,metadata) VALUES($1::uuid,'privacy.request_decided',$2::uuid,'platform:reviewer',jsonb_build_object('decision',$3::text,'reason',$4::text))`, id, reviewer, state, reason); err != nil {
		return PrivacyRequest{}, err
	}
	if err != nil {
		return PrivacyRequest{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return PrivacyRequest{}, err
	}
	r.ID = id
	r.State = state
	r.DecisionReason = reason
	r.RetentionOutcome = retention
	r.LegalHoldApplies = hold
	r.DecidedBy = reviewer
	return r, nil
}

func (s *Store) decidePrivacyMemory(id, reviewer, decision, reason string, version int64) (PrivacyRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.privacy[id]
	if r == nil || r.Version != version || reviewer == r.RequesterUserID {
		return PrivacyRequest{}, errors.New("privacy request conflict")
	}
	if decision != "APPROVED" && decision != "REJECTED" && decision != "CLARIFICATION_REQUIRED" {
		return PrivacyRequest{}, errors.New("privacy decision is invalid")
	}
	r.State = decision
	r.DecisionReason = reason
	r.DecidedBy = reviewer
	r.Version++
	if decision == "APPROVED" && (r.RequestType == "RESTRICTION" || r.RequestType == "DELETION") {
		s.restrictions[r.RequesterUserID] = true
	}
	return *r, nil
}

func (s *Store) CompletePrivacy(ctx context.Context, id, reviewer, secondApprover string, version int64) (PrivacyRequest, error) {
	if reviewer == secondApprover || secondApprover == "" {
		return PrivacyRequest{}, errors.New("dual control is required")
	}
	if s.pool == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		r := s.privacy[id]
		if r == nil || r.Version != version || r.DecidedBy != reviewer {
			return PrivacyRequest{}, errors.New("privacy request conflict")
		}
		r.SecondApprovedBy = secondApprover
		r.State = "COMPLETED"
		r.CompletedAt = s.now()
		r.Version++
		if r.RequestType == "ACCESS" || r.RequestType == "PORTABILITY" {
			r.ExportReference = "privacy-export/" + id
			r.ExportExpiresAt = s.now().Add(7 * 24 * time.Hour)
		}
		return *r, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PrivacyRequest{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var r PrivacyRequest
	err = tx.QueryRow(ctx, `UPDATE app.privacy_requests SET state='COMPLETED',second_approved_by=$2::uuid,completed_at=now(),version=version+1,updated_at=now() WHERE id=$1::uuid AND version=$3 AND decided_by IS NOT NULL AND decided_by<>$2::uuid AND state IN ('APPROVED','PARTIALLY_APPROVED') RETURNING requester_user_id::text,request_type,version,completed_at`, id, secondApprover, version).Scan(&r.RequesterUserID, &r.RequestType, &r.Version, &r.CompletedAt)
	if err != nil {
		return PrivacyRequest{}, errors.New("privacy request conflict")
	}
	if r.RequestType == "ACCESS" || r.RequestType == "PORTABILITY" {
		var authoritative []byte
		if err = tx.QueryRow(ctx, `SELECT jsonb_build_object('generated_at',now(),'request_id',$1::text,'profile',to_jsonb(u),'memberships',COALESCE((SELECT jsonb_agg(to_jsonb(m)) FROM app.memberships m WHERE m.user_id=u.id),'[]'::jsonb),'privacy_requests',COALESCE((SELECT jsonb_agg(to_jsonb(pr)-'details') FROM app.privacy_requests pr WHERE pr.requester_user_id=u.id),'[]'::jsonb)) FROM app.users u WHERE u.id=$2::uuid`, id, r.RequesterUserID).Scan(&authoritative); err != nil {
			return PrivacyRequest{}, err
		}
		payload := authoritative
		sum := sha256.Sum256(payload)
		r.ExportReference = "privacy-export/" + id
		r.ExportExpiresAt = s.now().Add(7 * 24 * time.Hour)
		_, err = tx.Exec(ctx, `INSERT INTO app.privacy_exports(request_id,object_reference,content_sha256,payload,expires_at) VALUES($1::uuid,$2,$3,$4::jsonb,$5)`, id, r.ExportReference, hex.EncodeToString(sum[:]), payload, r.ExportExpiresAt)
		if err != nil {
			return PrivacyRequest{}, err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.privacy_request_events(request_id,event_type,actor_user_id,actor_reference) VALUES($1::uuid,'privacy.request_completed',$2::uuid,'platform:second-approver')`, id, secondApprover); err != nil {
		return PrivacyRequest{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return PrivacyRequest{}, err
	}
	r.ID = id
	r.State = "COMPLETED"
	r.SecondApprovedBy = secondApprover
	return r, nil
}

func (s *Store) PrivacyExport(ctx context.Context, requestID, userID string) (json.RawMessage, error) {
	if s.pool == nil {
		return nil, errors.New("privacy export is unavailable in the development memory store")
	}
	var payload []byte
	err := s.pool.QueryRow(ctx, `UPDATE app.privacy_exports e SET downloaded_at=COALESCE(downloaded_at,now()) FROM app.privacy_requests r WHERE e.request_id=r.id AND e.request_id=$1::uuid AND r.requester_user_id=$2::uuid AND r.state='COMPLETED' AND e.expires_at>now() RETURNING e.payload`, requestID, userID).Scan(&payload)
	if err != nil {
		return nil, errors.New("privacy export is unavailable or expired")
	}
	return json.RawMessage(payload), nil
}

func validPrivacyType(v string) bool {
	switch v {
	case "ACCESS", "CORRECTION", "DELETION", "RESTRICTION", "OBJECTION", "CONSENT_WITHDRAWAL", "PORTABILITY":
		return true
	}
	return false
}
func onlyPhone(m map[string]bool) bool { return len(m) == 1 && m["verified_phone"] }
func normalize(v string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(v)), ""))
}
func (s *Store) digest(v string) []byte {
	m := hmac.New(sha256.New, s.secret)
	_, _ = m.Write([]byte(v))
	return m.Sum(nil)
}
func nullableDigest(s *Store, v string) any {
	if v == "" {
		return nil
	}
	return s.digest(v)
}
func jsonBytes(v any) []byte { b, _ := json.Marshal(v); return b }
func randomID() string       { b := make([]byte, 16); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func randomToken() string    { b := make([]byte, 32); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func randomReadableCode() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	h := strings.ToUpper(hex.EncodeToString(b))
	return h[:4] + "-" + h[4:8] + "-" + h[8:12]
}

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadRecovery(ctx context.Context, q rowQuerier, id string, lock bool) (RecoveryRequest, error) {
	suffix := ""
	if lock {
		suffix = " FOR UPDATE"
	}
	var r RecoveryRequest
	var risk []byte
	var cool *time.Time
	err := q.QueryRow(ctx, `SELECT id::text,target_user_id::text,state,requested_channel,risk_facts,independent_factor_count,COALESCE(reviewer_user_id::text,''),COALESCE(review_reason,''),cooling_off_until,expires_at,version,created_at FROM app.account_recovery_requests WHERE id=$1::uuid`+suffix, id).Scan(&r.ID, &r.TargetUserID, &r.State, &r.RequestedChannel, &risk, &r.IndependentFactorCount, &r.ReviewerUserID, &r.ReviewReason, &cool, &r.ExpiresAt, &r.Version, &r.CreatedAt)
	_ = json.Unmarshal(risk, &r.RiskFacts)
	if cool != nil {
		r.CoolingOffUntil = *cool
	}
	return r, err
}

// SetRecoveryDelivery is configured once during runtime construction. Delivery
// must succeed before an approval commits; the raw token is never persisted.
func (s *Store) SetRecoveryDelivery(deliver func(context.Context, RecoveryRequest, string) error) {
	s.recoveryDelivery = deliver
}
func (s *Store) SetRecoveryReset(reset func(string) error) { s.recoveryReset = reset }
func (s *Store) SendRecoveryInstructions(ctx context.Context, r RecoveryRequest, token string) error {
	if s.recoveryDelivery == nil {
		if s.pool != nil {
			return errors.New("recovery delivery is not configured")
		}
		return nil
	}
	return s.recoveryDelivery(ctx, r, token)
}
