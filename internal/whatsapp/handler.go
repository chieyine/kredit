package whatsapp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	CommandCreateCredit  = "create_credit"
	CommandRecordPayment = "record_payment"
	CommandConfirm       = "confirm"
	CommandQuery         = "query"
	CommandUnknown       = "unknown"
)

type Command struct {
	Kind                 string `json:"kind"`
	BuyerName            string `json:"buyer_name,omitempty"`
	AmountKobo           int64  `json:"amount_kobo,omitempty"`
	DueDate              string `json:"due_date,omitempty"`
	Query                string `json:"query,omitempty"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
}
type Event struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	Text      string `json:"text"`
	Signature string `json:"signature"`
}
type Handler struct {
	mu     sync.Mutex
	secret []byte
	seen   map[string]string
	parser func(string) (Command, error)
	pool   *pgxpool.Pool
}

func NewPostgresHandler(pool *pgxpool.Pool, secret string) *Handler {
	handler := NewHandler(secret)
	handler.pool = pool
	return handler
}

func NewHandler(secret string) *Handler {
	return &Handler{secret: []byte(secret), seen: map[string]string{}, parser: ParseCommand}
}
func (h *Handler) Sign(event Event) string {
	payload := event.ID + "|" + event.From + "|" + event.Text
	mac := hmac.New(sha256.New, h.secret)
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
func (h *Handler) Verify(event Event) bool {
	if len(h.secret) == 0 || event.ID == "" || event.From == "" || len(event.ID) > 512 || len(event.From) > 64 || len(event.Text) > 4096 || strings.ContainsAny(event.ID+event.From, "|\r\n") {
		return false
	}
	expected := h.Sign(event)
	return hmac.Equal([]byte(expected), []byte(event.Signature))
}
func (h *Handler) Handle(ctx context.Context, event Event) (Command, error) {
	if !h.Verify(event) {
		return Command{}, errors.New("invalid WhatsApp webhook signature")
	}
	payloadHash := sha256.Sum256([]byte(event.ID + "|" + event.From + "|" + event.Text))
	fingerprint := hex.EncodeToString(payloadHash[:])
	if h.pool != nil {
		senderHash := sha256.Sum256([]byte(event.From))
		command, parseErr := h.parser(event.Text)
		commandType := command.Kind
		if commandType == "" {
			commandType = CommandUnknown
		}
		var inserted string
		err := h.pool.QueryRow(ctx, `INSERT INTO app.messaging_events(provider,provider_event_id,sender,payload_hash,command_type,processed_at) VALUES('whatsapp',$1,$2,$3,$4,now()) ON CONFLICT(provider,provider_event_id) DO NOTHING RETURNING id::text`, event.ID, hex.EncodeToString(senderHash[:]), fingerprint, commandType).Scan(&inserted)
		if errors.Is(err, pgx.ErrNoRows) {
			var original string
			if err := h.pool.QueryRow(ctx, `SELECT payload_hash FROM app.messaging_events WHERE provider='whatsapp' AND provider_event_id=$1`, event.ID).Scan(&original); err != nil {
				return Command{}, err
			}
			if original != fingerprint {
				return Command{}, errors.New("WhatsApp event identifier was reused for different content")
			}
			return Command{}, nil
		}
		if err != nil {
			return Command{}, err
		}
		return command, parseErr
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if original, exists := h.seen[event.ID]; exists {
		if original != fingerprint {
			return Command{}, errors.New("WhatsApp event identifier was reused for different content")
		}
		return Command{}, nil
	}
	h.seen[event.ID] = fingerprint
	return h.parser(event.Text)
}

func ParseCommand(text string) (Command, error) {
	if len(text) > 4096 {
		return Command{}, errors.New("command is too long")
	}
	normalized := strings.TrimSpace(text)
	lower := strings.ToLower(normalized)
	if normalized == "" {
		return Command{Kind: CommandUnknown}, errors.New("empty command")
	}
	if lower == "create credit" || strings.HasPrefix(lower, "create credit ") || strings.HasPrefix(lower, "create credit\n") {
		body := strings.TrimSpace(normalized[len("create credit"):])
		parts := strings.Split(body, ",")
		if len(parts) < 3 {
			return Command{}, errors.New("create credit requires buyer, amount, and due date")
		}
		buyer := strings.TrimSpace(parts[0])
		if buyer == "" {
			return Command{}, errors.New("buyer name is required")
		}
		amount, err := parseAmount(strings.Join(parts[1:len(parts)-1], ","))
		if err != nil {
			return Command{}, err
		}
		due := strings.TrimSpace(parts[len(parts)-1])
		if _, err := time.Parse("2 January 2006", due); err != nil {
			return Command{}, errors.New("due date must be like 30 September 2026")
		}
		return Command{Kind: CommandCreateCredit, BuyerName: buyer, AmountKobo: amount, DueDate: due, RequiresConfirmation: true}, nil
	}
	if strings.Contains(lower, " paid ") {
		index := strings.Index(lower, " paid ")
		buyer := strings.TrimSpace(normalized[:index])
		amount, err := parseAmount(normalized[index+len(" paid "):])
		if err != nil {
			return Command{}, err
		}
		return Command{Kind: CommandRecordPayment, BuyerName: buyer, AmountKobo: amount, RequiresConfirmation: true}, nil
	}
	if strings.HasPrefix(lower, "how much") || strings.HasPrefix(lower, "who is overdue") || strings.HasPrefix(lower, "what is due") || strings.HasPrefix(lower, "check ") || strings.HasPrefix(lower, "show ") {
		return Command{Kind: CommandQuery, Query: normalized}, nil
	}
	if lower == "yes" || lower == "confirm" || lower == "ok" || strings.HasPrefix(lower, "confirm ") {
		return Command{Kind: CommandConfirm, Query: normalized}, nil
	}
	return Command{Kind: CommandUnknown}, errors.New("unsupported command")
}
func parseAmount(value string) (int64, error) {
	if len(value) > 128 {
		return 0, errors.New("amount is too long")
	}
	clean := strings.ToLower(strings.TrimSpace(value))
	clean = strings.ReplaceAll(clean, "₦", "")
	clean = strings.ReplaceAll(clean, "ngn", "")
	clean = strings.ReplaceAll(clean, ",", "")
	multiplier := int64(100)
	if strings.HasSuffix(clean, "m") {
		multiplier = 100000000
		clean = strings.TrimSuffix(clean, "m")
	} else if strings.HasSuffix(clean, "k") {
		multiplier = 100000
		clean = strings.TrimSuffix(clean, "k")
	}
	clean = strings.TrimSpace(clean)
	if clean == "" || strings.HasPrefix(clean, "-") || strings.HasPrefix(clean, "+") {
		return 0, errors.New("amount must be positive")
	}
	parts := strings.Split(clean, ".")
	if len(parts) > 2 || parts[0] == "" || (len(parts) == 2 && parts[1] == "") {
		return 0, errors.New("amount must be a valid decimal")
	}
	whole, fraction := parts[0], ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if _, err := strconv.ParseUint(whole, 10, 64); err != nil {
		return 0, errors.New("amount is too large")
	}
	numerator := new(big.Int)
	if _, ok := numerator.SetString(whole+fraction, 10); !ok {
		return 0, errors.New("amount must be numeric")
	}
	numerator.Mul(numerator, big.NewInt(multiplier))
	denominator := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(len(fraction))), nil)
	quotient, remainder := new(big.Int).QuoRem(numerator, denominator, new(big.Int))
	if remainder.Sign() != 0 || quotient.Sign() <= 0 || !quotient.IsInt64() {
		return 0, errors.New("amount must resolve to whole kobo")
	}
	return quotient.Int64(), nil
}
func ConfirmationSummary(command Command) string {
	switch command.Kind {
	case CommandCreateCredit:
		return fmt.Sprintf("Create credit for %s: %d kobo due %s. Confirm securely before sending.", command.BuyerName, command.AmountKobo, command.DueDate)
	case CommandRecordPayment:
		return fmt.Sprintf("Record payment from %s: %d kobo. Confirm securely before recording.", command.BuyerName, command.AmountKobo)
	default:
		return "Review this request securely in Kredit."
	}
}
