// Package paystack implements Paystack's hosted direct-debit authorization and
// charge APIs. Authorization codes never enter public mandate metadata.
package paystack

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"kredit/internal/collections"
	"kredit/internal/ledger"
	"kredit/internal/mandates"
)

type EmailLookup func(context.Context, string) (string, error)
type Client struct {
	name, secret, callback, endpoint string
	live                             bool
	email                            EmailLookup
	http                             *http.Client
}

func New(name, secret, callback string, live bool, email EmailLookup) (*Client, error) {
	u, e := url.Parse(callback)
	prefix := "sk_test_"
	if live {
		prefix = "sk_live_"
	}
	if name == "" || !strings.HasPrefix(secret, prefix) || strings.ContainsAny(secret, "\r\n") || e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || email == nil {
		return nil, errors.New("Paystack requires an account name, matching secret key, HTTPS return URL and buyer email lookup")
	}
	return &Client{name: name, secret: secret, callback: callback, live: live, email: email, endpoint: "https://api.paystack.co", http: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) Name() string { return c.name }
func (c *Client) Capabilities() collections.Capabilities {
	return collections.Capabilities{AuthorizationSession: true, OneTime: true, Recurring: true, Variable: true, SupportedCurrencies: []string{"NGN"}}
}
func (c *Client) call(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, e := json.Marshal(in)
		if e != nil {
			return e
		}
		body = bytes.NewReader(b)
	}
	req, e := http.NewRequestWithContext(ctx, method, c.endpoint+path, body)
	if e != nil {
		return e
	}
	req.Header.Set("Authorization", "Bearer "+c.secret)
	req.Header.Set("Content-Type", "application/json")
	res, e := c.http.Do(req)
	if e != nil {
		return errors.New("Paystack request outcome could not be confirmed")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("Paystack returned HTTP %d; reconcile before retrying", res.StatusCode)
	}
	var envelope struct {
		Status bool            `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	if e = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&envelope); e != nil {
		return errors.New("Paystack response was invalid")
	}
	if !envelope.Status {
		return errors.New("Paystack did not confirm the request")
	}
	if out != nil {
		return json.Unmarshal(envelope.Data, out)
	}
	return nil
}
func (c *Client) CreateAuthorizationSession(ctx context.Context, in mandates.AuthorizationInput) (mandates.Mandate, error) {
	if in.AmountCeiling <= 0 || in.UserID == "" {
		return mandates.Mandate{}, errors.New("buyer and positive authorization ceiling required")
	}
	email, e := c.email(ctx, in.UserID)
	if e != nil || email == "" {
		return mandates.Mandate{}, errors.New("a verified email is required for Paystack bank authorization")
	}
	var data struct {
		Reference   string `json:"reference"`
		RedirectURL string `json:"redirect_url"`
	}
	e = c.call(ctx, "POST", "/customer/authorization/initialize", map[string]string{"email": email, "channel": "direct_debit", "callback_url": c.callback}, &data)
	if e != nil {
		return mandates.Mandate{}, e
	}
	u, e := url.Parse(data.RedirectURL)
	if e != nil || u.Scheme != "https" || u.User != nil || (u.Hostname() != "paystack.com" && !strings.HasSuffix(u.Hostname(), ".paystack.com")) || data.Reference == "" {
		return mandates.Mandate{}, errors.New("Paystack returned invalid authorization details")
	}
	return mandates.Mandate{ProviderAdapter: "paystack", Provider: c.name, ProviderID: data.Reference, Reference: in.Reference, Status: mandates.Pending, AmountCeiling: in.AmountCeiling, Variable: true, AuthorizationURL: data.RedirectURL}, nil
}

type authorization struct {
	Code     string `json:"authorization_code"`
	Channel  string `json:"channel"`
	Active   bool   `json:"active"`
	Customer struct {
		Email string `json:"email"`
	} `json:"customer"`
}

func (c *Client) authorization(ctx context.Context, id string) (authorization, error) {
	var a authorization
	if id == "" {
		return a, errors.New("authorization reference required")
	}
	e := c.call(ctx, "GET", "/customer/authorization/verify/"+url.PathEscape(id), nil, &a)
	if e == nil && (a.Channel != "direct_debit" || a.Code == "" || a.Customer.Email == "") {
		e = errors.New("Paystack did not return a bank authorization")
	}
	return a, e
}
func (c *Client) GetMandate(ctx context.Context, id string) (mandates.Mandate, error) {
	a, e := c.authorization(ctx, id)
	if e != nil {
		return mandates.Mandate{}, e
	}
	state := mandates.Paused
	if a.Active {
		state = mandates.Active
	}
	return mandates.Mandate{Provider: c.name, ProviderAdapter: "paystack", ProviderID: id, Status: state, Variable: true}, nil
}
func (c *Client) CancelMandate(ctx context.Context, id, reason string) (mandates.Mandate, error) {
	a, e := c.authorization(ctx, id)
	if e != nil {
		return mandates.Mandate{}, e
	}
	if e = c.call(ctx, "POST", "/customer/authorization/deactivate", map[string]string{"authorization_code": a.Code}, nil); e != nil {
		return mandates.Mandate{}, e
	}
	return mandates.Mandate{Provider: c.name, ProviderID: id, Status: mandates.Cancelled}, nil
}
func (c *Client) RestoreAuthorization(context.Context, string) (mandates.Mandate, error) {
	return mandates.Mandate{}, errors.New("start a new bank authorization to restore cancelled Paystack permission")
}
func (c *Client) Submit(ctx context.Context, in collections.Request) (collections.Response, error) {
	if in.AmountKobo <= 0 || in.Currency != "NGN" || in.ExternalReference == "" {
		return collections.Response{}, errors.New("positive NGN amount and a stable reference required")
	}
	if in.SettlementRoute != nil && in.SettlementRoute.Method == "provider_split" {
		return collections.Response{}, errors.New("Paystack automatic seller split is not configured")
	}
	a, e := c.authorization(ctx, in.MandateReference)
	if e != nil {
		return collections.Response{}, e
	}
	if !a.Active {
		return collections.Response{}, errors.New("bank authorization is not active")
	}
	// Only a server-to-server verification can recognize money. Even a successful
	// charge response remains pending until the original request is reconciled.
	e = c.call(ctx, "POST", "/transaction/charge_authorization", map[string]any{"authorization_code": a.Code, "email": a.Customer.Email, "amount": int64(in.AmountKobo), "currency": "NGN", "reference": in.ExternalReference}, nil)
	if e != nil {
		return collections.Response{}, e
	}
	return collections.Response{State: collections.ProviderPending, ProviderCollectionID: in.ExternalReference}, nil
}
func (c *Client) Get(context.Context, string) (collections.Response, error) {
	return collections.Response{}, errors.New("original collection request is required for Paystack verification")
}
func (c *Client) GetByReference(ctx context.Context, in collections.Request) (collections.Response, error) {
	var t struct {
		Reference     string `json:"reference"`
		Status        string `json:"status"`
		Domain        string `json:"domain"`
		Currency      string `json:"currency"`
		Amount        int64  `json:"amount"`
		Authorization struct {
			Code    string `json:"authorization_code"`
			Channel string `json:"channel"`
		} `json:"authorization"`
		Customer struct {
			Email string `json:"email"`
		} `json:"customer"`
	}
	if in.ExternalReference == "" || in.AmountKobo <= 0 {
		return collections.Response{}, errors.New("original reference and amount required")
	}
	e := c.call(ctx, "GET", "/transaction/verify/"+url.PathEscape(in.ExternalReference), nil, &t)
	if e != nil {
		return collections.Response{}, e
	}
	a, e := c.authorization(ctx, in.MandateReference)
	if e != nil {
		return collections.Response{}, e
	}
	domain := "test"
	if c.live {
		domain = "live"
	}
	if t.Reference != in.ExternalReference || t.Domain != domain || t.Currency != "NGN" || in.Currency != "NGN" || t.Amount != int64(in.AmountKobo) || t.Authorization.Code != a.Code || t.Authorization.Channel != "direct_debit" || !strings.EqualFold(t.Customer.Email, a.Customer.Email) {
		return collections.Response{}, errors.New("Paystack payment does not match the reserved debit")
	}
	out := collections.Response{State: collections.ProviderPending, ProviderCollectionID: in.ExternalReference}
	switch t.Status {
	case "success":
		out.State = collections.ProviderSucceeded
		out.SucceededAmountKobo = ledger.Money(t.Amount)
		out.SettlementState = collections.ProviderSettlementPending
	case "failed", "abandoned":
		out.State = collections.ProviderFailed
		out.FailureCode = "provider_declined"
	// A reversed transaction needs the controlled reversal flow; it must not be
	// mistaken for a failed debit and retried against the buyer.
	case "reversed":
		return collections.Response{}, errors.New("Paystack reversal requires financial review")
	}
	return out, nil
}

// Internal normalized signals are signed separately from native payloads.
func (c *Client) Sign(w collections.Webhook) string {
	w.Signature = ""
	b, _ := json.Marshal(w)
	m := hmac.New(sha512.New, []byte(c.secret))
	m.Write([]byte("kredit-reconciliation:"))
	m.Write(b)
	return hex.EncodeToString(m.Sum(nil))
}
func (c *Client) VerifyWebhook(w collections.Webhook) bool {
	return w.Signature != "" && hmac.Equal([]byte(w.Signature), []byte(c.Sign(w)))
}

type Notice struct {
	EventID   string `json:"event_id"`
	Type      string `json:"type"`
	Reference string `json:"reference"`
}

func (c *Client) ParseWebhook(signature string, raw []byte) (Notice, error) {
	supplied, e := hex.DecodeString(signature)
	m := hmac.New(sha512.New, []byte(c.secret))
	m.Write(raw)
	if e != nil || !hmac.Equal(supplied, m.Sum(nil)) || len(raw) > 1<<20 {
		return Notice{}, errors.New("invalid Paystack signature")
	}
	var event struct {
		Event string `json:"event"`
		Data  struct {
			Reference string `json:"reference"`
			Domain    string `json:"domain"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &event) != nil {
		return Notice{}, errors.New("invalid Paystack event")
	}
	// Authorization activation is refreshed through the authenticated buyer flow;
	// Paystack's authorization event has no initialization reference to route by.
	if event.Event != "charge.success" {
		return Notice{}, nil
	}
	domain := "test"
	if c.live {
		domain = "live"
	}
	if event.Data.Domain != domain || event.Data.Reference == "" {
		return Notice{}, errors.New("Paystack event mode or reference mismatch")
	}
	return Notice{EventID: hex.EncodeToString(m.Sum(nil)), Type: event.Event, Reference: event.Data.Reference}, nil
}

