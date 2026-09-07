package access

import "testing"

func TestRolePermissions(t *testing.T) {
	if !Can(RoleOwner, PermissionManageMembers) {
		t.Fatal("owner should manage members")
	}
	if Can(RoleSales, PermissionManageMembers) {
		t.Fatal("sales must not manage members")
	}
	if !RequiresStepUp(PermissionInviteMembers) {
		t.Fatal("member invitations require step-up authentication")
	}
}

func TestPlatformOwnerPermissionsAndExclusivity(t *testing.T) {
	if !CanPlatform(PlatformOwner, PermissionPlatformOwner) {
		t.Fatal("platform owner must have PermissionPlatformOwner")
	}
	if !CanPlatform(PlatformOwner, PermissionPlatformSettings) {
		t.Fatal("platform owner must have PermissionPlatformSettings")
	}
	if !CanPlatform(PlatformOwner, PermissionAdminFinancial) {
		t.Fatal("platform owner must have PermissionAdminFinancial")
	}

	nonOwners := []PlatformRole{
		PlatformAdministrator,
		PlatformFinanceOperator,
		PlatformApprover,
		PlatformPolicyManager,
		PlatformComplianceReviewer,
		PlatformDisputeReviewer,
		PlatformSupportAgent,
		PlatformAccessAdministrator,
	}

	for _, role := range nonOwners {
		if CanPlatform(role, PermissionPlatformOwner) {
			t.Fatalf("role %s must not have PermissionPlatformOwner", role)
		}
		if CanPlatform(role, PermissionPlatformSettings) {
			t.Fatalf("role %s must not have PermissionPlatformSettings", role)
		}
	}

	if !RequiresStepUp(PermissionPlatformOwner) {
		t.Fatal("PermissionPlatformOwner must require step-up MFA")
	}
	if !RequiresStepUp(PermissionPlatformSettings) {
		t.Fatal("PermissionPlatformSettings must require step-up MFA")
	}
}
