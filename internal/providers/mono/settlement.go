package mono

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"kredit/internal/settlement"
	"net/http"
	"strings"
)

func (c *Client) Banks(ctx context.Context) ([]settlement.Bank, error) {
	var out envelope[struct {
		Banks []struct {
			Name    string `json:"name"`
			NIPCode string `json:"nip_code"`
		} `json:"banks"`
	}]
	if err := c.request(ctx, http.MethodGet, "/v3/banks/list", nil, &out); err != nil {
		return nil, err
	}
	if !successfulEnvelope(out.Status) {
		return nil, errors.New("bank list is unavailable")
	}
	banks := []settlement.Bank{}
	for _, bank := range out.Data.Banks {
		banks = append(banks, settlement.Bank{Code: bank.NIPCode, Name: bank.Name})
	}
	return banks, settlement.ValidateBanks(banks)
}

func (c *Client) ConnectionIdentity() string {
	mac := hmac.New(sha256.New, []byte(c.secret))
	_, _ = mac.Write([]byte("kredit-settlement-account:" + c.baseURL))
	return hex.EncodeToString(mac.Sum(nil))
}

// CreateDestination implements Mono's documented payout sub-account API. An
// accepted sub-account is not proof of ownership or completed payment settlement.
func (c *Client) CreateDestination(ctx context.Context, in settlement.Input) (settlement.Destination, error) {
	if err := settlement.ValidateInput(in); err != nil {
		return settlement.Destination{}, err
	}
	var out envelope[struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		AccountNumber string `json:"account_number"`
		NIPCode       string `json:"nip_code"`
	}]
	err := c.request(ctx, http.MethodPost, "/v2/payments/payout/sub-account", map[string]string{"nip_code": in.BankCode, "account_number": in.AccountNumber}, &out)
	if err != nil || !successfulEnvelope(out.Status) || out.Data.AccountNumber != in.AccountNumber {
		return settlement.Destination{}, settlement.ErrUnknown
	}
	result := settlement.Destination{ProviderReference: out.Data.ID, BankCode: out.Data.NIPCode, AccountName: strings.TrimSpace(out.Data.Name), AccountLast4: in.AccountNumber[6:]}
	if err = settlement.ValidateDestination(in, result); err != nil {
		return settlement.Destination{}, err
	}
	return result, nil
}
