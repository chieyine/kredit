package mono

import (
	"context"
	"errors"
	"net/http"
	"regexp"
)

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
	if !regexp.MustCompile(`^[0-9]{11}$`).MatchString(in.BVN) || in.FirstName == "" || in.LastName == "" || in.Email == "" || in.Phone == "" || in.Address == "" || len(in.Address) > 100 || in.ConsentVersion == "" {
		return "", errors.New("complete customer details and verification consent are required")
	}
	var out envelope[struct {
		ID string `json:"id"`
	}]
	body := map[string]any{"first_name": in.FirstName, "last_name": in.LastName, "email": in.Email, "phone": in.Phone, "address": in.Address, "identity": map[string]string{"type": "bvn", "number": in.BVN}}
	if err := c.request(ctx, http.MethodPost, "/v2/customers", body, &out); err != nil {
		return "", err
	}
	if out.Status != "successful" || out.Data.ID == "" {
		return "", errors.New("mono customer registration was not confirmed")
	}
	return out.Data.ID, nil
}
