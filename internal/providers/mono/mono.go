// Package mono contains all Mono Sweep API and webhook vocabulary. Core Kredit
// packages only exchange provider-neutral mandates, debit requests, and events.
package mono

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
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
	"kredit/internal/identifier"
	"kredit/internal/ledger"
	"kredit/internal/mandates"
)

const DefaultBaseURL = "https://api.withmono.com"

type CustomerResolver func(context.Context, string, string) (string, error)

type Client struct {
	initiationDisabled                          bool
	baseURL, secret, webhookSecret, redirectURL string
	partial                                     bool
	resolve                                     CustomerResolver
	http                                        *http.Client
	now                                         func() time.Time
}

func New(baseURL, secret, webhookSecret, redirectURL string, partial bool, resolver CustomerResolver) (*Client, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return nil, errors.New("mono HTTPS API endpoint is required")
	}
	if secret == "" || webhookSecret == "" || redirectURL == "" || resolver == nil {
		return nil, errors.New("mono secret, webhook secret, redirect URL, and customer resolver are required")
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), secret: secret, webhookSecret: webhookSecret, redirectURL: redirectURL, partial: partial, resolve: resolver, http: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, now: func() time.Time { return time.Now().UTC() }}, nil
}

// ReconciliationOnly returns an immutable configuration that can observe and
// cancel existing mandates, but cannot create customers, mandates or debits.
func (c *Client) ReconciliationOnly() *Client {
	copy := *c
	copy.initiationDisabled = true
	return &copy
}
func (c *Client) Name() string { return "mono-sweep" }
func (c *Client) Capabilities() collections.Capabilities {
	return collections.Capabilities{AuthorizationSession: true, Recurring: true, Variable: true, MultiAccount: true, PartialRecovery: c.partial}
}

type envelope[T any] struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Data      T         `json:"data"`
}
type mandateData struct {
	ID          string    `json:"id"`
	MandateID   string    `json:"mandate_id"`
	Status      string    `json:"status"`
	Reference   string    `json:"reference"`
	Amount      int64     `json:"amount"`
	Balance     int64     `json:"balance"`
	MandateType string    `json:"mandate_type"`
	DebitType   string    `json:"debit_type"`
	MonoURL     string    `json:"mono_url"`
	Approved    bool      `json:"approved"`
	Ready       bool      `json:"ready_to_debit"`
	Start       time.Time `json:"-"`
	End         time.Time `json:"-"`
	StartRaw    string    `json:"start_date"`
	EndRaw      string    `json:"end_date"`
}
type debitData struct {
	Currency        string `json:"currency"`
	LiveMode        bool   `json:"live_mode"`
	Amount          int64  `json:"amount"`
	Collected       int64  `json:"collected_amount"`
	Pending         int64  `json:"pending_amount"`
	Status          string `json:"status"`
	Mandate         string `json:"mandate"`
	Reference       string `json:"reference"`
	ReferenceNumber string `json:"reference_number"`
	ResponseCode    string `json:"response_code"`
	Message         string `json:"message"`
}

