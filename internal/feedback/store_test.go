package feedback

import (
	"context"
	"testing"
)

func TestSubmitValidatesAndStoresPrivacySafeFeedback(t *testing.T) {
	store := NewStore()
	entry, err := store.Submit(context.Background(), Input{UserID: "11111111-1111-4111-8111-111111111111", OrganizationID: "22222222-2222-4222-8222-222222222222", Area: "seller", Screen: "overview", Answer: "partly"})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Answer != "partly" || entry.Area != "seller" || entry.CreatedAt.IsZero() {
		t.Fatalf("unexpected entry: %#v", entry)
	}
}

func TestSubmitRejectsUnscopedOrFreeFormValues(t *testing.T) {
	store := NewStore()
	tests := []Input{
		{UserID: "bad", Area: "buyer", Screen: "overview", Answer: "yes"},
		{UserID: "11111111-1111-4111-8111-111111111111", Area: "seller", Screen: "overview", Answer: "yes"},
		{UserID: "11111111-1111-4111-8111-111111111111", Area: "buyer", Screen: "other", Answer: "yes"},
		{UserID: "11111111-1111-4111-8111-111111111111", Area: "buyer", Screen: "overview", Answer: "maybe"},
	}
	for _, input := range tests {
		if _, err := store.Submit(context.Background(), input); err == nil {
			t.Fatalf("expected validation error for %#v", input)
		}
	}
}

func TestSubmitCountsOneAnswerPerPersonAndPageEachMonth(t *testing.T) {
	store := NewStore()
	input := Input{UserID: "11111111-1111-4111-8111-111111111111", Area: "buyer", Screen: "overview", Answer: "yes"}
	if _, err := store.Submit(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	input.Answer = "no"
	if _, err := store.Submit(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if len(store.rows) != 1 || store.rows[0].Answer != "yes" {
		t.Fatalf("expected first monthly answer only, got %#v", store.rows)
	}
}
