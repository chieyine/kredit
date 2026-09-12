package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const MesajEndpoint = "https://api.mesaj.cloud:25274/client/sms/send/bulk"

type MesajProvider struct {
	endpoint, token, sender string
	client                  *http.Client
}

func NewMesajProvider(endpoint, token, sender string) (*MesajProvider, error) {
	if endpoint != MesajEndpoint || strings.TrimSpace(token) == "" || strings.ContainsAny(token, "\r\n") {
		return nil, errors.New("mesaj requires its SMS endpoint, bearer token and approved sender ID")
	}
	if !validMesajSender(sender) {
		return nil, errors.New("mesaj bulk SMS requires an approved alphanumeric sender ID of at most 11 characters")
	}
	return &MesajProvider{endpoint, token, sender, &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (p *MesajProvider) Channel() string            { return ChannelSMS }
func (p *MesajProvider) SubmissionIdentity() string { return p.endpoint + ":" + p.sender }
func (p *MesajProvider) Send(ctx context.Context, m Message) (string, error) {
	if m.Channel != ChannelSMS || m.EventID == "" || strings.TrimSpace(m.Body) == "" || !validMessagingPhone(m.Destination) {
		return "", permanentDeliveryError{400}
	}
	messageType := "TRANSACTIONAL"
	if m.Template == "ProductUpdate" {
		messageType = "PROMOTIONAL"
	}
	payload := map[string]any{"data": map[string]any{"sender_id": p.sender, "message": m.Body, "type": messageType, "recipients": []string{strings.TrimPrefix(m.Destination, "+")}}}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	var result struct {
		Status    string `json:"status"`
		MessageID string `json:"message_id"`
		Accepted  int    `json:"accepted"`
		Rejected  int    `json:"rejected"`
	}
	if err = p.request(req, &result); err != nil {
		return "", err
	}
	if result.Status != "success" || result.Accepted != 1 || result.Rejected != 0 || strings.TrimSpace(result.MessageID) == "" || len(result.MessageID) > 512 {
		return "", ErrSubmissionUnknown
	}
	return result.MessageID, nil
}

// The published bulk contract has no authenticated receipt endpoint. A send
// acknowledgement alone must never create delivery-dependent financial evidence.
func (p *MesajProvider) DeliveryStatus(context.Context, string) (DeliveryStatus, error) {
	return DeliveryStatus{}, ErrDeliveryLookupUnsupported
}
func (p *MesajProvider) request(req *http.Request, result any) error {
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")
	response, err := p.client.Do(req)
	if err != nil {
		return errors.New("mesaj request could not be confirmed")
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, (256<<10)+1))
	if err != nil || len(body) > 256<<10 {
		return errors.New("mesaj response could not be read")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if response.StatusCode >= 400 && response.StatusCode < 500 && response.StatusCode != 408 {
			return permanentDeliveryError{response.StatusCode}
		}
		return fmt.Errorf("mesaj response unconfirmed (status %d)", response.StatusCode)
	}
	if json.Unmarshal(body, result) != nil {
		return errors.New("mesaj returned an invalid response")
	}
	return nil
}
func validMessagingPhone(value string) bool {
	value = strings.TrimPrefix(value, "+")
	if len(value) < 8 || len(value) > 15 || value[0] == '0' {
		return false
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func validMesajSender(sender string) bool {
	if len(sender) == 0 || len(sender) > 11 {
		return false
	}
	for _, c := range sender {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}
