package mono

import (
	"context"
	"encoding/json"
	"errors"
	"kredit/internal/billing"
	"kredit/internal/collections"
	"kredit/internal/ledger"
	"net/http"
	"net/url"
)

func (c *Client) CreateFeeCustomer(ctx context.Context, in billing.FeeCustomer) (string, error) {
	first, last, err := BusinessCustomerNames(in.BusinessName)
	if err != nil {
		return "", err
	}
	return c.CreateCustomer(ctx, CustomerInput{FirstName: first, LastName: last, Email: in.Email, Phone: in.Phone, Address: in.Address, BVN: in.BVN, ConsentVersion: "seller-fees-v1"})
}
func (c *Client) FeeCustomerIdentity(ctx context.Context, id string) (string, error) {
	return c.CustomerIdentity(ctx, id)
}
func (c *Client) CreateFeeAuthorization(ctx context.Context, in billing.FeeAuthorization) (billing.FeeAuthorization, error) {
	if c.initiationDisabled || !validReference(in.Customer) || !validReference(in.Reference) || in.Ceiling < 20000 || in.Ceiling > 2500000000 || !in.EndsAt.After(in.StartsAt) {
		return in, errors.New("valid separate fee authorization required")
	}
	body := map[string]any{"amount": in.Ceiling, "type": "recurring-debit", "method": "mandate", "mandate_type": "emandate", "debit_type": "variable", "description": "Kredit seller platform fees", "reference": in.Reference, "redirect_url": c.redirectURL, "customer": map[string]string{"id": in.Customer}, "start_date": in.StartsAt.Format("2006-01-02"), "end_date": in.EndsAt.Format("2006-01-02")}
	var out envelope[mandateData]
	if err := c.request(ctx, http.MethodPost, "/v2/payments/initiate", body, &out); err != nil {
		return in, err
	}
	id := out.Data.MandateID
	if id == "" {
		id = out.Data.ID
	}
	if !successfulEnvelope(out.Status) || !validReference(id) || (out.Data.Reference != "" && out.Data.Reference != in.Reference) || validateHostedAuthorizationURL(out.Data.MonoURL) != nil {
		return in, errors.New("fee authorization needs reconciliation")
	}
	in.ID, in.URL = id, out.Data.MonoURL
	return in, nil
}
func (c *Client) ReadFeeAuthorization(ctx context.Context, id string) (billing.FeeAuthorization, error) {
	var out envelope[struct {
		mandateData
		Customer json.RawMessage `json:"customer"`
	}]
	if !validReference(id) {
		return billing.FeeAuthorization{}, errors.New("invalid mandate")
	}
	if err := c.request(ctx, http.MethodGet, "/v3/payments/mandates/"+url.PathEscape(id), nil, &out); err != nil {
		return billing.FeeAuthorization{}, err
	}
	d := out.Data
	var customer string
	if json.Unmarshal(d.Customer, &customer) != nil {
		var v struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(d.Customer, &v)
		customer = v.ID
	}
	if !successfulEnvelope(out.Status) || d.ID != id || d.MandateType != "emandate" || d.DebitType != "variable" || d.Amount <= 0 || !validReference(customer) {
		return billing.FeeAuthorization{}, errors.New("separate variable fee mandate was not confirmed")
	}
	start, err := parseMandateDate(d.StartRaw)
	if err != nil {
		return billing.FeeAuthorization{}, err
	}
	end, err := parseMandateDate(d.EndRaw)
	if err != nil || end.IsZero() {
		return billing.FeeAuthorization{}, errors.New("mandate expiry is required")
	}
	return billing.FeeAuthorization{ID: id, Reference: d.Reference, Customer: customer, Ceiling: d.Amount, StartsAt: start, EndsAt: end, Ready: d.Status == "approved" && d.Approved && d.Ready && !c.now().Before(start) && c.now().Before(end)}, nil
}
func feeResult(out collections.Response) billing.FeeDebitResult {
	state := "pending"
	switch out.State {
	case collections.ProviderSucceeded:
		state = "succeeded"
	case collections.ProviderFailed:
		state = "failed"
	case collections.ProviderReversed:
		state = "reversed"
	}
	return billing.FeeDebitResult{State: state, Amount: int64(out.SucceededAmountKobo)}
}
func (c *Client) SubmitFeeDebit(ctx context.Context, in billing.FeeDebitRequest) (billing.FeeDebitResult, error) {
	if c.initiationDisabled || !validReference(in.Mandate) || !validReference(in.Reference) || in.Amount < 20000 || in.Amount > 2500000000 {
		return billing.FeeDebitResult{}, errors.New("valid fee debit and active account required")
	}
	var out envelope[debitData]
	err := c.request(ctx, http.MethodPost, "/v3/payments/mandates/"+url.PathEscape(in.Mandate)+"/debit", map[string]any{"amount": in.Amount, "reference": in.Reference, "narration": "Kredit seller fees Ref:" + in.Reference, "fee_bearer": "business"}, &out)
	if err != nil {
		return billing.FeeDebitResult{}, err
	}
	if !successfulEnvelope(out.Status) || (out.Data.Reference != "" && out.Data.Reference != in.Reference) || (out.Data.Mandate != "" && out.Data.Mandate != in.Mandate) {
		return billing.FeeDebitResult{}, errors.New("fee debit response requires reconciliation")
	}
	return feeResult(debitResponse(out.Data, in.Mandate, in.Reference, ledger.Money(in.Amount), c.live)), nil
}
func (c *Client) ReadFeeDebit(ctx context.Context, in billing.FeeDebitRequest) (billing.FeeDebitResult, error) {
	out, err := c.GetByReference(ctx, collections.Request{MandateReference: in.Mandate, ExternalReference: in.Reference, AmountKobo: ledger.Money(in.Amount), Currency: "NGN"})
	return feeResult(out), err
}

var _ billing.FeeProvider = (*Client)(nil)

func (c *Client) ValidateFeeCustomer(in billing.FeeCustomer) error {
	if c.initiationDisabled {
		return errors.New("activate the original account before starting fee setup")
	}
	first, last, err := BusinessCustomerNames(in.BusinessName)
	if err != nil {
		return err
	}
	_, err = ValidateCustomerInput(CustomerInput{FirstName: first, LastName: last, Email: in.Email, Phone: in.Phone, Address: in.Address, BVN: in.BVN, ConsentVersion: "seller-fees-v1"})
	return err
}
