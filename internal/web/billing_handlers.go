package web

import (
	"encoding/json"
	"kredit/internal/access"
	"kredit/internal/billing"
	"net/http"
)

func (s *Server) feeInvoices(w http.ResponseWriter, r *http.Request) {
	org, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", "Invalid business reference.")
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, org, access.PermissionReadFinancial)
	if !ok {
		return
	}
	s.writeFeeInvoices(w, r, org, user.ID)
}
func (s *Server) adminFeeInvoices(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformSettings)
	if !ok {
		return
	}
	org, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", "Invalid business reference.")
		return
	}
	s.auditPlatformRead(r, user.ID, "billing.invoices.viewed", "organization", org)
	s.writeFeeInvoices(w, r, org, user.ID)
}
func (s *Server) writeFeeInvoices(w http.ResponseWriter, r *http.Request, org, actor string) {
	if s.runtime.Database == nil {
		writeProblem(w, 503, "billing_unavailable", "Fee bills are unavailable.")
		return
	}
	items, err := billing.NewStore(s.runtime.Database.Raw()).List(r.Context(), org, actor)
	if err != nil {
		writeProblem(w, 503, "billing_unavailable", "Fee bills could not be loaded.")
		return
	}
	writeJSON(w, 200, map[string]any{"invoices": items})
}
func (s *Server) listBillingReview(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformSettings)
	if !ok {
		return
	}
	if s.runtime.Database == nil {
		writeProblem(w, 503, "billing_unavailable", "Billing review is unavailable.")
		return
	}
	tx, err := s.runtime.Database.Raw().Begin(r.Context())
	if err != nil {
		writeProblem(w, 503, "billing_unavailable", "Billing review is unavailable.")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	_, err = tx.Exec(r.Context(), `SELECT set_config('app.current_user_id',$1,true)`, user.ID)
	var raw json.RawMessage
	if err == nil {
		err = tx.QueryRow(r.Context(), `SELECT app.invoice_billing_review()`).Scan(&raw)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		writeProblem(w, 503, "billing_unavailable", "Billing reviews could not be loaded.")
		return
	}
	s.auditPlatformRead(r, user.ID, "billing.review.viewed", "fee_invoice", "")
	writeJSON(w, 200, raw)
}
func (s *Server) approveInvoiceBilling(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformSettings)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	org, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", "Invalid business reference.")
		return
	}
	var in struct {
		Version      int64  `json:"version"`
		Reference    string `json:"billing_reference"`
		Instructions string `json:"payment_instructions"`
	}
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	if s.runtime.Database == nil {
		writeProblem(w, 503, "billing_unavailable", "Durable billing is unavailable.")
		return
	}
	p, summary, err := s.runtime.Onboarding.ApproveInvoiceBilling(org, user.ID, in.Reference, in.Version, in.Instructions)
	s.finishOnboardingChange(w, r, user, org, "supplier.onboarding.billing.approved", p, summary, err)
}
func (s *Server) recordFeeInvoiceReceipt(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformSettings)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	org, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", "Invalid business reference.")
		return
	}
	id, err := pathID(r, "invoiceID")
	if err != nil {
		writeProblem(w, 400, "invalid_invoice", "Invalid invoice reference.")
		return
	}
	var in billing.Receipt
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	if s.runtime.Database == nil {
		writeProblem(w, 503, "billing_unavailable", "Durable billing is unavailable.")
		return
	}
	invoice, err := billing.NewStore(s.runtime.Database.Raw()).RecordReceipt(r.Context(), org, user.ID, id, in)
	if err != nil {
		writeProblem(w, 409, "receipt_not_recorded", "The receipt was not confirmed. Check the invoice balance, date and bank reference before retrying.")
		return
	}
	writeJSON(w, 200, map[string]any{"invoice": invoice})
}
