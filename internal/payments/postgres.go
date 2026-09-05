package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"kredit/internal/db"

	"kredit/internal/ledger"
	"kredit/internal/outbox"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore is the production payment system of record. Each mutation
// commits the payment, schedule allocations, obligation balance, journal,
// fees, aggregate snapshot, and outbox event in one database transaction.
type PostgresStore struct {
	pool       *pgxpool.Pool
	outbox     *outbox.Store
	invalidate func(string)
	now        func() time.Time
}

var _ Service = (*PostgresStore)(nil)

func NewPostgresStore(pool *pgxpool.Pool, events *outbox.Store, invalidate func(string)) *PostgresStore {
	return &PostgresStore{pool: pool, outbox: events, invalidate: invalidate, now: func() time.Time { return time.Now().UTC() }}
}

func (s *PostgresStore) Record(input RecordInput) (Payment, Allocation, error) {
	return s.RecordContext(context.Background(), input)
}

func (s *PostgresStore) RecordContext(ctx context.Context, input RecordInput) (Payment, Allocation, error) {
	if s == nil || s.pool == nil {
		return Payment{}, Allocation{}, errors.New("payment database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Payment{}, Allocation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	payment, allocation, err := s.RecordTx(ctx, tx, input)
	if err != nil {
		return Payment{}, Allocation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Payment{}, Allocation{}, err
	}
	s.AfterCommit(payment.ObligationID)
	return payment, allocation, nil
}

// AfterCommit invalidates cached projections only after the owning transaction commits.
func (s *PostgresStore) AfterCommit(obligationID string) {
	if s.invalidate != nil {
		s.invalidate(obligationID)
	}
}

// RecordTx composes payment recognition with another domain mutation. The caller
// owns rollback/commit and must call AfterCommit following a successful commit.
func (s *PostgresStore) RecordTx(ctx context.Context, tx pgx.Tx, input RecordInput) (Payment, Allocation, error) {
	if s == nil || s.pool == nil {
		return Payment{}, Allocation{}, errors.New("payment database is not configured")
	}
	if input.ObligationID == "" || input.AmountKobo <= 0 || input.RecordedBy == "" || input.IdempotencyKey == "" {
		return Payment{}, Allocation{}, errors.New("obligation, amount, recorder, and idempotency key are required")
	}
	if !validSource(input.SourceType) {
		return Payment{}, Allocation{}, errors.New("invalid payment source")
	}
	if input.SourceType == SourceCollected && !validCollectedPayment(input) {
		return Payment{}, Allocation{}, errors.New("collected payments require collection-worker provenance, provider identity, and an attempt idempotency key")
	}

	// Tenant identity must come from an authenticated HTTP boundary or a
	// tenant-scoped worker job, never from the obligation being requested.
	if err := db.SetObligationContext(ctx, tx, input.ObligationID); err != nil {
		return Payment{}, Allocation{}, err
	}
	if existing, allocation, found, err := loadByIdempotency(ctx, tx, input.IdempotencyKey); err != nil {
		return Payment{}, Allocation{}, err
	} else if found {
		if !samePaymentIntent(existing, input) {
			return Payment{}, Allocation{}, errors.New("idempotency key was reused for a different payment")
		}
		return existing, allocation, nil
	}

	var snapshot ObligationSnapshot
	var creditRequestID string
	err := tx.QueryRow(ctx, `
		SELECT o.id::text, c.buyer_user_id::text, o.supplier_organization_id::text,
		       o.principal_kobo, o.outstanding_kobo, c.collection_at, o.currency, c.id::text, c.fee_terms
		FROM app.obligations o
		JOIN app.credit_requests c ON c.id = o.credit_request_id
		WHERE o.id = $1::uuid
		FOR UPDATE OF o, c`, input.ObligationID).Scan(
		&snapshot.ID, &snapshot.BuyerUserID, &snapshot.SupplierOrganizationID,
		&snapshot.PrincipalKobo, &snapshot.OutstandingKobo, &snapshot.CollectionAt,
		&snapshot.Currency, &creditRequestID, &snapshot.FeeTerms)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, Allocation{}, errors.New("obligation not found")
	}
	if err != nil {
		return Payment{}, Allocation{}, fmt.Errorf("lock obligation: %w", err)
	}
	if existing, allocation, found, err := loadByIdempotency(ctx, tx, input.IdempotencyKey); err != nil {
		return Payment{}, Allocation{}, err
	} else if found {
		if !samePaymentIntent(existing, input) {
			return Payment{}, Allocation{}, errors.New("idempotency key was reused for a different payment")
		}
		return existing, allocation, nil
	}

	if defaultCurrency(input.Currency, snapshot.Currency) != snapshot.Currency {
		return Payment{}, Allocation{}, errors.New("payment currency must match the obligation")
	}
	if input.AmountKobo > snapshot.OutstandingKobo {
		return Payment{}, Allocation{}, errors.New("payment exceeds authoritative outstanding amount")
	}

	now := s.now()
	paidAt := input.PaidAt
	if paidAt.IsZero() {
		paidAt = now
	}
	if paidAt.After(now.Add(5 * time.Minute)) {
		return Payment{}, Allocation{}, errors.New("payment date cannot be in the future")
	}
	payment := Payment{
		ID: newIdentifier(), ObligationID: input.ObligationID,
		BuyerUserID: snapshot.BuyerUserID, SupplierOrganizationID: snapshot.SupplierOrganizationID,
		SourceType: input.SourceType, AmountKobo: input.AmountKobo,
		Currency: defaultCurrency(input.Currency, snapshot.Currency), Provider: strings.TrimSpace(input.Provider),
		ProviderReference: strings.TrimSpace(input.ProviderReference), State: StateRecognized,
		PaidAt: paidAt, RecognizedAt: now, RecordedBy: input.RecordedBy,
	}
	if payment.SourceType == SourceCollected && !paidAt.Before(snapshot.CollectionAt) {
		fee, feeErr := snapshot.FeeTerms.Collection(payment.AmountKobo)
		if feeErr != nil {
			return Payment{}, Allocation{}, feeErr
		}
		payment.CollectionFeeKobo = fee
	}
	command, err := tx.Exec(ctx, `
		INSERT INTO app.payments
		(id,obligation_id,buyer_user_id,supplier_organization_id,source_type,amount_kobo,currency,provider,provider_reference,state,paid_at,recognized_at,recorded_by,recorded_by_reference,idempotency_key,collection_fee_kobo)
		VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,NULLIF($8,''),NULLIF($9,''),$10,$11,$12,
		        CASE WHEN $13 ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$' THEN $13::uuid ELSE NULL END,
		$13,$14,$15)
		ON CONFLICT (idempotency_key) DO NOTHING`, payment.ID, payment.ObligationID, payment.BuyerUserID,
		payment.SupplierOrganizationID, payment.SourceType, int64(payment.AmountKobo), payment.Currency,
		payment.Provider, payment.ProviderReference, payment.State, payment.PaidAt, payment.RecognizedAt,
		payment.RecordedBy, input.IdempotencyKey, int64(payment.CollectionFeeKobo))
	if err != nil {
		return Payment{}, Allocation{}, fmt.Errorf("insert payment: %w", err)
	}
	if command.RowsAffected() == 0 {
		existing, allocation, found, loadErr := loadByIdempotency(ctx, tx, input.IdempotencyKey)
		if loadErr != nil {
			return Payment{}, Allocation{}, loadErr
		}
		if !found {
			return Payment{}, Allocation{}, errors.New("concurrent idempotent payment could not be loaded")
		}
		if !samePaymentIntent(existing, input) {
			return Payment{}, Allocation{}, errors.New("idempotency key was reused for a different payment")
		}
		return existing, allocation, nil
	}

	allocations, err := allocateScheduleTx(ctx, tx, payment)
	if err != nil {
		return Payment{}, Allocation{}, err
	}
	newOutstanding := snapshot.OutstandingKobo - payment.AmountKobo
	if err := updateOutstandingTx(ctx, tx, creditRequestID, payment.ObligationID, newOutstanding, snapshot.PrincipalKobo); err != nil {
		return Payment{}, Allocation{}, err
	}
	settlementAccount := ledger.AccountVoluntarySettlement
	if payment.SourceType == SourceCollected {
		settlementAccount = ledger.AccountCollectionSettlement
	}
	if err := postLedgerTx(ctx, tx, "payment_recognized", payment.ID, "payment:"+input.IdempotencyKey, paidAt,
		settlementAccount, ledger.AccountTradeReceivable, payment.AmountKobo); err != nil {
		return Payment{}, Allocation{}, err
	}
	if payment.CollectionFeeKobo > 0 {
		if _, err := tx.Exec(ctx, `INSERT INTO app.fees (supplier_organization_id,obligation_id,payment_id,fee_type,basis_amount_kobo,rate_basis_points,amount_kobo,currency,state,accrued_at) VALUES ($1::uuid,$2::uuid,$3::uuid,'collection',$4,$8,$5,$6,'accrued',$7) ON CONFLICT (payment_id,fee_type) DO NOTHING`, payment.SupplierOrganizationID, payment.ObligationID, payment.ID, int64(payment.AmountKobo), int64(payment.CollectionFeeKobo), payment.Currency, paidAt, collectionRate(snapshot.FeeTerms)); err != nil {
			return Payment{}, Allocation{}, fmt.Errorf("insert collection fee: %w", err)
		}
		if err := postLedgerTx(ctx, tx, "collection_fee_accrued", payment.ID, "collection-fee:"+input.IdempotencyKey, paidAt,
			ledger.AccountSupplierFeeReceivable, ledger.AccountPlatformCollectionRevenue, payment.CollectionFeeKobo); err != nil {
			return Payment{}, Allocation{}, err
		}
	}
	if err := s.appendEvent(ctx, tx, payment.ID, "payment.recognized", payment, "payment:"+input.IdempotencyKey); err != nil {
		return Payment{}, Allocation{}, err
	}
	if newOutstanding == 0 {
		if err := s.appendEvent(ctx, tx, payment.ID, "notification.requested", map[string]any{"event": "OBLIGATION_FULLY_REPAID", "obligation_id": payment.ObligationID}, "repaid:"+payment.ID); err != nil {
			return Payment{}, Allocation{}, err
		}
	}
	return payment, allocations[0], nil
}

func allocateScheduleTx(ctx context.Context, tx pgx.Tx, payment Payment) ([]Allocation, error) {
	rows, err := tx.Query(ctx, `
		SELECT i.id::text, i.principal_due_kobo-i.allocated_kobo
		FROM app.schedule_items i
		JOIN app.repayment_schedules s ON s.id=i.schedule_id
		WHERE s.obligation_id=$1::uuid AND i.state NOT IN ('PAID','CANCELLED')
		ORDER BY i.sequence FOR UPDATE OF i`, payment.ObligationID)
	if err != nil {
		return nil, err
	}
	type openItem struct {
		id     string
		amount ledger.Money
	}
	items := []openItem{}
	for rows.Next() {
		var item openItem
		if err := rows.Scan(&item.id, &item.amount); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	remaining := payment.AmountKobo
	allocations := []Allocation{}
	for _, item := range items {
		if remaining == 0 {
			break
		}
		take := item.amount
		if take > remaining {
			take = remaining
		}
		collected := ledger.Money(0)
		if payment.SourceType == SourceCollected {
			collected = take
		}
		if _, err := tx.Exec(ctx, `UPDATE app.schedule_items SET allocated_kobo=allocated_kobo+$2, collected_kobo=collected_kobo+$3, state=CASE WHEN allocated_kobo+$2=principal_due_kobo THEN 'PAID' ELSE 'PARTIALLY_PAID' END WHERE id=$1::uuid`, item.id, int64(take), int64(collected)); err != nil {
			return nil, err
		}
		allocation := Allocation{PaymentID: payment.ID, ObligationID: payment.ObligationID, ScheduleItemID: item.id, AmountKobo: take, CreatedAt: payment.RecognizedAt}
		allocations = append(allocations, allocation)
		remaining -= take
	}
	if len(items) > 0 && remaining > 0 {
		return nil, errors.New("allocation exceeds schedule outstanding")
	}
	if len(allocations) == 0 {
		allocations = append(allocations, Allocation{PaymentID: payment.ID, ObligationID: payment.ObligationID, AmountKobo: payment.AmountKobo, CreatedAt: payment.RecognizedAt})
	}
	for index, allocation := range allocations {
		if _, err := tx.Exec(ctx, `INSERT INTO app.payment_allocations (payment_id,obligation_id,schedule_item_id,amount_kobo,allocation_order,created_at) VALUES ($1::uuid,$2::uuid,NULLIF($3,'')::uuid,$4,$5,$6)`, allocation.PaymentID, allocation.ObligationID, allocation.ScheduleItemID, int64(allocation.AmountKobo), index+1, allocation.CreatedAt); err != nil {
			return nil, fmt.Errorf("insert payment allocation: %w", err)
		}
	}
	return allocations, nil
}

func (s *PostgresStore) Reverse(paymentID, actor, reason string) (Payment, error) {
	return s.ReverseContext(context.Background(), paymentID, actor, reason)
}

func (s *PostgresStore) ReverseContext(ctx context.Context, paymentID, actor, reason string) (Payment, error) {
	if s == nil || s.pool == nil {
		return Payment{}, errors.New("payment database is not configured")
	}
	if paymentID == "" || actor == "" || strings.TrimSpace(reason) == "" {
		return Payment{}, errors.New("payment, reversal actor, and reason are required")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Payment{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := db.SetTenantContext(ctx, tx); err != nil {
		return Payment{}, err
	}
	payment, err := loadPayment(ctx, tx, paymentID, true)
	if err != nil {
		return Payment{}, err
	}
	if payment.State == StateReversed {
		return payment, nil
	}
	var principal ledger.Money
	var creditRequestID string
	if err := tx.QueryRow(ctx, `SELECT principal_kobo,credit_request_id::text FROM app.obligations WHERE id=$1::uuid FOR UPDATE`, payment.ObligationID).Scan(&principal, &creditRequestID); err != nil {
		return Payment{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE app.payments SET state='reversed' WHERE id=$1::uuid`, payment.ID); err != nil {
		return Payment{}, err
	}
	now := s.now()
	reversalID := newIdentifier()
	if _, err := tx.Exec(ctx, `INSERT INTO app.payments (id,obligation_id,buyer_user_id,supplier_organization_id,source_type,amount_kobo,currency,provider,provider_reference,state,paid_at,recognized_at,recorded_by,recorded_by_reference,reversal_of,idempotency_key,collection_fee_kobo) VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,NULLIF($8,''),NULL,'reversed',$9,$10,CASE WHEN $11 ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$' THEN $11::uuid ELSE NULL END,$11,$12::uuid,$13,$14)`, reversalID, payment.ObligationID, payment.BuyerUserID, payment.SupplierOrganizationID, payment.SourceType, int64(payment.AmountKobo), payment.Currency, payment.Provider, payment.PaidAt, now, actor, payment.ID, "reversal:"+payment.ID, int64(payment.CollectionFeeKobo)); err != nil {
		return Payment{}, fmt.Errorf("insert payment reversal: %w", err)
	}
	rows, err := tx.Query(ctx, `SELECT schedule_item_id::text,amount_kobo FROM app.payment_allocations WHERE payment_id=$1::uuid AND schedule_item_id IS NOT NULL ORDER BY allocation_order`, payment.ID)
	if err != nil {
		return Payment{}, err
	}
	type target struct {
		id     string
		amount ledger.Money
	}
	targets := []target{}
	for rows.Next() {
		var item target
		if err := rows.Scan(&item.id, &item.amount); err != nil {
			rows.Close()
			return Payment{}, err
		}
		targets = append(targets, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return Payment{}, err
	}
	for _, item := range targets {
		collected := ledger.Money(0)
		if payment.SourceType == SourceCollected {
			collected = item.amount
		}
		command, err := tx.Exec(ctx, `UPDATE app.schedule_items SET allocated_kobo=allocated_kobo-$2,collected_kobo=collected_kobo-$3,state=CASE WHEN allocated_kobo-$2=0 THEN CASE WHEN now()<due_at THEN 'OPEN' WHEN now()<collection_at THEN 'IN_GRACE' ELSE 'OVERDUE' END ELSE 'PARTIALLY_PAID' END WHERE id=$1::uuid AND allocated_kobo >= $2 AND collected_kobo >= $3`, item.id, int64(item.amount), int64(collected))
		if err != nil {
			return Payment{}, err
		}
		if command.RowsAffected() != 1 {
			return Payment{}, fmt.Errorf("schedule item %s no longer holds the reversed allocation", item.id)
		}
	}
	var outstanding ledger.Money
	if err := tx.QueryRow(ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, payment.ObligationID).Scan(&outstanding); err != nil {
		return Payment{}, err
	}
	nextOutstanding, err := ledger.CheckedAdd(outstanding, payment.AmountKobo)
	if err != nil || nextOutstanding > principal {
		return Payment{}, errors.New("reversal would exceed original principal")
	}
	if err := updateOutstandingTx(ctx, tx, creditRequestID, payment.ObligationID, nextOutstanding, principal); err != nil {
		return Payment{}, err
	}
	settlement := ledger.AccountVoluntarySettlement
	if payment.SourceType == SourceCollected {
		settlement = ledger.AccountCollectionSettlement
	}
	if err := postLedgerTx(ctx, tx, "payment_reversed", payment.ID, "reversal:"+payment.ID, now, ledger.AccountTradeReceivable, settlement, payment.AmountKobo); err != nil {
		return Payment{}, err
	}
	if payment.CollectionFeeKobo > 0 {
		if _, err := tx.Exec(ctx, `UPDATE app.fees SET state='refunded' WHERE payment_id=$1::uuid AND fee_type='collection'`, payment.ID); err != nil {
			return Payment{}, err
		}
		if err := postLedgerTx(ctx, tx, "collection_fee_reversed", payment.ID, "collection-fee-reversal:"+payment.ID, now, ledger.AccountPlatformCollectionRevenue, ledger.AccountSupplierFeeReceivable, payment.CollectionFeeKobo); err != nil {
			return Payment{}, err
		}
	}
	payment.State = StateReversed
	if err := s.appendEvent(ctx, tx, payment.ID, "payment.reversed", map[string]any{"payment_id": payment.ID, "reversal_id": reversalID, "reason": strings.TrimSpace(reason)}, "payment-reversal:"+payment.ID); err != nil {
		return Payment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Payment{}, err
	}
	if s.invalidate != nil {
		s.invalidate(payment.ObligationID)
	}
	return payment, nil
}

// List delegates to Read so a query failure surfaces as an error rather than an
// empty payment history.
func (s *PostgresStore) List(obligationID string) ([]Payment, error) {
	return s.Read(obligationID)
}

func (s *PostgresStore) Get(paymentID string) (Payment, error) {
	return s.GetContext(context.Background(), paymentID)
}

func (s *PostgresStore) GetContext(ctx context.Context, paymentID string) (Payment, error) {
	if s == nil || s.pool == nil {
		return Payment{}, errors.New("payment database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Payment{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := db.SetTenantContext(ctx, tx); err != nil {
		return Payment{}, err
	}
	return loadPayment(ctx, tx, paymentID, false)
}

func (s *PostgresStore) Rebuild(obligationID string) (ledger.Money, error) {
	return s.RebuildContext(context.Background(), obligationID)
}

func (s *PostgresStore) RebuildContext(ctx context.Context, obligationID string) (ledger.Money, error) {
	if s == nil || s.pool == nil {
		return 0, errors.New("payment database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := db.SetObligationContext(ctx, tx, obligationID); err != nil {
		return 0, err
	}
	var principal, current ledger.Money
	var requestID string
	if err := tx.QueryRow(ctx, `SELECT principal_kobo,outstanding_kobo,credit_request_id::text FROM app.obligations WHERE id=$1::uuid FOR UPDATE`, obligationID).Scan(&principal, &current, &requestID); err != nil {
		return 0, err
	}
	var paid ledger.Money
	if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(amount_kobo),0) FROM app.payments WHERE obligation_id=$1::uuid AND state='recognized'`, obligationID).Scan(&paid); err != nil {
		return 0, err
	}
	var forgiven ledger.Money
	if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(p.credit_kobo-p.debit_kobo),0) FROM ledger.postings p JOIN ledger.accounts a ON a.id=p.account_id JOIN ledger.transactions t ON t.id=p.transaction_id WHERE a.code=$2 AND ((t.event_type='write_off' AND t.reference_type='obligation' AND t.reference_id=$1) OR (t.event_type='dispute_adjustment' AND t.reference_type='dispute' AND EXISTS(SELECT 1 FROM app.disputes d WHERE d.id::text=t.reference_id AND d.obligation_id=$1::uuid)))`, obligationID, ledger.AccountTradeReceivable).Scan(&forgiven); err != nil {
		return 0, err
	}
	if paid < 0 || paid > principal || forgiven < 0 || forgiven > principal-paid {
		return 0, errors.New("recognized reductions exceed principal")
	}
	expected := principal - paid - forgiven
	if expected < 0 {
		return 0, errors.New("payments exceed principal")
	}
	if expected != current {
		if err := db.GuardUnreservedReduction(ctx, tx, obligationID, int64(expected)); err != nil {
			return 0, err
		}
		if err := s.appendEvent(ctx, tx, obligationID, "obligation.balance_rebuilt", map[string]any{"previous_kobo": current, "outstanding_kobo": expected}, "balance-rebuild:"+newIdentifier()); err != nil {
			return 0, err
		}
		if err := updateOutstandingTx(ctx, tx, requestID, obligationID, expected, principal); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	if s.invalidate != nil {
		s.invalidate(obligationID)
	}
	return expected, nil
}

const paymentSelect = `SELECT p.id::text,p.obligation_id::text,p.buyer_user_id::text,p.supplier_organization_id::text,p.source_type,p.amount_kobo,p.currency,COALESCE(p.provider,''),COALESCE(p.provider_reference,''),p.state,p.paid_at,p.recognized_at,p.recorded_by_reference,COALESCE(p.reversal_of::text,''),p.collection_fee_kobo FROM app.payments p`

type rowScanner interface{ Scan(...any) error }

func scanPayment(row rowScanner) (Payment, error) {
	var p Payment
	err := row.Scan(&p.ID, &p.ObligationID, &p.BuyerUserID, &p.SupplierOrganizationID, &p.SourceType, &p.AmountKobo, &p.Currency, &p.Provider, &p.ProviderReference, &p.State, &p.PaidAt, &p.RecognizedAt, &p.RecordedBy, &p.ReversalOf, &p.CollectionFeeKobo)
	return p, err
}
func loadPayment(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id string, lock bool) (Payment, error) {
	suffix := ` WHERE p.id=$1::uuid`
	if lock {
		suffix += ` FOR UPDATE`
	}
	p, err := scanPayment(q.QueryRow(ctx, paymentSelect+suffix, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, errors.New("payment not found")
	}
	return p, err
}
func loadByIdempotency(ctx context.Context, tx pgx.Tx, key string) (Payment, Allocation, bool, error) {
	p, err := scanPayment(tx.QueryRow(ctx, paymentSelect+` WHERE p.idempotency_key=$1`, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, Allocation{}, false, nil
	}
	if err != nil {
		return Payment{}, Allocation{}, false, err
	}
	var a Allocation
	err = tx.QueryRow(ctx, `SELECT payment_id::text,obligation_id::text,COALESCE(schedule_item_id::text,''),amount_kobo,created_at FROM app.payment_allocations WHERE payment_id=$1::uuid ORDER BY allocation_order LIMIT 1`, p.ID).Scan(&a.PaymentID, &a.ObligationID, &a.ScheduleItemID, &a.AmountKobo, &a.CreatedAt)
	return p, a, true, err
}

func updateOutstandingTx(ctx context.Context, tx pgx.Tx, requestID, obligationID string, outstanding, principal ledger.Money) error {
	var status string
	switch outstanding {
	case 0:
		status = "PAID"
	case principal:
		status = "UNPAID"
	default:
		status = "PARTIALLY_PAID"
	}
	if _, err := tx.Exec(ctx, `UPDATE app.obligations SET outstanding_kobo=$2,payment_status=$3 WHERE id=$1::uuid`, obligationID, int64(outstanding), status); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE app.credit_requests SET version=version+1,updated_at=now() WHERE id=$1::uuid`, requestID); err != nil {
		return err
	}
	command, err := tx.Exec(ctx, `UPDATE app.credit_aggregate_snapshots SET aggregate=jsonb_set(jsonb_set(jsonb_set(aggregate,'{obligation,outstanding_kobo}',to_jsonb($2::bigint),false),'{obligation,payment_status}',to_jsonb($3::text),false),'{request,version}',to_jsonb(version+1),false),version=version+1,updated_at=now() WHERE credit_request_id=$1`, requestID, int64(outstanding), status)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return errors.New("credit aggregate snapshot not found")
	}
	return nil
}

func postLedgerTx(ctx context.Context, tx pgx.Tx, eventType, referenceID, key string, effectiveAt time.Time, debitAccount, creditAccount string, amount ledger.Money) error {
	var id string
	err := tx.QueryRow(ctx, `INSERT INTO ledger.transactions(event_type,reference_type,reference_id,idempotency_key,effective_at) VALUES($1,'payment',$2,$3,$4) ON CONFLICT(idempotency_key) DO NOTHING RETURNING id::text`, eventType, referenceID, key, effectiveAt).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, posting := range []struct {
		account       string
		debit, credit ledger.Money
	}{{debitAccount, amount, 0}, {creditAccount, 0, amount}} {
		if _, err := tx.Exec(ctx, `INSERT INTO ledger.postings(transaction_id,account_id,debit_kobo,credit_kobo) SELECT $1::uuid,id,$3,$4 FROM ledger.accounts WHERE code=$2`, id, posting.account, int64(posting.debit), int64(posting.credit)); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresStore) appendEvent(ctx context.Context, tx pgx.Tx, aggregateID, eventType string, value any, key string) error {
	if s.outbox == nil {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.outbox.AppendTx(ctx, tx, outbox.Event{AggregateType: "payment", AggregateID: aggregateID, EventType: eventType, Payload: payload, IdempotencyKey: key})
	return err
}

func (s *PostgresStore) Read(obligationID string) ([]Payment, error) {
	return s.ReadContext(context.Background(), obligationID)
}

func (s *PostgresStore) ReadContext(ctx context.Context, obligationID string) ([]Payment, error) {
	if s == nil || s.pool == nil || obligationID == "" {
		return nil, errors.New("payment database or obligation unavailable")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := db.SetTenantContext(ctx, tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, paymentSelect+` WHERE p.obligation_id=$1::uuid ORDER BY p.recognized_at,p.id`, obligationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Payment{}
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func collectionRate(terms *ledger.FeeTerms) int64 { _, rate := terms.Rates(); return rate }
