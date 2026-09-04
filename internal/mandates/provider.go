package mandates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Status string

const (
	NotStarted Status = "NOT_STARTED"
	Pending    Status = "PENDING"
	Active     Status = "ACTIVE"
	Paused     Status = "PAUSED"
	Cancelled  Status = "CANCELLED"
	Expired    Status = "EXPIRED"
	Failed     Status = "FAILED"
)

type AuthorizationOptions struct {
	AmountCeiling int64 `json:"amount_ceiling_kobo,omitempty"`
}

type AuthorizationInput struct {
	SupplierOrganizationID string
	RequiredUntil          time.Time
	UserID                 string
	BusinessID             string
	AmountCeiling          int64
	Purpose                string
}

type Mandate struct {
	ID                     string    `json:"id"`
	SupplierOrganizationID string    `json:"supplier_organization_id"`
	Provider               string    `json:"provider"`
	ProviderID             string    `json:"provider_id"`
	UserID                 string    `json:"user_id"`
	BusinessID             string    `json:"business_id"`
	Status                 Status    `json:"status"`
	AmountCeiling          int64     `json:"amount_ceiling_kobo"`
	AuthorizationURL       string    `json:"authorization_url,omitempty"`
	StartsAt               time.Time `json:"starts_at,omitempty"`
	EndsAt                 time.Time `json:"ends_at,omitempty"`
	Variable               bool      `json:"variable"`
	MultiAccount           bool      `json:"multi_account_recovery"`
	PartialRecovery        bool      `json:"partial_recovery"`
	CreatedAt              time.Time `json:"created_at"`
	ActivatedAt            time.Time `json:"activated_at,omitempty"`
}

type Provider interface {
	Name() string
	CreateAuthorizationSession(ctx context.Context, input AuthorizationInput) (Mandate, error)
	GetMandate(ctx context.Context, providerID string) (Mandate, error)
	CancelMandate(ctx context.Context, providerID, reason string) (Mandate, error)
	RestoreAuthorization(ctx context.Context, providerID string) (Mandate, error)
}

// TradeLineResolver exposes only the safe persisted facts needed to bind a
// recurring line to an authorization owned by the selected buyer business.
type TradeLineResolver interface {
	ResolveTradeLineMandate(context.Context, string, string, string, string) (Mandate, error)
}

type MockProvider struct {
	mu    sync.RWMutex
	items map[string]Mandate
	now   func() time.Time
}

// PostgresProvider stores provider-neutral mandate state in PostgreSQL while
// retaining the same Provider contract as the deterministic development
// implementation. Real provider API calls remain an explicit adapter concern;
// this boundary prevents mandate state from disappearing on restart.
type PostgresProvider struct {
	pool   *pgxpool.Pool
	name   string
	remote Provider
}

func NewPostgresProviderWithRemote(pool *pgxpool.Pool, remote Provider) *PostgresProvider {
	if remote == nil {
		return NewPostgresProvider(pool, "postgres-mandate")
	}
	return &PostgresProvider{pool: pool, name: remote.Name(), remote: remote}
}

var _ Provider = (*PostgresProvider)(nil)

func NewPostgresProvider(pool *pgxpool.Pool, providerName string) *PostgresProvider {
	if providerName == "" {
		providerName = "postgres-mandate"
	}
	return &PostgresProvider{pool: pool, name: providerName}
}

func (p *PostgresProvider) Name() string { return p.name }

