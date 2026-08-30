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

func TestMembershipStatusChangesProtectOwnerAndActor(t *testing.T) {
	store := NewStore()
	organization, _, err := store.Create("owner-1", CreateInput{LegalName: "First Ltd", BusinessType: "limited_company", BusinessAddress: "Lagos", Industry: "retail"})
	if err != nil {
		t.Fatal(err)
	}
	_, member, err := store.Invite("owner-1", organization.ID, "finance@example.test", "email", "finance-1", access.RoleFinance)
	if err != nil {
		t.Fatal(err)
	}
	store.ActivateInvitations(member.UserID)
	if updated, err := store.ChangeStatus(organization.ID, "owner-1", member.UserID, "suspended"); err != nil || updated.Status != "suspended" {
		t.Fatalf("suspend member: status=%q err=%v", updated.Status, err)
	}
	if updated, err := store.ChangeStatus(organization.ID, "owner-1", member.UserID, "active"); err != nil || updated.Status != "active" {
		t.Fatalf("restore member: status=%q err=%v", updated.Status, err)
	}
	if _, err := store.ChangeStatus(organization.ID, "owner-1", "owner-1", "suspended"); err == nil {
		t.Fatal("owner membership must be protected")
	}
	if _, err := store.ChangeStatus(organization.ID, member.UserID, member.UserID, "removed"); err == nil {
		t.Fatal("self-removal must be rejected")
	}
	if updated, err := store.ChangeStatus(organization.ID, "owner-1", member.UserID, "removed"); err != nil || updated.Status != "removed" {
		t.Fatalf("remove member: status=%q err=%v", updated.Status, err)
	}
	if _, ok := store.Membership(organization.ID, member.UserID); ok {
		t.Fatal("removed membership must no longer authorize access")
	}
}

func TestInvitationActivatesAfterUserAuthentication(t *testing.T) {
	store := NewStore()
	organization, _, err := store.Create("owner-1", CreateInput{LegalName: "Supplier Ltd", BusinessType: "registered_business", BusinessAddress: "Lagos", Industry: "wholesale"})
	if err != nil {
		t.Fatal(err)
	}
	invitation, membership, err := store.Invite("owner-1", organization.ID, "sales@example.test", "email", "user-2", access.RoleSales)
	if err != nil {
		t.Fatal(err)
	}
	if invitation.Status != "pending" || membership.Status != "invited" {
		t.Fatalf("unexpected invitation state: invitation=%+v membership=%+v", invitation, membership)
	}
	activated := store.ActivateInvitations("user-2")
	if len(activated) != 1 || activated[0].Status != "active" || activated[0].AcceptedAt.IsZero() {
		t.Fatalf("invitation did not activate: %+v", activated)
	}
}
