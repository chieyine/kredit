package access

import "errors"

type Role string

const (
	RoleOwner         Role = "owner"
	RoleAdministrator Role = "administrator"
	RoleFinance       Role = "finance"
	RoleSales         Role = "sales"
	RoleCollections   Role = "collections"
	RoleViewer        Role = "viewer"
)

// PlatformRole is intentionally separate from supplier organisation
// membership. Platform operators never inherit supplier permissions by
// accident and must use an explicitly audited break-glass session.
type PlatformRole string

const (
	PlatformSupportAgent       PlatformRole = "support_agent"
	PlatformComplianceReviewer PlatformRole = "compliance_reviewer"
	PlatformDisputeReviewer    PlatformRole = "dispute_reviewer"
	PlatformAdministrator      PlatformRole = "platform_admin"
)

func (r PlatformRole) Valid() bool {
	switch r {
	case PlatformSupportAgent, PlatformComplianceReviewer, PlatformDisputeReviewer, PlatformAdministrator:
		return true
	default:
		return false
	}
}

type Permission string

const (
	PermissionReadOrganization   Permission = "organization:read"
	PermissionManageOrganization Permission = "organization:manage"
	PermissionInviteMembers      Permission = "members:invite"
	PermissionManageMembers      Permission = "members:manage"
	PermissionInviteBuyers       Permission = "buyers:invite"
	PermissionCreateCredit       Permission = "credit:create"
	PermissionReleaseGoods       Permission = "goods:release"
	PermissionReadAudit          Permission = "audit:read"
	PermissionReadFinancial      Permission = "financial:read"
	PermissionManageFinancial    Permission = "financial:manage"
	PermissionSupportSearch      Permission = "support:search"
	PermissionManageCases        Permission = "support:cases:manage"
	PermissionReviewCompliance   Permission = "compliance:review"
	PermissionReviewDisputes     Permission = "disputes:review"
	PermissionProviderOperations Permission = "providers:operate"
	PermissionBreakGlass         Permission = "platform:break_glass"
	PermissionRecoverAccounts    Permission = "accounts:recover"
	PermissionReviewPrivacy      Permission = "privacy:review"
	PermissionManageRiskHold     Permission = "risk_hold:manage"
)

func ParseRole(value string) (Role, error) {
	role := Role(value)
	if !role.Valid() {
		return "", errors.New("invalid organisation role")
	}
	return role, nil
}

func CanPlatform(role PlatformRole, permission Permission) bool {
	if !role.Valid() {
		return false
	}
	switch permission {
	case PermissionSupportSearch, PermissionManageCases:
		return role == PlatformSupportAgent || role == PlatformAdministrator
	case PermissionReviewCompliance, PermissionRecoverAccounts, PermissionReviewPrivacy, PermissionManageRiskHold:
		return role == PlatformComplianceReviewer || role == PlatformAdministrator
	case PermissionReviewDisputes:
		return role == PlatformDisputeReviewer || role == PlatformAdministrator
	case PermissionProviderOperations:
		return role == PlatformAdministrator || role == PlatformComplianceReviewer
	case PermissionBreakGlass:
		return role == PlatformAdministrator
	default:
		return false
	}
}

func (r Role) Valid() bool {
	switch r {
	case RoleOwner, RoleAdministrator, RoleFinance, RoleSales, RoleCollections, RoleViewer:
		return true
	default:
		return false
	}
}

func Can(role Role, permission Permission) bool {
	if role == RoleOwner {
		return true
	}
	switch permission {
	case PermissionReadOrganization:
		return role.Valid()
	case PermissionManageOrganization:
		return role == RoleAdministrator
	case PermissionInviteMembers, PermissionManageMembers:
		return role == RoleAdministrator
	case PermissionInviteBuyers:
		return role == RoleAdministrator || role == RoleSales
	case PermissionCreateCredit, PermissionReleaseGoods:
		return role == RoleAdministrator || role == RoleSales
	case PermissionReadAudit:
		return role == RoleAdministrator || role == RoleFinance || role == RoleViewer
	case PermissionReadFinancial:
		return role == RoleFinance || role == RoleCollections || role == RoleViewer
	case PermissionManageFinancial:
		return role == RoleFinance || role == RoleCollections
	default:
		return false
	}
}

func RequiresStepUp(permission Permission) bool {
	switch permission {
	case PermissionManageOrganization, PermissionInviteMembers, PermissionManageMembers, PermissionInviteBuyers, PermissionCreateCredit, PermissionReleaseGoods, PermissionManageFinancial:
		return true
	default:
		return false
	}
}
