package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"kredit/internal/access"
	"kredit/internal/buyers"
	"kredit/internal/config"
	"kredit/internal/organizations"
)

type directoryBuyerFixture struct{ buyers.Service }

func (directoryBuyerFixture) ListCustomers(string) []buyers.Customer {
	return []buyers.Customer{{BuyerUserID: "customer-1", LegalName: "Customer supplies", Status: "active"}}
}

func TestSalesCanSelectCustomersWithoutFinancialMutationAccess(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	s := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	owner, err := s.runtime.Auth.FindOrCreateUser("directory-owner@example.test", "email")
	if err != nil {
		t.Fatal(err)
	}
	org, _, err := s.runtime.Organizations.Create(owner.ID, organizations.CreateInput{LegalName: "Directory supplies", BusinessType: "limited_company", BusinessAddress: "Lagos", Industry: "retail"})
	if err != nil {
		t.Fatal(err)
	}
	challenge, code, err := s.runtime.Auth.RequestOTP("directory-sales@example.test", "email", "login")
	if err != nil {
		t.Fatal(err)
	}
	sales, _, token, err := s.runtime.Auth.VerifyOTP(challenge.ID, code, "test")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.runtime.Organizations.Invite(owner.ID, org.ID, sales.Email, "email", sales.ID, access.RoleSales); err != nil {
		t.Fatal(err)
	}
	s.runtime.Organizations.ActivateInvitations(sales.ID)
	s.runtime.Buyers = directoryBuyerFixture{Service: s.runtime.Buyers}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/"+org.ID+"/customers", nil)
	request.AddCookie(&http.Cookie{Name: "kredit_session", Value: token})
	response := httptest.NewRecorder()
	s.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("sales cannot select a customer: %d %s", response.Code, response.Body.String())
	}
	var directory struct {
		Customers []map[string]any `json:"customers"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &directory); err != nil {
		t.Fatal(err)
	}
	if len(directory.Customers) != 1 || directory.Customers[0]["legal_name"] != "Customer supplies" {
		t.Fatalf("customer identity missing: %s", response.Body.String())
	}
	for _, field := range []string{"outstanding_kobo", "request_count"} {
		if _, exists := directory.Customers[0][field]; exists {
			t.Fatalf("sales directory exposed financial field %s", field)
		}
	}
	if access.Can(access.RoleSales, access.PermissionManageFinancial) {
		t.Fatal("customer selection must not grant financial mutations")
	}
	if _, err = s.runtime.Organizations.ChangeStatus(org.ID, owner.ID, sales.ID, "suspended"); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	s.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("suspended sales member retained directory access: %d", response.Code)
	}
}
