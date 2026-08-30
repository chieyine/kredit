package credit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"kredit/internal/ledger"
	"kredit/internal/mandates"
	"kredit/internal/payments"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore preserves the existing credit lifecycle invariants while
// durably snapshotting the aggregate after every mutation. The normalized
// tables remain available for reporting and this boundary is deliberately
// fail-closed when PostgreSQL is unavailable.
type PostgresStore struct {
	*Store
	pool *pgxpool.Pool
}

var _ Service = (*PostgresStore)(nil)

func NewPostgresStore(pool *pgxpool.Pool, development *Store) *PostgresStore {
	if development == nil {
		development = NewStore(nil, ledger.NewStore())
	}
	return &PostgresStore{Store: development, pool: pool}
}

func (s *PostgresStore) Create(input CreateInput) (CreditRequest, error) {
	if s.pool == nil {
		return CreditRequest{}, errors.New("credit database is not configured")
	}
	request, err := s.Store.Create(input)
	if err != nil {
		return CreditRequest{}, err
	}
	if err := s.persist(request.ID); err != nil {
		s.discard(request.ID)
		return CreditRequest{}, err
	}
	return request, nil
}

func (s *PostgresStore) ActivateTradeLineDrawdown(input TradeLineActivationInput) (View, *ledger.Transaction, error) {
	if s.pool == nil {
		return View{}, nil, errors.New("credit database is not configured")
	}
	if err := s.hydrateForTenant(input.DrawdownID, input.BuyerUserID, input.SupplierOrganizationID); err == nil {
		view, viewErr := s.Store.GetForSupplier(input.DrawdownID, input.SupplierOrganizationID)
		if viewErr == nil && view.Obligation != nil {
			return view, nil, nil
		}
	}
	s.mu.Lock()
	activationHook := s.onActivated
	s.onActivated = nil
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.onActivated = activationHook
		s.mu.Unlock()
	}()
	view, transaction, err := s.Store.ActivateTradeLineDrawdown(input)
	if err != nil {
		return View{}, transaction, err
	}
	if err := s.persist(input.DrawdownID); err != nil {
		return View{}, transaction, err
	}
	if activationHook != nil && view.Obligation != nil {
		activationHook(view.Request, *view.Obligation)
	}
	return view, transaction, nil
}

// ActivateTradeLineDrawdownTx builds and persists the credit aggregate and
// balanced activation journal inside the caller's trade-line transaction. The
// returned finalize function installs the committed projection only after the
// caller successfully commits.
func (s *PostgresStore) ActivateTradeLineDrawdownTx(ctx context.Context, tx pgx.Tx, input TradeLineActivationInput) (View, *ledger.Transaction, func(), error) {
	if s == nil || s.pool == nil || tx == nil {
		return View{}, nil, nil, errors.New("credit activation transaction is not configured")
	}
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true), set_config('app.current_organization_id',$2,true)`, input.BuyerUserID, input.SupplierOrganizationID); err != nil {
		return View{}, nil, nil, err
	}
	var existingPayload []byte
	err := tx.QueryRow(ctx, `SELECT aggregate FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1`, input.DrawdownID).Scan(&existingPayload)
	if err == nil {
		var existing View
		if err := json.Unmarshal(existingPayload, &existing); err != nil {
			return View{}, nil, nil, fmt.Errorf("decode existing drawdown credit aggregate: %w", err)
		}
		if existing.Obligation == nil {
			return View{}, nil, nil, errors.New("drawdown activation is incomplete")
		}
		return existing, nil, func() { s.installView(existing) }, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return View{}, nil, nil, err
	}
	s.mu.RLock()
	postgresLedger, ok := s.ledger.(*ledger.PostgresStore)
	now := s.now
	newID := s.newID
	mandateProvider := s.mandates
	s.mu.RUnlock()
	if !ok {
		return View{}, nil, nil, errors.New("transactional ledger activation is unavailable")
	}
	temporary := NewStore(mandateProvider, &activationTxLedger{ctx: ctx, tx: tx, store: postgresLedger})
	temporary.now = now
	temporary.newID = newID
	view, transaction, err := temporary.ActivateTradeLineDrawdown(input)
	if err != nil {
		return View{}, transaction, nil, err
	}
	if err := syncNormalizedCredit(ctx, tx, view); err != nil {
		return View{}, transaction, nil, err
	}
	payload, err := json.Marshal(view)
	if err != nil {
		return View{}, transaction, nil, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO app.credit_aggregate_snapshots (credit_request_id,supplier_organization_id,buyer_user_id,aggregate,version,updated_at) VALUES($1,$2,$3,$4::jsonb,$5,$6) ON CONFLICT(credit_request_id) DO NOTHING`, view.Request.ID, view.Request.SupplierOrganizationID, view.Request.BuyerUserID, payload, view.Request.Version, view.Request.UpdatedAt); err != nil {
		return View{}, transaction, nil, err
	}
	return view, transaction, func() { s.installView(view) }, nil
}

