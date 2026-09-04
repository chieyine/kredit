package buyers

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"kredit/internal/identity"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore is the durable buyer identity boundary. Invitation lookup is
// performed through a SECURITY DEFINER function because the public token does
// not reveal an organisation context; all subsequent writes use request-local
// user and organisation RLS settings.
type PostgresStore struct {
	pool        *pgxpool.Pool
	key         []byte
	identity    identity.IdentityProvider
	guardMu     sync.RWMutex
	inviteGuard func(CreateInvitationInput) error
	acceptGuard func(AcceptInput) error
}

var _ Service = (*PostgresStore)(nil)

func NewPostgresStore(pool *pgxpool.Pool, tokenHashKey string, provider identity.IdentityProvider) *PostgresStore {
	if tokenHashKey == "" {
		tokenHashKey = "development-only-change-me"
	}
	return &PostgresStore{pool: pool, key: []byte(tokenHashKey), identity: provider}
}

func (s *PostgresStore) SetInvitationGuard(guard func(CreateInvitationInput) error) {
	s.guardMu.Lock()
	defer s.guardMu.Unlock()
	s.inviteGuard = guard
}

func (s *PostgresStore) SetAcceptanceGuard(guard func(AcceptInput) error) {
	s.guardMu.Lock()
	defer s.guardMu.Unlock()
	s.acceptGuard = guard
}

