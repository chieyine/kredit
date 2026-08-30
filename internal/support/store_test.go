package support

import "testing"

func TestCaseTimelineIsAppendOnly(t *testing.T) {
	store := NewStore()
	item, err := store.Open("payment", "obligation-1", "operator-1", "org-1", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Transition(item.ID, "operator-1", InProgress, "review started"); err != nil {
		t.Fatal(err)
	}
	if got := len(store.Timeline(item.ID)); got != 2 {
		t.Fatalf("timeline length = %d, want 2", got)
	}
}
