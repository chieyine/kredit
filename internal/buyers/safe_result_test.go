package buyers

import "testing"

func TestSafeResultTextReadsBooleansAndStrings(t *testing.T) {
	got := safeResultText(map[string]any{"name_match": true, "registration_match": "true"})
	if got["name_match"] != "true" || got["registration_match"] != "true" {
		t.Fatalf("unexpected result %#v", got)
	}
	if safeResultText(nil) != nil {
		t.Fatal("a missing result should stay missing")
	}
}
