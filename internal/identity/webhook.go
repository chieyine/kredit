package identity

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WebhookProvider delegates vendor-specific identity work to an approved
// connector and accepts only Kredit's restricted, safe-result contract.
type WebhookProvider struct {
	name          string
	endpoint      string
	token         string
	webhookSecret string
	client        *http.Client
}

func NewWebhookProvider(name, endpoint, token, webhookSecret string) (*WebhookProvider, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return nil, errors.New("valid identity connector endpoint is required")
	}
	if strings.TrimSpace(name) == "" || strings.Contains(strings.ToLower(name), "mock") || strings.TrimSpace(token) == "" || strings.TrimSpace(webhookSecret) == "" {
		return nil, errors.New("certified identity provider name, token, and webhook secret are required")
	}
	return &WebhookProvider{name: name, endpoint: strings.TrimRight(endpoint, "/"), token: token, webhookSecret: webhookSecret, client: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (p *WebhookProvider) Name() string { return p.name }
func (p *WebhookProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{PersonVerification: true, BusinessVerification: true, AuthorityVerification: true, WebhookVerification: true}
}
func (p *WebhookProvider) CreatePersonVerification(ctx context.Context, input PersonVerificationInput) (VerificationSession, error) {
	return p.create(ctx, "person", input)
}
func (p *WebhookProvider) CreateBusinessVerification(ctx context.Context, input BusinessVerificationInput) (VerificationSession, error) {
	return p.create(ctx, "business", input)
}
func (p *WebhookProvider) CreateAuthorityVerification(ctx context.Context, input AuthorityVerificationInput) (VerificationSession, error) {
	return p.create(ctx, "authority", input)
}
func (p *WebhookProvider) create(ctx context.Context, kind string, input any) (VerificationSession, error) {
	var result VerificationSession
	if err := p.request(ctx, http.MethodPost, "/verifications/"+kind, input, &result); err != nil {
		return VerificationSession{}, err
	}
	if result.ProviderID == "" || result.State == "" {
		return VerificationSession{}, errors.New("identity connector returned an incomplete session")
	}
	result.Provider = p.name
	return result, nil
}
func (p *WebhookProvider) GetVerification(ctx context.Context, id string) (ProviderVerification, error) {
	if strings.TrimSpace(id) == "" {
		return ProviderVerification{}, errors.New("provider verification id is required")
	}
	var result ProviderVerification
	err := p.request(ctx, http.MethodGet, "/verifications/"+url.PathEscape(id), nil, &result)
	return result, err
}
func (p *WebhookProvider) VerifyWebhook(_ context.Context, headers http.Header, body []byte) (VerifiedIdentityEvent, error) {
	signature := strings.TrimSpace(headers.Get("X-Kredit-Signature"))
	mac := hmac.New(sha256.New, []byte(p.webhookSecret))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if signature == "" || !hmac.Equal([]byte(strings.ToLower(signature)), []byte(expected)) {
		return VerifiedIdentityEvent{}, errors.New("identity webhook signature is invalid")
	}
	var event VerifiedIdentityEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return VerifiedIdentityEvent{}, err
	}
	if event.ProviderID == "" || event.SubjectID == "" || event.State == "" {
		return VerifiedIdentityEvent{}, errors.New("identity webhook event is incomplete")
	}
	return event, nil
}
func (p *WebhookProvider) request(ctx context.Context, method, path string, input, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, p.endpoint+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+p.token)
	request.Header.Set("Accept", "application/json")
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("identity connector request failed: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("identity connector returned status %d", response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(output); err != nil {
		return fmt.Errorf("decode identity connector response: %w", err)
	}
	return nil
}
