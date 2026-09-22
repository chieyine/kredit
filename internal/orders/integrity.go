package orders

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

var (
	ErrAuthority                       = errors.New("current order authority required")
	ErrConflict                        = errors.New("order evidence conflicts with an existing record")
	ErrFinancialApplicationUnavailable = errors.New("financial credit-note application requires the durable ledger")
)

const maxOrderMoney int64 = 9007199254740991

func validateLineItems(orderID string, items []LineItem) error {
	if orderID == "" || len(items) == 0 || len(items) > 500 {
		return ErrInvalidInput
	}
	for _, item := range items {
		if strings.TrimSpace(item.Description) == "" || len(item.Description) > 2000 || len(item.SKU) > 200 ||
			item.UnitPriceKobo < 0 || int64(item.UnitPriceKobo) > maxOrderMoney || item.Quantity <= 0 || item.Quantity > maxOrderMoney ||
			(item.UnitPriceKobo > 0 && item.Quantity > maxOrderMoney/int64(item.UnitPriceKobo)) ||
			item.FulfilledQuantity != 0 || item.ReturnedQuantity != 0 {
			return ErrInvalidInput
		}
	}
	return nil
}
func validateDispatch(input DispatchInput) error {
	if input.OrderID == "" || input.SupplierOrganizationID == "" || input.DispatchedBy == "" || len(input.Items) == 0 || len(input.Items) > 500 || len(input.TrackingReference) > 500 || len(input.Carrier) > 200 {
		return ErrInvalidInput
	}
	seen := make(map[string]bool, len(input.Items))
	for _, item := range input.Items {
		if item.LineItemID == "" || item.Quantity <= 0 || item.Quantity > maxOrderMoney || seen[item.LineItemID] {
			return ErrInvalidInput
		}
		seen[item.LineItemID] = true
	}
	return nil
}
func validateDelivery(input DeliveryInput) error {
	if input.ShipmentID == "" || input.OrderID == "" || input.ReceivedBy == "" || strings.TrimSpace(input.SignedProof) == "" || len(input.SignedProof) > 131072 || len(input.ConditionNotes) > 4000 {
		return ErrInvalidInput
	}
	return nil
}
func validateCreditNote(input CreditNoteInput) error {
	if input.OrderID == "" || input.SupplierOrganizationID == "" || input.IssuedBy == "" || input.AmountKobo <= 0 || int64(input.AmountKobo) > maxOrderMoney || strings.TrimSpace(input.Reason) == "" || len(input.Reason) > 4000 {
		return ErrInvalidInput
	}
	return nil
}
func sortedShipmentItems(items []ShipmentItem) []ShipmentItem {
	out := append([]ShipmentItem(nil), items...)
	sort.Slice(out, func(i, j int) bool { return out[i].LineItemID < out[j].LineItemID })
	return out
}
func sameReceiptIntent(receipt DeliveryReceipt, input DeliveryInput, proof string) bool {
	return receipt.ShipmentID == input.ShipmentID && receipt.OrderID == input.OrderID && receipt.ReceivedBy == input.ReceivedBy && receipt.ConditionNotes == input.ConditionNotes && receipt.SignedProofHash == proof
}
func cloneShipment(shipment Shipment) Shipment {
	shipment.Items = append([]ShipmentItem(nil), shipment.Items...)
	return shipment
}
func cloneCreditNote(note CreditNote) CreditNote {
	if note.ApprovedAt != nil {
		at := *note.ApprovedAt
		note.ApprovedAt = &at
	}
	return note
}

// Lock authority before financial records, then an existing obligation before
// its credit request. In particular a note approval must never hold a request
// while waiting for an obligation held by payment reversal or balance repair.
// Every lookup retains the authenticated tenant and the parent's RLS boundary.
func lockSupplierOrder(ctx context.Context, tx pgx.Tx, orderID string, roles []string) error {
	const authorizedOrder = `SELECT cr.id::text FROM app.credit_requests cr
        JOIN app.memberships m ON m.organization_id=cr.supplier_organization_id AND m.user_id=app.current_user_id()
        JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id
        WHERE cr.id=$1::uuid AND cr.supplier_organization_id=app.current_organization_id()
        AND m.status='active' AND m.role=ANY($2::text[]) AND u.status='active' AND o.status<>'suspended'`
	var id string
	if err := tx.QueryRow(ctx, authorizedOrder+` FOR SHARE OF m,u,o`, orderID, roles).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	var lockedObligation string
	err := tx.QueryRow(ctx, `SELECT id::text FROM app.obligations
        WHERE credit_request_id=$1::uuid AND supplier_organization_id=app.current_organization_id()
        FOR UPDATE`, orderID).Scan(&lockedObligation)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err := tx.QueryRow(ctx, authorizedOrder+` FOR UPDATE OF cr`, orderID, roles).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	// An activation may have committed while this transaction waited for the
	// request. Refuse before taking a late obligation lock (which would invert
	// the order). The caller rolls back; it must start a fresh transaction.
	var currentObligation string
	if err := tx.QueryRow(ctx, `SELECT COALESCE((SELECT id::text FROM app.obligations
        WHERE credit_request_id=$1::uuid AND supplier_organization_id=app.current_organization_id()),'')`, orderID).Scan(&currentObligation); err != nil {
		return err
	}
	if currentObligation != lockedObligation {
		return ErrConflict
	}
	return nil
}
