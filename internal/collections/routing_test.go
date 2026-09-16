package collections

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"kredit/internal/ledger"
	"kredit/internal/payments"
)

type routingProvider struct {
	name         string
	capabilities Capabilities
	response     Response
	submitErr    error
	submits      int
}

func (p *routingProvider) Name() string               { return p.name }
func (p *routingProvider) Capabilities() Capabilities { return p.capabilities }
func (p *routingProvider) Submit(_ context.Context, request Request) (Response, error) {
	p.submits++
	if p.submitErr != nil {
		return Response{}, p.submitErr
	}
	response := p.response
	if response.State == "" {
		response = Response{State: ProviderSucceeded, SucceededAmountKobo: request.AmountKobo}
	}
	if response.ProviderCollectionID == "" {
		response.ProviderCollectionID = p.name + "-collection"
	}
	return response, nil
}
func (p *routingProvider) Get(_ context.Context, id string) (Response, error) {
	return Response{State: ProviderPending, ProviderCollectionID: id}, nil
}

// The signature carries the provider name so a webhook authenticated by one
// provider can never be accepted by another.
func (p *routingProvider) Sign(Webhook) string { return "signed:" + p.name }
func (p *routingProvider) VerifyWebhook(event Webhook) bool {
	return event.Signature == "signed:"+p.name
}

type openCircuitProvider struct{ *routingProvider }

func (openCircuitProvider) Health() HealthStatus {
	return HealthStatus{State: CircuitOpen, Healthy: false}
}

type unapprovedProvider struct{ *routingProvider }

func (unapprovedProvider) Enabled() bool { return false }

type approvedProvider struct{ *routingProvider }

func (approvedProvider) Enabled() bool { return true }

// wrappingProvider embeds the Provider interface, so it exposes no approval of
// its own. It stands in for ResilientProvider, which must not be able to
// answer the approval question on the adapter's behalf.
type wrappingProvider struct{ Provider }

func (w wrappingProvider) Unwrap() Provider { return w.Provider }

func everyCapability() Capabilities {
	return Capabilities{AuthorizationSession: true, OneTime: true, Recurring: true, Variable: true, Settlement: true, Reversal: true}
}

func routingEngine(t *testing.T, provider Provider) (*Engine, *ObligationSnapshot) {
	t.Helper()
	snapshot := &ObligationSnapshot{ID: "obl-1", BuyerUserID: "buyer", Currency: "NGN", Active: true, OutstandingKobo: 100000, MandateActive: true, MandateRemainingKobo: 100000, CollectionEnabled: true, ProviderSupported: true, Version: 1}
	apply := func(_ string, delta ledger.Money) error { snapshot.OutstandingKobo += delta; return nil }
	paymentStore := payments.NewStore(ledger.NewStore(), func(_ string) (payments.ObligationSnapshot, error) {
		return payments.ObligationSnapshot{ID: snapshot.ID, BuyerUserID: snapshot.BuyerUserID, PrincipalKobo: 100000, OutstandingKobo: snapshot.OutstandingKobo, Currency: "NGN"}, nil
	}, apply)
	engine := NewEngine(provider, paymentStore, func(string) (ObligationSnapshot, error) { return *snapshot, nil }, func(string, time.Time) (ledger.Money, error) { return 100000, nil })
	return engine, snapshot
}

// Selection order must come from configuration, not from Go's map ordering.
// Repeating the run makes a map-iteration dependency fail rather than flake.
func TestProviderSelectionOrderIsDeterministic(t *testing.T) {
	want := []string{"first", "tie-registered-first", "tie-registered-second", "last"}
	for run := 0; run < 64; run++ {
		registry := NewRegistry()
		for _, entry := range []struct {
			name string
			rank int
		}{{"last", 9}, {"first", 0}, {"tie-registered-first", 4}, {"tie-registered-second", 4}} {
			if err := registry.Register(&routingProvider{name: entry.name, capabilities: everyCapability()}, RoleActive, entry.rank); err != nil {
				t.Fatal(err)
			}
		}
		if names := registry.Names(); !slices.Equal(names, want) {
			t.Fatalf("selection order %v, want %v", names, want)
		}
		selection, err := registry.SelectForAuthorization(SelectionRequest{Required: []Capability{CapabilityAuthorization}})
		if err != nil || selection.Name != "first" {
			t.Fatalf("selection=%q err=%v", selection.Name, err)
		}
	}
}

