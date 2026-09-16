package paystack

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"kredit/internal/settlement"
	"net/url"
)

func (c *Client) Banks(ctx context.Context) ([]settlement.Bank, error) {
	var banks []settlement.Bank
	if e := c.call(ctx, "GET", "/bank?country=nigeria&currency=NGN&perPage=100", nil, &banks); e != nil {
		return nil, e
	}
	return banks, settlement.ValidateBanks(banks)
}
func (c *Client) ConnectionIdentity() string {
	m := hmac.New(sha256.New, []byte(c.secret))
	m.Write([]byte("kredit-settlement-account:" + c.endpoint))
	return hex.EncodeToString(m.Sum(nil))
}
func (c *Client) CreateDestination(ctx context.Context, in settlement.Input) (settlement.Destination, error) {
	if e := settlement.ValidateInput(in); e != nil {
		return settlement.Destination{}, e
	}
	if in.BusinessName == "" {
		return settlement.Destination{}, errors.New("business name required")
	}
	var holder struct {
		Number string `json:"account_number"`
		Name   string `json:"account_name"`
	}
	if e := c.call(ctx, "GET", "/bank/resolve?account_number="+url.QueryEscape(in.AccountNumber)+"&bank_code="+url.QueryEscape(in.BankCode), nil, &holder); e != nil {
		return settlement.Destination{}, e
	}
	if holder.Number != in.AccountNumber || holder.Name == "" {
		return settlement.Destination{}, settlement.ErrUnknown
	}
	var out struct {
		Code     string `json:"subaccount_code"`
		Number   string `json:"account_number"`
		Name     string `json:"account_name"`
		Currency string `json:"currency"`
		Domain   string `json:"domain"`
	}
	e := c.call(ctx, "POST", "/subaccount", map[string]any{"business_name": in.BusinessName, "bank_code": in.BankCode, "account_number": in.AccountNumber, "percentage_charge": 0, "description": "Kredit seller " + in.OrganizationID, "primary_contact_email": in.Email}, &out)
	domain := "test"
	if c.live {
		domain = "live"
	}
	if e != nil || out.Number != in.AccountNumber || out.Name != holder.Name || out.Currency != "NGN" || out.Domain != domain {
		return settlement.Destination{}, settlement.ErrUnknown
	}
	d := settlement.Destination{ProviderReference: out.Code, BankCode: in.BankCode, AccountName: out.Name, AccountLast4: in.AccountNumber[6:]}
	return d, settlement.ValidateDestination(in, d)
}