type activationTxLedger struct {
	ctx   context.Context
	tx    pgx.Tx
	store *ledger.PostgresStore
}

func (l *activationTxLedger) PostActivation(id string, principal ledger.Money, at time.Time, key string) (ledger.Transaction, error) {
	return l.store.PostActivationTx(l.ctx, l.tx, id, principal, at, key)
}
func (*activationTxLedger) PostPayment(string, ledger.Money, string, time.Time, string) (ledger.Transaction, error) {
	return ledger.Transaction{}, errors.New("unsupported transactional ledger operation")
}
func (*activationTxLedger) PostPaymentReversal(string, ledger.Money, string, time.Time, string) (ledger.Transaction, error) {
	return ledger.Transaction{}, errors.New("unsupported transactional ledger operation")
}
func (*activationTxLedger) PostCollectionFee(string, ledger.Money, time.Time, string) (ledger.Transaction, error) {
	return ledger.Transaction{}, errors.New("unsupported transactional ledger operation")
}
func (*activationTxLedger) PostCollectionFeeReversal(string, ledger.Money, time.Time, string) (ledger.Transaction, error) {
	return ledger.Transaction{}, errors.New("unsupported transactional ledger operation")
}
func (*activationTxLedger) PostAdjustment(string, ledger.Money, string, time.Time, string) (ledger.Transaction, error) {
	return ledger.Transaction{}, errors.New("unsupported transactional ledger operation")
}
func (*activationTxLedger) PostFeeWaiver(string, ledger.Money, time.Time, string) (ledger.Transaction, error) {
	return ledger.Transaction{}, errors.New("unsupported transactional ledger operation")
}
func (*activationTxLedger) GetByReference(string) ([]ledger.Transaction, error) {
	return nil, errors.New("unsupported transactional ledger operation")
}

func (s *PostgresStore) UpdateDraft(requestID, actorID string, input UpdateDraftInput) (CreditRequest, error) {
	view, err := s.mutate(requestID, func() (View, error) {
		request, err := s.Store.UpdateDraft(requestID, actorID, input)
		if err != nil {
			return View{}, err
		}
		return s.Store.GetForSupplier(request.ID, request.SupplierOrganizationID)
	})
	return view.Request, err
}

