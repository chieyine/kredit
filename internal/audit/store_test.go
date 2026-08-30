package audit

import "testing"

func TestAppendDropsSensitiveMetadata(t *testing.T) {
	store := NewStore()
	event := store.Append(Event{OrganizationID: "org", Metadata: map[string]string{
		"phone":       "2348000000000",
		"safe_code":   "invite.accepted",
		"description": "line\nvalue",
	}})
	if _, ok := event.Metadata["phone"]; ok {
		t.Fatal("phone metadata must not be retained")
	}
	if event.Metadata["safe_code"] != "invite.accepted" || event.Metadata["description"] != "line value" {
		t.Fatalf("unexpected sanitized metadata: %#v", event.Metadata)
	}
}
