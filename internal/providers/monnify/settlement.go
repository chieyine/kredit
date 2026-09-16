package monnify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"kredit/internal/settlement"
)

func (c *Client) ConnectionIdentity() string {
	m := hmac.New(sha256.New, []byte(c.secret))
	m.Write([]byte("kredit-settlement-account:" + c.endpoint + ":" + c.key + ":" + c.contract))
	return hex.EncodeToString(m.Sum(nil))
}
func (c *Client) CreateDestination(ctx context.Context, in settlement.Input) (settlement.Destination, error) {
	if e := settlement.ValidateInput(in); e != nil {
		return settlement.Destination{}, e
	}
	if in.Email == "" {
		return settlement.Destination{}, errors.New("a business contact email is required")
	}
	var out []struct {
		Code     string `json:"subAccountCode"`
		Number   string `json:"accountNumber"`
		Name     string `json:"accountName"`
		Bank     string `json:"bankCode"`
		Currency string `json:"currencyCode"`
	}
	e := c.call(ctx, "POST", "/api/v1/sub-accounts", []map[string]any{{"currencyCode": "NGN", "accountNumber": in.AccountNumber, "bankCode": in.BankCode, "email": in.Email, "defaultSplitPercentage": 0}}, &out)
	if e != nil || len(out) != 1 || out[0].Number != in.AccountNumber || out[0].Bank != in.BankCode || out[0].Currency != "NGN" {
		return settlement.Destination{}, settlement.ErrUnknown
	}
	d := settlement.Destination{ProviderReference: out[0].Code, BankCode: out[0].Bank, AccountName: out[0].Name, AccountLast4: in.AccountNumber[6:]}
	return d, settlement.ValidateDestination(in, d)
}
