package buyers

import (
	"context"
	"crypto/sha256"
	"errors"
	"sync"
	"time"

	"kredit/internal/identifier"
	"kredit/internal/ledger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTermsBatchNotFound = errors.New("terms import batch not found")
	ErrTermsInvalidRow    = errors.New("terms import row is invalid")
	ErrTermsDualControl   = errors.New("maker-checker dual control required: uploader cannot approve terms import batch")
	ErrTermsAlreadyClosed = errors.New("terms import batch is already decided or closed")
)

type TermsImportRowInput struct {
	CustomerName            string       `json:"customer_name"`
	ContactEmail            string       `json:"contact_email"`
	ContactPhone            string       `json:"contact_phone"`
	ProposedCreditLimitKobo ledger.Money `json:"proposed_credit_limit_kobo"`
	ProposedGraceHours      int          `json:"proposed_grace_hours"`
	OpeningBalanceKobo      ledger.Money `json:"opening_balance_kobo"`
	OpeningBalanceReference string       `json:"opening_balance_reference"`
}

type TermsImportRow struct {
	ID                      string       `json:"id"`
	BatchID                 string       `json:"batch_id"`
	RowNumber               int          `json:"row_number"`
	CustomerName            string       `json:"customer_name"`
	ContactEmail            string       `json:"contact_email"`
	ContactPhone            string       `json:"contact_phone"`
	ProposedCreditLimitKobo ledger.Money `json:"proposed_credit_limit_kobo"`
	ProposedGraceHours      int          `json:"proposed_grace_hours"`
	OpeningBalanceKobo      ledger.Money `json:"opening_balance_kobo"`
	OpeningBalanceReference string       `json:"opening_balance_reference"`
	ValidationStatus        string       `json:"validation_status"`
	ValidationError         string       `json:"validation_error"`
	InvitationID            string       `json:"invitation_id,omitempty"`
	CreatedAt               time.Time    `json:"created_at"`
}

