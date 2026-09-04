package web

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kredit/internal/access"
	"kredit/internal/operations"

	"github.com/google/uuid"
)

func (s *Server) adminCapabilities(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, "")
	if !ok {
		return
	}
	roles, err := s.adminRoles(r, user.ID)
	if err != nil {
		writeProblem(w, 503, "access_unavailable", "Admin permissions could not be loaded")
		return
	}
	permissions := map[access.Permission]bool{}
	for _, p := range []access.Permission{access.PermissionReadFinancial, access.PermissionManagePolicies, access.PermissionApproveChanges, access.PermissionAdminFinancial, access.PermissionManageAccess, access.PermissionSupportSearch, access.PermissionManageCases, access.PermissionReviewCompliance, access.PermissionReviewDisputes, access.PermissionProviderOperations, access.PermissionOperateJobs, access.PermissionOperateCollections, access.PermissionSuspendAccounts, access.PermissionRecoverAccounts, access.PermissionReviewPrivacy, access.PermissionBreakGlass} {
		for _, role := range roles {
			permissions[p] = permissions[p] || access.CanPlatform(role, p)
		}
	}
	writeJSON(w, 200, map[string]any{"actor_id": user.ID, "roles": roles, "permissions": permissions})
}
func (s *Server) adminRoles(r *http.Request, actor string) ([]access.PlatformRole, error) {
	rows, err := s.runtime.Database.Raw().Query(r.Context(), `SELECT role FROM app.platform_role_assignments WHERE user_id=$1::uuid AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at>now())`, actor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []access.PlatformRole{}
	for rows.Next() {
		var v access.PlatformRole
		if err = rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Server) reviewKinds(r *http.Request, actor string) ([]string, error) {
	roles, err := s.adminRoles(r, actor)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for kind, p := range map[string]access.Permission{"support": access.PermissionManageCases, "policy": access.PermissionManagePolicies, "financial_change": access.PermissionAdminFinancial, "dispute": access.PermissionReviewDisputes, "financial_review": access.PermissionProviderOperations, "recovery": access.PermissionRecoverAccounts, "privacy": access.PermissionReviewPrivacy} {
		for _, role := range roles {
			if access.CanPlatform(role, p) {
				out = append(out, kind)
				break
			}
		}
	}
	return out, nil
}
func (s *Server) adminInbox(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, "")
	if !ok {
		return
	}
	kinds, err := s.reviewKinds(r, user.ID)
	if err != nil {
		writeProblem(w, 503, "inbox_unavailable", "Inbox could not be loaded")
		return
	}
	var result []byte
	err = s.runtime.Database.Raw().QueryRow(r.Context(), `SELECT COALESCE(jsonb_agg(item ORDER BY due_at,id,kind),'[]'::jsonb) FROM (SELECT q.id,q.kind,COALESCE(a.due_at,q.due_at) due_at,to_jsonb(q)||jsonb_build_object('owner_id',a.owner_id,'owner',CASE WHEN a.owner_id IS NOT NULL THEN app.admin_actor_name(a.owner_id) END,'due_at',COALESCE(a.due_at,q.due_at),'author',app.admin_actor_name(q.author_id)) item FROM app.admin_review_queue q LEFT JOIN app.admin_review_assignments a ON a.kind=q.kind AND a.resource_id=q.id WHERE q.kind=ANY($1::text[]) AND ($2='' OR q.title ILIKE '%'||$2||'%' OR q.id::text=$2) ORDER BY COALESCE(a.due_at,q.due_at),q.id,q.kind LIMIT 201 OFFSET $3) page`, kinds, strings.TrimSpace(r.URL.Query().Get("q")), adminOffset(r)).Scan(&result)
	if err != nil {
		writeProblem(w, 503, "inbox_unavailable", "Inbox could not be loaded")
		return
	}
	writeJSON(w, 200, map[string]any{"items": json.RawMessage(result), "actor_id": user.ID})
}
func adminOffset(r *http.Request) int {
	v, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if v < 0 || v > 1000000 {
		return 0
	}
	return v
}
func (s *Server) assignAdminReview(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, "")
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	var in struct {
		Kind    string    `json:"kind"`
		ID      string    `json:"id"`
		OwnerID string    `json:"owner_id"`
		DueAt   time.Time `json:"due_at"`
		Reason  string    `json:"reason"`
	}
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	if _, err := uuid.Parse(in.ID); err != nil {
		writeProblem(w, 400, "invalid_review", "Choose an existing review")
		return
	}
	if in.OwnerID != "" {
		if _, err := uuid.Parse(in.OwnerID); err != nil {
			writeProblem(w, 400, "invalid_owner", "Choose a reviewer")
			return
		}
	}
	kinds, err := s.reviewKinds(r, user.ID)
	if err != nil {
		writeProblem(w, 503, "inbox_unavailable", "Permissions could not be checked")
		return
	}
	allowed := false
	for _, k := range kinds {
		allowed = allowed || k == in.Kind
	}
	if !allowed {
		writeProblem(w, 403, "review_forbidden", "This review is outside your role")
		return
	}
	if len(strings.TrimSpace(in.Reason)) < 8 || len(in.Reason) > 2000 || in.DueAt.IsZero() || in.DueAt.After(time.Now().AddDate(1, 0, 0)) {
		writeProblem(w, 400, "invalid_assignment", "Provide a deadline within one year and a reason of 8 to 2000 characters")
		return
	}
	tx, err := s.runtime.Database.Raw().Begin(r.Context())
	if err != nil {
		policyFailure(w, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	var exists bool
	if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM app.admin_review_queue WHERE kind=$1 AND id=$2::uuid)`, in.Kind, in.ID).Scan(&exists); err != nil || !exists {
		writeProblem(w, 409, "review_closed", "This review is no longer open")
		return
	}
	if in.OwnerID != "" {
		roles := map[string][]string{"support": {"platform_admin", "support_agent"}, "policy": {"platform_admin", "policy_manager", "approver"}, "financial_change": {"platform_admin", "finance_operator", "approver"}, "dispute": {"platform_admin", "dispute_reviewer"}, "financial_review": {"platform_admin", "compliance_reviewer"}, "recovery": {"platform_admin", "compliance_reviewer"}, "privacy": {"platform_admin", "compliance_reviewer"}}[in.Kind]
		if err = tx.QueryRow(r.Context(), `SELECT app.has_admin_role($1::uuid,$2::text[])`, in.OwnerID, roles).Scan(&exists); err != nil || !exists {
			writeProblem(w, 409, "invalid_owner", "The owner needs active access to this review")
			return
		}
	}
	if _, err = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,645))`, in.Kind+in.ID); err != nil {
		policyFailure(w, err)
		return
	}
	var before []byte
	if err = tx.QueryRow(r.Context(), `SELECT COALESCE((SELECT to_jsonb(a) FROM app.admin_review_assignments a WHERE kind=$1 AND resource_id=$2::uuid),'{}'::jsonb)`, in.Kind, in.ID).Scan(&before); err != nil {
		policyFailure(w, err)
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO app.admin_review_assignments(kind,resource_id,owner_id,due_at) VALUES($1,$2::uuid,NULLIF($3,'')::uuid,$4) ON CONFLICT(kind,resource_id) DO UPDATE SET owner_id=EXCLUDED.owner_id,due_at=EXCLUDED.due_at,updated_at=now()`, in.Kind, in.ID, in.OwnerID, in.DueAt)
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO app.admin_assignment_events(kind,resource_id,actor_id,before_values,after_values,reason) SELECT kind,resource_id,$3::uuid,$4::jsonb,to_jsonb(a),$5 FROM app.admin_review_assignments a WHERE kind=$1 AND resource_id=$2::uuid`, in.Kind, in.ID, user.ID, before, strings.TrimSpace(in.Reason))
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		policyFailure(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"saved": true})
}
func (s *Server) proposeAdminChange(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionAdminFinancial)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	store, ok := s.runtime.Operations.(*operations.PostgresStore)
	if !ok {
		writeProblem(w, 503, "workflow_unavailable", "Persistent financial workflows are unavailable")
		return
	}
	var in operations.ChangeProposal
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	if err := store.ProposeChange(r.Context(), user.ID, in); err != nil {
		policyFailure(w, err)
		return
	}
	writeJSON(w, 201, map[string]string{"id": in.ID, "state": "pending"})
}
func (s *Server) decideAdminChange(w http.ResponseWriter, r *http.Request) {
	s.decideFinancialChange(w, r, false)
}
func (s *Server) decideBuyerChange(w http.ResponseWriter, r *http.Request) {
	s.decideFinancialChange(w, r, true)
}
func (s *Server) decideFinancialChange(w http.ResponseWriter, r *http.Request, buyer bool) {
	session, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !buyer {
		if _, _, _, ok = s.requirePlatformAccess(w, r, access.PermissionAdminFinancial); !ok {
			return
		}
	}
	if !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	store, ok := s.runtime.Operations.(*operations.PostgresStore)
	if !ok {
		writeProblem(w, 503, "workflow_unavailable", "Persistent financial workflows are unavailable")
		return
	}
	var in struct {
		Action string `json:"action"`
		Reason string `json:"reason"`
	}
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	if err := store.DecideChange(r.Context(), r.PathValue("changeID"), user.ID, in.Action, in.Reason, buyer); err != nil {
		policyFailure(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"recorded": true})
}
func (s *Server) adminChanges(w http.ResponseWriter, r *http.Request) {
	_, _, _, ok := s.requirePlatformAccess(w, r, access.PermissionAdminFinancial)
	if !ok {
		return
	}
	var result []byte
	err := s.runtime.Database.Raw().QueryRow(r.Context(), `SELECT COALESCE(jsonb_agg(to_jsonb(c)||jsonb_build_object('proposer',app.admin_actor_name(c.proposed_by),'approver',CASE WHEN c.approved_by IS NOT NULL THEN app.admin_actor_name(c.approved_by) END) ORDER BY c.created_at DESC),'[]'::jsonb) FROM (SELECT * FROM app.admin_change_requests WHERE ($1='' OR id::text=$1 OR obligation_id::text=$1) ORDER BY created_at DESC,id DESC LIMIT 101 OFFSET $2)c`, r.URL.Query().Get("q"), adminOffset(r)).Scan(&result)
	if err != nil {
		writeProblem(w, 503, "workflow_unavailable", "Financial changes could not be loaded")
		return
	}
	writeJSON(w, 200, map[string]any{"changes": json.RawMessage(result)})
}
func (s *Server) buyerChanges(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if s.runtime.Database == nil {
		writeProblem(w, 503, "workflow_unavailable", "Changes could not be loaded")
		return
	}
	var result []byte
	err := s.runtime.Database.Raw().QueryRow(r.Context(), `SELECT COALESCE(jsonb_agg(jsonb_build_object('id',c.id,'obligation_id',c.obligation_id,'state',c.state,'reason',c.reason,'expires_at',c.expires_at,'items',c.before_values->'items','dates',c.proposed_values->'dates') ORDER BY c.created_at DESC),'[]'::jsonb) FROM (SELECT * FROM app.admin_change_requests WHERE buyer_id=$1::uuid AND kind='schedule_amendment' AND state<>'pending' ORDER BY created_at DESC,id DESC LIMIT 101 OFFSET $2)c`, user.ID, adminOffset(r)).Scan(&result)
	if err != nil {
		writeProblem(w, 503, "workflow_unavailable", "Changes could not be loaded")
		return
	}
	writeJSON(w, 200, map[string]any{"changes": json.RawMessage(result)})
}
func (s *Server) adminHistory(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, "")
	if !ok {
		return
	}
	kinds, err := s.reviewKinds(r, user.ID)
	if err != nil {
		policyFailure(w, err)
		return
	}
	allowed := []string{}
	for _, k := range kinds {
		if k == "policy" {
			allowed = append(allowed, k)
		}
		if k == "financial_change" {
			allowed = append(allowed, "write_off", "fee_waiver", "schedule_amendment")
		}
	}
	if len(kinds) > 0 {
		allowed = append(allowed, "assignment")
	}
	var result []byte
	err = s.runtime.Database.Raw().QueryRow(r.Context(), `SELECT COALESCE(jsonb_agg(to_jsonb(h) ORDER BY h.created_at DESC,h.id DESC),'[]'::jsonb) FROM (SELECT * FROM app.admin_change_history WHERE kind=ANY($1::text[]) AND ($2='' OR reason ILIKE '%'||$2||'%' OR proposer ILIKE '%'||$2||'%' OR approver ILIKE '%'||$2||'%' OR id::text=$2) AND ($3='' OR kind=$3) AND (kind<>'assignment' OR after_values->>'kind'=ANY($5::text[])) ORDER BY created_at DESC,id DESC LIMIT 101 OFFSET $4)h`, allowed, strings.TrimSpace(r.URL.Query().Get("q")), r.URL.Query().Get("kind"), adminOffset(r), kinds).Scan(&result)
	if err != nil {
		writeProblem(w, 503, "history_unavailable", "Change history could not be loaded")
		return
	}
	s.auditPlatformRead(r, user.ID, "admin.change_history.viewed", "admin_change", "")
	if r.URL.Query().Get("format") == "csv" {
		var rows []map[string]any
		decoder := json.NewDecoder(bytes.NewReader(result))
		decoder.UseNumber()
		if decoder.Decode(&rows) != nil {
			writeProblem(w, 503, "history_unavailable", "History could not be exported")
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="admin-change-history.csv"`)
		cw := csv.NewWriter(w)
		fields := []string{"id", "kind", "created_at", "state", "reason", "proposer", "approver", "before_values", "after_values", "events"}
		_ = cw.Write(fields)
		for _, row := range rows[:min(len(rows), 100)] {
			cells := []string{}
			for _, key := range fields {
				value := row[key]
				cell := fmt.Sprint(value)
				if value == nil {
					cell = ""
				}
				if key == "before_values" || key == "after_values" || key == "events" {
					b, _ := json.Marshal(value)
					cell = string(b)
				}
				if strings.ContainsAny(strings.TrimLeft(cell, " \t\r\n")[:min(1, len(strings.TrimLeft(cell, " \t\r\n")))], "=+-@") {
					cell = "'" + cell
				}
				cells = append(cells, cell)
			}
			_ = cw.Write(cells)
		}
		cw.Flush()
		return
	}
	writeJSON(w, 200, map[string]any{"items": json.RawMessage(result)})
}
