package orders

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"kredit/internal/db"
	"kredit/internal/identifier"
	"kredit/internal/ledger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound       = errors.New("record not found")
	ErrInvalidInput   = errors.New("invalid input")
	ErrDualControl    = errors.New("maker-checker dual control required: issuer cannot approve own credit note")
	ErrAlreadyDecided = errors.New("credit note is already approved or closed")
)

type LineItem struct {
	ID                string       `json:"id"`
	OrderID           string       `json:"order_id"`
	SKU               string       `json:"sku"`
	Description       string       `json:"description"`
	UnitPriceKobo     ledger.Money `json:"unit_price_kobo"`
	Quantity          int64        `json:"quantity"`
	FulfilledQuantity int64        `json:"fulfilled_quantity"`
	ReturnedQuantity  int64        `json:"returned_quantity"`
	TotalKobo         ledger.Money `json:"total_kobo"`
	CreatedAt         time.Time    `json:"created_at"`
}

type Shipment struct {
	ID                     string         `json:"id"`
	OrderID                string         `json:"order_id"`
	SupplierOrganizationID string         `json:"supplier_organization_id"`
	TrackingReference      string         `json:"tracking_reference"`
	Carrier                string         `json:"carrier"`
	DispatchedBy           string         `json:"dispatched_by"`
	DispatchedAt           time.Time      `json:"dispatched_at"`
	Status                 string         `json:"status"`
	Items                  []ShipmentItem `json:"items,omitempty"`
}

type ShipmentItem struct {
	ShipmentID string `json:"shipment_id"`
	LineItemID string `json:"line_item_id"`
	Quantity   int64  `json:"quantity"`
}

type DeliveryReceipt struct {
	ID              string    `json:"id"`
	ShipmentID      string    `json:"shipment_id"`
	OrderID         string    `json:"order_id"`
	ReceivedBy      string    `json:"received_by"`
	ReceivedAt      time.Time `json:"received_at"`
	ConditionNotes  string    `json:"condition_notes"`
	SignedProofHash string    `json:"signed_proof_hash"`
}

type CreditNote struct {
	ID                     string       `json:"id"`
	OrderID                string       `json:"order_id"`
	ObligationID           string       `json:"obligation_id,omitempty"`
	SupplierOrganizationID string       `json:"supplier_organization_id"`
	AmountKobo             ledger.Money `json:"amount_kobo"`
	Reason                 string       `json:"reason"`
	IssuedBy               string       `json:"issued_by"`
	ApprovedBy             string       `json:"approved_by,omitempty"`
	Status                 string       `json:"status"`
	CreatedAt              time.Time    `json:"created_at"`
	ApprovedAt             *time.Time   `json:"approved_at,omitempty"`
}

type DispatchInput struct {
	OrderID                string
	SupplierOrganizationID string
	TrackingReference      string
	Carrier                string
	DispatchedBy           string
	Items                  []ShipmentItem
}

type DeliveryInput struct {
	ShipmentID     string
	OrderID        string
	ReceivedBy     string
	ConditionNotes string
	SignedProof    string
}

type CreditNoteInput struct {
	OrderID                string
	ObligationID           string
	SupplierOrganizationID string
	AmountKobo             ledger.Money
	Reason                 string
	IssuedBy               string
}

type Service interface {
	CreateLineItems(ctx context.Context, orderID string, items []LineItem) error
	ListLineItems(ctx context.Context, orderID string) ([]LineItem, error)
	DispatchShipment(ctx context.Context, input DispatchInput) (Shipment, error)
	RecordDeliveryReceipt(ctx context.Context, input DeliveryInput) (DeliveryReceipt, error)
	CreateCreditNote(ctx context.Context, input CreditNoteInput) (CreditNote, error)
	ApproveCreditNote(ctx context.Context, noteID, reviewerID string) error
	ListShipments(ctx context.Context, orderID string) ([]Shipment, error)
	ListCreditNotes(ctx context.Context, orderID string) ([]CreditNote, error)
}

// In-memory Store for unit tests and local simulation
type MemoryStore struct {
	mu          sync.RWMutex
	lineItems   map[string][]LineItem
	shipments   map[string]*Shipment
	receipts    map[string]*DeliveryReceipt
	creditNotes map[string]*CreditNote
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		lineItems:   make(map[string][]LineItem),
		shipments:   make(map[string]*Shipment),
		receipts:    make(map[string]*DeliveryReceipt),
		creditNotes: make(map[string]*CreditNote),
	}
}

