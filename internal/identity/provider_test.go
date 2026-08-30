package identity

import (
	"context"
	"testing"
)

func TestMockProviderReturnsSafeVerifiedResults(t *testing.T) {
	provider := NewMockProvider()
	session, err := provider.CreatePersonVerification(context.Background(), PersonVerificationInput{SubjectID: "person-1", FullName: "Ada Example"})
	if err != nil {
		t.Fatal(err)
	}
	if session.State != "verified" || session.VerificationLevel != 2 {
		t.Fatalf("unexpected verification session: %+v", session)
	}
	result, err := provider.GetVerification(context.Background(), session.ProviderID)
	if err != nil {
		t.Fatal(err)
	}
	if result.SafeResult["verified_name"] != "Ada Example" {
		t.Fatalf("unexpected safe result: %+v", result.SafeResult)
	}
}

func TestUnavailableProviderFailsClosed(t *testing.T) {
	p := NewUnavailableProvider("identity adapter missing")
	if p.Capabilities() != (ProviderCapabilities{}) {
		t.Fatal("unavailable provider must expose no capabilities")
	}
	if _, err := p.CreatePersonVerification(context.Background(), PersonVerificationInput{SubjectID: "person-1", FullName: "Test"}); err == nil {
		t.Fatal("expected unavailable identity provider error")
	}
}
