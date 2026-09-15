package whatsapp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	IntentCreateCredit = "create_credit"
	IntentConfirm      = "confirm"
	IntentRecordPayment = "record_payment"
	IntentQueryBalance = "query_balance"
	IntentHelp         = "help"
	IntentUnknown      = "unknown"

	geminiModel = "gemini-3.6-flash"
)

type AIResult struct {
	Intent     string `json:"intent"`
	BuyerName  string `json:"buyer_name,omitempty"`
	AmountKobo int64  `json:"amount_kobo,omitempty"`
	DueDate    string `json:"due_date,omitempty"`
	Items      string `json:"items,omitempty"`
	PaymentRef string `json:"payment_ref,omitempty"`
	Summary    string `json:"summary,omitempty"`
	RawText    string `json:"raw_text,omitempty"`
}

type AIParser struct {
	apiKey string
	client *http.Client
}

func NewAIParser(apiKey string) *AIParser {
	return &AIParser{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *AIParser) Enabled() bool {
	return p != nil && strings.TrimSpace(p.apiKey) != ""
}

const systemPrompt = `You are Kredit's intelligent WhatsApp assistant for commerce and trade credit in Nigeria.
Kredit empowers suppliers and merchants to record trade credit, invoice buyers, and collect payments.
Analyze the user's input (which may be text or a voice note in Nigerian English, informal market language, or Nigerian Pidgin).

Understand Nigerian trade terms:
- "give am credit", "credit sale", "buy on credit", "take goods", "balance later", "owe me" -> intent: "create_credit"
- "k" means thousand NGN (e.g., 50k = 50,000 NGN; 1.5m = 1,500,000 NGN; 350k = 350,000 NGN).
  Always calculate amount_kobo as: NGN amount * 100. For example: 350k NGN = 35000000 kobo.
- "yes", "confirm", "proceed", "go ahead", "send am", "correct", "true" -> intent: "confirm"
- "paid", "he pay", "received money", "give cash", "transfer" -> intent: "record_payment"
- "who owes me", "balance", "my debtors", "how much am I owed", "pending money" -> intent: "query_balance"
- "help", "how to use", "what can you do" -> intent: "help"

Dates: Calculate relative dates based on current date. Format due_date as YYYY-MM-DD.

Return ONLY a valid JSON object matching this schema:
{
  "intent": "create_credit" | "confirm" | "record_payment" | "query_balance" | "help" | "unknown",
  "buyer_name": string (name of buyer/merchant, empty if not applicable),
  "amount_kobo": integer (amount in kobo, 0 if not applicable),
  "due_date": string (YYYY-MM-DD or empty if not mentioned),
  "items": string (goods/services described or empty),
  "summary": string (a concise 1-sentence recap of what was understood),
  "raw_text": string (the transcribed text if audio, or original message)
}`

func (p *AIParser) ParseText(ctx context.Context, text string) (AIResult, error) {
	if !p.Enabled() {
		return AIResult{Intent: IntentUnknown, RawText: text}, errors.New("Gemini AI is not configured")
	}

	reqBody := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{"text": systemPrompt + "\n\nUser message: " + text},
				},
			},
		},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
		},
	}

	return p.callGemini(ctx, reqBody)
}

func (p *AIParser) ParseAudio(ctx context.Context, audioBytes []byte, mimeType string) (AIResult, error) {
	if !p.Enabled() {
		return AIResult{Intent: IntentUnknown}, errors.New("Gemini AI is not configured")
	}
	if len(audioBytes) == 0 {
		return AIResult{Intent: IntentUnknown}, errors.New("audio data is empty")
	}
	if mimeType == "" {
		mimeType = "audio/ogg"
	}
	// Normalize mime type for Gemini
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}

	encoded := base64.StdEncoding.EncodeToString(audioBytes)
	reqBody := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{
						"text": systemPrompt + "\n\nListen to the voice note attached and extract the details:",
					},
					{
						"inlineData": map[string]string{
							"mimeType": mimeType,
							"data":     encoded,
						},
					},
				},
			},
		},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
		},
	}

	return p.callGemini(ctx, reqBody)
}

func (p *AIParser) callGemini(ctx context.Context, payload map[string]any) (AIResult, error) {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return AIResult{Intent: IntentUnknown}, fmt.Errorf("marshal gemini request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/%s:generateContent?key=%s", geminiModel, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBytes))
	if err != nil {
		return AIResult{Intent: IntentUnknown}, fmt.Errorf("create gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return AIResult{Intent: IntentUnknown}, fmt.Errorf("gemini api call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AIResult{Intent: IntentUnknown}, fmt.Errorf("read gemini response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return AIResult{Intent: IntentUnknown}, fmt.Errorf("gemini error (status %d): %s", resp.StatusCode, string(body))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return AIResult{Intent: IntentUnknown}, fmt.Errorf("unmarshal gemini envelope: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return AIResult{Intent: IntentUnknown}, errors.New("gemini returned no candidates")
	}

	rawJSON := geminiResp.Candidates[0].Content.Parts[0].Text
	// Strip any possible markdown codeblock wraps
	rawJSON = strings.TrimSpace(rawJSON)
	if strings.HasPrefix(rawJSON, "```json") {
		rawJSON = strings.TrimPrefix(rawJSON, "```json")
	} else if strings.HasPrefix(rawJSON, "```") {
		rawJSON = strings.TrimPrefix(rawJSON, "```")
	}
	rawJSON = strings.TrimSuffix(rawJSON, "```")
	rawJSON = strings.TrimSpace(rawJSON)

	var result AIResult
	if err := json.Unmarshal([]byte(rawJSON), &result); err != nil {
		return AIResult{Intent: IntentUnknown, RawText: rawJSON}, fmt.Errorf("unmarshal structured result: %w", err)
	}

	return result, nil
}
