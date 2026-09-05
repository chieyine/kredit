package db

import (
	"context"
	"strings"
)

type tenantContextKey struct{}

type TenantContext struct {
	UserID         string
	OrganizationID string
}

// WithTenantContext carries identity that has already been authenticated and
// authorized by the HTTP or worker boundary. Repository code may use this
// identity to establish transaction-local PostgreSQL RLS settings; it must not
// derive tenant identity from the row the caller is trying to access.
func WithTenantContext(ctx context.Context, userID, organizationID string) context.Context {
	return context.WithValue(ctx, tenantContextKey{}, TenantContext{
		UserID:         strings.TrimSpace(userID),
		OrganizationID: strings.TrimSpace(organizationID),
	})
}

func TenantFromContext(ctx context.Context) (TenantContext, bool) {
	if ctx == nil {
		return TenantContext{}, false
	}
	identity, ok := ctx.Value(tenantContextKey{}).(TenantContext)
	if !ok {
		return TenantContext{}, false
	}
	identity.UserID = strings.TrimSpace(identity.UserID)
	identity.OrganizationID = strings.TrimSpace(identity.OrganizationID)
	if identity.UserID == "" && identity.OrganizationID == "" {
		return TenantContext{}, false
	}
	return identity, true
}