func (s *PostgresStore) CountBusinesses() int {
	if s == nil || s.pool == nil {
		return 0
	}
	var count int
	if err := s.pool.QueryRow(context.Background(), `SELECT app.business_count()`).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *PostgresStore) CreateInvitation(actorUserID, organizationID string, input CreateInvitationInput) (CreateInvitationResult, error) {
	if strings.TrimSpace(actorUserID) == "" || strings.TrimSpace(organizationID) == "" {
		return CreateInvitationResult{}, errors.New("actor and organization are required")
	}
	if err := validateInvitationInput(input); err != nil {
		return CreateInvitationResult{}, err
	}
	s.guardMu.RLock()
	guard := s.inviteGuard
	s.guardMu.RUnlock()
	if guard != nil {
		if err := guard(input); err != nil {
			return CreateInvitationResult{}, err
		}
	}
	rawToken, err := randomToken()
	if err != nil {
		return CreateInvitationResult{}, err
	}
	ciphertext, err := s.encrypt([]byte(normalizeTarget(input.Target)))
	if err != nil {
		return CreateInvitationResult{}, err
	}
	now := time.Now().UTC()
	expires := now.Add(7 * 24 * time.Hour)
	id := newUUID()
	tx, err := s.beginTx(actorUserID, organizationID)
	if err != nil {
		return CreateInvitationResult{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var invitation Invitation
	if err := tx.QueryRow(context.Background(), `
		INSERT INTO app.buyer_invitations
			(id, organization_id, target_type, target_hash, target_ciphertext,
			 proposed_legal_name, proposed_trading_name, proposed_business_type,
			 proposed_address, proposed_industry, status, created_by, created_at, expires_at, token_hash)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7,''), $8, $9, $10, 'pending', $11, $12, $13, $14)
		RETURNING id::text, organization_id::text, target_type,
			proposed_legal_name, COALESCE(proposed_trading_name,''),
			proposed_business_type, proposed_address, proposed_industry,
			status, expires_at, created_at`, id, organizationID, input.TargetType,
		s.hashTarget(input.TargetType, input.Target), ciphertext,
		strings.TrimSpace(input.LegalName), strings.TrimSpace(input.TradingName),
		strings.TrimSpace(input.BusinessType), strings.TrimSpace(input.BusinessAddress),
		strings.TrimSpace(input.Industry), actorUserID, now, expires, s.hashToken(rawToken)).Scan(
		&invitation.ID, &invitation.OrganizationID, &invitation.TargetType,
		&invitation.ProposedLegalName, &invitation.ProposedTradingName,
		&invitation.ProposedBusinessType, &invitation.ProposedAddress,
		&invitation.ProposedIndustry, &invitation.Status, &invitation.ExpiresAt,
		&invitation.CreatedAt); err != nil {
		return CreateInvitationResult{}, fmt.Errorf("create buyer invitation: %w", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return CreateInvitationResult{}, fmt.Errorf("commit buyer invitation: %w", err)
	}
	return CreateInvitationResult{Invitation: invitation, RawToken: rawToken}, nil
}

func (s *PostgresStore) Preview(rawToken string) (InvitationPreview, error) {
	row, err := s.lookup(rawToken)
	if err != nil {
		return InvitationPreview{}, err
	}
	if row.LookupAttempts >= 10 {
		return InvitationPreview{}, errors.New("invitation lookup rate limit exceeded")
	}
	return InvitationPreview{ID: row.ID, OrganizationID: row.OrganizationID, TargetType: row.TargetType, ProposedLegalName: row.LegalName, ProposedTradingName: row.TradingName, ProposedBusinessType: row.BusinessType, ProposedAddress: row.Address, ProposedIndustry: row.Industry, Status: row.Status, ExpiresAt: row.ExpiresAt}, nil
}

func (s *PostgresStore) InvitationTarget(rawToken string) (string, string, error) {
	row, err := s.lookup(rawToken)
	if err != nil {
		return "", "", err
	}
	if row.Status != "pending" || row.LookupAttempts >= 10 {
		return "", "", errors.New("invitation is no longer available")
	}
	value, err := s.decrypt(row.TargetCiphertext)
	if err != nil {
		return "", "", errors.New("invitation target is unavailable")
	}
	return row.TargetType, string(value), nil
}

func (s *PostgresStore) Accept(ctx context.Context, rawToken, userID string, input AcceptInput) (Portal, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(input.FullName) == "" {
		return Portal{}, errors.New("authenticated user and full name are required")
	}
	if s.identity == nil {
		return Portal{}, errors.New("identity provider is not configured")
	}
	s.guardMu.RLock()
	guard := s.acceptGuard
	s.guardMu.RUnlock()
	if guard != nil {
		if err := guard(input); err != nil {
			return Portal{}, err
		}
	}
	row, err := s.lookup(rawToken)
	if err != nil {
		return Portal{}, err
	}
	if row.Status != "pending" {
		return Portal{}, errors.New("invitation is no longer available")
	}
	legalName := defaultValue(input.LegalName, row.LegalName)
	tradingName := defaultValue(input.TradingName, row.TradingName)
	businessType := defaultValue(input.BusinessType, row.BusinessType)
	address := defaultValue(input.BusinessAddress, row.Address)
	industry := defaultValue(input.Industry, row.Industry)
	if legalName == "" || businessType == "" || address == "" || industry == "" {
		return Portal{}, errors.New("complete buyer business details are required")
	}
	tx, err := s.beginTx(userID, row.OrganizationID)
	if err != nil {
		return Portal{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	// Serialize identity reuse for concurrent invitations for the same user.
	var lockedUser string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM app.users WHERE id=$1::uuid FOR UPDATE`, userID).Scan(&lockedUser); err != nil {
		return Portal{}, err
	}
	personID, businessID, representativeID := newUUID(), newUUID(), newUUID()
	err = tx.QueryRow(ctx, `SELECT id::text FROM app.persons WHERE user_id=$1::uuid`, userID).Scan(&personID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Portal{}, err
	}
	err = tx.QueryRow(ctx, `SELECT id::text FROM app.businesses WHERE owner_user_id=$1::uuid AND lower(legal_name)=lower($2) AND business_address=$3 AND business_type=$4 ORDER BY created_at,id LIMIT 1`, userID, legalName, address, businessType).Scan(&businessID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Portal{}, err
	}

	err = tx.QueryRow(ctx, `SELECT id::text FROM app.business_representatives WHERE business_id=$1::uuid AND person_id=$2::uuid AND authority_verification_status IN ('pending','verified')`, businessID, personID).Scan(&representativeID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Portal{}, err
	}

	personSession, err := s.identity.CreatePersonVerification(ctx, identity.PersonVerificationInput{SubjectID: personID, FullName: strings.TrimSpace(input.FullName)})
	if err != nil {
		return Portal{}, err
	}
	businessSession, err := s.identity.CreateBusinessVerification(ctx, identity.BusinessVerificationInput{SubjectID: businessID, LegalName: legalName, BusinessType: businessType, Address: address})
	if err != nil {
		return Portal{}, err
	}
	authoritySession, err := s.identity.CreateAuthorityVerification(ctx, identity.AuthorityVerificationInput{SubjectID: representativeID, PersonID: personID, BusinessID: businessID, RoleTitle: "authorised representative"})
	if err != nil {
		return Portal{}, err
	}
	if !verificationComplete(personSession) || !verificationComplete(businessSession) || !verificationComplete(authoritySession) {
		return Portal{}, errors.New("identity, business, and authority verification must complete before onboarding")
	}
	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `INSERT INTO app.persons (id, user_id, full_name, status, created_at) VALUES ($1, $2, $3, 'verified', $4) ON CONFLICT (user_id) DO NOTHING`, personID, userID, strings.TrimSpace(input.FullName), now); err != nil {
		return Portal{}, fmt.Errorf("create buyer person: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO app.businesses (id, owner_user_id, legal_name, trading_name, business_type, business_address, industry, status, created_at) VALUES ($1, $2, $3, NULLIF($4,''), $5, $6, $7, 'verified', $8) ON CONFLICT (id) DO NOTHING`, businessID, userID, legalName, tradingName, businessType, address, industry, now); err != nil {
		return Portal{}, fmt.Errorf("create buyer business: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO app.business_representatives (id, business_id, person_id, role_title, authority_type, authority_verification_status, created_at) VALUES ($1, $2, $3, $4, $5, 'verified', $6) ON CONFLICT (id) DO NOTHING`, representativeID, businessID, personID, "authorised representative", "buyer_acceptance", now); err != nil {
		return Portal{}, fmt.Errorf("create buyer representative: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO app.trade_relationships (supplier_organization_id,buyer_business_id,status) VALUES ($1::uuid,$2::uuid,'active') ON CONFLICT (supplier_organization_id,buyer_business_id) DO UPDATE SET status='active',updated_at=now()`, row.OrganizationID, businessID); err != nil {
		return Portal{}, fmt.Errorf("create supplier buyer relationship: %w", err)
	}
	if err := insertVerification(ctx, tx, personID, "person", personSession, now); err != nil {
		return Portal{}, err
	}
	if err := insertVerification(ctx, tx, businessID, "business", businessSession, now); err != nil {
		return Portal{}, err
	}
	if err := insertVerification(ctx, tx, representativeID, "authority", authoritySession, now); err != nil {
		return Portal{}, err
	}
	for _, consent := range []string{"buyer_portal", "identity_verification", "privacy_notice"} {
		if _, err := tx.Exec(ctx, `INSERT INTO app.identity_consents (id, user_id, consent_type, version, accepted_at) VALUES ($1, $2, $3, 'v1', $4)`, newUUID(), userID, consent, now); err != nil {
			return Portal{}, fmt.Errorf("record buyer consent: %w", err)
		}
	}
	result, err := tx.Exec(ctx, `UPDATE app.buyer_invitations SET status = 'accepted', accepted_at = $2, accepted_by_user_id = $3 WHERE id = $1 AND status = 'pending'`, row.ID, now, userID)
	if err != nil || result.RowsAffected() != 1 {
		return Portal{}, errors.New("invitation was accepted by another session")
	}
	if err := tx.Commit(ctx); err != nil {
		return Portal{}, fmt.Errorf("commit buyer acceptance: %w", err)
	}
	return s.Portal(userID)
}

func (s *PostgresStore) ListCustomers(organizationID string) []Customer {
	if s == nil || s.pool == nil || strings.TrimSpace(organizationID) == "" {
		return []Customer{}
	}
	ctx := context.Background()
	tx, err := s.beginTx("", organizationID)
	if err != nil {
		return []Customer{}
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `SELECT buyer_user_id::text,buyer_business_id::text,legal_name,COALESCE(trading_name,''),industry,status FROM app.supplier_customers($1::uuid)`, organizationID)
	if err != nil {
		return []Customer{}
	}
	defer rows.Close()
	result := []Customer{}
	for rows.Next() {
		var customer Customer
		if err := rows.Scan(&customer.BuyerUserID, &customer.BuyerBusinessID, &customer.LegalName, &customer.TradingName, &customer.Industry, &customer.Status); err != nil {
			return []Customer{}
		}
		result = append(result, customer)
	}
	if rows.Err() != nil {
		return []Customer{}
	}
	return result
}

func (s *PostgresStore) Portal(userID string) (Portal, error) {
	tx, err := s.beginTx(userID, "")
	if err != nil {
		return Portal{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var person Person
	if err := tx.QueryRow(context.Background(), `SELECT id::text, user_id::text, full_name, status, created_at FROM app.persons WHERE user_id = $1`, userID).Scan(&person.ID, &person.UserID, &person.FullName, &person.Status, &person.CreatedAt); err != nil {
		return Portal{}, errors.New("buyer portal profile not found")
	}
	var business Business
	var representative Representative
	if err := tx.QueryRow(context.Background(), `SELECT b.id::text, b.owner_user_id::text, b.legal_name, COALESCE(b.trading_name,''), b.business_type, b.business_address, b.industry, b.status, b.created_at, r.id::text, r.person_id::text, r.role_title, r.authority_type, r.authority_verification_status, r.created_at FROM app.businesses b JOIN app.business_representatives r ON r.business_id = b.id WHERE b.owner_user_id = $1 AND r.person_id = $2 ORDER BY b.created_at DESC LIMIT 1`, userID, person.ID).Scan(&business.ID, &business.OwnerUserID, &business.LegalName, &business.TradingName, &business.BusinessType, &business.BusinessAddress, &business.Industry, &business.Status, &business.CreatedAt, &representative.ID, &representative.PersonID, &representative.RoleTitle, &representative.AuthorityType, &representative.AuthorityStatus, &representative.CreatedAt); err != nil {
		return Portal{}, errors.New("buyer business profile not found")
	}
	verificationCases := make([]VerificationCase, 0)
	rows, err := tx.Query(context.Background(), `SELECT id::text, subject_type, subject_id::text, provider, provider_reference, verification_level, state, reasons, safe_result, started_at, completed_at, expires_at FROM app.verification_cases WHERE subject_id IN ($1::uuid, $2::uuid, $3::uuid) ORDER BY started_at`, person.ID, business.ID, representative.ID)
	if err != nil {
		return Portal{}, fmt.Errorf("load buyer verification: %w", err)
	}
	for rows.Next() {
		var item VerificationCase
		var reasons []string
		var safe map[string]string
		var completed, expires *time.Time
		if err := rows.Scan(&item.ID, &item.SubjectType, &item.SubjectID, &item.Provider, &item.ProviderReference, &item.VerificationLevel, &item.State, &reasons, &safe, &item.StartedAt, &completed, &expires); err != nil {
			rows.Close()
			return Portal{}, err
		}
		item.Reasons, item.SafeResult = reasons, safe
		if completed != nil {
			item.CompletedAt = *completed
		}
		if expires != nil {
			item.ExpiresAt = *expires
		}
		verificationCases = append(verificationCases, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Portal{}, err
	}
	consents := make([]Consent, 0)
	consentRows, err := tx.Query(context.Background(), `SELECT id::text, user_id::text, consent_type, version, accepted_at FROM app.identity_consents WHERE user_id = $1 ORDER BY accepted_at`, userID)
	if err != nil {
		return Portal{}, err
	}
	for consentRows.Next() {
		var item Consent
		if err := consentRows.Scan(&item.ID, &item.UserID, &item.ConsentType, &item.Version, &item.AcceptedAt); err != nil {
			consentRows.Close()
			return Portal{}, err
		}
		consents = append(consents, item)
	}
	consentRows.Close()
	if err := consentRows.Err(); err != nil {
		return Portal{}, err
	}
	accounts := make([]BankAccountReference, 0)
	accountRows, err := tx.Query(context.Background(), `SELECT id::text, owner_type, owner_id::text, provider, provider_reference, COALESCE(bank_code,''), COALESCE(masked_account,''), COALESCE(account_name_result,''), COALESCE(account_type,''), ownership_state, active, created_at FROM app.bank_account_references WHERE (owner_type = 'person' AND owner_id = $1) OR (owner_type = 'business' AND owner_id = $2) ORDER BY created_at`, person.ID, business.ID)
	if err != nil {
		return Portal{}, err
	}
	for accountRows.Next() {
		var item BankAccountReference
		if err := accountRows.Scan(&item.ID, &item.OwnerType, &item.OwnerID, &item.Provider, &item.ProviderReference, &item.BankCode, &item.MaskedAccount, &item.AccountNameResult, &item.AccountType, &item.OwnershipState, &item.Active, &item.CreatedAt); err != nil {
			accountRows.Close()
			return Portal{}, err
		}
		accounts = append(accounts, item)
	}
	accountRows.Close()
	if err := accountRows.Err(); err != nil {
		return Portal{}, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return Portal{}, err
	}
	return Portal{Person: person, Business: business, Representative: representative, VerificationCases: verificationCases, Consents: consents, BankAccounts: accounts}, nil
}

func (s *PostgresStore) AddBankAccountReference(userID, ownerType, ownerID string, reference BankAccountReference) (BankAccountReference, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(ownerID) == "" || reference.Provider == "" || reference.ProviderReference == "" {
		return BankAccountReference{}, errors.New("bank account provider reference is required")
	}
	tx, err := s.beginTx(userID, "")
	if err != nil {
		return BankAccountReference{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	reference.ID, reference.OwnerType, reference.OwnerID = newUUID(), ownerType, ownerID
	reference.Active = true
	reference.OwnershipState = defaultValue(reference.OwnershipState, "pending")
	reference.CreatedAt = time.Now().UTC()
	if err := tx.QueryRow(context.Background(), `INSERT INTO app.bank_account_references (id, owner_type, owner_id, provider, provider_reference, bank_code, masked_account, account_name_result, account_type, ownership_state, active, created_at) VALUES ($1, $2, $3, $4, $5, NULLIF($6,''), NULLIF($7,''), NULLIF($8,''), NULLIF($9,''), $10, true, $11) RETURNING id::text`, reference.ID, reference.OwnerType, reference.OwnerID, reference.Provider, reference.ProviderReference, reference.BankCode, reference.MaskedAccount, reference.AccountNameResult, reference.AccountType, reference.OwnershipState, reference.CreatedAt).Scan(&reference.ID); err != nil {
		return BankAccountReference{}, fmt.Errorf("save bank account reference: %w", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return BankAccountReference{}, err
	}
	return reference, nil
}

type invitationRow struct {
	ID, OrganizationID, TargetType                                  string
	TargetHash, TargetCiphertext                                    []byte
	LegalName, TradingName, BusinessType, Address, Industry, Status string
	LookupAttempts                                                  int
	CreatedAt, ExpiresAt                                            time.Time
	AcceptedAt                                                      *time.Time
	AcceptedByUserID                                                *string
}

func (s *PostgresStore) lookup(rawToken string) (invitationRow, error) {
	if strings.TrimSpace(rawToken) == "" {
		return invitationRow{}, errors.New("invitation token is required")
	}
	var row invitationRow
	err := s.pool.QueryRow(context.Background(), `SELECT id::text, organization_id::text, target_type, target_hash, target_ciphertext, proposed_legal_name, COALESCE(proposed_trading_name,''), proposed_business_type, proposed_address, proposed_industry, status, lookup_attempts, created_at, expires_at, accepted_at, accepted_by_user_id::text FROM app.buyer_invitation_by_token_hash($1)`, s.hashToken(rawToken)).Scan(&row.ID, &row.OrganizationID, &row.TargetType, &row.TargetHash, &row.TargetCiphertext, &row.LegalName, &row.TradingName, &row.BusinessType, &row.Address, &row.Industry, &row.Status, &row.LookupAttempts, &row.CreatedAt, &row.ExpiresAt, &row.AcceptedAt, &row.AcceptedByUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return invitationRow{}, errors.New("invitation is invalid or expired")
	}
	if err != nil {
		return invitationRow{}, fmt.Errorf("lookup buyer invitation: %w", err)
	}
	return row, nil
}

func (s *PostgresStore) beginTx(userID, organizationID string) (pgx.Tx, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("buyer database is not configured")
	}
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(context.Background(), `SELECT set_config('app.current_user_id', $1, true), set_config('app.current_organization_id', $2, true)`, userID, organizationID); err != nil {
		_ = tx.Rollback(context.Background())
		return nil, err
	}
	return tx, nil
}

func (s *PostgresStore) hashToken(token string) []byte {
	mac := hmac.New(sha256.New, s.key)
	_, _ = mac.Write([]byte(token))
	return mac.Sum(nil)
}
func (s *PostgresStore) hashTarget(kind, value string) []byte {
	mac := hmac.New(sha256.New, s.key)
	_, _ = mac.Write([]byte(kind + ":" + normalizeTarget(value)))
	return mac.Sum(nil)
}

func (s *PostgresStore) encrypt(value []byte) ([]byte, error) {
	key := sha256.Sum256(s.key)
	block, err := aes.NewCipher(key[:])
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
	return gcm.Seal(nonce, nonce, value, nil), nil
}
func (s *PostgresStore) decrypt(value []byte) ([]byte, error) {
	key := sha256.Sum256(s.key)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(value) < gcm.NonceSize() {
		return nil, errors.New("invalid ciphertext")
	}
	return gcm.Open(nil, value[:gcm.NonceSize()], value[gcm.NonceSize():], nil)
}

func insertVerification(ctx context.Context, tx pgx.Tx, subjectID, subjectType string, session identity.VerificationSession, now time.Time) error {
	reasons, _ := json.Marshal([]string{})
	safe, err := json.Marshal(identity.SafeVerificationResult(session.SafeResult))
	if err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `INSERT INTO app.verification_cases (id, subject_type, subject_id, provider, provider_reference, verification_level, state, reasons, safe_result, started_at, completed_at, expires_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10, $10, $11) ON CONFLICT(provider,provider_reference) DO UPDATE SET state=EXCLUDED.state,reasons=EXCLUDED.reasons,safe_result=EXCLUDED.safe_result,completed_at=EXCLUDED.completed_at,expires_at=EXCLUDED.expires_at WHERE app.verification_cases.subject_id=EXCLUDED.subject_id AND app.verification_cases.subject_type=EXCLUDED.subject_type`, newUUID(), subjectType, subjectID, session.Provider, session.ProviderID, session.VerificationLevel, session.State, reasons, safe, now, session.ExpiresAt)
	if err != nil {
		return fmt.Errorf("save verification case: %w", err)
	}
	if result.RowsAffected() != 1 {
		return errors.New("verification provider reference belongs to another identity")
	}
	return nil
}

func newUUID() string {
	if value, err := uuid.NewV7(); err == nil {
		return value.String()
	}
	return uuid.New().String()
}

func (s *PostgresStore) ReadCustomers(organizationID string) ([]Customer, error) {
	if s == nil || s.pool == nil || strings.TrimSpace(organizationID) == "" {
		return nil, errors.New("customer database or organization is unavailable")
	}
	ctx := context.Background()
	tx, err := s.beginTx("", organizationID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `SELECT buyer_user_id::text,buyer_business_id::text,legal_name,COALESCE(trading_name,''),industry,status FROM app.supplier_customers($1::uuid)`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Customer{}
	for rows.Next() {
		var customer Customer
		if err := rows.Scan(&customer.BuyerUserID, &customer.BuyerBusinessID, &customer.LegalName, &customer.TradingName, &customer.Industry, &customer.Status); err != nil {
			return nil, err
		}
		result = append(result, customer)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
