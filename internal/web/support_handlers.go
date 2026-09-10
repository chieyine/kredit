package web

import (
	"errors"
	"net/http"
	"strings"

	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/support"

	"github.com/jackc/pgx/v5"
)

type supportCaseRequest struct {
	SubjectType string `json:"subject_type"`
	SubjectID   string `json:"subject_id"`
	Message     string `json:"message"`
	BreakGlass  bool   `json:"break_glass"`
}

type supportTransitionRequest struct {
	State string `json:"state"`
	Note  string `json:"note"`
}

func (s *Server) openSupportCase(w http.ResponseWriter, r *http.Request) {
	organizationID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionManageOrganization)
	if !ok || !s.requireCSRF(w, r) {
		return
	}
	var input supportCaseRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if input.BreakGlass {
		writeProblem(w, http.StatusForbidden, "break_glass_forbidden", "You need an approved support role to open this.")
		return
	}
	note := strings.TrimSpace(input.Message)
	if len(note) > 2000 {
		writeProblem(w, http.StatusUnprocessableEntity, "support_message_invalid", "Your message is too long. Keep it to 2,000 characters or less.")
		return
	}
	item, events, err := s.runtime.Support.OpenWithNoteEvents(r.Context(), input.SubjectType, input.SubjectID, user.ID, organizationID, input.BreakGlass, note)
	if err != nil {
		if errors.Is(err, support.ErrInvalidInput) {
			writeProblem(w, http.StatusUnprocessableEntity, "support_case_invalid", err.Error())
		} else {
			writeProblem(w, http.StatusServiceUnavailable, "support_case_unavailable", "The support case could not be saved. Please try again.")
		}
		return
	}

	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: organizationID, Action: "support.case_opened", ResourceType: "support_case", ResourceID: item.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context())})
	writeJSON(w, http.StatusCreated, map[string]any{"case": item, "events": events})
}

func (s *Server) listSupportCases(w http.ResponseWriter, r *http.Request) {
	organizationID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	if _, _, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionReadAudit); !ok {
		return
	}
	items, err := s.runtime.Support.ReadForOrganization(r.Context(), organizationID)
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "support_cases_unavailable", "Support cases could not be loaded. Please try again.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cases": items})
}

func (s *Server) transitionSupportCase(w http.ResponseWriter, r *http.Request) {
	organizationID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	caseID, err := pathID(r, "caseID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionManageOrganization)
	if !ok || !s.requireCSRF(w, r) {
		return
	}
	item, _, readErr := s.runtime.Support.Read(r.Context(), caseID)
	if readErr != nil && !errors.Is(readErr, pgx.ErrNoRows) {
		writeProblem(w, http.StatusServiceUnavailable, "support_case_unavailable", "The support case could not be loaded. Please try again.")
		return
	}
	if errors.Is(readErr, pgx.ErrNoRows) || item.OrganizationID != organizationID {
		writeProblem(w, http.StatusNotFound, "support_case_not_found", "We could not find that support case.")
		return
	}
	var input supportTransitionRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	updated, event, err := s.runtime.Support.TransitionContext(r.Context(), caseID, user.ID, support.State(input.State), input.Note)
	if err != nil {
		switch {
		case errors.Is(err, support.ErrInvalidInput):
			writeProblem(w, http.StatusUnprocessableEntity, "support_case_invalid", err.Error())
		case errors.Is(err, support.ErrClosed):
			writeProblem(w, http.StatusConflict, "support_case_closed", "This case is closed. Open a new case if you need more help.")
		case errors.Is(err, pgx.ErrNoRows):
			writeProblem(w, http.StatusNotFound, "support_case_not_found", "We could not find that support case.")
		default:
			writeProblem(w, http.StatusServiceUnavailable, "support_case_unavailable", "The change could not be confirmed. Check the case before retrying.")
		}
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: organizationID, Action: "support.case_transitioned", ResourceType: "support_case", ResourceID: caseID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"state": input.State}})
	writeJSON(w, http.StatusOK, map[string]any{"case": updated, "event": event})
}
