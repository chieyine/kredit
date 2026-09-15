package web

import (
	"testing"
	"time"
)

func TestTraderIdentityDecisionRequiresCurrentMatchingEvidence(t *testing.T) {
	future := time.Now().Add(time.Hour)
	for _, tc := range []struct {
		name   string
		level  int
		safe   map[string]string
		expiry time.Time
		want   string
	}{
		{"verified NIN", 2, map[string]string{"nin_status": "verified", "verified_name": "Ada Okafor"}, future, "approved"},
		{"verified BVN reordered", 2, map[string]string{"bvn_status": "verified", "verified_name": "Okafor Ada"}, future, "approved"},
		{"wrong name", 2, map[string]string{"nin_status": "verified", "verified_name": "Someone Else"}, future, "provider_review"},
		{"partial name", 2, map[string]string{"nin_status": "verified", "verified_name": "Ada"}, future, "provider_review"},
		{"CAC only", 2, map[string]string{"cac_status": "verified", "verified_name": "Ada Okafor"}, future, "provider_review"},
		{"low confidence", 1, map[string]string{"nin_status": "verified", "verified_name": "Ada Okafor"}, future, "provider_review"},
		{"expired", 2, map[string]string{"nin_status": "verified", "verified_name": "Ada Okafor"}, time.Now().Add(-time.Hour), "provider_review"},
		{"no expiry", 2, map[string]string{"nin_status": "verified", "verified_name": "Ada Okafor"}, time.Time{}, "provider_review"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, reason := supplierIdentityDecision(true, "Ada Okafor", "verified", tc.level, tc.safe, tc.expiry)
			if state != tc.want {
				t.Fatalf("got %s", state)
			}
			if state == "approved" && reason != "owner_identity_approved" {
				t.Fatal("personal approval must remain distinct from CAC")
			}
		})
	}
}
