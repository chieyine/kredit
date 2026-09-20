package web

import (
	"context"
	"errors"
	"kredit/internal/buyers"
	"net/http/httptest"
	"testing"
)

type scopedBusinessDirectory struct {
	buyers.Service
	profiles  []buyers.Business
	customers []buyers.Customer
	err       error
}

func (d scopedBusinessDirectory) ListBusinessProfiles(context.Context, string) ([]buyers.Business, error) {
	return d.profiles, d.err
}

func TestPurchasingScopeRequiresOwnedBusinessAndMatchingWorkspace(t *testing.T) {
	s := &Server{runtime: &Runtime{Buyers: scopedBusinessDirectory{profiles: []buyers.Business{{ID: "profile-a", WorkspaceID: "workspace-a"}, {ID: "profile-b", WorkspaceID: "workspace-b"}}}}}
	cases := []struct {
		query, id string
		status    int
	}{
		{"?business_id=profile-a", "profile-a", 200},
		{"?organization=workspace-b", "profile-b", 200},
		{"?business_id=profile-a&organization=workspace-b", "", 404},
		{"?business_id=outsider", "", 404},
		{"?organization=outsider", "", 404},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			w := httptest.NewRecorder()
			id, ok := s.buyerWorkspaceScope(w, httptest.NewRequest("GET", "/api/v1/buyer/history"+tc.query, nil), "owner")
			if id != tc.id || w.Code != tc.status || ok != (tc.status == 200) {
				t.Fatalf("scope=%q ok=%v status=%d", id, ok, w.Code)
			}
		})
	}
	s.runtime.Buyers = scopedBusinessDirectory{err: errors.New("database unavailable")}
	w := httptest.NewRecorder()
	_, ok := s.buyerWorkspaceScope(w, httptest.NewRequest("GET", "/api/v1/buyer/history?business_id=profile-a", nil), "owner")
	if ok || w.Code != 503 {
		t.Fatal("directory failure widened business scope")
	}
}

func (d scopedBusinessDirectory) ReadCustomers(string) ([]buyers.Customer, error) {
	return d.customers, d.err
}

func TestCustomerReportScopeRequiresExactBusinessAndReportsDirectoryFailure(t *testing.T) {
	s := &Server{runtime: &Runtime{Buyers: scopedBusinessDirectory{customers: []buyers.Customer{{BuyerUserID: "owner", BuyerBusinessID: "business-a"}}}}}
	for _, tc := range []struct {
		owner, business string
		status          int
	}{{"owner", "business-a", 200}, {"owner", "business-b", 404}, {"other", "business-a", 404}} {
		w := httptest.NewRecorder()
		_, ok := s.supplierCustomerBusinessScope(w, httptest.NewRequest("GET", "/history?buyer_business_id="+tc.business, nil), "supplier", tc.owner)
		if w.Code != tc.status || ok != (tc.status == 200) {
			t.Fatalf("owner=%s business=%s status=%d", tc.owner, tc.business, w.Code)
		}
	}
	s.runtime.Buyers = scopedBusinessDirectory{err: errors.New("database unavailable")}
	w := httptest.NewRecorder()
	_, ok := s.supplierCustomerBusinessScope(w, httptest.NewRequest("GET", "/history?buyer_business_id=business-a", nil), "supplier", "owner")
	if ok || w.Code != 503 {
		t.Fatal("directory outage was presented as a missing customer")
	}
}
