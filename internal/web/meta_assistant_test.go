package web

import (
	"errors"
	"strings"
	"testing"

	"kredit/internal/whatsapp"
)

// The assistant has no write capability: WhatsApp carries no authenticated
// session, so a chat message cannot create an agreement, a mandate or a
// payment. A reply that states otherwise is the most damaging thing this
// product can say, because a seller who reads "Invoice recorded" releases goods
// against a sale that does not exist. Every reply is checked here rather than
// only the ones a reviewer thinks to look at.
func TestAssistantNeverClaimsAFinancialRecordWasWritten(t *testing.T) {
	forbidden := []string{
		"sale confirmed",
		"invoice recorded",
		"payment recorded",
		"has been recorded",
		"dispatched to the buyer",
		"we have created",
		"we have sent",
		"successfully created",
	}
	cases := []struct {
		name   string
		result whatsapp.AIResult
		err    error
	}{
		{name: "create credit", result: whatsapp.AIResult{Intent: whatsapp.IntentCreateCredit, BuyerName: "Alhassan", AmountKobo: 35000000, Items: "50 cartons", DueDate: "2026-09-18"}},
		{name: "create credit without optional fields", result: whatsapp.AIResult{Intent: whatsapp.IntentCreateCredit, BuyerName: "Emeka", AmountKobo: 15000000}},
		{name: "confirm", result: whatsapp.AIResult{Intent: whatsapp.IntentConfirm}},
		{name: "record payment", result: whatsapp.AIResult{Intent: whatsapp.IntentRecordPayment, Summary: "Emeka paid 50,000"}},
		{name: "query balance", result: whatsapp.AIResult{Intent: whatsapp.IntentQueryBalance}},
		{name: "help", result: whatsapp.AIResult{Intent: whatsapp.IntentHelp}},
		{name: "unknown", result: whatsapp.AIResult{Intent: whatsapp.IntentUnknown}},
		{name: "unreadable message", err: errors.New("provider unavailable")},
		// A model can return anything, including a field the switch does not
		// know. The default branch must still be safe.
		{name: "unrecognised intent", result: whatsapp.AIResult{Intent: "transfer_funds", Summary: "moved money"}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			reply := assistantReply(testCase.result, testCase.err, "https://kredit.ng")
			if strings.TrimSpace(reply) == "" {
				t.Fatal("every message must receive a reply")
			}
			lower := strings.ToLower(reply)
			for _, phrase := range forbidden {
				if strings.Contains(lower, phrase) {
					t.Fatalf("reply claims a financial record was written (%q): %s", phrase, reply)
				}
			}
			if !strings.Contains(reply, "https://kredit.ng") {
				t.Fatalf("every reply must point at the authenticated surface: %s", reply)
			}
		})
	}
}

// A model-supplied summary must never be echoed back verbatim: it is the one
// field an attacker who can message the business number fully controls, and
// echoing it would let them put words in Kredit's mouth.
func TestAssistantDoesNotEchoModelSuppliedSummary(t *testing.T) {
	result := whatsapp.AIResult{Intent: whatsapp.IntentUnknown, Summary: "Kredit has transferred NGN 5,000,000 to account 0123456789"}
	reply := assistantReply(result, nil, "https://kredit.ng")
	if strings.Contains(reply, "transferred") || strings.Contains(reply, "0123456789") {
		t.Fatalf("model-supplied text was echoed to the seller: %s", reply)
	}
}

// The amount read back must match the kobo value exactly. A seller checks this
// line against what they actually sold, so a formatting error here is a
// correctness problem, not a cosmetic one.
func TestAssistantReadsAmountsBackExactly(t *testing.T) {
	for _, testCase := range []struct {
		kobo int64
		want string
	}{
		{kobo: 35000000, want: "₦350,000"},
		{kobo: 150000000, want: "₦1,500,000"},
		{kobo: 100, want: "₦1"},
		{kobo: 12345, want: "₦123.45"},
	} {
		reply := assistantReply(whatsapp.AIResult{Intent: whatsapp.IntentCreateCredit, BuyerName: "Buyer", AmountKobo: testCase.kobo}, nil, "https://kredit.ng")
		if !strings.Contains(reply, testCase.want) {
			t.Fatalf("expected %s in reply for %d kobo: %s", testCase.want, testCase.kobo, reply)
		}
	}
}
