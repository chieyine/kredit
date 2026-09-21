package monnify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"kredit/internal/collections"
	"kredit/internal/ledger"
	"kredit/internal/mandates"
	"kredit/internal/platform/httpjson"
	"kredit/internal/providers/bankdebit"
)

type Client struct {
	name, key, secret, contract, callback, endpoint string
	store                                           bankdebit.RecoveryStore
	http                                            *http.Client
	mu                                              sync.Mutex
	token                                           string
	expires                                         time.Time
}

func New(name, key, secret, contract, callback string, live bool, store bankdebit.RecoveryStore) (*Client, error) {
	u, e := url.Parse(callback)
	if name == "" || key == "" || secret == "" || contract == "" || strings.ContainsAny(key+secret, "\r\n") || store == nil || e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return nil, errors.New("monnify requires API key, secret, contract code, HTTPS return URL and durable storage")
	}
	endpoint := "https://sandbox.monnify.com"
	if live {
		endpoint = "https://api.monnify.com"
	}
	return &Client{name: name, key: key, secret: secret, contract: contract, callback: callback, endpoint: endpoint, store: store, http: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) Name() string { return c.name }
func (c *Client) Capabilities() collections.Capabilities {
	return collections.Capabilities{AuthorizationSession: true, OneTime: true, Recurring: true, Variable: true, SupportedCurrencies: []string{"NGN"}}
}
func (c *Client) request(ctx context.Context, method, path, auth string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, e := json.Marshal(in)
		if e != nil {
			return e
		}
		body = bytes.NewReader(b)
	}
	r, e := http.NewRequestWithContext(ctx, method, c.endpoint+path, body)
	if e != nil {
		return e
	}
	r.Header.Set("Authorization", auth)
	r.Header.Set("Content-Type", "application/json")
	res, e := c.http.Do(r)
	if e != nil {
		return errors.New("monnify request outcome is unconfirmed")
	}
	// The complete response read is checked below; Close only releases it.
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("monnify returned HTTP %d; reconcile before retrying", res.StatusCode)
	}
	var envelope struct {
		Successful bool            `json:"requestSuccessful"`
		Code       string          `json:"responseCode"`
		Body       json.RawMessage `json:"responseBody"`
	}
	if e = httpjson.Decode(res.Body, 1<<20, &envelope); e != nil || !envelope.Successful || envelope.Code != "0" {
		return errors.New("monnify did not confirm the request")
	}
	if out != nil {
		return json.Unmarshal(envelope.Body, out)
	}
	return nil
}
func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.expires) {
		return c.token, nil
	}
	var data struct {
		Token   string `json:"accessToken"`
		Expires int64  `json:"expiresIn"`
	}
	e := c.request(ctx, "POST", "/api/v1/auth/login", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.key+":"+c.secret)), nil, &data)
	if e != nil {
		return "", e
	}
	if data.Token == "" || data.Expires <= 60 {
		return "", errors.New("monnify login returned invalid credentials")
	}
	c.token = data.Token
	c.expires = time.Now().Add(time.Duration(min(data.Expires, 3600)-30) * time.Second)
	return c.token, nil
}
func (c *Client) call(ctx context.Context, method, path string, in, out any) error {
	token, e := c.accessToken(ctx)
	if e != nil {
		return e
	}
	return c.request(ctx, method, path, "Bearer "+token, in, out)
}
func (c *Client) CreateAuthorizationSession(ctx context.Context, in mandates.AuthorizationInput) (mandates.Mandate, error) {
	m, e := c.store.Create(ctx, c.name, in)
	m.ProviderAdapter = "monnify"
	return m, e
}
func (c *Client) Enrollment(ctx context.Context, ref string) (bankdebit.Enrollment, error) {
	return c.store.Load(ctx, c.name, ref)
}
func (c *Client) CompleteEnrollment(ctx context.Context, ref, user string, d bankdebit.Details) (bankdebit.Enrollment, error) {
	v, e := c.store.Begin(ctx, c.name, ref, user, d)
	if e != nil {
		return v, e
	}
	end := v.Input.RequiredUntil.Add(24 * time.Hour)
	if end.Before(time.Now().Add(30 * 24 * time.Hour)) {
		end = time.Now().Add(30 * 24 * time.Hour)
	}
	// The published OpenAPI schema uses mandateStartDate/mandateEndDate and
	// mandateCode/debitAmount; the introductory guide uses obsolete aliases.
	var result struct {
		Reference string `json:"mandateReference"`
		Code      string `json:"mandateCode"`
	}
	e = c.call(ctx, "POST", "/api/v1/direct-debit/mandate/create", map[string]any{"contractCode": c.contract, "mandateReference": ref, "mandateAmount": bankdebit.Naira(v.Input.AmountCeiling), "autoRenew": false, "customerCancellation": true, "customerName": d.Name, "customerEmailAddress": d.Email, "customerPhoneNumber": d.Phone, "customerAddress": d.Address, "customerAccountNumber": d.AccountNumber, "customerAccountBankCode": d.BankCode, "mandateDescription": "Kredit bank permission", "mandateStartDate": time.Now().UTC().Format("2006-01-02T15:04:05"), "mandateEndDate": end.UTC().Format("2006-01-02T15:04:05"), "redirectUrl": c.callback, "debitAmount": nil}, &result)
	if e != nil {
		return v, e
	}
	if result.Reference != ref || result.Code == "" {
		return v, errors.New("monnify returned mismatched authorization")
	}
	// Save the known identity before fetching activation instructions, so a
	// delayed GET never loses a successfully created mandate.
	v.Result = bankdebit.Result{Reference: ref}
	if e = c.store.Confirm(ctx, v, v.Result); e != nil {
		return v, e
	}
	v.State = "CONFIRMED"
	status, e := c.readMandate(ctx, v)
	if e != nil {
		return v, e
	}
	v.Result.AuthorizationURL = safeURL(status.AuthorizationLink)
	return v, nil
}

