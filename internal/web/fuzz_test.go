package web

import (
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzProblemDetailMapping covers the "problem-details mapping" target README
// section 40.3 requires. Section 14.6 forbids returning raw database or
// provider errors to a client, so the mapper must hold that line for every
// input, stay inside its length bound, and always emit valid UTF-8.
func FuzzProblemDetailMapping(f *testing.F) {
	f.Add(400, "invalid_request", "goods description is required")
	f.Add(500, "internal_error", "pq: connection refused")
	f.Add(409, "conflict", strings.Repeat("é", 400))
	f.Add(422, "credit_request_invalid", "see https://provider.example/debug")
	f.Fuzz(func(t *testing.T, status int, code, detail string) {
		result := safeProblemDetail(status, code, detail)

		if len(result) > 512 {
			t.Fatalf("problem detail exceeded its bound at %d bytes", len(result))
		}
		if !utf8.ValidString(result) {
			t.Fatalf("problem detail is not valid UTF-8: %q", result)
		}
		if status >= http.StatusInternalServerError && result != "the operation could not be completed" {
			t.Fatalf("a server error leaked a specific detail: %q", result)
		}
		lower := strings.ToLower(result)
		for _, marker := range []string{"postgres", "pgx", "sql:", "database", "connection", "secret", "password", "http://", "https://"} {
			if strings.Contains(lower, marker) {
				t.Fatalf("problem detail leaked %q: %q", marker, result)
			}
		}
	})
}
