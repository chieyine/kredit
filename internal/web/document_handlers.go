package web

import (
	"bytes"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/documents"
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
		writeProblem(w, http.StatusBadRequest, "invalid_document", "That file could not be read. Please try uploading it again.")
		return
	}
	if len(content) > 2<<20 {
		writeProblem(w, http.StatusRequestEntityTooLarge, "document_too_large", "That file is too big. It must be 2 MB or smaller.")
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
	doc, readErr := s.runtime.Documents.ReadForTenant(r.Context(), documentID, user.ID, organizationID)
	if errors.Is(readErr, documents.ErrNotFound) {
		writeProblem(w, http.StatusNotFound, "document_not_found", "We could not find that document.")
		return
	}
	if readErr != nil {
		writeProblem(w, http.StatusServiceUnavailable, "document_unavailable", "The document could not be opened. Try again.")
		return
	}
	if strings.HasPrefix(doc.Purpose, "dispute_") {
		writeProblem(w, 403, "dispute_access_required", "Open this document from its dispute so the evidence access checks can run.")
		return
	}
	url, err := s.runtime.Documents.SignedDownloadForTenant(r.Context(), documentID, user.ID, organizationID, 10*time.Minute)
	if err != nil {
		status := http.StatusServiceUnavailable
		if errors.Is(err, documents.ErrScanNotClean) {
			status = http.StatusConflict
		}
		writeProblem(w, status, "document_unavailable", "The document is unavailable. It must finish scanning before it can be downloaded.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": doc, "url": url, "expires_in_seconds": 600})
}

func (s *Server) buyerInvoice(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	id, err := pathID(r, "requestID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", "The sale reference is invalid.")
		return
	}
	view, err := s.runtime.Credit.GetForBuyer(id, user.ID)
	if err != nil || view.Request.State == "DRAFT" || view.Request.InvoiceDocumentID == "" {
		writeProblem(w, 404, "invoice_not_found", "No invoice is available for this sale.")
		return
	}
	doc, err := s.runtime.Documents.ReadForTenant(r.Context(), view.Request.InvoiceDocumentID, user.ID, view.Request.SupplierOrganizationID)
	if err != nil || doc.Purpose != "credit_invoice" || doc.SHA256 != view.Request.InvoiceDocumentHash {
		writeProblem(w, 404, "invoice_not_found", "The invoice could not be confirmed.")
		return
	}
	url, err := s.runtime.Documents.SignedDownloadForTenant(r.Context(), doc.ID, user.ID, view.Request.SupplierOrganizationID, 5*time.Minute)
	if err != nil {
		writeProblem(w, 409, "invoice_unavailable", "The invoice must pass its safety check before it can be opened.")
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: view.Request.SupplierOrganizationID, Action: "sale.invoice_downloaded", ResourceType: "document", ResourceID: doc.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context())})
	writeJSON(w, 200, map[string]any{"url": url, "expires_in_seconds": 300})
}
