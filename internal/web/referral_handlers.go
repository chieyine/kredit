package web

import (
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"kredit/internal/access"
	"kredit/internal/referrals"
	"net/http"
	"strings"
)

func (s *Server) dsa(w http.ResponseWriter, r *http.Request) {
	if s.runtime.Database == nil {
		writeProblem(w, 503, "dsa_unavailable", "The referral programme is unavailable.")
		return
	}
	store := &referrals.Store{Pool: s.runtime.Database.Raw()}
	if code := r.PathValue("code"); code != "" {
		if len(code) > 40 {
			writeProblem(w, 400, "invalid_code", "Invalid referral code.")
			return
		}
		v, e := store.Lookup(r.Context(), code)
		if e != nil {
			writeProblem(w, 404, "code_unavailable", "This referral code is unavailable.")
			return
		}
		writeJSON(w, 200, v)
		return
	}
	session, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	admin := strings.HasPrefix(r.URL.Path, "/api/v1/ops/")
	if admin {
		if _, _, _, ok = s.requirePlatformAccess(w, r, access.PermissionPlatformOwner); !ok {
			return
		}
	}
	if r.Method == "GET" {
		agent := r.URL.Query().Get("agent")
		cursors := map[string]string{}
		for _, k := range []string{"agents", "referrals", "earnings", "payouts"} {
			v := r.URL.Query().Get(k + "_before")
			if v != "" {
				if _, e := uuid.Parse(v); e != nil {
					writeProblem(w, 400, "invalid_cursor", "Invalid page reference.")
					return
				}
			}
			cursors[k] = v
		}
		if agent != "" {
			if _, e := uuid.Parse(agent); e != nil {
				writeProblem(w, 400, "invalid_agent", "Invalid agent.")
				return
			}
		}
		v, e := store.Read(r.Context(), user.ID, admin, agent, cursors)
		if e != nil {
			writeProblem(w, 503, "dsa_unavailable", "Could not load referral records.")
			return
		}
		writeJSON(w, 200, v)
		return
	}
	if !s.requireCSRF(w, r) || !s.requireFreshMFA(w, session) {
		return
	}
	var in referrals.Input
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	if in.ID != "" {
		if _, e := uuid.Parse(in.ID); e != nil {
			writeProblem(w, 400, "invalid_reference", "Invalid record reference.")
			return
		}
	}
	if org := r.PathValue("organizationID"); org != "" {
		if _, e := uuid.Parse(org); e != nil {
			writeProblem(w, 400, "invalid_business", "Invalid business.")
			return
		}
		if !in.Consent {
			writeProblem(w, 422, "consent_required", "Confirm that this agent introduced your business to Kredit.")
			return
		}
		if e := store.Claim(r.Context(), user.ID, org, in.Code); e != nil {
			writeProblem(w, 409, "referral_unconfirmed", e.Error())
			return
		}
		writeJSON(w, 200, map[string]bool{"saved": true})
		return
	}
	result, e := store.Act(r.Context(), user.ID, admin, in)
	if e != nil {
		detail := e.Error()
		var databaseError *pgconn.PgError
		if errors.As(e, &databaseError) {
			detail = "The action could not be confirmed. Refresh its record and retry."
			if databaseError.Code == "23505" {
				detail = "This bank reference or pending payout already exists. Refresh its record."
			}
		}
		writeProblem(w, 409, "dsa_action_unconfirmed", detail)
		return
	}
	writeJSON(w, 200, result)
}
