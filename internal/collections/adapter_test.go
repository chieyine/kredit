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
