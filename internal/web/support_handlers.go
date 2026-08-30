package web

import (
	"net/http"

	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/support"
)

type supportCaseRequest struct {
	SubjectType string `json:"subject_type"`
	SubjectID   string `json:"subject_id"`
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
		writeProblem(w, http.StatusForbidden, "break_glass_forbidden", "break-glass support access requires an approved platform support role")
		return
	}
	item, err := s.runtime.Support.Open(input.SubjectType, input.SubjectID, user.ID, organizationID, input.BreakGlass)
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "support_case_invalid", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: organizationID, Action: "support.case_opened", ResourceType: "support_case", ResourceID: item.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context())})
	writeJSON(w, http.StatusCreated, map[string]any{"case": item, "events": s.runtime.Support.Timeline(item.ID)})
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
	writeJSON(w, http.StatusOK, map[string]any{"cases": s.runtime.Support.ListForOrganization(organizationID)})
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
	item, exists := s.runtime.Support.Get(caseID)
	if !exists || item.OrganizationID != organizationID {
		writeProblem(w, http.StatusNotFound, "support_case_not_found", "support case was not found")
		return
	}
	var input supportTransitionRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	updated, event, err := s.runtime.Support.Transition(caseID, user.ID, support.State(input.State), input.Note)
	if err != nil {
		writeProblem(w, http.StatusConflict, "support_case_transition_failed", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: organizationID, Action: "support.case_transitioned", ResourceType: "support_case", ResourceID: caseID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"state": input.State}})
	writeJSON(w, http.StatusOK, map[string]any{"case": updated, "event": event})
}
