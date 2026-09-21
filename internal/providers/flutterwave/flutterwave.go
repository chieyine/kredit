package flutterwave

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
	"regexp"
	"strings"
	"time"

	"kredit/internal/collections"
	"kredit/internal/ledger"
	"kredit/internal/mandates"
	"kredit/internal/platform/httpjson"
	"kredit/internal/providers/bankdebit"
)

type Client struct {
	name, secret, webhookSecret, endpoint string
	store                                 bankdebit.RecoveryStore
	http                                  *http.Client
}

func New(name, secret, webhook string, live bool, store bankdebit.RecoveryStore) (*Client, error) {
	if name == "" || !strings.HasPrefix(secret, "FLWSECK") || strings.ContainsAny(secret+webhook, "\r\n") || len(webhook) < 32 || store == nil || strings.Contains(secret, "TEST") == live {
		return nil, errors.New("Flutterwave account, matching secret key, webhook secret and durable storage are required")
	}
	return &Client{name: name, secret: secret, webhookSecret: webhook, store: store, endpoint: "https://api.flutterwave.com/v3", http: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
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
	r, e := http.NewRequestWithContext(ctx, method, c.endpoint+path, body)
	if e != nil {
		return e
	}
	r.Header.Set("Authorization", "Bearer "+c.secret)
	r.Header.Set("Content-Type", "application/json")
	res, e := c.http.Do(r)
	if e != nil {
		return errors.New("Flutterwave request outcome is unconfirmed")
	}
	// The complete response read is checked below; Close only releases it.
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("Flutterwave returned HTTP %d; reconcile before retrying", res.StatusCode)
	}
	var envelope struct {
		Status string          `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	if e = httpjson.Decode(res.Body, 1<<20, &envelope); e != nil || envelope.Status != "success" {
		return errors.New("Flutterwave did not confirm the request")
	}
	if out != nil {
		return json.Unmarshal(envelope.Data, out)
	}
	return nil
}
func (c *Client) CreateAuthorizationSession(ctx context.Context, in mandates.AuthorizationInput) (mandates.Mandate, error) {
	if in.RequiredUntil.After(time.Now().Add(364 * 24 * time.Hour)) {
		return mandates.Mandate{}, errors.New("Flutterwave bank permission cannot cover this sale's repayment period")
	}
	m, e := c.store.Create(ctx, c.name, in)
	m.ProviderAdapter = "flutterwave"
	return m, e
}
func (c *Client) Enrollment(ctx context.Context, ref string) (bankdebit.Enrollment, error) {
	return c.store.Load(ctx, c.name, ref)
}
func (c *Client) CompleteEnrollment(ctx context.Context, ref, user string, d bankdebit.Details) (bankdebit.Enrollment, error) {
	banks, _ := c.Banks(ctx)
	supported := false
	for _, bank := range banks {
		if bank.Code == d.BankCode {
			supported = true
			break
		}
	}
	if !supported {
		return bankdebit.Enrollment{}, errors.New("choose a bank supported for direct debit")
	}
	v, e := c.store.Begin(ctx, c.name, ref, user, d)
	if e != nil {
		return v, e
	}
	end := v.Input.RequiredUntil.Add(24 * time.Hour)
	if end.Before(time.Now().Add(30 * 24 * time.Hour)) {
		end = time.Now().Add(30 * 24 * time.Hour)
	}
	var data struct {
		Reference string      `json:"reference"`
		Currency  string      `json:"currency"`
		Amount    json.Number `json:"amount"`
		Consent   struct {
			BankName      string      `json:"bank_name"`
			AccountName   string      `json:"account_name"`
			AccountNumber string      `json:"account_number"`
			Amount        json.Number `json:"amount"`
		} `json:"mandate_consent"`
	}
	e = c.call(ctx, "POST", "/accounts/tokenize", map[string]any{"email": d.Email, "amount": bankdebit.Naira(v.Input.AmountCeiling), "address": d.Address, "phone_number": d.Phone, "account_bank": d.BankCode, "account_number": d.AccountNumber, "end_date": end.UTC().Format(time.RFC3339), "narration": "Kredit bank permission " + ref}, &data)
	if e != nil {
		return v, e
	}
	amount, e := bankdebit.Kobo(data.Amount)
	fee, feeErr := bankdebit.Kobo(data.Consent.Amount)
	if e != nil || feeErr != nil || amount != v.Input.AmountCeiling || data.Currency != "NGN" || data.Reference == "" || fee <= 0 || data.Consent.BankName == "" || len(data.Consent.AccountNumber) != 10 {
		return v, errors.New("Flutterwave returned incomplete activation instructions")
	}
	result := bankdebit.Result{Reference: data.Reference, Transfer: &bankdebit.ConsentTransfer{BankName: data.Consent.BankName, AccountName: data.Consent.AccountName, AccountNumber: data.Consent.AccountNumber, AmountKobo: fee, ExpiresAt: time.Now().Add(10 * time.Minute)}}
	if e = c.store.Confirm(ctx, v, result); e != nil {
		return v, e
	}
	v.Result = result
	v.State = "CONFIRMED"
	return v, nil
}

type token struct {
	Narration  string      `json:"narration"`
	Token      string      `json:"token"`
	Reference  string      `json:"reference"`
	Status     string      `json:"status"`
	Currency   string      `json:"currency"`
	Amount     json.Number `json:"amount"`
	StartsAt   time.Time   `json:"start_date"`
	EndsAt     time.Time   `json:"end_date"`
	CustomerID int64       `json:"customer_id"`
}

func (c *Client) readToken(ctx context.Context, v bankdebit.Enrollment) (token, error) {
	var t token
	if v.State != "CONFIRMED" || v.Result.Reference == "" {
		return t, errors.New("bank authorization needs completion")
	}
	e := c.call(ctx, "GET", "/accounts/token/"+url.PathEscape(v.Result.Reference), nil, &t)
	if e != nil {
		return t, e
	}
	amount, e := bankdebit.Kobo(t.Amount)
	if e != nil || t.Reference != v.Result.Reference || t.Narration != "Kredit bank permission "+v.Reference || t.Currency != "NGN" || amount <= 0 || amount > v.Input.AmountCeiling {
		return t, errors.New("Flutterwave authorization does not match the saved permission")
	}
	return t, nil
}
func (c *Client) GetMandate(ctx context.Context, ref string) (mandates.Mandate, error) {
	v, e := c.Enrollment(ctx, ref)
	if e != nil {
		return mandates.Mandate{}, e
	}
	m := mandates.Mandate{Provider: c.name, ProviderAdapter: "flutterwave", ProviderID: ref, Status: mandates.Pending, AmountCeiling: v.Input.AmountCeiling, Variable: true}
	if v.State == "CANCELLED" {
		m.Status = mandates.Cancelled
		return m, nil
	}
	if v.State != "CONFIRMED" {
		return m, nil
	}
	t, e := c.readToken(ctx, v)
	if e != nil {
		return m, e
	}
	m.StartsAt = t.StartsAt
	m.EndsAt = t.EndsAt
	m.AmountCeiling, _ = bankdebit.Kobo(t.Amount)
	switch strings.ToUpper(t.Status) {
	case "ACTIVE":
		if t.Token == "" {
			return m, errors.New("active token missing")
		}
		m.Status = mandates.Active
	case "SUSPENDED":
		m.Status = mandates.Paused
	case "DELETED":
		m.Status = mandates.Cancelled
	}
	return m, nil
}
func (c *Client) CancelMandate(ctx context.Context, ref, reason string) (mandates.Mandate, error) {
	v, e := c.Enrollment(ctx, ref)
	if e != nil {
		return mandates.Mandate{}, e
	}
	if v.State == "DRAFT" || v.State == "CANCELLED" {
		e = c.store.CancelDraft(ctx, v)
	} else if v.State == "CONFIRMED" {
		e = c.call(ctx, "PUT", "/accounts/token/"+url.PathEscape(v.Result.Reference), map[string]string{"status": "DELETED"}, nil)
	} else {
		e = errors.New("authorization outcome must be confirmed before cancellation")
	}
	if e != nil {
		return mandates.Mandate{}, e
	}
	m, e := c.GetMandate(ctx, ref)
	if e == nil && m.Status != mandates.Cancelled {
		e = errors.New("bank cancellation is awaiting confirmation")
	}
	return m, e
}
func (c *Client) RestoreAuthorization(context.Context, string) (mandates.Mandate, error) {
	return mandates.Mandate{}, errors.New("give fresh bank permission from your sale")
}
func (c *Client) Submit(ctx context.Context, in collections.Request) (collections.Response, error) {
	if in.AmountKobo <= 0 || in.Currency != "NGN" || in.ExternalReference == "" {
		return collections.Response{}, errors.New("positive NGN amount and original reference required")
	}
	if in.SettlementRoute != nil && in.SettlementRoute.Method == "provider_split" {
		return collections.Response{}, errors.New("Flutterwave seller split is not configured")
	}
	v, e := c.Enrollment(ctx, in.MandateReference)
	if e != nil {
		return collections.Response{}, e
	}
	t, e := c.readToken(ctx, v)
	if e != nil {
		return collections.Response{}, e
	}
	limit, _ := bankdebit.Kobo(t.Amount)
	if t.Status != "ACTIVE" || t.Token == "" || time.Now().Before(t.StartsAt) || !time.Now().Before(t.EndsAt) || int64(in.AmountKobo) > limit {
		return collections.Response{}, errors.New("bank authorization is not available for this amount")
	}
	e = c.call(ctx, "POST", "/tokenized-charges", map[string]any{"token": t.Token, "email": v.Details.Email, "amount": bankdebit.Naira(int64(in.AmountKobo)), "tx_ref": in.ExternalReference, "type": "account"}, nil)
	if e != nil {
		return collections.Response{}, e
	}
	return collections.Response{State: collections.ProviderPending, ProviderCollectionID: in.ExternalReference}, nil
}
func (c *Client) Get(context.Context, string) (collections.Response, error) {
	return collections.Response{}, errors.New("original debit request required")
}
func (c *Client) GetByReference(ctx context.Context, in collections.Request) (collections.Response, error) {
	if in.ExternalReference == "" || in.Currency != "NGN" || in.AmountKobo <= 0 {
		return collections.Response{}, errors.New("original NGN debit request required")
	}
	v, e := c.Enrollment(ctx, in.MandateReference)
	if e != nil {
		return collections.Response{}, e
	}
	var data struct {
		Reference   string      `json:"tx_ref"`
		Status      string      `json:"status"`
		Amount      json.Number `json:"amount"`
		Currency    string      `json:"currency"`
		PaymentType string      `json:"payment_type"`
		AuthModel   string      `json:"auth_model"`
		Customer    struct {
			Email string `json:"email"`
		} `json:"customer"`
		Account struct {
			Number string `json:"account_number"`
			Bank   string `json:"bank_code"`
		} `json:"account"`
	}
	e = c.call(ctx, "GET", "/transactions/verify_by_reference?tx_ref="+url.QueryEscape(in.ExternalReference), nil, &data)
	if e != nil {
		return collections.Response{}, e
	}
	amount, e := bankdebit.Kobo(data.Amount)
	if e != nil || amount != int64(in.AmountKobo) || data.Reference != in.ExternalReference || data.Currency != "NGN" || data.PaymentType != "account" || data.AuthModel != "EMANDATE" || !strings.EqualFold(data.Customer.Email, v.Details.Email) || data.Account.Number != v.Details.AccountNumber || data.Account.Bank != v.Details.BankCode {
		return collections.Response{}, errors.New("Flutterwave transaction does not match the reserved debit")
	}
	out := collections.Response{State: collections.ProviderPending, ProviderCollectionID: in.ExternalReference}
	switch data.Status {
	case "successful":
		out.State = collections.ProviderSucceeded
		out.SucceededAmountKobo = ledger.Money(amount)
		out.SettlementState = collections.ProviderSettlementPending
	case "failed":
		out.State = collections.ProviderFailed
		out.FailureCode = "provider_declined"
	}
	return out, nil
}
func (c *Client) Sign(w collections.Webhook) string {
	w.Signature = ""
	b, _ := json.Marshal(w)
	m := hmac.New(sha256.New, []byte(c.secret))
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
	// v3 direct debit uses the documented verif-hash contract. Only an
	// authenticated API lookup can turn this notification into a payment.
	if len(raw) > 1<<20 || !hmac.Equal([]byte(headers.Get("verif-hash")), []byte(c.webhookSecret)) {
		return bankdebit.Notice{}, errors.New("invalid Flutterwave webhook")
	}
	var data struct {
		Event string `json:"event"`
		Data  struct {
			Reference string `json:"reference"`
			TxRef     string `json:"tx_ref"`
			Narration string `json:"narration"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &data) != nil {
		return bankdebit.Notice{}, errors.New("invalid Flutterwave payload")
	}
	hash := sha256.Sum256(raw)
	n := bankdebit.Notice{EventID: hex.EncodeToString(hash[:]), Type: data.Event}
	switch data.Event {
	case "charge.completed":
		n.PaymentReference = data.Data.TxRef
		if n.PaymentReference == "" {
			return bankdebit.Notice{}, errors.New("payment reference missing")
		}
	case "account.tokenize":
		n.MandateReference = strings.TrimPrefix(data.Data.Narration, "Kredit bank permission ")
		if n.MandateReference == "" {
			return bankdebit.Notice{}, errors.New("mandate reference missing")
		}
	default:
		return bankdebit.Notice{}, nil
	}
	return n, nil
}

func (c *Client) Recover(ctx context.Context, actor, ref, action, providerRef, reason string) error {
	v, e := c.store.Review(ctx, actor, c.name, ref)
	if e != nil {
		return e
	}
	result := bankdebit.Result{}
	if action == "link" {
		v.State = "CONFIRMED"
		v.Result.Reference = providerRef
		t, e := c.readToken(ctx, v)
		if e != nil {
			return e
		}
		if t.Status != "ACTIVE" && t.Status != "APPROVED" {
			return errors.New("only a customer-authorized bank permission can be recovered")
		}
		result = v.Result
	}
	return c.store.Resolve(ctx, actor, v, action, result, reason)
}

func (c *Client) ValidateAuthorization(_ context.Context, in mandates.AuthorizationInput) error {
	if in.RequiredUntil.After(time.Now().Add(364 * 24 * time.Hour)) {
		return errors.New("Flutterwave bank permission cannot cover this repayment period; choose another collector")
	}
	return nil
}
