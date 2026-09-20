package web

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/buyers"
)

func (s *Server) getTermsImportService() buyers.TermsImportService {
	s.initDomainServices()
	return s.domainServices.terms
}

func (s *Server) termsImports(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", err.Error())
		return
	}
	permission := termsImportPermission(r)
	_, user, membership, ok := s.requireOrganizationAccess(w, r, orgID, permission)
	if !ok {
		return
	}
	if !access.Can(membership.Role, access.PermissionInviteBuyers) && !access.Can(membership.Role, access.PermissionApproveBusinessCredit) {
		writeProblem(w, 403, "terms_import_forbidden", "You do not have access to terms imports.")
		return
	}
	if r.Method != "GET" && !s.requireCSRF(w, r) {
		return
	}

	svc := s.getTermsImportService()
	fail := func(err error) {
		switch {
		case errors.Is(err, buyers.ErrTermsAuthority):
			writeProblem(w, 403, "terms_import_forbidden", "Current business authority is required.")
		case errors.Is(err, buyers.ErrTermsDualControl):
			writeProblem(w, 403, "dual_control_required", "Maker-checker separation required: the person who uploaded this batch cannot approve it.")
		case errors.Is(err, buyers.ErrTermsAlreadyClosed):
			writeProblem(w, 409, "batch_already_closed", "This terms import batch has already been decided.")
		case errors.Is(err, buyers.ErrTermsBatchNotFound), errors.Is(err, pgx.ErrNoRows):
			writeProblem(w, 404, "batch_not_found", "Terms import batch not found.")
		case errors.Is(err, buyers.ErrTermsInvalidRow):
			writeProblem(w, 422, "invalid_row", err.Error())
		default:
			s.logger.ErrorContext(r.Context(), "terms import failed", "request_id", requestIDFromContext(r.Context()))
			writeProblem(w, 503, "terms_import_failed", "Terms imports are temporarily unavailable. Please retry.")
		}
	}

	batchID := r.PathValue("batchID")
	if batchID != "" {
		if _, err := pathID(r, "batchID"); err != nil {
			writeProblem(w, 400, "invalid_batch", "A valid batch identifier is required.")
			return
		}
	}
	if r.Method == "GET" {
		if batchID != "" {
			batch, rows, err := svc.GetTermsBatch(r.Context(), user.ID, orgID, batchID)
			if err != nil {
				fail(err)
				return
			}
			writeJSON(w, 200, map[string]any{
				"batch": batch,
				"rows":  rows,
			})
			return
		}
		batches, err := svc.ListTermsBatches(r.Context(), user.ID, orgID)
		if err != nil {
			fail(err)
			return
		}
		writeJSON(w, 200, map[string]any{
			"batches": batches,
		})
		return
	}

	if r.Method == "POST" {
		if batchID != "" {
			var in struct {
				Decision string `json:"decision"`
			}
			if err := decodeJSON(w, r, &in); err != nil {
				writeProblem(w, 400, "invalid_request", err.Error())
				return
			}
			decided, err := svc.ReviewTermsBatch(r.Context(), user.ID, orgID, batchID, in.Decision)
			if err != nil {
				fail(err)
				return
			}
			s.runtime.Audit.Append(audit.Event{
				ActorUserID:    user.ID,
				OrganizationID: orgID,
				Action:         "terms_import.batch." + in.Decision,
				ResourceType:   "partner_terms_import_batch",
				ResourceID:     batchID,
				Outcome:        "success",
				RequestID:      requestIDFromContext(r.Context()),
			})
			writeJSON(w, 200, decided)
			return
		}

		var in struct {
			Rows []buyers.TermsImportRowInput `json:"rows"`
		}
		if err := decodeJSON(w, r, &in); err != nil {
			writeProblem(w, 400, "invalid_request", err.Error())
			return
		}
		if len(in.Rows) == 0 {
			writeProblem(w, 422, "empty_batch", "Provide at least one row in the batch.")
			return
		}

		sourceHash, err := buyers.TermsSourceHash(in.Rows)
		if err != nil {
			fail(err)
			return
		}

		batch, err := svc.StageTermsBatch(r.Context(), user.ID, orgID, sourceHash, in.Rows)
		if err != nil {
			fail(err)
			return
		}
		s.runtime.Audit.Append(audit.Event{
			ActorUserID:    user.ID,
			OrganizationID: orgID,
			Action:         "terms_import.batch.staged",
			ResourceType:   "partner_terms_import_batch",
			ResourceID:     batch.ID,
			Outcome:        "success",
			RequestID:      requestIDFromContext(r.Context()),
		})
		writeJSON(w, 201, batch)
		return
	}
}

// A batch path identifies review. The decision is in JSON, not a route variable.
func termsImportPermission(r *http.Request) access.Permission {
	if r.Method == http.MethodGet {
		return access.PermissionReadOrganization
	}
	if r.PathValue("batchID") != "" {
		return access.PermissionApproveBusinessCredit
	}
	return access.PermissionInviteBuyers
}
