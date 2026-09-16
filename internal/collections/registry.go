package collections

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"kredit/internal/ledger"
)

// A Nigerian direct debit mandate is an instruction held by one provider
// against one bank account. No other provider can present it, and no provider
// can adopt one created elsewhere. Routing therefore has exactly two
// operations:
//
//   - Selection, which happens once, before a mandate exists, and decides
//     which provider will be asked to create it.
//   - Resolution, which is a lookup by the provider name already recorded
//     against a mandate or a submitted attempt.
//
// There is deliberately no third operation. Nothing here moves an existing
// mandate to another provider, and nothing here retries a submitted debit
// somewhere else: a debit whose outcome is unknown may already have moved the
// buyer's money, so sending it to a second provider is how a platform debits a
// customer twice. Failover belongs at selection time or not at all.

// ProviderRole records what a registered collection provider may be used for.
type ProviderRole string

const (
	// RoleActive may be selected for a new bank authorization, and may debit
	// the mandates it holds.
	RoleActive ProviderRole = "active"
	// RoleReconcileOnly may never be selected again. It stays registered so
	// the mandates and attempts it already holds can still be read, reconciled
	// and cancelled.
	RoleReconcileOnly ProviderRole = "reconcile_only"
)

// Refusal reasons. These are stable identifiers, not sentences: they are
// recorded against an attempt and shown on the operations surface, so an
// operator can tell why a provider was passed over without reading logs.
const (
	ReasonProviderNotRegistered = "provider_not_registered"
	ReasonProviderReconcileOnly = "provider_reconcile_only"
	ReasonProviderNotApproved   = "provider_not_approved"
	ReasonProviderCircuitOpen   = "provider_circuit_open"
	ReasonAmountBelowMinimum    = "amount_below_provider_minimum"
	ReasonAmountAboveMaximum    = "amount_above_provider_maximum"
	ReasonPolicyNotSupported    = "collection_policy_not_supported"
	ReasonCurrencyNotSupported  = "currency_not_supported"
	ReasonNoEligibleProvider    = "no_eligible_collection_provider"
)

// CapabilityRefusal names the one capability a provider is missing, so a
// refusal says "capability_not_supported:variable_amount_collection" rather
// than a single opaque code covering every capability at once.
func CapabilityRefusal(capability Capability) string {
	return "capability_not_supported:" + string(capability)
}

// EnabledProvider reports whether a provider's written approval and feature
// flag currently permit it to be used. ApprovedAdapter implements it. A
// provider that does not implement it carries no approval record at all;
// see Registry.RequireApproval.
type EnabledProvider interface{ Enabled() bool }

// SelectionRequest is what a provider has to be able to do. A zero value asks
// only that the provider be registered, active and approved.
type SelectionRequest struct {
	// Required capabilities. Every one must be supported.
	Required []Capability
	// Policy is the obligation's collection policy, validated against the
	// provider's capabilities by ValidatePolicy.
	Policy string
	// Currency is the obligation currency. It is checked only against
	// providers that declare SupportedCurrencies.
	Currency string
	// AmountKobo is the amount to be collected, checked against the
	// provider's declared minimum and maximum. Zero skips the bounds check,
	// which is what mandate authorization wants: the ceiling is known, the
	// individual debit amounts are not.
	AmountKobo ledger.Money
}

// Candidate is one provider's evaluation against a SelectionRequest. It is
// returned for every candidate, eligible or not, because the reason a
// provider was passed over is the part an operator needs.
type Candidate struct {
	Name         string       `json:"name"`
	Role         ProviderRole `json:"role"`
	Rank         int          `json:"rank"`
	Eligible     bool         `json:"eligible"`
	Reasons      []string     `json:"reasons,omitempty"`
	Capabilities Capabilities `json:"capabilities"`
	Health       HealthStatus `json:"health"`
}

// Selection is the outcome of routing. Provider is the one provider that will
// be asked; Candidates records everything that was considered.
type Selection struct {
	Provider   Provider    `json:"-"`
	Name       string      `json:"name"`
	Pinned     bool        `json:"pinned"`
	Candidates []Candidate `json:"candidates"`
}

// ProviderRefusedError carries the refusal reasons programmatically so a
// caller can record them without parsing the message.
type ProviderRefusedError struct {
	ProviderName string
	Reasons      []string
}

func (e *ProviderRefusedError) Error() string {
	name := strings.TrimSpace(e.ProviderName)
	if name == "" {
		return "no approved collection provider can take this collection"
	}
	if len(e.Reasons) == 0 {
		return "collection provider " + name + " is unavailable"
	}
	return fmt.Sprintf("collection provider %s cannot take this collection (%s)", name, strings.Join(e.Reasons, ", "))
}

