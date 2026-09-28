package credit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"kredit/internal/access"
	"kredit/internal/businesspolicy"
	"kredit/internal/db"
	"kredit/internal/ledger"
	"kredit/internal/legalpublication"
	"kredit/internal/mandates"
	"kredit/internal/payments"
	"kredit/internal/schedules"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore preserves the existing credit lifecycle invariants while
// durably snapshotting the aggregate after every mutation. The normalized
// tables remain available for reporting and this boundary is deliberately
// fail-closed when PostgreSQL is unavailable.
//
// The embedded Store doubles as a process-local projection of the committed
// aggregate. That projection is a cache, never a source of truth: PostgreSQL is
// authoritative (README section 10.2) and the API and worker are separate
// processes, so a projection this process loaded earlier may already be behind
// what another process committed.
type PostgresStore struct {
	*Store
	pool *pgxpool.Pool

	// projectionMu guards the cache bookkeeping only. It is never held while
	// s.mu is taken, so the two locks cannot deadlock against each other.
	projectionMu sync.Mutex
	pinned       map[string]int
	loadedAt     map[string]time.Time

	// deemedAcceptanceNotice is how long a delivered goods-release notice must
	// have been sitting with the buyer before silence may activate the
	// obligation. Zero disables the notice requirement and is refused for any
	// deployment that can move real money (internal/config validates this).
	deemedAcceptanceNotice time.Duration
}

// projectionLimit bounds the process-local projection. Without it the maps grow
// with every credit request the process has ever served, which is an unbounded
// leak in a long-running API process.
const projectionLimit = 512

var _ Service = (*PostgresStore)(nil)

func NewPostgresStore(pool *pgxpool.Pool, development *Store) *PostgresStore {
	if development == nil {
		development = NewStore(nil, ledger.NewStore())
	}
	store := &PostgresStore{Store: development, pool: pool, pinned: map[string]int{}, loadedAt: map[string]time.Time{}, deemedAcceptanceNotice: deemedAcceptanceWindow}
	store.purchasePermission = store.checkPurchasePermission
	store.purchaseActions = store.readPurchaseActions
	store.SetDeemedAcceptanceGate(store.assertDeemedAcceptanceEvidence)
	return store
}

// SetDeemedAcceptanceNotice overrides how long a delivered goods-release notice
// must have been with the buyer before silence may activate the obligation.
func (s *PostgresStore) SetDeemedAcceptanceNotice(minimum time.Duration) {
	s.projectionMu.Lock()
	defer s.projectionMu.Unlock()
	s.deemedAcceptanceNotice = minimum
}

func (s *PostgresStore) deemedAcceptanceNoticeWindow() time.Duration {
	s.projectionMu.Lock()
	defer s.projectionMu.Unlock()
	return s.deemedAcceptanceNotice
}

// assertDeemedAcceptanceEvidence answers the only question that matters before
// silence is treated as consent: can we prove this buyer was told, and has the
// buyer ever shown that our notices reach them?
//
// Every failure path returns an error, including an unreachable database. The
// worst outcome of refusing is that a supplier waits for an explicit
// confirmation. The worst outcome of allowing without evidence is a debit
// against a buyer who never heard from us.
func (s *PostgresStore) assertDeemedAcceptanceEvidence(ctx context.Context, requestID string) error {
	if s == nil || s.pool == nil {
		return errors.New("deemed acceptance requires the authoritative database")
	}
	seconds := int64(s.deemedAcceptanceNoticeWindow() / time.Second)
	var released, noticeDelivered, buyerRespondedBefore bool
	err := s.pool.QueryRow(ctx, `
WITH subject AS (
    SELECT id, buyer_business_id FROM app.credit_requests WHERE id = $1::uuid
), latest_release AS (
    SELECT g.id
    FROM app.goods_releases g
    JOIN subject ON subject.id = g.credit_request_id
    ORDER BY g.released_at DESC
    LIMIT 1
)
SELECT
    EXISTS(SELECT 1 FROM latest_release),
    EXISTS(
        SELECT 1
        FROM latest_release
        JOIN app.outbox_events e
          ON e.idempotency_key = 'goods-release-notification:' || latest_release.id::text
        JOIN app.notifications n ON n.event_reference = 'outbox:' || e.id::text
        JOIN app.notification_delivery_receipts receipt ON receipt.notification_id = n.id
        WHERE n.state IN ('delivered','read')
          AND receipt.received_at <= now() - make_interval(secs => $2::double precision)
    ),
    EXISTS(
        SELECT 1
        FROM app.receipt_confirmations rc
        JOIN app.credit_requests c ON c.id = rc.credit_request_id
        JOIN subject ON subject.buyer_business_id = c.buyer_business_id
        WHERE rc.credit_request_id <> subject.id
          AND rc.issue_reason IS DISTINCT FROM $3
    )`, requestID, seconds, deemedAcceptanceReason).Scan(&released, &noticeDelivered, &buyerRespondedBefore)
	if err != nil {
		return fmt.Errorf("deemed acceptance evidence unavailable: %w", err)
	}
	if !released {
		return errors.New("deemed acceptance requires a recorded goods release")
	}
	if !buyerRespondedBefore {
		return errors.New("deemed acceptance is not available for a buyer's first trade credit")
	}
	if seconds > 0 && !noticeDelivered {
		return errors.New("deemed acceptance requires a confirmed goods-release notice delivered for the full waiting period")
	}
	return nil
}

