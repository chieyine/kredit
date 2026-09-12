package billing

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"kredit/internal/access"
	"kredit/internal/ledger"
)

type FeeBank struct {
	Provider    string `json:"provider"`
	Allocated   int64  `json:"allocated_kobo"`
	Received    int64  `json:"received_kobo"`
	Returned    int64  `json:"returned_kobo"`
	Outstanding int64  `json:"outstanding_kobo"`
}
type DebitRecord struct {
	ReviewRequired bool      `json:"review_required"`
	ID             string    `json:"id"`
	Invoice        string    `json:"invoice_id"`
	Provider       string    `json:"provider"`
	Amount         int64     `json:"amount_kobo"`
	State          string    `json:"state"`
	CreatedAt      time.Time `json:"created_at"`
}

func feeBank(ctx context.Context, tx pgx.Tx, org, provider string) (FeeBank, error) {
	v := FeeBank{Provider: provider}
	err := tx.QueryRow(ctx, `SELECT COALESCE((SELECT sum(x.amount_kobo) FROM app.split_fee_allocations x JOIN app.payments p ON p.id=x.payment_id WHERE x.supplier_organization_id=$1::uuid AND p.provider=$2 AND p.state='recognized'),0)+COALESCE((SELECT sum(d.amount_kobo) FROM app.fee_debits d JOIN app.fee_authorizations a ON a.id=d.authorization_id WHERE d.organization_id=$1::uuid AND a.provider=$2 AND d.state='succeeded'),0),COALESCE(sum(amount_kobo) FILTER(WHERE direction='received'),0),COALESCE(sum(amount_kobo) FILTER(WHERE direction='returned'),0) FROM app.fee_bank_receipts WHERE organization_id=$1::uuid AND provider=$2`, org, provider).Scan(&v.Allocated, &v.Received, &v.Returned)
	v.Outstanding = v.Allocated - v.Received + v.Returned
	return v, err
}
func (s *FeeService) Operations(ctx context.Context, org, actor string) ([]FeeBank, []DebitRecord, error) {
	tx, err := s.store.begin(ctx, org, actor)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `SELECT d.id::text,d.invoice_id::text,a.provider,d.amount_kobo,d.state,d.created_at,d.review_required FROM app.fee_debits d JOIN app.fee_authorizations a ON a.id=d.authorization_id WHERE d.organization_id=$1::uuid ORDER BY d.created_at DESC`, org)
	if err != nil {
		return nil, nil, err
	}
	debits := []DebitRecord{}
	for rows.Next() {
		var d DebitRecord
		if err = rows.Scan(&d.ID, &d.Invoice, &d.Provider, &d.Amount, &d.State, &d.CreatedAt, &d.ReviewRequired); err != nil {
			break
		}
		debits = append(debits, d)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return nil, nil, err
	}
	rows, err = tx.Query(ctx, `SELECT provider FROM app.fee_authorizations WHERE organization_id=$1::uuid UNION SELECT p.provider FROM app.split_fee_allocations x JOIN app.payments p ON p.id=x.payment_id WHERE x.supplier_organization_id=$1::uuid UNION SELECT provider FROM app.fee_bank_receipts WHERE organization_id=$1::uuid`, org)
	if err != nil {
		return nil, nil, err
	}
	providers := []string{}
	for rows.Next() {
		var p string
		if err = rows.Scan(&p); err != nil {
			break
		}
		providers = append(providers, p)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return nil, nil, err
	}
	banks := []FeeBank{}
	for _, p := range providers {
		v, e := feeBank(ctx, tx, org, p)
		if e != nil {
			return nil, nil, e
		}
		banks = append(banks, v)
	}
	return banks, debits, nil
}
func (s *FeeService) BankReceipt(ctx context.Context, org, actor, provider string, in Receipt) error {
	in.Reference = strings.TrimSpace(in.Reference)
	in.ReceivedAt = in.ReceivedAt.UTC().Truncate(time.Microsecond)
	if in.Reference == "" || len(in.Reference) > 200 || in.Amount <= 0 || len(strings.TrimSpace(in.Evidence)) < 20 || len(in.Evidence) > 2000 || in.ReceivedAt.IsZero() || in.ReceivedAt.After(time.Now()) || (in.Direction != "received" && in.Direction != "returned") {
		return errors.New("enter the completed bank movement and evidence")
	}
	tx, err := s.store.begin(ctx, org, actor)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionPlatformOwner); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,92841))`, org); err != nil {
		return err
	}
	var previousProvider, previousEvidence, previousDirection string
	var previousAmount int64
	var previousDate time.Time
	err = tx.QueryRow(ctx, `SELECT provider,evidence,direction,amount_kobo,occurred_at FROM app.fee_bank_receipts WHERE organization_id=$1::uuid AND bank_reference=$2`, org, in.Reference).Scan(&previousProvider, &previousEvidence, &previousDirection, &previousAmount, &previousDate)
	if err == nil {
		if previousProvider == provider && previousEvidence == in.Evidence && previousDirection == in.Direction && previousAmount == in.Amount && previousDate.Equal(in.ReceivedAt) {
			return tx.Commit(ctx)
		}
		return errors.New("bank reference already has different evidence")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	var earliest *time.Time
	if err = tx.QueryRow(ctx, `SELECT min(at) FROM (SELECT p.paid_at AS at FROM app.split_fee_allocations x JOIN app.payments p ON p.id=x.payment_id WHERE x.supplier_organization_id=$1::uuid AND p.provider=$2 UNION ALL SELECT d.created_at FROM app.fee_debits d JOIN app.fee_authorizations a ON a.id=d.authorization_id WHERE d.organization_id=$1::uuid AND a.provider=$2) events`, org, provider).Scan(&earliest); err != nil {
		return err
	}
	if earliest == nil || in.ReceivedAt.Before(*earliest) {
		return errors.New("bank evidence cannot predate the original fee collection")
	}
	v, err := feeBank(ctx, tx, org, provider)
	if err != nil {
		return err
	}
	limit := v.Outstanding
	if in.Direction == "returned" {
		limit = v.Received - v.Returned
	}
	if in.Amount > limit {
		return errors.New("bank amount exceeds the outstanding or returnable fee balance")
	}
	var id string
	if err = tx.QueryRow(ctx, `INSERT INTO app.fee_bank_receipts(organization_id,provider,bank_reference,amount_kobo,direction,occurred_at,recorded_by,evidence) VALUES($1::uuid,$2,$3,$4,$5,$6,$7::uuid,$8) RETURNING id::text`, org, provider, in.Reference, in.Amount, in.Direction, in.ReceivedAt, actor, in.Evidence).Scan(&id); err != nil {
		return err
	}
	if _, err = ledger.NewPostgresStore(nil).PostFeeBankTx(ctx, tx, id, ledger.Money(in.Amount), in.Direction == "returned", in.ReceivedAt); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.audit_events(actor_user_id,organization_id,action,resource_type,resource_id,outcome,severity,metadata) VALUES($1::uuid,$2::uuid,'billing.bank.recorded','fee_bank_receipt',$3,'success','high',jsonb_build_object('provider',$4::text,'amount_kobo',$5::bigint,'direction',$6::text))`, actor, org, id, provider, in.Amount, in.Direction); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *FeeService) ReviewDebit(ctx context.Context, org, actor, id, action, evidence string) error {
	tx, err := s.store.begin(ctx, org, actor)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionPlatformOwner); err != nil {
		return err
	}
	if action == "reconcile" {
		if err = tx.Commit(ctx); err != nil {
			return err
		}
		return s.reconcile(ctx, org, id)
	}
	if action == "clear_review" {
		if len(strings.TrimSpace(evidence)) < 20 || len(evidence) > 2000 {
			return errors.New("record the bank evidence reviewed")
		}
		var provider, mandate, state string
		var amount int64
		if err = tx.QueryRow(ctx, `SELECT a.provider,a.mandate_reference,d.state,d.amount_kobo FROM app.fee_debits d JOIN app.fee_authorizations a ON a.id=d.authorization_id WHERE d.id=$1::uuid AND d.organization_id=$2::uuid FOR UPDATE OF d`, id, org).Scan(&provider, &mandate, &state, &amount); err != nil {
			return err
		}
		p := s.Providers[provider]
		if p == nil {
			return errors.New("original provider account is required")
		}
		result, err := p.ReadFeeDebit(ctx, FeeDebitRequest{mandate, "fee-debit-" + id, amount})
		if err != nil {
			return err
		}
		if (state != "succeeded" && state != "failed") || result.State != state || (state == "succeeded" && result.Amount != amount) {
			return errors.New("provider evidence still conflicts with the recorded fee debit")
		}
		if _, err = tx.Exec(ctx, `UPDATE app.fee_debits SET review_required=false,checked_at=now() WHERE id=$1::uuid`, id); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO app.audit_events(actor_user_id,organization_id,action,resource_type,resource_id,outcome,severity,metadata) VALUES($1::uuid,$2::uuid,'billing.debit.reviewed','fee_debit',$3,'success','high',jsonb_build_object('evidence',$4::text))`, actor, org, id, evidence); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if action != "reversed" || len(strings.TrimSpace(evidence)) < 20 || len(evidence) > 2000 {
		return errors.New("record the completed bank reversal evidence")
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,92841))`, org); err != nil {
		return err
	}
	var amount int64
	var state string
	if err = tx.QueryRow(ctx, `SELECT amount_kobo,state FROM app.fee_debits WHERE id=$1::uuid AND organization_id=$2::uuid FOR UPDATE`, id, org).Scan(&amount, &state); err != nil {
		return err
	}
	if state == "reversed" {
		return nil
	}
	if state != "succeeded" {
		return errors.New("only a previously confirmed fee debit can be reversed")
	}
	if _, err = tx.Exec(ctx, `UPDATE app.fee_debits SET state='reversed',review_required=false,checked_at=now() WHERE id=$1::uuid`, id); err != nil {
		return err
	}
	if _, err = ledger.NewPostgresStore(nil).PostFeeDebitReversalTx(ctx, tx, id, ledger.Money(amount), time.Now()); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.audit_events(actor_user_id,organization_id,action,resource_type,resource_id,outcome,severity,metadata) VALUES($1::uuid,$2::uuid,'billing.debit.reversed','fee_debit',$3,'success','critical',jsonb_build_object('evidence',$4::text,'amount_kobo',$5::bigint))`, actor, org, id, evidence, amount); err != nil {
		return err
	}
	if err = notify(ctx, tx, org, id+":reversed", "FeeDebitReversed"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
