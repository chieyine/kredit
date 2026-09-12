package settlement

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

type ReceiptStore struct{ pool *pgxpool.Pool }

func NewReceiptStore(pool *pgxpool.Pool) *ReceiptStore { return &ReceiptStore{pool: pool} }

type Payout struct {
	AttemptID      string         `json:"attempt_id"`
	ObligationID   string         `json:"obligation_id"`
	OrganizationID string         `json:"organization_id"`
	Route          map[string]any `json:"route"`
	PaymentID      string         `json:"payment_id"`
	PaymentState   string         `json:"payment_state"`
	Collected      ledger.Money   `json:"collected_kobo"`
	Paid           ledger.Money   `json:"paid_kobo"`
	Returned       ledger.Money   `json:"returned_kobo"`
	Outstanding    ledger.Money   `json:"outstanding_kobo"`
	Receipts       []BankReceipt  `json:"receipts"`
}
type BankReceipt struct {
	ID         string       `json:"id"`
	Reference  string       `json:"reference"`
	Amount     ledger.Money `json:"amount_kobo"`
	Direction  string       `json:"direction"`
	OccurredAt time.Time    `json:"occurred_at"`
	Evidence   string       `json:"evidence"`
}

func (s *ReceiptStore) begin(ctx context.Context, actor, org string) (pgx.Tx, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, actor, org); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}
func (s *ReceiptStore) List(ctx context.Context, actor, org string) ([]Payout, error) {
	tx, err := s.begin(ctx, actor, org)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if org == "" {
		if err = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionPlatformOwner); err != nil {
			return nil, err
		}
	}
	rows, err := tx.Query(ctx, `SELECT attempt_id::text,supplier_organization_id::text FROM app.collection_settlement_routes WHERE ($1='' OR supplier_organization_id=NULLIF($1,'')::uuid) ORDER BY created_at DESC,attempt_id DESC`, org)
	if err != nil {
		return nil, err
	}
	var refs [][2]string
	for rows.Next() {
		var pair [2]string
		if err = rows.Scan(&pair[0], &pair[1]); err != nil {
			rows.Close()
			return nil, err
		}
		refs = append(refs, pair)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	items := []Payout{}
	for _, ref := range refs {
		readTx, err := s.begin(ctx, actor, ref[1])
		if err != nil {
			return nil, err
		}
		v, err := readPayout(ctx, readTx, ref[0], ref[1])
		_ = readTx.Rollback(ctx)
		if err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, nil
}
func readPayout(ctx context.Context, tx pgx.Tx, attempt, org string) (Payout, error) {
	v := Payout{AttemptID: attempt, OrganizationID: org, Receipts: []BankReceipt{}}
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT r.obligation_id::text,r.route,COALESCE(p.id::text,''),COALESCE(p.state,''),COALESCE(p.amount_kobo,0) FROM app.collection_settlement_routes r LEFT JOIN app.payments p ON p.idempotency_key='collection-attempt:'||r.attempt_id::text AND p.obligation_id=r.obligation_id AND p.supplier_organization_id=r.supplier_organization_id WHERE r.attempt_id=$1::uuid AND r.supplier_organization_id=$2::uuid`, attempt, org).Scan(&v.ObligationID, &raw, &v.PaymentID, &v.PaymentState, &v.Collected)
	if err != nil {
		return v, err
	}
	if err = json.Unmarshal(raw, &v.Route); err != nil {
		return v, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text,bank_reference,amount_kobo,direction,occurred_at,evidence FROM app.seller_settlement_receipts WHERE attempt_id=$1::uuid AND supplier_organization_id=$2::uuid ORDER BY created_at,id`, attempt, org)
	if err != nil {
		return v, err
	}
	defer rows.Close()
	for rows.Next() {
		var receipt BankReceipt
		if err = rows.Scan(&receipt.ID, &receipt.Reference, &receipt.Amount, &receipt.Direction, &receipt.OccurredAt, &receipt.Evidence); err != nil {
			return v, err
		}
		v.Receipts = append(v.Receipts, receipt)
		if receipt.Direction == "paid" {
			v.Paid += receipt.Amount
		} else {
			v.Returned += receipt.Amount
		}
	}
	var frozen struct {
		Fee ledger.Money `json:"fee_amount_kobo"`
	}
	if err = json.Unmarshal(raw, &frozen); err != nil {
		return v, err
	}
	if v.PaymentState == "recognized" {
		var allocated ledger.Money
		if err = tx.QueryRow(ctx, `SELECT COALESCE(sum(amount_kobo),0) FROM app.split_fee_allocations WHERE payment_id=$1::uuid`, v.PaymentID).Scan(&allocated); err != nil {
			return v, err
		}
		v.Outstanding = max(0, v.Collected-allocated-v.Paid+v.Returned)
	}
	return v, rows.Err()
}
func (s *ReceiptStore) Record(ctx context.Context, actor, org, attempt string, in BankReceipt) (Payout, error) {
	in.OccurredAt = in.OccurredAt.UTC().Truncate(time.Microsecond)
	in.Reference = strings.TrimSpace(in.Reference)
	in.Evidence = strings.TrimSpace(in.Evidence)
	if in.Reference == "" || len(in.Reference) > 160 || in.Amount <= 0 || len(in.Evidence) < 20 || len(in.Evidence) > 2000 || in.OccurredAt.IsZero() || in.OccurredAt.After(time.Now()) || (in.Direction != "paid" && in.Direction != "returned") {
		return Payout{}, errors.New("record the completed bank transfer reference, amount, date, direction and evidence")
	}
	tx, err := s.begin(ctx, actor, org)
	if err != nil {
		return Payout{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionPlatformOwner); err != nil {
		return Payout{}, err
	}
	var obligation string
	if err = tx.QueryRow(ctx, `SELECT obligation_id::text FROM app.collection_settlement_routes WHERE attempt_id=$1::uuid AND supplier_organization_id=$2::uuid`, attempt, org).Scan(&obligation); err != nil {
		return Payout{}, err
	}
	if _, err = tx.Exec(ctx, `SELECT id FROM app.obligations WHERE id=$1::uuid FOR UPDATE`, obligation); err != nil {
		return Payout{}, err
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('seller-settlement:'||$1,0))`, attempt); err != nil {
		return Payout{}, err
	}
	v, err := readPayout(ctx, tx, attempt, org)
	if err != nil {
		return v, err
	}
	for _, old := range v.Receipts {
		if old.Reference == in.Reference {
			if old.Amount != in.Amount || old.Direction != in.Direction || !old.OccurredAt.Equal(in.OccurredAt) || old.Evidence != in.Evidence {
				return v, errors.New("that bank reference already has different evidence")
			}
			return v, tx.Commit(ctx)
		}
	}
	if v.PaymentID == "" {
		return v, errors.New("a recognized customer payment is required before recording seller settlement")
	}
	if (in.Direction == "paid" && (v.PaymentState != "recognized" || in.Amount > v.Outstanding)) || (in.Direction == "returned" && in.Amount > v.Paid-v.Returned) {
		return v, errors.New("the transfer exceeds the unsettled or returnable amount")
	}
	var paidAt time.Time
	if err = tx.QueryRow(ctx, `SELECT paid_at FROM app.payments WHERE id=$1::uuid`, v.PaymentID).Scan(&paidAt); err != nil {
		return v, err
	}
	if in.OccurredAt.Before(paidAt) {
		return v, errors.New("seller settlement cannot predate the customer payment")
	}
	err = tx.QueryRow(ctx, `INSERT INTO app.seller_settlement_receipts(payment_id,attempt_id,supplier_organization_id,bank_reference,amount_kobo,direction,occurred_at,recorded_by,evidence) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$8::uuid,$9) RETURNING id::text`, v.PaymentID, attempt, org, in.Reference, in.Amount, in.Direction, in.OccurredAt, actor, in.Evidence).Scan(&in.ID)
	if err != nil {
		return v, err
	}
	if _, err = ledger.NewPostgresStore(s.pool).PostSellerSettlementTx(ctx, tx, in.ID, in.Amount, in.Direction, in.OccurredAt); err != nil {
		return v, err
	}
	meta, _ := json.Marshal(map[string]any{"attempt_id": attempt, "payment_id": v.PaymentID, "amount_kobo": in.Amount, "direction": in.Direction})
	if _, err = tx.Exec(ctx, `INSERT INTO app.audit_events(actor_user_id,organization_id,action,resource_type,resource_id,outcome,severity,metadata) VALUES($1::uuid,$2::uuid,'seller.settlement.recorded','seller_settlement_receipt',$3,'success','high',$4::jsonb)`, actor, org, in.ID, meta); err != nil {
		return v, err
	}
	notice := notifications.Event{ID: "seller-settlement:" + in.ID, Type: "SellerSettlementRecorded", OrganizationID: org, Priority: notifications.PriorityCritical, Reference: attempt, NextAction: "Open your bank settings to review the recorded seller payout or return.", SecurePath: "/app/settings/settlement"}
	payload, err := json.Marshal(map[string]any{"notification": notice})
	if err != nil {
		return v, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key) VALUES('seller_settlement_receipt',$1,'notification.requested',$2::jsonb,$3)`, in.ID, payload, notice.ID); err != nil {
		return v, err
	}
	updated, err := readPayout(ctx, tx, attempt, org)
	if err != nil {
		return v, err
	}
	return updated, tx.Commit(ctx)
}