// pin marks an aggregate as being mutated. A pinned aggregate is never
// refreshed from PostgreSQL or evicted, so a concurrent read cannot replace the
// in-memory graph underneath an in-flight command.
func (s *PostgresStore) pin(requestID string) {
	s.projectionMu.Lock()
	defer s.projectionMu.Unlock()
	if s.pinned == nil {
		s.pinned = map[string]int{}
	}
	s.pinned[requestID]++
}

func (s *PostgresStore) unpin(requestID string) {
	s.projectionMu.Lock()
	defer s.projectionMu.Unlock()
	if s.pinned[requestID] <= 1 {
		delete(s.pinned, requestID)
		return
	}
	s.pinned[requestID]--
}

func (s *PostgresStore) isPinned(requestID string) bool {
	s.projectionMu.Lock()
	defer s.projectionMu.Unlock()
	return s.pinned[requestID] > 0
}

// markLoaded records that an aggregate is resident and evicts the coldest
// unpinned entries once the projection is over its bound.
func (s *PostgresStore) markLoaded(requestIDs ...string) {
	if len(requestIDs) == 0 {
		return
	}
	s.projectionMu.Lock()
	if s.loadedAt == nil {
		s.loadedAt = map[string]time.Time{}
	}
	now := time.Now()
	keep := make(map[string]bool, len(requestIDs))
	for _, id := range requestIDs {
		s.loadedAt[id] = now
		keep[id] = true
	}
	var victims []string
	if overflow := len(s.loadedAt) - projectionLimit; overflow > 0 {
		type candidate struct {
			id string
			at time.Time
		}
		ordered := make([]candidate, 0, len(s.loadedAt))
		for id, at := range s.loadedAt {
			if keep[id] || s.pinned[id] > 0 {
				continue
			}
			ordered = append(ordered, candidate{id: id, at: at})
		}
		sort.Slice(ordered, func(i, j int) bool {
			if !ordered[i].at.Equal(ordered[j].at) {
				return ordered[i].at.Before(ordered[j].at)
			}
			return ordered[i].id < ordered[j].id
		})
		if overflow > len(ordered) {
			overflow = len(ordered)
		}
		for _, item := range ordered[:overflow] {
			victims = append(victims, item.id)
			delete(s.loadedAt, item.id)
		}
	}
	s.projectionMu.Unlock()
	for _, id := range victims {
		s.discard(id)
	}
}

// releaseListing unpins a batch loaded for one listing and makes it evictable.
// Entries stay pinned while the listing is being read so eviction cannot remove
// rows the caller is about to return.
func (s *PostgresStore) releaseListing(requestIDs []string) {
	for _, id := range requestIDs {
		s.unpin(id)
	}
	s.markLoaded(requestIDs...)
}

