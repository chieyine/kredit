package buyers

import (
	"context"
	"testing"

	"kredit/internal/identity"
)

func TestBuyerInvitationIsSingleUseAndCreatesVerifiedPortal(t *testing.T) {
	store := NewStore("test-key", identity.NewMockProvider())
	result, err := store.CreateInvitation("sales-user", "supplier-org", CreateInvitationInput{Target: "buyer@example.test", TargetType: "email", LegalName: "Royal Pharmacy Ltd", BusinessType: "limited_company", BusinessAddress: "Lagos", Industry: "pharmacy"})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := store.Preview(result.RawToken)
	if err != nil {
		t.Fatal(err)
	}
	if preview.ProposedLegalName != "Royal Pharmacy Ltd" || preview.Status != "pending" {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	if _, target, err := store.InvitationTarget(result.RawToken); err != nil || target != "buyer@example.test" {
		t.Fatalf("unexpected invitation target: %q, %v", target, err)
	}
	portal, err := store.Accept(context.Background(), result.RawToken, "buyer-user", AcceptInput{FullName: "Buyer Representative"})
	if err != nil {
		t.Fatal(err)
	}
	if portal.Person.Status != "verified" || portal.Business.Status != "verified" || portal.Representative.AuthorityStatus != "verified" {
		t.Fatalf("unexpected portal verification state: %+v", portal)
	}
	if len(portal.VerificationCases) != 3 || len(portal.Consents) != 3 {
		t.Fatalf("expected identity cases and consents: cases=%d consents=%d", len(portal.VerificationCases), len(portal.Consents))
	}
	if _, err := store.Accept(context.Background(), result.RawToken, "other-user", AcceptInput{FullName: "Replay"}); err == nil {
		t.Fatal("accepted invitation must not be reusable")
	}
}
