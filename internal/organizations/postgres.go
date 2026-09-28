package organizations

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"kredit/internal/access"
	"kredit/internal/onboarding"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore persists businesses and their owners while
// setting the request-local tenant context on every transaction.
type PostgresStore struct {
	pool              *pgxpool.Pool
	guardMu           sync.RWMutex
	createGuard       func(string, CreateInput) error
	organizationLimit int64
}

var _ Service = (*PostgresStore)(nil)

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) SetCreateGuard(guard func(string, CreateInput) error) {
	s.guardMu.Lock()
	defer s.guardMu.Unlock()
	s.createGuard = guard
}

func (s *PostgresStore) SetOrganizationLimit(limit int64) {
	s.guardMu.Lock()
	defer s.guardMu.Unlock()
	s.organizationLimit = limit
}

func (s *PostgresStore) Count() int {
	var count int
	if err := s.pool.QueryRow(context.Background(), `SELECT app.organization_count()`).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *PostgresStore) Create(ownerUserID string, input CreateInput) (Organization, Membership, error) {
	if strings.TrimSpace(ownerUserID) == "" {
		return Organization{}, Membership{}, errors.New("owner user is required")
	}
	if err := validateCreateInput(input); err != nil {
		return Organization{}, Membership{}, err
	}
	s.guardMu.RLock()
	guard := s.createGuard
	limit := s.organizationLimit
	s.guardMu.RUnlock()
	if guard != nil {
		if err := guard(ownerUserID, input); err != nil {
			return Organization{}, Membership{}, err
		}
	}
	now := time.Now().UTC()
	organizationID := newUUID()
	membershipID := newUUID()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := s.beginTenantTxContext(ctx, ownerUserID, organizationID)
	if err != nil {
		return Organization{}, Membership{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if limit > 0 {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('supplier-organization-limit',0))`); err != nil {
			return Organization{}, Membership{}, err
		}
		var count int64
		if err := tx.QueryRow(ctx, `SELECT app.organization_count()`).Scan(&count); err != nil {
			return Organization{}, Membership{}, err
		}
		if count >= limit {
			return Organization{}, Membership{}, errors.New("supplier organization limit reached")
		}
	}
	var organization Organization
	if err := tx.QueryRow(ctx, `
		INSERT INTO app.organizations
			(id, legal_name, trading_name, business_type, registration_info, business_address, industry, status, default_timezone, default_currency, created_at, updated_at, version)
		VALUES ($1, $2, NULLIF($3,''), $4, to_jsonb($5::text), $6, $7, 'onboarding', $8, $9, $10, $10, 1)
		RETURNING id::text, legal_name, COALESCE(trading_name,''), business_type,
		          COALESCE(registration_info #>> '{}',''), business_address, industry,
		          status, default_timezone, default_currency::text, created_at, updated_at, version`,
		organizationID, strings.TrimSpace(input.LegalName), strings.TrimSpace(input.TradingName), strings.TrimSpace(input.BusinessType), strings.TrimSpace(input.RegistrationInfo), strings.TrimSpace(input.BusinessAddress), strings.TrimSpace(input.Industry), defaultOr(input.Timezone, "Africa/Lagos"), defaultOr(input.Currency, "NGN"), now).Scan(
		&organization.ID, &organization.LegalName, &organization.TradingName, &organization.BusinessType, &organization.RegistrationInfo, &organization.BusinessAddress, &organization.Industry, &organization.Status, &organization.DefaultTimezone, &organization.DefaultCurrency, &organization.CreatedAt, &organization.UpdatedAt, &organization.Version); err != nil {
		return Organization{}, Membership{}, fmt.Errorf("create organization: %w", err)
	}
	var membership Membership
	if err := tx.QueryRow(ctx, `
		INSERT INTO app.memberships (id, organization_id, user_id, role, status, created_at)
		VALUES ($1, $2, $3, $4, 'active', $5)
		RETURNING id::text, organization_id::text, user_id::text, role, status, created_at`, membershipID, organizationID, ownerUserID, access.RoleOwner, now).Scan(membershipScanArgs(&membership)...); err != nil {
		return Organization{}, Membership{}, fmt.Errorf("create owner membership: %w", err)
	}
	var emailVerified, phoneVerified bool
	if err := tx.QueryRow(ctx, `SELECT COALESCE(normalized_email,'')<>'',COALESCE(normalized_phone,'')<>'' FROM app.users WHERE id=$1::uuid`, ownerUserID).Scan(&emailVerified, &phoneVerified); err != nil {
		return Organization{}, Membership{}, fmt.Errorf("read owner contact evidence: %w", err)
	}
	if _, err := onboarding.EnsureProfileTx(ctx, tx, organizationID, ownerUserID, emailVerified, phoneVerified, now); err != nil {
		return Organization{}, Membership{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Organization{}, Membership{}, fmt.Errorf("commit organization: %w", err)
	}
	return organization, membership, nil
}

func (s *PostgresStore) Get(organizationID string) (Organization, bool) {
	item, err := s.Read(context.Background(), organizationID)
	return item, err == nil
}

func (s *PostgresStore) Read(ctx context.Context, organizationID string) (Organization, error) {
	tx, err := s.beginTenantTxContext(ctx, "", organizationID)
	if err != nil {
		return Organization{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var organization Organization
	err = tx.QueryRow(ctx, `
		SELECT id::text, legal_name, COALESCE(trading_name,''), business_type,
		       COALESCE(registration_info #>> '{}',''), business_address, industry,
		       status, default_timezone, default_currency::text, created_at, updated_at, version
		FROM app.organizations WHERE id = $1`, organizationID).Scan(
		&organization.ID, &organization.LegalName, &organization.TradingName, &organization.BusinessType, &organization.RegistrationInfo, &organization.BusinessAddress, &organization.Industry, &organization.Status, &organization.DefaultTimezone, &organization.DefaultCurrency, &organization.CreatedAt, &organization.UpdatedAt, &organization.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Organization{}, err
	}
	if err != nil {
		return Organization{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Organization{}, err
	}
	return organization, nil
}

func (s *PostgresStore) ListForUser(userID string) []Organization {
	result, err := s.ReadForUser(context.Background(), userID)
	if err != nil {
		panic(fmt.Errorf("organizations: failed to read organizations for user %s: %w", userID, err))
	}
	return result
}

func (s *PostgresStore) ReadForUser(ctx context.Context, userID string) ([]Organization, error) {
	tx, err := s.pool.Begin(ctx)
	if err == nil {
		_, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id','',true)`, userID)
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `SELECT organization_id::text FROM app.memberships WHERE user_id = $1 AND status='active' AND role='owner' ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var organizationIDs []string
	for rows.Next() {
		var organizationID string
		if err := rows.Scan(&organizationID); err != nil {
			return nil, err
		}
		organizationIDs = append(organizationIDs, organizationID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]Organization, 0, len(organizationIDs))
	for _, organizationID := range organizationIDs {
		if _, err := tx.Exec(ctx, `SELECT set_config('app.current_organization_id', $1, true)`, organizationID); err != nil {
			return nil, err
		}
		var organization Organization
		if err := tx.QueryRow(ctx, `
			SELECT id::text, legal_name, COALESCE(trading_name,''), business_type,
			       COALESCE(registration_info #>> '{}',''), business_address, industry,
			       status, default_timezone, default_currency::text, created_at, updated_at, version
			FROM app.organizations WHERE id = $1`, organizationID).Scan(
			&organization.ID, &organization.LegalName, &organization.TradingName, &organization.BusinessType, &organization.RegistrationInfo, &organization.BusinessAddress, &organization.Industry, &organization.Status, &organization.DefaultTimezone, &organization.DefaultCurrency, &organization.CreatedAt, &organization.UpdatedAt, &organization.Version); err != nil {
			return nil, err
		}
		result = append(result, organization)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

// Membership reports whether the account currently owns the business.
func (s *PostgresStore) Membership(organizationID, userID string) (Membership, bool) {
	membership, err := s.ReadMembership(context.Background(), organizationID, userID)
	return membership, err == nil
}

// ReadMembership is the authorization lookup. A database failure returns that
// failure rather than an empty membership, so the caller can answer "we could
// not check" instead of "you do not have access".
func (s *PostgresStore) ReadMembership(ctx context.Context, organizationID, userID string) (Membership, error) {
	tx, err := s.beginTenantTxContext(ctx, userID, organizationID)
	if err != nil {
		return Membership{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var membership Membership
	err = tx.QueryRow(ctx, `
		SELECT id::text, organization_id::text, user_id::text, role, status, created_at
		FROM app.memberships WHERE organization_id = $1 AND user_id = $2 AND status = 'active' AND role='owner'`, organizationID, userID).Scan(membershipScanArgs(&membership)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrMembershipNotFound
	}
	if err != nil {
		return Membership{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Membership{}, err
	}
	return membership, nil
}

func (s *PostgresStore) beginTenantTx(userID, organizationID string) (pgx.Tx, error) {
	return s.beginTenantTxContext(context.Background(), userID, organizationID)
}

func (s *PostgresStore) beginTenantTxContext(ctx context.Context, userID, organizationID string) (pgx.Tx, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("organization database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_user_id', $1, true), set_config('app.current_organization_id', $2, true)`, userID, organizationID); err != nil {
		_ = tx.Rollback(context.Background())
		return nil, fmt.Errorf("set organization context: %w", err)
	}
	return tx, nil
}

func membershipScanArgs(membership *Membership) []any {
	return []any{&membership.ID, &membership.OrganizationID, &membership.UserID, &membership.Role, &membership.Status, &membership.CreatedAt}
}

func newUUID() string {
	if value, err := uuid.NewV7(); err == nil {
		return value.String()
	}
	return uuid.New().String()
}

func (s *PostgresStore) ReadOwner(ctx context.Context, organizationID string) (Membership, error) {
	tx, err := s.beginTenantTxContext(ctx, "", organizationID)
	if err != nil {
		return Membership{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var owner Membership
	err = tx.QueryRow(ctx, `SELECT id::text, organization_id::text, user_id::text, role, status, created_at
 FROM app.memberships WHERE organization_id=$1::uuid AND role='owner' AND status='active'`, organizationID).Scan(membershipScanArgs(&owner)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrMembershipNotFound
	}
	if err != nil {
		return Membership{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Membership{}, err
	}
	return owner, nil
}
