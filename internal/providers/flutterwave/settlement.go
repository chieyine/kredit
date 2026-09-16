package flutterwave

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
	m.Write([]byte("kredit-settlement-account:" + c.endpoint))
	return hex.EncodeToString(m.Sum(nil))
}
func (c *Client) CreateDestination(ctx context.Context, in settlement.Input) (settlement.Destination, error) {
	if e := settlement.ValidateInput(in); e != nil {
		return settlement.Destination{}, e
	}
	if in.BusinessName == "" || in.Phone == "" {
		return settlement.Destination{}, errors.New("business name and verified contact phone are required")
	}
	var holder struct {
		Number string `json:"account_number"`
		Name   string `json:"account_name"`
	}
	e := c.call(ctx, "POST", "/accounts/resolve", map[string]string{"account_bank": in.BankCode, "account_number": in.AccountNumber}, &holder)
	if e != nil || holder.Number != in.AccountNumber || holder.Name == "" {
		return settlement.Destination{}, settlement.ErrUnknown
	}
	var out struct {
		Code   string `json:"subaccount_id"`
		Number string `json:"account_number"`
		Bank   string `json:"account_bank"`
	}
	e = c.call(ctx, "POST", "/subaccounts", map[string]any{"account_bank": in.BankCode, "account_number": in.AccountNumber, "business_name": in.BusinessName, "business_email": in.Email, "business_mobile": in.Phone, "country": "NG", "split_type": "flat", "split_value": 0}, &out)
	if e != nil || out.Number != in.AccountNumber || out.Bank != in.BankCode {
		return settlement.Destination{}, settlement.ErrUnknown
	}
	d := settlement.Destination{ProviderReference: out.Code, BankCode: out.Bank, AccountName: holder.Name, AccountLast4: in.AccountNumber[6:]}
	return d, settlement.ValidateDestination(in, d)
}
