package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Connector implements Kredit's fee-provider contract. Provider-specific APIs
// belong behind this adapter boundary; changing an account never reroutes work.
type Connector struct {
	name, endpoint, token string
	client                *http.Client
	readOnly              bool
}

func NewConnector(name, endpoint, token string, readOnly bool) (*Connector, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || token == "" || name == "" {
		return nil, errors.New("valid HTTPS fee connector account required")
	}
	return &Connector{name, strings.TrimRight(endpoint, "/"), token, &http.Client{Timeout: 25 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, readOnly}, nil
}
func (c *Connector) Name() string { return c.name }
func (c *Connector) call(ctx context.Context, method, path string, in, out any) error {
	if c.readOnly && method != http.MethodGet {
		return errors.New("saved fee account only supports reconciliation")
	}
	var b []byte
	var err error
	if in != nil {
		b, err = json.Marshal(in)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return errors.New("fee provider request is unconfirmed")
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.New("fee provider response is unconfirmed")
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 || json.Unmarshal(raw, out) != nil {
		return errors.New("invalid fee provider response")
	}
	return nil
}
func (c *Connector) CreateFeeCustomer(ctx context.Context, in FeeCustomer) (string, error) {
	var out struct {
		ID string `json:"id"`
	}
	err := c.call(ctx, "POST", "/fee-customers", in, &out)
	if err == nil && out.ID == "" {
		err = errors.New("customer reference missing")
	}
	return out.ID, err
}
func (c *Connector) FeeCustomerIdentity(ctx context.Context, id string) (string, error) {
	var out struct {
		ID  string `json:"id"`
		BVN string `json:"bvn"`
	}
	err := c.call(ctx, "GET", "/fee-customers/"+url.PathEscape(id), nil, &out)
	if err == nil && (out.ID != id || len(out.BVN) != 11) {
		err = errors.New("customer identity not confirmed")
	}
	return out.BVN, err
}
func (c *Connector) CreateFeeAuthorization(ctx context.Context, in FeeAuthorization) (FeeAuthorization, error) {
	var out FeeAuthorization
	err := c.call(ctx, "POST", "/fee-authorizations", in, &out)
	u, e := url.Parse(out.URL)
	if err == nil && (out.ID == "" || out.Reference != in.Reference || out.Customer != in.Customer || out.Ceiling != in.Ceiling || !out.StartsAt.Equal(in.StartsAt) || !out.EndsAt.Equal(in.EndsAt) || e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil) {
		err = errors.New("fee authorization response does not match")
	}
	return out, err
}
func (c *Connector) ReadFeeAuthorization(ctx context.Context, id string) (FeeAuthorization, error) {
	var out FeeAuthorization
	err := c.call(ctx, "GET", "/fee-authorizations/"+url.PathEscape(id), nil, &out)
	if err == nil && out.ID != id {
		err = errors.New("fee authorization identity mismatch")
	}
	return out, err
}
func (c *Connector) debit(ctx context.Context, method, path string, in FeeDebitRequest) (FeeDebitResult, error) {
	var out struct {
		FeeDebitResult
		Mandate   string `json:"mandate"`
		Reference string `json:"reference"`
		Requested int64  `json:"requested_amount_kobo"`
	}
	err := c.call(ctx, method, path, in, &out)
	if err == nil && (out.Reference != in.Reference || out.Mandate != in.Mandate || out.Requested != in.Amount) {
		err = errors.New("fee debit identity mismatch")
	}
	return out.FeeDebitResult, err
}
func (c *Connector) SubmitFeeDebit(ctx context.Context, in FeeDebitRequest) (FeeDebitResult, error) {
	return c.debit(ctx, "POST", "/fee-debits", in)
}
func (c *Connector) ReadFeeDebit(ctx context.Context, in FeeDebitRequest) (FeeDebitResult, error) {
	return c.debit(ctx, "GET", "/fee-debits/"+url.PathEscape(in.Reference), in)
}
