package web

import (
	"net/http"
	"time"

	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/erp"
	"kredit/internal/ledger"
)

func (s *Server) erpReconcile(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}

	var in struct {
		PeriodStart time.Time            `json:"period_start"`
		PeriodEnd   time.Time            `json:"period_end"`
		External    []erp.ExternalRecord `json:"external_records"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_payload", err.Error())
		return
	}
	if in.PeriodStart.IsZero() {
		in.PeriodStart = time.Now().UTC().AddDate(0, -1, 0)
	}
	if in.PeriodEnd.IsZero() {
		in.PeriodEnd = time.Now().UTC()
	}

	var internal []erp.InternalRecord
	if s.runtime.Database != nil {
		rows, err := s.runtime.Database.Raw().Query(r.Context(), `
			SELECT t.reference_id, t.id::text, p.debit_kobo, t.effective_at
			FROM ledger.transactions t
			JOIN ledger.postings p ON p.transaction_id = t.id
			JOIN ledger.accounts a ON a.id = p.account_id
			JOIN app.obligations o ON o.id::text = t.reference_id
			WHERE o.supplier_organization_id = $1::uuid
			  AND a.code = 'TRADE_RECEIVABLE_CONTROL'
			  AND t.effective_at >= $2 AND t.effective_at <= $3
			ORDER BY t.effective_at, t.id
		`, orgID, in.PeriodStart, in.PeriodEnd)
		if err != nil {
			writeProblem(w, 500, "ledger_read_failed", err.Error())
			return
		}
		defer rows.Close()
		for rows.Next() {
			var rec erp.InternalRecord
			var amt int64
			if err := rows.Scan(&rec.Reference, &rec.TransactionID, &amt, &rec.Date); err != nil {
				writeProblem(w, 500, "ledger_scan_failed", err.Error())
				return
			}
			rec.AmountKobo = ledger.Money(amt)
			internal = append(internal, rec)
		}
		if rows.Err() != nil {
			writeProblem(w, 500, "ledger_read_failed", rows.Err().Error())
			return
		}
	} else if s.runtime.Reports != nil {
		receivables, err := s.runtime.Reports.ReceivablesForSupplier(r.Context(), orgID)
		if err != nil {
			writeProblem(w, 500, "reports_unavailable", err.Error())
			return
		}
		for _, row := range receivables.Rows {
			t, _ := time.Parse(time.RFC3339, row.DueDate)
			if t.IsZero() {
				t, _ = time.Parse("2006-01-02", row.DueDate)
			}
			if s.runtime.Ledger != nil {
				txs, err := s.runtime.Ledger.GetByReference(row.ObligationID)
				if err != nil {
					writeProblem(w, 500, "ledger_read_failed", err.Error())
					return
				}
				for _, tx := range txs {
					for _, p := range tx.Postings {
						if p.Account == ledger.AccountTradeReceivable && p.Debit > 0 {
							internal = append(internal, erp.InternalRecord{
								Reference:     row.ObligationID,
								TransactionID: tx.ID,
								AmountKobo:    p.Debit,
								Date:          tx.EffectiveAt,
							})
						}
					}
				}
			} else {
				internal = append(internal, erp.InternalRecord{
					Reference:     row.ObligationID,
					TransactionID: row.ObligationID,
					AmountKobo:    row.PrincipalKobo,
					Date:          t,
				})
			}
		}
	}

	report, err := erp.ReconcileExternal(orgID, in.PeriodStart, in.PeriodEnd, internal, in.External)
	if err != nil {
		writeProblem(w, 500, "reconciliation_failed", err.Error())
		return
	}

	s.runtime.Audit.Append(audit.Event{
		ActorUserID:    user.ID,
		OrganizationID: orgID,
		Action:         "erp.reconciled",
		ResourceType:   "erp_reconciliation",
		ResourceID:     report.ReportHash,
		Outcome:        "success",
		RequestID:      requestIDFromContext(r.Context()),
	})

	writeJSON(w, 200, report)
}
