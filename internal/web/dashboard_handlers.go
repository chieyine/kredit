package web

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"kredit/internal/access"
	"kredit/internal/buyers"
	"kredit/internal/db"
	"kredit/internal/disputes"
	"kredit/internal/ledger"
	"kredit/internal/mandates"
	"kredit/internal/tradelines"
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
		_, scheduleItems, err := s.runtime.Schedules.ForContext(r.Context()).GetForObligation(view.Obligation.ID)
		if financialReadError(w, err) {
			return
		}
		var overdue int64
		for _, item := range scheduleItems {
			if item.State != "PAID" && item.State != "CANCELLED" && !now.Before(item.CollectionAt) {
				remaining := item.PrincipalDueKobo - item.AllocatedKobo
				if remaining <= 0 {
					continue
				}
				total, err := ledger.CheckedAdd(ledger.Money(overdue), remaining)
				if financialReadError(w, err) {
					return
				}
				overdue = int64(total)
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
	type customerSummary struct {
		fields       map[string]any
		requestCount int
		outstanding  ledger.Money
	}
	customers := map[string]*customerSummary{}
	var buyerRows []buyers.Customer
	if source, ok := s.runtime.Buyers.(interface {
		ReadCustomers(string) ([]buyers.Customer, error)
	}); ok {
		var err error
		buyerRows, err = source.ReadCustomers(organizationID)
		if financialReadError(w, err) {
			return
		}
	} else {
		buyerRows = s.runtime.Buyers.ListCustomers(organizationID)
	}
	for _, customer := range buyerRows {
		customers[customer.BuyerUserID+"\x00"+customer.BuyerBusinessID] = &customerSummary{fields: map[string]any{"id": customer.BuyerUserID, "buyer_user_id": customer.BuyerUserID, "buyer_business_id": customer.BuyerBusinessID, "legal_name": customer.LegalName, "trading_name": customer.TradingName, "industry": customer.Industry, "state": customer.Status}}
	}
	financialRows, readErr := s.runtime.readCreditForSupplier(r.Context(), organizationID)
	if financialReadError(w, readErr) {
		return
	}
	for _, view := range financialRows {
		key := view.Request.BuyerUserID + "\x00" + view.Request.BuyerBusinessID
		customer := customers[key]
		if customer == nil {
			customer = &customerSummary{fields: map[string]any{"id": view.Request.BuyerUserID, "buyer_user_id": view.Request.BuyerUserID, "buyer_business_id": view.Request.BuyerBusinessID, "legal_name": view.Request.BuyerLegalName, "trading_name": view.Request.BuyerTradingName, "state": "ACTIVE"}}
			customers[key] = customer
		}
		customer.requestCount++
		if view.Obligation != nil {
			total, err := ledger.CheckedAdd(customer.outstanding, view.Obligation.OutstandingKobo)
			if financialReadError(w, err) {
				return
			}
			customer.outstanding = total
		}
	}
	canReadFinancial := access.Can(membership.Role, access.PermissionReadFinancial)
	type sortableCustomer struct {
		key, legalName string
		fields         map[string]any
	}
	sorted := make([]sortableCustomer, 0, len(customers))
	for key, customer := range customers {
		if canReadFinancial {
			customer.fields["request_count"] = customer.requestCount
			customer.fields["outstanding_kobo"] = int64(customer.outstanding)
		}
		legalName, _ := customer.fields["legal_name"].(string)
		sorted = append(sorted, sortableCustomer{key: key, legalName: legalName, fields: customer.fields})
	}
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].legalName != sorted[j].legalName {
			return sorted[i].legalName < sorted[j].legalName
		}
		return sorted[i].key < sorted[j].key
	})
	items := make([]map[string]any, 0, len(sorted))
	for _, customer := range sorted {
		items = append(items, customer.fields)
	}
	writeJSON(w, http.StatusOK, map[string]any{"customers": items})
}

func (s *Server) listBuyerMandates(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	businessID, scopedOK := s.buyerOwnerWorkspaceScope(w, r, user.ID)
	if !scopedOK {
		return
	}
	if reader, ok := s.runtime.Mandates.(mandates.BuyerReader); ok {
		items, err := reader.ReadForBuyer(r.Context(), user.ID)
		if financialReadError(w, err) {
			return
		}
		if businessID != "" {
			items = purchasingRows(items, func(item mandates.Mandate) bool { return item.BusinessID == businessID })
		}
		writeJSON(w, http.StatusOK, map[string]any{"mandates": items})
		return
	}
	items := []any{}
	seen := map[string]bool{}
	financialRows7, readErr7 := s.runtime.readCreditForBuyer(r.Context(), user.ID)
	if financialReadError(w, readErr7) {
		return
	}
	for _, view := range financialRows7 {
		if (businessID == "" || view.Request.BuyerBusinessID == businessID) && view.Mandate != nil && !seen[view.Mandate.ID] {
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
	businessID, scopedOK := s.buyerOwnerWorkspaceScope(w, r, user.ID)
	if !scopedOK {
		return
	}
	workspaceID := strings.TrimSpace(r.URL.Query().Get("organization"))
	var financialRows8 []tradelines.TradeLine
	var readErr8 error
	if workspaceID != "" {
		financialRows8, readErr8 = s.runtime.readTradeLinesForBuyerOrganization(r.Context(), workspaceID)
	} else {
		financialRows8, readErr8 = s.runtime.readTradeLinesForBuyer(r.Context(), user.ID)
	}
	if financialReadError(w, readErr8) {
		return
	}
	if businessID != "" {
		financialRows8 = purchasingRows(financialRows8, func(item tradelines.TradeLine) bool { return item.BuyerBusinessID == businessID })
	}
	writeJSON(w, http.StatusOK, map[string]any{"trade_lines": financialRows8})
}

func (s *Server) listBuyerDisputes(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	businessID, scopedOK := s.buyerOwnerWorkspaceScope(w, r, user.ID)
	if !scopedOK {
		return
	}
	obligations, scopedOK := s.buyerWorkspaceObligations(w, r, user.ID, businessID)
	if !scopedOK {
		return
	}
	financialRows9, readErr9 := s.runtime.readDisputesForBuyer(r.Context(), user.ID)
	if financialReadError(w, readErr9) {
		return
	}
	if businessID != "" {
		financialRows9 = purchasingRows(financialRows9, func(item disputes.Dispute) bool { return obligations[item.ObligationID] })
	}
	writeJSON(w, http.StatusOK, map[string]any{"disputes": financialRows9})
}
