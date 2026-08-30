package db

import "testing"

func TestSetRuntimeDefaultPreservesExplicitDatabaseSetting(t *testing.T) {
	values := map[string]string{"statement_timeout": "45000"}
	setRuntimeDefault(values, "statement_timeout", "30000")
	setRuntimeDefault(values, "lock_timeout", "5000")
	if values["statement_timeout"] != "45000" {
		t.Fatalf("explicit statement timeout was overwritten: %q", values["statement_timeout"])
	}
	if values["lock_timeout"] != "5000" {
		t.Fatalf("default lock timeout was not set: %q", values["lock_timeout"])
	}
}
