package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
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
	payload := map[string]any{
		"channel": "email",
		"to":      []string{recipient.Address},
		"subject": emailSubject(message.Template),
		"text":    message.Body,
		"html":    buildEmailHTML(message),
	}
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

func buildEmailHTML(message Message) string {
	subject := emailSubject(message.Template)
	code := message.AuthenticationCode
	if code == "" && (strings.Contains(message.Template, "OTP") || strings.Contains(message.Template, "Authentication")) {
		re := regexp.MustCompile(`\b\d{6}\b`)
		code = re.FindString(message.Body)
	}

	var content strings.Builder
	if code != "" {
		content.WriteString(`<h1 style="margin: 0 0 16px 0; font-size: 20px; font-weight: 700; color: #0f172a; line-height: 1.3;">Sign-in Verification Code</h1>
<p style="margin: 0 0 24px 0; font-size: 15px; color: #334155; line-height: 1.6;">
  Please use the verification code below to complete your sign-in to your Kredit account.
</p>
<div style="background-color: #f8fafc; border: 1px dashed #cbd5e1; border-radius: 10px; padding: 22px 24px; text-align: center; margin: 0 0 24px 0;">
  <span style="font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace; font-size: 36px; font-weight: 700; letter-spacing: 10px; color: #0f172a; padding-left: 10px;">` + html.EscapeString(code) + `</span>
</div>
<p style="margin: 0; font-size: 13px; color: #64748b; line-height: 1.5;">
  This code will expire in <strong>10 minutes</strong>. If you did not request this verification code, someone may have entered your email address by mistake. Never share this code with anyone.
</p>`)
	} else if message.SecureLink != "" {
		content.WriteString(`<h1 style="margin: 0 0 16px 0; font-size: 20px; font-weight: 700; color: #0f172a; line-height: 1.3;">` + html.EscapeString(subject) + `</h1>
<p style="margin: 0 0 24px 0; font-size: 15px; color: #334155; line-height: 1.6;">` + html.EscapeString(message.Body) + `</p>
<div style="text-align: center; margin: 28px 0;">
  <a href="` + html.EscapeString(message.SecureLink) + `" style="background-color: #0f172a; color: #ffffff; padding: 14px 28px; border-radius: 8px; text-decoration: none; font-weight: 600; font-size: 14px; display: inline-block;">Continue to Kredit &rarr;</a>
</div>
<p style="margin: 0; font-size: 12px; color: #94a3b8; word-break: break-all; line-height: 1.5;">
  Or copy this link into your browser: <br><span style="color: #64748b;">` + html.EscapeString(message.SecureLink) + `</span>
</p>`)
	} else {
		content.WriteString(`<h1 style="margin: 0 0 16px 0; font-size: 20px; font-weight: 700; color: #0f172a; line-height: 1.3;">` + html.EscapeString(subject) + `</h1>
<p style="margin: 0; font-size: 15px; color: #334155; line-height: 1.6;">` + html.EscapeString(message.Body) + `</p>`)
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + html.EscapeString(subject) + `</title>
</head>
<body style="margin: 0; padding: 0; background-color: #f1f5f9; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; -webkit-font-smoothing: antialiased;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color: #f1f5f9; padding: 40px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width: 520px; background-color: #ffffff; border-radius: 12px; border: 1px solid #e2e8f0; overflow: hidden; box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.05);">
          <tr>
            <td style="padding: 28px 36px 20px 36px; border-bottom: 1px solid #f1f5f9;">
              <table role="presentation" width="100%" cellpadding="0" cellspacing="0">
                <tr>
                  <td>
                    <span style="font-size: 20px; font-weight: 800; letter-spacing: -0.5px; color: #0f172a; text-transform: uppercase;">KREDIT</span>
                  </td>
                  <td align="right">
                    <span style="font-size: 11px; font-weight: 600; color: #059669; background-color: #ecfdf5; border: 1px solid #a7f3d0; padding: 4px 10px; border-radius: 9999px; text-transform: uppercase; letter-spacing: 0.5px;">Security</span>
                  </td>
                </tr>
              </table>
            </td>
          </tr>
          <tr>
            <td style="padding: 32px 36px;">
              ` + content.String() + `
            </td>
          </tr>
          <tr>
            <td style="background-color: #f8fafc; padding: 22px 36px; border-top: 1px solid #f1f5f9;">
              <p style="margin: 0 0 6px 0; font-size: 12px; font-weight: 600; color: #475569; line-height: 1.5;">
                Kredit Technologies Limited <span style="font-weight: 400; color: #94a3b8;">(RC 9834452)</span>
              </p>
              <p style="margin: 0 0 8px 0; font-size: 11px; color: #94a3b8; line-height: 1.5;">
                House No. 348, Jamaina Road, Pompomari Bypass, Maiduguri, Borno State, Nigeria
              </p>
              <p style="margin: 0; font-size: 11px; color: #94a3b8;">
                Need assistance? Email <a href="mailto:hello@kredit.ng" style="color: #2563eb; text-decoration: none;">hello@kredit.ng</a>
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`
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
