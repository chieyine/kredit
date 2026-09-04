package tradelines

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"kredit/internal/businesspolicy"
	"kredit/internal/ledger"
	"kredit/internal/outbox"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore serializes each trade-line aggregate with a row lock and
// commits the line, drawdowns, and reservations together.
type PostgresStore struct {
	*Store
	pool                           *pgxpool.Pool
	outbox                         *outbox.Store
	transactionalActivationHandler func(context.Context, pgx.Tx, ActivationInput) (string, func(), error)
}

var _ Service = (*PostgresStore)(nil)

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{Store: NewStore(), pool: pool}
}

func NewPostgresStoreWithOutbox(pool *pgxpool.Pool, outboxStore *outbox.Store) *PostgresStore {
	return &PostgresStore{Store: NewStore(), pool: pool, outbox: outboxStore}
}

func (s *PostgresStore) SetTransactionalActivationHandler(handler func(context.Context, pgx.Tx, ActivationInput) (string, func(), error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.transactionalActivationHandler = handler
}

func (s *PostgresStore) local() *Store {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &Store{lines: map[string]*TradeLine{}, drawdowns: map[string]*Drawdown{}, reservations: map[string]*Reservation{}, byKey: map[string]string{}, byLine: map[string][]string{}, now: s.now, newID: s.newID, lineGuard: s.lineGuard, maxDrawdownsPerLineDay: s.maxDrawdownsPerLineDay, maxActiveExposureKobo: s.maxActiveExposureKobo, activationHandler: s.activationHandler}
}

func (s *PostgresStore) CreateLine(input CreateLineInput) (TradeLine, error) {
	if s == nil || s.pool == nil {
		return TradeLine{}, errors.New("trade-line database is not configured")
	}
	local := s.local()
	line, err := local.CreateLine(input)
	if err != nil {
		return TradeLine{}, err
	}
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return TradeLine{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, input.BuyerBusinessID); err != nil {
		return TradeLine{}, err
	}
	if local.maxActiveExposureKobo > 0 {
		var exposure ledger.Money
		if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(current_exposure_kobo+reserved_pending_kobo),0) FROM app.trade_lines WHERE buyer_business_id=$1::uuid AND state NOT IN('CLOSED','EXPIRED')`, input.BuyerBusinessID).Scan(&exposure); err != nil {
			return TradeLine{}, err
		}
		totalExposure, addErr := ledger.CheckedAdd(exposure, input.ApprovedLimitKobo)
		if addErr != nil || totalExposure > local.maxActiveExposureKobo {
			return TradeLine{}, errors.New("buyer active exposure exceeds configured pilot limit")
		}
	}
	if err := persistLineTx(ctx, tx, line); err != nil {
		return TradeLine{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return TradeLine{}, err
	}
	return line, nil
}

func (s *PostgresStore) Get(id string) (TradeLine, bool) {
	if s == nil || s.pool == nil {
		return TradeLine{}, false
	}
	line, err := scanLine(s.pool.QueryRow(context.Background(), lineSelect+` WHERE id=$1::uuid`, id))
	return line, err == nil
}
func (s *PostgresStore) ListForSupplier(org string) []TradeLine {
	return s.list(`supplier_organization_id=$1::uuid`, org)
}
func (s *PostgresStore) ListForBuyer(user string) []TradeLine {
	return s.list(`buyer_user_id=$1::uuid`, user)
}
func (s *PostgresStore) list(where, value string) []TradeLine {
	if s == nil || s.pool == nil || value == "" {
		return []TradeLine{}
	}
	rows, err := s.pool.Query(context.Background(), lineSelect+` WHERE `+where+` ORDER BY updated_at DESC`, value)
	if err != nil {
		return []TradeLine{}
	}
	defer rows.Close()
	out := []TradeLine{}
	for rows.Next() {
		line, err := scanLine(rows)
		if err != nil {
			return []TradeLine{}
		}
		out = append(out, line)
	}
	if rows.Err() != nil {
		return []TradeLine{}
	}
	return out
}

func (s *PostgresStore) ReserveDrawdown(input CreateDrawdownInput) (Drawdown, Reservation, TradeLine, error) {
	ctx := context.Background()
	tx, local, lineID, err := s.loadForMutation(ctx, input.LineID, "")
	if err != nil {
		return Drawdown{}, Reservation{}, TradeLine{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	policy, err := businesspolicy.ReadTx(ctx, tx)
	if err != nil {
		return Drawdown{}, Reservation{}, TradeLine{}, err
	}
	input.FeeTerms = &ledger.FeeTerms{PolicyRevision: policy.Revision, BaseBPS: policy.Values.BaseFeeBPS, CollectionBPS: policy.Values.CollectionFeeBPS}
	drawdown, reservation, line, err := local.ReserveDrawdown(input)
	if err != nil {
		return Drawdown{}, Reservation{}, TradeLine{}, err
	}
	if err := persistAggregateTx(ctx, tx, local, lineID); err != nil {
		return Drawdown{}, Reservation{}, TradeLine{}, err
	}
	if err := s.appendAggregateEventsTx(ctx, tx, local, lineID); err != nil {
		return Drawdown{}, Reservation{}, TradeLine{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Drawdown{}, Reservation{}, TradeLine{}, err
	}
	return drawdown, reservation, line, nil
}
func (s *PostgresStore) ConfirmDrawdown(drawdownID, buyer, agreementHash string) (Drawdown, TradeLine, error) {
	ctx := context.Background()
	tx, local, lineID, err := s.loadForMutation(ctx, "", drawdownID)
	if err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	drawdown, line, err := local.ConfirmDrawdown(drawdownID, buyer, agreementHash)
	if err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	if err := persistAggregateTx(ctx, tx, local, lineID); err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	if err := s.appendAggregateEventsTx(ctx, tx, local, lineID); err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	return drawdown, line, nil
}
func (s *PostgresStore) ReleaseDrawdown(input ReleaseInput) (Drawdown, TradeLine, error) {
	ctx := context.Background()
	tx, local, lineID, err := s.loadForMutation(ctx, "", input.DrawdownID)
	if err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	drawdown, line, err := local.ReleaseDrawdown(input)
	if err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	if err := persistAggregateTx(ctx, tx, local, lineID); err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	if err := s.appendAggregateEventsTx(ctx, tx, local, lineID); err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	return drawdown, line, nil
}
func (s *PostgresStore) RecordDrawdownReceipt(input ReceiptInput) (Drawdown, TradeLine, error) {
	ctx := context.Background()
	tx, local, lineID, err := s.loadForMutation(ctx, "", input.DrawdownID)
	if err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var finalize func()
	s.mu.RLock()
	transactionalHandler := s.transactionalActivationHandler
	s.mu.RUnlock()
	if transactionalHandler != nil {
		local.SetActivationHandler(func(activation ActivationInput) (string, error) {
			obligationID, commitProjection, err := transactionalHandler(ctx, tx, activation)
			if err == nil {
				finalize = commitProjection
			}
			return obligationID, err
		})
	}
	drawdown, line, err := local.RecordDrawdownReceipt(input)
	if err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	if err := persistAggregateTx(ctx, tx, local, lineID); err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	if err := s.appendAggregateEventsTx(ctx, tx, local, lineID); err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	if finalize != nil {
		finalize()
	}
	return drawdown, line, nil
}
func (s *PostgresStore) CancelDrawdown(authorizedLineID, drawdownID, actorID string) (Drawdown, TradeLine, error) {
	if authorizedLineID == "" {
		return Drawdown{}, TradeLine{}, errors.New("authorized trade line is required")
	}
	ctx := context.Background()
	tx, local, lineID, err := s.loadForMutation(ctx, authorizedLineID, drawdownID)
	if err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	drawdown, line, err := local.CancelDrawdown(authorizedLineID, drawdownID, actorID)
	if err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	if err := persistAggregateTx(ctx, tx, local, lineID); err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	if err := s.appendAggregateEventsTx(ctx, tx, local, lineID); err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	return drawdown, line, nil
}

// UpdateOutstanding acknowledges an authoritative balance. Financial writes update
// exposure in the same database transaction through obligation_drawdown_exposure.
func (s *PostgresStore) UpdateOutstanding(drawdownID string, outstanding ledger.Money) (TradeLine, error) {
	ctx := context.Background()
	tx, local, lineID, err := s.loadForMutation(ctx, "", drawdownID)
	if err != nil {
		return TradeLine{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	d := local.drawdowns[drawdownID]
	if d == nil || d.State != DrawdownActivated || d.OutstandingKobo == nil || outstanding != *d.OutstandingKobo {
		return TradeLine{}, errors.New("outstanding must match the authoritative obligation")
	}
	return cloneLine(*local.lines[lineID]), nil
}
func (s *PostgresStore) Suspend(lineID, reason string) (TradeLine, error) {
	return s.mutateLine(lineID, func(local *Store) (TradeLine, error) { return local.Suspend(lineID, reason) })
}
func (s *PostgresStore) Resume(lineID string) (TradeLine, error) {
	return s.mutateLine(lineID, func(local *Store) (TradeLine, error) { return local.Resume(lineID) })
}
func (s *PostgresStore) ReduceLimit(lineID string, limit ledger.Money, expectedVersion int64) (TradeLine, error) {
	return s.mutateLine(lineID, func(local *Store) (TradeLine, error) { return local.ReduceLimit(lineID, limit, expectedVersion) })
}
func (s *PostgresStore) SetMandateState(lineID, mandateID string, active bool) (TradeLine, error) {
	return s.mutateLine(lineID, func(local *Store) (TradeLine, error) { return local.SetMandateState(lineID, mandateID, active) })
}
func (s *PostgresStore) mutateLine(lineID string, operation func(*Store) (TradeLine, error)) (TradeLine, error) {
	ctx := context.Background()
	tx, local, id, err := s.loadForMutation(ctx, lineID, "")
	if err != nil {
		return TradeLine{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	line, err := operation(local)
	if err != nil {
		return TradeLine{}, err
	}
	if err := persistAggregateTx(ctx, tx, local, id); err != nil {
		return TradeLine{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return TradeLine{}, err
	}
	return line, nil
}
func (s *PostgresStore) Statement(lineID string) (Statement, error) {
	if s == nil || s.pool == nil {
		return Statement{}, errors.New("trade-line database is not configured")
	}
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Statement{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	local := s.local()
	if err := loadAggregateTx(ctx, tx, local, lineID, false); err != nil {
		return Statement{}, err
	}
	return local.Statement(lineID)
}

func (s *PostgresStore) loadForMutation(ctx context.Context, lineID, drawdownID string) (pgx.Tx, *Store, string, error) {
	if s == nil || s.pool == nil {
		return nil, nil, "", errors.New("trade-line database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, "", err
	}
	if lineID == "" {
		if err := tx.QueryRow(ctx, `SELECT trade_line_id::text FROM app.drawdowns WHERE id=$1::uuid`, drawdownID).Scan(&lineID); err != nil {
			_ = tx.Rollback(ctx)
			return nil, nil, "", errors.New("drawdown not found")
		}
	}
	local := s.local()
	if err := loadAggregateTx(ctx, tx, local, lineID, true); err != nil {
		_ = tx.Rollback(ctx)
		return nil, nil, "", err
	}
	return tx, local, lineID, nil
}

const lineSelect = `SELECT id::text,supplier_organization_id::text,buyer_user_id::text,buyer_business_id::text,approved_limit_kobo,current_exposure_kobo,reserved_pending_kobo,available_limit_kobo,cadence,default_grace_hours,start_at,end_at,state,COALESCE(mandate_id::text,''),mandate_active,COALESCE(suspension_reason,''),terms_version,created_at,updated_at,version FROM app.trade_lines`

type scanner interface{ Scan(...any) error }

func scanLine(row scanner) (TradeLine, error) {
	var v TradeLine
	err := row.Scan(&v.ID, &v.SupplierOrganizationID, &v.BuyerUserID, &v.BuyerBusinessID, &v.ApprovedLimitKobo, &v.CurrentExposureKobo, &v.ReservedPendingKobo, &v.AvailableLimitKobo, &v.Cadence, &v.DefaultGraceHours, &v.StartAt, &v.EndAt, &v.State, &v.MandateID, &v.MandateActive, &v.SuspensionReason, &v.TermsVersion, &v.CreatedAt, &v.UpdatedAt, &v.Version)
	return v, err
}
func loadAggregateTx(ctx context.Context, tx pgx.Tx, local *Store, lineID string, lock bool) error {
	query := lineSelect + ` WHERE id=$1::uuid`
	if lock {
		query += ` FOR UPDATE`
	}
	line, err := scanLine(tx.QueryRow(ctx, query, lineID))
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("trade line not found")
	}
	if err != nil {
		return err
	}
	local.lines[line.ID] = &line
	rows, err := tx.Query(ctx, `SELECT id::text,trade_line_id::text,principal_kobo,goods_description,COALESCE(invoice_reference,''),COALESCE(invoice_document_hash,''),COALESCE(due_date::text,''),collection_at,grace_hours,terms_version,agreement_hash,state,COALESCE(reservation_id::text,''),COALESCE(obligation_id::text,''),buyer_confirmed_at,COALESCE(release_actor_id::text,''),COALESCE(delivery_method,''),COALESCE(release_notes,''),COALESCE(release_evidence_reference,''),released_at,COALESCE(receipt_state,''),COALESCE(receipt_actor_id::text,''),COALESCE(receipt_issue_reason,''),COALESCE(receipt_dispute_id::text,''),receipt_at,activated_at,created_at,fee_terms,(SELECT o.outstanding_kobo FROM app.obligations o WHERE o.id=app.drawdowns.obligation_id) FROM app.drawdowns WHERE trade_line_id=$1::uuid ORDER BY created_at`, lineID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var d Drawdown
		var confirmed, released, receiptAt, activated *time.Time
		if err := rows.Scan(&d.ID, &d.TradeLineID, &d.PrincipalKobo, &d.GoodsDescription, &d.InvoiceReference, &d.InvoiceDocumentHash, &d.DueDate, &d.CollectionAt, &d.GraceHours, &d.TermsVersion, &d.AgreementHash, &d.State, &d.ReservationID, &d.ObligationID, &confirmed, &d.ReleaseActorID, &d.DeliveryMethod, &d.ReleaseNotes, &d.ReleaseEvidenceReference, &released, &d.ReceiptState, &d.ReceiptActorID, &d.ReceiptIssueReason, &d.ReceiptDisputeID, &receiptAt, &activated, &d.CreatedAt, &d.FeeTerms, &d.OutstandingKobo); err != nil {
			rows.Close()
			return err
		}
		if confirmed != nil {
			d.BuyerConfirmedAt = *confirmed
		}
		if activated != nil {
			d.ActivatedAt = *activated
		}
		if released != nil {
			d.ReleasedAt = *released
		}
		if receiptAt != nil {
			d.ReceiptAt = *receiptAt
		}
		local.drawdowns[d.ID] = &d
		local.byLine[lineID] = append(local.byLine[lineID], d.ID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	rows, err = tx.Query(ctx, `SELECT id::text,trade_line_id::text,drawdown_id::text,amount_kobo,state,expires_at,idempotency_key,created_at FROM app.drawdown_reservations WHERE trade_line_id=$1::uuid`, lineID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var r Reservation
		if err := rows.Scan(&r.ID, &r.TradeLineID, &r.DrawdownID, &r.AmountKobo, &r.State, &r.ExpiresAt, &r.IdempotencyKey, &r.CreatedAt); err != nil {
			rows.Close()
			return err
		}
		local.reservations[r.ID] = &r
		local.byKey[r.IdempotencyKey] = r.DrawdownID
	}
	err = rows.Err()
	rows.Close()
	return err
}

func persistAggregateTx(ctx context.Context, tx pgx.Tx, local *Store, lineID string) error {
	line := local.lines[lineID]
	if line == nil {
		return errors.New("trade line not found")
	}
	if err := persistLineTx(ctx, tx, *line); err != nil {
		return err
	}
	for _, id := range local.byLine[lineID] {
		d := local.drawdowns[id]
		if d == nil {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO app.drawdowns(fee_terms,id,trade_line_id,principal_kobo,goods_description,invoice_reference,invoice_document_hash,due_date,collection_at,grace_hours,terms_version,agreement_hash,state,reservation_id,obligation_id,buyer_confirmed_at,release_actor_id,delivery_method,release_notes,release_evidence_reference,released_at,receipt_state,receipt_actor_id,receipt_issue_reason,receipt_dispute_id,receipt_at,activated_at,created_at) VALUES($28::jsonb,$1::uuid,$2::uuid,$3,$4,NULLIF($5,''),NULLIF($6,''),$7::date,$8,$9,$10,$11,$12,NULLIF($13,'')::uuid,NULLIF($14,'')::uuid,$15,NULLIF($16,'')::uuid,NULLIF($17,''),NULLIF($18,''),NULLIF($19,''),$20,NULLIF($21,''),NULLIF($22,'')::uuid,NULLIF($23,''),NULLIF($24,'')::uuid,$25,$26,$27) ON CONFLICT(id) DO UPDATE SET state=EXCLUDED.state,reservation_id=EXCLUDED.reservation_id,obligation_id=EXCLUDED.obligation_id,buyer_confirmed_at=EXCLUDED.buyer_confirmed_at,release_actor_id=EXCLUDED.release_actor_id,delivery_method=EXCLUDED.delivery_method,release_notes=EXCLUDED.release_notes,release_evidence_reference=EXCLUDED.release_evidence_reference,released_at=EXCLUDED.released_at,receipt_state=EXCLUDED.receipt_state,receipt_actor_id=EXCLUDED.receipt_actor_id,receipt_issue_reason=EXCLUDED.receipt_issue_reason,receipt_dispute_id=EXCLUDED.receipt_dispute_id,receipt_at=EXCLUDED.receipt_at,activated_at=EXCLUDED.activated_at`, d.ID, d.TradeLineID, int64(d.PrincipalKobo), d.GoodsDescription, d.InvoiceReference, d.InvoiceDocumentHash, d.DueDate, d.CollectionAt, d.GraceHours, d.TermsVersion, d.AgreementHash, d.State, d.ReservationID, d.ObligationID, nullableTime(d.BuyerConfirmedAt), d.ReleaseActorID, d.DeliveryMethod, d.ReleaseNotes, d.ReleaseEvidenceReference, nullableTime(d.ReleasedAt), d.ReceiptState, d.ReceiptActorID, d.ReceiptIssueReason, d.ReceiptDisputeID, nullableTime(d.ReceiptAt), nullableTime(d.ActivatedAt), d.CreatedAt, feeTermsJSON(d.FeeTerms)); err != nil {
			return fmt.Errorf("persist drawdown: %w", err)
		}
		if d.ReceiptDisputeID != "" {
			if _, err := tx.Exec(ctx, `INSERT INTO app.drawdown_receipt_disputes(id,drawdown_id,supplier_organization_id,buyer_user_id,state,reason,created_at,updated_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'OPEN',$5,$6,$6) ON CONFLICT(drawdown_id) DO UPDATE SET reason=EXCLUDED.reason,updated_at=EXCLUDED.updated_at`, d.ReceiptDisputeID, d.ID, line.SupplierOrganizationID, line.BuyerUserID, d.ReceiptIssueReason, d.ReceiptAt); err != nil {
				return fmt.Errorf("persist drawdown receipt dispute: %w", err)
			}
		}
	}
	for _, r := range local.reservations {
		if r.TradeLineID != lineID {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO app.drawdown_reservations(id,trade_line_id,drawdown_id,amount_kobo,state,expires_at,idempotency_key,created_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$8) ON CONFLICT(id) DO UPDATE SET state=EXCLUDED.state`, r.ID, r.TradeLineID, r.DrawdownID, int64(r.AmountKobo), r.State, r.ExpiresAt, r.IdempotencyKey, r.CreatedAt); err != nil {
			return fmt.Errorf("persist reservation: %w", err)
		}
	}
	return nil
}
func persistLineTx(ctx context.Context, tx pgx.Tx, v TradeLine) error {
	_, err := tx.Exec(ctx, `INSERT INTO app.trade_lines(id,supplier_organization_id,buyer_user_id,buyer_business_id,approved_limit_kobo,current_exposure_kobo,reserved_pending_kobo,available_limit_kobo,cadence,default_grace_hours,start_at,end_at,state,mandate_id,mandate_active,suspension_reason,terms_version,version,created_at,updated_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,$9,$10,$11,$12,$13,NULLIF($14,'')::uuid,$15,NULLIF($16,''),$17,$18,$19,$20) ON CONFLICT(id) DO UPDATE SET approved_limit_kobo=EXCLUDED.approved_limit_kobo,current_exposure_kobo=EXCLUDED.current_exposure_kobo,reserved_pending_kobo=EXCLUDED.reserved_pending_kobo,available_limit_kobo=EXCLUDED.available_limit_kobo,state=EXCLUDED.state,mandate_id=EXCLUDED.mandate_id,mandate_active=EXCLUDED.mandate_active,suspension_reason=EXCLUDED.suspension_reason,version=EXCLUDED.version,updated_at=EXCLUDED.updated_at`, v.ID, v.SupplierOrganizationID, v.BuyerUserID, v.BuyerBusinessID, int64(v.ApprovedLimitKobo), int64(v.CurrentExposureKobo), int64(v.ReservedPendingKobo), int64(v.AvailableLimitKobo), v.Cadence, v.DefaultGraceHours, v.StartAt, v.EndAt, v.State, v.MandateID, v.MandateActive, v.SuspensionReason, v.TermsVersion, v.Version, v.CreatedAt, v.UpdatedAt)
	return err
}
func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

func (s *PostgresStore) appendAggregateEventsTx(ctx context.Context, tx pgx.Tx, local *Store, lineID string) error {
	if s.outbox == nil {
		return nil
	}
	line := local.lines[lineID]
	if line == nil {
		return errors.New("trade line not found for outbox event")
	}
	for _, drawdownID := range local.byLine[lineID] {
		drawdown := local.drawdowns[drawdownID]
		if drawdown == nil {
			continue
		}
		events := []string{}
		switch drawdown.State {
		case DrawdownPending:
			events = append(events, "TradeLineDrawdownReserved")
		case DrawdownConfirmed:
			events = append(events, "TradeLineDrawdownConfirmed", "TradeLineDrawdownSafeToRelease")
		case DrawdownGoodsReleased:
			events = append(events, "TradeLineDrawdownGoodsReleased", "TradeLineDrawdownReceiptRequired")
		case DrawdownReceiptIssue:
			events = append(events, "TradeLineDrawdownReceiptIssueReported")
		case DrawdownActivated:
			events = append(events, "TradeLineDrawdownReceiptConfirmed", "TradeLineDrawdownActivated")
		case DrawdownCancelled:
			events = append(events, "TradeLineDrawdownCancelled")
		case DrawdownExpired:
			events = append(events, "TradeLineDrawdownExpired")
		}
		payload, err := json.Marshal(map[string]any{"trade_line_id": line.ID, "drawdown_id": drawdown.ID, "state": drawdown.State, "principal_kobo": drawdown.PrincipalKobo, "obligation_id": drawdown.ObligationID})
		if err != nil {
			return err
		}
		for _, eventType := range events {
			if _, err := s.outbox.AppendTx(ctx, tx, outbox.Event{AggregateType: "trade_line_drawdown", AggregateID: drawdown.ID, EventType: eventType, Payload: payload, IdempotencyKey: "trade-line-drawdown:" + drawdown.ID + ":" + eventType}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *PostgresStore) readList(where, value string) ([]TradeLine, error) {
	if s == nil || s.pool == nil || value == "" {
		return nil, errors.New("trade line database or scope is unavailable")
	}
	rows, err := s.pool.Query(context.Background(), lineSelect+` WHERE `+where+` ORDER BY updated_at DESC`, value)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TradeLine{}
	for rows.Next() {
		line, err := scanLine(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, line)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PostgresStore) ReadForSupplier(id string) ([]TradeLine, error) {
	return s.readList(`supplier_organization_id=$1::uuid`, id)
}

func (s *PostgresStore) ReadForBuyer(id string) ([]TradeLine, error) {
	return s.readList(`buyer_user_id=$1::uuid`, id)
}

func feeTermsJSON(f *ledger.FeeTerms) []byte { b, _ := json.Marshal(f); return b }
