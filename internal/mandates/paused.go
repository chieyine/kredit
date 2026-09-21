package mandates

import (
	"context"
	"errors"
)

// PausedProvider preserves reads and cancellation when new permissions stop.
type PausedProvider struct{ Provider }

func NewPausedProvider(p Provider) *PausedProvider { return &PausedProvider{Provider: p} }
func (p *PausedProvider) CreateAuthorizationSession(context.Context, AuthorizationInput) (Mandate, error) {
	return Mandate{}, errors.New("new bank permissions are paused on this provider")
}
func (p *PausedProvider) RestoreAuthorization(context.Context, string) (Mandate, error) {
	return Mandate{}, errors.New("new bank permissions are paused on this provider")
}
func (p *PausedProvider) ValidateAuthorization(context.Context, AuthorizationInput) error {
	return errors.New("new bank permissions are paused on this provider")
}

func (p *PausedProvider) RecoverAuthorization(ctx context.Context, in AuthorizationInput, id string) (Mandate, error) {
	if recoverer, ok := p.Provider.(interface {
		RecoverAuthorization(context.Context, AuthorizationInput, string) (Mandate, error)
	}); ok {
		return recoverer.RecoverAuthorization(ctx, in, id)
	}
	m, e := p.GetMandate(ctx, id)
	if e != nil {
		return m, e
	}
	if m.Reference != in.Reference || m.Reference == "" || m.AmountCeiling != in.AmountCeiling || !m.Variable || m.EndsAt.Before(in.RequiredUntil) {
		return Mandate{}, errors.New("provider permission does not match the original request")
	}
	return m, nil
}
