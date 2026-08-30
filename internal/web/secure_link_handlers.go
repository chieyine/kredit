package web

import (
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) resolveSecureLink(w http.ResponseWriter, r *http.Request) {
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