func (c *Client) CreateAuthorizationSession(ctx context.Context, in mandates.AuthorizationInput) (mandates.Mandate, error) {
	if c.initiationDisabled {
		return mandates.Mandate{}, errors.New("new Mono authorizations are disabled")
	}
	customer, err := c.resolve(ctx, in.UserID, in.BusinessID)
	if err != nil || customer == "" {
		return mandates.Mandate{}, errors.New("registered Mono customer binding is required")
	}
	start := c.now()
	end := start.AddDate(1, 0, 0)
	if in.RequiredUntil.After(end) {
		end = in.RequiredUntil.Add(24 * time.Hour)
	}
	reference := strings.ReplaceAll(identifier.FromKey("mono-mandate", in.BusinessID+":"+in.Purpose), "-", "")
	body := map[string]any{"amount": in.AmountCeiling, "type": "recurring-debit", "method": "mandate", "mandate_type": "sweep", "debit_type": "variable", "allow_partial_sweep": c.partial, "description": "Kredit trade credit repayments", "reference": reference, "redirect_url": c.redirectURL, "customer": map[string]string{"id": customer}, "start_date": start.Format("2006-01-02"), "end_date": end.Format("2006-01-02"), "meta": map[string]string{"kredit_business_id": in.BusinessID}}
	var out envelope[mandateData]
	if err = c.request(ctx, http.MethodPost, "/v2/payments/initiate", body, &out); err != nil {
		return mandates.Mandate{}, err
	}
	id := out.Data.MandateID
	if id == "" {
		id = out.Data.ID
	}
	if !successfulEnvelope(out.Status) || id == "" || out.Data.MonoURL == "" {
		return mandates.Mandate{}, errors.New("mono returned an incomplete mandate authorization")
	}
	return mandates.Mandate{Provider: "mono-sweep", ProviderID: id, UserID: in.UserID, BusinessID: in.BusinessID, Status: mandates.Pending, AmountCeiling: in.AmountCeiling, AuthorizationURL: out.Data.MonoURL, StartsAt: start, EndsAt: end, Variable: true, MultiAccount: true, PartialRecovery: c.partial, CreatedAt: c.now()}, nil
}
func (c *Client) GetMandate(ctx context.Context, id string) (mandates.Mandate, error) {
	var out envelope[mandateData]
	if err := c.request(ctx, http.MethodGet, "/v3/payments/mandates/"+url.PathEscape(id), nil, &out); err != nil {
		return mandates.Mandate{}, err
	}
	d := out.Data
	if !successfulEnvelope(out.Status) || d.ID != id || d.Amount <= 0 || d.MandateType != "sweep" || d.DebitType != "variable" {
		return mandates.Mandate{}, errors.New("mono mandate identity or ceiling was not confirmed")
	}
	status := mandates.Pending
	switch strings.ToLower(d.Status) {
	case "cancelled":
		status = mandates.Cancelled
	case "paused", "suspended":
		status = mandates.Paused
	case "expired":
		status = mandates.Expired
	case "failed", "rejected":
		status = mandates.Failed
	}
	if status == mandates.Pending && d.Status == "approved" && d.Approved && d.Ready {
		status = mandates.Active
	}
	start, _ := time.Parse(time.RFC3339, d.StartRaw)
	end, _ := time.Parse(time.RFC3339, d.EndRaw)
	return mandates.Mandate{Provider: "mono-sweep", ProviderID: id, Status: status, AmountCeiling: d.Amount, StartsAt: start, EndsAt: end, Variable: true, MultiAccount: true, PartialRecovery: c.partial}, nil
}
func (c *Client) CancelMandate(ctx context.Context, id, _ string) (mandates.Mandate, error) {
	var out envelope[json.RawMessage]
	if err := c.request(ctx, http.MethodPatch, "/v3/payments/mandates/"+url.PathEscape(id)+"/cancel", nil, &out); err != nil {
		return mandates.Mandate{}, err
	}
	if out.Status != "success" {
		return mandates.Mandate{}, errors.New("mono did not confirm mandate cancellation")
	}
	return mandates.Mandate{Provider: "mono-sweep", ProviderID: id, Status: mandates.Cancelled}, nil
}
func (c *Client) RestoreAuthorization(context.Context, string) (mandates.Mandate, error) {
	return mandates.Mandate{}, errors.New("a cancelled Mono mandate requires fresh customer authorization")
}