type mandate struct {
	Reference         string      `json:"mandateReference"`
	Code              string      `json:"mandateCode"`
	Status            string      `json:"mandateStatus"`
	Amount            json.Number `json:"mandateAmount"`
	Contract          string      `json:"contractCode"`
	Email             string      `json:"customerEmailAddress"`
	Account           string      `json:"customerAccountNumber"`
	Bank              string      `json:"customerAccountBankCode"`
	Start             string      `json:"startDate"`
	End               string      `json:"endDate"`
	AuthorizationLink string      `json:"authorizationLink"`
}

func safeURL(raw string) string {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.User != nil || (!strings.HasSuffix(u.Hostname(), ".monnify.com") && u.Hostname() != "monnify.com") {
		return ""
	}
	return raw
}
func (c *Client) readMandate(ctx context.Context, v bankdebit.Enrollment) (mandate, error) {
	var result []mandate
	if v.State != "CONFIRMED" {
		return mandate{}, errors.New("complete bank authorization first")
	}
	e := c.call(ctx, "GET", "/api/v1/direct-debit/mandate/?mandateReferences="+url.QueryEscape(v.Result.Reference), nil, &result)
	if e != nil {
		return mandate{}, e
	}
	if len(result) != 1 {
		return mandate{}, errors.New("monnify authorization is not uniquely identified")
	}
	m := result[0]
	amount, e := bankdebit.Kobo(m.Amount)
	if e != nil || m.Reference != v.Result.Reference || m.Code == "" || m.Contract != c.contract || amount <= 0 || amount > v.Input.AmountCeiling || !strings.EqualFold(m.Email, v.Details.Email) || m.Account != v.Details.AccountNumber || m.Bank != v.Details.BankCode {
		return m, errors.New("monnify authorization does not match the saved bank permission")
	}
	return m, nil
}
func date(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000-0700", "2006-01-02T15:04:05"} {
		if t, e := time.Parse(layout, s); e == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("monnify returned an invalid authorization date")
}
func (c *Client) GetMandate(ctx context.Context, ref string) (mandates.Mandate, error) {
	v, e := c.Enrollment(ctx, ref)
	if e != nil {
		return mandates.Mandate{}, e
	}
	m := mandates.Mandate{Provider: c.name, ProviderAdapter: "monnify", ProviderID: ref, AmountCeiling: v.Input.AmountCeiling, Status: mandates.Pending, Variable: true}
	if v.State == "CANCELLED" {
		m.Status = mandates.Cancelled
		return m, nil
	}
	if v.State != "CONFIRMED" {
		return m, nil
	}
	data, e := c.readMandate(ctx, v)
	if e != nil {
		return m, e
	}
	m.AmountCeiling, _ = bankdebit.Kobo(data.Amount)
	m.StartsAt, e = date(data.Start)
	if e != nil {
		return m, e
	}
	m.EndsAt, e = date(data.End)
	if e != nil {
		return m, e
	}
	switch data.Status {
	case "ACTIVATED", "ACTIVE":
		m.Status = mandates.Active
	case "SUSPENDED":
		m.Status = mandates.Paused
	case "CANCELLED":
		m.Status = mandates.Cancelled
	case "EXPIRED", "AUTHORIZATION_EXPIRED":
		m.Status = mandates.Expired
	}
	return m, nil
}
func (c *Client) Instructions(ctx context.Context, ref string) (bankdebit.Enrollment, error) {
	v, e := c.Enrollment(ctx, ref)
	if e != nil || v.State != "CONFIRMED" {
		return v, e
	}
	m, e := c.readMandate(ctx, v)
	if e != nil {
		return v, e
	}
	v.Result.AuthorizationURL = safeURL(m.AuthorizationLink)
	return v, nil
}
func (c *Client) CancelMandate(ctx context.Context, ref, reason string) (mandates.Mandate, error) {
	v, e := c.Enrollment(ctx, ref)
	if e != nil {
		return mandates.Mandate{}, e
	}
	switch v.State {
	case "DRAFT", "CANCELLED":
		e = c.store.CancelDraft(ctx, v)
	case "CONFIRMED":
		m, err := c.readMandate(ctx, v)
		if err != nil {
			return mandates.Mandate{}, err
		}
		e = c.call(ctx, "PATCH", "/api/v1/direct-debit/mandate/cancel-mandate/"+url.PathEscape(m.Code), nil, nil)
	default:
		e = errors.New("authorization outcome must be confirmed first")
	}
	if e != nil {
		return mandates.Mandate{}, e
	}
	m, e := c.GetMandate(ctx, ref)
	if e == nil && m.Status != mandates.Cancelled {
		e = errors.New("bank cancellation is awaiting confirmation; check again shortly")
	}
	return m, e
}
func (c *Client) RestoreAuthorization(context.Context, string) (mandates.Mandate, error) {
	return mandates.Mandate{}, errors.New("give fresh bank permission from your sale")
}
func (c *Client) Submit(ctx context.Context, in collections.Request) (collections.Response, error) {
	if in.AmountKobo <= 0 || in.Currency != "NGN" || in.ExternalReference == "" {
		return collections.Response{}, errors.New("original positive NGN debit required")
	}
	if in.SettlementRoute != nil && in.SettlementRoute.Method == "provider_split" {
		return collections.Response{}, errors.New("monnify seller split is not configured")
	}
	v, e := c.Enrollment(ctx, in.MandateReference)
	if e != nil {
		return collections.Response{}, e
	}
	m, e := c.readMandate(ctx, v)
	if e != nil {
		return collections.Response{}, e
	}
	start, e1 := date(m.Start)
	end, e2 := date(m.End)
	amount, _ := bankdebit.Kobo(m.Amount)
	if e1 != nil || e2 != nil || time.Now().Before(start) || !time.Now().Before(end) || (m.Status != "ACTIVATED" && m.Status != "ACTIVE") || int64(in.AmountKobo) > amount {
		return collections.Response{}, errors.New("bank permission is not available for this debit")
	}
	e = c.call(ctx, "POST", "/api/v1/direct-debit/mandate/debit", map[string]any{"paymentReference": in.ExternalReference, "mandateCode": m.Code, "debitAmount": bankdebit.Naira(int64(in.AmountKobo)), "narration": "Kredit scheduled repayment", "customerEmail": v.Details.Email}, nil)
	if e != nil {
		return collections.Response{}, e
	}
	return collections.Response{State: collections.ProviderPending, ProviderCollectionID: in.ExternalReference}, nil
}
func (c *Client) Get(context.Context, string) (collections.Response, error) {
	return collections.Response{}, errors.New("original debit request required")
}
func (c *Client) GetByReference(ctx context.Context, in collections.Request) (collections.Response, error) {
	if in.ExternalReference == "" || in.AmountKobo <= 0 || in.Currency != "NGN" {
		return collections.Response{}, errors.New("original NGN debit required")
	}
	v, e := c.Enrollment(ctx, in.MandateReference)
	if e != nil {
		return collections.Response{}, e
	}
	m, e := c.readMandate(ctx, v)
	if e != nil {
		return collections.Response{}, e
	}
	var data struct {
		Status      string      `json:"transactionStatus"`
		Reference   string      `json:"paymentReference"`
		Amount      json.Number `json:"debitAmount"`
		MandateCode string      `json:"mandateCode"`
	}
	e = c.call(ctx, "GET", "/api/v1/direct-debit/mandate/debit-status?paymentReference="+url.QueryEscape(in.ExternalReference), nil, &data)
	if e != nil {
		return collections.Response{}, e
	}
	amount, e := bankdebit.Kobo(data.Amount)
	if e != nil || amount != int64(in.AmountKobo) || data.Reference != in.ExternalReference || data.MandateCode != m.Code {
		return collections.Response{}, errors.New("monnify payment does not match the reserved debit")
	}
	out := collections.Response{State: collections.ProviderPending, ProviderCollectionID: in.ExternalReference}
	switch data.Status {
	case "PAID":
		out.State = collections.ProviderSucceeded
		out.SucceededAmountKobo = ledger.Money(amount)
		out.SettlementState = collections.ProviderSettlementPending
	case "FAILED":
		out.State = collections.ProviderFailed
		out.FailureCode = "provider_declined"
	}
	return out, nil
}
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
func (c *Client) LocalReference(ctx context.Context, ref string) (string, error) {
	if !regexp.MustCompile(`^[a-f0-9]{32}$`).MatchString(ref) {
		return "", errors.New("invalid local authorization reference")
	}
	return ref, nil
}
func (c *Client) ParseNotice(headers http.Header, raw []byte) (bankdebit.Notice, error) {
	supplied, e := hex.DecodeString(headers.Get("monnify-signature"))
	m := hmac.New(sha512.New, []byte(c.secret))
	m.Write(raw)
	if e != nil || len(raw) > 1<<20 || !hmac.Equal(supplied, m.Sum(nil)) {
		return bankdebit.Notice{}, errors.New("invalid Monnify signature")
	}
	var event struct {
		Type string `json:"eventType"`
		Data struct {
			PaymentReference string `json:"paymentReference"`
		} `json:"eventData"`
	}
	if json.Unmarshal(raw, &event) != nil {
		return bankdebit.Notice{}, errors.New("invalid Monnify payload")
	}
	if event.Type != "SUCCESSFUL_TRANSACTION" {
		return bankdebit.Notice{}, nil
	}
	if event.Data.PaymentReference == "" {
		return bankdebit.Notice{}, errors.New("payment reference required")
	}
	return bankdebit.Notice{EventID: hex.EncodeToString(m.Sum(nil)), Type: event.Type, PaymentReference: event.Data.PaymentReference}, nil
}

func (c *Client) Recover(ctx context.Context, actor, ref, action, providerRef, reason string) error {
	v, e := c.store.Review(ctx, actor, c.name, ref)
	if e != nil {
		return e
	}
	result := bankdebit.Result{}
	if action == "link" {
		if providerRef != ref {
			return errors.New("use the original Monnify mandate reference")
		}
		v.State = "CONFIRMED"
		v.Result.Reference = providerRef
		m, e := c.readMandate(ctx, v)
		if e != nil {
			return e
		}
		result = bankdebit.Result{Reference: providerRef, AuthorizationURL: safeURL(m.AuthorizationLink)}
	}
	return c.store.Resolve(ctx, actor, v, action, result, reason)
}
