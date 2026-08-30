package web

import (
	"net/http"
	"time"

	"kredit/internal/access"
	"kredit/internal/audit"
)

type documentUploadSlotRequest struct {
	Purpose        string `json:"purpose"`
	FileName       string `json:"file_name"`
	ContentType    string `json:"content_type"`
	RetentionClass string `json:"retention_class"`
	SizeBytes      int64  `json:"size_bytes"`
}

func (s *Server) completeDocumentUpload(w http.ResponseWriter, r *http.Request) {
	organizationID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	documentID, err := pathID(r, "documentID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionCreateCredit)
	if !ok || !s.requireCSRF(w, r) {
		return
	}
	document, exists := s.runtime.Documents.Get(documentID)
	if !exists || document.OrganizationID != organizationID {
		writeProblem(w, http.StatusNotFound, "document_not_found", "document was not found")
		return
	}
	document, err = s.runtime.Documents.CompleteUpload(r.Context(), documentID)
	if err != nil {
		writeProblem(w, http.StatusConflict, "document_upload_incomplete", err.Error())
		return
	}
	if s.config.Environment == "development" && s.runtime.DocumentScanner != nil {
		if scanned, scanErr := s.runtime.Documents.Scan(r.Context(), document.ID, s.runtime.DocumentScanner); scanErr == nil {
			document = scanned
		}
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: organizationID, Action: "document.upload_completed", ResourceType: "document", ResourceID: document.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"scan_state": string(document.ScanState)}})
	writeJSON(w, http.StatusAccepted, map[string]any{"document": document, "scan_state": document.ScanState})
}

func (s *Server) createDocumentUploadSlot(w http.ResponseWriter, r *http.Request) {
	organizationID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionCreateCredit)
	if !ok || !s.requireCSRF(w, r) {
		return
	}
	var input documentUploadSlotRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	doc, url, err := s.runtime.Documents.CreateUpload(r.Context(), organizationID, user.ID, input.Purpose, input.FileName, input.ContentType, input.RetentionClass, input.SizeBytes, 10*time.Minute)
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "document_upload_invalid", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"document": doc, "upload_url": url, "expires_in_seconds": 600})
}
