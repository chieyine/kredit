package organizations

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"kredit/internal/access"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore persists organizations, memberships, and invitations while
// setting the request-local tenant context on every transaction.
type PostgresStore struct {
	pool         *pgxpool.Pool
	tokenHashKey []byte
	guardMu      sync.RWMutex
	createGuard  func(string, CreateInput) error
}

var _ Service = (*PostgresStore)(nil)

func NewPostgresStore(pool *pgxpool.Pool, tokenHashKey string) *PostgresStore {
	if tokenHashKey == "" {
		tokenHashKey = "development-only-change-me"
	}
	return &PostgresStore{pool: pool, tokenHashKey: []byte(tokenHashKey)}
}

func (s *PostgresStore) SetCreateGuard(guard func(string, CreateInput) error) {
	s.guardMu.Lock()
	defer s.guardMu.Unlock()
	s.createGuard = guard
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
	s.guardMu.RUnlock()
	if guard != nil {
		if err := guard(ownerUserID, input); err != nil {
			return Organization{}, Membership{}, err
		}
	}
	now := time.Now().UTC()
	organizationID := newUUID()
	membershipID := newUUID()
	tx, err := s.beginTenantTx(ownerUserID, organizationID)
	if err != nil {
		return Organization{}, Membership{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var organization Organization
	if err := tx.QueryRow(context.Background(), `
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
	if err := tx.QueryRow(context.Background(), `
		INSERT INTO app.memberships (id, organization_id, user_id, role, status, accepted_at, created_at)
		VALUES ($1, $2, $3, $4, 'active', $5, $5)
		RETURNING id::text, organization_id::text, user_id::text, role, status,
		          COALESCE(invited_by::text,''), COALESCE(invited_at, '0001-01-01'::timestamptz),
		          COALESCE(accepted_at, '0001-01-01'::timestamptz), created_at`, membershipID, organizationID, ownerUserID, access.RoleOwner, now).Scan(membershipScanArgs(&membership)...); err != nil {
		return Organization{}, Membership{}, fmt.Errorf("create owner membership: %w", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return Organization{}, Membership{}, fmt.Errorf("commit organization: %w", err)
	}
	return organization, membership, nil
}

func (s *PostgresStore) Get(organizationID string) (Organization, bool) {
	tx, err := s.beginTenantTx("", organizationID)
	if err != nil {
		return Organization{}, false
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var organization Organization
	err = tx.QueryRow(context.Background(), `
		SELECT id::text, legal_name, COALESCE(trading_name,''), business_type,
		       COALESCE(registration_info #>> '{}',''), business_address, industry,
		       status, default_timezone, default_currency::text, created_at, updated_at, version
		FROM app.organizations WHERE id = $1`, organizationID).Scan(
		&organization.ID, &organization.LegalName, &organization.TradingName, &organization.BusinessType, &organization.RegistrationInfo, &organization.BusinessAddress, &organization.Industry, &organization.Status, &organization.DefaultTimezone, &organization.DefaultCurrency, &organization.CreatedAt, &organization.UpdatedAt, &organization.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Organization{}, false
	}
	if err != nil {
		return Organization{}, false
	}
	_ = tx.Commit(context.Background())
	return organization, true
}

func (s *PostgresStore) ListForUser(userID string) []Organization {
	tx, err := s.beginTenantTx(userID, "")
	if err != nil {
		return nil
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	rows, err := tx.Query(context.Background(), `SELECT organization_id::text FROM app.memberships WHERE user_id = $1 AND status NOT IN ('removed','suspended') ORDER BY created_at`, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var organizationIDs []string
	for rows.Next() {
		var organizationID string
		if err := rows.Scan(&organizationID); err != nil {
			return nil
		}
		organizationIDs = append(organizationIDs, organizationID)
	}
	if err := rows.Err(); err != nil {
		return nil
	}
	result := make([]Organization, 0, len(organizationIDs))
	for _, organizationID := range organizationIDs {
		if _, err := tx.Exec(context.Background(), `SELECT set_config('app.current_organization_id', $1, true)`, organizationID); err != nil {
			return nil
		}
		var organization Organization
		if err := tx.QueryRow(context.Background(), `
			SELECT id::text, legal_name, COALESCE(trading_name,''), business_type,
			       COALESCE(registration_info #>> '{}',''), business_address, industry,
			       status, default_timezone, default_currency::text, created_at, updated_at, version
			FROM app.organizations WHERE id = $1`, organizationID).Scan(
			&organization.ID, &organization.LegalName, &organization.TradingName, &organization.BusinessType, &organization.RegistrationInfo, &organization.BusinessAddress, &organization.Industry, &organization.Status, &organization.DefaultTimezone, &organization.DefaultCurrency, &organization.CreatedAt, &organization.UpdatedAt, &organization.Version); err != nil {
			return nil
		}
		result = append(result, organization)
	}
	_ = tx.Commit(context.Background())
	return result
}

func (s *PostgresStore) Membership(organizationID, userID string) (Membership, bool) {
	tx, err := s.beginTenantTx(userID, organizationID)
	if err != nil {
		return Membership{}, false
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var membership Membership
	err = tx.QueryRow(context.Background(), `
		SELECT id::text, organization_id::text, user_id::text, role, status,
		       COALESCE(invited_by::text,''), COALESCE(invited_at, '0001-01-01'::timestamptz),
		       COALESCE(accepted_at, '0001-01-01'::timestamptz), created_at
		FROM app.memberships WHERE organization_id = $1 AND user_id = $2 AND status = 'active'`, organizationID, userID).Scan(membershipScanArgs(&membership)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, false
	}
	if err != nil {
		return Membership{}, false
	}
	_ = tx.Commit(context.Background())
	return membership, true
}

func (s *PostgresStore) ListMembers(organizationID string) []Membership {
	tx, err := s.beginTenantTx("", organizationID)
	if err != nil {
		return nil
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	rows, err := tx.Query(context.Background(), `
		SELECT id::text, organization_id::text, user_id::text, role, status,
		       COALESCE(invited_by::text,''), COALESCE(invited_at, '0001-01-01'::timestamptz),
		       COALESCE(accepted_at, '0001-01-01'::timestamptz), created_at
		FROM app.memberships WHERE organization_id = $1 ORDER BY created_at`, organizationID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := make([]Membership, 0)
	for rows.Next() {
		var membership Membership
		if err := rows.Scan(membershipScanArgs(&membership)...); err != nil {
			return nil
		}
		result = append(result, membership)
	}
	if err := rows.Err(); err != nil {
		return nil
	}
	_ = tx.Commit(context.Background())
	return result
}

func (s *PostgresStore) Invite(actorUserID, organizationID, target, targetType string, targetUserID string, role access.Role) (Invitation, Membership, error) {
	if !role.Valid() || role == access.RoleOwner {
		return Invitation{}, Membership{}, errors.New("only non-owner roles may be invited")
	}
	if strings.TrimSpace(target) == "" || targetUserID == "" || (targetType != "phone" && targetType != "email") {
		return Invitation{}, Membership{}, errors.New("invitation target is required")
	}
	tx, err := s.beginTenantTx(actorUserID, organizationID)
	if err != nil {
		return Invitation{}, Membership{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var exists bool
	if err := tx.QueryRow(context.Background(), `SELECT EXISTS (SELECT 1 FROM app.organizations WHERE id = $1)`, organizationID).Scan(&exists); err != nil || !exists {
		return Invitation{}, Membership{}, errors.New("organisation not found")
	}
	var alreadyMember bool
	if err := tx.QueryRow(context.Background(), `SELECT EXISTS (SELECT 1 FROM app.memberships WHERE organization_id = $1 AND user_id = $2 AND status <> 'removed')`, organizationID, targetUserID).Scan(&alreadyMember); err != nil {
		return Invitation{}, Membership{}, fmt.Errorf("check membership: %w", err)
	}
	if alreadyMember {
		return Invitation{}, Membership{}, errors.New("user already belongs to organisation")
	}
	now := time.Now().UTC()
	expiresAt := now.Add(7 * 24 * time.Hour)
	invitationID, membershipID := newUUID(), newUUID()
	var invitation Invitation
	if err := tx.QueryRow(context.Background(), `
		INSERT INTO app.organization_invitations (id, organization_id, target_type, target_hash, role, status, invited_by, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7, $8)
		RETURNING id::text, organization_id::text, target_type, role, status, invited_by::text, expires_at, created_at`, invitationID, organizationID, targetType, s.hashTarget(targetType, target), role, actorUserID, now, expiresAt).Scan(&invitation.ID, &invitation.OrganizationID, &invitation.TargetType, &invitation.Role, &invitation.Status, &invitation.InvitedBy, &invitation.ExpiresAt, &invitation.CreatedAt); err != nil {
		return Invitation{}, Membership{}, fmt.Errorf("create invitation: %w", err)
	}
	invitation.Target = strings.TrimSpace(target)
	var membership Membership
	if err := tx.QueryRow(context.Background(), `
		INSERT INTO app.memberships (id, organization_id, user_id, role, status, invited_by, invited_at, created_at)
		VALUES ($1, $2, $3, $4, 'invited', $5, $6, $6)
		RETURNING id::text, organization_id::text, user_id::text, role, status,
		          COALESCE(invited_by::text,''), COALESCE(invited_at, '0001-01-01'::timestamptz),
		          COALESCE(accepted_at, '0001-01-01'::timestamptz), created_at`, membershipID, organizationID, targetUserID, role, actorUserID, now).Scan(membershipScanArgs(&membership)...); err != nil {
		return Invitation{}, Membership{}, fmt.Errorf("create invited membership: %w", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return Invitation{}, Membership{}, fmt.Errorf("commit invitation: %w", err)
	}
	return invitation, membership, nil
}

func (s *PostgresStore) ActivateInvitations(userID string) []Membership {
	tx, err := s.beginTenantTx(userID, "")
	if err != nil {
		return nil
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	rows, err := tx.Query(context.Background(), `
		SELECT id::text, organization_id::text
		FROM app.memberships WHERE user_id = $1 AND status = 'invited' FOR UPDATE`, userID)
	if err != nil {
		return nil
	}
	type pending struct{ id, organizationID string }
	var memberships []pending
	for rows.Next() {
		var item pending
		if err := rows.Scan(&item.id, &item.organizationID); err != nil {
			rows.Close()
			return nil
		}
		memberships = append(memberships, item)
	}
	if rows.Err() != nil {
		rows.Close()
		return nil
	}
	rows.Close()
	now := time.Now().UTC()
	result := make([]Membership, 0, len(memberships))
	for _, item := range memberships {
		if _, err := tx.Exec(context.Background(), `SELECT set_config('app.current_organization_id', $1, true)`, item.organizationID); err != nil {
			return nil
		}
		var membership Membership
		if err := tx.QueryRow(context.Background(), `
			UPDATE app.memberships SET status = 'active', accepted_at = $2 WHERE id = $1
			RETURNING id::text, organization_id::text, user_id::text, role, status,
			          COALESCE(invited_by::text,''), COALESCE(invited_at, '0001-01-01'::timestamptz),
			          COALESCE(accepted_at, '0001-01-01'::timestamptz), created_at`, item.id, now).Scan(membershipScanArgs(&membership)...); err != nil {
			return nil
		}
		result = append(result, membership)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil
	}
	return result
}

func (s *PostgresStore) ChangeRole(organizationID, actorUserID, targetUserID string, role access.Role) (Membership, error) {
	if !role.Valid() || role == access.RoleOwner {
		return Membership{}, errors.New("invalid target role")
	}
	tx, err := s.beginTenantTx(actorUserID, organizationID)
	if err != nil {
		return Membership{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var membership Membership
	err = tx.QueryRow(context.Background(), `
		UPDATE app.memberships SET role = $3
		WHERE organization_id = $1 AND user_id = $2 AND status = 'active' AND role <> 'owner' AND user_id <> $4
		RETURNING id::text, organization_id::text, user_id::text, role, status,
		          COALESCE(invited_by::text,''), COALESCE(invited_at, '0001-01-01'::timestamptz),
		          COALESCE(accepted_at, '0001-01-01'::timestamptz), created_at`, organizationID, targetUserID, role, actorUserID).Scan(membershipScanArgs(&membership)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, errors.New("membership not found")
	}
	if err != nil {
		return Membership{}, fmt.Errorf("change member role: %w", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return Membership{}, fmt.Errorf("commit member role: %w", err)
	}
	return membership, nil
}

func (s *PostgresStore) ChangeStatus(organizationID, actorUserID, targetUserID, status string) (Membership, error) {
	if status != "active" && status != "suspended" && status != "removed" {
		return Membership{}, errors.New("invalid membership status")
	}
	tx, err := s.beginTenantTx(actorUserID, organizationID)
	if err != nil {
		return Membership{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var membership Membership
	err = tx.QueryRow(context.Background(), `
		UPDATE app.memberships SET status = $3
		WHERE organization_id = $1 AND user_id = $2 AND status <> 'removed'
		  AND role <> 'owner' AND user_id <> $4
		  AND NOT ($3 = 'active' AND status = 'invited')
		RETURNING id::text, organization_id::text, user_id::text, role, status,
		          COALESCE(invited_by::text,''), COALESCE(invited_at, '0001-01-01'::timestamptz),
		          COALESCE(accepted_at, '0001-01-01'::timestamptz), created_at`, organizationID, targetUserID, status, actorUserID).Scan(membershipScanArgs(&membership)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, errors.New("membership cannot be changed")
	}
	if err != nil {
		return Membership{}, fmt.Errorf("change member status: %w", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return Membership{}, fmt.Errorf("commit member status: %w", err)
	}
	return membership, nil
}

func (s *PostgresStore) beginTenantTx(userID, organizationID string) (pgx.Tx, error) {
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(context.Background(), `SELECT set_config('app.current_user_id', $1, true), set_config('app.current_organization_id', $2, true)`, userID, organizationID); err != nil {
		_ = tx.Rollback(context.Background())
		return nil, fmt.Errorf("set organization context: %w", err)
	}
	return tx, nil
}

func membershipScanArgs(membership *Membership) []any {
	return []any{&membership.ID, &membership.OrganizationID, &membership.UserID, &membership.Role, &membership.Status, &membership.InvitedBy, &membership.InvitedAt, &membership.AcceptedAt, &membership.CreatedAt}
}

func (s *PostgresStore) hashTarget(channel, target string) []byte {
	mac := hmac.New(sha256.New, s.tokenHashKey)
	_, _ = mac.Write([]byte(channel + ":" + normalizeTarget(channel, target)))
	return mac.Sum(nil)
}

func normalizeTarget(channel, target string) string {
	target = strings.ToLower(strings.TrimSpace(target))
	if channel != "phone" {
		return target
	}
	var b strings.Builder
	for _, r := range target {
		if (r >= '0' && r <= '9') || (r == '+' && b.Len() == 0) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func newUUID() string {
	if value, err := uuid.NewV7(); err == nil {
		return value.String()
	}
	return uuid.New().String()
}

func (s *PostgresStore) ReadMembers(organizationID string) ([]Membership, error) {
	tx, err := s.beginTenantTx("", organizationID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	rows, err := tx.Query(context.Background(), `
		SELECT id::text, organization_id::text, user_id::text, role, status,
		       COALESCE(invited_by::text,''), COALESCE(invited_at, '0001-01-01'::timestamptz),
		       COALESCE(accepted_at, '0001-01-01'::timestamptz), created_at
		FROM app.memberships WHERE organization_id = $1 ORDER BY created_at`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Membership, 0)
	for rows.Next() {
		var membership Membership
		if err := rows.Scan(membershipScanArgs(&membership)...); err != nil {
			return nil, err
		}
		result = append(result, membership)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if err = tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return result, nil
}
