package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WebhookProvider is a provider-neutral delivery adapter. A production
// deployment points it at an approved internal connector for email, SMS, or
// WhatsApp; provider-specific credentials and payloads remain outside the
// financial application.
type WebhookProvider struct {
	channel  string
	endpoint string
	token    string
	client   *http.Client
}

func NewWebhookProvider(channel, endpoint, token string) (*WebhookProvider, error) {
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel != ChannelEmail && channel != ChannelSMS && channel != ChannelWhatsApp {
		return nil, errors.New("unsupported notification channel")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.ForceQuery || strings.ContainsAny(token, "\r\n") || strings.TrimSpace(token) == "" {
		return nil, errors.New("notification endpoint and token are required")
	}
	return &WebhookProvider{channel: channel, endpoint: strings.TrimSpace(endpoint), token: strings.TrimSpace(token), client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (p *WebhookProvider) Channel() string { return p.channel }

func (p *WebhookProvider) Send(ctx context.Context, message Message) (string, error) {
	if p == nil || p.client == nil {
		return "", errors.New("notification provider is unavailable")
	}
	if strings.TrimSpace(message.Destination) == "" {
		return "", errors.New("notification destination is required")
	}
	payload, err := json.Marshal(map[string]any{
		"channel": message.Channel, "destination": message.Destination,
		"event_id": message.EventID, "template": message.Template,
		"template_version": message.TemplateVersion, "body": message.Body,
		"secure_link": message.SecureLink,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.token)
	// Connectors must treat this value as an idempotency key. It makes a
	// provider retry safe when the worker loses its database lease after the
	// remote service accepted a message but before the local success update.
	req.Header.Set("Idempotency-Key", message.EventID+":"+message.Channel)
	response, err := p.client.Do(req)
	if err != nil {
		return "", errors.New("notification connector request failed")
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("notification connector returned status %d", response.StatusCode)
	}
	var result struct {
		MessageID string `json:"message_id"`
	}
	if err := json.Unmarshal(body, &result); err != nil || strings.TrimSpace(result.MessageID) == "" {
		return "", errors.New("notification connector returned an invalid response")
	}
	return result.MessageID, nil
}

// Connector contract v1: GET the configured endpoint with message_id, using
// the same bearer authentication. This is a Kredit contract, not a vendor API.
func (p *WebhookProvider) DeliveryStatus(ctx context.Context, id string) (DeliveryStatus, error) {
	var result DeliveryStatus
	endpoint := p.endpoint + "?" + url.Values{"message_id": {id}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return result, err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	response, err := p.client.Do(req)
	if err != nil {
		return result, errors.New("connector delivery lookup unavailable")
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	if err != nil || len(body) > 64<<10 {
		return result, errors.New("connector delivery response is too large or incomplete")
	}
	if response.StatusCode != http.StatusOK || json.Unmarshal(body, &result) != nil {
		return result, errors.New("connector delivery lookup could not be confirmed")
	}
	if result.ID != id || result.Channel != p.channel || len(result.To) != 1 {
		return result, ErrInvalidDeliveryReceipt
	}
	return result, nil
}