// evaluateCandidate is the single place routing policy lives. Both the
// standalone Registry and the collection Engine use it, so a provider cannot
// be eligible in one and refused in the other.
func evaluateCandidate(provider Provider, role ProviderRole, rank int, requireApproval bool, request SelectionRequest) Candidate {
	candidate := Candidate{Role: role, Rank: rank, Health: HealthStatus{State: CircuitClosed, Healthy: true}}
	if provider == nil {
		candidate.Reasons = []string{ReasonProviderNotRegistered}
		return candidate
	}
	candidate.Name = provider.Name()
	if capabilities, ok := provider.(CapabilityProvider); ok {
		candidate.Capabilities = capabilities.Capabilities()
	}
	if health, ok := provider.(HealthProvider); ok {
		candidate.Health = health.Health()
	}
	reasons := []string{}
	if role != RoleActive {
		reasons = append(reasons, ReasonProviderReconcileOnly)
	}
	// An unapproved provider is refused here as well as inside
	// ApprovedAdapter.Submit. The adapter is the authoritative gate; this is
	// so the refusal is visible before an attempt is recorded, rather than as
	// a failed debit after one.
	if approval, recorded := approvalOf(provider); recorded {
		if !approval.Enabled() {
			reasons = append(reasons, ReasonProviderNotApproved)
		}
	} else if requireApproval {
		reasons = append(reasons, ReasonProviderNotApproved)
	}
	if candidate.Health.State == CircuitOpen {
		reasons = append(reasons, ReasonProviderCircuitOpen)
	}
	for _, capability := range request.Required {
		if !candidate.Capabilities.Supports(capability) {
			reasons = append(reasons, CapabilityRefusal(capability))
		}
	}
	if err := ValidatePolicy(request.Policy, candidate.Capabilities); err != nil {
		reasons = append(reasons, ReasonPolicyNotSupported)
	}
	if !candidate.Capabilities.SupportsCurrency(request.Currency) {
		reasons = append(reasons, ReasonCurrencyNotSupported)
	}
	if request.AmountKobo > 0 {
		if candidate.Capabilities.MinimumAmountKobo > 0 && request.AmountKobo < candidate.Capabilities.MinimumAmountKobo {
			reasons = append(reasons, ReasonAmountBelowMinimum)
		}
		if candidate.Capabilities.MaximumAmountKobo > 0 && request.AmountKobo > candidate.Capabilities.MaximumAmountKobo {
			reasons = append(reasons, ReasonAmountAboveMaximum)
		}
	}
	candidate.Reasons = reasons
	candidate.Eligible = len(reasons) == 0
	return candidate
}

// approvalOf finds a provider's approval record through any wrappers. A
// provider is normally ResilientProvider(ApprovedAdapter(connector)), and the
// circuit breaker must not be able to answer the approval question on the
// adapter's behalf: a wrapper that forwards Enabled would report an
// unapproved provider and a provider with no approval record identically. The
// bound stops a wrapper cycle from spinning here.
func approvalOf(provider Provider) (EnabledProvider, bool) {
	for depth := 0; provider != nil && depth < 8; depth++ {
		if approval, ok := provider.(EnabledProvider); ok {
			return approval, true
		}
		wrapper, ok := provider.(interface{ Unwrap() Provider })
		if !ok {
			return nil, false
		}
		provider = wrapper.Unwrap()
	}
	return nil, false
}

// pinnedSelection evaluates exactly one provider: the one already recorded
// against a mandate. It is the whole of the pinning rule, shared by Registry
// and Engine so neither can quietly grow a fallback the other lacks.
func pinnedSelection(entry *registration, name string, requireApproval bool, request SelectionRequest) (Selection, error) {
	trimmed := strings.TrimSpace(name)
	if entry == nil {
		return Selection{Name: trimmed, Pinned: true}, &ProviderRefusedError{ProviderName: trimmed, Reasons: []string{ReasonProviderNotRegistered}}
	}
	candidate := evaluateCandidate(entry.provider, entry.role, entry.rank, requireApproval, request)
	selection := Selection{Name: candidate.Name, Pinned: true, Candidates: []Candidate{candidate}}
	if !candidate.Eligible {
		return selection, &ProviderRefusedError{ProviderName: candidate.Name, Reasons: slices.Clone(candidate.Reasons)}
	}
	selection.Provider = entry.provider
	return selection, nil
}

// firstEligibleSelection takes the first eligible provider from an ordered
// list and still evaluates the rest, because the reasons the others were
// passed over are what an operator needs to fix the routing.
func firstEligibleSelection(entries []*registration, requireApproval bool, request SelectionRequest) (Selection, error) {
	selection := Selection{Candidates: make([]Candidate, 0, len(entries))}
	for _, entry := range entries {
		candidate := evaluateCandidate(entry.provider, entry.role, entry.rank, requireApproval, request)
		selection.Candidates = append(selection.Candidates, candidate)
		if candidate.Eligible && selection.Provider == nil {
			selection.Provider = entry.provider
			selection.Name = candidate.Name
		}
	}
	if selection.Provider == nil {
		return selection, &ProviderRefusedError{Reasons: []string{ReasonNoEligibleProvider}}
	}
	return selection, nil
}

