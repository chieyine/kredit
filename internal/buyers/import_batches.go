package buyers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"kredit/internal/access"
)

var ErrImportInvalid = errors.New("invalid import roster")

var ErrImportConflict = errors.New("the import has changed or cannot perform this action")
var ErrImportAuthority = errors.New("current permission to invite business customers is required")

// Import contacts are encrypted at rest. They are proposals, never balances,
// accepted terms, verified identities or payment authorizations.
type ImportContact struct {
	SourceReference string `json:"source_reference"`
	Target          string `json:"target"`
	TargetType      string `json:"target_type"`
	LegalName       string `json:"legal_name"`
	TradingName     string `json:"trading_name"`
	BusinessType    string `json:"business_type"`
	BusinessAddress string `json:"business_address"`
	Industry        string `json:"industry"`
}

func (c ImportContact) invitation() CreateInvitationInput {
	return CreateInvitationInput{SourceReference: c.SourceReference, Target: c.Target, TargetType: c.TargetType, LegalName: c.LegalName, TradingName: c.TradingName, BusinessType: c.BusinessType, BusinessAddress: c.BusinessAddress, Industry: c.Industry}
}

type ImportBatch struct {
	ID            string          `json:"id"`
	SourceHash    string          `json:"source_hash"`
	State         string          `json:"state"`
	RowCount      int             `json:"row_count"`
	CreatedAt     time.Time       `json:"created_at"`
	Contacts      []ImportContact `json:"contacts,omitempty"`
	CompletedRows []int           `json:"completed_rows"`
}
type ImportBatchService interface {
	SaveImport(context.Context, string, string, string, []ImportContact) (ImportBatch, error)
	ListImports(context.Context, string, string) ([]ImportBatch, error)
	ReadImport(context.Context, string, string, string) (ImportBatch, error)
	TransitionImport(context.Context, string, string, string, string) (ImportBatch, error)
	CreateImportInvitation(context.Context, string, string, string, int) (CreateInvitationResult, ImportContact, error)
}

var _ ImportBatchService = (*PostgresStore)(nil)
var importSourceHash = regexp.MustCompile(`^[0-9a-f]{64}$`)
var importReference = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
var importEmail = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
var importPhone = regexp.MustCompile(`^\+234[789][01][0-9]{8}$`)

func validateImport(hash string, contacts []ImportContact) error {
	if !importSourceHash.MatchString(hash) || len(contacts) < 1 || len(contacts) > 200 {
		return errors.New("a file fingerprint and 1 to 200 contacts are required")
	}
	refs, targets := map[string]bool{}, map[string]bool{}
	for n, c := range contacts {
		fail := func() error {
			return fmt.Errorf("row %d: review the reference, contact and required business details", n+1)
		}
		if !importReference.MatchString(c.SourceReference) || refs[c.SourceReference] {
			return fail()
		}
		refs[c.SourceReference] = true
		if err := validateInvitationInput(c.invitation()); err != nil {
			return fail()
		}
		if (c.TargetType == "email" && !importEmail.MatchString(c.Target)) || (c.TargetType == "phone" && !importPhone.MatchString(c.Target)) {
			return fail()
		}
		target := c.TargetType + ":" + normalizeTarget(c.Target)
		if targets[target] {
			return fail()
		}
		targets[target] = true
		switch c.BusinessType {
		case "unregistered_business", "registered_business", "sole_proprietor", "limited_company", "partnership":
		default:
			return fail()
		}
		for _, v := range []string{c.Target, c.LegalName, c.TradingName, c.BusinessAddress, c.Industry} {
			if len(v) > 500 || strings.ContainsRune(v, 0) {
				return fail()
			}
		}
	}
	return nil
}

