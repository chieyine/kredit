package ledger

import "testing"

func TestJournalReplayAcceptsReorderedPostingsButPreservesMultiplicity(t *testing.T) {
	store := NewStore()
	original := Transaction{EventType: "audit", ReferenceType: "payment", ReferenceID: "one", IdempotencyKey: "audit-replay", Postings: []Posting{{Account: "a", Debit: 10}, {Account: "a", Debit: 10}, {Account: "b", Credit: 20}}}
	first, err := store.post(original)
	if err != nil {
		t.Fatal(err)
	}
	replay := original
	replay.Postings = []Posting{original.Postings[2], original.Postings[0], original.Postings[1]}
	got, err := store.post(replay)
	if err != nil || got.ID != first.ID {
		t.Fatalf("reordered replay: %v %v", got, err)
	}
	replay.Postings = []Posting{{Account: "a", Debit: 20}, {Account: "b", Credit: 10}, {Account: "b", Credit: 10}}
	if _, err := store.post(replay); err == nil {
		t.Fatal("balanced but changed posting multiplicity was accepted")
	}
}