type TermsImportBatch struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	SourceHash     string     `json:"source_hash"`
	TotalRows      int        `json:"total_rows"`
	ValidRows      int        `json:"valid_rows"`
	InvalidRows    int        `json:"invalid_rows"`
	State          string     `json:"state"`
	UploadedBy     string     `json:"uploaded_by"`
	ApprovedBy     string     `json:"approved_by,omitempty"`
	CancelledBy    string     `json:"cancelled_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	ApprovedAt     *time.Time `json:"approved_at,omitempty"`
	CancelledAt    *time.Time `json:"cancelled_at,omitempty"`
}

type TermsImportService interface {
	StageTermsBatch(ctx context.Context, userID, orgID, sourceHash string, rows []TermsImportRowInput) (TermsImportBatch, error)
	ReviewTermsBatch(ctx context.Context, reviewerID, orgID, batchID, decision string) (TermsImportBatch, error)
	ListTermsBatches(ctx context.Context, userID, orgID string) ([]TermsImportBatch, error)
	GetTermsBatch(ctx context.Context, userID, orgID, batchID string) (TermsImportBatch, []TermsImportRow, error)
}

type MemoryTermsImportStore struct {
	mu      sync.RWMutex
	batches map[string]*TermsImportBatch
	rows    map[string][]TermsImportRow
	ledger  ledger.Service
}

func NewMemoryTermsImportStore(l ...ledger.Service) *MemoryTermsImportStore {
	var led ledger.Service
	if len(l) > 0 {
		led = l[0]
	}
	return &MemoryTermsImportStore{
		batches: make(map[string]*TermsImportBatch),
		rows:    make(map[string][]TermsImportRow),
		ledger:  led,
	}
}

func (m *MemoryTermsImportStore) StageTermsBatch(ctx context.Context, userID, orgID, sourceHash string, inputRows []TermsImportRowInput) (TermsImportBatch, error) {
	if len(inputRows) == 0 || len(inputRows) > 500 {
		return TermsImportBatch{}, errors.New("batch row count must be between 1 and 500")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	batchID := identifier.New()
	batch := &TermsImportBatch{
		ID:             batchID,
		OrganizationID: orgID,
		SourceHash:     sourceHash,
		TotalRows:      len(inputRows),
		State:          "staged",
		UploadedBy:     userID,
		CreatedAt:      time.Now().UTC(),
	}

	validCount := 0
	invalidCount := 0
	storedRows := make([]TermsImportRow, len(inputRows))

	for i, r := range inputRows {
		status := "valid"
		errMsg := ""
		if r.CustomerName == "" {
			status = "invalid"
			errMsg = "Customer name is required"
			invalidCount++
		} else if r.ProposedCreditLimitKobo < 0 || r.OpeningBalanceKobo < 0 {
			status = "invalid"
			errMsg = "Financial amounts cannot be negative"
			invalidCount++
		} else {
			validCount++
		}

		storedRows[i] = TermsImportRow{
			ID:                      identifier.New(),
			BatchID:                 batchID,
			RowNumber:               i + 1,
			CustomerName:            r.CustomerName,
			ContactEmail:            r.ContactEmail,
			ContactPhone:            r.ContactPhone,
			ProposedCreditLimitKobo: r.ProposedCreditLimitKobo,
			ProposedGraceHours:      r.ProposedGraceHours,
			OpeningBalanceKobo:      r.OpeningBalanceKobo,
			OpeningBalanceReference: r.OpeningBalanceReference,
			ValidationStatus:        status,
			ValidationError:         errMsg,
			CreatedAt:               time.Now().UTC(),
		}
	}

	batch.ValidRows = validCount
	batch.InvalidRows = invalidCount
	m.batches[batchID] = batch
	m.rows[batchID] = storedRows

	return *batch, nil
}

func (m *MemoryTermsImportStore) ReviewTermsBatch(ctx context.Context, reviewerID, orgID, batchID, decision string) (TermsImportBatch, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	batch, ok := m.batches[batchID]
	if !ok || batch.OrganizationID != orgID {
		return TermsImportBatch{}, ErrTermsBatchNotFound
	}
	if batch.UploadedBy == reviewerID {
		return TermsImportBatch{}, ErrTermsDualControl
	}
	if batch.State != "staged" && batch.State != "reviewing" {
		return TermsImportBatch{}, ErrTermsAlreadyClosed
	}

	now := time.Now().UTC()
	if decision == "approved" {
		batch.State = "approved"
		batch.ApprovedBy = reviewerID
		batch.ApprovedAt = &now
		for i := range m.rows[batchID] {
			if m.rows[batchID][i].ValidationStatus == "valid" {
				m.rows[batchID][i].ValidationStatus = "applied"
				if m.rows[batchID][i].OpeningBalanceKobo > 0 && m.ledger != nil {
					_, _ = m.ledger.PostActivation(m.rows[batchID][i].ID, m.rows[batchID][i].OpeningBalanceKobo, now, "opening-balance:"+m.rows[batchID][i].ID)
				}
			}
		}
	} else if decision == "cancelled" {
		batch.State = "cancelled"
		batch.CancelledBy = reviewerID
		batch.CancelledAt = &now
	} else {
		return TermsImportBatch{}, errors.New("decision must be approved or cancelled")
	}

	return *batch, nil
}

func (m *MemoryTermsImportStore) ListTermsBatches(ctx context.Context, userID, orgID string) ([]TermsImportBatch, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []TermsImportBatch
	for _, b := range m.batches {
		if b.OrganizationID == orgID {
			out = append(out, *b)
		}
	}
	return out, nil
}

func (m *MemoryTermsImportStore) GetTermsBatch(ctx context.Context, userID, orgID, batchID string) (TermsImportBatch, []TermsImportRow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	batch, ok := m.batches[batchID]
	if !ok || batch.OrganizationID != orgID {
		return TermsImportBatch{}, nil, ErrTermsBatchNotFound
	}
	rows := m.rows[batchID]
	outRows := make([]TermsImportRow, len(rows))
	copy(outRows, rows)
	return *batch, outRows, nil
}

// PostgresTermsImportStore implements TermsImportService against PostgreSQL
type PostgresTermsImportStore struct {
	pool   *pgxpool.Pool
	ledger ledger.Service
}

func NewPostgresTermsImportStore(pool *pgxpool.Pool, l ...ledger.Service) *PostgresTermsImportStore {
	var led ledger.Service
	if len(l) > 0 {
		led = l[0]
	}
	return &PostgresTermsImportStore{pool: pool, ledger: led}
}

func (p *PostgresTermsImportStore) SetLedger(l ledger.Service) {
	p.ledger = l
}

func (p *PostgresTermsImportStore) StageTermsBatch(ctx context.Context, userID, orgID, sourceHash string, inputRows []TermsImportRowInput) (TermsImportBatch, error) {
	if p.pool == nil {
		return TermsImportBatch{}, errors.New("database pool unavailable")
	}
	if len(inputRows) == 0 || len(inputRows) > 500 {
		return TermsImportBatch{}, errors.New("batch row count must be between 1 and 500")
	}

	hash := sha256.Sum256([]byte(sourceHash))

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return TermsImportBatch{}, err
	}
	defer tx.Rollback(ctx)

	var batch TermsImportBatch
	validCount := 0
	invalidCount := 0

	for _, r := range inputRows {
		if r.CustomerName == "" || r.ProposedCreditLimitKobo < 0 || r.OpeningBalanceKobo < 0 {
			invalidCount++
		} else {
			validCount++
		}
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO app.partner_terms_import_batches (
			organization_id, source_hash, payload_hash, payload_ciphertext, total_rows, valid_rows, invalid_rows, state, uploaded_by
		) VALUES (
			$1::uuid, $2, $3, $4, $5, $6, $7, 'staged', $8::uuid
		) RETURNING id::text, organization_id::text, source_hash, total_rows, valid_rows, invalid_rows, state, uploaded_by::text, created_at
	`, orgID, sourceHash, hash[:], []byte(sourceHash), len(inputRows), validCount, invalidCount, userID).
		Scan(&batch.ID, &batch.OrganizationID, &batch.SourceHash, &batch.TotalRows, &batch.ValidRows, &batch.InvalidRows, &batch.State, &batch.UploadedBy, &batch.CreatedAt)
	if err != nil {
		return TermsImportBatch{}, err
	}

	for i, r := range inputRows {
		status := "valid"
		errMsg := ""
		if r.CustomerName == "" {
			status = "invalid"
			errMsg = "Customer name is required"
		} else if r.ProposedCreditLimitKobo < 0 || r.OpeningBalanceKobo < 0 {
			status = "invalid"
			errMsg = "Financial amounts cannot be negative"
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO app.partner_terms_import_rows (
				batch_id, row_number, customer_name, contact_email, contact_phone, proposed_credit_limit_kobo, proposed_grace_hours, opening_balance_kobo, opening_balance_reference, validation_status, validation_error
			) VALUES (
				$1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
			)
		`, batch.ID, i+1, r.CustomerName, r.ContactEmail, r.ContactPhone, int64(r.ProposedCreditLimitKobo), r.ProposedGraceHours, int64(r.OpeningBalanceKobo), r.OpeningBalanceReference, status, errMsg)
		if err != nil {
			return TermsImportBatch{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return TermsImportBatch{}, err
	}
	return batch, nil
}

func (p *PostgresTermsImportStore) ReviewTermsBatch(ctx context.Context, reviewerID, orgID, batchID, decision string) (TermsImportBatch, error) {
	if p.pool == nil {
		return TermsImportBatch{}, errors.New("database pool unavailable")
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return TermsImportBatch{}, err
	}
	defer tx.Rollback(ctx)

	if orgID != "" || reviewerID != "" {
		if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, reviewerID, orgID); err != nil {
			return TermsImportBatch{}, err
		}
	}

	var uploadedBy, state string
	err = tx.QueryRow(ctx, `
		SELECT uploaded_by::text, state FROM app.partner_terms_import_batches
		WHERE id = $1::uuid AND organization_id = $2::uuid FOR UPDATE
	`, batchID, orgID).Scan(&uploadedBy, &state)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TermsImportBatch{}, ErrTermsBatchNotFound
		}
		return TermsImportBatch{}, err
	}
	if uploadedBy == reviewerID {
		return TermsImportBatch{}, ErrTermsDualControl
	}
	if state != "staged" && state != "reviewing" {
		return TermsImportBatch{}, ErrTermsAlreadyClosed
	}

	var batch TermsImportBatch
	now := time.Now().UTC()
	if decision == "approved" {
		err = tx.QueryRow(ctx, `
			UPDATE app.partner_terms_import_batches
			SET state = 'approved', approved_by = $3::uuid, approved_at = $4
			WHERE id = $1::uuid AND organization_id = $2::uuid
			RETURNING id::text, organization_id::text, source_hash, total_rows, valid_rows, invalid_rows, state, uploaded_by::text, COALESCE(approved_by::text, ''), created_at, approved_at
		`, batchID, orgID, reviewerID, now).
			Scan(&batch.ID, &batch.OrganizationID, &batch.SourceHash, &batch.TotalRows, &batch.ValidRows, &batch.InvalidRows, &batch.State, &batch.UploadedBy, &batch.ApprovedBy, &batch.CreatedAt, &batch.ApprovedAt)
		if err != nil {
			return TermsImportBatch{}, err
		}

		rows, err := tx.Query(ctx, `
			SELECT id::text, opening_balance_kobo
			FROM app.partner_terms_import_rows
			WHERE batch_id = $1::uuid AND validation_status = 'valid'
		`, batchID)
		if err != nil {
			return TermsImportBatch{}, err
		}
		defer rows.Close()

		type rowBal struct {
			id  string
			bal int64
		}
		var validRows []rowBal
		for rows.Next() {
			var rb rowBal
			if err := rows.Scan(&rb.id, &rb.bal); err == nil {
				validRows = append(validRows, rb)
			}
		}
		rows.Close()

		_, err = tx.Exec(ctx, `
			UPDATE app.partner_terms_import_rows
			SET validation_status = 'applied'
			WHERE batch_id = $1::uuid AND validation_status = 'valid'
		`, batchID)
		if err != nil {
			return TermsImportBatch{}, err
		}

		for _, vr := range validRows {
			if vr.bal > 0 && p.ledger != nil {
				_, err = p.ledger.PostActivation(vr.id, ledger.Money(vr.bal), now, "opening-balance:"+vr.id)
				if err != nil {
					return TermsImportBatch{}, err
				}
			}
		}
	} else if decision == "cancelled" {
		err = tx.QueryRow(ctx, `
			UPDATE app.partner_terms_import_batches
			SET state = 'cancelled', cancelled_by = $3::uuid, cancelled_at = $4
			WHERE id = $1::uuid AND organization_id = $2::uuid
			RETURNING id::text, organization_id::text, source_hash, total_rows, valid_rows, invalid_rows, state, uploaded_by::text, COALESCE(cancelled_by::text, ''), created_at, cancelled_at
		`, batchID, orgID, reviewerID, now).
			Scan(&batch.ID, &batch.OrganizationID, &batch.SourceHash, &batch.TotalRows, &batch.ValidRows, &batch.InvalidRows, &batch.State, &batch.UploadedBy, &batch.CancelledBy, &batch.CreatedAt, &batch.CancelledAt)
		if err != nil {
			return TermsImportBatch{}, err
		}
	} else {
		return TermsImportBatch{}, errors.New("decision must be approved or cancelled")
	}

	return batch, tx.Commit(ctx)
}

func (p *PostgresTermsImportStore) ListTermsBatches(ctx context.Context, userID, orgID string) ([]TermsImportBatch, error) {
	if p.pool == nil {
		return nil, errors.New("database pool unavailable")
	}
	rows, err := p.pool.Query(ctx, `
		SELECT id::text, organization_id::text, source_hash, total_rows, valid_rows, invalid_rows, state, uploaded_by::text, COALESCE(approved_by::text, ''), COALESCE(cancelled_by::text, ''), created_at, approved_at, cancelled_at
		FROM app.partner_terms_import_batches
		WHERE organization_id = $1::uuid
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TermsImportBatch
	for rows.Next() {
		var b TermsImportBatch
		if err := rows.Scan(&b.ID, &b.OrganizationID, &b.SourceHash, &b.TotalRows, &b.ValidRows, &b.InvalidRows, &b.State, &b.UploadedBy, &b.ApprovedBy, &b.CancelledBy, &b.CreatedAt, &b.ApprovedAt, &b.CancelledAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (p *PostgresTermsImportStore) GetTermsBatch(ctx context.Context, userID, orgID, batchID string) (TermsImportBatch, []TermsImportRow, error) {
	if p.pool == nil {
		return TermsImportBatch{}, nil, errors.New("database pool unavailable")
	}
	var b TermsImportBatch
	err := p.pool.QueryRow(ctx, `
		SELECT id::text, organization_id::text, source_hash, total_rows, valid_rows, invalid_rows, state, uploaded_by::text, COALESCE(approved_by::text, ''), COALESCE(cancelled_by::text, ''), created_at, approved_at, cancelled_at
		FROM app.partner_terms_import_batches
		WHERE id = $1::uuid AND organization_id = $2::uuid
	`, batchID, orgID).
		Scan(&b.ID, &b.OrganizationID, &b.SourceHash, &b.TotalRows, &b.ValidRows, &b.InvalidRows, &b.State, &b.UploadedBy, &b.ApprovedBy, &b.CancelledBy, &b.CreatedAt, &b.ApprovedAt, &b.CancelledAt)
	if err != nil {
		return TermsImportBatch{}, nil, ErrTermsBatchNotFound
	}

	rows, err := p.pool.Query(ctx, `
		SELECT id::text, batch_id::text, row_number, customer_name, contact_email, contact_phone, proposed_credit_limit_kobo, proposed_grace_hours, opening_balance_kobo, opening_balance_reference, validation_status, validation_error, COALESCE(invitation_id::text, ''), created_at
		FROM app.partner_terms_import_rows
		WHERE batch_id = $1::uuid
		ORDER BY row_number
	`, batchID)
	if err != nil {
		return TermsImportBatch{}, nil, err
	}
	defer rows.Close()

	var outRows []TermsImportRow
	for rows.Next() {
		var r TermsImportRow
		var lim, bal int64
		if err := rows.Scan(&r.ID, &r.BatchID, &r.RowNumber, &r.CustomerName, &r.ContactEmail, &r.ContactPhone, &lim, &r.ProposedGraceHours, &bal, &r.OpeningBalanceReference, &r.ValidationStatus, &r.ValidationError, &r.InvitationID, &r.CreatedAt); err != nil {
			return TermsImportBatch{}, nil, err
		}
		r.ProposedCreditLimitKobo = ledger.Money(lim)
		r.OpeningBalanceKobo = ledger.Money(bal)
		outRows = append(outRows, r)
	}
	return b, outRows, rows.Err()
}
