package buyers

import (
	"context"
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
	ID                   string     `json:"id"`
	OrganizationID       string     `json:"organization_id"`
	SourceHash           string     `json:"source_hash"`
	FinancialApplication string     `json:"financial_application"`
	TotalRows            int        `json:"total_rows"`
	ValidRows            int        `json:"valid_rows"`
	InvalidRows          int        `json:"invalid_rows"`
	State                string     `json:"state"`
	UploadedBy           string     `json:"uploaded_by"`
	ApprovedBy           string     `json:"approved_by,omitempty"`
	CancelledBy          string     `json:"cancelled_by,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	ApprovedAt           *time.Time `json:"approved_at,omitempty"`
	CancelledAt          *time.Time `json:"cancelled_at,omitempty"`
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
	canonicalHash, err := TermsSourceHash(inputRows)
	if err != nil {
		return TermsImportBatch{}, err
	}
	if sourceHash != "" && sourceHash != canonicalHash {
		return TermsImportBatch{}, ErrTermsInvalidRow
	}
	sourceHash = canonicalHash
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, existing := range m.batches {
		if existing.OrganizationID == orgID && existing.SourceHash == sourceHash {
			return cloneTermsBatch(*existing), nil
		}
	}
	batchID := identifier.New()
	batch := &TermsImportBatch{
		ID:                   batchID,
		OrganizationID:       orgID,
		SourceHash:           sourceHash,
		FinancialApplication: "not_applied",
		TotalRows:            len(inputRows),
		State:                "staged",
		UploadedBy:           userID,
		CreatedAt:            time.Now().UTC(),
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

	return cloneTermsBatch(*batch), nil
}

func (m *MemoryTermsImportStore) ReviewTermsBatch(ctx context.Context, reviewerID, orgID, batchID, decision string) (TermsImportBatch, error) {
	if reviewerID == "" || orgID == "" {
		return TermsImportBatch{}, ErrTermsAuthority
	}
	if err := ctx.Err(); err != nil {
		return TermsImportBatch{}, err
	}
	if decision != "approved" && decision != "cancelled" {
		return TermsImportBatch{}, ErrTermsInvalidRow
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	batch, ok := m.batches[batchID]
	if !ok || batch.OrganizationID != orgID {
		return TermsImportBatch{}, ErrTermsBatchNotFound
	}
	if batch.UploadedBy == reviewerID {
		return TermsImportBatch{}, ErrTermsDualControl
	}
	if batch.State == decision && ((decision == "approved" && batch.ApprovedBy == reviewerID) || (decision == "cancelled" && batch.CancelledBy == reviewerID)) {
		return cloneTermsBatch(*batch), nil
	}
	if batch.State != "staged" && batch.State != "reviewing" {
		return TermsImportBatch{}, ErrTermsAlreadyClosed
	}

	now := time.Now().UTC()
	if decision == "approved" {
		batch.State = "approved"
		batch.ApprovedBy = reviewerID
		batch.ApprovedAt = &now
		// Review records independent approval only. Applying credit terms or
		// an opening balance needs a separately linked, accepted obligation.
		// A staging-row ID must never be used as an obligation/journal reference.
	} else if decision == "cancelled" {
		batch.State = "cancelled"
		batch.CancelledBy = reviewerID
		batch.CancelledAt = &now
	} else {
		return TermsImportBatch{}, errors.New("decision must be approved or cancelled")
	}

	return cloneTermsBatch(*batch), nil
}

func (m *MemoryTermsImportStore) ListTermsBatches(ctx context.Context, userID, orgID string) ([]TermsImportBatch, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []TermsImportBatch{}
	for _, b := range m.batches {
		if b.OrganizationID == orgID {
			out = append(out, cloneTermsBatch(*b))
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
	return cloneTermsBatch(*batch), outRows, nil
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
	canonicalHash, err := TermsSourceHash(inputRows)
	if err != nil {
		return TermsImportBatch{}, err
	}
	if sourceHash != "" && sourceHash != canonicalHash {
		return TermsImportBatch{}, ErrTermsInvalidRow
	}
	sourceHash = canonicalHash
	tx, err := p.beginTermsTx(ctx, userID, orgID, "stage")
	if err != nil {
		return TermsImportBatch{}, err
	}
	defer tx.Rollback(ctx)
	validCount := 0
	for _, r := range inputRows {
		if r.CustomerName != "" {
			validCount++
		}
	}
	batch, err := scanTermsBatch(tx.QueryRow(ctx, `
        INSERT INTO app.partner_terms_import_batches
        (organization_id,source_hash,payload_hash,payload_ciphertext,total_rows,valid_rows,invalid_rows,state,uploaded_by)
        VALUES($1::uuid,$2,decode($2,'hex'),$3,$4,$5,$6,'staged',$7::uuid)
        ON CONFLICT(organization_id,source_hash) DO NOTHING
        RETURNING `+termsBatchColumns, orgID, sourceHash, []byte{}, len(inputRows), validCount, len(inputRows)-validCount, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		batch, err = scanTermsBatch(tx.QueryRow(ctx, `SELECT `+termsBatchColumns+`
            FROM app.partner_terms_import_batches WHERE organization_id=$1::uuid AND source_hash=$2`, orgID, sourceHash))
		if err != nil {
			return TermsImportBatch{}, err
		}
		return batch, tx.Commit(ctx)
	}
	if err != nil {
		return TermsImportBatch{}, err
	}
	// Structured rows below are the input record. No unencrypted input is
	// mislabeled as ciphertext: there is no separately retained raw upload.
	for i, r := range inputRows {
		status, detail := "valid", ""
		if r.CustomerName == "" {
			status, detail = "invalid", "Customer name is required"
		}
		if _, err := tx.Exec(ctx, `INSERT INTO app.partner_terms_import_rows
            (batch_id,row_number,customer_name,contact_email,contact_phone,proposed_credit_limit_kobo,proposed_grace_hours,opening_balance_kobo,opening_balance_reference,validation_status,validation_error)
            VALUES($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, batch.ID, i+1, r.CustomerName, r.ContactEmail, r.ContactPhone, int64(r.ProposedCreditLimitKobo), r.ProposedGraceHours, int64(r.OpeningBalanceKobo), r.OpeningBalanceReference, status, detail); err != nil {
			return TermsImportBatch{}, err
		}
	}
	return batch, tx.Commit(ctx)
}

