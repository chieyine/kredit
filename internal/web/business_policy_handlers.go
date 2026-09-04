package web

import (
	"encoding/json"
	"errors"
	"net/http"

	"kredit/internal/access"
	"kredit/internal/businesspolicy"

	"github.com/jackc/pgx/v5/pgconn"
)

func policyFailure(w http.ResponseWriter, err error) {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		writeProblem(w, 409, "policy_conflict", "The change could not be applied. Refresh the settings and try again.")
		return
	}
	writeProblem(w, 409, "policy_conflict", err.Error())
}
func (s *Server) businessPolicies(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionManagePolicies)
	if !ok {
		return
	}
	if s.runtime.BusinessPolicies == nil {
		writeProblem(w, 503, "policy_unavailable", "Business policies require database persistence")
		return
	}
	current, err := s.runtime.BusinessPolicies.Read(r.Context())
	if err != nil {
		writeProblem(w, 503, "policy_unavailable", "Business policies could not be loaded")
		return
	}
	changes, events, err := s.runtime.BusinessPolicies.History(r.Context())
	if err != nil {
		writeProblem(w, 503, "policy_unavailable", "Policy history could not be loaded")
		return
	}
	var actors []byte
	if err = s.runtime.Database.Raw().QueryRow(r.Context(), `WITH visible AS (SELECT id,proposed_by,decided_by FROM app.business_policy_changes ORDER BY revision DESC LIMIT 100) SELECT COALESCE(jsonb_object_agg(id,app.admin_actor_name(id)),'{}'::jsonb) FROM (SELECT proposed_by id FROM visible UNION SELECT decided_by FROM visible WHERE decided_by IS NOT NULL UNION SELECT actor_id FROM app.business_policy_events WHERE change_id IN(SELECT id FROM visible))a`).Scan(&actors); err != nil {
		writeProblem(w, 503, "policy_unavailable", "Administrator names could not be loaded")
		return
	}
	var canPropose, canApprove bool
	if err = s.runtime.Database.Raw().QueryRow(r.Context(), `SELECT app.has_admin_role($1::uuid,ARRAY['platform_admin','policy_manager']),app.has_admin_role($1::uuid,ARRAY['platform_admin','approver'])`, user.ID).Scan(&canPropose, &canApprove); err != nil {
		writeProblem(w, 503, "policy_unavailable", "Policy permissions could not be loaded")
		return
	}
	writeJSON(w, 200, map[string]any{"can_propose": canPropose, "can_approve": canApprove, "actors": json.RawMessage(actors), "current": current, "changes": changes, "events": events, "fields": businesspolicy.Catalog(), "actor_id": user.ID, "deployment_limits": businesspolicy.Defaults(s.config)})
}
func (s *Server) proposeBusinessPolicy(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionManagePolicies)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	if s.runtime.BusinessPolicies == nil {
		writeProblem(w, 503, "policy_unavailable", "Business policies require database persistence")
		return
	}
	var in businesspolicy.Proposal
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", "Provide valid complete settings, a reason, and an effective date")
		return
	}
	id, err := s.runtime.BusinessPolicies.Propose(r.Context(), user.ID, in)
	if err != nil {
		policyFailure(w, err)
		return
	}
	writeJSON(w, 201, map[string]string{"id": id, "state": "pending"})
}
func (s *Server) decideBusinessPolicy(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionManagePolicies)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	if s.runtime.BusinessPolicies == nil {
		writeProblem(w, 503, "policy_unavailable", "Business policies require database persistence")
		return
	}
	id, err := pathID(r, "changeID")
	if err != nil {
		writeProblem(w, 400, "invalid_request", "Invalid change identifier")
		return
	}
	var in struct {
		Action string `json:"action"`
		Reason string `json:"reason"`
	}
	if err = decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", "Provide a decision and reason")
		return
	}
	if err = s.runtime.BusinessPolicies.Decide(r.Context(), id, user.ID, in.Action, in.Reason); err != nil {
		policyFailure(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "recorded"})
}

func (s *Server) publicPricing(w http.ResponseWriter, r *http.Request) {
	values := businesspolicy.Defaults(s.config)
	var revision int64
	if s.runtime.BusinessPolicies != nil {
		snapshot, err := s.runtime.BusinessPolicies.Read(r.Context())
		if err != nil {
			writeProblem(w, 503, "pricing_unavailable", "Current pricing could not be loaded")
			return
		}
		values = snapshot.Values
		revision = snapshot.Revision
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"policy_revision": revision, "base_bps": values.BaseFeeBPS, "collection_bps": values.CollectionFeeBPS})
}