func (s *PostgresStore) Send(requestID, actorID string) (View, error) {
	return s.mutate(requestID, func() (View, error) { return s.Store.Send(requestID, actorID) })
}
func (s *PostgresStore) Cancel(requestID, actorID string) (View, error) {
	return s.mutate(requestID, func() (View, error) { return s.Store.Cancel(requestID, actorID) })
}
func (s *PostgresStore) Review(requestID, buyerUserID string) (View, error) {
	return s.mutate(requestID, func() (View, error) { return s.Store.Review(requestID, buyerUserID) })
}
func (s *PostgresStore) Decline(requestID, buyerUserID string) (View, error) {
	return s.mutate(requestID, func() (View, error) { return s.Store.Decline(requestID, buyerUserID) })
}
func (s *PostgresStore) AuthorizeMandate(ctx context.Context, requestID, buyerUserID string) (View, error) {
	return s.mutate(requestID, func() (View, error) { return s.Store.AuthorizeMandate(ctx, requestID, buyerUserID) })
}
func (s *PostgresStore) SetMandate(requestID, buyerUserID string, mandate mandates.Mandate) (View, error) {
	return s.mutate(requestID, func() (View, error) { return s.Store.SetMandate(requestID, buyerUserID, mandate) })
}
func (s *PostgresStore) Accept(requestID, buyerUserID, agreementID, agreementHash, mandateID, authLevel string, identityVerified, authorityVerified bool) (View, error) {
	return s.mutate(requestID, func() (View, error) {
		return s.Store.Accept(requestID, buyerUserID, agreementID, agreementHash, mandateID, authLevel, identityVerified, authorityVerified)
	})
}
func (s *PostgresStore) Release(requestID, supplierOrgID, actorID, deliveryMethod, notes string) (View, error) {
	return s.mutate(requestID, func() (View, error) { return s.Store.Release(requestID, supplierOrgID, actorID, deliveryMethod, notes) })
}
func (s *PostgresStore) RecordReceipt(requestID, buyerUserID, state, issueReason string) (View, *ledger.Transaction, error) {
	if err := s.hydrate(requestID); err != nil {
		return View{}, nil, err
	}
	// The database obligation must exist before the schedule hook inserts rows
	// that reference it. Defer the hook until the normalized aggregate commits.
	s.mu.Lock()
	activationHook := s.onActivated
	s.onActivated = nil
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.onActivated = activationHook
		s.mu.Unlock()
	}()
	view, transaction, err := s.Store.RecordReceipt(requestID, buyerUserID, state, issueReason)
	if err != nil {
		return View{}, transaction, err
	}
	if err := s.persist(requestID); err != nil {
		return View{}, transaction, err
	}
	if activationHook != nil && view.Obligation != nil {
		activationHook(view.Request, *view.Obligation)
	}
	return view, transaction, nil
}

func (s *PostgresStore) GetForSupplier(requestID, organizationID string) (View, error) {
	if err := s.hydrateForTenant(requestID, "", organizationID); err != nil {
		return View{}, err
	}
	return s.Store.GetForSupplier(requestID, organizationID)
}
func (s *PostgresStore) GetForBuyer(requestID, buyerUserID string) (View, error) {
	if err := s.hydrateForTenant(requestID, buyerUserID, ""); err != nil {
		return View{}, err
	}
	return s.Store.GetForBuyer(requestID, buyerUserID)
}
func (s *PostgresStore) GetPublic(requestID string) (View, error) {
	if err := s.hydrate(requestID); err != nil {
		return View{}, err
	}
	return s.Store.GetPublic(requestID)
}
func (s *PostgresStore) GetByObligationForBuyer(obligationID, buyerUserID string) (View, error) {
	if err := s.hydrateByObligationForTenant(obligationID, buyerUserID, ""); err != nil {
		return View{}, err
	}
	return s.Store.GetByObligationForBuyer(obligationID, buyerUserID)
}
func (s *PostgresStore) ListForSupplier(organizationID string) []View {
	_ = s.hydrateList("supplier_organization_id", organizationID)
	return s.Store.ListForSupplier(organizationID)
}
func (s *PostgresStore) ListForBuyer(buyerUserID string) []View {
	_ = s.hydrateList("buyer_user_id", buyerUserID)
	return s.Store.ListForBuyer(buyerUserID)
}
func (s *PostgresStore) PaymentSnapshot(obligationID string) (payments.ObligationSnapshot, error) {
	if err := s.hydrateByObligation(obligationID); err != nil {
		return payments.ObligationSnapshot{}, err
	}
	return s.Store.PaymentSnapshot(obligationID)
}
func (s *PostgresStore) ApplyPayment(obligationID string, delta ledger.Money) error {
	if err := s.hydrateByObligation(obligationID); err != nil {
		return err
	}
	if err := s.Store.ApplyPayment(obligationID, delta); err != nil {
		return err
	}
	return s.persistByObligation(obligationID)
}
func (s *PostgresStore) ApplyAdjustment(obligationID string, reduction ledger.Money) error {
	if err := s.hydrateByObligation(obligationID); err != nil {
		return err
	}
	if err := s.Store.ApplyAdjustment(obligationID, reduction); err != nil {
		return err
	}
	return s.persistByObligation(obligationID)
}
func (s *PostgresStore) CollectionState(obligationID string) (CollectionState, error) {
	if err := s.hydrateByObligation(obligationID); err != nil {
		return CollectionState{}, err
	}
	return s.Store.CollectionState(obligationID)
}
func (s *PostgresStore) CollectionStateForOrganization(obligationID, organizationID string) (CollectionState, error) {
	if err := s.hydrateByObligationForTenant(obligationID, "", organizationID); err != nil {
		return CollectionState{}, err
	}
	return s.Store.CollectionStateForOrganization(obligationID, organizationID)
}
func (s *PostgresStore) ObligationBelongsToOrganization(obligationID, organizationID string) bool {
	_ = s.hydrateByObligationForTenant(obligationID, "", organizationID)
	return s.Store.ObligationBelongsToOrganization(obligationID, organizationID)
}

