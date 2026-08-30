package web

import (
	"net/http"
	"sort"
	"time"

	"kredit/internal/access"
)

func (s *Server) listOrganizationPayments(w http.ResponseWriter, r *http.Request) {
	organizationID, _ := pathID(r, "organizationID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionReadFinancial); !ok {
		return
	}
	items := []map[string]any{}
	for _, view := range s.runtime.Credit.ListForSupplier(organizationID) {
		if view.Obligation == nil {
			continue
		}
		for _, payment := range s.runtime.Payments.List(view.Obligation.ID) {
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
	if _, _, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionReadFinancial); !ok {
		return
	}
	items := []map[string]any{}
	for _, view := range s.runtime.Credit.ListForSupplier(organizationID) {
		if view.Obligation == nil {
			continue
		}
		for _, attempt := range s.runtime.Collections.ListAttempts(view.Obligation.ID) {
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
	for _, view := range s.runtime.Credit.ListForSupplier(organizationID) {
		if view.Obligation == nil || view.Obligation.OutstandingKobo <= 0 {
			continue
		}
		_, scheduleItems, err := s.runtime.Schedules.GetForObligation(view.Obligation.ID)
		if err != nil {
			continue
		}
		var overdue int64
		for _, item := range scheduleItems {
			if !now.Before(item.CollectionAt) {
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
	if _, _, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionReadFinancial); !ok {
		return
	}
	customers := map[string]map[string]any{}
	for _, customer := range s.runtime.Buyers.ListCustomers(organizationID) {
		customers[customer.BuyerUserID] = map[string]any{"id": customer.BuyerUserID, "buyer_user_id": customer.BuyerUserID, "buyer_business_id": customer.BuyerBusinessID, "legal_name": customer.LegalName, "trading_name": customer.TradingName, "industry": customer.Industry, "state": customer.Status, "request_count": 0, "outstanding_kobo": int64(0)}
	}
	for _, view := range s.runtime.Credit.ListForSupplier(organizationID) {
		customer := customers[view.Request.BuyerUserID]
		if customer == nil {
			customer = map[string]any{"id": view.Request.BuyerUserID, "buyer_user_id": view.Request.BuyerUserID, "buyer_business_id": view.Request.BuyerBusinessID, "legal_name": view.Request.BuyerLegalName, "trading_name": view.Request.BuyerTradingName, "state": "ACTIVE", "request_count": 0, "outstanding_kobo": int64(0)}
			customers[view.Request.BuyerUserID] = customer
		}
		customer["request_count"] = customer["request_count"].(int) + 1
		if view.Obligation != nil {
			customer["outstanding_kobo"] = customer["outstanding_kobo"].(int64) + int64(view.Obligation.OutstandingKobo)
		}
	}
	items := make([]map[string]any, 0, len(customers))
	for _, customer := range customers {
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
	for _, view := range s.runtime.Credit.ListForBuyer(user.ID) {
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
	writeJSON(w, http.StatusOK, map[string]any{"trade_lines": s.runtime.TradeLines.ListForBuyer(user.ID)})
}

func (s *Server) listBuyerDisputes(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"disputes": s.runtime.Disputes.ListForBuyer(user.ID)})
}
