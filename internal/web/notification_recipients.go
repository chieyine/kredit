package web

import (
	"context"
	"errors"
	"kredit/internal/access"
	"kredit/internal/db"
	"kredit/internal/notifications"
	"kredit/internal/organizations"

	"github.com/jackc/pgx/v5"
)

// EmitNotification resolves people and contact details before queueing a message.
// An organization UUID is never a destination or a notification recipient user.
func (r *Runtime) EmitNotification(ctx context.Context, event notifications.Event) ([]notifications.Delivery, error) {
	if event.RecipientID == "" && event.OrganizationID != "" {
		owner, err := r.Organizations.ReadOwner(ctx, event.OrganizationID)
		if err != nil {
			return nil, err
		}
		event.ID += ":" + owner.UserID
		event.RecipientID = owner.UserID
		event.OrganizationAudience = true
	}

	if event.RecipientID == "" {
		return nil, errors.New("notification recipient is required")
	}
	if r.Database != nil {
		tx, err := r.Database.Raw().Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, event.RecipientID); err != nil {
			return nil, err
		}
		if err = tx.QueryRow(ctx, `SELECT COALESCE(normalized_email,''),COALESCE(normalized_phone,'') FROM app.users WHERE id=$1::uuid`, event.RecipientID).Scan(&event.Email, &event.Phone); err != nil {
			return nil, err
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, err
		}
		event.DeferDelivery = true
	} else if r.Auth != nil {
		if user, err := r.Auth.UserByID(event.RecipientID); err == nil {
			event.Email, event.Phone = user.Email, user.Phone
		}
	}
	if r.Database != nil && event.Email == "" && event.Phone == "" {
		return nil, errors.New("notification recipient has no contact destination")
	}
	return r.Notifications.Emit(ctx, event)
}

// Check the recipient's current ownership before delivering business information.
func (r *Runtime) organizationNotificationAllowed(ctx context.Context, recipient, org string) (bool, error) {
	if r.Database == nil {
		member, err := r.readMembership(ctx, org, recipient)
		if errors.Is(err, organizations.ErrMembershipNotFound) {
			return false, nil
		}
		return err == nil && member.Status == "active" && access.Can(member.Role, access.PermissionReadFinancial), err
	}
	ctx = db.WithTenantContext(ctx, recipient, org)
	scoped := &db.ScopedDatabase{Pool: r.Database.Raw()}
	var role access.Role
	err := scoped.QueryRow(ctx, `SELECT m.role FROM app.memberships m
 JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id
 WHERE m.organization_id=$1::uuid AND m.user_id=$2::uuid AND m.role='owner'
 AND m.status='active' AND u.status='active' AND o.status NOT IN ('suspended','closed')`, org, recipient).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil && access.Can(role, access.PermissionReadFinancial), err
}
