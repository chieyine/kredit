package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"kredit/internal/identifier"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool, now: func() time.Time { return time.Now().UTC() }}
}

const profileSelect = `SELECT organization_id::text, version, authorized_representative_name, authorized_representative_title,
 COALESCE(owner_email_verified_at,'0001-01-01'::timestamptz), COALESCE(owner_phone_verified_at,'0001-01-01'::timestamptz),
 kyb_state, COALESCE(kyb_provider_reference,''), COALESCE(kyb_reason_code,''), COALESCE(kyb_submitted_at,'0001-01-01'::timestamptz), COALESCE(kyb_decided_at,'0001-01-01'::timestamptz), COALESCE(kyb_expires_at,'0001-01-01'::timestamptz),
 settlement_state, COALESCE(settlement_provider,''), COALESCE(settlement_provider_reference,''), COALESCE(settlement_bank_name,''), COALESCE(settlement_account_name,''), COALESCE(settlement_account_last4,''), COALESCE(settlement_reason_code,''), COALESCE(settlement_changed_at,'0001-01-01'::timestamptz),
 billing_state, COALESCE(billing_method,''), COALESCE(billing_provider_reference,''), COALESCE(billing_cycle,''), COALESCE(billing_changed_at,'0001-01-01'::timestamptz),
 COALESCE(default_credit_limit_kobo,0), COALESCE(default_payment_days,0), COALESCE(default_grace_hours,0), COALESCE(default_credit_policy_updated_at,'0001-01-01'::timestamptz),
 COALESCE(terms_version,''), COALESCE(terms_accepted_at,'0001-01-01'::timestamptz), COALESCE(privacy_version,''), COALESCE(privacy_accepted_at,'0001-01-01'::timestamptz),
 COALESCE(owner_mfa_verified_at,'0001-01-01'::timestamptz), finance_mfa_complete, readiness_state, readiness_changed_at, created_at, updated_at
 FROM app.supplier_onboarding_profiles WHERE organization_id=$1`