type registration struct {
	provider Provider
	role     ProviderRole
	rank     int
	sequence int
}

// Registry holds every collection provider a deployment can reach. It is the
// only place a provider is chosen for a new bank authorization.
type Registry struct {
	mu              sync.RWMutex
	entries         map[string]*registration
	sequence        int
	requireApproval bool
}

func NewRegistry() *Registry { return &Registry{entries: map[string]*registration{}} }

// RequireApproval refuses any provider that carries no approval record at all,
// rather than only those whose record is invalid. A deployment turns it on
// once every provider it can select is wrapped in an ApprovedAdapter; until
// then it would report an approved provider as unapproved.
func (r *Registry) RequireApproval(required bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requireApproval = required
}

// Register adds a provider. Rank orders selection, lowest first; ties are
// broken by registration order, so selection is deterministic and never
// depends on map iteration.
func (r *Registry) Register(provider Provider, role ProviderRole, rank int) error {
	name, err := providerIdentity(provider)
	if err != nil {
		return err
	}
	switch role {
	case RoleActive, RoleReconcileOnly:
	default:
		return errors.New("unknown collection provider role")
	}
	if rank < 0 {
		return errors.New("provider preference rank cannot be negative")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for existing := range r.entries {
		// Two entries a human reads as one name are refused outright. A debit
		// routed to the wrong account because of letter case is not a class of
		// bug worth leaving open.
		if strings.EqualFold(existing, name) {
			return errors.New("provider identity is already registered")
		}
	}
	r.sequence++
	r.entries[name] = &registration{provider: provider, role: role, rank: rank, sequence: r.sequence}
	return nil
}

func providerIdentity(provider Provider) (string, error) {
	if provider == nil {
		return "", errors.New("provider identity is required")
	}
	name := strings.TrimSpace(provider.Name())
	if name == "" {
		return "", errors.New("provider identity is required")
	}
	return name, nil
}

// Resolve returns the provider recorded against work that already exists. It
// is the reconciliation path: it matches by name, ignores role, rank and
// health, and never substitutes another provider. A provider that has been
// removed from configuration must be restored, not replaced.
func (r *Registry) Resolve(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry := r.entries[strings.TrimSpace(name)]
	if entry == nil {
		return nil, errors.New("the original collection provider must be configured to reconcile this attempt")
	}
	return entry.provider, nil
}

// ForMandate returns the provider that holds an existing mandate. It
// evaluates that provider and no other: if it is refused, the collection does
// not happen. Routing a mandate's debit to a different provider is not a
// fallback, it is a debit the buyer never authorized.
func (r *Registry) ForMandate(name string, request SelectionRequest) (Selection, error) {
	r.mu.RLock()
	entry := r.entries[strings.TrimSpace(name)]
	requireApproval := r.requireApproval
	r.mu.RUnlock()
	return pinnedSelection(entry, name, requireApproval, request)
}

// SelectForAuthorization chooses the provider for a mandate that does not yet
// exist. This is the only method that picks between providers, and it is
// reachable only before anything has been authorized, so choosing a different
// provider here cannot move or duplicate an existing instruction.
func (r *Registry) SelectForAuthorization(request SelectionRequest) (Selection, error) {
	return firstEligibleSelection(r.ordered(), r.approvalRequired(), request)
}

// Evaluate reports every registered provider against a request without
// choosing one. The operations surface uses it to show why routing would
// refuse, before anyone tries.
func (r *Registry) Evaluate(request SelectionRequest) []Candidate {
	entries := r.ordered()
	requireApproval := r.approvalRequired()
	candidates := make([]Candidate, 0, len(entries))
	for _, entry := range entries {
		candidates = append(candidates, evaluateCandidate(entry.provider, entry.role, entry.rank, requireApproval, request))
	}
	return candidates
}

// Names lists every registered provider in selection order.
func (r *Registry) Names() []string {
	entries := r.ordered()
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.provider.Name())
	}
	return names
}

func (r *Registry) approvalRequired() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.requireApproval
}

// ordered returns registrations in selection order. Sorting by rank and then
// by registration sequence keeps the choice deterministic; ranging over the
// map alone would let Go's map ordering decide which provider takes a debit.
func (r *Registry) ordered() []*registration {
	r.mu.RLock()
	entries := make([]*registration, 0, len(r.entries))
	for _, entry := range r.entries {
		entries = append(entries, entry)
	}
	r.mu.RUnlock()
	return sortRegistrations(entries)
}

func sortRegistrations(entries []*registration) []*registration {
	slices.SortStableFunc(entries, func(a, b *registration) int {
		if a.rank != b.rank {
			return a.rank - b.rank
		}
		return a.sequence - b.sequence
	})
	return entries
}
