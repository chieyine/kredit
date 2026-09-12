package collections

import "errors"

// RegisterRetainedProvider preserves reconciliation for an existing provider
// after the deployment selects a different provider for new authorizations.
// Configure adapters at startup; no attempt is transferred between providers.
func (e *Engine) RegisterRetainedProvider(provider Provider) error {
	if provider == nil || provider.Name() == "" {
		return errors.New("provider identity is required")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.provider != nil && e.provider.Name() == provider.Name() {
		return errors.New("active provider identity is already registered")
	}
	if e.retainedProviders == nil {
		e.retainedProviders = map[string]Provider{}
	}
	if _, exists := e.retainedProviders[provider.Name()]; exists {
		return errors.New("provider identity is already registered")
	}
	e.retainedProviders[provider.Name()] = provider
	return nil
}

// Caller holds e.mu. Always use the persisted attempt identity.
func (e *Engine) providerForLocked(name string) Provider {
	if e.provider != nil && e.provider.Name() == name {
		return e.provider
	}
	return e.retainedProviders[name]
}
func (e *Engine) providerFor(name string) (Provider, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	provider := e.providerForLocked(name)
	if provider == nil {
		return nil, errors.New("the original collection provider must be configured to reconcile this attempt")
	}
	return provider, nil
}
