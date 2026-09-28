package organizations

import (
	"testing"

	"kredit/internal/access"
)

func TestOrganizationMembershipsAreTenantScoped(t *testing.T) {
	store := NewStore()
	first, ownerMembership, err := store.Create("owner-1", CreateInput{LegalName: "First Ltd", BusinessType: "limited_company", BusinessAddress: "Lagos", Industry: "pharmaceuticals"})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := store.Create("owner-2", CreateInput{LegalName: "Second Ltd", BusinessType: "limited_company", BusinessAddress: "Abuja", Industry: "wholesale"})
	if err != nil {
		t.Fatal(err)
	}
	if ownerMembership.OrganizationID != first.ID {
		t.Fatal("owner membership has wrong organisation")
	}
	if _, ok := store.Membership(second.ID, "owner-1"); ok {
		t.Fatal("user from first organisation must not access second organisation")
	}
	if ownerMembership.Role != access.RoleOwner {
		t.Fatal("creator must be owner")
	}
}

func TestUnregisteredBusinessCanCreateOrganization(t *testing.T) {
	store := NewStore()
	organization, membership, err := store.Create("owner-informal", CreateInput{
		LegalName:       "Amina Yusuf",
		TradingName:     "Amina Food Store",
		BusinessType:    "unregistered_business",
		BusinessAddress: "Mushin, Lagos",
		Industry:        "food",
	})
	if err != nil {
		t.Fatal(err)
	}
	if organization.BusinessType != "unregistered_business" {
		t.Fatalf("unexpected business type: %q", organization.BusinessType)
	}
	if membership.Role != access.RoleOwner {
		t.Fatal("unregistered business creator must be its owner")
	}
}
