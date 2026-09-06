package web

import (
	"net/http"
	"sort"
	"time"

	"kredit/internal/access"
	"kredit/internal/buyers"
	"kredit/internal/db"
	"kredit/internal/ledger"
)

func (s *Server) listOrganizationPayments(w http.ResponseWriter, r *http.Request) {
	organizationID, _ := pathID(r, "organizationID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionReadFinancial)
	if !ok {
		return
	}
	r = r.WithContext(db.WithTenantContext(r.Context(), user.ID, organizationID))
	items := []map[string]any{}
	financialRows1, readErr1 := s.runtime.readCreditForSupplier(r.Context(), organizationID)
	if financialReadError(w, readErr1) {
		return
	}
	for _, view := range financialRows1 {
		if view.Obligation == nil {
			continue
		}
		financialRows2, readErr2 := s.runtime.readPayments(r.Context(), view.Obligation.ID)
		if financialReadError(w, readErr2) {
			return
		}
		for _, payment := range financialRows2 {
			items = append(items, map[string]any{
				"id": view.Request.ID, "payment_id": payment.ID, "reference": payment.ProviderReference,
				"buyer_legal_name": view.Request.BuyerLegalName, "description": view.Request.GoodsDescription,
				"amount_kobo": payment.AmountKobo, "source_type": payment.SourceType, "state": payment.State,
				"paid_at": payment.PaidAt,
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"payments": items})
}

func (s *Server) listOrganizationCollections(w http.ResponseWriter, r *http.Request) {
	organizationID, _ := pathID(r, "organizationID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionReadFinancial)
	if !ok {
		return
	}
	r = r.WithContext(db.WithTenantContext(r.Context(), user.ID, organizationID))
	items := []map[string]any{}
	financialRows3, readErr3 := s.runtime.readCreditForSupplier(r.Context(), organizationID)
	if financialReadError(w, readErr3) {
		return
	}
	for _, view := range financialRows3 {
		if view.Obligation == nil {
			continue
		}
		financialRows4, readErr4 := s.runtime.readCollectionsAttemptsContext(r.Context(), view.Obligation.ID)
		if financialReadError(w, readErr4) {
			return
		}
		for _, attempt := range financialRows4 {
			items = append(items, map[string]any{
				"id": view.Request.ID, "attempt_id": attempt.ID, "buyer_legal_name": view.Request.BuyerLegalName,
				"description": view.Request.GoodsDescription, "amount_kobo": attempt.RequestedAmountKobo,
				"state": attempt.State, "created_at": attempt.RequestedAt,
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"collections": items})
}

func (s *Server) listOrganizationOverdue(w http.ResponseWriter, r *http.Request) {
	organizationID, _ := pathID(r, "organizationID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionReadFinancial); !ok {
		return
	}
	now := time.Now().UTC()
	items := []map[string]any{}
	financialRows5, readErr5 := s.runtime.readCreditForSupplier(r.Context(), organizationID)
	if financialReadError(w, readErr5) {
		return
	}
	for _, view := range financialRows5 {
		if view.Obligation == nil || view.Obligation.OutstandingKobo <= 0 {
			continue
		}
		_, scheduleItems, err := s.runtime.Schedules.GetForObligation(view.Obligation.ID)
		if financialReadError(w, err) {
			return
		}
		var overdue int64
		for _, item := range scheduleItems {
			if item.State != "PAID" && item.State != "CANCELLED" && !now.Before(item.CollectionAt) {
				overdue += int64(item.PrincipalDueKobo - item.AllocatedKobo)
			}
		}
		if overdue > 0 {
			items = append(items, map[string]any{"id": view.Request.ID, "buyer_legal_name": view.Request.BuyerLegalName, "description": view.Request.GoodsDescription, "state": "OVERDUE", "amount_kobo": overdue, "due_date": view.Request.DueDate})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"overdue": items})
}

func (s *Server) listOrganizationCustomers(w http.ResponseWriter, r *http.Request) {
	organizationID, _ := pathID(r, "organizationID")
	_, _, membership, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionReadOrganization)
	if !ok {
		return
	}
	customers := map[string]map[string]any{}
	var buyerRows []buyers.Customer
	if source, ok := s.runtime.Buyers.(interface {
		ReadCustomers(string) ([]buyers.Customer, error)
	}); ok {
		var err error
		buyerRows, err = source.ReadCustomers(organizationID)
		if financialReadError(w, err) {
			return
		}
	}
	if buyerRows == nil {
		buyerRows = s.runtime.Buyers.ListCustomers(organizationID)
	}
	for _, customer := range buyerRows {
		customers[customer.BuyerUserID] = map[string]any{"id": customer.BuyerUserID, "buyer_user_id": customer.BuyerUserID, "buyer_business_id": customer.BuyerBusinessID, "legal_name": customer.LegalName, "trading_name": customer.TradingName, "industry": customer.Industry, "state": customer.Status, "request_count": 0, "outstanding_kobo": int64(0)}
	}
	financialRows6, readErr6 := s.runtime.readCreditForSupplier(r.Context(), organizationID)
	if financialReadError(w, readErr6) {
		return
	}
	for _, view := range financialRows6 {
		customer := customers[view.Request.BuyerUserID]
		if customer == nil {
			customer = map[string]any{"id": view.Request.BuyerUserID, "buyer_user_id": view.Request.BuyerUserID, "buyer_business_id": view.Request.BuyerBusinessID, "legal_name": view.Request.BuyerLegalName, "trading_name": view.Request.BuyerTradingName, "state": "ACTIVE", "request_count": 0, "outstanding_kobo": int64(0)}
			customers[view.Request.BuyerUserID] = customer
		}
		customer["request_count"] = customer["request_count"].(int) + 1
		if view.Obligation != nil {
			total, err := ledger.CheckedAdd(ledger.Money(customer["outstanding_kobo"].(int64)), view.Obligation.OutstandingKobo)
			if financialReadError(w, err) {
				return
			}
			customer["outstanding_kobo"] = int64(total)
		}
	}
	items := make([]map[string]any, 0, len(customers))
	for _, customer := range customers {
		if !access.Can(membership.Role, access.PermissionReadFinancial) {
			delete(customer, "outstanding_kobo")
			delete(customer, "request_count")
		}
		items = append(items, customer)
	}
	sort.Slice(items, func(i, j int) bool { return items[i]["legal_name"].(string) < items[j]["legal_name"].(string) })
	writeJSON(w, http.StatusOK, map[string]any{"customers": items})
}

func (s *Server) listBuyerMandates(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	items := []any{}
	seen := map[string]bool{}
	financialRows7, readErr7 := s.runtime.readCreditForBuyer(r.Context(), user.ID)
	if financialReadError(w, readErr7) {
		return
	}
	for _, view := range financialRows7 {
		if view.Mandate != nil && !seen[view.Mandate.ID] {
			seen[view.Mandate.ID] = true
			items = append(items, *view.Mandate)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"mandates": items})
}

func (s *Server) listBuyerTradeLines(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	financialRows8, readErr8 := s.runtime.readTradeLinesForBuyer(user.ID)
	if financialReadError(w, readErr8) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"trade_lines": financialRows8})
}

func (s *Server) listBuyerDisputes(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	financialRows9, readErr9 := s.runtime.readDisputesForBuyer(user.ID)
	if financialReadError(w, readErr9) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"disputes": financialRows9})
}
