package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kredit/internal/referrals"
)

func TestReferralLookupSeparatesMissingCodesFromOutages(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{"missing", referrals.ErrCodeUnavailable, http.StatusNotFound},
		{"wrapped missing", fmt.Errorf("lookup: %w", referrals.ErrCodeUnavailable), http.StatusNotFound},
		{"unconfigured", referrals.ErrUnavailable, http.StatusServiceUnavailable},
		{"cancelled", context.Canceled, http.StatusServiceUnavailable},
		{"timeout", context.DeadlineExceeded, http.StatusServiceUnavailable},
		{"database error", errors.New("private-connection-diagnostic"), http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			writeReferralLookupProblem(response, tc.err)
			if response.Code != tc.status {
				t.Fatalf("status = %d, want %d", response.Code, tc.status)
			}
			if strings.Contains(response.Body.String(), "private-connection-diagnostic") {
				t.Fatal("connection details leaked in a public response")
			}
			if tc.status == http.StatusServiceUnavailable && strings.Contains(response.Body.String(), "code_unavailable") {
				t.Fatal("an outage was presented as an invalid referral")
			}
		})
	}
}
