package whatsapp

import (
	"context"
	"testing"
)

func TestSignedWebhookParsesStructuredCreateCommandAndDeduplicates(t *testing.T) {
	handler := NewHandler("secret")
	event := Event{ID: "wa-1", From: "2348000000000", Text: "Create credit\nRoyal Pharmacy, ₦1.2m, 30 September 2026"}
	event.Signature = handler.Sign(event)
	command, err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Fatal(err)
	}
	if command.Kind != CommandCreateCredit || command.BuyerName != "Royal Pharmacy" || command.AmountKobo != 120000000 || !command.RequiresConfirmation {
		t.Fatalf("command=%+v", command)
	}
	duplicate, err := handler.Handle(context.Background(), event)
	if err != nil || duplicate.Kind != "" {
		t.Fatalf("duplicate=%+v err=%v", duplicate, err)
	}
}
func TestPaymentCommandNeverAcceptsCredentials(t *testing.T) {
	command, err := ParseCommand("Royal Pharmacy paid ₦500,000")
	if err != nil {
		t.Fatal(err)
	}
	if command.Kind != CommandRecordPayment || command.AmountKobo != 50000000 {
		t.Fatalf("command=%+v", command)
	}
	if _, err := ParseCommand("send my BVN 123"); err == nil {
		t.Fatal("sensitive command should not be supported")
	}
	confirmCmd, err := ParseCommand("YES")
	if err != nil || confirmCmd.Kind != CommandConfirm {
		t.Fatalf("expected confirm command, got %+v, err=%v", confirmCmd, err)
	}
}

func TestParseAmountUsesExactIntegerKoboArithmetic(t *testing.T) {
	tests := map[string]int64{
		"₦1.25": 125,
		"1.2m":  120000000,
		"2.5k":  250000,
	}
	for input, want := range tests {
		got, err := parseAmount(input)
		if err != nil || got != want {
			t.Fatalf("parseAmount(%q)=%d/%v, want %d", input, got, err, want)
		}
	}
	if _, err := parseAmount("0.001"); err == nil {
		t.Fatal("fractional kobo must be rejected")
	}
}
