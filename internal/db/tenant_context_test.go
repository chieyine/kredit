package db

import (
	"context"
	"testing"
)

func TestOrganizationContextRetainsAuthenticatedActor(t *testing.T) {
	ctx := WithTenantContext(context.Background(), "staff", "previous")
	id, ok := TenantFromContext(WithOrganizationContext(ctx, "selected"))
	if !ok || id.UserID != "staff" || id.OrganizationID != "selected" {
		t.Fatalf("identity was discarded: %#v", id)
	}
	background, _ := TenantFromContext(WithOrganizationContext(context.Background(), "supplier"))
	if background.UserID != "" || background.OrganizationID != "supplier" {
		t.Fatalf("background identity invented: %#v", background)
	}
}
