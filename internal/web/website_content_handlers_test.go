package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"kredit/internal/platformsettings"
)

func verifyWebsitePublication(t *testing.T, server *Server, ownerToken, aal1Token string) {
	t.Helper()
	request := func(method, path, token string, payload any) *httptest.ResponseRecorder {
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		if token != "" {
			req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
		}
		if method != "GET" {
			req.Header.Set("Origin", "http://localhost")
			req.Header.Set("Sec-Fetch-Site", "same-origin")
			req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "website-csrf-fixture"})
			req.Header.Set("X-CSRF-Token", "website-csrf-fixture")
		}
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		return rec
	}
	t.Run("guide and contact publication", func(t *testing.T) {
		for _, page := range []string{"guide-how-to-sell-goods-on-credit-in-nigeria", "contact"} {
			copy := platformsettings.WebsiteCopy{Title: "Reviewed editorial fixture", Introduction: "Published from the owner editor", Sections: []platformsettings.WebsiteSection{}}
			if page == "contact" {
				copy.Contact = &platformsettings.WebsiteContact{SupportEmail: "help@example.test", PrivacyEmail: "privacy@example.test", Address: "Test office"}
			} else {
				copy.Sections = []platformsettings.WebsiteSection{{Heading: "Section", Body: "Reviewed paragraph", Points: []string{"A reviewed point"}}}
				copy.Guide = &platformsettings.WebsiteGuide{Description: "Reviewed description", Category: "Credit sales", Keyphrase: "Guide", FAQ: []platformsettings.WebsiteSection{}, Sources: []platformsettings.WebsiteSource{}, Related: []string{}}
			}
			path := "/api/v1/ops/website/" + page
			before := request("GET", path, ownerToken, nil)
			var version struct {
				Version int `json:"version"`
			}
			if before.Code != 200 || json.Unmarshal(before.Body.Bytes(), &version) != nil {
				t.Fatalf("open editor: %d", before.Code)
			}
			old := request("GET", "/api/v1/website/"+page, "", nil).Body.String()
			for i, action := range []string{"save", "publish"} {
				rec := request("POST", path, ownerToken, map[string]any{"action": action, "expected_version": version.Version + i, "copy": copy, "reason": "Verify owner editorial publication"})
				if rec.Code != 200 {
					t.Fatalf("%s %s: %d %s", page, action, rec.Code, rec.Body.String())
				}
				if action == "save" && request("GET", "/api/v1/website/"+page, "", nil).Body.String() != old {
					t.Fatal("draft leaked publicly")
				}
			}
			public := request("GET", "/api/v1/website/"+page, "", nil)
			if public.Code != 200 || !bytes.Contains(public.Body.Bytes(), []byte("Reviewed editorial fixture")) {
				t.Fatal("publication missing")
			}
		}
		catalog := request("GET", "/api/v1/website-guides", "", nil)
		if catalog.Code != 200 || !bytes.Contains(catalog.Body.Bytes(), []byte("Reviewed editorial fixture")) || bytes.Contains(catalog.Body.Bytes(), []byte(`"draft"`)) || bytes.Contains(catalog.Body.Bytes(), []byte("integrations.")) {
			t.Fatalf("unsafe or missing guide catalog: %d", catalog.Code)
		}
	})

	private := "/api/v1/ops/website/home"
	if rec := request("GET", private, aal1Token, nil); rec.Code == 200 {
		t.Fatal("draft accessible without fresh owner MFA")
	}
	rec := request("GET", private, ownerToken, nil)
	if rec.Code != 200 {
		t.Fatalf("read draft: %d %s", rec.Code, rec.Body.String())
	}
	var before struct {
		Version int                              `json:"version"`
		Content *platformsettings.WebsiteContent `json:"content"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &before); err != nil {
		t.Fatal(err)
	}
	oldPublic := request("GET", "/api/v1/website/home", "", nil).Body.String()
	copy := platformsettings.WebsiteCopy{Title: "Synthetic owner draft", Accent: "Preview first.", Introduction: "This is an isolated publication fixture.", Sections: []platformsettings.WebsiteSection{}}
	mutate := func(action string, version int, value platformsettings.WebsiteCopy) *httptest.ResponseRecorder {
		return request("POST", private, ownerToken, map[string]any{"action": action, "expected_version": version, "copy": value, "reason": "Synthetic content publication verification"})
	}
	rec = mutate("save", before.Version, copy)
	if rec.Code != 200 {
		t.Fatalf("save draft: %d %s", rec.Code, rec.Body.String())
	}
	if public := request("GET", "/api/v1/website/home", "", nil).Body.String(); public != oldPublic {
		t.Fatal("saving draft changed the public page")
	}
	changed := copy
	changed.Title = "Unreviewed edit"
	if rec = mutate("publish", before.Version+1, changed); rec.Code != 409 {
		t.Fatalf("unreviewed content published: %d", rec.Code)
	}
	if rec = mutate("publish", before.Version, copy); rec.Code != 409 {
		t.Fatalf("stale version published: %d", rec.Code)
	}
	rec = mutate("publish", before.Version+1, copy)
	if rec.Code != 200 {
		t.Fatalf("publish saved draft: %d %s", rec.Code, rec.Body.String())
	}
	var public struct {
		Publication platformsettings.WebsitePublication `json:"publication"`
	}
	rec = request("GET", "/api/v1/website/home", "", nil)
	if err := json.Unmarshal(rec.Body.Bytes(), &public); err != nil {
		t.Fatal(err)
	}
	if public.Publication.Copy.Title != copy.Title || len(public.Publication.ContentHash) != 64 || public.Publication.Version != before.Version+2 {
		t.Fatalf("published record missing evidence: %+v", public.Publication)
	}
	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if _, leaked := raw["content"]; leaked {
		t.Fatal("public response contains private editor state")
	}
	// The generic settings endpoint cannot bypass the publication workflow.
	rec = request("POST", "/api/v1/ops/platform-settings", ownerToken, map[string]any{"key": "website.home", "value": map[string]any{"draft": copy}, "reason": "Attempt direct content replacement", "expected_version": before.Version + 2})
	if rec.Code != 400 {
		t.Fatalf("generic settings bypass: %d", rec.Code)
	}
	// Legal publications remain retrievable after subsequent drafts/publications.
	private = "/api/v1/ops/website/complaints"
	rec = request("GET", private, ownerToken, nil)
	if rec.Code != 200 {
		t.Fatalf("legal editor: %d %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &before); err != nil {
		t.Fatal(err)
	}
	copy = platformsettings.WebsiteCopy{Title: "Complaint procedure", Introduction: "Synthetic local publication fixture.", Sections: []platformsettings.WebsiteSection{{Heading: "Contact", Body: "Keep your sale reference."}}}
	firstVersion := before.Version + 2
	for i := 0; i < 2; i++ {
		if i == 1 {
			copy.Sections[0].Body = "A later published procedure."
		}
		if rec = mutate("save", before.Version+i*2, copy); rec.Code != 200 {
			t.Fatalf("legal save: %d %s", rec.Code, rec.Body.String())
		}
		if rec = mutate("publish", before.Version+i*2+1, copy); rec.Code != 200 {
			t.Fatalf("legal publish: %d %s", rec.Code, rec.Body.String())
		}
	}
	rec = request("GET", "/api/v1/website/complaints?version="+platformsettings.LegalDocumentVersion("complaints", firstVersion), "", nil)
	if rec.Code != 200 {
		t.Fatalf("legal archive: %d %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &public); err != nil {
		t.Fatal(err)
	}
	if public.Publication.Copy.Sections[0].Body != "Keep your sale reference." || public.Publication.DocumentVersion != platformsettings.LegalDocumentVersion("complaints", firstVersion) {
		t.Fatal("historical publication was overwritten")
	}
	rec = request("GET", "/api/v1/website/complaints?version="+platformsettings.LegalDocumentVersion("complaints", firstVersion-1), "", nil)
	if rec.Code != 404 {
		t.Fatalf("private draft exposed as publication: %d", rec.Code)
	}

}
