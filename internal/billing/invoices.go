// Package billing records platform fees separately from buyer repayments.
package billing

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kredit/internal/access"
	"kredit/internal/ledger"
	"kredit/internal/notifications"
)

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

type Line struct {
	Deducted     int64  `json:"deducted_kobo"`
	FeeID        string `json:"fee_id"`
	ObligationID string `json:"obligation_id"`
	Type         string `json:"type"`
	Amount       int64  `json:"amount_kobo"`
	Credit       int64  `json:"credit_kobo"`
}
type Invoice struct {
	Deducted            int64     `json:"deducted_kobo"`
	ID                  string    `json:"id"`
	OrganizationID      string    `json:"organization_id"`
	BusinessName        string    `json:"business_name"`
	BusinessAddress     string    `json:"business_address"`
	PaymentInstructions string    `json:"payment_instructions"`
	IssuedAt            time.Time `json:"issued_at"`
	DueAt               time.Time `json:"due_at"`
	PeriodEnd           time.Time `json:"period_end"`
	Currency            string    `json:"currency"`
	Total               int64     `json:"total_kobo"`
	Credit              int64     `json:"credit_kobo"`
	ProviderReceived    int64     `json:"provider_received_kobo"`
	Received            int64     `json:"received_kobo"`
	Refunded            int64     `json:"refunded_kobo"`
	Outstanding         int64     `json:"outstanding_kobo"`
	CreditBalance       int64     `json:"credit_balance_kobo"`
	Lines               []Line    `json:"lines"`
}

