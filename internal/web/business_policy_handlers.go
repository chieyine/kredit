package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"kredit/internal/access"
	"kredit/internal/businesspolicy"

	"github.com/jackc/pgx/v5/pgconn"
)

func policyFailure(w http.ResponseWriter, err error) {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		writeProblem(w, 409, "policy_conflict", "We could not save that change. Refresh the page and try again.")
		return
	}
	writeProblem(w, 409, "policy_conflict", "This change could not be confirmed. Reload the current record and check your permissions and requested values before retrying.")
}

// monoAdminStatus exposes only non-secret deployment state. Credential values
// are intentionally never serialized into an admin response; operators only
// need to know whether the deployment has them and which readiness gates remain.
func (s *Server) monoAdminStatus() map[string]any {
	mode := "disabled"
	if s.config.MonoSweepEnabled {
		mode = "sandbox"
		if s.config.Environment == "production" {
			mode = "live"
		}
	}
	blockers := []string{}
	if s.config.CollectionProvider != "mono-sweep" {
		blockers = append(blockers, "Collection provider is not set to Mono Sweep")
	}
	if strings.TrimSpace(s.config.MonoSecretKey) == "" {
		blockers = append(blockers, "Mono secret key is not configured")
	}
	if strings.TrimSpace(s.config.MonoWebhookSecret) == "" {
		blockers = append(blockers, "Mono webhook secret is not configured")
	}
	if strings.TrimSpace(s.config.MonoRedirectURL) == "" {
		blockers = append(blockers, "Mono redirect URL is not configured")
	}
	if s.config.Environment == "production" && strings.TrimSpace(s.config.ProviderCertificationReference) == "" {
		blockers = append(blockers, "Provider certification evidence is not recorded")
	}
	return map[string]any{
		"provider":                         s.config.CollectionProvider,
		"environment":                      s.config.Environment,
		"mode":                             mode,
		"sweep_enabled":                    s.config.MonoSweepEnabled,
		"partial_sweep_enabled":            s.config.PartialSweepEnabled,
		"automatic_collection_enabled":     s.config.AutomaticCollectionEnabled,
		"automatic_retry_enabled":          s.config.AutomaticRetryEnabled,
		"secret_key_configured":            strings.TrimSpace(s.config.MonoSecretKey) != "",
		"webhook_secret_configured":        strings.TrimSpace(s.config.MonoWebhookSecret) != "",
		"redirect_url_configured":          strings.TrimSpace(s.config.MonoRedirectURL) != "",
		"redirect_url":                     s.config.MonoRedirectURL,
		"provider_certification_recorded":  strings.TrimSpace(s.config.ProviderCertificationReference) != "",
		"ready_for_configured_environment": len(blockers) == 0,
		"blockers":                         blockers,
	}
}

func (s *Server) businessPolicies(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionManagePolicies)
	if !ok {
		return
	}
	if s.runtime.BusinessPolicies == nil {
		writeProblem(w, 503, "policy_unavailable", "Business settings cannot be saved right now. Please try again shortly.")
		return
	}
	current, err := s.runtime.BusinessPolicies.Read(r.Context())
	if err != nil {
		writeProblem(w, 503, "policy_unavailable", "We could not open your business settings. Please try again.")
		return
	}
	changes, events, err := s.runtime.BusinessPolicies.History(r.Context())
	if err != nil {
		writeProblem(w, 503, "policy_unavailable", "We could not open the history of your settings. Please try again.")
		return
	}
	var actors []byte
	if err = s.runtime.Database.Raw().QueryRow(r.Context(), `WITH visible AS (SELECT id,proposed_by,decided_by FROM app.business_policy_changes ORDER BY revision DESC LIMIT 100) SELECT COALESCE(jsonb_object_agg(id,app.admin_actor_name(id)),'{}'::jsonb) FROM (SELECT proposed_by id FROM visible UNION SELECT decided_by FROM visible WHERE decided_by IS NOT NULL UNION SELECT actor_id FROM app.business_policy_events WHERE change_id IN(SELECT id FROM visible))a`).Scan(&actors); err != nil {
		writeProblem(w, 503, "policy_unavailable", "We could not load the administrator names. Please try again.")
		return
	}
	var canPropose, canApprove bool
	if err = s.runtime.Database.Raw().QueryRow(r.Context(), `SELECT app.has_admin_role($1::uuid,ARRAY['platform_owner','platform_admin','policy_manager']),app.has_admin_role($1::uuid,ARRAY['platform_owner','platform_admin','approver'])`, user.ID).Scan(&canPropose, &canApprove); err != nil {
		writeProblem(w, 503, "policy_unavailable", "We could not check who is allowed to change this. Please try again.")
		return
	}
	writeJSON(w, 200, map[string]any{"can_propose": canPropose, "can_approve": canApprove, "actors": json.RawMessage(actors), "current": current, "changes": changes, "events": events, "fields": businesspolicy.Catalog(), "actor_id": user.ID, "deployment_limits": businesspolicy.Defaults(s.config), "mono": s.monoAdminStatus()})
}

func (s *Server) proposeBusinessPolicy(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionManagePolicies)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	if s.runtime.BusinessPolicies == nil {
		writeProblem(w, 503, "policy_unavailable", "Business settings cannot be saved right now. Please try again shortly.")
		return
	}
	var in businesspolicy.Proposal
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", "Fill in every setting, say why, and choose the day it starts.")
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
		writeProblem(w, 503, "policy_unavailable", "Business settings cannot be saved right now. Please try again shortly.")
		return
	}
	id, err := pathID(r, "changeID")
	if err != nil {
		writeProblem(w, 400, "invalid_request", "That change reference is not valid.")
		return
	}
	var in struct {
		Action string `json:"action"`
		Reason string `json:"reason"`
	}
	if err = decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", "Choose a decision and say why.")
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
			writeProblem(w, 503, "pricing_unavailable", "We could not load today's rates. Please try again.")
			return
		}
		values = snapshot.Values
		revision = snapshot.Revision
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"policy_revision": revision, "base_bps": values.BaseFeeBPS, "collection_bps": values.CollectionFeeBPS})
}
