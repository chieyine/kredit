package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestConfigCheckFailsClosedWithoutLaunchEvidence(t *testing.T) {
	command := exec.Command("go", "run", ".")
	command.Env = append(command.Environ(), "APP_ENV=production", "LEGAL_APPROVAL_REFERENCE=")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("expected incomplete production configuration to fail")
	}
	if !strings.Contains(string(output), "production configuration rejected") {
		t.Fatalf("unexpected output: %s", output)
	}
}
