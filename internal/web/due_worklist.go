package web

import (
	"kredit/internal/access"
	"kredit/internal/ledger"
	"net/http"
	"sort"
	"time"
)

// This list uses the current repayment schedule, including agreed changes.
// A due payment is not a permission to debit the customer's bank.
func (s *Server) listOrganizationDue(w http.ResponseWriter, r *http.Request) {
	org, _ := pathID(r, "organizationID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, org, access.PermissionReadFinancial); !ok {
		return
	}
	views, err := s.runtime.readCreditForSupplier(r.Context(), org)
	if financialReadError(w, err) {
		return
	}
	now := time.Now().UTC()
	horizon := now.Add(7 * 24 * time.Hour)
	type dueItem struct {
		id    string
		dueAt time.Time
		body  map[string]any
	}
	dueItems := []dueItem{}
	for _, v := range views {
		if v.Obligation == nil || v.Obligation.OutstandingKobo <= 0 {
			continue
		}
		_, schedule, err := s.runtime.Schedules.ForContext(r.Context()).GetForObligation(v.Obligation.ID)
		if financialReadError(w, err) {
			return
		}
		var amount ledger.Money
		var due time.Time
		for _, item := range schedule {
			remaining := item.PrincipalDueKobo - item.AllocatedKobo
			if item.State == "CANCELLED" || remaining <= 0 || item.DueAt.After(horizon) {
				continue
			}
			if due.IsZero() || item.DueAt.Before(due) {
				due = item.DueAt
			}
			amount, err = ledger.CheckedAdd(amount, remaining)
			if financialReadError(w, err) {
				return
			}
		}
		if amount <= 0 {
			continue
		}
		if amount > v.Obligation.OutstandingKobo {
			amount = v.Obligation.OutstandingKobo
		}
		state := "DUE_SOON"
		if !due.After(now) {
			state = "DUE"
		}
		dueItems = append(dueItems, dueItem{id: v.Request.ID, dueAt: due, body: map[string]any{"id": v.Request.ID, "credit_request_id": v.Request.ID, "obligation_id": v.Obligation.ID, "buyer_legal_name": v.Request.BuyerLegalName, "amount_kobo": amount, "due_at": due, "state": state}})
	}
	sort.SliceStable(dueItems, func(i, j int) bool {
		if !dueItems[i].dueAt.Equal(dueItems[j].dueAt) {
			return dueItems[i].dueAt.Before(dueItems[j].dueAt)
		}
		return dueItems[i].id < dueItems[j].id
	})
	items := make([]map[string]any, 0, len(dueItems))
	for _, item := range dueItems {
		items = append(items, item.body)
	}
	writeJSON(w, 200, map[string]any{"due": items, "as_of": now, "through": horizon})
}
