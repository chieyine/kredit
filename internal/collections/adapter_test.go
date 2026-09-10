package collections

import (
	"context"
	"testing"
	"time"

	"kredit/internal/ledger"
)

func TestApprovedAdapterRequiresWrittenApprovalAndPilotLimit(t *testing.T) {
	mock := NewMockProvider("secret")
	adapter := NewApprovedAdapter(mock, ApprovalRecord{}, true)
	if _, err := adapter.Submit(context.Background(), Request{AmountKobo: 100}); err == nil {
		t.Fatal("expected approval gate")
	}
	approval := ApprovalRecord{ProviderName: mock.Name(), WrittenReference: "approval-123", ApprovedBy: "compliance", ApprovedAt: time.Now().UTC().Add(-time.Minute), AllowedCapabilities: []Capability{CapabilityOneTime}, PilotLimitKobo: 100}
	adapter = NewApprovedAdapter(mock, approval, true)
	if _, err := adapter.Submit(context.Background(), Request{AmountKobo: 101}); err == nil {
		t.Fatal("expected pilot limit gate")
	}
	if _, err := adapter.Submit(context.Background(), Request{AmountKobo: 100}); err != nil {
		t.Fatal(err)
	}
}

func TestProviderStatusExposesSandboxCapabilities(t *testing.T) {
	mock := NewMockProvider("secret")
	engine := NewEngine(mock, nil, func(string) (ObligationSnapshot, error) { return ObligationSnapshot{}, nil }, func(string, time.Time) (ledger.Money, error) { return 0, nil })
	status := engine.ProviderStatus()
	if status.Name == "" || !status.Capabilities.Settlement {
		t.Fatalf("unexpected provider status: %#v", status)
	}
}

func TestApprovalCapabilitiesCannotBeChangedThroughSharedSlices(t *testing.T) {
	mock := NewMockProvider("secret")
	approval := ApprovalRecord{ProviderName: mock.Name(), WrittenReference: "approval", ApprovedBy: "reviewer", ApprovedAt: time.Now().Add(-time.Minute), AllowedCapabilities: []Capability{CapabilityOneTime}, PilotLimitKobo: 100}
	adapter := NewApprovedAdapter(mock, approval, true)
	approval.AllowedCapabilities[0] = CapabilityReversal
	returned := adapter.Approval()
	returned.AllowedCapabilities[0] = CapabilityReversal
	if _, err := adapter.Cancel(context.Background(), "collection"); err == nil {
		t.Fatal("external mutation authorized cancellation")
	}
	if !adapter.Approval().Allows(CapabilityOneTime) {
		t.Fatal("approval changed")
	}
	for _, amount := range []ledger.Money{0, -1} {
		if _, err := adapter.Submit(context.Background(), Request{AmountKobo: amount}); err == nil {
			t.Fatal("nonpositive debit accepted")
		}
	}
}
