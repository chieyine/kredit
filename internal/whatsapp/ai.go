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
	"sync"
	"time"
)

const (
	IntentCreateCredit  = "create_credit"
	IntentConfirm       = "confirm"
	IntentRecordPayment = "record_payment"
	IntentQueryBalance  = "query_balance"
	IntentHelp          = "help"
	IntentUnknown       = "unknown"

	defaultGeminiModel = "gemini-3.8-flash"
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

// senderBudget is the per-sender request allowance. Every inbound WhatsApp
// message would otherwise reach the model provider, and a voice note also costs
// a media download, so anyone able to message the business number could drive
// unbounded spend. The window is deliberately small: this assistant reads
// messages back to a seller, it does not hold a conversation.
const (
	senderBudget       = 12
	senderWindow       = 10 * time.Minute
	maxTrackedSenders  = 10000
	maxTranscribeBytes = 4 << 20
)

type senderWindowState struct {
	started time.Time
	count   int
}

type AIParser struct {
	apiKey   string
	model    string
	client   *http.Client
	mu       sync.Mutex
	senders  map[string]senderWindowState
	prunedAt time.Time
	now      func() time.Time
}

func NewAIParser(apiKey string) *AIParser {
	return NewAIParserWithModel(apiKey, defaultGeminiModel)
}

func NewAIParserWithModel(apiKey, model string) *AIParser {
	if strings.TrimSpace(model) == "" {
		model = defaultGeminiModel
	}
	return &AIParser{
		apiKey: apiKey,
		model:  model,
		// A redirect must not forward the provider API key or customer content.
		client:  &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		senders: map[string]senderWindowState{},
		now:     func() time.Time { return time.Now() },
	}
}

func (p *AIParser) Enabled() bool {
	return p != nil && strings.TrimSpace(p.apiKey) != ""
}

// Allow records one request for a sender and reports whether it may proceed.
func (p *AIParser) Allow(sender string) bool {
	if p == nil {
		return false
	}
	now := p.now()
	p.mu.Lock()
	defer p.mu.Unlock()
	if now.Sub(p.prunedAt) >= senderWindow {
		p.prunedAt = now
		for key, state := range p.senders {
			if now.Sub(state.started) >= senderWindow {
				delete(p.senders, key)
			}
		}
	}
	// A flood of distinct senders must not grow this map without bound. Refusing
	// new senders is the safe direction: the fallback reply still works.
	if _, tracked := p.senders[sender]; !tracked && len(p.senders) >= maxTrackedSenders {
		return false
	}
	state := p.senders[sender]
	if state.started.IsZero() || now.Sub(state.started) >= senderWindow {
		state = senderWindowState{started: now}
	}
	state.count++
	p.senders[sender] = state
	return state.count <= senderBudget
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
		return AIResult{Intent: IntentUnknown, RawText: text}, errors.New("gemini AI is not configured")
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
		return AIResult{Intent: IntentUnknown}, errors.New("gemini AI is not configured")
	}
	if len(audioBytes) == 0 {
		return AIResult{Intent: IntentUnknown}, errors.New("audio data is empty")
	}
	if len(audioBytes) > maxTranscribeBytes {
		return AIResult{Intent: IntentUnknown}, errors.New("voice note is too long to read")
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

	// The key travels in a header, never the query string: a URL reaches proxy
	// access logs, error reports and browser-style referrer chains.
	model := p.model
	if strings.TrimSpace(model) == "" {
		model = defaultGeminiModel
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/%s:generateContent", model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBytes))
	if err != nil {
		return AIResult{Intent: IntentUnknown}, fmt.Errorf("create gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return AIResult{Intent: IntentUnknown}, fmt.Errorf("gemini api call failed: %w", err)
	}
	// The complete body read determines the result; closing only releases the response.
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil {
		return AIResult{Intent: IntentUnknown}, fmt.Errorf("read gemini response: %w", err)
	}

	if len(body) > 1<<20 {
		return AIResult{Intent: IntentUnknown}, errors.New("gemini response exceeds the size limit")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// The provider echoes submitted content in some error bodies. Logging the
		// status alone keeps customer names and amounts out of our logs.
		return AIResult{Intent: IntentUnknown}, fmt.Errorf("gemini error (status %d)", resp.StatusCode)
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