func (s *PostgresStore) mutate(requestID string, operation func() (View, error)) (View, error) {
	if s.pool == nil {
		return View{}, errors.New("credit database is not configured")
	}
	if err := s.hydrate(requestID); err != nil {
		return View{}, err
	}
	view, err := operation()
	if err != nil {
		return View{}, err
	}
	if err := s.persist(requestID); err != nil {
		s.discard(requestID)
		return View{}, err
	}
	return view, nil
}

// discard removes a failed local mutation so the next attempt is forced to
// reload the committed aggregate from PostgreSQL. Keeping a partially-mutated
// in-memory graph after a failed write would make retries non-deterministic.
func (s *PostgresStore) discard(requestID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := s.requests[requestID]
	if request == nil {
		return
	}
	for id, agreement := range s.agreements {
		if agreement.CreditRequestID == requestID {
			delete(s.agreements, id)
		}
	}
	for id, acceptance := range s.acceptances {
		if acceptance.CreditRequestID == requestID {
			delete(s.acceptances, id)
		}
	}
	for id, release := range s.releases {
		if release.CreditRequestID == requestID {
			delete(s.releases, id)
		}
	}
	for id, obligation := range s.obligations {
		if obligation.CreditRequestID == requestID {
			delete(s.obligations, id)
		}
	}
	delete(s.receipts, requestID)
	delete(s.requests, requestID)
}

func (s *PostgresStore) hydrate(requestID string) error {
	s.mu.RLock()
	_, exists := s.requests[requestID]
	s.mu.RUnlock()
	if exists {
		return nil
	}
	return errors.New("credit tenant context is required before loading an aggregate")
}

func (s *PostgresStore) hydrateForTenant(requestID, userID, organizationID string) error {
	s.mu.RLock()
	_, exists := s.requests[requestID]
	s.mu.RUnlock()
	if exists {
		return nil
	}
	if strings.TrimSpace(userID) == "" && strings.TrimSpace(organizationID) == "" {
		return errors.New("credit tenant context is required before loading an aggregate")
	}
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(context.Background(), `SELECT set_config('app.current_user_id', $1, true), set_config('app.current_organization_id', $2, true)`, userID, organizationID); err != nil {
		return err
	}
	var payload []byte
	if err := tx.QueryRow(context.Background(), `SELECT app.credit_snapshot_by_id($1)`, requestID).Scan(&payload); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("credit request not found")
		}
		return fmt.Errorf("load credit aggregate: %w", err)
	}
	if len(payload) == 0 {
		return errors.New("credit request not found")
	}
	var view View
	if err := json.Unmarshal(payload, &view); err != nil {
		return fmt.Errorf("decode credit aggregate: %w", err)
	}
	s.installView(view)
	return tx.Commit(context.Background())
}

