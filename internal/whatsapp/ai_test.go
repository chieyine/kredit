package whatsapp

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestGeminiAIParsing(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY is not set, skipping live test")
	}
	parser := NewAIParser(apiKey)
	if !parser.Enabled() {
		t.Fatal("expected parser to be enabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := parser.ParseText(ctx, "I just gave Alhassan 50 cartons of indomie for 350k to pay on Friday")
	if err != nil {
		t.Fatalf("ParseText failed: %v", err)
	}

	if result.Intent != IntentCreateCredit {
		t.Errorf("expected intent %s, got %s", IntentCreateCredit, result.Intent)
	}
	if result.BuyerName != "Alhassan" {
		t.Errorf("expected buyer Alhassan, got %s", result.BuyerName)
	}
	if result.AmountKobo != 35000000 {
		t.Errorf("expected amount_kobo 35000000, got %d", result.AmountKobo)
	}
}