func (p *PostgresProvider) CreateAuthorizationSession(ctx context.Context, input AuthorizationInput) (Mandate, error) {
	if strings.TrimSpace(input.UserID) == "" || strings.TrimSpace(input.BusinessID) == "" || input.AmountCeiling <= 0 {
		return Mandate{}, errors.New("mandate user, business, and positive ceiling are required")
	}
	if p == nil || p.pool == nil {
		return Mandate{}, errors.New("mandate database is not configured")
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return Mandate{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, input.UserID); err != nil {
		return Mandate{}, err
	}
	var acquired bool
	if err = tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended($1,82419))`, p.name+":"+input.BusinessID).Scan(&acquired); err != nil {
		return Mandate{}, err
	}
	if !acquired {
		return Mandate{}, errors.New("mandate authorization is already in progress; retry the same request")
	}
	var saved []byte
	// Reuse only mandates explicitly scoped to the same supplier, with enough
	// unused capacity and a validity period covering the requested horizon.
	err = tx.QueryRow(ctx, `SELECT metadata FROM app.payment_mandates m WHERE provider=$1 AND buyer_subject_id=$2::uuid AND supplier_organization_id=NULLIF($4,'')::uuid AND state IN ('active','pending') AND metadata->>'variable'='true' AND amount_ceiling_kobo >= $3 AND (ends_at IS NULL OR ends_at >= $5) AND (starts_at IS NULL OR starts_at <= now()) AND amount_ceiling_kobo-COALESCE((SELECT SUM(a.succeeded_amount_kobo) FROM app.collection_attempts a JOIN app.collection_reservations r ON r.id=a.reservation_id WHERE r.mandate_id=m.id),0)-COALESCE((SELECT SUM(reserved_amount_kobo) FROM app.collection_reservations r WHERE r.mandate_id=m.id AND r.state IN ('PROCESSING','COMPLETED')),0) >= $3 ORDER BY created_at DESC LIMIT 1`, p.name, input.BusinessID, input.AmountCeiling, input.SupplierOrganizationID, input.RequiredUntil).Scan(&saved)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Mandate{}, err
	}
	if err == nil && len(saved) > 2 {
		var existing Mandate
		if json.Unmarshal(saved, &existing) == nil && existing.ID != "" {
			return existing, nil
		}
	}
	var mandate Mandate
	providerID := uuid.NewString()
	providerState := Active
	if p.remote != nil {
		remote, err := p.remote.CreateAuthorizationSession(ctx, input)
		if err != nil {
			return Mandate{}, err
		}
		mandate = remote
		providerID = remote.ProviderID
		providerState = remote.Status
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO app.payment_mandates
			(buyer_subject_type, buyer_subject_id, provider, provider_mandate_id,
			 mandate_type, amount_ceiling_kobo, state, accepted_disclosure_version)
		VALUES ('business', $1::uuid, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (provider,provider_mandate_id) DO UPDATE SET provider_updated_at=now()
		RETURNING id::text, provider, provider_mandate_id, buyer_subject_id::text,
			amount_ceiling_kobo, state, created_at`,
		input.BusinessID, p.name, providerID, input.Purpose, input.AmountCeiling, strings.ToLower(string(providerState)), "v1",
	).Scan(&mandate.ID, &mandate.Provider, &mandate.ProviderID, &mandate.BusinessID, &mandate.AmountCeiling, &mandate.Status, &mandate.CreatedAt)
	if err != nil {
		return Mandate{}, err
	}
	mandate.UserID = input.UserID
	mandate.SupplierOrganizationID = input.SupplierOrganizationID
	mandate.Status = Status(strings.ToUpper(string(mandate.Status)))
	if mandate.Status == Active {
		mandate.ActivatedAt = mandate.CreatedAt
	}
	metadata, _ := json.Marshal(mandate)
	if _, err = tx.Exec(ctx, `UPDATE app.payment_mandates SET metadata=$2::jsonb,starts_at=$3,ends_at=$4,supplier_organization_id=NULLIF($5,'')::uuid WHERE id=$1::uuid`, mandate.ID, metadata, nullableTime(mandate.StartsAt), nullableTime(mandate.EndsAt), input.SupplierOrganizationID); err != nil {
		return Mandate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Mandate{}, err
	}
	return mandate, nil
}

