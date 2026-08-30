package mandates

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

// WebhookProvider delegates mandate creation and lookup to the approved
// collection connector using the provider-neutral mandate contract.
type WebhookProvider struct {
	name, endpoint, token string
	client                *http.Client
}

func NewWebhookProvider(name, endpoint, token string) (*WebhookProvider, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return nil, errors.New("valid mandate connector endpoint is required")
	}
	if name == "" || strings.Contains(strings.ToLower(name), "mock") || token == "" {
		return nil, errors.New("certified mandate provider and token are required")
	}
	return &WebhookProvider{name: name, endpoint: strings.TrimRight(endpoint, "/"), token: token, client: &http.Client{Timeout: 20 * time.Second}}, nil
}
func (p *WebhookProvider) Name() string { return p.name }
func (p *WebhookProvider) CreateAuthorizationSession(ctx context.Context, input AuthorizationInput) (Mandate, error) {
	var result Mandate
	err := p.request(ctx, http.MethodPost, "/mandates", input, &result)
	if err == nil && (result.ProviderID == "" || result.Status == "") {
		err = errors.New("mandate connector returned an incomplete response")
	}
	result.Provider = p.name
	result.Status = Status(strings.ToUpper(string(result.Status)))
	return result, err
}
func (p *WebhookProvider) GetMandate(ctx context.Context, id string) (Mandate, error) {
	if id == "" {
		return Mandate{}, errors.New("provider mandate id is required")
	}
	var result Mandate
	err := p.request(ctx, http.MethodGet, "/mandates/"+url.PathEscape(id), nil, &result)
	result.Provider = p.name
	result.Status = Status(strings.ToUpper(string(result.Status)))
	return result, err
}
func (p *WebhookProvider) CancelMandate(ctx context.Context, id, reason string) (Mandate, error) {
	if id == "" || strings.TrimSpace(reason) == "" {
		return Mandate{}, errors.New("mandate id and cancellation reason are required")
	}
	var result Mandate
	err := p.request(ctx, http.MethodPost, "/mandates/"+url.PathEscape(id)+"/cancel", map[string]string{"reason": reason}, &result)
	result.Provider = p.name
	result.Status = Status(strings.ToUpper(string(result.Status)))
	return result, err
}
func (p *WebhookProvider) RestoreAuthorization(ctx context.Context, id string) (Mandate, error) {
	if id == "" {
		return Mandate{}, errors.New("mandate id is required")
	}
	var result Mandate
	err := p.request(ctx, http.MethodPost, "/mandates/"+url.PathEscape(id)+"/restore", map[string]string{}, &result)
	result.Provider = p.name
	result.Status = Status(strings.ToUpper(string(result.Status)))
	return result, err
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
		return fmt.Errorf("mandate connector request failed: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("mandate connector returned status %d", response.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(output)
}
