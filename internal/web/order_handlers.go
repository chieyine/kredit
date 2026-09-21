package web

import (
	"errors"
	"net/http"
	"strings"

	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/db"
	"kredit/internal/ledger"
	"kredit/internal/orders"

	"github.com/jackc/pgx/v5"
)

func (s *Server) getOrdersService() orders.Service {
	s.initDomainServices()
	return s.orders
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
	if _, err := s.runtime.getCreditForSupplier(r.Context(), requestID, orgID); err != nil {
		s.writeOrderProblem(w, r, err)
		return
	}
	svc := s.getOrdersService()
	if err := svc.CreateLineItems(r.Context(), requestID, items); err != nil {
		s.writeOrderProblem(w, r, err)
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

	if _, err := s.runtime.getCreditForSupplier(r.Context(), requestID, orgID); err != nil {
		s.writeOrderProblem(w, r, err)
		return
	}
	svc := s.getOrdersService()
	items, err := svc.ListLineItems(r.Context(), requestID)
	if err != nil {
		s.writeOrderProblem(w, r, err)
		return
	}
	shipments, err := svc.ListShipments(r.Context(), requestID)
	if err != nil {
		s.writeOrderProblem(w, r, err)
		return
	}
	creditNotes, err := svc.ListCreditNotes(r.Context(), requestID)
	if err != nil {
		s.writeOrderProblem(w, r, err)
		return
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

	if _, err := s.runtime.getCreditForSupplier(r.Context(), requestID, orgID); err != nil {
		s.writeOrderProblem(w, r, err)
		return
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
		s.writeOrderProblem(w, r, err)
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
	orgID := r.PathValue("organizationID")
	requestID, err := pathID(r, "requestID")
	if err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	_, user, ok := s.requireAuth(w, r)
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

	*r = *r.WithContext(db.WithTenantContext(r.Context(), user.ID, ""))
	// The durable receipt policy checks current purchasing authority again
	// under locks. Memory simulation must still authenticate the real buyer.
	if s.runtime.Database == nil {
		view, err := s.runtime.Credit.GetForBuyer(requestID, user.ID)
		if err != nil || (orgID != "" && orgID != view.Request.SupplierOrganizationID) {
			writeProblem(w, 404, "order_not_found", "Order was not found.")
			return
		}
		orgID = view.Request.SupplierOrganizationID
	} else {
		var supplier string
		scoped := &db.ScopedDatabase{Pool: s.runtime.Database.Raw()}
		err := scoped.QueryRow(r.Context(), `SELECT supplier_organization_id::text FROM app.credit_requests
            WHERE id=$1::uuid AND app.can_purchase(buyer_business_id,'receive')`, requestID).Scan(&supplier)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && orgID != "" && orgID != supplier) {
			writeProblem(w, 404, "order_not_found", "Order was not found.")
			return
		}
		if err != nil {
			s.writeOrderProblem(w, r, err)
			return
		}
		// Audit attribution only: the database context stays user-scoped.
		orgID = supplier
	}
	if shipmentID := r.PathValue("shipmentID"); shipmentID != "" {
		if in.ShipmentID != "" && in.ShipmentID != shipmentID {
			writeProblem(w, 400, "invalid_shipment", "Shipment identifiers disagree.")
			return
		}
		in.ShipmentID = shipmentID
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
		s.writeOrderProblem(w, r, err)
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

	if _, err := s.runtime.getCreditForSupplier(r.Context(), requestID, orgID); err != nil {
		s.writeOrderProblem(w, r, err)
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
		s.writeOrderProblem(w, r, err)
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
		s.writeOrderProblem(w, r, err)
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
