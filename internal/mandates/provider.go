package mandates

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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

type AuthorizationInput struct {
	UserID        string
	BusinessID    string
	AmountCeiling int64
	Purpose       string
}

type Mandate struct {
	ID            string    `json:"id"`
	Provider      string    `json:"provider"`
	ProviderID    string    `json:"provider_id"`
	UserID        string    `json:"user_id"`
	BusinessID    string    `json:"business_id"`
	Status        Status    `json:"status"`
	AmountCeiling int64     `json:"amount_ceiling_kobo"`
	CreatedAt     time.Time `json:"created_at"`
	ActivatedAt   time.Time `json:"activated_at,omitempty"`
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
	ResolveTradeLineMandate(context.Context, string, string, string) (Mandate, error)
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
	providerID := uuid.NewString()
	providerState := Active
	if p.remote != nil {
		remoteMandate, err := p.remote.CreateAuthorizationSession(ctx, input)
		if err != nil {
			return Mandate{}, err
		}
		providerID = remoteMandate.ProviderID
		providerState = remoteMandate.Status
	}
	var mandate Mandate
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return Mandate{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_user_id', $1, true)`, input.UserID); err != nil {
		return Mandate{}, err
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
	mandate.Status = Status(strings.ToUpper(string(mandate.Status)))
	if mandate.Status == Active {
		mandate.ActivatedAt = mandate.CreatedAt
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
		if _, err := tx.Exec(ctx, `UPDATE app.payment_mandates SET state=$3,provider_updated_at=now() WHERE provider=$1 AND provider_mandate_id=$2`, p.name, providerID, strings.ToLower(string(remoteMandate.Status))); err != nil {
			return Mandate{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Mandate{}, err
		}
		mandate.Status = remoteMandate.Status
	}
	mandate.Status = Status(strings.ToUpper(string(mandate.Status)))
	if mandate.Status == Active {
		mandate.ActivatedAt = mandate.CreatedAt
	}
	return mandate, nil
}

func (p *PostgresProvider) ResolveTradeLineMandate(ctx context.Context, mandateID, buyerUserID, buyerBusinessID string) (Mandate, error) {
	if p == nil || p.pool == nil || mandateID == "" || buyerUserID == "" || buyerBusinessID == "" {
		return Mandate{}, errors.New("mandate and buyer identity are required")
	}
	var mandate Mandate
	if err := p.pool.QueryRow(ctx, `SELECT * FROM app.trade_line_mandate($1::uuid,$2::uuid,$3::uuid)`, mandateID, buyerUserID, buyerBusinessID).Scan(&mandate.ID, &mandate.Provider, &mandate.ProviderID, &mandate.BusinessID, &mandate.UserID, &mandate.AmountCeiling, &mandate.Status, &mandate.CreatedAt); err != nil {
		return Mandate{}, errors.New("mandate not found for buyer business")
	}
	mandate.Status = Status(strings.ToUpper(string(mandate.Status)))
	if mandate.Status == Active {
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
	if p.remote != nil {
		if _, err = p.remote.CancelMandate(ctx, providerID, reason); err != nil {
			return Mandate{}, err
		}
	}
	if mandate.Status == Cancelled {
		return mandate, nil
	}
	_, err = p.pool.Exec(ctx, `WITH changed AS (UPDATE app.payment_mandates SET state='cancelled',provider_updated_at=now() WHERE provider=$1 AND provider_mandate_id=$2 RETURNING id) INSERT INTO app.mandate_events(mandate_id,provider_event_id,old_state,new_state,reason_code,event_at) SELECT id,$3,$4,'cancelled',$5,now() FROM changed ON CONFLICT DO NOTHING`, p.name, providerID, "buyer-cancel:"+providerID, strings.ToLower(string(mandate.Status)), strings.TrimSpace(reason))
	if err != nil {
		return Mandate{}, err
	}
	mandate.Status = Cancelled
	return mandate, nil
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
		restored.UserID, restored.BusinessID, restored.Provider = previous.UserID, previous.BusinessID, p.name
		err = p.pool.QueryRow(ctx, `INSERT INTO app.payment_mandates(buyer_subject_type,buyer_subject_id,provider,provider_mandate_id,mandate_type,amount_ceiling_kobo,state,accepted_disclosure_version,provider_updated_at) VALUES ('business',$1::uuid,$2,$3,'restored',$4,'active','v1',now()) RETURNING id::text,created_at`, restored.BusinessID, p.name, restored.ProviderID, restored.AmountCeiling).Scan(&restored.ID, &restored.CreatedAt)
		if err != nil {
			return Mandate{}, err
		}
		restored.ActivatedAt = restored.CreatedAt
		return restored, nil
	}
	return p.CreateAuthorizationSession(ctx, AuthorizationInput{UserID: previous.UserID, BusinessID: previous.BusinessID, AmountCeiling: previous.AmountCeiling, Purpose: "restored"})
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
	mandate := Mandate{ID: fmt.Sprintf("mandate-%d", now.UnixNano()), Provider: p.Name(), ProviderID: fmt.Sprintf("mock-mandate-%d", now.UnixNano()), UserID: input.UserID, BusinessID: input.BusinessID, Status: Active, AmountCeiling: input.AmountCeiling, CreatedAt: now, ActivatedAt: now}
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

func (p *MockProvider) ResolveTradeLineMandate(_ context.Context, mandateID, buyerUserID, buyerBusinessID string) (Mandate, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, mandate := range p.items {
		if mandate.ID == mandateID && mandate.UserID == buyerUserID && mandate.BusinessID == buyerBusinessID {
			return mandate, nil
		}
	}
	return Mandate{}, errors.New("mandate not found for buyer business")
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
	return p.CreateAuthorizationSession(ctx, AuthorizationInput{UserID: previous.UserID, BusinessID: previous.BusinessID, AmountCeiling: previous.AmountCeiling, Purpose: "restored"})
}
