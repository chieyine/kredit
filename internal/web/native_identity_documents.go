package web

import (
	"bytes"
	"encoding/base64"
	"kredit/internal/audit"
	"kredit/internal/identity"
	"net/http"
	"time"
)

func (s *Server) nativeIdentityDocument(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	write := r.Method == http.MethodPost
	if write && !s.requireCSRF(w, r) {
		return
	}
	selected, err := identity.Resolve(s.runtime.Identity, r.PathValue("provider"))
	if err != nil {
		writeProblem(w, 409, "account_unavailable", err.Error())
		return
	}
	native, ok := selected.(*identity.NativeLookup)
	if !ok {
		writeProblem(w, 404, "case_not_found", "The verification could not be found.")
		return
	}
	ctx := identity.WithActor(r.Context(), user.ID)
	id := r.PathValue("caseID")
	owner, document, err := native.EvidenceTarget(ctx, id, write)
	if err != nil {
		writeProblem(w, 404, "case_not_found", "The verification could not be found.")
		return
	}
	if write {
		var in documentUploadRequest
		if err = decodeJSONLimit(w, r, &in, 3<<20); err != nil {
			writeProblem(w, 400, "invalid_document", "Choose a PDF or image up to 2 MB.")
			return
		}
		content, err := base64.StdEncoding.DecodeString(in.ContentBase64)
		if err != nil || len(content) == 0 || len(content) > 2<<20 {
			writeProblem(w, 400, "invalid_document", "Choose a PDF or image up to 2 MB.")
			return
		}
		doc, err := s.runtime.Documents.Add(ctx, "", owner, "identity_"+id, in.FileName, in.ContentType, "identity_evidence", int64(len(content)), bytes.NewReader(content))
		if err != nil {
			writeProblem(w, 422, "upload_incomplete", err.Error())
			return
		}
		if err = native.AttachEvidence(ctx, id, doc.ID); err != nil {
			writeProblem(w, 409, "attachment_incomplete", err.Error())
			return
		}
		writeJSON(w, 202, map[string]any{"document": map[string]string{"id": doc.ID, "scan_state": string(doc.ScanState)}})
		return
	}
	doc, err := s.runtime.Documents.ReadForTenant(ctx, document, owner, "")
	if err != nil || doc.Purpose != "identity_"+id || doc.UploadedBy != owner || doc.OrganizationID != "" {
		writeProblem(w, 404, "document_not_found", "No attached document was found.")
		return
	}
	if r.URL.Query().Get("download") != "true" {
		writeJSON(w, 200, map[string]any{"document": map[string]string{"id": doc.ID, "scan_state": string(doc.ScanState)}})
		return
	}
	link, err := s.runtime.Documents.SignedDownloadForTenant(ctx, document, owner, "", 5*time.Minute)
	if err != nil {
		writeProblem(w, 409, "scan_pending", "The document must pass its safety check before opening.")
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "identity.evidence_downloaded", ResourceType: "document", ResourceID: doc.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context())})
	writeJSON(w, 200, map[string]string{"url": link})
}