func (m *MemoryStore) CreateLineItems(ctx context.Context, orderID string, items []LineItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range items {
		if items[i].ID == "" {
			items[i].ID = identifier.New()
		}
		items[i].OrderID = orderID
		items[i].TotalKobo = items[i].UnitPriceKobo * ledger.Money(items[i].Quantity)
		items[i].CreatedAt = time.Now().UTC()
	}
	m.lineItems[orderID] = append(m.lineItems[orderID], items...)
	return nil
}

func (m *MemoryStore) ListLineItems(ctx context.Context, orderID string) ([]LineItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items, ok := m.lineItems[orderID]
	if !ok {
		return []LineItem{}, nil
	}
	out := make([]LineItem, len(items))
	copy(out, items)
	return out, nil
}

func (m *MemoryStore) DispatchShipment(ctx context.Context, input DispatchInput) (Shipment, error) {
	if input.OrderID == "" || input.SupplierOrganizationID == "" || input.DispatchedBy == "" || len(input.Items) == 0 {
		return Shipment{}, ErrInvalidInput
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	shipmentID := identifier.New()
	items := make([]ShipmentItem, len(input.Items))
	for i, it := range input.Items {
		items[i] = ShipmentItem{
			ShipmentID: shipmentID,
			LineItemID: it.LineItemID,
			Quantity:   it.Quantity,
		}
		// Update fulfilled quantity on line item
		for li := range m.lineItems[input.OrderID] {
			if m.lineItems[input.OrderID][li].ID == it.LineItemID {
				m.lineItems[input.OrderID][li].FulfilledQuantity += it.Quantity
			}
		}
	}
	s := &Shipment{
		ID:                     shipmentID,
		OrderID:                input.OrderID,
		SupplierOrganizationID: input.SupplierOrganizationID,
		TrackingReference:      input.TrackingReference,
		Carrier:                input.Carrier,
		DispatchedBy:           input.DispatchedBy,
		DispatchedAt:           time.Now().UTC(),
		Status:                 "in_transit",
		Items:                  items,
	}
	m.shipments[shipmentID] = s
	return *s, nil
}

func (m *MemoryStore) ListShipments(ctx context.Context, orderID string) ([]Shipment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Shipment
	for _, s := range m.shipments {
		if s.OrderID == orderID {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (m *MemoryStore) RecordDeliveryReceipt(ctx context.Context, input DeliveryInput) (DeliveryReceipt, error) {
	if input.ShipmentID == "" || input.OrderID == "" || input.ReceivedBy == "" {
		return DeliveryReceipt{}, ErrInvalidInput
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	ship, ok := m.shipments[input.ShipmentID]
	if !ok {
		return DeliveryReceipt{}, ErrNotFound
	}
	ship.Status = "delivered"
	hash := sha256.Sum256([]byte(input.SignedProof))
	rec := &DeliveryReceipt{
		ID:              identifier.New(),
		ShipmentID:      input.ShipmentID,
		OrderID:         input.OrderID,
		ReceivedBy:      input.ReceivedBy,
		ReceivedAt:      time.Now().UTC(),
		ConditionNotes:  input.ConditionNotes,
		SignedProofHash: hex.EncodeToString(hash[:]),
	}
	m.receipts[rec.ID] = rec
	return *rec, nil
}

func (m *MemoryStore) CreateCreditNote(ctx context.Context, input CreditNoteInput) (CreditNote, error) {
	if input.OrderID == "" || input.SupplierOrganizationID == "" || input.AmountKobo <= 0 || input.IssuedBy == "" || input.Reason == "" {
		return CreditNote{}, ErrInvalidInput
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cn := &CreditNote{
		ID:                     identifier.New(),
		OrderID:                input.OrderID,
		ObligationID:           input.ObligationID,
		SupplierOrganizationID: input.SupplierOrganizationID,
		AmountKobo:             input.AmountKobo,
		Reason:                 input.Reason,
		IssuedBy:               input.IssuedBy,
		Status:                 "draft",
		CreatedAt:              time.Now().UTC(),
	}
	m.creditNotes[cn.ID] = cn
	return *cn, nil
}

func (m *MemoryStore) ApproveCreditNote(ctx context.Context, noteID, reviewerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cn, ok := m.creditNotes[noteID]
	if !ok {
		return ErrNotFound
	}
	if cn.IssuedBy == reviewerID {
		return ErrDualControl
	}
	if cn.Status != "draft" {
		return ErrAlreadyDecided
	}
	now := time.Now().UTC()
	cn.Status = "approved"
	cn.ApprovedBy = reviewerID
	cn.ApprovedAt = &now
	return nil
}

func (m *MemoryStore) ListCreditNotes(ctx context.Context, orderID string) ([]CreditNote, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []CreditNote
	for _, cn := range m.creditNotes {
		if cn.OrderID == orderID {
			out = append(out, *cn)
		}
	}
	return out, nil
}

// PostgresStore implements Service with SQL and Row-Level Security
type PostgresStore struct {
	pool   *pgxpool.Pool
	ledger ledger.Service
}

func NewPostgresStore(pool *pgxpool.Pool, l ...ledger.Service) *PostgresStore {
	var led ledger.Service
	if len(l) > 0 {
		led = l[0]
	}
	return &PostgresStore{pool: pool, ledger: led}
}

func (p *PostgresStore) SetLedger(l ledger.Service) {
	p.ledger = l
}

func (p *PostgresStore) beginTx(ctx context.Context, explicitOrg, explicitUser string) (pgx.Tx, error) {
	if p.pool == nil {
		return nil, errors.New("database pool unavailable")
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	identity, _ := db.TenantFromContext(ctx)
	orgID := identity.OrganizationID
	if orgID == "" {
		orgID = explicitOrg
	}
	userID := identity.UserID
	if userID == "" {
		userID = explicitUser
	}
	if orgID != "" || userID != "" {
		if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, userID, orgID); err != nil {
			_ = tx.Rollback(ctx)
			return nil, err
		}
	}
	return tx, nil
}

func (p *PostgresStore) CreateLineItems(ctx context.Context, orderID string, items []LineItem) error {
	tx, err := p.beginTx(ctx, "", "")
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, it := range items {
		total := it.UnitPriceKobo * ledger.Money(it.Quantity)
		_, err := tx.Exec(ctx, `
			INSERT INTO app.order_line_items (order_id, sku, description, unit_price_kobo, quantity, fulfilled_quantity, returned_quantity, total_kobo)
			VALUES ($1::uuid, $2, $3, $4, $5, 0, 0, $6)
		`, orderID, it.SKU, it.Description, int64(it.UnitPriceKobo), it.Quantity, int64(total))
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (p *PostgresStore) ListLineItems(ctx context.Context, orderID string) ([]LineItem, error) {
	tx, err := p.beginTx(ctx, "", "")
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		SELECT id::text, order_id::text, sku, description, unit_price_kobo, quantity, fulfilled_quantity, returned_quantity, total_kobo, created_at
		FROM app.order_line_items
		WHERE order_id = $1::uuid
		ORDER BY created_at, id
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LineItem
	for rows.Next() {
		var it LineItem
		var up, tot int64
		if err := rows.Scan(&it.ID, &it.OrderID, &it.SKU, &it.Description, &up, &it.Quantity, &it.FulfilledQuantity, &it.ReturnedQuantity, &tot, &it.CreatedAt); err != nil {
			return nil, err
		}
		it.UnitPriceKobo = ledger.Money(up)
		it.TotalKobo = ledger.Money(tot)
		out = append(out, it)
	}
	return out, rows.Err()
}

func (p *PostgresStore) DispatchShipment(ctx context.Context, input DispatchInput) (Shipment, error) {
	tx, err := p.beginTx(ctx, input.SupplierOrganizationID, input.DispatchedBy)
	if err != nil {
		return Shipment{}, err
	}
	defer tx.Rollback(ctx)

	var s Shipment
	err = tx.QueryRow(ctx, `
		INSERT INTO app.order_shipments (order_id, supplier_organization_id, tracking_reference, carrier, dispatched_by, status)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5::uuid, 'in_transit')
		RETURNING id::text, order_id::text, supplier_organization_id::text, tracking_reference, carrier, dispatched_by::text, dispatched_at, status
	`, input.OrderID, input.SupplierOrganizationID, input.TrackingReference, input.Carrier, input.DispatchedBy).
		Scan(&s.ID, &s.OrderID, &s.SupplierOrganizationID, &s.TrackingReference, &s.Carrier, &s.DispatchedBy, &s.DispatchedAt, &s.Status)
	if err != nil {
		return Shipment{}, err
	}

	for _, item := range input.Items {
		_, err := tx.Exec(ctx, `
			INSERT INTO app.order_shipment_items (shipment_id, line_item_id, quantity)
			VALUES ($1::uuid, $2::uuid, $3)
		`, s.ID, item.LineItemID, item.Quantity)
		if err != nil {
			return Shipment{}, err
		}
		_, err = tx.Exec(ctx, `
			UPDATE app.order_line_items
			SET fulfilled_quantity = fulfilled_quantity + $2
			WHERE id = $1::uuid
		`, item.LineItemID, item.Quantity)
		if err != nil {
			return Shipment{}, err
		}
		s.Items = append(s.Items, ShipmentItem{
			ShipmentID: s.ID,
			LineItemID: item.LineItemID,
			Quantity:   item.Quantity,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return Shipment{}, err
	}
	return s, nil
}

func (p *PostgresStore) RecordDeliveryReceipt(ctx context.Context, input DeliveryInput) (DeliveryReceipt, error) {
	hash := sha256.Sum256([]byte(input.SignedProof))
	hashHex := hex.EncodeToString(hash[:])

	tx, err := p.beginTx(ctx, "", input.ReceivedBy)
	if err != nil {
		return DeliveryReceipt{}, err
	}
	defer tx.Rollback(ctx)

	var rec DeliveryReceipt
	err = tx.QueryRow(ctx, `
		INSERT INTO app.order_delivery_receipts (shipment_id, order_id, received_by, condition_notes, signed_proof_hash)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5)
		RETURNING id::text, shipment_id::text, order_id::text, received_by::text, received_at, condition_notes, signed_proof_hash
	`, input.ShipmentID, input.OrderID, input.ReceivedBy, input.ConditionNotes, hashHex).
		Scan(&rec.ID, &rec.ShipmentID, &rec.OrderID, &rec.ReceivedBy, &rec.ReceivedAt, &rec.ConditionNotes, &rec.SignedProofHash)
	if err != nil {
		return DeliveryReceipt{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE app.order_shipments SET status = 'delivered' WHERE id = $1::uuid
	`, input.ShipmentID)
	if err != nil {
		return DeliveryReceipt{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return DeliveryReceipt{}, err
	}
	return rec, nil
}

func (p *PostgresStore) CreateCreditNote(ctx context.Context, input CreditNoteInput) (CreditNote, error) {
	tx, err := p.beginTx(ctx, input.SupplierOrganizationID, input.IssuedBy)
	if err != nil {
		return CreditNote{}, err
	}
	defer tx.Rollback(ctx)

	var cn CreditNote
	var amount int64
	var obID *string
	if input.ObligationID != "" {
		obID = &input.ObligationID
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO app.order_credit_notes (order_id, obligation_id, supplier_organization_id, amount_kobo, reason, issued_by, status)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6::uuid, 'draft')
		RETURNING id::text, order_id::text, COALESCE(obligation_id::text, ''), supplier_organization_id::text, amount_kobo, reason, issued_by::text, status, created_at
	`, input.OrderID, obID, input.SupplierOrganizationID, int64(input.AmountKobo), input.Reason, input.IssuedBy).
		Scan(&cn.ID, &cn.OrderID, &cn.ObligationID, &cn.SupplierOrganizationID, &amount, &cn.Reason, &cn.IssuedBy, &cn.Status, &cn.CreatedAt)
	if err != nil {
		return CreditNote{}, err
	}
	cn.AmountKobo = ledger.Money(amount)
	if err := tx.Commit(ctx); err != nil {
		return CreditNote{}, err
	}
	return cn, nil
}

// ledgerAdjuster is the ledger's transactional adjustment method. Approving a
// credit note forgives debt, so the journal entry and the balance it forgives
// have to commit or roll back together; ledger.PostAdjustment opens its own
// transaction and cannot do that.
type ledgerAdjuster interface {
	PostAdjustmentTx(context.Context, pgx.Tx, string, ledger.Money, string, time.Time, string) (ledger.Transaction, error)
}

// ApproveCreditNote forgives buyer debt, so it follows the same sequence as
// operations.adjustTx (the write-off path): lock the obligation, refuse an
// amount it cannot cover, refuse to consume principal a bank is already
// collecting, post the journal inside this transaction, reduce the schedule,
// then move the balance through the one shared writer.
func (p *PostgresStore) ApproveCreditNote(ctx context.Context, noteID, reviewerID string) error {
	tx, err := p.beginTx(ctx, "", reviewerID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var obID *string
	var suppOrgID string
	var amountKobo int64
	err = tx.QueryRow(ctx, `
		UPDATE app.order_credit_notes
		SET status = 'approved', approved_by = $2::uuid, approved_at = now()
		WHERE id = $1::uuid AND status = 'draft' AND issued_by <> $2::uuid
		RETURNING obligation_id::text, supplier_organization_id::text, amount_kobo
	`, noteID, reviewerID).Scan(&obID, &suppOrgID, &amountKobo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrDualControl
		}
		return err
	}
	if suppOrgID != "" {
		if _, err = tx.Exec(ctx, `SELECT set_config('app.current_organization_id',$1,true)`, suppOrgID); err != nil {
			return err
		}
	}

	// A credit note on an order with no activated obligation forgives nothing
	// yet. Approving it is a record, and it must not post forgiveness.
	if obID == nil || *obID == "" {
		return tx.Commit(ctx)
	}
	obligation, amount := *obID, ledger.Money(amountKobo)

	var requestID string
	var outstanding, principal ledger.Money
	if err = tx.QueryRow(ctx, `SELECT credit_request_id::text,outstanding_kobo,principal_kobo FROM app.obligations WHERE id=$1::uuid AND supplier_organization_id=$2::uuid FOR UPDATE`, obligation, suppOrgID).Scan(&requestID, &outstanding, &principal); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	// Clamping the balance while posting the full amount to the journal would
	// leave the difference as a permanent ledger discrepancy. Refuse instead.
	if amount > outstanding {
		return fmt.Errorf("%w: credit note of %d kobo exceeds the %d kobo outstanding", ErrInvalidInput, amountKobo, int64(outstanding))
	}
	if err = db.GuardUnreservedReduction(ctx, tx, obligation, int64(outstanding-amount)); err != nil {
		return err
	}
	adjuster, ok := p.ledger.(ledgerAdjuster)
	if !ok {
		return errors.New("credit notes against an obligation require a ledger that posts inside this transaction")
	}
	if _, err = adjuster.PostAdjustmentTx(ctx, tx, noteID, amount, "credit_note", time.Now().UTC(), "credit-note-approval:"+noteID); err != nil {
		return err
	}
	if err = db.ReduceSchedulePrincipalTx(ctx, tx, obligation, outstanding, amount, false); err != nil {
		return err
	}
	if err = db.UpdateObligationBalanceTx(ctx, tx, requestID, obligation, outstanding-amount, principal); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (p *PostgresStore) ListCreditNotes(ctx context.Context, orderID string) ([]CreditNote, error) {
	tx, err := p.beginTx(ctx, "", "")
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id::text, order_id::text, COALESCE(obligation_id::text, ''), supplier_organization_id::text, amount_kobo, reason, issued_by::text, COALESCE(approved_by::text, ''), status, created_at, approved_at
		FROM app.order_credit_notes
		WHERE order_id = $1::uuid
		ORDER BY created_at DESC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CreditNote
	for rows.Next() {
		var cn CreditNote
		var amount int64
		if err := rows.Scan(&cn.ID, &cn.OrderID, &cn.ObligationID, &cn.SupplierOrganizationID, &amount, &cn.Reason, &cn.IssuedBy, &cn.ApprovedBy, &cn.Status, &cn.CreatedAt, &cn.ApprovedAt); err != nil {
			return nil, err
		}
		cn.AmountKobo = ledger.Money(amount)
		out = append(out, cn)
	}
	return out, rows.Err()
}

func (p *PostgresStore) ListShipments(ctx context.Context, orderID string) ([]Shipment, error) {
	tx, err := p.beginTx(ctx, "", "")
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id::text, order_id::text, supplier_organization_id::text, tracking_reference, carrier, dispatched_by::text, dispatched_at, status
		FROM app.order_shipments
		WHERE order_id = $1::uuid
		ORDER BY dispatched_at DESC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Shipment
	for rows.Next() {
		var s Shipment
		if err := rows.Scan(&s.ID, &s.OrderID, &s.SupplierOrganizationID, &s.TrackingReference, &s.Carrier, &s.DispatchedBy, &s.DispatchedAt, &s.Status); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
