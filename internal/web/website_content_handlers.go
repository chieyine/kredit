package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"kredit/internal/access"
	"kredit/internal/platformsettings"

	"github.com/jackc/pgx/v5"
)

func (s *Server) websiteContent(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	page := r.PathValue("page")
	if _, ok := platformsettings.WebsitePages[page]; !ok {
		writeProblem(w, 404, "page_not_found", "This page cannot be edited here.")
		return
	}
	private := r.URL.Path != "/api/v1/website/"+page
	if private {
		session, _, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformOwner)
		if !ok || !s.requireFreshMFA(w, session) {
			return
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	if s.runtime.PlatformSettings == nil {
		if !private && r.URL.Query().Get("version") == "" {
			writeJSON(w, 200, map[string]any{"publication": nil})
			return
		}
		writeProblem(w, 503, "content_unavailable", "Website content could not be opened.")
		return
	}
	if version := r.URL.Query().Get("version"); !private && version != "" {
		number, err := strconv.Atoi(strings.TrimPrefix(version, "legal-"+page+"-v"))
		archive, available := s.runtime.PlatformSettings.(platformsettings.WebsiteArchive)
		if err != nil || !available || version != platformsettings.LegalDocumentVersion(page, number) {
			writeProblem(w, 404, "publication_not_found", "That published version was not found.")
			return
		}
		publication, err := archive.PublishedWebsiteVersion(r.Context(), page, number)
		if errors.Is(err, pgx.ErrNoRows) {
			writeProblem(w, 404, "publication_not_found", "That published version was not found.")
			return
		}
		if err != nil {
			writeProblem(w, 503, "content_unavailable", "That published version could not be opened.")
			return
		}
		writeJSON(w, 200, map[string]any{"publication": publication})
		return
	}
	setting, err := s.runtime.PlatformSettings.Get(r.Context(), platformsettings.WebsitePrefix+page, false)
	if errors.Is(err, pgx.ErrNoRows) {
		if private {
			writeJSON(w, 200, map[string]any{"version": 0, "content": nil})
		} else {
			writeJSON(w, 200, map[string]any{"publication": nil})
		}
		return
	}
	if err != nil {
		writeProblem(w, 503, "content_unavailable", "Website content could not be opened.")
		return
	}
	var content platformsettings.WebsiteContent
	if err = json.Unmarshal(setting.Value, &content); err != nil {
		writeProblem(w, 503, "content_unavailable", "Website content could not be opened.")
		return
	}
	if content.Published != nil {
		if err := platformsettings.ValidateWebsitePublication(page, *content.Published); err != nil {
			writeProblem(w, 503, "content_unavailable", "Published content could not be verified.")
			return
		}
	}
	if private {
		writeJSON(w, 200, map[string]any{"version": setting.Version, "content": content})
		return
	}
	writeJSON(w, 200, map[string]any{"publication": content.Published})
}

type websiteChangeInput struct {
	ExpectedVersion *int                         `json:"expected_version"`
	Action          string                       `json:"action"`
	Copy            platformsettings.WebsiteCopy `json:"copy"`
	Reason          string                       `json:"reason"`
}

func (s *Server) changeWebsiteContent(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformOwner)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	page := r.PathValue("page")
	if _, ok := platformsettings.WebsitePages[page]; !ok {
		writeProblem(w, 404, "page_not_found", "This page cannot be edited here.")
		return
	}
	var in websiteChangeInput
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(in.Reason)) < 4 || utf8.RuneCountInString(in.Reason) > 500 {
		writeProblem(w, 422, "invalid_reason", "Enter a reason between 4 and 500 characters.")
		return
	}
	if in.ExpectedVersion == nil || *in.ExpectedVersion < 0 || (in.Action != "save" && in.Action != "publish" && in.Action != "restore") {
		writeProblem(w, 400, "invalid_content_change", "Reload the page and choose save, publish or restore.")
		return
	}
	updater, ok := s.runtime.PlatformSettings.(platformsettings.ValidatedUpdater)
	if !ok {
		writeProblem(w, 503, "content_unavailable", "Website content could not be saved.")
		return
	}
	if err := platformsettings.ValidateWebsitePageCopy(page, in.Copy); err != nil {
		writeProblem(w, 422, "invalid_content", err.Error())
		return
	}
	key := platformsettings.WebsitePrefix + page
	initial, _ := json.Marshal(platformsettings.WebsiteContent{Draft: in.Copy})
	updated, err := updater.UpdateValidated(r.Context(), user.ID, key, initial, in.Reason, *in.ExpectedVersion, func(snapshot platformsettings.Service, _ json.RawMessage) (json.RawMessage, error) {
		current, err := snapshot.Get(r.Context(), key, false)
		var content platformsettings.WebsiteContent
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		if err == nil {
			if err = json.Unmarshal(current.Value, &content); err != nil {
				return nil, err
			}
		}
		if current.Version != *in.ExpectedVersion {
			return nil, platformsettings.ErrVersionConflict
		}
		switch in.Action {
		case "save":
			content.Draft = in.Copy
		case "restore":
			// Restore the previous publication into a new draft. Publishing remains
			// an explicit action, and the settings history keeps both old versions.
			if content.Published == nil {
				return nil, errors.New("there is no published copy to restore")
			}
			content.Draft = content.Published.Copy
		case "publish":
			if current.Version == 0 {
				return nil, errors.New("save and preview a draft before publishing")
			}
			saved, _ := json.Marshal(content.Draft)
			reviewed, _ := json.Marshal(in.Copy)
			if string(saved) != string(reviewed) {
				return nil, platformsettings.ErrVersionConflict
			}
			hash := sha256.Sum256(saved)
			content.Published = &platformsettings.WebsitePublication{Copy: content.Draft, Version: current.Version + 1, PublishedAt: time.Now().UTC().Format(time.RFC3339), ContentHash: hex.EncodeToString(hash[:])}
			if platformsettings.IsLegalPage(page) {
				content.Published.DocumentVersion = platformsettings.LegalDocumentVersion(page, current.Version+1)
				content.Published.EffectiveDate = time.Now().In(time.FixedZone("Africa/Lagos", 3600)).Format("2006-01-02")
			}
		}
		return json.Marshal(content)
	})
	if errors.Is(err, platformsettings.ErrVersionConflict) {
		writeProblem(w, 409, "content_changed", "The draft changed. Reload and review it before publishing.")
		return
	}
	if err != nil {
		writeProblem(w, 503, "content_not_saved", "The saved result could not be confirmed. Reload the editor before retrying.")
		return
	}
	var content platformsettings.WebsiteContent
	if json.Unmarshal(updated.Value, &content) != nil {
		writeProblem(w, 503, "content_unavailable", "Reload to confirm the result.")
		return
	}
	writeJSON(w, 200, map[string]any{"version": updated.Version, "content": content})
}

func (s *Server) publishedGuides(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if s.runtime.PlatformSettings == nil {
		writeJSON(w, 200, map[string]any{"publications": map[string]any{}})
		return
	}
	reader, ok := s.runtime.PlatformSettings.(interface {
		PublishedGuides(context.Context) (map[string]platformsettings.WebsitePublication, error)
	})
	if !ok {
		writeProblem(w, 503, "content_unavailable", "Guide publications could not be opened.")
		return
	}
	publications, err := reader.PublishedGuides(r.Context())
	if err != nil {
		writeProblem(w, 503, "content_unavailable", "Guide publications could not be opened.")
		return
	}
	writeJSON(w, 200, map[string]any{"publications": publications})
}