func (c *Client) Submit(ctx context.Context, in collections.Request) (collections.Response, error) {
	if c.initiationDisabled {
		return collections.Response{}, errors.New("new Mono collections are disabled")
	}
	if in.AmountKobo <= 0 || in.Currency != "NGN" || in.ExternalReference == "" {
		return collections.Response{}, errors.New("valid NGN debit amount and reference are required")
	}
	if in.MandateReference == "" {
		return collections.Response{}, errors.New("active provider mandate is required")
	}
	narration := fmt.Sprintf("Kredit repayment Ref:%s", in.ExternalReference)
	body := map[string]any{"amount": in.AmountKobo, "reference": in.ExternalReference, "narration": narration}
	var out envelope[debitData]
	if err := c.request(ctx, http.MethodPost, "/v3/payments/mandates/"+url.PathEscape(in.MandateReference)+"/debit", body, &out); err != nil {
		return collections.Response{}, err
	}
	if !successfulEnvelope(out.Status) || (out.Data.Reference != "" && out.Data.Reference != in.ExternalReference) || (out.Data.Mandate != "" && out.Data.Mandate != in.MandateReference) {
		return collections.Response{}, errors.New("mono did not confirm a matching debit response; reconciliation required")
	}
	return debitResponse(out.Data, in.MandateReference, in.ExternalReference, in.AmountKobo), nil
}
func (c *Client) Get(context.Context, string) (collections.Response, error) {
	return collections.Response{}, errors.New("mono debit lookup requires the persisted mandate and reference")
}
func (c *Client) GetByReference(ctx context.Context, in collections.Request) (collections.Response, error) {
	var out envelope[debitData]
	path := "/v3/payments/mandates/" + url.PathEscape(in.MandateReference) + "/debit/" + url.PathEscape(in.ExternalReference)
	if err := c.request(ctx, http.MethodGet, path, nil, &out); err != nil {
		return collections.Response{}, err
	}
	if !successfulEnvelope(out.Status) || out.Data.Reference != in.ExternalReference || out.Data.Mandate != in.MandateReference {
		return collections.Response{}, errors.New("mono debit identity did not match the persisted request")
	}
	return debitResponse(out.Data, in.MandateReference, in.ExternalReference, in.AmountKobo), nil
}
func debitResponse(d debitData, mandate, reference string, requested ledger.Money) collections.Response {
	if d.LiveMode || (d.Currency != "" && d.Currency != "NGN") {
		return collections.Response{State: collections.ProviderPending}
	}
	state := collections.ProviderPending
	amount := ledger.Money(d.Amount)
	if d.Collected > 0 {
		amount = ledger.Money(d.Collected)
	}
	switch strings.ToLower(d.Status) {
	case "successful":
		state = collections.ProviderSucceeded
	case "partial-debit-successful":
		state = collections.ProviderPartial
	case "failed":
		state = collections.ProviderFailed
	}
	if state == collections.ProviderPartial {
		amount = ledger.Money(d.Collected)
	}
	if state == collections.ProviderFailed && d.Collected > 0 {
		state = collections.ProviderPartial
	}
	if state == collections.ProviderPending || state == collections.ProviderFailed {
		amount = 0
	}
	if (state == collections.ProviderSucceeded || state == collections.ProviderPartial) && (amount <= 0 || amount > requested) {
		state = collections.ProviderPending
		amount = 0
	}
	return collections.Response{State: state, ProviderCollectionID: mandate + ":" + reference, SucceededAmountKobo: amount, FailureCode: d.ResponseCode, Retryable: false}
}
func successfulEnvelope(status string) bool { return status == "successful" || status == "success" }
func (c *Client) Sign(event collections.Webhook) string {
	event.Signature = ""
	encoded, _ := json.Marshal(event)
	mac := hmac.New(sha256.New, []byte(c.webhookSecret))
	mac.Write(encoded)
	return hex.EncodeToString(mac.Sum(nil))
}
func (c *Client) VerifyWebhook(event collections.Webhook) bool {
	return event.Signature != "" && hmac.Equal([]byte(event.Signature), []byte(c.Sign(event)))
} // Raw Mono payloads are verified and translated before this boundary.
func (c *Client) request(ctx context.Context, method, path string, input, output any) error {
	var body io.Reader
	if input != nil {
		b, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	r, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	r.Header.Set("mono-sec-key", c.secret)
	r.Header.Set("Accept", "application/json")
	if input != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(r)
	if err != nil {
		return errors.New("mono request outcome is unknown; reconcile before retrying")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("mono returned HTTP %d", resp.StatusCode)
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(output); err != nil {
		return errors.New("mono returned an invalid response")
	}
	return nil
}
func (c *Client) VerifySecret(value string) bool {
	return value != "" && subtle.ConstantTimeCompare([]byte(value), []byte(c.webhookSecret)) == 1
}
