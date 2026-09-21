package web

import (
	"errors"
	"kredit/internal/access"
	"kredit/internal/networkops"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Server) networkOperations(w http.ResponseWriter, r *http.Request) {
	org, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", "Choose a valid business.")
		return
	}
	permission := access.PermissionReadOrganization
	if r.Method != "GET" {
		permission = access.PermissionManageOrganization
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, org, permission)
	if !ok {
		return
	}
	if r.Method != "GET" && !s.requireCSRF(w, r) {
		return
	}
	if s.runtime.Database == nil {
		writeProblem(w, 503, "network_unavailable", "Business records require the database.")
		return
	}
	store := networkops.Store{Pool: s.runtime.Database.Raw()}
	fail := func(err error) {
		var pg *pgconn.PgError
		switch {
		case errors.Is(err, networkops.ErrAuthority):
			writeProblem(w, 403, "network_authority_required", "Current business management authority is required.")
		case errors.Is(err, networkops.ErrConflict):
			writeProblem(w, 409, "network_changed", "Refresh the record before making another change.")
		case errors.Is(err, networkops.ErrInvalid), errors.As(err, &pg) && (pg.Code == "23514" || pg.Code == "23503" || pg.Code == "22P02"):
			writeProblem(w, 422, "invalid_network_record", "Check the branch, customer and current account manager.")
		case errors.As(err, &pg) && pg.Code == "23505":
			writeProblem(w, 409, "network_changed", "That branch name already exists. Refresh the records.")
		default:
			writeProblem(w, 503, "network_unavailable", "The result could not be confirmed. Refresh before retrying.")
		}
	}
	if r.PathValue("memberID") != "" {
		id, e := pathID(r, "memberID")
		if e != nil {
			fail(networkops.ErrInvalid)
			return
		}
		var in struct {
			Mode      string    `json:"mode"`
			BranchIDs *[]string `json:"branch_ids"`
			Version   *int64    `json:"version"`
		}
		if e = decodeJSON(w, r, &in); e != nil {
			writeProblem(w, 400, "invalid_request", e.Error())
			return
		}
		if in.Version == nil || in.BranchIDs == nil {
			fail(networkops.ErrInvalid)
			return
		}
		for _, bid := range *in.BranchIDs {
			if _, e = uuid.Parse(bid); e != nil {
				fail(networkops.ErrInvalid)
				return
			}
		}
		out, e := store.SaveScope(r.Context(), user.ID, org, networkops.Scope{UserID: id, Mode: in.Mode, BranchIDs: *in.BranchIDs, Version: *in.Version})
		if e != nil {
			fail(e)
			return
		}
		writeJSON(w, 200, out)
		return
	}
	if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/branch-access") {
		out, e := store.ReadScopes(r.Context(), user.ID, org)
		if e != nil {
			fail(e)
			return
		}
		writeJSON(w, 200, out)
		return
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
	if r.PathValue("branchID") != "" {
		id, e := pathID(r, "branchID")
		if e != nil {
			writeProblem(w, 400, "invalid_branch", "Choose a valid branch.")
			return
		}
		var in struct {
			Name      string `json:"name"`
			Territory string `json:"territory"`
			Active    *bool  `json:"active"`
			Version   *int64 `json:"version"`
		}
		if e = decodeJSON(w, r, &in); e != nil {
			writeProblem(w, 400, "invalid_request", e.Error())
			return
		}
		if in.Active == nil || in.Version == nil {
			fail(networkops.ErrInvalid)
			return
		}
		out, e := store.SaveBranch(r.Context(), user.ID, org, networkops.Branch{ID: id, Name: in.Name, Territory: in.Territory, Active: *in.Active, Version: *in.Version})
		if e != nil {
			fail(e)
			return
		}
		writeJSON(w, 200, out)
		return
	}
	id, e := pathID(r, "businessID")
	if e != nil {
		writeProblem(w, 400, "invalid_customer", "Choose a valid customer.")
		return
	}
	var in struct {
		BranchID  *string `json:"branch_id"`
		ManagerID *string `json:"manager_id"`
		Version   *int64  `json:"version"`
	}
	if e = decodeJSON(w, r, &in); e != nil {
		writeProblem(w, 400, "invalid_request", e.Error())
		return
	}
	if in.BranchID == nil || in.ManagerID == nil || in.Version == nil {
		fail(networkops.ErrInvalid)
		return
	}
	for _, value := range []string{*in.BranchID, *in.ManagerID} {
		if value != "" {
			if _, e = uuid.Parse(value); e != nil {
				fail(networkops.ErrInvalid)
				return
			}
		}
	}
	out, e := store.Assign(r.Context(), user.ID, org, networkops.Partner{BusinessID: id, BranchID: *in.BranchID, ManagerID: *in.ManagerID, Version: *in.Version})
	if e != nil {
		fail(e)
		return
	}
	writeJSON(w, 200, out)
}
