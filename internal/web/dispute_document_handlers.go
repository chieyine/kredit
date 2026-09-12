package web

import (
	"bytes"
	"encoding/base64"
	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/disputes"
	"kredit/internal/documents"
	"net/http"
	"strings"
	"time"
)

func (s *Server) disputeDocumentAccess(w http.ResponseWriter, r *http.Request, write bool) (disputes.Dispute, string, bool) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return disputes.Dispute{}, "", false
	}
	id, err := pathID(r, "disputeID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", "The dispute reference is invalid.")
		return disputes.Dispute{}, "", false
	}
	item, _, _, err := s.runtime.Disputes.Get(id)
	if err != nil {
		writeProblem(w, 404, "dispute_not_found", "The dispute could not be found.")
		return item, "", false
	}
	if strings.HasPrefix(r.URL.Path, "/api/v1/ops/") {
		if write {
			writeProblem(w, 403, "upload_not_allowed", "Only the parties can submit evidence.")
			return item, "", false
		}
		if _, _, _, ok = s.requirePlatformAccess(w, r, access.PermissionReviewDisputes); !ok {
			return item, "", false
		}
	} else if strings.HasPrefix(r.URL.Path, "/api/v1/buyer/") {
		if item.BuyerUserID != user.ID {
			writeProblem(w, 404, "dispute_not_found", "The dispute could not be found.")
			return item, "", false
		}
	} else {
		org, err := pathID(r, "organizationID")
		if err != nil || org != item.SupplierOrganizationID {
			writeProblem(w, 404, "dispute_not_found", "The dispute could not be found.")
			return item, "", false
		}
		permission := access.PermissionReadFinancial
		if write {
			permission = access.PermissionManageDisputes
		}
		if _, _, _, ok = s.requireOrganizationAccess(w, r, org, permission); !ok {
			return item, "", false
		}
	}
	if write && !s.requireCSRF(w, r) {
		return item, "", false
	}
	if write && (item.State == disputes.StateResolved || item.State == disputes.StateWithdrawn) {
		writeProblem(w, 409, "dispute_closed", "This dispute is closed.")
		return item, "", false
	}
	return item, user.ID, true
}
func (s *Server) uploadDisputeDocument(w http.ResponseWriter, r *http.Request) {
	item, user, ok := s.disputeDocumentAccess(w, r, true)
	if !ok {
		return
	}
	var input documentUploadRequest
	if err := decodeJSONLimit(w, r, &input, 3<<20); err != nil {
		writeProblem(w, 400, "invalid_file", "Choose a PDF or image up to 2 MB.")
		return
	}
	content, err := base64.StdEncoding.DecodeString(input.ContentBase64)
	if err != nil || len(content) == 0 || len(content) > 2<<20 {
		writeProblem(w, 400, "invalid_file", "Choose a PDF or image up to 2 MB.")
		return
	}
	doc, err := s.runtime.Documents.Add(r.Context(), item.SupplierOrganizationID, user, "dispute_"+item.ID, input.FileName, input.ContentType, "dispute_evidence", int64(len(content)), bytes.NewReader(content))
	if err != nil {
		writeProblem(w, 422, "document_invalid", err.Error())
		return
	}
	if s.config.Environment == "development" && s.runtime.DocumentScanner != nil {
		if scanned, scanErr := s.runtime.Documents.Scan(r.Context(), doc.ID, s.runtime.DocumentScanner); scanErr == nil {
			doc = scanned
		}
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user, OrganizationID: item.SupplierOrganizationID, Action: "dispute.document_uploaded", ResourceType: "document", ResourceID: doc.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context())})
	writeJSON(w, 202, map[string]any{"document": doc})
}
func (s *Server) disputeDocument(w http.ResponseWriter, r *http.Request) {
	item, user, ok := s.disputeDocumentAccess(w, r, false)
	if !ok {
		return
	}
	documentID, err := pathID(r, "documentID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", "The document reference is invalid.")
		return
	}
	_, evidence, _, err := s.runtime.Disputes.Get(item.ID)
	if err != nil {
		writeProblem(w, 503, "document_unavailable", "The document could not be opened.")
		return
	}
	attached := false
	submitter := ""
	for _, entry := range evidence {
		if entry.DocumentID == documentID {
			attached = true
			submitter = entry.SubmittedBy
			break
		}
	}
	doc, err := s.runtime.Documents.ReadForTenant(r.Context(), documentID, user, item.SupplierOrganizationID)
	// Before submission, only its uploader can inspect scan progress.
	if err != nil || doc.Purpose != "dispute_"+item.ID || (!attached && doc.UploadedBy != user) || (attached && doc.UploadedBy != submitter) {
		writeProblem(w, 404, "document_not_found", "This document is not part of the dispute.")
		return
	}
	if !strings.HasSuffix(r.URL.Path, "/download") {
		writeJSON(w, 200, map[string]any{"document": doc})
		return
	}
	if !attached {
		writeProblem(w, 409, "evidence_required", "Attach the document to the dispute before sharing it.")
		return
	}
	url, err := s.runtime.Documents.SignedDownloadForTenant(r.Context(), documentID, user, item.SupplierOrganizationID, 5*time.Minute)
	if err != nil {
		writeProblem(w, 409, "document_unavailable", "The file must pass its safety check before it can be opened.")
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user, OrganizationID: item.SupplierOrganizationID, Action: "dispute.document_downloaded", ResourceType: "document", ResourceID: doc.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context())})
	writeJSON(w, 200, map[string]any{"url": url, "expires_in_seconds": 300})
}
func (s *Server) validateDisputeDocument(r *http.Request, item disputes.Dispute, user, id string) error {
	if strings.TrimSpace(id) == "" {
		return nil
	}
	doc, err := s.runtime.Documents.ReadForTenant(r.Context(), id, user, item.SupplierOrganizationID)
	if err != nil {
		return err
	}
	if doc.UploadedBy != user || doc.Purpose != "dispute_"+item.ID || doc.UploadCompletedAt.IsZero() || doc.ScanState != documents.ScanClean {
		return documents.ErrScanNotClean
	}
	return nil
}