func (s *Store) begin(ctx context.Context, org, actor string) (pgx.Tx, error) {
	if s == nil || s.pool == nil || org == "" {
		return nil, errors.New("billing database is unavailable")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_organization_id',$1,true),set_config('app.current_user_id',$2,true)`, org, actor); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}

// PeriodEnd is the start of the current calendar period in Lagos. Only fees
// before this boundary are billed; restarts and missed runs include older fees.
func PeriodEnd(now time.Time, cycle string) (time.Time, error) {
	loc := time.FixedZone("Africa/Lagos", 3600)
	local := now.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	switch cycle {
	case "monthly":
		start = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
	case "weekly":
		start = start.AddDate(0, 0, -(int(start.Weekday())+6)%7)
	default:
		return time.Time{}, errors.New("unsupported invoice cycle")
	}
	return start.UTC(), nil
}

func (s *Store) IssueDue(ctx context.Context, now time.Time) error {
	rows, err := s.pool.Query(ctx, `SELECT organization_id::text FROM app.invoice_billing_work()`)
	if err != nil {
		return err
	}
	orgs := []string{}
	for rows.Next() {
		var org string
		if err = rows.Scan(&org); err != nil {
			break
		}
		orgs = append(orgs, org)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return err
	}
	for _, org := range orgs {
		if err = s.Issue(ctx, org, now); err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) Issue(ctx context.Context, org string, now time.Time) error {
	tx, err := s.begin(ctx, org, "")
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var ref, cycle, instructions, name, address string
	err = tx.QueryRow(ctx, `SELECT p.billing_provider_reference,p.billing_cycle,a.payment_instructions,o.legal_name,o.business_address FROM app.supplier_onboarding_profiles p JOIN app.invoice_billing_approvals a ON a.organization_id=p.organization_id AND a.billing_reference=p.billing_provider_reference JOIN app.organizations o ON o.id=p.organization_id WHERE p.organization_id=$1::uuid AND p.billing_state='configured' AND p.billing_method IN ('consolidated_invoice','authorized_debit','split_settlement') FOR UPDATE OF p`, org).Scan(&ref, &cycle, &instructions, &name, &address)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if cycle == "per_settlement" {
		cycle = "monthly"
	}
	end, err := PeriodEnd(now, cycle)
	if err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.fee_invoices WHERE organization_id=$1::uuid AND period_end=$2)`, org, end).Scan(&exists); err != nil || exists {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT f.id::text,f.amount_kobo-f.waived_kobo-f.collected_kobo,f.waived_kobo,f.collected_kobo FROM app.fees f WHERE f.supplier_organization_id=$1::uuid AND f.state='accrued' AND f.amount_kobo>f.waived_kobo+f.collected_kobo AND f.accrued_at<$2 AND NOT EXISTS(SELECT 1 FROM app.fee_invoice_lines l WHERE l.fee_id=f.id) AND NOT EXISTS(SELECT 1 FROM app.collection_settlement_routes r JOIN app.collection_attempts a ON a.id=r.attempt_id WHERE r.obligation_id=f.obligation_id AND a.state IN ('PENDING','SUBMITTED','UNKNOWN') AND r.route->>'billing_method'='split_settlement') ORDER BY f.id FOR UPDATE OF f`, org, end)
	if err != nil {
		return err
	}
	type fee struct {
		id                        string
		amount, waived, collected int64
	}
	fees := []fee{}
	for rows.Next() {
		var f fee
		if err = rows.Scan(&f.id, &f.amount, &f.waived, &f.collected); err != nil {
			break
		}
		fees = append(fees, f)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return err
	}
	if len(fees) == 0 {
		return nil
	}
	var id string
	if err = tx.QueryRow(ctx, `INSERT INTO app.fee_invoices(organization_id,billing_reference,cycle,period_end,issued_at,due_at,business_name,business_address,payment_instructions) VALUES($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id::text`, org, ref, cycle, end, now, now.AddDate(0, 0, 7), name, address, instructions).Scan(&id); err != nil {
		return err
	}
	for _, f := range fees {
		if _, err = tx.Exec(ctx, `INSERT INTO app.fee_invoice_lines(invoice_id,fee_id,organization_id,amount_kobo,waived_at_issue_kobo,collected_at_issue_kobo) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6)`, id, f.id, org, f.amount, f.waived, f.collected); err != nil {
			return err
		}
	}
	if err = notify(ctx, tx, org, id, "FeeInvoiceIssued"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Store) List(ctx context.Context, org, actor string) ([]Invoice, error) {
	tx, err := s.begin(ctx, org, actor)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `SELECT id::text FROM app.fee_invoices WHERE organization_id=$1::uuid ORDER BY issued_at DESC,id DESC LIMIT 100`, org)
	if err != nil {
		return nil, err
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
	if err != nil {
		return nil, err
	}
	invoices := []Invoice{}
	for _, id := range ids {
		v, e := read(ctx, tx, org, id)
		if e != nil {
			return nil, e
		}
		invoices = append(invoices, v)
	}
	return invoices, tx.Commit(ctx)
}
func read(ctx context.Context, tx pgx.Tx, org, id string) (Invoice, error) {
	v := Invoice{Currency: "NGN", Lines: []Line{}}
	err := tx.QueryRow(ctx, `SELECT id::text,organization_id::text,business_name,business_address,payment_instructions,issued_at,due_at,period_end FROM app.fee_invoices WHERE id=$1::uuid AND organization_id=$2::uuid`, id, org).Scan(&v.ID, &v.OrganizationID, &v.BusinessName, &v.BusinessAddress, &v.PaymentInstructions, &v.IssuedAt, &v.DueAt, &v.PeriodEnd)
	if err != nil {
		return v, err
	}
	rows, err := tx.Query(ctx, `SELECT f.id::text,f.obligation_id::text,f.fee_type,l.amount_kobo+l.collected_at_issue_kobo,CASE WHEN f.state IN ('waived','refunded') THEN l.amount_kobo+l.collected_at_issue_kobo ELSE LEAST(l.amount_kobo+l.collected_at_issue_kobo,GREATEST(0,f.waived_kobo-l.waived_at_issue_kobo)) END,CASE WHEN f.state='refunded' THEN 0 ELSE LEAST(l.collected_at_issue_kobo,f.collected_kobo) END FROM app.fee_invoice_lines l JOIN app.fees f ON f.id=l.fee_id WHERE l.invoice_id=$1::uuid ORDER BY f.accrued_at,f.id`, id)
	if err != nil {
		return v, err
	}
	for rows.Next() {
		var line Line
		if err = rows.Scan(&line.FeeID, &line.ObligationID, &line.Type, &line.Amount, &line.Credit, &line.Deducted); err != nil {
			break
		}
		v.Lines = append(v.Lines, line)
		v.Deducted += line.Deducted
		a, e := ledger.CheckedAdd(ledger.Money(v.Total), ledger.Money(line.Amount))
		if e != nil {
			err = e
			break
		}
		v.Total = int64(a)
		v.Credit += line.Credit
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return v, err
	}
	if err = tx.QueryRow(ctx, `SELECT COALESCE(sum(amount_kobo) FILTER(WHERE direction='received'),0),COALESCE(sum(amount_kobo) FILTER(WHERE direction='refunded'),0) FROM app.fee_invoice_receipts WHERE invoice_id=$1::uuid`, id).Scan(&v.Received, &v.Refunded); err != nil {
		return v, err
	}
	if err = tx.QueryRow(ctx, `SELECT COALESCE(sum(amount_kobo),0) FROM app.fee_debits WHERE invoice_id=$1::uuid AND state='succeeded'`, id).Scan(&v.ProviderReceived); err != nil {
		return v, err
	}
	balance := v.Total - v.Credit - v.Deducted - v.Received - v.ProviderReceived + v.Refunded
	v.Outstanding = max(0, balance)
	v.CreditBalance = max(0, -balance)
	return v, nil
}

type Receipt struct {
	Direction  string    `json:"direction"`
	Reference  string    `json:"bank_reference"`
	Amount     int64     `json:"amount_kobo"`
	ReceivedAt time.Time `json:"received_at"`
	Evidence   string    `json:"evidence"`
}

// RecordReceipt requires the platform owner to attest to an actual credit in
// Kredit's bank statement. Seller claims and messaging callbacks never call it.
func (s *Store) RecordReceipt(ctx context.Context, org, actor, id string, in Receipt) (Invoice, error) {
	if in.Direction == "" {
		in.Direction = "received"
	}
	if in.Direction != "received" && in.Direction != "refunded" {
		return Invoice{}, errors.New("invalid bank movement direction")
	}
	in.Reference = strings.TrimSpace(in.Reference)
	in.Evidence = strings.TrimSpace(in.Evidence)
	if len(in.Reference) < 3 || len(in.Reference) > 200 || len(in.Evidence) < 20 || len(in.Evidence) > 2000 || in.Amount <= 0 || in.ReceivedAt.IsZero() || in.ReceivedAt.After(time.Now().Add(time.Minute)) {
		return Invoice{}, errors.New("provide the received amount, bank reference, date and evidence")
	}
	tx, err := s.begin(ctx, org, actor)
	if err != nil {
		return Invoice{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionPlatformSettings); err != nil {
		return Invoice{}, err
	}
	// Match the obligation-first lock order used by reversals and fee waivers.
	rows, err := tx.Query(ctx, `SELECT o.id FROM app.obligations o WHERE o.id IN (SELECT f.obligation_id FROM app.fees f JOIN app.fee_invoice_lines l ON l.fee_id=f.id WHERE l.invoice_id=$1::uuid AND l.organization_id=$2::uuid) ORDER BY o.id FOR UPDATE`, id, org)
	if err != nil {
		return Invoice{}, err
	}
	for rows.Next() {
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return Invoice{}, err
	}
	// Immutable invoice rows have no UPDATE grant. A transaction lock serializes receipts.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "fee-invoice:"+id); err != nil {
		return Invoice{}, err
	}
	var locked string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM app.fee_invoices WHERE id=$1::uuid AND organization_id=$2::uuid`, id, org).Scan(&locked); err != nil {
		return Invoice{}, err
	}
	var pending bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.fee_debits WHERE invoice_id=$1::uuid AND (state='pending' OR review_required))`, id).Scan(&pending); err != nil {
		return Invoice{}, err
	}
	if pending {
		return Invoice{}, errors.New("reconcile the pending fee debit before recording another receipt")
	}
	var oldInvoice, oldEvidence, oldDirection string
	var oldAmount int64
	var oldDate time.Time
	err = tx.QueryRow(ctx, `SELECT invoice_id::text,amount_kobo,received_at,evidence,direction FROM app.fee_invoice_receipts WHERE bank_reference=$1`, in.Reference).Scan(&oldInvoice, &oldAmount, &oldDate, &oldEvidence, &oldDirection)
	if err == nil {
		if oldInvoice != id || oldAmount != in.Amount || !oldDate.Equal(in.ReceivedAt) || oldEvidence != in.Evidence || oldDirection != in.Direction {
			return Invoice{}, errors.New("bank reference already used for another receipt")
		}
		v, e := read(ctx, tx, org, id)
		if e != nil {
			return v, e
		}
		return v, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, err
	}
	v, err := read(ctx, tx, org, id)
	if err != nil {
		return v, err
	}
	limit := v.Outstanding
	if in.Direction == "refunded" {
		limit = v.CreditBalance
	}
	if in.Amount > limit || in.ReceivedAt.Before(v.IssuedAt) {
		return v, errors.New("receipt exceeds the bill balance or predates the bill")
	}
	var receiptID string
	if err = tx.QueryRow(ctx, `INSERT INTO app.fee_invoice_receipts(invoice_id,organization_id,bank_reference,amount_kobo,received_at,recorded_by,evidence,direction) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6::uuid,$7,$8) RETURNING id::text`, id, org, in.Reference, in.Amount, in.ReceivedAt, actor, in.Evidence, in.Direction).Scan(&receiptID); err != nil {
		return v, err
	}
	journal := ledger.NewPostgresStore(s.pool)
	if in.Direction == "refunded" {
		_, err = journal.PostFeeRefundTx(ctx, tx, receiptID, ledger.Money(in.Amount), in.ReceivedAt)
	} else {
		_, err = journal.PostFeeReceiptTx(ctx, tx, receiptID, ledger.Money(in.Amount), in.ReceivedAt)
	}
	if err != nil {
		return v, err
	}
	if in.Direction == "received" && in.Amount == v.Outstanding {
		if _, err = tx.Exec(ctx, `UPDATE app.fees SET state='paid',paid_at=$2 WHERE id IN (SELECT fee_id FROM app.fee_invoice_lines WHERE invoice_id=$1::uuid) AND state='accrued'`, id, in.ReceivedAt); err != nil {
			return v, err
		}
	}
	meta, _ := json.Marshal(map[string]any{"invoice_id": id, "amount_kobo": in.Amount, "evidence": in.Evidence, "bank_reference": in.Reference, "direction": in.Direction})
	if _, err = tx.Exec(ctx, `INSERT INTO app.audit_events(actor_user_id,organization_id,action,resource_type,resource_id,outcome,severity,metadata) VALUES($1::uuid,$2::uuid,'billing.receipt.recorded','fee_invoice_receipt',$3,'success','high',$4::jsonb)`, actor, org, receiptID, meta); err != nil {
		return v, err
	}
	kind := "FeeInvoicePaymentRecorded"
	if in.Direction == "refunded" {
		kind = "FeeInvoiceRefundRecorded"
	}
	if err = notify(ctx, tx, org, receiptID, kind); err != nil {
		return v, err
	}
	v, err = read(ctx, tx, org, id)
	if err != nil {
		return v, err
	}
	return v, tx.Commit(ctx)
}
func notify(ctx context.Context, tx pgx.Tx, org, id, kind string) error {
	event := notifications.Event{ID: "billing:" + id, Type: kind, OrganizationID: org, Priority: notifications.PriorityCritical, Reference: id, NextAction: "Open Kredit fees to see your bill and payment record.", SecurePath: "/app/settings/billing"}
	raw, err := json.Marshal(map[string]any{"notification": event})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key) VALUES('fee_invoice',$1,'notification.requested',$2::jsonb,$3) ON CONFLICT(idempotency_key) DO NOTHING`, id, raw, event.ID)
	return err
}
