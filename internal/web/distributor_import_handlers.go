package web

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/buyers"
)

// Batch commands use the same step-up and CSRF boundary as individual invites.
func (s *Server) distributorImports(w http.ResponseWriter, r *http.Request) {
	org, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, org, access.PermissionInviteBuyers)
	if !ok {
		return
	}
	if r.Method != "GET" && !s.requireCSRF(w, r) {
		return
	}
	service, ok := s.runtime.Buyers.(buyers.ImportBatchService)
	if !ok {
		writeProblem(w, 503, "imports_unavailable", "Saved imports require the business database.")
		return
	}
	fail := func(err error) {
		switch {
		case errors.Is(err, buyers.ErrImportInvalid):
			writeProblem(w, 422, "invalid_import", err.Error())
		case errors.Is(err, pgx.ErrNoRows):
			writeProblem(w, 404, "import_not_found", "This import is not available to your business.")
		case errors.Is(err, buyers.ErrImportAuthority):
			writeProblem(w, 403, "import_authority_required", "Your permission to manage imports has changed.")
		case errors.Is(err, buyers.ErrImportConflict):
			writeProblem(w, 409, "import_conflict", "This batch cannot perform that action. Refresh its progress before continuing.")
		default:
			writeProblem(w, 503, "import_unavailable", "The import could not be confirmed. Refresh before retrying; completed invitations are retained.")
		}
	}
	id := r.PathValue("batchID")
	if id != "" {
		if _, err = pathID(r, "batchID"); err != nil {
			writeProblem(w, 400, "invalid_import", err.Error())
			return
		}
	}
	if r.Method == "GET" {
		if id == "" {
			items, err := service.ListImports(r.Context(), user.ID, org)
			if err != nil {
				fail(err)
				return
			}
			writeJSON(w, 200, map[string]any{"batches": items})
			return
		}
		batch, err := service.ReadImport(r.Context(), user.ID, org, id)
		if err != nil {
			fail(err)
			return
		}
		writeJSON(w, 200, batch)
		return
	}
	if id == "" {
		var input struct {
			SourceHash string                 `json:"source_hash"`
			Contacts   []buyers.ImportContact `json:"contacts"`
		}
		if err = decodeJSON(w, r, &input); err != nil {
			writeProblem(w, 400, "invalid_import", err.Error())
			return
		}
		batch, err := service.SaveImport(r.Context(), user.ID, org, input.SourceHash, input.Contacts)
		if err != nil {
			fail(err)
			return
		}
		s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: org, Action: "distributor.import.saved", ResourceType: "distributor_import", ResourceID: batch.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context())})
		writeJSON(w, 201, batch)
		return
	}
	if row := r.PathValue("rowNumber"); row != "" {
		n, err := strconv.Atoi(row)
		if err != nil || n < 1 || n > 200 {
			writeProblem(w, 400, "invalid_row", "Select a row between 1 and 200.")
			return
		}
		result, contact, err := service.CreateImportInvitation(r.Context(), user.ID, org, id, n)
		if err != nil {
			fail(err)
			return
		}
		url := strings.TrimRight(s.config.PublicBaseURL, "/") + "/buyer-invitations/" + result.RawToken
		// Once committed, a retry recovers the link without sending twice. A process
		// crash during delivery leaves an explicit manual handoff, never a false Sent.
		delivery := "existing_invitation"
		if result.Invitation.Status == "accepted" {
			delivery = "accepted"
			url = ""
		} else if !result.Replayed {
			delivery = "sent"
			if err = s.runtime.Notifications.SendInvitation(r.Context(), contact.Target, contact.TargetType, url); err != nil {
				delivery = "manual_handoff_required"
			}
		}
		s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: org, Action: "distributor.import.invitation_created", ResourceType: "buyer_invitation", ResourceID: result.Invitation.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"batch_id": id, "row": row, "delivery_state": delivery}})
		writeJSON(w, 202, map[string]any{"invitation_url": url, "delivery_state": delivery})
		return
	}
	var input struct {
		Action string `json:"action"`
	}
	if err = decodeJSON(w, r, &input); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	batch, err := service.TransitionImport(r.Context(), user.ID, org, id, input.Action)
	if err != nil {
		fail(err)
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: org, Action: "distributor.import." + input.Action, ResourceType: "distributor_import", ResourceID: id, Outcome: "success", RequestID: requestIDFromContext(r.Context())})
	writeJSON(w, 200, batch)
}
