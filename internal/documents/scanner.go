package documents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
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
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" || parsed.RawQuery != "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return nil, errors.New("document scanner endpoint is invalid")
	}
	return &WebhookScanner{endpoint: strings.TrimSpace(endpoint), token: strings.TrimSpace(token), client: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
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
	document, err := s.readContext(ctx, id, "", "")
	if err != nil {
		return Document{}, err
	}
	if document.ScanState != ScanPending && document.ScanState != ScanQuarantine {
		return document, nil
	}
	if document.UploadCompletedAt.IsZero() || s.objects == nil {
		return Document{}, errors.New("document upload is not complete")
	}
	// Claim at execution, not discovery: an old queued job must never borrow
	// the attempt identity of a newer worker after the discovery lease expires.
	if s.pool != nil {
		err := s.pool.QueryRow(ctx, `UPDATE app.documents SET scan_attempts=scan_attempts+1,scan_lease_until=now()+interval '5 minutes' WHERE id=$1::uuid AND scan_state=$2 AND scan_attempts=$3 AND upload_completed_at IS NOT NULL AND (scan_lease_until IS NULL OR scan_lease_until<=now()) AND scan_attempts<5 RETURNING scan_attempts,scan_lease_until`, document.ID, string(document.ScanState), document.ScanAttempts).Scan(&document.ScanAttempts, &document.ScanLeaseUntil)
		if errors.Is(err, pgx.ErrNoRows) {
			return Document{}, errors.New("document scan is already claimed or needs review")
		}
		if err != nil {
			return Document{}, err
		}
	} else {
		s.mu.Lock()
		current, exists := s.items[id]
		if !exists || current.ScanState != document.ScanState || current.ScanAttempts != document.ScanAttempts || current.ScanAttempts >= 5 || current.ScanLeaseUntil.After(s.now()) {
			s.mu.Unlock()
			return Document{}, errors.New("document scan is already claimed or changed")
		}
		current.ScanAttempts++
		current.ScanLeaseUntil = s.now().Add(5 * time.Minute)
		s.items[id] = current
		document = current
		s.mu.Unlock()
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	url, err := s.objects.SignedURL(ctx, document.ObjectKey, 5*time.Minute)
	if err != nil {
		return Document{}, err
	}
	state, err := scanner.Scan(ctx, document, url)
	if err != nil {
		return Document{}, err
	}
	return s.completeScan(ctx, document, state)
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
	rows, err := s.pool.Query(ctx, `SELECT id::text FROM app.documents WHERE scan_state='PENDING' AND upload_completed_at IS NOT NULL AND scan_attempts<5 AND (scan_lease_until IS NULL OR scan_lease_until<=now()) ORDER BY created_at LIMIT $1`, limit)
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
