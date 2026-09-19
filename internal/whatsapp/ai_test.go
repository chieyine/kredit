package whatsapp

import (
	"testing"
	"time"
)

// The previous test here called the live Gemini endpoint and skipped whenever
// GEMINI_API_KEY was unset, which is always true in CI. It therefore never ran,
// and when it did run it asserted exact field values from a non-deterministic
// model. The behaviour worth pinning is ours, not the model's.

func TestAssistantIsDisabledWithoutAKey(t *testing.T) {
	if (&AIParser{}).Enabled() {
		t.Fatal("a parser with no key must report itself disabled")
	}
	var missing *AIParser
	if missing.Enabled() {
		t.Fatal("a nil parser must report itself disabled")
	}
	if missing.Allow("+2348012345678") {
		t.Fatal("a nil parser must never admit a request")
	}
	if !NewAIParser("configured-key").Enabled() {
		t.Fatal("a configured parser must report itself enabled")
	}
}

func TestAIParserModelConfiguration(t *testing.T) {
	defaultParser := NewAIParser("key")
	if defaultParser.model != "gemini-3.8-flash" {
		t.Fatalf("expected default model gemini-3.8-flash, got %s", defaultParser.model)
	}

	customParser := NewAIParserWithModel("key", "gemini-custom")
	if customParser.model != "gemini-custom" {
		t.Fatalf("expected custom model gemini-custom, got %s", customParser.model)
	}

	fallbackParser := NewAIParserWithModel("key", "  ")
	if fallbackParser.model != "gemini-3.8-flash" {
		t.Fatalf("expected whitespace model to fall back to gemini-3.8-flash, got %s", fallbackParser.model)
	}
}

// Every inbound message would otherwise reach the model provider, and a voice
// note also costs a media download. Anyone able to message the business number
// can send as many as they like, so the budget is what bounds the spend.
func TestSenderBudgetIsEnforcedPerSenderAndResets(t *testing.T) {
	parser := NewAIParser("configured-key")
	now := time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC)
	parser.now = func() time.Time { return now }

	for attempt := 1; attempt <= senderBudget; attempt++ {
		if !parser.Allow("+2348011111111") {
			t.Fatalf("request %d of %d was refused inside the budget", attempt, senderBudget)
		}
	}
	if parser.Allow("+2348011111111") {
		t.Fatal("a sender past its budget must be refused")
	}

	// The budget is per sender, so one noisy number must not silence another.
	if !parser.Allow("+2348022222222") {
		t.Fatal("a different sender must have its own budget")
	}

	now = now.Add(senderWindow + time.Second)
	if !parser.Allow("+2348011111111") {
		t.Fatal("the budget must reset once the window has passed")
	}
}

// A flood of distinct senders must not grow the tracking map without bound.
// Refusing the new sender is the safe direction: the caller still replies, just
// without the assistant.
func TestSenderTrackingIsBounded(t *testing.T) {
	parser := NewAIParser("configured-key")
	fixed := time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC)
	parser.now = func() time.Time { return fixed }
	for index := 0; index < maxTrackedSenders; index++ {
		parser.Allow(string(rune(index)) + "-sender")
	}
	if len(parser.senders) > maxTrackedSenders {
		t.Fatalf("tracked senders grew past the cap: %d", len(parser.senders))
	}
	if parser.Allow("one-more-sender") {
		t.Fatal("a new sender must be refused once the cap is reached")
	}
}

func TestAudioIsBounded(t *testing.T) {
	parser := NewAIParser("configured-key")
	if _, err := parser.ParseAudio(t.Context(), make([]byte, maxTranscribeBytes+1), "audio/ogg"); err == nil {
		t.Fatal("oversized audio must be refused before it reaches the provider")
	}
	if _, err := parser.ParseAudio(t.Context(), nil, "audio/ogg"); err == nil {
		t.Fatal("empty audio must be refused")
	}
}
