package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"
)

const SendlyEndpoint = "https://api.sendlyai.com/v1/messages"

var ErrRecipientSuppressed = errors.New("email recipient is suppressed by the delivery provider")

type permanentDeliveryError struct{ status int }

func (e permanentDeliveryError) Error() string {
	return fmt.Sprintf("message provider rejected the request (status %d)", e.status)
}

// SendlyProvider uses the published /v1 API. One request has one recipient so
// delivery evidence can never be borrowed from another member of a bulk send.
type SendlyProvider struct {
	endpoint, token, from string
	client                *http.Client
}

func NewSendlyProvider(endpoint, token, from string) (*SendlyProvider, error) {
	if endpoint != SendlyEndpoint || strings.TrimSpace(token) == "" || strings.ContainsAny(token, "\r\n") {
		return nil, errors.New("sendly endpoint and API key are required")
	}
	if from != "" {
		if _, err := mail.ParseAddress(from); err != nil {
			return nil, errors.New("sendly sender must be an email address")
		}
	}
	return &SendlyProvider{endpoint: endpoint, token: token, from: from, client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (p *SendlyProvider) Channel() string { return ChannelEmail }
func (p *SendlyProvider) Send(ctx context.Context, message Message) (string, error) {
	if message.Channel != ChannelEmail || message.EventID == "" || strings.TrimSpace(message.Body) == "" {
		return "", errors.New("email identity and content are required")
	}
	recipient, err := mail.ParseAddress(message.Destination)
	if err != nil {
		return "", errors.New("email recipient is invalid")
	}
	payload := map[string]any{"channel": "email", "to": []string{recipient.Address}, "subject": emailSubject(message.Template), "text": message.Body}
	if p.from != "" {
		payload["from"] = p.from
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Idempotency-Key", message.EventID+":"+ChannelEmail)
	var result struct {
		ID       string `json:"id"`
		Accepted int    `json:"accepted"`
		Skipped  int    `json:"skipped"`
		Sandbox  bool   `json:"sandbox"`
	}
	if err = p.request(req, &result); err != nil {
		return "", err
	}
	if result.Accepted == 0 && result.Skipped == 1 {
		return "", ErrRecipientSuppressed
	}
	if result.Accepted != 1 || result.Skipped != 0 || result.ID == "" || result.Sandbox {
		return "", errors.New("sendly did not accept one live email")
	}
	return result.ID, nil
}
func emailSubject(template string) string {
	switch template {
	case "OTP", "AuthenticationOTP", "AuthenticationCode":
		return "Your Kredit sign-in code"
	case "BuyerInvitation":
		return "Your Kredit business invitation"
	case "AccountRecoveryContinuation":
		return "Continue your Kredit account recovery"
	case "CollectionNotice", "CollectionScheduled":
		return "Your upcoming Kredit payment"
	case "PaymentReceived", "CollectionSucceeded":
		return "Your Kredit payment update"
	default:
		return "An update from Kredit"
	}
}

type DeliveryStatus struct {
	ID          string     `json:"id"`
	Channel     string     `json:"channel"`
	To          []string   `json:"to"`
	Status      string     `json:"status"`
	DeliveredAt *time.Time `json:"deliveredAt"`
}
type DeliveryStatusProvider interface {
	DeliveryStatus(context.Context, string) (DeliveryStatus, error)
}

func (p *SendlyProvider) DeliveryStatus(ctx context.Context, id string) (DeliveryStatus, error) {
	var result DeliveryStatus
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint+"/"+url.PathEscape(id), nil)
	if err != nil {
		return result, err
	}
	err = p.request(req, &result)
	return result, err
}
func (p *SendlyProvider) request(req *http.Request, result any) error {
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")
	response, err := p.client.Do(req)
	if err != nil {
		return errors.New("sendly request failed")
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	if err != nil || len(body) > 64<<10 {
		return errors.New("sendly response could not be read")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if response.StatusCode >= 400 && response.StatusCode < 500 && response.StatusCode != 408 && response.StatusCode != 429 {
			return permanentDeliveryError{response.StatusCode}
		}
		return fmt.Errorf("sendly temporarily unavailable (status %d)", response.StatusCode)
	}
	if err = json.Unmarshal(body, result); err != nil {
		return errors.New("sendly returned an invalid response")
	}
	return nil
}

func NewDeliveryProvider(channel, endpoint, token, from string) (Provider, error) {
	if channel == ChannelEmail {
		return NewSendlyProvider(endpoint, token, from)
	}
	if channel == ChannelSMS {
		return NewMesajProvider(endpoint, token, from)
	}
	return NewWebhookProvider(channel, endpoint, token)
}

func deliveryFailureState(err error) string {
	if errors.Is(err, ErrRecipientSuppressed) {
		return StateSuppressed
	}
	return StateFailed
}
func permanentDeliveryFailure(err error) bool {
	var permanent permanentDeliveryError
	return errors.Is(err, ErrRecipientSuppressed) || errors.As(err, &permanent)
}
