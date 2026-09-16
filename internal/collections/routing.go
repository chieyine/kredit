package collections

import (
	"errors"
	"strings"
)

// The engine's view of routing. Registry holds providers for code that has no
// engine — mandate authorization, and the operations surface. The engine keeps
// its own registrations because a running engine already owns the active
// provider and the retained ones, and two copies of that would drift.
//
// Both evaluate candidates through evaluateCandidate, so a provider cannot be
// eligible in one and refused in the other.

// RegisterRetainedProvider preserves reconciliation for an existing provider
// after the deployment selects a different provider for new authorizations.
// Configure adapters at startup; no attempt is transferred between providers.
func (e *Engine) RegisterRetainedProvider(provider Provider) error {
	name, err := providerIdentity(provider)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.provider != nil && e.provider.Name() == name {
		return errors.New("active provider identity is already registered")
	}
	if e.nameTakenLocked(name) {
		return errors.New("provider identity is already registered")
	}
	if e.retainedProviders == nil {
		e.retainedProviders = map[string]Provider{}
	}
	e.retainedProviders[name] = provider
	return nil
}

// RegisterActiveProvider adds a provider that may be chosen for a new bank
// authorization, alongside the deployment's configured active provider. Rank
// orders the choice, lowest first; the configured active provider stays ahead
// of every provider registered at the same rank.
//
// This does not make an existing mandate portable. A mandate registered here
// is still debited only by the provider that created it.
func (e *Engine) RegisterActiveProvider(provider Provider, rank int) error {
	name, err := providerIdentity(provider)
	if err != nil {
		return err
	}
	if rank < 0 {
		return errors.New("provider preference rank cannot be negative")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.provider != nil && e.provider.Name() == name {
		return errors.New("active provider identity is already registered")
	}
	if e.nameTakenLocked(name) {
		return errors.New("provider identity is already registered")
	}
	if e.activeProviders == nil {
		e.activeProviders = map[string]*registration{}
	}
	e.activeSequence++
	e.activeProviders[name] = &registration{provider: provider, role: RoleActive, rank: rank, sequence: e.activeSequence}
	return nil
}

// SetRequireApproval refuses any provider that carries no written approval
// record at all, rather than only those whose record is invalid.
//
// It is deliberately off in the runtime today. Turning it on before every
// configured provider is wrapped in an ApprovedAdapter would make the
// operations surface report an approved provider as unapproved, and a status
// page that is wrong about approval is worse than one that is silent. Turn it
// on with the first provider that is selected rather than pinned.
func (e *Engine) SetRequireApproval(required bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.requireApproval = required
}

// Caller holds e.mu. Two identities that differ only by letter case are the
// same identity here: a debit routed to the wrong bank account because of
// letter case is not a class of bug worth leaving open.
func (e *Engine) nameTakenLocked(name string) bool {
	if e.provider != nil && strings.EqualFold(e.provider.Name(), name) {
		return true
	}
	for existing := range e.activeProviders {
		if strings.EqualFold(existing, name) {
			return true
		}
	}
	for existing := range e.retainedProviders {
		if strings.EqualFold(existing, name) {
			return true
		}
	}
	return false
}

// Caller holds e.mu. Always use the persisted attempt identity.
func (e *Engine) providerForLocked(name string) Provider {
	if e.provider != nil && e.provider.Name() == name {
		return e.provider
	}
	if entry := e.activeProviders[name]; entry != nil {
		return entry.provider
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

// Caller holds e.mu. The configured active provider carries sequence 0 so it
// sorts ahead of everything registered at the same rank, which keeps the
// single-provider deployment's behaviour exactly as it was.
func (e *Engine) activeRegistrationsLocked() []*registration {
	entries := make([]*registration, 0, len(e.activeProviders)+1)
	if e.provider != nil {
		entries = append(entries, &registration{provider: e.provider, role: RoleActive})
	}
	for _, entry := range e.activeProviders {
		entries = append(entries, entry)
	}
	return sortRegistrations(entries)
}

// Caller holds e.mu. Reconcile-only providers are deliberately not visible
// here: they may still be read and reconciled, never debited again.
func (e *Engine) activeRegistrationLocked(name string) *registration {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil
	}
	if e.provider != nil && e.provider.Name() == trimmed {
		return &registration{provider: e.provider, role: RoleActive}
	}
	return e.activeProviders[trimmed]
}

// activeRegistration is activeRegistrationLocked for callers that hold no lock.
func (e *Engine) activeRegistration(name string) *registration {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.activeRegistrationLocked(name)
}

func (e *Engine) retainedProviderExists(name string) bool {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, exists := e.retainedProviders[trimmed]
	return exists
}

func (e *Engine) hasActiveProvider() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.provider != nil || len(e.activeProviders) > 0
}

// debitRegistration resolves the provider for a debit. A debit always has a
// mandate and the mandate names its provider, so this is a lookup, not a
// choice: there is no rank, no health comparison and no second candidate. It
// returns nil when the named provider is missing or is reconcile-only, and the
// collection is then refused rather than sent somewhere else.
//
// A snapshot with no mandate provider is a deployment that has only ever had
// one, and falls back to the configured active provider.
func (e *Engine) debitRegistration(snapshot ObligationSnapshot) *registration {
	e.mu.Lock()
	defer e.mu.Unlock()
	if strings.TrimSpace(snapshot.MandateProvider) == "" {
		if e.provider == nil {
			return nil
		}
		return &registration{provider: e.provider, role: RoleActive}
	}
	return e.activeRegistrationLocked(snapshot.MandateProvider)
}

// reserveEveryProvider replaces every provider this engine could debit with a
// capture proxy, so a prepared submission records its request instead of
// reaching a bank. It is used on the short-lived engine that runs inside a
// database transaction; the durable engine's providers are untouched.
func (e *Engine) reserveEveryProvider() *capturedRequest {
	capture := &capturedRequest{}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.provider != nil {
		e.provider = &reserveProvider{Provider: e.provider, capture: capture}
	}
	for name, entry := range e.activeProviders {
		e.activeProviders[name] = &registration{provider: &reserveProvider{Provider: entry.provider, capture: capture}, role: entry.role, rank: entry.rank, sequence: entry.sequence}
	}
	return capture
}

// ResolveMandateProvider evaluates the provider that holds an existing
// mandate, and no other. If it is refused, the collection does not happen:
// sending a mandate's debit to a different provider is not a fallback, it is
// a debit the buyer never authorized.
func (e *Engine) ResolveMandateProvider(name string, request SelectionRequest) (Selection, error) {
	e.mu.Lock()
	entry := e.activeRegistrationLocked(name)
	requireApproval := e.requireApproval
	e.mu.Unlock()
	return pinnedSelection(entry, name, requireApproval, request)
}

// SelectProvider chooses the provider for a mandate that does not yet exist.
// It is the only method on the engine that picks between providers, and it is
// reachable only before anything has been authorized, so choosing differently
// here cannot move or duplicate an existing instruction. Nothing on the debit
// path calls it.
func (e *Engine) SelectProvider(request SelectionRequest) (Selection, error) {
	e.mu.Lock()
	entries := e.activeRegistrationsLocked()
	requireApproval := e.requireApproval
	e.mu.Unlock()
	return firstEligibleSelection(entries, requireApproval, request)
}