// The provider that holds the mandate takes the debit even when another
// provider is ranked ahead of it and is perfectly healthy.
func TestMandateProviderIsUsedEvenWhenAnotherRanksAhead(t *testing.T) {
	preferred := &routingProvider{name: "preferred", capabilities: everyCapability()}
	engine, snapshot := routingEngine(t, preferred)
	holder := &routingProvider{name: "mandate-holder", capabilities: everyCapability()}
	if err := engine.RegisterActiveProvider(holder, 7); err != nil {
		t.Fatal(err)
	}
	if selection, err := engine.SelectProvider(SelectionRequest{}); err != nil || selection.Name != "preferred" {
		t.Fatalf("a new authorization should prefer the ranked provider: %q %v", selection.Name, err)
	}
	snapshot.MandateProvider = "mandate-holder"
	attempt, err := engine.Start(context.Background(), snapshot.ID, "pinned", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Provider != "mandate-holder" || holder.submits != 1 {
		t.Fatalf("debit left the mandate's provider: attempt=%q submits=%d", attempt.Provider, holder.submits)
	}
	if preferred.submits != 0 {
		t.Fatal("a mandate's debit was routed to a provider that does not hold it")
	}
}

// A provider whose circuit is open takes its own mandates' debits and fails.
// The failure must stay with it: the money may already have moved, so offering
// the same debit to a healthy provider is how a buyer is charged twice.
func TestFailedDebitIsNeverOfferedToASecondProvider(t *testing.T) {
	healthy := &routingProvider{name: "healthy", capabilities: everyCapability()}
	engine, snapshot := routingEngine(t, healthy)
	broken := &routingProvider{name: "broken", capabilities: everyCapability(), submitErr: errors.New("provider timeout")}
	if err := engine.RegisterActiveProvider(openCircuitProvider{broken}, 1); err != nil {
		t.Fatal(err)
	}
	snapshot.MandateProvider = "broken"
	attempt, err := engine.Start(context.Background(), snapshot.ID, "no-failover", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if attempt.State != AttemptUnknown {
		t.Fatalf("an unacknowledged debit must stay unknown, got %q", attempt.State)
	}
	if attempt.Provider != "broken" || broken.submits != 1 {
		t.Fatalf("attempt=%q submits=%d", attempt.Provider, broken.submits)
	}
	if healthy.submits != 0 {
		t.Fatal("an unknown debit outcome was retried against another provider")
	}
	// A retry stays with the originating provider whether or not it is allowed
	// to proceed. The invariant is about where the debit can go, not about
	// whether the retry succeeds.
	_, _ = engine.Retry(context.Background(), attempt.ID, time.Now().UTC())
	if healthy.submits != 0 {
		t.Fatal("a retry moved an unknown debit to another provider")
	}
}

func TestUnregisteredMandateProviderRefusesTheCollection(t *testing.T) {
	active := &routingProvider{name: "active", capabilities: everyCapability()}
	engine, snapshot := routingEngine(t, active)
	snapshot.MandateProvider = "a-provider-nobody-configured"
	eligibility, err := engine.Eligibility(snapshot.ID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if eligibility.Eligible || !slices.Contains(eligibility.Reasons, "mandate_provider_mismatch") {
		t.Fatalf("eligibility=%+v", eligibility)
	}
	if _, err = engine.Start(context.Background(), snapshot.ID, "unknown-provider", time.Now().UTC()); err == nil {
		t.Fatal("a debit was accepted for a provider this deployment cannot reach")
	}
	if active.submits != 0 {
		t.Fatal("an unresolvable mandate fell back to the configured provider")
	}
}

// A retained provider keeps reconciling what it already holds and is never
// chosen again, for a new authorization or for a mandate it still holds.
func TestRetainedProviderReconcilesButIsNeverDebitedAgain(t *testing.T) {
	active := &routingProvider{name: "active", capabilities: everyCapability()}
	engine, snapshot := routingEngine(t, active)
	retired := &routingProvider{name: "retired", capabilities: everyCapability()}
	if err := engine.RegisterRetainedProvider(retired); err != nil {
		t.Fatal(err)
	}
	snapshot.MandateProvider = "retired"
	eligibility, err := engine.Eligibility(snapshot.ID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if eligibility.Eligible || !slices.Contains(eligibility.Reasons, ReasonProviderReconcileOnly) {
		t.Fatalf("a retained provider should be refused by name: %+v", eligibility.Reasons)
	}
	if _, err = engine.Start(context.Background(), snapshot.ID, "retired-debit", time.Now().UTC()); err == nil {
		t.Fatal("a retained provider took a new debit")
	}
	if retired.submits != 0 || active.submits != 0 {
		t.Fatalf("retired=%d active=%d", retired.submits, active.submits)
	}
	if _, err := engine.providerFor("retired"); err != nil {
		t.Fatalf("a retained provider must still reconcile its own attempts: %v", err)
	}
	selection, err := engine.SelectProvider(SelectionRequest{})
	if err != nil || selection.Name != "active" {
		t.Fatalf("selection=%q err=%v", selection.Name, err)
	}
	for _, candidate := range selection.Candidates {
		if candidate.Name == "retired" {
			t.Fatal("a retained provider was offered as a candidate for a new authorization")
		}
	}
}

// Pinning refuses rather than substituting. The refusal names the reason and
// returns no provider at all.
func TestPinnedMandateNeverFallsBackToAnotherProvider(t *testing.T) {
	registry := NewRegistry()
	healthy := &routingProvider{name: "healthy", capabilities: everyCapability()}
	broken := &routingProvider{name: "broken", capabilities: everyCapability()}
	if err := registry.Register(healthy, RoleActive, 0); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(openCircuitProvider{broken}, RoleActive, 1); err != nil {
		t.Fatal(err)
	}
	selection, err := registry.ForMandate("broken", SelectionRequest{})
	if selection.Provider != nil {
		t.Fatal("a refused mandate provider still returned a provider")
	}
	if len(selection.Candidates) != 1 || selection.Candidates[0].Name != "broken" {
		t.Fatalf("pinning considered more than the mandate's provider: %+v", selection.Candidates)
	}
	var refused *ProviderRefusedError
	if !errors.As(err, &refused) || !slices.Contains(refused.Reasons, ReasonProviderCircuitOpen) {
		t.Fatalf("err=%v", err)
	}
	// The same registry would happily select the healthy provider for a new
	// authorization. Pinning is what stops it here, not availability.
	if fresh, selectErr := registry.SelectForAuthorization(SelectionRequest{}); selectErr != nil || fresh.Name != "healthy" {
		t.Fatalf("fresh=%q err=%v", fresh.Name, selectErr)
	}
}

func TestSelectionSkipsUnapprovedProviders(t *testing.T) {
	registry := NewRegistry()
	unapproved := &routingProvider{name: "unapproved", capabilities: everyCapability()}
	approved := &routingProvider{name: "approved", capabilities: everyCapability()}
	if err := registry.Register(unapprovedProvider{unapproved}, RoleActive, 0); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(approvedProvider{approved}, RoleActive, 1); err != nil {
		t.Fatal(err)
	}
	selection, err := registry.SelectForAuthorization(SelectionRequest{})
	if err != nil || selection.Name != "approved" {
		t.Fatalf("selection=%q err=%v", selection.Name, err)
	}
	if !slices.Contains(selection.Candidates[0].Reasons, ReasonProviderNotApproved) {
		t.Fatalf("reasons=%v", selection.Candidates[0].Reasons)
	}
}

// A wrapper must not be able to answer the approval question for the provider
// it wraps. Routing looks through wrappers for the real approval record.
func TestApprovalIsFoundThroughWrappers(t *testing.T) {
	inner := unapprovedProvider{&routingProvider{name: "wrapped", capabilities: everyCapability()}}
	registry := NewRegistry()
	if err := registry.Register(wrappingProvider{inner}, RoleActive, 0); err != nil {
		t.Fatal(err)
	}
	candidates := registry.Evaluate(SelectionRequest{})
	if len(candidates) != 1 || !slices.Contains(candidates[0].Reasons, ReasonProviderNotApproved) {
		t.Fatalf("a wrapper hid an unapproved provider: %+v", candidates)
	}
	if _, err := registry.SelectForAuthorization(SelectionRequest{}); err == nil {
		t.Fatal("an unapproved provider was selected through a wrapper")
	}
}

// Without an approval record there is nothing to check, which is right in
// development and wrong in production. RequireApproval is the difference.
func TestRequireApprovalRefusesProvidersWithNoApprovalRecord(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(&routingProvider{name: "unrecorded", capabilities: everyCapability()}, RoleActive, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.SelectForAuthorization(SelectionRequest{}); err != nil {
		t.Fatalf("a provider with no approval record must be selectable in development: %v", err)
	}
	registry.RequireApproval(true)
	if _, err := registry.SelectForAuthorization(SelectionRequest{}); err == nil {
		t.Fatal("production selected a provider carrying no written approval")
	}
}

func TestCapabilityAndAmountBoundsRefusals(t *testing.T) {
	capabilities := everyCapability()
	capabilities.Variable = false
	capabilities.MinimumAmountKobo = 25000
	capabilities.MaximumAmountKobo = 2000000000
	registry := NewRegistry()
	if err := registry.Register(&routingProvider{name: "bounded", capabilities: capabilities}, RoleActive, 0); err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		name    string
		request SelectionRequest
		reason  string
	}{
		{"missing capability", SelectionRequest{Required: []Capability{CapabilityVariable}}, CapabilityRefusal(CapabilityVariable)},
		{"below minimum", SelectionRequest{AmountKobo: 24999}, ReasonAmountBelowMinimum},
		{"above maximum", SelectionRequest{AmountKobo: 2000000001}, ReasonAmountAboveMaximum},
		{"unknown capability", SelectionRequest{Required: []Capability{Capability("a_capability_this_build_does_not_know")}}, CapabilityRefusal("a_capability_this_build_does_not_know")},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			candidates := registry.Evaluate(testCase.request)
			if len(candidates) != 1 || candidates[0].Eligible || !slices.Contains(candidates[0].Reasons, testCase.reason) {
				t.Fatalf("candidates=%+v want reason %q", candidates, testCase.reason)
			}
		})
	}
	if candidates := registry.Evaluate(SelectionRequest{AmountKobo: 25000, Required: []Capability{CapabilityOneTime}}); !candidates[0].Eligible {
		t.Fatalf("an in-bounds request was refused: %+v", candidates[0].Reasons)
	}
}

func TestDeclaredCurrenciesAreEnforcedAndAbsenceIsNotRefusal(t *testing.T) {
	undeclared := everyCapability()
	declared := everyCapability()
	declared.SupportedCurrencies = []string{"NGN"}
	if !undeclared.SupportsCurrency("USD") {
		t.Fatal("a provider that declares no currencies must not be refused for one")
	}
	if !declared.SupportsCurrency("ngn") || declared.SupportsCurrency("USD") {
		t.Fatal("declared currency matching is wrong")
	}
	registry := NewRegistry()
	if err := registry.Register(&routingProvider{name: "naira-only", capabilities: declared}, RoleActive, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.SelectForAuthorization(SelectionRequest{Currency: "USD"}); err == nil {
		t.Fatal("a provider was selected for a currency it does not support")
	}
}

func TestCapabilitiesSupportsIsTotalAndFailsClosed(t *testing.T) {
	full := Capabilities{AuthorizationSession: true, OneTime: true, Recurring: true, Variable: true, Settlement: true, Reversal: true, MultiAccount: true, PartialRecovery: true, AutomaticRetries: true}
	for _, capability := range []Capability{CapabilityAuthorization, CapabilityOneTime, CapabilityRecurring, CapabilityVariable, CapabilitySettlement, CapabilityReversal, CapabilityMultiAccount, CapabilityPartialRecovery, CapabilityAutomaticRetries} {
		if !full.Supports(capability) {
			t.Fatalf("%q is not mapped to a capability flag", capability)
		}
		if (Capabilities{}).Supports(capability) {
			t.Fatalf("%q was reported by a provider that declares nothing", capability)
		}
	}
	if full.Supports(Capability("")) || full.Supports(Capability("future_capability")) {
		t.Fatal("an unrecognised capability must not be reported as supported")
	}
}

func TestRegistrationRefusesAmbiguousOrInvalidIdentities(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(&routingProvider{name: "paystack"}, RoleActive, 0); err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		name     string
		provider Provider
		role     ProviderRole
		rank     int
	}{
		{"same identity", &routingProvider{name: "paystack"}, RoleActive, 1},
		{"identity differing only by case", &routingProvider{name: "PayStack"}, RoleActive, 1},
		{"no identity", &routingProvider{name: "   "}, RoleActive, 1},
		{"nil provider", nil, RoleActive, 1},
		{"unknown role", &routingProvider{name: "other"}, ProviderRole("preferred"), 1},
		{"negative rank", &routingProvider{name: "other"}, RoleActive, -1},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := registry.Register(testCase.provider, testCase.role, testCase.rank); err == nil {
				t.Fatal("registration was accepted")
			}
		})
	}
	if names := registry.Names(); len(names) != 1 {
		t.Fatalf("a refused registration was stored: %v", names)
	}
}

