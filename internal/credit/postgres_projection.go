package credit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"kredit/internal/platform/txcleanup"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// hydrateForTenant reloads the committed aggregate on every read. Returning a
// projection this process loaded earlier would serve a supplier or buyer a
// balance the worker has already changed, and PostgreSQL is authoritative.
// installView never replaces a resident aggregate with an older version.
func (s *PostgresStore) hydrateForTenant(requestID, userID, organizationID string) error {
	view, err := s.loadViewForTenant(requestID, userID, organizationID)
	if err != nil {
		return err
	}
	s.installView(view)
	s.markLoaded(requestID)
	return nil
}

func (s *PostgresStore) loadViewForTenant(requestID, userID, organizationID string) (View, error) {
	if s == nil || s.pool == nil {
		return View{}, errors.New("credit database is not configured")
	}
	if strings.TrimSpace(userID) == "" && strings.TrimSpace(organizationID) == "" {
		return View{}, errors.New("credit tenant context is required before loading an aggregate")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return View{}, err
	}
	defer func() { _ = txcleanup.Rollback(ctx, tx, nil) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, userID, organizationID); err != nil {
		return View{}, err
	}
	var payload []byte
	if err = tx.QueryRow(ctx, `SELECT app.credit_snapshot_by_id($1)`, requestID).Scan(&payload); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return View{}, ErrRequestNotFound
		}
		return View{}, fmt.Errorf("load credit aggregate: %w", err)
	}
	if len(payload) == 0 {
		return View{}, ErrRequestNotFound
	}
	var view View
	if err = json.Unmarshal(payload, &view); err != nil {
		return View{}, fmt.Errorf("decode credit aggregate: %w", err)
	}
	if view.Request.ID != requestID {
		return View{}, errors.New("credit projection identity mismatch")
	}
	return view, tx.Commit(ctx)
}

func (s *PostgresStore) hydrateByObligation(obligationID string) error {
	s.mu.RLock()
	for _, obligation := range s.obligations {
		if obligation.ID == obligationID {
			s.mu.RUnlock()
			return nil
		}
	}
	s.mu.RUnlock()
	return errors.New("credit tenant context is required before loading an obligation")
}

func (s *PostgresStore) hydrateByObligationForTenant(obligationID, userID, organizationID string) error {
	if s == nil || s.pool == nil {
		return errors.New("credit database is not configured")
	}
	if strings.TrimSpace(userID) == "" && strings.TrimSpace(organizationID) == "" {
		return errors.New("credit tenant context is required before loading an obligation")
	}
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(context.Background(), `SELECT set_config('app.current_user_id', $1, true), set_config('app.current_organization_id', $2, true)`, userID, organizationID); err != nil {
		return err
	}
	var id string
	if err := tx.QueryRow(context.Background(), `SELECT credit_request_id FROM app.credit_snapshot_by_obligation($1)`, obligationID).Scan(&id); err != nil {
		return errors.New("obligation not found")
	}
	var payload []byte
	if err := tx.QueryRow(context.Background(), `SELECT app.credit_snapshot_by_id($1)`, id).Scan(&payload); err != nil {
		return ErrRequestNotFound
	}
	var view View
	if err := json.Unmarshal(payload, &view); err != nil {
		return fmt.Errorf("decode credit aggregate: %w", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return err
	}
	s.installView(view)
	s.markLoaded(view.Request.ID)
	return nil
}

func (s *PostgresStore) installView(view View) {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := cloneRequest(view.Request)
	if current := s.requests[request.ID]; current != nil && current.Version > request.Version {
		return
	}
	s.requests[request.ID] = &request
	if view.Agreement.ID != "" {
		agreement := cloneAgreement(view.Agreement)
		s.agreements[agreement.ID] = &agreement
	}
	if view.Acceptance != nil {
		acceptance := *view.Acceptance
		s.acceptances[acceptance.ID] = &acceptance
	}
	if view.Mandate != nil {
		mandate := *view.Mandate
		s.mandateMap[mandate.ID] = &mandate
		if mandate.ProviderID != "" {
			s.mandateMap[mandate.ProviderID] = &mandate
		}
	}
	if view.Release != nil {
		release := *view.Release
		s.releases[release.ID] = &release
	}
	if view.Receipts != nil {
		receipts := make([]*ReceiptConfirmation, 0, len(view.Receipts))
		for _, item := range view.Receipts {
			value := item
			receipts = append(receipts, &value)
		}
		s.receipts[request.ID] = receipts
	}
	if view.Obligation != nil {
		obligation := *view.Obligation
		s.obligations[obligation.ID] = &obligation
	}
}