func (s *PostgresStore) forgetProjection(requestID string) {
	s.projectionMu.Lock()
	delete(s.loadedAt, requestID)
	s.projectionMu.Unlock()
}

func (s *PostgresStore) Create(input CreateInput) (CreditRequest, error) {
	if s.pool == nil {
		return CreditRequest{}, errors.New("credit database is not configured")
	}
	policy, err := businesspolicy.ReadTx(context.Background(), s.pool)
	if err != nil {
		return CreditRequest{}, err
	}
	input.FeeTerms = &ledger.FeeTerms{PolicyRevision: policy.Revision, BaseBPS: policy.Values.BaseFeeBPS, CollectionBPS: policy.Values.CollectionFeeBPS}
	local := s.commandStore()
	request, err := local.Create(input)
	if err != nil {
		return CreditRequest{}, err
	}
	if policy.Values.EnhancedReview > 0 && int64(request.PrincipalKobo) >= policy.Values.EnhancedReview {
		local.requests[request.ID].RequiresEnhancedReview = true
		request.RequiresEnhancedReview = true
	}
	view, err := local.GetPublic(request.ID)
	if err != nil {
		return CreditRequest{}, err
	}
	if err := s.persistViewAs(view, input.CreatedBy, access.PermissionCreateCredit); err != nil {
		return CreditRequest{}, err
	}
	s.installView(view)
	s.markLoaded(request.ID)
	return request, nil
}

