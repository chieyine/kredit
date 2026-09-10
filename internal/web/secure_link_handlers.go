package web

import (
	"encoding/hex"
	"net/http"
	"net/url"
	pathpkg "path"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func (s *Server) resolveSecureLink(w http.ResponseWriter, r *http.Request) {
	if s.runtime.Notifications == nil {
		writeProblem(w, http.StatusServiceUnavailable, "secure_link_unavailable", "This link could not be checked. Please try again.")
		return
	}
	encodedPath := strings.TrimSpace(r.URL.Query().Get("path"))
	signature := strings.TrimSpace(r.URL.Query().Get("sig"))
	expiresUnix, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("exp")), 10, 64)
	if err != nil || encodedPath == "" || signature == "" {
		writeProblem(w, http.StatusBadRequest, "secure_link_invalid", "This secure link is invalid.")
		return
	}
	decoded, err := hex.DecodeString(encodedPath)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "secure_link_invalid", "This secure link is invalid.")
		return
	}
	path := string(decoded)
	if !safeSecureRedirect(path) || !s.runtime.Notifications.VerifySecureLink(path, time.Unix(expiresUnix, 0), signature) {
		writeProblem(w, http.StatusGone, "secure_link_expired", "This secure link is invalid or has expired.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]string{"redirect_to": path})
}

func safeSecureRedirect(path string) bool {
	if !utf8.ValidString(path) || strings.IndexFunc(path, unicode.IsControl) >= 0 {
		return false
	}
	parsed, err := url.ParseRequestURI(path)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return false
	}
	decoded := parsed.Path
	if !utf8.ValidString(decoded) || strings.IndexFunc(decoded, unicode.IsControl) >= 0 || strings.Contains(decoded, "\\") {
		return false
	}
	if pathpkg.Clean(decoded) != strings.TrimSuffix(decoded, "/") {
		return false
	}
	path = decoded
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.ContainsAny(path, "\r\n\\") {
		return false
	}
	for _, prefix := range []string{"/app/", "/buyer/", "/buyer-invitations/", "/pay/", "/c/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