func (s *PostgresStore) hydrateByObligation(obligationID string) error {
	s.mu.RLock()
	for _, obligation := range s.obligations {
		if obligation.ID == obligationID {
			s.mu.RUnlock()
			return nil
		}
	}
	s.mu.RUnlock()
	return errors.New("credit tenant context is required before loading an obligation")
}

func (s *PostgresStore) hydrateByObligationForTenant(obligationID, userID, organizationID string) error {
	s.mu.RLock()
	for _, obligation := range s.obligations {
		if obligation.ID == obligationID {
			s.mu.RUnlock()
			return nil
		}
	}
	s.mu.RUnlock()
	if strings.TrimSpace(userID) == "" && strings.TrimSpace(organizationID) == "" {
		return errors.New("credit tenant context is required before loading an obligation")
	}
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(context.Background(), `SELECT set_config('app.current_user_id', $1, true), set_config('app.current_organization_id', $2, true)`, userID, organizationID); err != nil {
		return err
	}
	var id string
	if err := tx.QueryRow(context.Background(), `SELECT credit_request_id FROM app.credit_snapshot_by_obligation($1)`, obligationID).Scan(&id); err != nil {
		return errors.New("obligation not found")
	}
	var payload []byte
	if err := tx.QueryRow(context.Background(), `SELECT app.credit_snapshot_by_id($1)`, id).Scan(&payload); err != nil {
		return errors.New("credit request not found")
	}
	var view View
	if err := json.Unmarshal(payload, &view); err != nil {
		return fmt.Errorf("decode credit aggregate: %w", err)
	}
	s.installView(view)
	return tx.Commit(context.Background())
}

func (s *PostgresStore) hydrateList(field, value string) error {
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	setting := "app.current_organization_id"
	if field == "buyer_user_id" {
		setting = "app.current_user_id"
	}
	if _, err := tx.Exec(context.Background(), `SELECT set_config($1, $2, true)`, setting, value); err != nil {
		return err
	}
	rows, err := tx.Query(context.Background(), fmt.Sprintf(`SELECT aggregate FROM app.credit_aggregate_snapshots WHERE %s = $1 ORDER BY updated_at DESC`, field), value)
	if err != nil {
		return err
	}
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			rows.Close()
			return err
		}
		var view View
		if err := json.Unmarshal(payload, &view); err != nil {
			rows.Close()
			return err
		}
		s.installView(view)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	return tx.Commit(context.Background())
}

func (s *PostgresStore) installView(view View) {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := view.Request
	s.requests[request.ID] = &request
	if view.Agreement.ID != "" {
		agreement := view.Agreement
		s.agreements[agreement.ID] = &agreement
	}
	if view.Acceptance != nil {
		acceptance := *view.Acceptance
		s.acceptances[acceptance.ID] = &acceptance
	}
	if view.Mandate != nil {
		mandate := *view.Mandate
		s.mandateMap[mandate.ID] = &mandate
	}
	if view.Release != nil {
		release := *view.Release
		s.releases[release.ID] = &release
	}
	if view.Receipts != nil {
		receipts := make([]*ReceiptConfirmation, 0, len(view.Receipts))
		for _, item := range view.Receipts {
			value := item
			receipts = append(receipts, &value)
		}
		s.receipts[request.ID] = receipts
	}
	if view.Obligation != nil {
		obligation := *view.Obligation
		s.obligations[obligation.ID] = &obligation
	}
}

