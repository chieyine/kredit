package web

import (
	"context"
	"errors"
	"kredit/internal/access"
	"kredit/internal/buyers"
	"kredit/internal/legalpublication"
	"kredit/internal/purchasing"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Server) purchasingAuthority(w http.ResponseWriter, r *http.Request) {
	org, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", "Choose a valid business.")
		return
	}
	permission := access.PermissionReadOrganization
	if r.Method != "GET" {
		permission = access.PermissionManageMembers
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, org, permission)
	if !ok {
		return
	}
	if r.Method != "GET" && !s.requireCSRF(w, r) {
		return
	}
	if s.runtime.Database == nil {
		writeProblem(w, 503, "purchasing_unavailable", "Purchasing permissions require the database.")
		return
	}
	store := purchasing.Store{Pool: s.runtime.Database.Raw()}
	fail := func(err error) {
		var pg *pgconn.PgError
		switch {
		case errors.Is(err, purchasing.ErrAuthority), errors.As(err, &pg) && pg.Code == "42501":
			writeProblem(w, 403, "purchasing_authority_required", "Only a current business owner can change purchasing permissions.")
		case errors.Is(err, purchasing.ErrConflict):
			writeProblem(w, 409, "purchasing_changed", "Permissions changed. Refresh before saving again.")
		case errors.Is(err, purchasing.ErrInvalid), errors.As(err, &pg) && (pg.Code == "23514" || pg.Code == "23503"):
			writeProblem(w, 422, "invalid_purchasing_permission", "Choose current staff, valid permissions, a non-negative limit and an expiry within one year.")
		default:
			writeProblem(w, 503, "purchasing_unavailable", "The result could not be confirmed. Refresh before retrying.")
		}
	}
	if r.Method == "GET" {
		out, e := store.Read(r.Context(), user.ID, org)
		if e != nil {
			fail(e)
			return
		}
		writeJSON(w, 200, out)
		return
	}
	target, e := pathID(r, "memberID")
	if e != nil {
		writeProblem(w, 400, "invalid_member", "Choose a valid team member.")
		return
	}
	var in struct {
		Actions         *[]string `json:"actions"`
		Ceiling         *int64    `json:"ceiling_kobo"`
		DrawdownCeiling *int64    `json:"drawdown_ceiling_kobo"`
		Expires         time.Time `json:"expires_at"`
		Version         *int64    `json:"version"`
	}
	if e = decodeJSON(w, r, &in); e != nil {
		writeProblem(w, 400, "invalid_request", e.Error())
		return
	}
	if in.Actions == nil || in.Ceiling == nil || in.Version == nil {
		fail(purchasing.ErrInvalid)
		return
	}
	var drawdownCeiling int64
	if in.DrawdownCeiling != nil {
		drawdownCeiling = *in.DrawdownCeiling
	}
	out, e := store.Save(r.Context(), user.ID, org, purchasing.Grant{
		UserID:              target,
		Actions:             *in.Actions,
		CeilingKobo:         *in.Ceiling,
		DrawdownCeilingKobo: drawdownCeiling,
		ExpiresAt:           in.Expires,
		Version:             *in.Version,
	})
	if e != nil {
		fail(e)
		return
	}
	writeJSON(w, 200, out)
}

func (s *Server) purchasingStaffSetup(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if r.Method != "GET" && !s.requireCSRF(w, r) {
		return
	}
	versions, err := legalpublication.SettingsReader(s.runtime.PlatformSettings)()
	if err != nil {
		writeProblem(w, 503, "notices_unavailable", "Current notices are unavailable.")
		return
	}
	if r.Method == "GET" {
		profiles, e := s.runtime.Buyers.ListBusinessProfiles(r.Context(), user.ID)
		if e != nil {
			writeProblem(w, 503, "purchasing_unavailable", "Your purchasing access could not be checked.")
			return
		}
		writeJSON(w, 200, map[string]any{"businesses": profiles, "legal_versions": versions, "identity_notice": buyers.IdentityNotice, "identity_notice_version": buyers.IdentityNoticeVersion})
		return
	}
	var in struct {
		BusinessID string `json:"business_id"`
		FullName   string `json:"full_name"`
		Consents   bool   `json:"consents_accepted"`
		Terms      string `json:"terms_version"`
		Privacy    string `json:"privacy_version"`
		Identity   string `json:"identity_notice_version"`
	}
	if err = decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	if _, err = uuid.Parse(in.BusinessID); err != nil {
		writeProblem(w, 400, "invalid_business", "Choose a valid business.")
		return
	}
	if !in.Consents || in.Terms != versions.Terms || in.Privacy != versions.Privacy || in.Identity != buyers.IdentityNoticeVersion {
		writeProblem(w, 422, "current_notices_required", "Review and accept the current notices.")
		return
	}
	store, ok := s.runtime.Buyers.(interface {
		EnrollPurchasingStaff(context.Context, string, string, buyers.AcceptInput) (buyers.Portal, error)
	})
	if !ok {
		writeProblem(w, 503, "purchasing_unavailable", "Staff setup requires the database.")
		return
	}
	portal, err := store.EnrollPurchasingStaff(r.Context(), user.ID, in.BusinessID, buyers.AcceptInput{FullName: in.FullName, ConsentsAccepted: in.Consents, TermsVersion: in.Terms, PrivacyVersion: in.Privacy, IdentityNoticeVersion: in.Identity})
	if err != nil {
		writeProblem(w, 422, "purchasing_setup_failed", "Setup could not be completed. Check that your purchasing access is current, then refresh and try again.")
		return
	}
	writeJSON(w, 200, map[string]any{"portal": portal})
}
