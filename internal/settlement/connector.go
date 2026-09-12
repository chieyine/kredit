package settlement

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Connector is Kredit's versioned adapter contract, not a third-party API URL.
// A replacement provider translates this contract in its own adapter.
type Connector struct {
	name, endpoint, token string
	client                *http.Client
}

func NewConnector(name, endpoint, token string) (*Connector, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || name == "" || len(token) < 32 || strings.ContainsAny(token, "\r\n") {
		return nil, errors.New("a named HTTPS settlement connector and access token are required")
	}
	return &Connector{name: name, endpoint: strings.TrimRight(endpoint, "/"), token: token, client: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Connector) Name() string { return c.name }
func (c *Connector) Banks(ctx context.Context) ([]Bank, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/v1/banks", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil || res.StatusCode != 200 {
		return nil, errors.New("bank list is unavailable")
	}
	var out struct {
		Banks []Bank `json:"banks"`
	}
	if json.Unmarshal(raw, &out) != nil {
		return nil, errors.New("bank list is invalid")
	}
	return out.Banks, ValidateBanks(out.Banks)
}
func (c *Connector) ConnectionIdentity() string {
	mac := hmac.New(sha256.New, []byte(c.token))
	_, _ = mac.Write([]byte(c.name + ":" + c.endpoint))
	return hex.EncodeToString(mac.Sum(nil))
}
func (c *Connector) CreateDestination(ctx context.Context, in Input) (Destination, error) {
	if err := ValidateInput(in); err != nil {
		return Destination{}, err
	}
	body, _ := json.Marshal(in)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/v1/destinations", bytes.NewReader(body))
	if err != nil {
		return Destination{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", in.Reference)
	res, err := c.client.Do(req)
	if err != nil {
		return Destination{}, ErrUnknown
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 65537))
	if err != nil || len(raw) > 65536 || res.StatusCode < 200 || res.StatusCode >= 300 {
		return Destination{}, ErrUnknown
	}
	var out struct {
		Reference      string      `json:"reference"`
		OrganizationID string      `json:"organization_id"`
		Destination    Destination `json:"destination"`
	}
	if json.Unmarshal(raw, &out) != nil || out.Reference != in.Reference || out.OrganizationID != in.OrganizationID {
		return Destination{}, ErrUnknown
	}
	if err = ValidateDestination(in, out.Destination); err != nil {
		return Destination{}, err
	}
	return out.Destination, nil
}
