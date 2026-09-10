package mono

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var bvnPattern = regexp.MustCompile(`^[0-9]{11}$`)

// CustomerInput is transient: callers must never persist or log this value.
type CustomerInput struct {
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	Address        string `json:"address"`
	BVN            string `json:"bvn"`
	ConsentVersion string `json:"consent_version"`
}

func (c *Client) CreateCustomer(ctx context.Context, in CustomerInput) (string, error) {
	if c.initiationDisabled {
		return "", errors.New("new Mono customers are disabled")
	}
	var err error
	in, err = ValidateCustomerInput(in)
	if err != nil {
		return "", err
	}
	var out envelope[struct {
		ID string `json:"id"`
	}]
	body := map[string]any{"first_name": in.FirstName, "last_name": in.LastName, "email": in.Email, "phone": in.Phone, "address": in.Address, "identity": map[string]string{"type": "bvn", "number": in.BVN}}
	if err := c.request(ctx, http.MethodPost, "/v2/customers", body, &out); err != nil {
		return "", err
	}
	if !successfulEnvelope(out.Status) || !validReference(out.Data.ID) {
		return "", errors.New("mono customer registration was not confirmed")
	}
	return out.Data.ID, nil
}

func ValidateCustomerInput(in CustomerInput) (CustomerInput, error) {
	in.FirstName, in.LastName = strings.TrimSpace(in.FirstName), strings.TrimSpace(in.LastName)
	in.Email, in.Phone = strings.TrimSpace(in.Email), strings.TrimSpace(in.Phone)
	in.Address, in.ConsentVersion = strings.TrimSpace(in.Address), strings.TrimSpace(in.ConsentVersion)
	if !bvnPattern.MatchString(in.BVN) || in.FirstName == "" || in.LastName == "" || in.Email == "" || in.Phone == "" || in.Address == "" || len(in.Address) > 100 || in.ConsentVersion == "" {
		return CustomerInput{}, errors.New("complete customer details and verification consent are required")
	}

	return in, nil
}

// CustomerIdentity reads only enough provider evidence to reconcile a known
// customer reference. Raw identity values remain transient and are never logged.
// Contract: https://docs.mono.co/api/customer/retrieve-a-customer
func (c *Client) CustomerIdentity(ctx context.Context, reference string) (string, error) {
	if !validReference(reference) {
		return "", errors.New("invalid customer reference")
	}
	var out envelope[struct {
		ID                 string `json:"id"`
		BVN                string `json:"bvn"`
		IdentificationNo   string `json:"identification_no"`
		IdentificationType string `json:"identification_type"`
	}]
	if err := c.request(ctx, http.MethodGet, "/v2/customers/"+url.PathEscape(reference), nil, &out); err != nil {
		return "", err
	}
	if !successfulEnvelope(out.Status) || out.Data.ID != reference {
		return "", errors.New("customer reference could not be verified")
	}
	value := out.Data.BVN
	if value == "" && strings.EqualFold(out.Data.IdentificationType, "bvn") {
		value = out.Data.IdentificationNo
	}
	if !bvnPattern.MatchString(value) {
		return "", errors.New("provider did not return sufficient identity evidence")
	}
	return value, nil
}
