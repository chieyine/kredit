package notifications

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"kredit/internal/platformsettings"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var metaEndpoint = regexp.MustCompile(`^https://graph\.facebook\.com/v[0-9]+\.[0-9]+/([0-9]+)/messages$`)
var metaTemplateName = regexp.MustCompile(`^[a-z0-9_]{1,512}$`)

type MetaProvider struct {
	config  platformsettings.NotificationConnector
	phoneID string
	client  *http.Client
}

func NewMetaProvider(config platformsettings.NotificationConnector) (*MetaProvider, error) {
	match := metaEndpoint.FindStringSubmatch(config.Endpoint)
	if len(match) != 2 || strings.TrimSpace(config.Token) == "" || strings.ContainsAny(config.Token, "\r\n") || len(config.WebhookSecret) < 32 || len(config.VerifyToken) < 32 || !regexp.MustCompile(`^[a-z]{2,3}(_[A-Z]{2})?$`).MatchString(config.Language) {
		return nil, errors.New("meta requires its versioned phone-number endpoint, access token, app secret, verification token and template language")
	}
	for _, name := range []string{config.AuthenticationTemplate, config.UtilityTemplate, config.MarketingTemplate} {
		if !metaTemplateName.MatchString(name) {
			return nil, errors.New("enter approved Meta authentication, utility and marketing template names")
		}
	}
	return &MetaProvider{config, match[1], &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (p *MetaProvider) Channel() string { return ChannelWhatsApp }
func (p *MetaProvider) SubmissionIdentity() string {
	return p.config.Endpoint + ":" + p.config.Language + ":" + p.config.AuthenticationTemplate + ":" + p.config.UtilityTemplate + ":" + p.config.MarketingTemplate
}
func (p *MetaProvider) PhoneID() string { return p.phoneID }
func VerifyMetaSignature(secret, signature string, body []byte) bool {
	if len(secret) < 32 || !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	given, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(given, mac.Sum(nil))
}
func (p *MetaProvider) Send(ctx context.Context, m Message) (string, error) {
	if m.Channel != ChannelWhatsApp || m.EventID == "" || !validMessagingPhone(m.Destination) || strings.TrimSpace(m.Body) == "" {
		return "", permanentDeliveryError{400}
	}
	name, value := p.config.UtilityTemplate, m.Body
	if m.Template == "ProductUpdate" {
		name = p.config.MarketingTemplate
	}
	components := []map[string]any{}
	if m.Template == "AuthenticationCode" {
		if len(m.AuthenticationCode) < 4 || len(m.AuthenticationCode) > 16 {
			return "", permanentDeliveryError{400}
		}
		name, value = p.config.AuthenticationTemplate, m.AuthenticationCode
	}
	components = append(components, map[string]any{"type": "body", "parameters": []map[string]string{{"type": "text", "text": value}}})
	if m.Template == "AuthenticationCode" {
		components = append(components, map[string]any{"type": "button", "sub_type": "url", "index": "0", "parameters": []map[string]string{{"type": "text", "text": value}}})
	}
	payload, _ := json.Marshal(map[string]any{"messaging_product": "whatsapp", "recipient_type": "individual", "to": strings.TrimPrefix(m.Destination, "+"), "type": "template", "template": map[string]any{"name": name, "language": map[string]string{"code": p.config.Language}, "components": components}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.config.Token)
	req.Header.Set("Content-Type", "application/json")
	response, err := p.client.Do(req)
	if err != nil {
		return "", ErrSubmissionUnknown
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	if err != nil || len(body) > 64<<10 {
		return "", ErrSubmissionUnknown
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if response.StatusCode >= 400 && response.StatusCode < 500 && response.StatusCode != 408 && response.StatusCode != 429 {
			return "", permanentDeliveryError{response.StatusCode}
		}
		return "", ErrSubmissionUnknown
	}
	var result struct {
		Product  string `json:"messaging_product"`
		Contacts []struct {
			ID string `json:"wa_id"`
		} `json:"contacts"`
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if json.Unmarshal(body, &result) != nil || result.Product != "whatsapp" || len(result.Messages) != 1 || !strings.HasPrefix(result.Messages[0].ID, "wamid.") || len(result.Messages[0].ID) > 512 || len(result.Contacts) != 1 || !sameDeliveryDestination(ChannelWhatsApp, result.Contacts[0].ID, m.Destination) {
		return "", ErrSubmissionUnknown
	}
	return result.Messages[0].ID, nil
}