func TestEngineRefusesAmbiguousProviderIdentities(t *testing.T) {
	active := &routingProvider{name: "active", capabilities: everyCapability()}
	engine, _ := routingEngine(t, active)
	if err := engine.RegisterActiveProvider(&routingProvider{name: "ACTIVE"}, 1); err == nil {
		t.Fatal("a second provider was registered under the active provider's identity")
	}
	if err := engine.RegisterActiveProvider(&routingProvider{name: "second"}, 1); err != nil {
		t.Fatal(err)
	}
	if err := engine.RegisterRetainedProvider(&routingProvider{name: "Second"}); err == nil {
		t.Fatal("a retained provider was registered under an active provider's identity")
	}
}

func TestProviderStatusReportsEveryProviderWithItsReasons(t *testing.T) {
	active := &routingProvider{name: "active", capabilities: everyCapability()}
	engine, _ := routingEngine(t, active)
	if err := engine.RegisterActiveProvider(unapprovedProvider{&routingProvider{name: "pending-approval", capabilities: everyCapability()}}, 3); err != nil {
		t.Fatal(err)
	}
	if err := engine.RegisterRetainedProvider(&routingProvider{name: "retired", capabilities: everyCapability()}); err != nil {
		t.Fatal(err)
	}
	status := engine.ProviderStatus()
	if status.Name != "active" || len(status.Providers) != 3 {
		t.Fatalf("status=%+v", status)
	}
	byName := map[string]Candidate{}
	for _, candidate := range status.Providers {
		byName[candidate.Name] = candidate
	}
	if !byName["active"].Eligible {
		t.Fatalf("active reasons=%v", byName["active"].Reasons)
	}
	if !slices.Contains(byName["pending-approval"].Reasons, ReasonProviderNotApproved) {
		t.Fatalf("pending reasons=%v", byName["pending-approval"].Reasons)
	}
	if byName["retired"].Role != RoleReconcileOnly || !slices.Contains(byName["retired"].Reasons, ReasonProviderReconcileOnly) {
		t.Fatalf("retired=%+v", byName["retired"])
	}
}

func TestRefusalErrorNamesTheProviderAndReasons(t *testing.T) {
	err := &ProviderRefusedError{ProviderName: "paystack", Reasons: []string{ReasonProviderCircuitOpen, ReasonAmountBelowMinimum}}
	message := err.Error()
	if !strings.Contains(message, "paystack") || !strings.Contains(message, ReasonProviderCircuitOpen) || !strings.Contains(message, ReasonAmountBelowMinimum) {
		t.Fatalf("message=%q", message)
	}
	empty := &ProviderRefusedError{Reasons: []string{ReasonNoEligibleProvider}}
	if empty.Error() == "" {
		t.Fatal("a refusal with no provider must still explain itself")
	}
}

func TestResolveNeverSubstitutesAnotherProvider(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(&routingProvider{name: "only"}, RoleActive, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Resolve("removed-from-configuration"); err == nil {
		t.Fatal("reconciliation fell back to a provider that never held the attempt")
	}
	provider, err := registry.Resolve("only")
	if err != nil || provider.Name() != "only" {
		t.Fatalf("provider=%v err=%v", provider, err)
	}
}
