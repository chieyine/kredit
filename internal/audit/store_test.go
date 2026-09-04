package audit

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

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

func TestEventJSONUsesThePublicActivitySchema(t *testing.T) {
	event := Event{ID: "e", At: time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC), ActorUserID: "u", OrganizationID: "o", Action: "member.invited", ResourceType: "member", ResourceID: "m", RequestID: "r"}
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, field := range []string{`"created_at"`, `"actor_user_id"`, `"organization_id"`, `"resource_type"`, `"request_id"`} {
		if !strings.Contains(text, field) {
			t.Fatalf("missing %s in %s", field, text)
		}
	}
	if strings.Contains(text, `"At"`) {
		t.Fatalf("internal field names leaked: %s", text)
	}
}
