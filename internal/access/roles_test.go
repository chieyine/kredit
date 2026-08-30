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
