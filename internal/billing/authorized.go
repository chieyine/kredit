package billing

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kredit/internal/access"
	"kredit/internal/ledger"
)

type FeeService struct {
	store       *Store
	Active      string
	Providers   map[string]FeeProvider
	Fingerprint func(string) string
}

func NewFeeService(pool *pgxpool.Pool, active string, providers map[string]FeeProvider, fingerprint func(string) string) *FeeService {
	return &FeeService{NewStore(pool), active, providers, fingerprint}
}

type SavedAuthorization struct {
	ID       string    `json:"id"`
	Provider string    `json:"provider"`
	State    string    `json:"state"`
	Customer string    `json:"-"`
	Mandate  string    `json:"mandate_reference"`
	URL      string    `json:"authorization_url"`
	Ceiling  int64     `json:"ceiling_kobo"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	Approved bool      `json:"approved"`
}

const feeAuthSelect = `SELECT id::text,provider,state,customer_reference,mandate_reference,authorization_url,ceiling_kobo,starts_at,ends_at,approved_by IS NOT NULL FROM app.fee_authorizations WHERE organization_id=$1::uuid`

func scanFeeAuth(row pgx.Row) (SavedAuthorization, error) {
	var a SavedAuthorization
	err := row.Scan(&a.ID, &a.Provider, &a.State, &a.Customer, &a.Mandate, &a.URL, &a.Ceiling, &a.StartsAt, &a.EndsAt, &a.Approved)
	return a, err
}
func (s *FeeService) List(ctx context.Context, org, actor string) ([]SavedAuthorization, error) {
	tx, err := s.store.begin(ctx, org, actor)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, feeAuthSelect+` ORDER BY created_at DESC`, org)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SavedAuthorization{}
	for rows.Next() {
		a, e := scanFeeAuth(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, a)
	}
	return items, rows.Err()
}
func lockFeeOwner(ctx context.Context, tx pgx.Tx, org, actor string) error {
	var role access.Role
	if err := tx.QueryRow(ctx, `SELECT m.role FROM app.memberships m JOIN app.users u ON u.id=m.user_id WHERE m.organization_id=$1::uuid AND m.user_id=$2::uuid AND m.status='active' AND u.status='active' FOR SHARE OF m,u`, org, actor).Scan(&role); err != nil {
		return err
	}
	if !access.Can(role, access.PermissionManageFinancial) {
		return errors.New("current business financial authority required")
	}
	return nil
}
func (s *FeeService) Start(ctx context.Context, org, actor string, in FeeCustomer, ceiling int64, consent string) (SavedAuthorization, error) {
	p := s.Providers[s.Active]
	if p == nil || s.Fingerprint == nil || consent != "seller-fees-v1" || ceiling < 20000 || ceiling > 2500000000 {
		return SavedAuthorization{}, errors.New("accept the fee debit terms and choose a valid lifetime ceiling")
	}
	tx, err := s.store.begin(ctx, org, actor)
	if err != nil {
		return SavedAuthorization{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = lockFeeOwner(ctx, tx, org, actor); err != nil {
		return SavedAuthorization{}, err
	}
	if err = tx.QueryRow(ctx, `SELECT legal_name FROM app.organizations WHERE id=$1::uuid`, org).Scan(&in.BusinessName); err != nil {
		return SavedAuthorization{}, err
	}
	if len(in.BVN) != 11 || strings.Trim(in.BVN, "0123456789") != "" || in.Email == "" || in.Phone == "" || len(in.Address) == 0 || len(in.Address) > 100 {
		return SavedAuthorization{}, errors.New("complete the business contact details and shareholder BVN")
	}
	if validator, ok := p.(interface{ ValidateFeeCustomer(FeeCustomer) error }); ok {
		if err = validator.ValidateFeeCustomer(in); err != nil {
			return SavedAuthorization{}, err
		}
	}
	start := time.Now().UTC().Truncate(24 * time.Hour)
	end := start.AddDate(1, 0, 0)
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO app.fee_authorizations(organization_id,provider,state,ceiling_kobo,starts_at,ends_at,identity_fingerprint,consent_version,created_by) VALUES($1::uuid,$2,'customer_pending',$3,$4,$5,$6,$7,$8::uuid) RETURNING id::text`, org, p.Name(), ceiling, start, end, s.Fingerprint(in.BVN), consent, actor).Scan(&id)
	if err != nil {
		return SavedAuthorization{}, errors.New("an existing fee setup must be completed or paused first")
	}
	if err = auditFeeSetup(ctx, tx, org, actor, id, "start"); err != nil {
		return SavedAuthorization{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return SavedAuthorization{}, err
	}
	customer, err := p.CreateFeeCustomer(ctx, in)
	if err != nil {
		return SavedAuthorization{}, errors.New("customer registration is unconfirmed; use admin recovery before retrying")
	}
	if customer == "" {
		return SavedAuthorization{}, errors.New("customer registration needs review")
	}
	return s.saveCustomer(ctx, org, actor, id, customer)
}
func (s *FeeService) saveCustomer(ctx context.Context, org, actor, id, customer string) (SavedAuthorization, error) {
	tx, err := s.store.begin(ctx, org, actor)
	if err != nil {
		return SavedAuthorization{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE app.fee_authorizations SET customer_reference=$3,state='customer_ready',updated_at=now() WHERE id=$1::uuid AND organization_id=$2::uuid AND state='customer_pending'`, id, org, customer)
	if err != nil {
		return SavedAuthorization{}, err
	}
	if tag.RowsAffected() != 1 {
		return SavedAuthorization{}, errors.New("refresh the saved fee setup")
	}
	a, err := scanFeeAuth(tx.QueryRow(ctx, feeAuthSelect+` AND id=$2::uuid`, org, id))
	if err != nil {
		return a, err
	}
	return a, tx.Commit(ctx)
}
func (s *FeeService) Authorize(ctx context.Context, org, actor, id string) (SavedAuthorization, error) {
	tx, err := s.store.begin(ctx, org, actor)
	if err != nil {
		return SavedAuthorization{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = lockFeeOwner(ctx, tx, org, actor); err != nil {
		return SavedAuthorization{}, err
	}
	a, err := scanFeeAuth(tx.QueryRow(ctx, feeAuthSelect+` AND id=$2::uuid FOR UPDATE`, org, id))
	if err != nil {
		return a, err
	}
	if a.State != "customer_ready" {
		return a, errors.New("this authorization is already started; use its saved link or admin recovery")
	}
	p := s.Providers[a.Provider]
	if p == nil || a.Provider != s.Active {
		return a, errors.New("the original account must be active to create the permission")
	}
	if _, err = tx.Exec(ctx, `UPDATE app.fee_authorizations SET state='mandate_pending',updated_at=now() WHERE id=$1::uuid`, id); err != nil {
		return a, err
	}
	if err = auditFeeSetup(ctx, tx, org, actor, id, "authorize"); err != nil {
		return a, err
	}
	if err = tx.Commit(ctx); err != nil {
		return a, err
	}
	v, err := p.CreateFeeAuthorization(ctx, FeeAuthorization{Customer: a.Customer, Reference: "fee-" + id, Ceiling: a.Ceiling, StartsAt: a.StartsAt, EndsAt: a.EndsAt})
	if err != nil {
		return a, errors.New("bank permission setup is unconfirmed; use admin recovery before retrying")
	}
	tx, err = s.store.begin(ctx, org, actor)
	if err != nil {
		return a, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `UPDATE app.fee_authorizations SET mandate_reference=$2,authorization_url=$3,updated_at=now() WHERE id=$1::uuid AND state='mandate_pending' AND mandate_reference=''`, id, v.ID, v.URL); err != nil {
		return a, err
	}
	a.Mandate, a.URL, a.State = v.ID, v.URL, "mandate_pending"
	return a, tx.Commit(ctx)
}

// Providers return day boundaries in either UTC or Lagos time. Consent is to
// the displayed calendar dates; readiness still uses the exact provider clock.
func sameFeeDay(a, b time.Time) bool {
	loc := time.FixedZone("Africa/Lagos", 3600)
	return !a.IsZero() && !b.IsZero() && a.In(loc).Format("2006-01-02") == b.In(loc).Format("2006-01-02")
}
func sameFeeAuthorization(a SavedAuthorization, v FeeAuthorization) bool {
	return v.ID == a.Mandate && v.Customer == a.Customer && v.Reference == "fee-"+a.ID && v.Ceiling == a.Ceiling && sameFeeDay(v.StartsAt, a.StartsAt) && sameFeeDay(v.EndsAt, a.EndsAt)
}

// Review recovers a saved request or approves a provider-confirmed permission.
// Human evidence alone cannot mark a bank mandate active.
func (s *FeeService) Review(ctx context.Context, org, actor, id, action, reference, evidence string) (SavedAuthorization, error) {
	tx, err := s.store.begin(ctx, org, actor)
	if err != nil {
		return SavedAuthorization{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionPlatformOwner); err != nil {
		return SavedAuthorization{}, err
	}
	a, err := scanFeeAuth(tx.QueryRow(ctx, feeAuthSelect+` AND id=$2::uuid FOR UPDATE`, org, id))
	if err != nil {
		return a, err
	}
	if len(strings.TrimSpace(evidence)) < 20 || len(evidence) > 2000 {
		return a, errors.New("record the evidence you reviewed")
	}
	p := s.Providers[a.Provider]
	if p == nil {
		return a, errors.New("save the original provider account before recovery")
	}
	switch action {
	case "customer":
		if a.State != "customer_pending" {
			return a, errors.New("customer recovery is not pending")
		}
		identity, err := p.FeeCustomerIdentity(ctx, reference)
		if err != nil {
			return a, err
		}
		var fingerprint string
		if err = tx.QueryRow(ctx, `SELECT identity_fingerprint FROM app.fee_authorizations WHERE id=$1::uuid`, id).Scan(&fingerprint); err != nil {
			return a, err
		}
		if s.Fingerprint(identity) != fingerprint {
			return a, errors.New("provider customer identity does not match the saved consent")
		}
		_, err = tx.Exec(ctx, `UPDATE app.fee_authorizations SET customer_reference=$2,state='customer_ready',updated_at=now() WHERE id=$1::uuid`, id, reference)
		if err != nil {
			return a, err
		}
	case "approve":
		if a.State != "mandate_pending" && a.State != "ready" {
			return a, errors.New("bank permission is not awaiting review")
		}
		if a.Mandate == "" {
			a.Mandate = reference
		}
		v, err := p.ReadFeeAuthorization(ctx, a.Mandate)
		if err != nil {
			return a, err
		}
		if !sameFeeAuthorization(a, v) || !v.Ready {
			return a, errors.New("the original fee mandate must match the customer, reference, ceiling, dates and bank readiness")
		}
		var billingRef, method string
		if err = tx.QueryRow(ctx, `SELECT billing_provider_reference,billing_method FROM app.supplier_onboarding_profiles WHERE organization_id=$1::uuid FOR UPDATE`, org).Scan(&billingRef, &method); err != nil {
			return a, err
		}
		if method != "authorized_debit" || billingRef != a.ID {
			return a, errors.New("the seller must select this fee permission in billing settings")
		}
		if _, err = tx.Exec(ctx, `UPDATE app.fee_authorizations SET mandate_reference=$2,state='ready',approved_by=$3::uuid,approved_at=now(),updated_at=now() WHERE id=$1::uuid`, id, a.Mandate, actor); err != nil {
			return a, err
		}
		if _, err = tx.Exec(ctx, `UPDATE app.supplier_onboarding_profiles SET billing_state='configured',billing_changed_at=now(),version=version+1 WHERE organization_id=$1::uuid`, org); err != nil {
			return a, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO app.invoice_billing_approvals(organization_id,billing_reference,payment_instructions,approved_by) VALUES($1::uuid,$2,$3,$4::uuid) ON CONFLICT(organization_id) DO UPDATE SET billing_reference=EXCLUDED.billing_reference,payment_instructions=EXCLUDED.payment_instructions,approved_by=EXCLUDED.approved_by,approved_at=now()`, org, id, evidence, actor); err != nil {
			return a, err
		}
	default:
		return a, errors.New("unsupported fee setup review")
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.audit_events(actor_user_id,organization_id,action,resource_type,resource_id,outcome,severity,metadata) VALUES($1::uuid,$2::uuid,'billing.authorization.reviewed','fee_authorization',$3,'success','high',jsonb_build_object('action',$4::text,'evidence',$5::text))`, actor, org, id, action, evidence); err != nil {
		return a, err
	}
	if err = notify(ctx, tx, org, id+":"+action, "FeeAuthorizationReviewed"); err != nil {
		return a, err
	}
	a, err = scanFeeAuth(tx.QueryRow(ctx, feeAuthSelect+` AND id=$2::uuid`, org, id))
	if err != nil {
		return a, err
	}
	return a, tx.Commit(ctx)
}
func (s *FeeService) Pause(ctx context.Context, org, actor, id string) error {
	tx, err := s.store.begin(ctx, org, actor)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = lockFeeOwner(ctx, tx, org, actor); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE app.fee_authorizations SET state='paused',updated_at=now() WHERE organization_id=$1::uuid AND id=$2::uuid AND state IN ('ready','customer_ready')`, org, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("complete or reconcile the pending setup before pausing")
	}
	if _, err = tx.Exec(ctx, `UPDATE app.supplier_onboarding_profiles SET billing_state='pending_verification',billing_changed_at=now(),version=version+1 WHERE organization_id=$1::uuid AND billing_provider_reference=$2`, org, id); err != nil {
		return err
	}
	if err = auditFeeSetup(ctx, tx, org, actor, id, "pause"); err != nil {
		return err
	}
	if err = notify(ctx, tx, org, id+":paused", "FeeAuthorizationPaused"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Run always reconciles old requests; only the currently selected account may
// submit new debits. A failed debit leaves the invoice payable by bank transfer.
func (s *FeeService) Run(ctx context.Context, org string) error {
	tx, err := s.store.begin(ctx, org, "")
	if err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT d.id::text FROM app.fee_debits d WHERE organization_id=$1::uuid AND state='pending' ORDER BY created_at`, org)
	if err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	_ = tx.Rollback(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err = s.reconcile(ctx, org, id); err != nil {
			return err
		}
	}
	if err = s.store.Issue(ctx, org, time.Now()); err != nil {
		return err
	}
	tx, err = s.store.begin(ctx, org, "")
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	a, err := scanFeeAuth(tx.QueryRow(ctx, feeAuthSelect+` AND state='ready' AND approved_by IS NOT NULL FOR UPDATE`, org))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	p := s.Providers[a.Provider]
	if p == nil || a.Provider != s.Active {
		return nil
	}
	var selected bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.supplier_onboarding_profiles WHERE organization_id=$1::uuid AND billing_state='configured' AND billing_method='authorized_debit' AND billing_provider_reference=$2 AND NOT EXISTS(SELECT 1 FROM app.fee_debits d WHERE d.authorization_id=$2::uuid AND d.review_required))`, org, a.ID).Scan(&selected); err != nil || !selected {
		return err
	}
	v, err := p.ReadFeeAuthorization(ctx, a.Mandate)
	if err != nil {
		return err
	}
	if !sameFeeAuthorization(a, v) || !v.Ready {
		return nil
	}
	var invoice string
	err = tx.QueryRow(ctx, `SELECT i.id::text FROM app.fee_invoices i WHERE i.organization_id=$1::uuid AND i.due_at<=now() AND NOT EXISTS(SELECT 1 FROM app.fee_debits d WHERE d.invoice_id=i.id) AND ((SELECT COALESCE(sum(CASE WHEN f.state IN ('waived','refunded') THEN 0 ELSE GREATEST(0,l.amount_kobo+l.collected_at_issue_kobo-GREATEST(0,f.waived_kobo-l.waived_at_issue_kobo)-LEAST(l.collected_at_issue_kobo,f.collected_kobo)) END),0) FROM app.fee_invoice_lines l JOIN app.fees f ON f.id=l.fee_id WHERE l.invoice_id=i.id)-(SELECT COALESCE(sum(CASE WHEN direction='received' THEN amount_kobo ELSE -amount_kobo END),0) FROM app.fee_invoice_receipts WHERE invoice_id=i.id)) BETWEEN 20000 AND 2500000000 ORDER BY i.due_at LIMIT 1`, org).Scan(&invoice)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "fee-invoice:"+invoice); err != nil {
		return err
	}
	var reserved bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.fee_debits WHERE invoice_id=$1::uuid)`, invoice).Scan(&reserved); err != nil || reserved {
		return err
	}
	lockedFees, e := tx.Query(ctx, `SELECT f.id FROM app.fees f JOIN app.fee_invoice_lines l ON l.fee_id=f.id WHERE l.invoice_id=$1::uuid ORDER BY f.id FOR SHARE OF f`, invoice)
	if e != nil {
		return e
	}
	for lockedFees.Next() {
	}
	e = lockedFees.Err()
	lockedFees.Close()
	if e != nil {
		return e
	}
	bill, err := read(ctx, tx, org, invoice)
	if err != nil {
		return err
	}
	if bill.Outstanding < 20000 || bill.Outstanding > 2500000000 {
		return nil
	}
	var used int64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(sum(amount_kobo),0) FROM app.fee_debits WHERE authorization_id=$1::uuid AND state IN ('pending','succeeded','reversed')`, a.ID).Scan(&used); err != nil {
		return err
	}
	if bill.Outstanding > a.Ceiling-used {
		return nil
	}
	var id string
	if err = tx.QueryRow(ctx, `INSERT INTO app.fee_debits(organization_id,authorization_id,invoice_id,amount_kobo,state) VALUES($1::uuid,$2::uuid,$3::uuid,$4,'pending') RETURNING id::text`, org, a.ID, invoice, bill.Outstanding).Scan(&id); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	_, err = p.SubmitFeeDebit(ctx, FeeDebitRequest{a.Mandate, "fee-debit-" + id, bill.Outstanding})
	if err != nil {
		return errors.New("fee debit result is unconfirmed; the saved reference will be reconciled")
	}
	return s.reconcile(ctx, org, id)
}
func (s *FeeService) reconcile(ctx context.Context, org, id string) error {
	tx, err := s.store.begin(ctx, org, "")
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var provider, mandate, invoice, state string
	var amount int64
	if err = tx.QueryRow(ctx, `SELECT a.provider,a.mandate_reference,d.invoice_id::text,d.amount_kobo,d.state FROM app.fee_debits d JOIN app.fee_authorizations a ON a.id=d.authorization_id WHERE d.id=$1::uuid AND d.organization_id=$2::uuid FOR UPDATE OF d`, id, org).Scan(&provider, &mandate, &invoice, &amount, &state); err != nil {
		return err
	}
	if state != "pending" {
		return nil
	}
	p := s.Providers[provider]
	if p == nil {
		return errors.New("original fee account is unavailable")
	}
	result, err := p.ReadFeeDebit(ctx, FeeDebitRequest{mandate, "fee-debit-" + id, amount})
	if err != nil {
		return err
	}
	if result.State != "succeeded" && result.State != "failed" {
		return nil
	}
	if result.State == "succeeded" && result.Amount != amount {
		return errors.New("fee debit amount requires reconciliation")
	}
	if _, err = tx.Exec(ctx, `UPDATE app.fee_debits SET state=$2,checked_at=now() WHERE id=$1::uuid`, id, result.State); err != nil {
		return err
	}
	if result.State == "succeeded" {
		if _, err = ledger.NewPostgresStore(nil).PostFeeProviderDebitTx(ctx, tx, id, ledger.Money(amount), time.Now()); err != nil {
			return err
		}
	}
	if err = notify(ctx, tx, org, id, "FeeDebit"+result.State); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *FeeService) Resume(ctx context.Context, org, actor, id string) error {
	tx, err := s.store.begin(ctx, org, actor)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = lockFeeOwner(ctx, tx, org, actor); err != nil {
		return err
	}
	a, err := scanFeeAuth(tx.QueryRow(ctx, feeAuthSelect+` AND id=$2::uuid FOR UPDATE`, org, id))
	if err != nil {
		return err
	}
	if a.State != "paused" || a.Mandate == "" || a.Provider != s.Active {
		return errors.New("choose a current, previously authorised fee permission")
	}
	p := s.Providers[a.Provider]
	if p == nil {
		return errors.New("original provider account is unavailable")
	}
	v, err := p.ReadFeeAuthorization(ctx, a.Mandate)
	if err != nil {
		return err
	}
	if !sameFeeAuthorization(a, v) || !v.Ready {
		return errors.New("the bank permission must still be current and ready")
	}
	if _, err = tx.Exec(ctx, `UPDATE app.fee_authorizations SET state='mandate_pending',approved_by=NULL,approved_at=NULL,updated_at=now() WHERE id=$1::uuid`, id); err != nil {
		return err
	}
	if err = auditFeeSetup(ctx, tx, org, actor, id, "resume"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Notice is a signal to check the original account. A callback cannot approve
// the seller's fee arrangement or prove that funds reached Kredit's bank.
func (s *FeeService) Notice(ctx context.Context, provider, reference, mandate, event string, review bool) (bool, error) {
	var org, authorization string
	var debit *string
	err := s.store.pool.QueryRow(ctx, `SELECT organization_id::text,authorization_id::text,debit_id::text FROM app.fee_notice_scope($1,$2,$3)`, provider, reference, mandate).Scan(&org, &authorization, &debit)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	tx, err := s.store.begin(ctx, org, "")
	if err != nil {
		return true, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	a, err := scanFeeAuth(tx.QueryRow(ctx, feeAuthSelect+` AND id=$2::uuid`, org, authorization))
	if err != nil {
		return true, err
	}
	if debit != nil && !review && a.Mandate != mandate {
		return true, errors.New("fee notice mandate does not match its saved request")
	}
	if debit != nil {
		if err = s.reconcile(ctx, org, *debit); err != nil {
			return true, err
		}
	}
	kind := "FeeBankPermissionChanged"
	if debit != nil {
		kind = "FeeDebitStatusChanged"
	}
	if review {
		kind = "FeeDebitNeedsReview"
		if debit != nil {
			if _, err = tx.Exec(ctx, `UPDATE app.fee_debits SET review_required=true WHERE id=$1::uuid`, *debit); err != nil {
				return true, err
			}
		}
	}
	if err = notify(ctx, tx, org, authorization+":"+event, kind); err != nil {
		return true, err
	}
	return true, tx.Commit(ctx)
}
func auditFeeSetup(ctx context.Context, tx pgx.Tx, org, actor, id, action string) error {
	_, err := tx.Exec(ctx, `INSERT INTO app.audit_events(actor_user_id,organization_id,action,resource_type,resource_id,outcome,severity,metadata) VALUES($1::uuid,$2::uuid,'billing.authorization.changed','fee_authorization',$3,'success','high',jsonb_build_object('action',$4::text))`, actor, org, id, action)
	return err
}
