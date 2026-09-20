package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/ledger"
	"kredit/internal/orders"
)

func (s *Server) getOrdersService() orders.Service {
	if s.runtime.Database != nil {
		return orders.NewPostgresStore(s.runtime.Database.Raw(), s.runtime.Ledger)
	}
	return orders.NewMemoryStore()
}

type lineItemInput struct {
	SKU           string `json:"sku"`
	Description   string `json:"description"`
	UnitPriceKobo int64  `json:"unit_price_kobo"`
	Quantity      int64  `json:"quantity"`
}

func (s *Server) createLineItems(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", err.Error())
		return
	}
	requestID, err := pathID(r, "requestID")
	if err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionCreateCredit)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var in struct {
		Items []lineItemInput `json:"items"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_payload", err.Error())
		return
	}
	if len(in.Items) == 0 {
		writeProblem(w, 400, "invalid_payload", "at least one line item is required")
		return
	}
	items := make([]orders.LineItem, len(in.Items))
	for i, it := range in.Items {
		items[i] = orders.LineItem{
			SKU:           it.SKU,
			Description:   it.Description,
			UnitPriceKobo: ledger.Money(it.UnitPriceKobo),
			Quantity:      it.Quantity,
		}
	}
	svc := s.getOrdersService()
	if err := svc.CreateLineItems(r.Context(), requestID, items); err != nil {
		writeProblem(w, 500, "create_line_items_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "order.line_items.created", requestID)
	writeJSON(w, 201, map[string]any{"status": "created", "count": len(items)})
}

func (s *Server) orderDeliveries(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", err.Error())
		return
	}
	requestID, err := pathID(r, "requestID")
	if err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	_, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadOrganization)
	if !ok {
		return
	}

	svc := s.getOrdersService()
	items, err := svc.ListLineItems(r.Context(), requestID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		items = []orders.LineItem{}
	}
	shipments, err := svc.ListShipments(r.Context(), requestID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		shipments = []orders.Shipment{}
	}
	creditNotes, err := svc.ListCreditNotes(r.Context(), requestID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		creditNotes = []orders.CreditNote{}
	}

	writeJSON(w, 200, map[string]any{
		"line_items":   items,
		"shipments":    shipments,
		"credit_notes": creditNotes,
	})
}

func (s *Server) createShipment(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", err.Error())
		return
	}
	requestID, err := pathID(r, "requestID")
	if err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReleaseGoods)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}

	var in struct {
		Carrier           string `json:"carrier"`
		TrackingReference string `json:"tracking_reference"`
		Items             []struct {
			LineItemID string `json:"line_item_id"`
			Quantity   int64  `json:"quantity"`
		} `json:"items"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}

	var items []orders.ShipmentItem
	for _, it := range in.Items {
		items = append(items, orders.ShipmentItem{
			LineItemID: it.LineItemID,
			Quantity:   it.Quantity,
		})
	}

	svc := s.getOrdersService()
	shipment, err := svc.DispatchShipment(r.Context(), orders.DispatchInput{
		OrderID:                requestID,
		SupplierOrganizationID: orgID,
		TrackingReference:      strings.TrimSpace(in.TrackingReference),
		Carrier:                strings.TrimSpace(in.Carrier),
		DispatchedBy:           user.ID,
		Items:                  items,
	})
	if err != nil {
		writeProblem(w, 422, "dispatch_failed", err.Error())
		return
	}

	s.runtime.Audit.Append(audit.Event{
		ActorUserID:    user.ID,
		OrganizationID: orgID,
		Action:         "order.shipment.dispatched",
		ResourceType:   "order_shipment",
		ResourceID:     shipment.ID,
		Outcome:        "success",
		RequestID:      requestIDFromContext(r.Context()),
	})

	writeJSON(w, 201, shipment)
}

func (s *Server) recordReceipt(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", err.Error())
		return
	}
	requestID, err := pathID(r, "requestID")
	if err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReleaseGoods)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}

	var in struct {
		ShipmentID     string `json:"shipment_id"`
		ConditionNotes string `json:"condition_notes"`
		SignedProof    string `json:"signed_proof"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}

	svc := s.getOrdersService()
	receipt, err := svc.RecordDeliveryReceipt(r.Context(), orders.DeliveryInput{
		ShipmentID:     in.ShipmentID,
		OrderID:        requestID,
		ReceivedBy:     user.ID,
		ConditionNotes: in.ConditionNotes,
		SignedProof:    in.SignedProof,
	})
	if err != nil {
		writeProblem(w, 422, "receipt_failed", err.Error())
		return
	}

	s.runtime.Audit.Append(audit.Event{
		ActorUserID:    user.ID,
		OrganizationID: orgID,
		Action:         "order.delivery.received",
		ResourceType:   "order_delivery_receipt",
		ResourceID:     receipt.ID,
		Outcome:        "success",
		RequestID:      requestIDFromContext(r.Context()),
	})

	writeJSON(w, 201, receipt)
}

func (s *Server) createCreditNote(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", err.Error())
		return
	}
	requestID, err := pathID(r, "requestID")
	if err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionCreateCredit)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}

	var in struct {
		AmountKobo   int64  `json:"amount_kobo"`
		Reason       string `json:"reason"`
		ObligationID string `json:"obligation_id"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}

	svc := s.getOrdersService()
	cn, err := svc.CreateCreditNote(r.Context(), orders.CreditNoteInput{
		OrderID:                requestID,
		ObligationID:           in.ObligationID,
		SupplierOrganizationID: orgID,
		AmountKobo:             ledger.Money(in.AmountKobo),
		Reason:                 in.Reason,
		IssuedBy:               user.ID,
	})
	if err != nil {
		writeProblem(w, 422, "credit_note_failed", err.Error())
		return
	}

	s.runtime.Audit.Append(audit.Event{
		ActorUserID:    user.ID,
		OrganizationID: orgID,
		Action:         "order.credit_note.created",
		ResourceType:   "order_credit_note",
		ResourceID:     cn.ID,
		Outcome:        "success",
		RequestID:      requestIDFromContext(r.Context()),
	})

	writeJSON(w, 201, cn)
}

func (s *Server) approveCreditNote(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", err.Error())
		return
	}
	noteID, err := pathID(r, "noteID")
	if err != nil {
		writeProblem(w, 400, "invalid_note", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionApproveBusinessCredit)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}

	svc := s.getOrdersService()
	err = svc.ApproveCreditNote(r.Context(), noteID, user.ID)
	if err != nil {
		if errors.Is(err, orders.ErrDualControl) {
			writeProblem(w, 403, "dual_control_required", "The issuer cannot approve their own credit note. A different authorized person must approve.")
			return
		}
		writeProblem(w, 422, "approval_failed", err.Error())
		return
	}

	s.runtime.Audit.Append(audit.Event{
		ActorUserID:    user.ID,
		OrganizationID: orgID,
		Action:         "order.credit_note.approved",
		ResourceType:   "order_credit_note",
		ResourceID:     noteID,
		Outcome:        "success",
		RequestID:      requestIDFromContext(r.Context()),
	})

	writeJSON(w, 200, map[string]string{"id": noteID, "status": "approved"})
}