// Row locks serialize writes against membership revocation/account suspension.
// The transaction retains these locks until the batch operation is committed.
func lockImportAuthority(ctx context.Context, tx pgx.Tx, actor, org string) error {
	var role access.Role
	err := tx.QueryRow(ctx, `SELECT m.role FROM app.memberships m JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id WHERE o.status NOT IN ('suspended','closed') AND m.organization_id=$1::uuid AND m.user_id=$2::uuid AND m.status='active' AND u.status='active' FOR SHARE OF m,u,o`, org, actor).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrImportAuthority
	}
	if err != nil {
		return err
	}
	if !access.Can(role, access.PermissionInviteBuyers) {
		return ErrImportAuthority
	}
	return nil
}
func (s *PostgresStore) importTx(ctx context.Context, actor, org string) (pgx.Tx, error) {
	tx, err := s.beginTxContext(ctx, actor, org)
	if err != nil {
		return nil, err
	}
	if err = lockImportAuthority(ctx, tx, actor, org); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}
func (s *PostgresStore) SaveImport(ctx context.Context, actor, org, hash string, contacts []ImportContact) (ImportBatch, error) {
	if err := validateImport(hash, contacts); err != nil {
		return ImportBatch{}, fmt.Errorf("%w: %v", ErrImportInvalid, err)
	}
	encoded, err := json.Marshal(contacts)
	if err != nil {
		return ImportBatch{}, err
	}
	digest := sha256.Sum256(encoded)
	encrypted, err := s.encrypt(encoded)
	if err != nil {
		return ImportBatch{}, err
	}
	tx, err := s.importTx(ctx, actor, org)
	if err != nil {
		return ImportBatch{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,168))`, org+":"+hash); err != nil {
		return ImportBatch{}, err
	}
	var id string
	var existing []byte
	err = tx.QueryRow(ctx, `SELECT id::text,payload_hash FROM app.distributor_import_batches WHERE organization_id=$1::uuid AND source_hash=$2`, org, hash).Scan(&id, &existing)
	if err == nil && !bytes.Equal(existing, digest[:]) {
		return ImportBatch{}, ErrImportConflict
	}
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `INSERT INTO app.distributor_import_batches(organization_id,source_hash,payload_hash,payload_ciphertext,row_count,created_by) VALUES($1::uuid,$2,$3,$4,$5,$6::uuid) RETURNING id::text`, org, hash, digest[:], encrypted, len(contacts), actor).Scan(&id)
	}
	if err != nil {
		return ImportBatch{}, err
	}
	batch, err := s.readImportTx(ctx, tx, org, id, false)
	if err != nil {
		return ImportBatch{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ImportBatch{}, err
	}
	return batch, nil
}
func (s *PostgresStore) readImportTx(ctx context.Context, tx pgx.Tx, org, id string, lock bool) (ImportBatch, error) {
	b := ImportBatch{CompletedRows: []int{}}
	var encoded []byte
	query := `SELECT id::text,source_hash,state,row_count,created_at,payload_ciphertext FROM app.distributor_import_batches WHERE organization_id=$1::uuid AND id=$2::uuid`
	if lock {
		query += " FOR UPDATE"
	}
	if err := tx.QueryRow(ctx, query, org, id).Scan(&b.ID, &b.SourceHash, &b.State, &b.RowCount, &b.CreatedAt, &encoded); err != nil {
		return b, err
	}
	raw, err := s.decrypt(encoded)
	if err != nil {
		return b, err
	}
	if err = json.Unmarshal(raw, &b.Contacts); err != nil {
		return b, err
	}
	if len(b.Contacts) != b.RowCount {
		return b, errors.New("import payload integrity failure")
	}
	rows, err := tx.Query(ctx, `SELECT row_number FROM app.distributor_import_rows WHERE batch_id=$1::uuid ORDER BY row_number`, id)
	if err != nil {
		return b, err
	}
	defer rows.Close()
	for rows.Next() {
		var n int
		if err = rows.Scan(&n); err != nil {
			return b, err
		}
		b.CompletedRows = append(b.CompletedRows, n)
	}
	return b, rows.Err()
}
func (s *PostgresStore) ReadImport(ctx context.Context, actor, org, id string) (ImportBatch, error) {
	tx, err := s.importTx(ctx, actor, org)
	if err != nil {
		return ImportBatch{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return s.readImportTx(ctx, tx, org, id, false)
}
func (s *PostgresStore) ListImports(ctx context.Context, actor, org string) ([]ImportBatch, error) {
	tx, err := s.importTx(ctx, actor, org)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `SELECT b.id::text,b.source_hash,b.state,b.row_count,b.created_at,ARRAY(SELECT row_number FROM app.distributor_import_rows r WHERE r.batch_id=b.id ORDER BY row_number) FROM app.distributor_import_batches b WHERE b.organization_id=$1::uuid ORDER BY b.created_at DESC,b.id DESC LIMIT 100`, org)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ImportBatch{}
	for rows.Next() {
		var b ImportBatch
		if err = rows.Scan(&b.ID, &b.SourceHash, &b.State, &b.RowCount, &b.CreatedAt, &b.CompletedRows); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}
func (s *PostgresStore) TransitionImport(ctx context.Context, actor, org, id, action string) (ImportBatch, error) {
	if action != "approve" && action != "cancel" {
		return ImportBatch{}, ErrImportConflict
	}
	tx, err := s.importTx(ctx, actor, org)
	if err != nil {
		return ImportBatch{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	b, err := s.readImportTx(ctx, tx, org, id, true)
	if err != nil {
		return b, err
	}
	switch action {
	case "approve":
		if b.State == "cancelled" {
			return b, ErrImportConflict
		}
		if b.State == "draft" {
			_, err = tx.Exec(ctx, `UPDATE app.distributor_import_batches SET state='approved',approved_by=$2::uuid,approved_at=now() WHERE id=$1::uuid`, id, actor)
			b.State = "approved"
		}
	case "cancel":
		if b.State == "completed" {
			return b, ErrImportConflict
		}
		if b.State != "cancelled" {
			_, err = tx.Exec(ctx, `UPDATE app.distributor_import_batches SET state='cancelled',cancelled_by=$2::uuid,cancelled_at=now() WHERE id=$1::uuid`, id, actor)
			b.State = "cancelled"
		}
	}
	if err != nil {
		return b, err
	}
	err = tx.Commit(ctx)
	return b, err
}
func (s *PostgresStore) CreateImportInvitation(ctx context.Context, actor, org, id string, n int) (CreateInvitationResult, ImportContact, error) {
	var empty CreateInvitationResult
	var contact ImportContact
	tx, err := s.importTx(ctx, actor, org)
	if err != nil {
		return empty, contact, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	b, err := s.readImportTx(ctx, tx, org, id, true)
	if err != nil {
		return empty, contact, err
	}
	if b.State != "approved" && b.State != "completed" || n < 1 || n > len(b.Contacts) {
		return empty, contact, ErrImportConflict
	}
	contact = b.Contacts[n-1]
	input := contact.invitation()
	s.guardMu.RLock()
	guard := s.inviteGuard
	s.guardMu.RUnlock()
	if guard != nil {
		if err = guard(input); err != nil {
			return empty, contact, err
		}
	}
	result, err := s.createInvitationTx(ctx, tx, actor, org, input, true)
	if err != nil {
		return empty, contact, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.distributor_import_rows(batch_id,row_number,invitation_id,created_by) VALUES($1::uuid,$2,$3::uuid,$4::uuid) ON CONFLICT(batch_id,row_number) DO NOTHING`, id, n, result.Invitation.ID, actor); err != nil {
		return empty, contact, err
	}
	if _, err = tx.Exec(ctx, `UPDATE app.distributor_import_batches b SET state='completed' WHERE b.id=$1::uuid AND b.row_count=(SELECT count(*) FROM app.distributor_import_rows r WHERE r.batch_id=b.id)`, id); err != nil {
		return empty, contact, err
	}
	if err = tx.Commit(ctx); err != nil {
		return empty, contact, err
	}
	return result, contact, nil
}