func (p *PostgresProvider) GetMandate(ctx context.Context, providerID string) (Mandate, error) {
	if p == nil || p.pool == nil {
		return Mandate{}, errors.New("mandate database is not configured")
	}
	var mandate Mandate
	if err := p.pool.QueryRow(ctx, `
		SELECT id, provider, provider_mandate_id, buyer_subject_id, buyer_user_id,
			amount_ceiling_kobo, state, created_at
		FROM app.payment_mandate_by_provider($1, $2)`, p.name, providerID,
	).Scan(&mandate.ID, &mandate.Provider, &mandate.ProviderID, &mandate.BusinessID, &mandate.UserID, &mandate.AmountCeiling, &mandate.Status, &mandate.CreatedAt); err != nil {
		return Mandate{}, errors.New("mandate not found")
	}
	if strings.EqualFold(string(mandate.Status), string(Cancelled)) || strings.EqualFold(string(mandate.Status), string(Expired)) {
		mandate.Status = Status(strings.ToUpper(string(mandate.Status)))
		return mandate, nil
	}
	if p.remote != nil {
		remoteMandate, err := p.remote.GetMandate(ctx, providerID)
		if err != nil {
			return Mandate{}, err
		}
		tx, err := p.pool.Begin(ctx)
		if err != nil {
			return Mandate{}, err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, mandate.UserID); err != nil {
			return Mandate{}, err
		}
		var metadata []byte
		var lockedState string
		if err := tx.QueryRow(ctx, `SELECT metadata,state FROM app.payment_mandates WHERE id=$1::uuid FOR UPDATE`, mandate.ID).Scan(&metadata, &lockedState); err != nil {
			return Mandate{}, err
		}
		var stored Mandate
		if json.Unmarshal(metadata, &stored) == nil && stored.ID != "" {
			mandate = stored
		}
		mandate.Status = Status(strings.ToUpper(lockedState))
		oldState := mandate.Status
		if mandate.Status != Cancelled && mandate.Status != Expired {
			mandate.Status = remoteMandate.Status
		}
		if remoteMandate.AmountCeiling > 0 && remoteMandate.AmountCeiling < mandate.AmountCeiling {
			mandate.AmountCeiling = remoteMandate.AmountCeiling
		}
		if !remoteMandate.StartsAt.IsZero() {
			mandate.StartsAt = remoteMandate.StartsAt
		}
		if !remoteMandate.EndsAt.IsZero() {
			mandate.EndsAt = remoteMandate.EndsAt
		}
		if mandate.Status == Active && mandate.ActivatedAt.IsZero() {
			mandate.ActivatedAt = time.Now().UTC()
		}
		if !mandate.EndsAt.IsZero() && !time.Now().Before(mandate.EndsAt) {
			mandate.Status = Expired
		}
		metadata, _ = json.Marshal(mandate)
		if _, err := tx.Exec(ctx, `UPDATE app.payment_mandates SET state=$3,metadata=$4::jsonb,starts_at=$5,ends_at=$6,amount_ceiling_kobo=$7,provider_updated_at=now() WHERE provider=$1 AND provider_mandate_id=$2`, p.name, providerID, strings.ToLower(string(mandate.Status)), metadata, nullableTime(mandate.StartsAt), nullableTime(mandate.EndsAt), mandate.AmountCeiling); err != nil {
			return Mandate{}, err
		}
		if oldState != mandate.Status {
			if _, err := tx.Exec(ctx, `INSERT INTO app.mandate_events(mandate_id,provider_event_id,old_state,new_state,reason_code,event_at) VALUES($1::uuid,$2,$3,$4,'provider_verified',now())`, mandate.ID, uuid.NewString(), strings.ToLower(string(oldState)), strings.ToLower(string(mandate.Status))); err != nil {
				return Mandate{}, err
			}
			if mandate.Status == Cancelled || mandate.Status == Expired || mandate.Status == Paused || mandate.Status == Failed {
				if _, err := tx.Exec(ctx, `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key) VALUES('payment_mandate',$1,'notification.requested',jsonb_build_object('event','MANDATE_REVOKED','status',$2::text),$3) ON CONFLICT(idempotency_key) DO NOTHING`, mandate.ID, string(mandate.Status), "mandate-verified-block:"+mandate.ID+":"+string(mandate.Status)); err != nil {
					return Mandate{}, err
				}
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return Mandate{}, err
		}

	}
	mandate.Status = Status(strings.ToUpper(string(mandate.Status)))
	if mandate.Status == Active && mandate.ActivatedAt.IsZero() {
		mandate.ActivatedAt = mandate.CreatedAt
	}
	return mandate, nil
}

func (p *PostgresProvider) ResolveTradeLineMandate(ctx context.Context, mandateID, buyerUserID, buyerBusinessID, supplierOrganizationID string) (Mandate, error) {
	if p == nil || p.pool == nil || mandateID == "" || buyerUserID == "" || buyerBusinessID == "" || supplierOrganizationID == "" {
		return Mandate{}, errors.New("mandate, buyer identity, and supplier organization are required")
	}
	var mandate Mandate
	if err := p.pool.QueryRow(ctx, `SELECT * FROM app.trade_line_mandate($1::uuid,$2::uuid,$3::uuid,$4::uuid)`, mandateID, buyerUserID, buyerBusinessID, supplierOrganizationID).Scan(&mandate.ID, &mandate.Provider, &mandate.ProviderID, &mandate.BusinessID, &mandate.UserID, &mandate.SupplierOrganizationID, &mandate.AmountCeiling, &mandate.Status, &mandate.CreatedAt); err != nil {
		return Mandate{}, errors.New("mandate not found for buyer business and supplier")
	}
	mandate.Status = Status(strings.ToUpper(string(mandate.Status)))
	if mandate.Status == Active && mandate.ActivatedAt.IsZero() {
		mandate.ActivatedAt = mandate.CreatedAt
	}
	return mandate, nil
}

func (p *PostgresProvider) CancelMandate(ctx context.Context, providerID, reason string) (Mandate, error) {
	mandate, err := p.GetMandate(ctx, providerID)
	if err != nil {
		return Mandate{}, err
	}
	if strings.TrimSpace(reason) == "" {
		return Mandate{}, errors.New("mandate cancellation reason is required")
	}
	if mandate.Status == Cancelled {
		return mandate, nil
	}
	if p.remote != nil {
		if _, err = p.remote.CancelMandate(ctx, providerID, reason); err != nil {
			return Mandate{}, err
		}
	}
	return p.BlockMandate(ctx, providerID, Cancelled, "buyer-cancel:"+providerID)

}

func (p *PostgresProvider) RestoreAuthorization(ctx context.Context, providerID string) (Mandate, error) {
	previous, err := p.GetMandate(ctx, providerID)
	if err != nil {
		return Mandate{}, err
	}
	if previous.Status != Cancelled && previous.Status != Expired && previous.Status != Failed {
		return Mandate{}, errors.New("only cancelled, expired, or failed mandates can be restored")
	}
	if p.remote != nil {
		restored, restoreErr := p.remote.RestoreAuthorization(ctx, providerID)
		if restoreErr != nil {
			return Mandate{}, restoreErr
		}
		if restored.ProviderID == "" || restored.Status != Active {
			return Mandate{}, errors.New("provider did not return a fresh active mandate")
		}
		if restored.AmountCeiling <= 0 {
			restored.AmountCeiling = previous.AmountCeiling
		}
		restored.UserID, restored.BusinessID, restored.SupplierOrganizationID, restored.Provider = previous.UserID, previous.BusinessID, previous.SupplierOrganizationID, p.name
		err = p.pool.QueryRow(ctx, `INSERT INTO app.payment_mandates(buyer_subject_type,buyer_subject_id,supplier_organization_id,provider,provider_mandate_id,mandate_type,amount_ceiling_kobo,state,accepted_disclosure_version,provider_updated_at) VALUES ('business',$1::uuid,$2::uuid,$3,$4,'restored',$5,'active','v1',now()) RETURNING id::text,created_at`, restored.BusinessID, restored.SupplierOrganizationID, p.name, restored.ProviderID, restored.AmountCeiling).Scan(&restored.ID, &restored.CreatedAt)
		if err != nil {
			return Mandate{}, err
		}
		restored.ActivatedAt = restored.CreatedAt
		return restored, nil
	}
	return p.CreateAuthorizationSession(ctx, AuthorizationInput{SupplierOrganizationID: previous.SupplierOrganizationID, UserID: previous.UserID, BusinessID: previous.BusinessID, AmountCeiling: previous.AmountCeiling, Purpose: "restored"})
}

func NewMockProvider() *MockProvider {
	return &MockProvider{items: make(map[string]Mandate), now: func() time.Time { return time.Now().UTC() }}
}

func (p *MockProvider) Name() string { return "mock-collection" }

func (p *MockProvider) CreateAuthorizationSession(_ context.Context, input AuthorizationInput) (Mandate, error) {
	if input.UserID == "" || input.BusinessID == "" || input.AmountCeiling <= 0 {
		return Mandate{}, errors.New("mandate user, business, and positive ceiling are required")
	}
	now := p.now()
	mandate := Mandate{ID: fmt.Sprintf("mandate-%d", now.UnixNano()), SupplierOrganizationID: input.SupplierOrganizationID, Provider: p.Name(), ProviderID: fmt.Sprintf("mock-mandate-%d", now.UnixNano()), UserID: input.UserID, BusinessID: input.BusinessID, Status: Active, AmountCeiling: input.AmountCeiling, CreatedAt: now, ActivatedAt: now}
	p.mu.Lock()
	p.items[mandate.ProviderID] = mandate
	p.mu.Unlock()
	return mandate, nil
}

func (p *MockProvider) GetMandate(_ context.Context, providerID string) (Mandate, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	mandate, ok := p.items[providerID]
	if !ok {
		return Mandate{}, errors.New("mandate not found")
	}
	return mandate, nil
}

func (p *MockProvider) ResolveTradeLineMandate(_ context.Context, mandateID, buyerUserID, buyerBusinessID, supplierOrganizationID string) (Mandate, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, mandate := range p.items {
		if mandate.ID == mandateID && mandate.UserID == buyerUserID && mandate.BusinessID == buyerBusinessID && mandate.SupplierOrganizationID == supplierOrganizationID {
			return mandate, nil
		}
	}
	return Mandate{}, errors.New("mandate not found for buyer business and supplier")
}

func (p *MockProvider) CancelMandate(_ context.Context, providerID, reason string) (Mandate, error) {
	if strings.TrimSpace(reason) == "" {
		return Mandate{}, errors.New("mandate cancellation reason is required")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	mandate, ok := p.items[providerID]
	if !ok {
		return Mandate{}, errors.New("mandate not found")
	}
	mandate.Status = Cancelled
	p.items[providerID] = mandate
	return mandate, nil
}

func (p *MockProvider) RestoreAuthorization(ctx context.Context, providerID string) (Mandate, error) {
	p.mu.RLock()
	previous, ok := p.items[providerID]
	p.mu.RUnlock()
	if !ok {
		return Mandate{}, errors.New("mandate not found")
	}
	if previous.Status != Cancelled && previous.Status != Expired && previous.Status != Failed {
		return Mandate{}, errors.New("only cancelled, expired, or failed mandates can be restored")
	}
	return p.CreateAuthorizationSession(ctx, AuthorizationInput{SupplierOrganizationID: previous.SupplierOrganizationID, UserID: previous.UserID, BusinessID: previous.BusinessID, AmountCeiling: previous.AmountCeiling, Purpose: "restored"})
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

// BlockingProvider records authenticated revocation signals immediately. A
// delayed or unavailable provider lookup must never allow another debit.
type BlockingProvider interface {
	BlockMandate(context.Context, string, Status, string) (Mandate, error)
}

func (p *PostgresProvider) BlockMandate(ctx context.Context, providerID string, status Status, eventID string) (Mandate, error) {
	if status != Cancelled && status != Paused && status != Expired && status != Failed {
		return Mandate{}, errors.New("provider block must be a non-collectible state")
	}
	var m Mandate
	if err := p.pool.QueryRow(ctx, `SELECT id,provider,provider_mandate_id,buyer_subject_id,buyer_user_id,amount_ceiling_kobo,state,created_at FROM app.payment_mandate_by_provider($1,$2)`, p.name, providerID).Scan(&m.ID, &m.Provider, &m.ProviderID, &m.BusinessID, &m.UserID, &m.AmountCeiling, &m.Status, &m.CreatedAt); err != nil {
		return Mandate{}, err
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return Mandate{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, m.UserID); err != nil {
		return Mandate{}, err
	}
	var metadata []byte
	var old string
	if err = tx.QueryRow(ctx, `SELECT metadata,state FROM app.payment_mandates WHERE id=$1::uuid FOR UPDATE`, m.ID).Scan(&metadata, &old); err != nil {
		return Mandate{}, err
	}
	var stored Mandate
	if json.Unmarshal(metadata, &stored) == nil && stored.ID != "" {
		m = stored
	}
	m.Status = status
	if old == "cancelled" || old == "expired" {
		m.Status = Status(strings.ToUpper(old))
	}
	metadata, _ = json.Marshal(m)
	if _, err = tx.Exec(ctx, `UPDATE app.payment_mandates SET state=$2,metadata=$3::jsonb,provider_updated_at=now() WHERE id=$1::uuid`, m.ID, strings.ToLower(string(m.Status)), metadata); err != nil {
		return Mandate{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.mandate_events(mandate_id,provider_event_id,old_state,new_state,reason_code,event_at) VALUES($1::uuid,$2,$3,$4,'authenticated_provider_block',now()) ON CONFLICT(mandate_id,provider_event_id) DO NOTHING`, m.ID, eventID, old, strings.ToLower(string(m.Status))); err != nil {
		return Mandate{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key) VALUES('payment_mandate',$1,'notification.requested',jsonb_build_object('event','MANDATE_REVOKED','status',$2::text),$3) ON CONFLICT(idempotency_key) DO NOTHING`, m.ID, string(m.Status), "mandate-block:"+m.ID+":"+eventID); err != nil {
		return Mandate{}, err
	}
	return m, tx.Commit(ctx)
}
