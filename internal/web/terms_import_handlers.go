package web

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/buyers"
)

func (s *Server) getTermsImportService() buyers.TermsImportService {
	if s.runtime.Database != nil {
		return buyers.NewPostgresTermsImportStore(s.runtime.Database.Raw(), s.runtime.Ledger)
	}
	return buyers.NewMemoryTermsImportStore(s.runtime.Ledger)
}

func (s *Server) termsImports(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", err.Error())
		return
	}
	permission := access.PermissionInviteBuyers
	if r.Method != "GET" && r.PathValue("decision") != "" {
		permission = access.PermissionApproveBusinessCredit
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, permission)
	if !ok {
		return
	}
	if r.Method != "GET" && !s.requireCSRF(w, r) {
		return
	}

	svc := s.getTermsImportService()
	fail := func(err error) {
		switch {
		case errors.Is(err, buyers.ErrTermsDualControl):
			writeProblem(w, 403, "dual_control_required", "Maker-checker separation required: the person who uploaded this batch cannot approve it.")
		case errors.Is(err, buyers.ErrTermsAlreadyClosed):
			writeProblem(w, 409, "batch_already_closed", "This terms import batch has already been decided.")
		case errors.Is(err, buyers.ErrTermsBatchNotFound), errors.Is(err, pgx.ErrNoRows):
			writeProblem(w, 404, "batch_not_found", "Terms import batch not found.")
		case errors.Is(err, buyers.ErrTermsInvalidRow):
			writeProblem(w, 422, "invalid_row", err.Error())
		default:
			writeProblem(w, 500, "terms_import_failed", err.Error())
		}
	}

	batchID := r.PathValue("batchID")
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

		hash := sha256.Sum256([]byte(r.Header.Get("Idempotency-Key") + string(rune(len(in.Rows)))))
		sourceHash := hex.EncodeToString(hash[:])

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