// Validation precedes the durable send fence: a missing email must not leave an
// interrupted provider request when no request could have been sent.
func (c *Client) ValidateAuthorization(ctx context.Context, in mandates.AuthorizationInput) error {
	if in.UserID == "" || in.AmountCeiling <= 0 {
		return errors.New("buyer and positive permission limit required")
	}
	email, e := c.email(ctx, in.UserID)
	if e != nil || email == "" {
		return errors.New("add and verify your email before setting up this bank permission")
	}
	return nil
}

// RecoverAuthorization is used only after an audited operator review. Paystack
// does not echo Kredit's client reference or local cap in this API: verify the
// original buyer, then retain the original locally agreed cap and validity.
func (c *Client) RecoverAuthorization(ctx context.Context, in mandates.AuthorizationInput, id string) (mandates.Mandate, error) {
	a, e := c.authorization(ctx, id)
	if e != nil {
		return mandates.Mandate{}, e
	}
	email, e := c.email(ctx, in.UserID)
	if e != nil || email == "" || !a.Active || !strings.EqualFold(email, a.Customer.Email) || in.AmountCeiling <= 0 {
		return mandates.Mandate{}, errors.New("Paystack permission is not active for the original buyer")
	}
	m := mandates.Mandate{Provider: c.name, ProviderAdapter: "paystack", ProviderID: id, Reference: in.Reference, Status: mandates.Active, Variable: true, AmountCeiling: in.AmountCeiling}
	if !in.RequiredUntil.IsZero() {
		m.EndsAt = in.RequiredUntil.Add(24 * time.Hour)
	}
	return m, nil
}
