package web

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"time"

	"kredit/internal/access"
	"kredit/internal/audit"
)

type documentUploadRequest struct {
	Purpose        string `json:"purpose"`
	FileName       string `json:"file_name"`
	ContentType    string `json:"content_type"`
	RetentionClass string `json:"retention_class"`
	ContentBase64  string `json:"content_base64"`
}

func (s *Server) uploadDocument(w http.ResponseWriter, r *http.Request) {
	organizationID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionCreateCredit)
	if !ok || !s.requireCSRF(w, r) {
		return
	}
	var input documentUploadRequest
	if err := decodeJSONLimit(w, r, &input, 3<<20); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	content, err := base64.StdEncoding.DecodeString(input.ContentBase64)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_document", "content_base64 must be valid base64")
		return
	}
	if len(content) > 2<<20 {
		writeProblem(w, http.StatusRequestEntityTooLarge, "document_too_large", "inline documents must be 2 MiB or smaller")
		return
	}
	doc, err := s.runtime.Documents.Add(r.Context(), organizationID, user.ID, input.Purpose, input.FileName, input.ContentType, input.RetentionClass, int64(len(content)), bytes.NewReader(content))
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "document_invalid", err.Error())
		return
	}
	if s.config.Environment == "development" && s.runtime.DocumentScanner != nil {
		if scanned, scanErr := s.runtime.Documents.Scan(r.Context(), doc.ID, s.runtime.DocumentScanner); scanErr == nil {
			doc = scanned
		}
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: organizationID, Action: "document.uploaded", ResourceType: "document", ResourceID: doc.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"purpose": doc.Purpose, "scan_state": string(doc.ScanState)}})
	writeJSON(w, http.StatusAccepted, map[string]any{"document": doc})
}

func (s *Server) documentDownload(w http.ResponseWriter, r *http.Request) {
	organizationID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionReadOrganization)
	if !ok {
		return
	}
	documentID, err := pathID(r, "documentID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	doc, exists := s.runtime.Documents.GetForTenant(r.Context(), documentID, user.ID, organizationID)
	if !exists || doc.OrganizationID != organizationID {
		writeProblem(w, http.StatusNotFound, "document_not_found", "document was not found")
		return
	}
	url, err := s.runtime.Documents.SignedDownloadForTenant(r.Context(), documentID, user.ID, organizationID, 10*time.Minute)
	if err != nil {
		writeProblem(w, http.StatusConflict, "document_unavailable", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": doc, "url": url, "expires_in_seconds": 600})
}