func (s *PostgresStore) persist(requestID string) error {
	s.mu.RLock()
	request := s.requests[requestID]
	if request == nil {
		s.mu.RUnlock()
		return errors.New("credit request not found")
	}
	view := s.viewLocked(request)
	version := request.Version
	s.mu.RUnlock()
	payload, err := json.Marshal(view)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(context.Background(), `SELECT set_config('app.current_organization_id', $1, true)`, request.SupplierOrganizationID); err != nil {
		return err
	}
	if err := syncNormalizedCredit(context.Background(), tx, view); err != nil {
		return err
	}
	var commandTag pgconn.CommandTag
	if version <= 1 {
		commandTag, err = tx.Exec(context.Background(), `INSERT INTO app.credit_aggregate_snapshots (credit_request_id, supplier_organization_id, buyer_user_id, aggregate, version, updated_at) VALUES ($1, $2, $3, $4::jsonb, $5, $6) ON CONFLICT (credit_request_id) DO NOTHING`, request.ID, request.SupplierOrganizationID, request.BuyerUserID, payload, version, time.Now().UTC())
	} else {
		commandTag, err = tx.Exec(context.Background(), `UPDATE app.credit_aggregate_snapshots SET aggregate = $3::jsonb, version = $4, updated_at = $5 WHERE credit_request_id = $1 AND supplier_organization_id = $2 AND version = $6`, request.ID, request.SupplierOrganizationID, payload, version, time.Now().UTC(), version-1)
	}
	if err != nil {
		return fmt.Errorf("persist credit aggregate: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("credit aggregate changed concurrently; retry the operation")
	}
	return tx.Commit(context.Background())
}

func syncNormalizedCredit(ctx context.Context, tx pgx.Tx, view View) error {
	r := view.Request
	_, err := tx.Exec(ctx, `
		INSERT INTO app.credit_requests
		(id, supplier_organization_id, buyer_user_id, buyer_business_id, principal_kobo, currency, goods_description, invoice_reference, invoice_document_hash, due_date, grace_hours, collection_at, state, agreement_version_id, mandate_id, acceptance_id, release_id, receipt_id, obligation_id, created_by, created_at, updated_at, version)
		VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,NULLIF($8,''),NULLIF($9,''),$10::date,$11,$12,$13,NULLIF($14,'')::uuid,NULLIF($15,'')::uuid,NULLIF($16,'')::uuid,NULLIF($17,'')::uuid,NULLIF($18,'')::uuid,NULLIF($19,'')::uuid,$20::uuid,$21,$22,$23)
		ON CONFLICT (id) DO UPDATE SET state=EXCLUDED.state, agreement_version_id=EXCLUDED.agreement_version_id, mandate_id=EXCLUDED.mandate_id, acceptance_id=EXCLUDED.acceptance_id, release_id=EXCLUDED.release_id, receipt_id=EXCLUDED.receipt_id, obligation_id=EXCLUDED.obligation_id, updated_at=EXCLUDED.updated_at, version=EXCLUDED.version`,
		r.ID, r.SupplierOrganizationID, r.BuyerUserID, r.BuyerBusinessID, int64(r.PrincipalKobo), r.Currency, r.GoodsDescription, r.InvoiceReference, r.InvoiceDocumentHash, r.DueDate, r.GraceHours, r.CollectionAt, r.State, r.AgreementVersionID, r.MandateID, r.AcceptanceID, r.ReleaseID, r.ReceiptID, r.ObligationID, r.CreatedBy, r.CreatedAt, r.UpdatedAt, r.Version)
	if err != nil {
		return fmt.Errorf("persist normalized credit request: %w", err)
	}
	if a := view.Agreement; a.ID != "" {
		if _, err := tx.Exec(ctx, `INSERT INTO app.agreement_versions (id,credit_request_id,version,canonical_json,document_hash,terms_version,privacy_version,created_by,created_at) VALUES ($1::uuid,$2::uuid,$3,$4::jsonb,$5,$6,$7,$8::uuid,$9) ON CONFLICT (id) DO NOTHING`, a.ID, a.CreditRequestID, a.Version, a.CanonicalJSON, a.DocumentHash, a.TermsVersion, a.PrivacyVersion, a.CreatedBy, a.CreatedAt); err != nil {
			return fmt.Errorf("persist agreement version: %w", err)
		}
	}
	if a := view.Acceptance; a != nil {
		if _, err := tx.Exec(ctx, `INSERT INTO app.agreement_acceptances (id,credit_request_id,agreement_version_id,accepting_user_id,person_id,business_id,acceptance_method,authentication_level,agreement_hash,mandate_provider_id,accepted_at) VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7,$8,$9,$10,$11) ON CONFLICT (id) DO NOTHING`, a.ID, a.CreditRequestID, a.AgreementVersionID, a.AcceptingUserID, a.PersonID, a.BusinessID, a.AcceptanceMethod, a.AuthenticationLevel, a.AgreementHash, a.MandateProviderID, a.AcceptedAt); err != nil {
			return fmt.Errorf("persist agreement acceptance: %w", err)
		}
	}
	if release := view.Release; release != nil {
		if _, err := tx.Exec(ctx, `INSERT INTO app.goods_releases (id,credit_request_id,supplier_actor_id,delivery_method,notes,released_at) VALUES ($1::uuid,$2::uuid,$3::uuid,$4,NULLIF($5,''),$6) ON CONFLICT (id) DO NOTHING`, release.ID, release.CreditRequestID, release.SupplierActorID, release.DeliveryMethod, release.Notes, release.ReleasedAt); err != nil {
			return fmt.Errorf("persist goods release: %w", err)
		}
	}
	for _, receipt := range view.Receipts {
		if _, err := tx.Exec(ctx, `INSERT INTO app.receipt_confirmations (id,credit_request_id,buyer_user_id,state,issue_reason,received_at) VALUES ($1::uuid,$2::uuid,$3::uuid,$4,NULLIF($5,''),$6) ON CONFLICT (id) DO NOTHING`, receipt.ID, receipt.CreditRequestID, receipt.BuyerUserID, receipt.State, receipt.IssueReason, receipt.ReceivedAt); err != nil {
			return fmt.Errorf("persist receipt confirmation: %w", err)
		}
	}
	if obligation := view.Obligation; obligation != nil {
		if _, err := tx.Exec(ctx, `INSERT INTO app.obligations (id,credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at) VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,$8,$9,$10,$11,$12::uuid,$13) ON CONFLICT (id) DO UPDATE SET lifecycle_status=EXCLUDED.lifecycle_status,payment_status=EXCLUDED.payment_status,outstanding_kobo=EXCLUDED.outstanding_kobo`, obligation.ID, obligation.CreditRequestID, obligation.AgreementVersionID, obligation.SupplierOrganizationID, obligation.BuyerBusinessID, int64(obligation.PrincipalKobo), obligation.Currency, obligation.LifecycleStatus, obligation.PaymentStatus, int64(obligation.OutstandingKobo), int64(obligation.BaseFeeKobo), obligation.LedgerTransactionID, obligation.ActivatedAt); err != nil {
			return fmt.Errorf("persist obligation: %w", err)
		}
	}
	return nil
}

func (s *PostgresStore) persistByObligation(obligationID string) error {
	s.mu.RLock()
	var requestID string
	for id, request := range s.requests {
		if request.ObligationID == obligationID {
			requestID = id
			break
		}
	}
	s.mu.RUnlock()
	if requestID == "" {
		return errors.New("credit request not found")
	}
	return s.persist(requestID)
}

// InvalidateObligation removes the process-local projection after another
// transactional repository (payments or disputes) updates its authoritative
// database state. The next read rehydrates the committed aggregate.
func (s *PostgresStore) InvalidateObligation(obligationID string) {
	s.mu.RLock()
	requestID := ""
	for _, obligation := range s.obligations {
		if obligation.ID == obligationID {
			requestID = obligation.CreditRequestID
			break
		}
	}
	s.mu.RUnlock()
	if requestID != "" {
		s.discard(requestID)
	}
}
