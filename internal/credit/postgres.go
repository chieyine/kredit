package credit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"kredit/internal/businesspolicy"
	"kredit/internal/ledger"
	"kredit/internal/mandates"
	"kredit/internal/payments"
	"kredit/internal/schedules"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	request, err := s.Store.Create(input)
	if err != nil {
		return CreditRequest{}, err
	}
	// The draft only exists in the projection until it is persisted, so it must
	// not be evictable in between.
	s.pin(request.ID)
	defer func() {
		s.unpin(request.ID)
		s.markLoaded(request.ID)
	}()
	if policy.Values.EnhancedReview > 0 && int64(request.PrincipalKobo) >= policy.Values.EnhancedReview {
		s.mu.Lock()
		if stored := s.requests[request.ID]; stored != nil {
			stored.RequiresEnhancedReview = true
		}
		s.mu.Unlock()
		request.RequiresEnhancedReview = true
	}
	if err := s.persist(request.ID); err != nil {
		s.discard(request.ID)
		return CreditRequest{}, err
	}
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
func (s *PostgresStore) AuthorizeMandate(ctx context.Context, requestID, buyerUserID string, options ...mandates.AuthorizationOptions) (View, error) {
	return s.mutate(requestID, func() (View, error) { return s.Store.AuthorizeMandate(ctx, requestID, buyerUserID, options...) })
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
	if s == nil || s.pool == nil {
		return View{}, nil, errors.New("credit database is not configured")
	}
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return View{}, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, buyerUserID); err != nil {
		return View{}, nil, err
	}
	var payload []byte
	if err = tx.QueryRow(ctx, `SELECT aggregate FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1 AND buyer_user_id=$2 FOR UPDATE`, requestID, buyerUserID).Scan(&payload); err != nil {
		return View{}, nil, err
	}
	var old View
	if err = json.Unmarshal(payload, &old); err != nil {
		return View{}, nil, err
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_organization_id',$1,true)`, old.Request.SupplierOrganizationID); err != nil {
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
	local.now = now
	local.newID = newID
	(&PostgresStore{Store: local}).installView(old)
	view, journal, err := local.RecordReceipt(requestID, buyerUserID, state, issueReason)
	if err != nil {
		return View{}, nil, err
	}
	if err = syncNormalizedCredit(ctx, tx, view); err != nil {
		return View{}, nil, err
	}
	if view.Obligation != nil {
		location, err := time.LoadLocation("Africa/Lagos")
		if err != nil {
			return View{}, nil, err
		}
		r := view.Request
		start, err := time.ParseInLocation("2006-01-02", r.DueDate, location)
		if err != nil {
			return View{}, nil, err
		}
		input := schedules.CreateInput{FirstCollectionAt: r.CollectionAt, ObligationID: view.Obligation.ID, PrincipalKobo: view.Obligation.PrincipalKobo, ScheduleType: r.ScheduleType, Count: r.ScheduleCount, StartDate: start, DueHour: r.CollectionAt.In(location).Hour(), DueMinute: r.CollectionAt.In(location).Minute(), Timezone: "Africa/Lagos", GraceHours: r.GraceHours, Cadence: r.ScheduleCadence, MonthEndPolicy: r.MonthEndPolicy, AllocationPolicy: "due_date_order"}
		if r.ScheduleType == "" || r.ScheduleType == "one_time" {
			input.ScheduleType = schedules.TypeEqual
			input.Count = 1
			input.Cadence = schedules.CadenceCustom
		}
		for _, term := range r.CustomScheduleItems {
			due, err := time.ParseInLocation("2006-01-02", term.DueDate, location)
			if err != nil {
				return View{}, nil, err
			}
			input.CustomItems = append(input.CustomItems, schedules.CustomItem{AmountKobo: term.AmountKobo, DueDate: due})
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
	if err = tx.Commit(ctx); err != nil {
		return View{}, nil, err
	}
	s.installView(view)
	if hook != nil && view.Obligation != nil {
		hook(view.Request, *view.Obligation)
	}
	return view, journal, nil
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
	loaded, _ := s.hydrateList("supplier_organization_id", organizationID)
	defer s.releaseListing(loaded)
	return s.Store.ListForSupplier(organizationID)
}
func (s *PostgresStore) ListForBuyer(buyerUserID string) []View {
	loaded, _ := s.hydrateList("buyer_user_id", buyerUserID)
	defer s.releaseListing(loaded)
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
	if s == nil || s.pool == nil {
		return CollectionState{}, errors.New("credit database is not configured")
	}
	// Workers must start from current persisted evidence, even after restart or
	// a payment in another process. Never depend on a prior portal cache read.
	var payload []byte
	var outstanding ledger.Money
	var lifecycle string
	if err := s.pool.QueryRow(context.Background(), `SELECT s.aggregate,o.outstanding_kobo,o.lifecycle_status FROM app.credit_aggregate_snapshots s JOIN app.obligations o ON o.credit_request_id::text=s.credit_request_id WHERE o.id=$1::uuid`, obligationID).Scan(&payload, &outstanding, &lifecycle); err != nil {
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
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CollectionState{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, state.BuyerUserID); err != nil {
		return CollectionState{}, err
	}
	err = tx.QueryRow(ctx, `SELECT GREATEST(0,m.amount_ceiling_kobo-COALESCE((SELECT SUM(a.succeeded_amount_kobo) FROM app.collection_attempts a JOIN app.collection_reservations r ON r.id=a.reservation_id WHERE r.mandate_id=m.id),0)-COALESCE((SELECT SUM(r.reserved_amount_kobo) FROM app.collection_reservations r WHERE r.mandate_id=m.id AND r.obligation_id<>o.id AND r.state IN ('PROCESSING','COMPLETED')),0)) FROM app.credit_requests c JOIN app.obligations o ON o.credit_request_id=c.id JOIN app.payment_mandates m ON m.id=c.mandate_id WHERE o.id=$1::uuid`, obligationID).Scan(&state.MandateRemainingKobo)
	return state, err
}
func (s *PostgresStore) CollectionStateForOrganization(obligationID, organizationID string) (CollectionState, error) {
	state, err := s.CollectionState(obligationID)
	if err != nil {
		return CollectionState{}, err
	}
	if state.SupplierOrganizationID != organizationID {
		return CollectionState{}, errors.New("obligation does not belong to organization")
	}
	return state, nil
}
func (s *PostgresStore) ObligationBelongsToOrganization(obligationID, organizationID string) bool {
	_ = s.hydrateByObligationForTenant(obligationID, "", organizationID)
	return s.Store.ObligationBelongsToOrganization(obligationID, organizationID)
}

func (s *PostgresStore) mutate(requestID string, operation func() (View, error)) (View, error) {
	if s.pool == nil {
		return View{}, errors.New("credit database is not configured")
	}
	// Held across the whole command so a concurrent read cannot refresh the
	// projection out from under the mutation, and so eviction cannot remove the
	// aggregate between the operation and its write.
	s.pin(requestID)
	defer s.unpin(requestID)
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

// hydrateForTenant reloads the committed aggregate on every read. Returning a
// projection this process loaded earlier would serve a supplier or buyer a
// balance the worker has already changed, and PostgreSQL is authoritative. An
// aggregate with an in-flight mutation is left alone: replacing it mid-command
// would discard the mutation.
func (s *PostgresStore) hydrateForTenant(requestID, userID, organizationID string) error {
	if s.isPinned(requestID) {
		return nil
	}
	s.mu.RLock()
	_, exists := s.requests[requestID]
	s.mu.RUnlock()
	if strings.TrimSpace(userID) == "" && strings.TrimSpace(organizationID) == "" {
		if exists {
			return nil
		}
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
	s.markLoaded(requestID)
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
	s.markLoaded(view.Request.ID)
	return tx.Commit(context.Background())
}

// hydrateList loads a tenant's aggregates and returns their identifiers pinned.
// The caller must pass them to releaseListing once it has read the projection;
// leaving them unpinned would let eviction remove rows mid-listing and return a
// short list.
func (s *PostgresStore) hydrateList(field, value string) ([]string, error) {
	loaded := []string{}
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return loaded, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	setting := "app.current_organization_id"
	if field == "buyer_user_id" {
		setting = "app.current_user_id"
	}
	if _, err := tx.Exec(context.Background(), `SELECT set_config($1, $2, true)`, setting, value); err != nil {
		return loaded, err
	}
	rows, err := tx.Query(context.Background(), fmt.Sprintf(`SELECT aggregate FROM app.credit_aggregate_snapshots WHERE %s = $1 ORDER BY updated_at DESC`, field), value)
	if err != nil {
		return loaded, err
	}
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			rows.Close()
			return loaded, err
		}
		var view View
		if err := json.Unmarshal(payload, &view); err != nil {
			rows.Close()
			return loaded, err
		}
		s.pin(view.Request.ID)
		s.installView(view)
		loaded = append(loaded, view.Request.ID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return loaded, err
	}
	rows.Close()
	return loaded, tx.Commit(context.Background())
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
	// Migration 070 refuses a deemed-acceptance receipt without delivered notice
	// evidence. The window travels with the transaction so the database applies
	// the same policy the application believes it is applying.
	if _, err := tx.Exec(context.Background(), `SELECT set_config('app.deemed_acceptance_min_seconds', $1, true)`, fmt.Sprint(int64(s.deemedAcceptanceNoticeWindow()/time.Second))); err != nil {
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
		(fee_terms, id, supplier_organization_id, buyer_user_id, buyer_business_id, principal_kobo, currency, goods_description, invoice_reference, invoice_document_hash, due_date, grace_hours, collection_at, state, agreement_version_id, mandate_id, acceptance_id, release_id, receipt_id, obligation_id, created_by, created_at, updated_at, version)
		VALUES ($24::jsonb,$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,NULLIF($8,''),NULLIF($9,''),$10::date,$11,$12,$13,NULLIF($14,'')::uuid,NULLIF($15,'')::uuid,NULLIF($16,'')::uuid,NULLIF($17,'')::uuid,NULLIF($18,'')::uuid,NULLIF($19,'')::uuid,$20::uuid,$21,$22,$23)
		ON CONFLICT (id) DO UPDATE SET principal_kobo=EXCLUDED.principal_kobo,goods_description=EXCLUDED.goods_description,invoice_reference=EXCLUDED.invoice_reference,invoice_document_hash=EXCLUDED.invoice_document_hash,due_date=EXCLUDED.due_date,grace_hours=EXCLUDED.grace_hours,collection_at=EXCLUDED.collection_at,state=EXCLUDED.state, agreement_version_id=EXCLUDED.agreement_version_id, mandate_id=EXCLUDED.mandate_id, acceptance_id=EXCLUDED.acceptance_id, release_id=EXCLUDED.release_id, receipt_id=EXCLUDED.receipt_id, obligation_id=EXCLUDED.obligation_id, updated_at=EXCLUDED.updated_at, version=EXCLUDED.version`,
		r.ID, r.SupplierOrganizationID, r.BuyerUserID, r.BuyerBusinessID, int64(r.PrincipalKobo), r.Currency, r.GoodsDescription, r.InvoiceReference, r.InvoiceDocumentHash, r.DueDate, r.GraceHours, r.CollectionAt, r.State, r.AgreementVersionID, r.MandateID, r.AcceptanceID, r.ReleaseID, r.ReceiptID, r.ObligationID, r.CreatedBy, r.CreatedAt, r.UpdatedAt, r.Version, feeTermsJSON(r.FeeTerms))
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
		if _, err := tx.Exec(ctx, `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key) VALUES('credit_request',$1,'notification.requested',jsonb_build_object('event','OBLIGATION_ACCEPTED','acceptance_id',$2::text),$3) ON CONFLICT(idempotency_key) DO NOTHING`, r.ID, a.ID, "acceptance-notification:"+a.ID); err != nil {
			return err
		}
	}
	if release := view.Release; release != nil {
		if _, err := tx.Exec(ctx, `INSERT INTO app.goods_releases (id,credit_request_id,supplier_actor_id,delivery_method,notes,released_at) VALUES ($1::uuid,$2::uuid,$3::uuid,$4,NULLIF($5,''),$6) ON CONFLICT (id) DO NOTHING`, release.ID, release.CreditRequestID, release.SupplierActorID, release.DeliveryMethod, release.Notes, release.ReleasedAt); err != nil {
			return fmt.Errorf("persist goods release: %w", err)
		}
		// The buyer is told, in the same transaction that records the release,
		// that goods are out and that silence has a deadline. The idempotency
		// key is derived from the release so the delivery receipt for this
		// notice can be found again when deemed acceptance is considered.
		if _, err := tx.Exec(ctx, `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key) VALUES('credit_request',$1,'notification.requested',jsonb_build_object('event','GOODS_RELEASED','release_id',$2::text,'ends_at',$4::text,'amount_kobo',$5::bigint),$3) ON CONFLICT(idempotency_key) DO NOTHING`, r.ID, release.ID, "goods-release-notification:"+release.ID, release.DeemedAcceptedAt.Format(time.RFC3339), int64(r.PrincipalKobo)); err != nil {
			return fmt.Errorf("queue goods release notice: %w", err)
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

func feeTermsJSON(f *ledger.FeeTerms) []byte { b, _ := json.Marshal(f); return b }

// AutoActivateMatured promotes every request whose deemed-acceptance window has
// elapsed. It must not take s.mu itself: the embedded Store takes the same
// (non-reentrant) mutex, and each activation is persisted through the durable
// RecordReceipt path below.
func (s *PostgresStore) AutoActivateMatured(ctx context.Context, asOf time.Time) ([]string, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("credit database is not configured")
	}
	s.mu.RLock()
	candidates := make([]string, 0)
	for id, request := range s.requests {
		if request.State == ReceiptConfirmationPending && request.DeemedAcceptedAt != nil && !asOf.Before(*request.DeemedAcceptedAt) {
			candidates = append(candidates, id)
		}
	}
	s.mu.RUnlock()
	sort.Strings(candidates)
	activated := make([]string, 0, len(candidates))
	for _, id := range candidates {
		s.mu.RLock()
		request := s.requests[id]
		buyerID := ""
		if request != nil {
			buyerID = request.BuyerUserID
		}
		s.mu.RUnlock()
		if buyerID == "" {
			continue
		}
		// The first-obligation and delivered-notice checks are answered from
		// PostgreSQL rather than the projection: the projection is a cache of
		// whatever this process happened to load, and absence from it is not
		// evidence that a buyer has no history.
		if err := s.deemedAcceptancePermitted(ctx, id); err != nil {
			continue
		}
		if _, _, err := s.RecordReceipt(id, buyerID, "confirmed", deemedAcceptanceReason); err == nil {
			activated = append(activated, id)
		}
	}
	return activated, nil
}
