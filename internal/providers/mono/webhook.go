package mono

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"kredit/internal/mandates"
	"strings"
)

// Notice is an allowlisted receipt, not a bank payload. Account numbers, BVN,
// names, narration, and credentials are discarded before durable storage.
type Notice struct {
	BlockStatus mandates.Status `json:"block_status,omitempty"`
	EventID     string          `json:"event_id"`
	Type        string          `json:"type"`
	MandateID   string          `json:"mandate_id"`
	Reference   string          `json:"reference,omitempty"`
	PayloadHash string          `json:"payload_hash"`
}

func (c *Client) ParseWebhook(secret string, raw []byte) (Notice, error) {
	if !c.VerifySecret(secret) {
		return Notice{}, errors.New("invalid Mono webhook authentication")
	}
	var event struct {
		ID   string `json:"event_id"`
		Type string `json:"event"`
		Data struct {
			LiveMode  *bool  `json:"live_mode"`
			ID        string `json:"id"`
			Mandate   string `json:"mandate"`
			Reference string `json:"reference_number"`
		} `json:"data"`
	}
	if len(raw) > 1<<20 || json.Unmarshal(raw, &event) != nil {
		return Notice{}, errors.New("invalid Mono webhook")
	}
	if event.Data.LiveMode != nil && *event.Data.LiveMode {
		return Notice{}, errors.New("live provider event rejected by sandbox adapter")
	}
	hash := sha256.Sum256(raw)
	digest := hex.EncodeToString(hash[:])
	if event.ID == "" {
		event.ID = "sha256:" + digest
	}
	if len(event.ID) > 256 {
		return Notice{}, errors.New("invalid Mono event identity")
	}
	mandate := event.Data.Mandate
	if mandate == "" {
		mandate = event.Data.ID
	}
	switch event.Type {
	case "events.mandates.created", "events.mandates.rejected", "events.mandates.approved", "events.mandates.ready", "events.mandate.action.pause", "events.mandate.action.cancel", "events.mandate.action.reinstate", "events.mandates.expired",
		"events.mandates.debit.processing", "events.mandates.debit.successful", "events.mandates.debit.failed", "events.mandates.debit_attempt.successful":
	default:
		return Notice{}, errors.New("unsupported Mono event")
	}
	if mandate == "" || len(mandate) > 256 || len(event.Data.Reference) > 256 {
		return Notice{}, errors.New("invalid Mono event reference")
	}
	if strings.Contains(event.Type, ".debit.") && event.Data.Reference == "" {
		return Notice{}, errors.New("debit reference is required")
	}
	block := map[string]mandates.Status{"events.mandate.action.cancel": mandates.Cancelled, "events.mandate.action.pause": mandates.Paused, "events.mandates.expired": mandates.Expired, "events.mandates.rejected": mandates.Failed}[event.Type]
	return Notice{BlockStatus: block, EventID: event.ID, Type: event.Type, MandateID: mandate, Reference: event.Data.Reference, PayloadHash: digest}, nil
}
