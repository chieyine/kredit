package collections

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

// WebhookProvider is the certified-provider connector boundary. Vendor SDKs
// stay behind the connector and the core receives one stable money contract.
type WebhookProvider struct {
	name, endpoint, token, secret string
	client                        *http.Client
}

func NewWebhookProvider(name, endpoint, token, secret string) (*WebhookProvider, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return nil, errors.New("valid collection connector endpoint is required")
	}
	if name == "" || strings.Contains(strings.ToLower(name), "mock") || token == "" || secret == "" {
		return nil, errors.New("certified collection provider, token, and webhook secret are required")
	}
	return &WebhookProvider{name: name, endpoint: strings.TrimRight(endpoint, "/"), token: token, secret: secret, client: &http.Client{Timeout: 20 * time.Second}}, nil
}
func (p *WebhookProvider) Name() string { return p.name }
func (p *WebhookProvider) Capabilities() Capabilities {
	return Capabilities{AuthorizationSession: true, OneTime: true, Recurring: true, Variable: true, Settlement: true, Reversal: true}
}
func (p *WebhookProvider) Submit(ctx context.Context, input Request) (Response, error) {
	var result Response
	err := p.request(ctx, http.MethodPost, "/collections", input, &result)
	if err == nil && (result.State == "" || result.ProviderCollectionID == "") {
		err = errors.New("collection connector returned an incomplete response")
	}
	return result, err
}
func (p *WebhookProvider) Get(ctx context.Context, id string) (Response, error) {
	if strings.TrimSpace(id) == "" {
		return Response{}, errors.New("provider collection id is required")
	}
	var result Response
	err := p.request(ctx, http.MethodGet, "/collections/"+url.PathEscape(id), nil, &result)
	return result, err
}
func (p *WebhookProvider) Cancel(ctx context.Context, id string) (Response, error) {
	if strings.TrimSpace(id) == "" {
		return Response{}, errors.New("provider collection id is required")
	}
	var result Response
	err := p.request(ctx, http.MethodPost, "/collections/"+url.PathEscape(id)+"/cancel", map[string]string{"reason": "operator_requested"}, &result)
	return result, err
}
func (p *WebhookProvider) Sign(event Webhook) string {
	event.Signature = ""
	payload, _ := json.Marshal(event)
	mac := hmac.New(sha256.New, []byte(p.secret))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
func (p *WebhookProvider) VerifyWebhook(event Webhook) bool {
	return event.Signature != "" && hmac.Equal([]byte(strings.ToLower(event.Signature)), []byte(p.Sign(event)))
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
		return fmt.Errorf("collection connector request failed: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("collection connector returned status %d", response.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(output)
}
