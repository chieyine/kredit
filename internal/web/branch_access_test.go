package web

import "testing"

func TestBranchRoutesDoNotInheritCompanyWideAccess(t *testing.T) {
	for _, p := range []string{"GET /api/v1/organizations/{organizationID}/members", "GET /api/v1/organizations/{organizationID}/audit", "GET /api/v1/organizations/{organizationID}/consumer-sales", "POST /api/v1/organizations/{organizationID}/documents", "GET /api/v1/organizations/{organizationID}/reports/unknown", ""} {
		if branchScopedRoute(p) {
			t.Fatalf("unallocated company-wide route allowed: %s", p)
		}
	}
	for _, p := range []string{
		"GET /api/v1/organizations/{organizationID}/credit-requests",
		"GET /api/v1/organizations/{organizationID}/credit-requests/{requestID}",
		"POST /api/v1/organizations/{organizationID}/credit-requests/{requestID}/release",
		"GET /api/v1/organizations/{organizationID}/reports/ageing",
		"POST /api/v1/organizations/{organizationID}/trade-lines/{lineID}/drawdowns/{drawdownID}/release",
		"POST /api/v1/organizations/{organizationID}/trade-lines/{lineID}/drawdowns/{drawdownID}/cancel",
		"POST /api/v1/organizations/{organizationID}/drawdowns/{drawdownID}/approval",
	} {
		if !branchScopedRoute(p) {
			t.Fatalf("customer-linked route unavailable: %s", p)
		}
	}
}
