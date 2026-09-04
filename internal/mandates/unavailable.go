package mandates

import (
	"context"
	"errors"
)

type UnavailableProvider struct{ name string }

func NewUnavailableProvider(name string) *UnavailableProvider {
	return &UnavailableProvider{name: name}
}
func (p *UnavailableProvider) Name() string { return p.name }
func (p *UnavailableProvider) CreateAuthorizationSession(context.Context, AuthorizationInput) (Mandate, error) {
	return Mandate{}, errors.New("mandate provider is disabled or unconfigured")
}
func (p *UnavailableProvider) GetMandate(context.Context, string) (Mandate, error) {
	return Mandate{}, errors.New("mandate provider is disabled or unconfigured")
}
func (p *UnavailableProvider) CancelMandate(context.Context, string, string) (Mandate, error) {
	return Mandate{}, errors.New("mandate provider is disabled or unconfigured")
}
func (p *UnavailableProvider) RestoreAuthorization(context.Context, string) (Mandate, error) {
	return Mandate{}, errors.New("mandate provider is disabled or unconfigured")
}
