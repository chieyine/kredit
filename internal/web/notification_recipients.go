package web

import (
	"context"
	"errors"
	"kredit/internal/access"
	"kredit/internal/notifications"
	"kredit/internal/organizations"
)

// EmitNotification resolves people and contact details before queueing a message.
// An organization UUID is never a destination or a notification recipient user.
func (r *Runtime) EmitNotification(ctx context.Context, event notifications.Event) ([]notifications.Delivery, error) {
	if event.RecipientID == "" && event.OrganizationID != "" {
		var result []notifications.Delivery
		var members []organizations.Membership
		if source, ok := r.Organizations.(interface {
			ReadMembers(string) ([]organizations.Membership, error)
		}); ok {
			var err error
			members, err = source.ReadMembers(event.OrganizationID)
			if err != nil {
				return nil, err
			}
		}
		if members == nil {
			members = r.Organizations.ListMembers(event.OrganizationID)
		}
		for _, member := range members {
			if member.Status != "active" || !access.Can(member.Role, access.PermissionReadFinancial) {
				continue
			}
			child := event
			child.ID = event.ID + ":" + member.UserID
			child.RecipientID = member.UserID
			delivered, err := r.EmitNotification(ctx, child)
			if err != nil {
				return result, err
			}
			result = append(result, delivered...)
		}
		if len(result) == 0 {
			return nil, errors.New("no eligible notification recipients")
		}
		return result, nil
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