func (p *PostgresTermsImportStore) ReviewTermsBatch(ctx context.Context, reviewerID, orgID, batchID, decision string) (TermsImportBatch, error) {
	if decision != "approved" && decision != "cancelled" {
		return TermsImportBatch{}, ErrTermsInvalidRow
	}
	tx, err := p.beginTermsTx(ctx, reviewerID, orgID, "review")
	if err != nil {
		return TermsImportBatch{}, err
	}
	defer tx.Rollback(ctx)
	batch, err := scanTermsBatch(tx.QueryRow(ctx, `SELECT `+termsBatchColumns+`
        FROM app.partner_terms_import_batches WHERE id=$1::uuid AND organization_id=$2::uuid FOR UPDATE`, batchID, orgID))
	if errors.Is(err, pgx.ErrNoRows) {
		return TermsImportBatch{}, ErrTermsBatchNotFound
	}
	if err != nil {
		return TermsImportBatch{}, err
	}
	if batch.UploadedBy == reviewerID {
		return TermsImportBatch{}, ErrTermsDualControl
	}
	if batch.State == decision && ((decision == "approved" && batch.ApprovedBy == reviewerID) || (decision == "cancelled" && batch.CancelledBy == reviewerID)) {
		return batch, tx.Commit(ctx)
	}
	if batch.State != "staged" && batch.State != "reviewing" {
		return TermsImportBatch{}, ErrTermsAlreadyClosed
	}
	if decision == "approved" {
		batch, err = scanTermsBatch(tx.QueryRow(ctx, `UPDATE app.partner_terms_import_batches
            SET state='approved',approved_by=$3::uuid,approved_at=statement_timestamp()
            WHERE id=$1::uuid AND organization_id=$2::uuid RETURNING `+termsBatchColumns, batchID, orgID, reviewerID))
	} else {
		batch, err = scanTermsBatch(tx.QueryRow(ctx, `UPDATE app.partner_terms_import_batches
            SET state='cancelled',cancelled_by=$3::uuid,cancelled_at=statement_timestamp()
            WHERE id=$1::uuid AND organization_id=$2::uuid RETURNING `+termsBatchColumns, batchID, orgID, reviewerID))
	}
	if err != nil {
		return TermsImportBatch{}, err
	}
	// Approval does not apply a row, originate debt, change a credit limit or
	// debit an account. Keep validation_status valid/invalid/duplicate.
	// Financial application is intentionally unavailable until each row has
	// verified business, agreement and obligation linkage with atomic posting.
	return batch, tx.Commit(ctx)
}

func (p *PostgresTermsImportStore) ListTermsBatches(ctx context.Context, userID, orgID string) ([]TermsImportBatch, error) {
	tx, err := p.beginTermsTx(ctx, userID, orgID, "read")
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT `+termsBatchColumns+` FROM app.partner_terms_import_batches
        WHERE organization_id=$1::uuid ORDER BY created_at DESC,id DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TermsImportBatch{}
	for rows.Next() {
		batch, err := scanTermsBatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, batch)
	}
	return out, rows.Err()
}

func (p *PostgresTermsImportStore) GetTermsBatch(ctx context.Context, userID, orgID, batchID string) (TermsImportBatch, []TermsImportRow, error) {
	tx, err := p.beginTermsTx(ctx, userID, orgID, "read")
	if err != nil {
		return TermsImportBatch{}, nil, err
	}
	defer tx.Rollback(ctx)
	batch, err := scanTermsBatch(tx.QueryRow(ctx, `SELECT `+termsBatchColumns+`
        FROM app.partner_terms_import_batches WHERE id=$1::uuid AND organization_id=$2::uuid`, batchID, orgID))
	if errors.Is(err, pgx.ErrNoRows) {
		return TermsImportBatch{}, nil, ErrTermsBatchNotFound
	}
	if err != nil {
		return TermsImportBatch{}, nil, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text,batch_id::text,row_number,customer_name,contact_email,contact_phone,
        proposed_credit_limit_kobo,proposed_grace_hours,opening_balance_kobo,opening_balance_reference,
        validation_status,validation_error,COALESCE(invitation_id::text,''),created_at
        FROM app.partner_terms_import_rows WHERE batch_id=$1::uuid ORDER BY row_number`, batchID)
	if err != nil {
		return TermsImportBatch{}, nil, err
	}
	defer rows.Close()
	out := []TermsImportRow{}
	for rows.Next() {
		var row TermsImportRow
		if err := rows.Scan(&row.ID, &row.BatchID, &row.RowNumber, &row.CustomerName, &row.ContactEmail, &row.ContactPhone,
			&row.ProposedCreditLimitKobo, &row.ProposedGraceHours, &row.OpeningBalanceKobo, &row.OpeningBalanceReference,
			&row.ValidationStatus, &row.ValidationError, &row.InvitationID, &row.CreatedAt); err != nil {
			return TermsImportBatch{}, nil, err
		}
		out = append(out, row)
	}
	return batch, out, rows.Err()
}