func scanProfile(row pgx.Row) (Profile, error) {
	var p Profile
	err := row.Scan(&p.OrganizationID, &p.Version, &p.AuthorizedRepresentativeName, &p.AuthorizedRepresentativeTitle, &p.OwnerEmailVerifiedAt, &p.OwnerPhoneVerifiedAt, &p.KYBState, &p.KYBProviderReference, &p.KYBReasonCode, &p.KYBSubmittedAt, &p.KYBDecidedAt, &p.KYBExpiresAt, &p.SettlementState, &p.SettlementProvider, &p.SettlementProviderReference, &p.SettlementBankName, &p.SettlementAccountName, &p.SettlementAccountLast4, &p.SettlementReasonCode, &p.SettlementChangedAt, &p.BillingState, &p.BillingMethod, &p.BillingProviderReference, &p.BillingCycle, &p.BillingChangedAt, &p.DefaultCreditLimitKobo, &p.DefaultPaymentDays, &p.DefaultGraceHours, &p.DefaultCreditPolicyUpdatedAt, &p.TermsVersion, &p.TermsAcceptedAt, &p.PrivacyVersion, &p.PrivacyAcceptedAt, &p.OwnerMFAVerifiedAt, &p.FinanceMFAComplete, &p.ReadinessState, &p.ReadinessChangedAt, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (s *PostgresStore) begin(org string) (pgx.Tx, error) {
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(context.Background(), `SELECT set_config('app.current_organization_id',$1,true)`, org); err != nil {
		_ = tx.Rollback(context.Background())
		return nil, err
	}
	return tx, nil
}

func (s *PostgresStore) Ensure(org, actor string, email, phone bool) (Profile, error) {
	tx, err := s.begin(org)
	if err != nil {
		return Profile{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	now := s.now()
	var actorUUID any
	if parsed, parseErr := uuid.Parse(actor); parseErr == nil {
		actorUUID = parsed
	}
	_, err = tx.Exec(context.Background(), `
		WITH inserted AS (
			INSERT INTO app.supplier_onboarding_profiles (organization_id,owner_email_verified_at,owner_phone_verified_at)
			VALUES ($1,$2,$3) ON CONFLICT (organization_id) DO NOTHING RETURNING *
		)
		INSERT INTO app.supplier_onboarding_revisions
			(id,organization_id,profile_version,change_type,actor_user_id,actor_reference,snapshot)
		SELECT public.uuidv7(),organization_id,version,'profile.created',$4,$5,to_jsonb(inserted)
		FROM inserted`, org, timeOrNil(email, now), timeOrNil(phone, now), actorUUID, actor)
	if err != nil {
		return Profile{}, fmt.Errorf("ensure onboarding profile: %w", err)
	}
	p, err := scanProfile(tx.QueryRow(context.Background(), profileSelect, org))
	if err != nil {
		return Profile{}, err
	}
	if err = tx.Commit(context.Background()); err != nil {
		return Profile{}, err
	}
	return p, nil
}
func timeOrNil(ok bool, t time.Time) any {
	if ok {
		return t
	}
	return nil
}

func (s *PostgresStore) Get(org string) (Profile, Summary, error) {
	tx, err := s.begin(org)
	if err != nil {
		return Profile{}, Summary{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	p, err := scanProfile(tx.QueryRow(context.Background(), profileSelect, org))
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, Summary{}, errors.New("onboarding profile not found")
	}
	if err != nil {
		return Profile{}, Summary{}, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return Profile{}, Summary{}, err
	}
	return p, summarize(p, s.now()), nil
}

type memoryMutation func(*Store) (Profile, Summary, error)

func (s *PostgresStore) apply(org, actor, change string, fn memoryMutation) (Profile, Summary, error) {
	tx, err := s.begin(org)
	if err != nil {
		return Profile{}, Summary{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	p, err := scanProfile(tx.QueryRow(context.Background(), profileSelect+` FOR UPDATE`, org))
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, Summary{}, errors.New("onboarding profile not found")
	}
	if err != nil {
		return Profile{}, Summary{}, err
	}
	previousVersion := p.Version
	mem := NewStore()
	mem.now = s.now
	mem.profiles[org] = &p
	next, summary, err := fn(mem)
	if err != nil {
		return Profile{}, Summary{}, err
	}
	snapshot, _ := json.Marshal(next)
	var actorUUID any
	if parsed, parseErr := uuid.Parse(actor); parseErr == nil {
		actorUUID = parsed
	}
	_, err = tx.Exec(context.Background(), `UPDATE app.supplier_onboarding_profiles SET version=$2,authorized_representative_name=$3,authorized_representative_title=$4,owner_email_verified_at=$5,owner_phone_verified_at=$6,kyb_state=$7,kyb_provider_reference=NULLIF($8,''),kyb_reason_code=NULLIF($9,''),kyb_submitted_at=$10,kyb_decided_at=$11,kyb_expires_at=$12,settlement_state=$13,settlement_provider=NULLIF($14,''),settlement_provider_reference=NULLIF($15,''),settlement_bank_name=NULLIF($16,''),settlement_account_name=NULLIF($17,''),settlement_account_last4=NULLIF($18,''),settlement_reason_code=NULLIF($19,''),settlement_changed_at=$20,billing_state=$21,billing_method=NULLIF($22,''),billing_provider_reference=NULLIF($23,''),billing_cycle=NULLIF($24,''),billing_changed_at=$25,default_credit_limit_kobo=NULLIF($26,0),default_payment_days=NULLIF($27,0),default_grace_hours=$28,default_credit_policy_updated_at=$29,terms_version=NULLIF($30,''),terms_accepted_at=$31,terms_accepted_by=CASE WHEN $31::timestamptz IS NULL THEN terms_accepted_by ELSE $32::uuid END,privacy_version=NULLIF($33,''),privacy_accepted_at=$34,privacy_accepted_by=CASE WHEN $34::timestamptz IS NULL THEN privacy_accepted_by ELSE $32::uuid END,owner_mfa_verified_at=$35,finance_mfa_complete=$36,readiness_state=$37,readiness_changed_at=$38,updated_at=$39 WHERE organization_id=$1 AND version=$40`, org, next.Version, next.AuthorizedRepresentativeName, next.AuthorizedRepresentativeTitle, nilTime(next.OwnerEmailVerifiedAt), nilTime(next.OwnerPhoneVerifiedAt), next.KYBState, next.KYBProviderReference, next.KYBReasonCode, nilTime(next.KYBSubmittedAt), nilTime(next.KYBDecidedAt), nilTime(next.KYBExpiresAt), next.SettlementState, next.SettlementProvider, next.SettlementProviderReference, next.SettlementBankName, next.SettlementAccountName, next.SettlementAccountLast4, next.SettlementReasonCode, nilTime(next.SettlementChangedAt), next.BillingState, next.BillingMethod, next.BillingProviderReference, next.BillingCycle, nilTime(next.BillingChangedAt), next.DefaultCreditLimitKobo, next.DefaultPaymentDays, next.DefaultGraceHours, nilTime(next.DefaultCreditPolicyUpdatedAt), next.TermsVersion, nilTime(next.TermsAcceptedAt), actorUUID, next.PrivacyVersion, nilTime(next.PrivacyAcceptedAt), nilTime(next.OwnerMFAVerifiedAt), next.FinanceMFAComplete, next.ReadinessState, next.ReadinessChangedAt, next.UpdatedAt, previousVersion)
	if err != nil {
		return Profile{}, Summary{}, fmt.Errorf("save onboarding profile: %w", err)
	}
	_, err = tx.Exec(context.Background(), `INSERT INTO app.supplier_onboarding_revisions(id,organization_id,profile_version,change_type,actor_user_id,actor_reference,snapshot) VALUES($1,$2,$3,$4,$5,$6,$7)`, identifier.New(), org, next.Version, change, actorUUID, actor, snapshot)
	if err != nil {
		return Profile{}, Summary{}, fmt.Errorf("record onboarding revision: %w", err)
	}
	if err = tx.Commit(context.Background()); err != nil {
		return Profile{}, Summary{}, err
	}
	return next, summary, nil
}
func nilTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func (s *PostgresStore) UpdateRepresentative(o, a string, i RepresentativeInput) (Profile, Summary, error) {
	return s.apply(o, a, "representative.updated", func(m *Store) (Profile, Summary, error) { return m.UpdateRepresentative(o, a, i) })
}
func (s *PostgresStore) RecordContactVerified(o, a, c string) (Profile, Summary, error) {
	return s.apply(o, a, "contact.verified", func(m *Store) (Profile, Summary, error) { return m.RecordContactVerified(o, a, c) })
}
func (s *PostgresStore) SubmitKYB(o, a, r string, v int64) (Profile, Summary, error) {
	return s.apply(o, a, "kyb.submitted", func(m *Store) (Profile, Summary, error) { return m.SubmitKYB(o, a, r, v) })
}
func (s *PostgresStore) RecordKYBDecision(o, a, st, reason string, e time.Time) (Profile, Summary, error) {
	return s.apply(o, a, "kyb.decision", func(m *Store) (Profile, Summary, error) { return m.RecordKYBDecision(o, a, st, reason, e) })
}
func (s *PostgresStore) UpdateSettlement(o, a string, i SettlementInput) (Profile, Summary, error) {
	return s.apply(o, a, "settlement.updated", func(m *Store) (Profile, Summary, error) { return m.UpdateSettlement(o, a, i) })
}
func (s *PostgresStore) RecordSettlementDecision(o, a, st, reason string) (Profile, Summary, error) {
	return s.apply(o, a, "settlement.decision", func(m *Store) (Profile, Summary, error) { return m.RecordSettlementDecision(o, a, st, reason) })
}
func (s *PostgresStore) UpdateBilling(o, a string, i BillingInput) (Profile, Summary, error) {
	return s.apply(o, a, "billing.updated", func(m *Store) (Profile, Summary, error) { return m.UpdateBilling(o, a, i) })
}
func (s *PostgresStore) UpdateCreditPolicy(o, a string, i CreditPolicyInput) (Profile, Summary, error) {
	return s.apply(o, a, "credit_policy.updated", func(m *Store) (Profile, Summary, error) { return m.UpdateCreditPolicy(o, a, i) })
}
func (s *PostgresStore) AcceptConsents(o, a string, v int64, t, p string) (Profile, Summary, error) {
	return s.apply(o, a, "consents.accepted", func(m *Store) (Profile, Summary, error) { return m.AcceptConsents(o, a, v, t, p) })
}
func (s *PostgresStore) SyncSecurity(o, a string, owner, finance bool) (Profile, Summary, error) {
	current, summary, err := s.Get(o)
	if err != nil {
		return Profile{}, Summary{}, err
	}
	if (!current.OwnerMFAVerifiedAt.IsZero()) == owner && current.FinanceMFAComplete == finance {
		return current, summary, nil
	}
	return s.apply(o, a, "security.synced", func(m *Store) (Profile, Summary, error) { return m.SyncSecurity(o, a, owner, finance) })
}
func (s *PostgresStore) Reconcile(now time.Time) []Profile {
	rows, err := s.pool.Query(context.Background(), `SELECT organization_id::text FROM app.reconcile_supplier_onboarding($1)`, now)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []Profile
	for rows.Next() {
		var organizationID string
		if rows.Scan(&organizationID) != nil {
			return nil
		}
		if profile, _, loadErr := s.Get(organizationID); loadErr == nil {
			result = append(result, profile)
		}
	}
	if rows.Err() != nil {
		return nil
	}
	return result
}
