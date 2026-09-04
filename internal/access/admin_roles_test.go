package access

import "testing"

func TestAdminDutiesRemainSeparate(t *testing.T) {
	for _, test := range []struct {
		role            PlatformRole
		allowed, denied Permission
	}{
		{PlatformFinanceOperator, PermissionAdminFinancial, PermissionApproveChanges},
		{PlatformPolicyManager, PermissionManagePolicies, PermissionApproveChanges},
		{PlatformApprover, PermissionApproveChanges, PermissionManageAccess},
		{PlatformAccessAdministrator, PermissionManageAccess, PermissionAdminFinancial},
		{PlatformSupportAgent, PermissionManageCases, PermissionManagePolicies},
	} {
		if !test.role.Valid() || !CanPlatform(test.role, test.allowed) || CanPlatform(test.role, test.denied) {
			t.Fatalf("role boundary failed: %+v", test)
		}
	}
}

func TestComplianceCannotExecuteOperationalCommands(t *testing.T) {
	for _, permission := range []Permission{PermissionOperateJobs, PermissionOperateCollections, PermissionSuspendAccounts} {
		if CanPlatform(PlatformComplianceReviewer, permission) {
			t.Fatalf("compliance reviewer received %s", permission)
		}
	}
}

func TestViewerCannotMutateDisputes(t *testing.T) {
	if !Can(RoleViewer, PermissionReadFinancial) || Can(RoleViewer, PermissionManageDisputes) {
		t.Fatal("viewer financial read access leaked into dispute mutation access")
	}
}
