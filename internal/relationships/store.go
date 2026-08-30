package relationships

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Consent struct {
	ID            string    `json:"id"`
	BuyerUserID   string    `json:"buyer_user_id"`
	SupplierOrgID string    `json:"supplier_organization_id"`
	ConsentType   string    `json:"consent_type"`
	Version       string    `json:"version"`
	EvidenceHash  string    `json:"evidence_hash"`
	Granted       bool      `json:"granted"`
	CreatedAt     time.Time `json:"created_at"`
}

type Store struct {
	mu       sync.RWMutex
	consents []Consent
	counter  int64
}

// Service is the relationship-consent boundary used by HTTP handlers. The
// development store and PostgreSQL implementation intentionally share the
// same contract so production cannot accidentally depend on process memory.
type Service interface {
	Record(string, string, string, string, string, bool) (Consent, error)
	List(string) []Consent
}

var _ Service = (*Store)(nil)

func NewStore() *Store { return &Store{} }

func (s *Store) Record(buyerUserID, supplierOrgID, consentType, version, evidenceHash string, granted bool) (Consent, error) {
	if strings.TrimSpace(buyerUserID) == "" || strings.TrimSpace(supplierOrgID) == "" || strings.TrimSpace(consentType) == "" || strings.TrimSpace(version) == "" || strings.TrimSpace(evidenceHash) == "" {
		return Consent{}, errors.New("consent identity, version, and evidence are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	item := Consent{ID: "consent-" + time.Now().UTC().Format("20060102150405.000000000"), BuyerUserID: buyerUserID, SupplierOrgID: supplierOrgID, ConsentType: consentType, Version: version, EvidenceHash: evidenceHash, Granted: granted, CreatedAt: time.Now().UTC()}
	s.consents = append(s.consents, item)
	return item, nil
}

func (s *Store) List(buyerUserID string) []Consent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Consent, 0)
	for _, item := range s.consents {
		if item.BuyerUserID == buyerUserID {
			result = append(result, item)
		}
	}
	return result
}

// PostgresStore persists consent evidence in the tenant-scoped relationship
// table. UUID validation is deliberate: production identifiers must be
// database identifiers, while the in-memory implementation remains available
// for deterministic development tests.
type PostgresStore struct {
	pool *pgxpool.Pool
}

var _ Service = (*PostgresStore)(nil)

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Record(buyerUserID, supplierOrgID, consentType, version, evidenceHash string, granted bool) (Consent, error) {
	if strings.TrimSpace(buyerUserID) == "" || strings.TrimSpace(supplierOrgID) == "" || strings.TrimSpace(consentType) == "" || strings.TrimSpace(version) == "" || strings.TrimSpace(evidenceHash) == "" {
		return Consent{}, errors.New("consent identity, version, and evidence are required")
	}
	if s == nil || s.pool == nil {
		return Consent{}, errors.New("relationship database is not configured")
	}
	ctx := context.Background()
	var item Consent
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Consent{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_user_id', $1, true), set_config('app.current_organization_id', $2, true)`, buyerUserID, supplierOrgID); err != nil {
		return Consent{}, err
	}
	var related bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM app.trade_relationships relationship
			JOIN app.businesses business ON business.id = relationship.buyer_business_id
			WHERE relationship.supplier_organization_id = $2::uuid
			  AND business.owner_user_id = $1::uuid
			  AND relationship.status IN ('invited', 'active')
		)`, buyerUserID, supplierOrgID).Scan(&related); err != nil {
		return Consent{}, err
	}
	if !related {
		return Consent{}, errors.New("buyer has no active relationship with this supplier")
	}
	if err := tx.QueryRow(ctx, `
			INSERT INTO app.relationship_consents
				(buyer_user_id, supplier_organization_id, consent_type, version, evidence_hash, granted)
			VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
			RETURNING id::text, buyer_user_id::text, supplier_organization_id::text,
				consent_type, version, evidence_hash, granted, created_at`,
		buyerUserID, supplierOrgID, consentType, version, evidenceHash, granted,
	).Scan(&item.ID, &item.BuyerUserID, &item.SupplierOrgID, &item.ConsentType, &item.Version, &item.EvidenceHash, &item.Granted, &item.CreatedAt); err != nil {
		return Consent{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Consent{}, err
	}
	return item, nil
}

func (s *PostgresStore) List(buyerUserID string) []Consent {
	if s == nil || s.pool == nil || strings.TrimSpace(buyerUserID) == "" {
		return []Consent{}
	}
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return []Consent{}
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_user_id', $1, true)`, buyerUserID); err != nil {
		return []Consent{}
	}
	rows, err := tx.Query(ctx, `
		SELECT id::text, buyer_user_id::text, supplier_organization_id::text,
			consent_type, version, evidence_hash, granted, created_at
		FROM app.relationship_consents
		WHERE buyer_user_id = $1::uuid
		ORDER BY created_at DESC`, buyerUserID)
	if err != nil {
		return []Consent{}
	}
	defer rows.Close()
	result := make([]Consent, 0)
	for rows.Next() {
		var item Consent
		if err := rows.Scan(&item.ID, &item.BuyerUserID, &item.SupplierOrgID, &item.ConsentType, &item.Version, &item.EvidenceHash, &item.Granted, &item.CreatedAt); err != nil {
			return []Consent{}
		}
		result = append(result, item)
	}
	if rows.Err() != nil {
		return []Consent{}
	}
	return result
}
