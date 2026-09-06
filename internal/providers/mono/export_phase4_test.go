//go:build integration

package mono

import (
	"net/http"
	"testing"
)

// NewPhase4FixtureClient is available only to integration tests. The production
// constructor and its TLS/redirect policy are unchanged. All provider HTTP in
// these tests goes to testClient's local HTTPS server, never to Mono.
func NewPhase4FixtureClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	return testClient(t, handler)
}