func (s *PostgresStore) ActivateTradeLineDrawdown(input TradeLineActivationInput) (View, *ledger.Transaction, error) {
	if s == nil || s.pool == nil {
		return View{}, nil, errors.New("credit database is not configured")
	}
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return View{}, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	view, journal, finalize, err := s.ActivateTradeLineDrawdownTx(ctx, tx, input)
	if err != nil {
		return View{}, nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return View{}, nil, err
	}
	finalize()
	s.mu.RLock()
	hook := s.onActivated
	s.mu.RUnlock()
	if hook != nil && view.Obligation != nil {
		hook(view.Request, *view.Obligation)
	}
	return view, journal, nil
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
func (l *activationTxLedger) PostActivationWithFee(id string, principal, fee ledger.Money, at time.Time, key string) (ledger.Transaction, error) {
	return l.store.PostActivationWithFeeTx(l.ctx, l.tx, id, principal, fee, at, key)
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
	view, err := s.mutateAs(requestID, actorID, access.PermissionCreateCredit, func(local *Store) (View, error) {
		request, err := local.UpdateDraft(requestID, actorID, input)
		if err != nil {
			return View{}, err
		}
		return local.GetForSupplier(request.ID, request.SupplierOrganizationID)
	})
	return view.Request, err
}

func (s *PostgresStore) Send(requestID, actorID string) (View, error) {
	versions, err := legalpublication.Resolve(s.legalReader)
	if err != nil {
		return View{}, err
	}
	return s.mutateAs(requestID, actorID, access.PermissionCreateCredit, func(local *Store) (View, error) { return local.sendWithLegalVersions(requestID, actorID, versions) })
}
func (s *PostgresStore) Cancel(requestID, actorID string) (View, error) {
	return s.mutateAs(requestID, actorID, access.PermissionCreateCredit, func(local *Store) (View, error) { return local.Cancel(requestID, actorID) })
}
func (s *PostgresStore) Review(requestID, buyerUserID string) (View, error) {
	if _, err := s.GetForBuyer(requestID, buyerUserID); err != nil {
		return View{}, err
	}
	return s.mutateAs(requestID, buyerUserID, "purchase:review", func(local *Store) (View, error) { return local.Review(requestID, buyerUserID) })
}
func (s *PostgresStore) Decline(requestID, buyerUserID string) (View, error) {
	if _, err := s.GetForBuyer(requestID, buyerUserID); err != nil {
		return View{}, err
	}
	return s.mutateAs(requestID, buyerUserID, "purchase:review", func(local *Store) (View, error) { return local.Decline(requestID, buyerUserID) })
}
func (s *PostgresStore) AuthorizeMandate(ctx context.Context, requestID, buyerUserID string, options ...mandates.AuthorizationOptions) (View, error) {
	if _, err := s.GetForBuyer(requestID, buyerUserID); err != nil {
		return View{}, err
	}
	return s.mutateAs(requestID, buyerUserID, "", func(local *Store) (View, error) {
		return local.AuthorizeMandate(ctx, requestID, buyerUserID, options...)
	})
}
func (s *PostgresStore) SetMandate(requestID, buyerUserID string, mandate mandates.Mandate) (View, error) {
	if _, err := s.GetForBuyer(requestID, buyerUserID); err != nil {
		return View{}, err
	}
	if err := s.hydrateForTenant(requestID, buyerUserID, ""); err != nil {
		return View{}, err
	}
	return s.mutateAs(requestID, buyerUserID, "", func(local *Store) (View, error) { return local.SetMandate(requestID, buyerUserID, mandate) })
}
func (s *PostgresStore) Accept(requestID, buyerUserID, agreementID, agreementHash, mandateID, authLevel string, identityVerified, authorityVerified bool) (View, error) {
	if _, err := s.GetForBuyer(requestID, buyerUserID); err != nil {
		return View{}, err
	}
	personID, err := s.acceptingPerson(buyerUserID)
	if err != nil {
		return View{}, err
	}
	return s.mutateAs(requestID, buyerUserID, "purchase:accept", func(local *Store) (View, error) {
		v, err := local.Accept(requestID, buyerUserID, agreementID, agreementHash, mandateID, authLevel, identityVerified, authorityVerified)
		if err != nil {
			return View{}, err
		}
		local.mu.Lock()
		if a := local.acceptances[v.Request.AcceptanceID]; a != nil {
			a.PersonID = personID
		}
		v = local.viewLocked(local.requests[requestID])
		local.mu.Unlock()
		return v, nil
	})
}
func (s *PostgresStore) Release(requestID, supplierOrgID, actorID, deliveryMethod, notes string) (View, error) {
	return s.mutateAs(requestID, actorID, access.PermissionReleaseGoods, func(local *Store) (View, error) {
		return local.Release(requestID, supplierOrgID, actorID, deliveryMethod, notes)
	})
}
func (s *PostgresStore) RecordReceipt(requestID, buyerUserID, state, issueReason string) (View, *ledger.Transaction, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.recordReceiptContext(ctx, requestID, buyerUserID, state, issueReason)
}

func (s *PostgresStore) recordReceiptContext(ctx context.Context, requestID, buyerUserID, state, issueReason string) (View, *ledger.Transaction, error) {
	return s.recordReceiptTransaction(ctx, requestID, buyerUserID, state, issueReason, false)
}

func (s *PostgresStore) recordReceiptTransaction(ctx context.Context, requestID, buyerUserID, state, issueReason string, system bool) (View, *ledger.Transaction, error) {
	if s == nil || s.pool == nil {
		return View{}, nil, errors.New("credit database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return View{}, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, buyerUserID); err != nil {
		return View{}, nil, err
	}
	var payload []byte
	if err = tx.QueryRow(ctx, `SELECT aggregate FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1 AND (buyer_user_id=$2 OR app.purchase_request_read(credit_request_id::uuid))`, requestID, buyerUserID).Scan(&payload); err != nil {
		return View{}, nil, err
	}
	var old View
	if err = json.Unmarshal(payload, &old); err != nil {
		return View{}, nil, err
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_organization_id',$1,true)`, old.Request.SupplierOrganizationID); err != nil {
		return View{}, nil, err
	}
	// Resolve the authorized buying scope first. Lock the committed aggregate
	// only after establishing that scope; FOR UPDATE itself requires a write
	// policy granted only after current owner authority is established.
	if !system {
		if _, err = tx.Exec(ctx, `SELECT app.lock_purchase_permission($1::uuid,'receive',$2)`, old.Request.BuyerBusinessID, int64(old.Request.PrincipalKobo)); err != nil {
			return View{}, nil, err
		}
	}
	if err = tx.QueryRow(ctx, `SELECT aggregate FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1 AND supplier_organization_id=$2 AND aggregate->'request'->>'buyer_business_id'=$3 FOR UPDATE`, requestID, old.Request.SupplierOrganizationID, old.Request.BuyerBusinessID).Scan(&payload); err != nil {
		return View{}, nil, err
	}
	if err = json.Unmarshal(payload, &old); err != nil {
		return View{}, nil, err
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.deemed_acceptance_min_seconds',$1,true)`, fmt.Sprint(int64(s.deemedAcceptanceNoticeWindow()/time.Second))); err != nil {
		return View{}, nil, err
	}
	s.mu.RLock()
	pgLedger, ok := s.ledger.(*ledger.PostgresStore)
	now, newID, mandateProvider, hook := s.now, s.newID, s.mandates, s.onActivated
	s.mu.RUnlock()
	if !ok {
		return View{}, nil, errors.New("transactional activation ledger unavailable")
	}
	local := NewStore(mandateProvider, &activationTxLedger{ctx: ctx, tx: tx, store: pgLedger})
	if !system {
		local.purchasePermission = func(r *CreditRequest, actor, action string) bool {
			return r.ID == requestID && actor == buyerUserID && action == "receive"
		}
	}
	local.now = now
	local.newID = newID
	(&PostgresStore{Store: local}).installView(old)
	systemID := ""
	if system {
		var hours int64
		if err = tx.QueryRow(ctx, `SELECT hours FROM app.system_acceptance_settings()`).Scan(&hours); err != nil {
			return View{}, nil, err
		}
		seconds := max(int64(s.deemedAcceptanceNoticeWindow()/time.Second), int64(259200), hours*3600)
		err = tx.QueryRow(ctx, `INSERT INTO app.system_acceptances(credit_request_id, supplier_organization_id,buyer_user_id,release_id,notification_id,receipt_channel,receipt_event_id,minimum_seconds) SELECT $1,organization_id,buyer_id,release_id,notification_id,receipt_channel,receipt_event_id,$2 FROM app.deemed_acceptance_evidence($1,$2) RETURNING id::text`, requestID, seconds).Scan(&systemID)
		if errors.Is(err, pgx.ErrNoRows) {
			return View{}, nil, errDeemedNotEligible
		}
		if err != nil {
			return View{}, nil, err
		}
	}
	view, journal, err := local.recordReceiptLocked(requestID, buyerUserID, state, issueReason, systemID)
	if err != nil {
		return View{}, nil, err
	}
	if err = syncNormalizedCreditWithEvidence(ctx, tx, view, system); err != nil {
		return View{}, nil, err
	}
	if view.Obligation != nil {
		input, err := repaymentScheduleInput(view.Request, view.Obligation.ID)
		if err != nil {
			return View{}, nil, err
		}
		if _, _, err = schedules.NewPostgresStore(s.pool).CreateTx(ctx, tx, input); err != nil {
			return View{}, nil, err
		}
	}
	payload, err = json.Marshal(view)
	if err != nil {
		return View{}, nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE app.credit_aggregate_snapshots SET aggregate=$2::jsonb,version=$3,updated_at=now() WHERE credit_request_id=$1`, requestID, payload, view.Request.Version); err != nil {
		return View{}, nil, err
	}
	if system {
		if _, err = tx.Exec(ctx, `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key) VALUES('credit_request',$1,'notification.requested',jsonb_build_object('event','SYSTEM_ACCEPTANCE','amount_kobo',$2::bigint),$3)`, requestID, int64(view.Request.PrincipalKobo), "system-acceptance:"+systemID); err != nil {
			return View{}, nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return View{}, nil, err
	}
	s.installView(view)
	if hook != nil && view.Obligation != nil {
		hook(view.Request, *view.Obligation)
	}
	if !system && buyerUserID != view.Request.BuyerUserID && view.Mandate != nil {
		view.Mandate.AuthorizationURL = ""
	}
	return view, journal, nil
}

// readerForTenant returns an isolated, committed view for this authorized read.
// The shared copy is a routing hint; it never grants read or write authority.
func (s *PostgresStore) readerForTenant(requestID, userID, organizationID string) (*Store, error) {
	view, err := s.loadViewForTenant(requestID, userID, organizationID)
	if err != nil {
		return nil, err
	}
	s.installView(view)
	s.markLoaded(requestID)
	local := s.commandStore()
	(&PostgresStore{Store: local}).installView(view)
	return local, nil
}

// GetForSupplierContext preserves the actual requester when loading supplier data.
func (s *PostgresStore) GetForSupplierContext(ctx context.Context, requestID, organizationID string) (View, error) {
	identity, _ := db.TenantFromContext(ctx)
	if identity.OrganizationID != "" && identity.OrganizationID != organizationID {
		return View{}, errors.New("credit scope does not match authorized business")
	}
	local, err := s.readerForTenant(requestID, identity.UserID, organizationID)
	if err != nil {
		return View{}, err
	}
	return local.GetForSupplier(requestID, organizationID)
}
func (s *PostgresStore) GetForSupplier(requestID, organizationID string) (View, error) {
	local, err := s.readerForTenant(requestID, "", organizationID)
	if err != nil {
		return View{}, err
	}
	return local.GetForSupplier(requestID, organizationID)
}
func (s *PostgresStore) GetForBuyer(requestID, buyerUserID string) (View, error) {
	local, err := s.readerForTenant(requestID, buyerUserID, "")
	if err != nil {
		return View{}, err
	}
	return local.GetForBuyer(requestID, buyerUserID)
}
func (s *PostgresStore) GetPublic(requestID string) (View, error) {
	if s == nil || s.pool == nil {
		return View{}, errors.New("credit database is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var payload []byte
	if err := s.pool.QueryRow(ctx, `SELECT app.public_payment_intent($1::uuid)`, requestID).Scan(&payload); err != nil {
		return View{}, err
	}
	if len(payload) == 0 {
		return View{}, ErrRequestNotFound
	}
	var view View
	if err := json.Unmarshal(payload, &view); err != nil {
		return View{}, err
	}
	return view, nil
}
func (s *PostgresStore) GetByObligationForBuyer(obligationID, buyerUserID string) (View, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	scoped := &db.ScopedDatabase{Pool: s.pool}
	var requestID string
	if err := scoped.QueryRow(db.WithTenantContext(ctx, buyerUserID, ""), `SELECT credit_request_id FROM app.credit_snapshot_by_obligation($1)`, obligationID).Scan(&requestID); err != nil {
		return View{}, err
	}
	return s.GetForBuyer(requestID, buyerUserID)
}
func (s *PostgresStore) ListForSupplier(organizationID string) []View {
	views, err := s.ReadForSupplier(context.Background(), organizationID)
	if err != nil {
		panic(fmt.Errorf("credit: failed to read supplier credit requests for %s: %w", organizationID, err))
	}
	return views
}
func (s *PostgresStore) ListForBuyer(buyerUserID string) []View {
	views, err := s.ReadForBuyer(context.Background(), buyerUserID)
	if err != nil {
		panic(fmt.Errorf("credit: failed to read buyer credit requests for %s: %w", buyerUserID, err))
	}
	return views
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
	return s.CollectionStateContext(context.Background(), obligationID)
}
func (s *PostgresStore) CollectionStateContext(ctx context.Context, obligationID string) (CollectionState, error) {
	if s == nil || s.pool == nil {
		return CollectionState{}, errors.New("credit database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CollectionState{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := db.SetObligationContext(ctx, tx, obligationID); err != nil {
		return CollectionState{}, err
	}
	// Workers must start from current persisted evidence, even after restart or
	// a payment in another process. Never depend on a prior portal cache read.
	var payload []byte
	var outstanding ledger.Money
	var lifecycle string
	if err := tx.QueryRow(ctx, `SELECT s.aggregate,o.outstanding_kobo,o.lifecycle_status FROM app.credit_aggregate_snapshots s JOIN app.obligations o ON o.credit_request_id::text=s.credit_request_id WHERE o.id=$1::uuid`, obligationID).Scan(&payload, &outstanding, &lifecycle); err != nil {
		return CollectionState{}, err
	}
	var view View
	if err := json.Unmarshal(payload, &view); err != nil || view.Obligation == nil {
		return CollectionState{}, errors.New("persisted collection evidence is incomplete")
	}
	view.Obligation.OutstandingKobo, view.Obligation.LifecycleStatus = outstanding, lifecycle
	reader := &PostgresStore{Store: NewStore(nil, nil)}
	reader.installView(view)
	state, err := reader.Store.CollectionState(obligationID)
	if err != nil || state.MandateReference == "" {
		return state, err
	}
	err = tx.QueryRow(ctx, `SELECT app.collection_mandate_capacity($1::uuid)`, obligationID).Scan(&state.MandateRemainingKobo)
	return state, err
}
func (s *PostgresStore) CollectionStateForOrganization(obligationID, organizationID string) (CollectionState, error) {
	state, err := s.CollectionStateContext(db.WithTenantContext(context.Background(), "", organizationID), obligationID)
	if err != nil {
		return CollectionState{}, err
	}
	if state.SupplierOrganizationID != organizationID {
		return CollectionState{}, errors.New("obligation does not belong to organization")
	}
	return state, nil
}
func (s *PostgresStore) ObligationBelongsToOrganization(obligationID, organizationID string) bool {
	if err := s.hydrateByObligationForTenant(obligationID, "", organizationID); err != nil {
		return false
	}
	return s.Store.ObligationBelongsToOrganization(obligationID, organizationID)
}

// commandStore copies dependencies and policy settings, but owns its aggregate
// maps. No uncommitted command state is visible through the shared projection.
func (s *PostgresStore) commandStore() *Store {
	s.mu.RLock()
	defer s.mu.RUnlock()
	local := NewStore(s.mandates, s.ledger)
	local.now, local.newID = s.now, s.newID
	local.creationGuard = s.creationGuard
	local.enhancedReviewThreshold = s.enhancedReviewThreshold
	local.maxActiveExposureKobo = s.maxActiveExposureKobo
	local.purchasePermission, local.purchaseActions = s.purchasePermission, s.purchaseActions
	local.legalReader = s.legalReader
	return local
}

func (s *PostgresStore) mutateAs(requestID, actorID string, permission access.Permission, operation func(*Store) (View, error)) (View, error) {
	if s.pool == nil {
		return View{}, errors.New("credit database is not configured")
	}
	// The earlier authorized read supplies only an immutable tenant-routing hint.
	// Load current committed terms again and recheck the actor in PostgreSQL.
	s.mu.RLock()
	cached := s.requests[requestID]
	organizationID := ""
	if cached != nil {
		organizationID = cached.SupplierOrganizationID
	}
	s.mu.RUnlock()
	if organizationID == "" {
		return View{}, ErrRequestNotFound
	}
	old, err := s.loadViewForTenant(requestID, actorID, organizationID)
	if err != nil {
		return View{}, err
	}
	local := s.commandStore()
	(&PostgresStore{Store: local}).installView(old)
	view, err := operation(local)
	if err != nil {
		return View{}, err
	}
	if view.Request.Version != old.Request.Version {
		if err := s.persistViewAs(view, actorID, permission); err != nil {
			return View{}, err
		}
		s.installView(view)
		s.markLoaded(requestID)
	}
	if strings.HasPrefix(string(permission), "purchase:") && view.Mandate != nil && actorID != view.Request.BuyerUserID {
		view.Mandate.AuthorizationURL = ""
	}
	return view, nil
}

// discard removes a failed local mutation so the next attempt is forced to
// reload the committed aggregate from PostgreSQL. Keeping a partially-mutated
// in-memory graph after a failed write would make retries non-deterministic.
func (s *PostgresStore) discard(requestID string) {
	// Dropped before the aggregate lock is taken, so the two mutexes are never
	// held at the same time.
	s.forgetProjection(requestID)
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
	// Mandates can be shared by several cached requests. Retain only those
	// still referenced after eviction, including both ID and provider-ID keys.
	referenced := make(map[string]bool, len(s.requests))
	for _, remaining := range s.requests {
		if remaining.MandateID != "" {
			referenced[remaining.MandateID] = true
		}
	}
	for key, mandate := range s.mandateMap {
		if mandate == nil || (!referenced[mandate.ID] && !referenced[mandate.ProviderID]) {
			delete(s.mandateMap, key)
		}
	}
}
