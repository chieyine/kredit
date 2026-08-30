package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	platformlogging "kredit/internal/platform/logging"
)

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

// decodeJSONRequest keeps request-body failures consistent across handlers.
// A handler must never return an empty success response merely because its
// JSON body was malformed, too large, or contained an unsupported field.
func decodeJSONRequest(w http.ResponseWriter, r *http.Request, destination any) bool {
	if err := decodeJSON(w, r, destination); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return false
	}
	return true
}

func writeProblem(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://api.kredit.com.ng/problems/" + code,
		"title":  code,
		"status": status,
		"detail": safeProblemDetail(status, code, detail),
	})
}

func safeProblemDetail(status int, code, detail string) string {
	if status >= http.StatusInternalServerError {
		return "the operation could not be completed"
	}
	detail = platformlogging.Redact(strings.TrimSpace(detail))
	lower := strings.ToLower(detail + " " + code)
	for _, marker := range []string{"postgres", "pgx", "sql:", "database", "connection", "provider", "secret", "password", "token", "stack", "panic", "http://", "https://"} {
		if strings.Contains(lower, marker) {
			return "the operation could not be completed"
		}
	}
	if len(detail) > 512 {
		return detail[:512]
	}
	return detail
}

func pathID(r *http.Request, name string) (string, error) {
	value := r.PathValue(name)
	if value == "" {
		return "", fmt.Errorf("missing path parameter %s", name)
	}
	return value, nil
}
