package documents

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

type Scanner interface {
	Scan(context.Context, Document, string) (ScanState, error)
}

type CleanDevelopmentScanner struct{}

func (CleanDevelopmentScanner) Scan(context.Context, Document, string) (ScanState, error) {
	return ScanClean, nil
}

type WebhookScanner struct {
	endpoint string
	token    string
	client   *http.Client
}

func NewWebhookScanner(endpoint, token string) (*WebhookScanner, error) {
	if strings.TrimSpace(endpoint) == "" || strings.TrimSpace(token) == "" {
		return nil, errors.New("document scanner endpoint and token are required")
	}
	return &WebhookScanner{endpoint: strings.TrimSpace(endpoint), token: strings.TrimSpace(token), client: &http.Client{Timeout: 30 * time.Second}}, nil
}

func (s *WebhookScanner) Scan(ctx context.Context, document Document, downloadURL string) (ScanState, error) {
	if strings.TrimSpace(downloadURL) == "" {
		return "", errors.New("scanner download URL is required")
	}
	payload, err := json.Marshal(map[string]any{"document_id": document.ID, "download_url": downloadURL, "sha256": document.SHA256, "content_type": document.ContentType, "size_bytes": document.SizeBytes})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+s.token)
	request.Header.Set("Idempotency-Key", "document-scan:"+document.ID+":"+document.SHA256)
	response, err := s.client.Do(request)
	if err != nil {
		return "", err
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("document scanner returned status %d", response.StatusCode)
	}
	var result struct {
		State ScanState `json:"state"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", errors.New("document scanner returned an invalid response")
	}
	if result.State != ScanClean && result.State != ScanRejected && result.State != ScanQuarantine {
		return "", errors.New("document scanner returned an invalid state")
	}
	return result.State, nil
}

func (s *Store) Scan(ctx context.Context, id string, scanner Scanner) (Document, error) {
	if scanner == nil {
		return Document{}, errors.New("document scanner is not configured")
	}
	document, ok := s.Get(id)
	if !ok {
		return Document{}, errors.New("document not found")
	}
	if document.ScanState != ScanPending && document.ScanState != ScanQuarantine {
		return document, nil
	}
	url, err := s.objects.SignedURL(ctx, document.ObjectKey, 5*time.Minute)
	if err != nil {
		return Document{}, err
	}
	state, err := scanner.Scan(ctx, document, url)
	if err != nil {
		return Document{}, err
	}
	return s.CompleteScan(id, state)
}

func (s *Store) PendingScanIDs(ctx context.Context, limit int) ([]string, error) {
	if s.pool == nil {
		return nil, errors.New("document database is not configured")
	}
	if limit < 1 || limit > 500 {
		return nil, errors.New("document scan limit must be between 1 and 500")
	}
	if _, err := s.pool.Exec(ctx, `UPDATE app.documents SET scan_state='QUARANTINED',scanned_at=now(),scan_lease_until=NULL WHERE scan_state='PENDING' AND ((upload_completed_at IS NULL AND upload_expires_at<=now()) OR (scan_attempts>=5 AND (scan_lease_until IS NULL OR scan_lease_until<=now())))`); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `WITH claimed AS (SELECT id FROM app.documents WHERE scan_state='PENDING' AND upload_completed_at IS NOT NULL AND scan_attempts<5 AND (scan_lease_until IS NULL OR scan_lease_until<=now()) ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT $1) UPDATE app.documents d SET scan_attempts=d.scan_attempts+1,scan_lease_until=now()+interval '5 minutes' FROM claimed WHERE d.id=claimed.id RETURNING d.id::text`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
