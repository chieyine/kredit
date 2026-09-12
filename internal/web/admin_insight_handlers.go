package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"kredit/internal/access"
	"kredit/internal/businesspolicy"
	"kredit/internal/db"
	"kredit/internal/operations"

	"github.com/jackc/pgx/v5"
)

func (s *Server) adminChangeContext(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionAdminFinancial)
	_ = session
	if !ok {
		return
	}
	store, ok := s.runtime.Operations.(*operations.PostgresStore)
	if !ok {
		writeProblem(w, 503, "workflow_unavailable", "Financial details are unavailable")
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeProblem(w, 400, "reference_required", "Enter the obligation or credit request reference")
		return
	}
	id, supplierOrgID, err := s.financialChangeIdentity(r, user.ID, q, false)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeProblem(w, 404, "reference_not_found", "No obligation was found for that reference")
		} else {
			writeProblem(w, 503, "workflow_unavailable", "Financial details could not be loaded. Try again.")
		}
		return
	}
	reqCtx := r.Context()
	if supplierOrgID != "" {
		reqCtx = db.WithTenantContext(reqCtx, user.ID, supplierOrgID)
	}
	v, err := store.ChangeContext(reqCtx, id)
	if err != nil {
		policyFailure(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"obligation_id": id, "snapshot": v})
}
func (s *Server) previewBusinessPolicy(w http.ResponseWriter, r *http.Request) {
	_, _, _, ok := s.requirePlatformAccess(w, r, access.PermissionManagePolicies)
	if !ok {
		return
	}
	if s.runtime.BusinessPolicies == nil {
		writeProblem(w, 503, "policy_unavailable", "Business settings are unavailable. Try again.")
		return
	}
	var in struct {
		Values       businesspolicy.Values `json:"values"`
		BaseRevision int64                 `json:"base_revision"`
	}
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	if err := in.Values.ValidateDeployment(s.config); err != nil {
		policyFailure(w, err)
		return
	}
	current, err := s.runtime.BusinessPolicies.Read(r.Context())
	if err != nil {
		policyFailure(w, err)
		return
	}
	if current.Revision != in.BaseRevision {
		writeProblem(w, 409, "stale_policy", "Refresh the policy before previewing")
		return
	}
	// Aggregate-only impact estimates use the complete persisted population. They
	// describe today's workload, not a guarantee of future collection success.
	var counts []byte
	err = s.runtime.Database.Raw().QueryRow(r.Context(), `SELECT app.admin_policy_impact($1::jsonb)`, policyJSON(in.Values)).Scan(&counts)
	if err != nil {
		writeProblem(w, 503, "preview_unavailable", "Policy impact could not be calculated")
		return
	}
	effects := []string{}
	for _, f := range businesspolicy.Catalog() {
		old := policyMap(current.Values)[f.Key]
		next := policyMap(in.Values)[f.Key]
		if old == next {
			continue
		}
		text := f.Help
		if f.Key == "base_fee_bps" || f.Key == "collection_fee_bps" {
			text = "Applies to offers created after the effective time. Existing offers and agreements retain their recorded rates."
		}
		effects = append(effects, f.Label+": "+text)
	}
	writeJSON(w, 200, map[string]any{"base_revision": current.Revision, "counts": json.RawMessage(counts), "effects": effects, "note": "Counts reflect current records. Eligibility, delivered notices, mandates and deployment limits are rechecked when work runs."})
}
func policyJSON(v any) []byte { b, _ := json.Marshal(v); return b }
func policyMap(v any) map[string]any {
	m := map[string]any{}
	_ = json.Unmarshal(policyJSON(v), &m)
	return m
}
func (s *Server) adminAttention(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, "")
	if !ok {
		return
	}
	kinds, err := s.reviewKinds(r, user.ID)
	if err != nil {
		policyFailure(w, err)
		return
	}
	var metrics []byte
	err = s.runtime.Database.Raw().QueryRow(r.Context(), `SELECT app.admin_attention($1::uuid,$2::text[])`, user.ID, kinds).Scan(&metrics)
	if err != nil {
		writeProblem(w, 503, "attention_unavailable", "Work needing attention could not be loaded")
		return
	}
	writeJSON(w, 200, map[string]any{"items": json.RawMessage(metrics)})
}

func (s *Server) adminAttentionDetails(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := s.requirePlatformAccess(w, r, access.PermissionProviderOperations); !ok {
		return
	}
	var b []byte
	if err := s.runtime.Database.Raw().QueryRow(r.Context(), `SELECT app.admin_attention_details()`).Scan(&b); err != nil {
		writeProblem(w, 503, "attention_unavailable", "Details could not be loaded")
		return
	}
	writeJSON(w, 200, json.RawMessage(b))
}
