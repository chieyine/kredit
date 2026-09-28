package credit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"kredit/internal/access"
	"kredit/internal/ledger"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *PostgresStore) persist(requestID string) error { return s.persistAs(requestID, "", "") }

func (s *PostgresStore) persistAs(requestID, actorID string, permission access.Permission) error {
	s.mu.RLock()
	request := s.requests[requestID]
	if request == nil {
		s.mu.RUnlock()
		return ErrRequestNotFound
	}
	view := s.viewLocked(request)
	s.mu.RUnlock()
	return s.persistViewAs(view, actorID, permission)
}

func (s *PostgresStore) persistViewAs(view View, actorID string, permission access.Permission) error {
	request := &view.Request
	version := request.Version
	payload, err := json.Marshal(view)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(context.Background(), `SELECT set_config('app.current_organization_id', $1, true),set_config('app.current_user_id',$2,true)`, request.SupplierOrganizationID, actorID); err != nil {
		return err
	}
	if strings.HasPrefix(string(permission), "purchase:") {
		if _, err := tx.Exec(context.Background(), `SELECT app.lock_purchase_permission($1::uuid,$2,$3)`, request.BuyerBusinessID, strings.TrimPrefix(string(permission), "purchase:"), int64(request.PrincipalKobo)); err != nil {
			return err
		}
	} else if permission != "" {
		if err := access.LockOrganizationAuthority(context.Background(), tx, actorID, request.SupplierOrganizationID, permission); err != nil {
			return err
		}
	}
	// Migration 070 refuses a deemed-acceptance receipt without delivered notice
	// evidence. The window travels with the transaction so the database applies
	// the same policy the application believes it is applying.
	if _, err := tx.Exec(context.Background(), `SELECT set_config('app.deemed_acceptance_min_seconds', $1, true)`, fmt.Sprint(int64(s.deemedAcceptanceNoticeWindow()/time.Second))); err != nil {
		return err
	}
	if err := syncNormalizedCredit(context.Background(), tx, view); err != nil {
		return err
	}
	var commandTag pgconn.CommandTag
	if version <= 1 {
		commandTag, err = tx.Exec(context.Background(), `INSERT INTO app.credit_aggregate_snapshots (credit_request_id, supplier_organization_id, buyer_user_id, aggregate, version, updated_at) VALUES ($1, $2, $3, $4::jsonb, $5, $6) ON CONFLICT (credit_request_id) DO NOTHING`, request.ID, request.SupplierOrganizationID, request.BuyerUserID, payload, version, time.Now().UTC())
	} else {
		commandTag, err = tx.Exec(context.Background(), `UPDATE app.credit_aggregate_snapshots SET aggregate = $3::jsonb, version = $4, updated_at = $5 WHERE credit_request_id = $1 AND supplier_organization_id = $2 AND version = $6`, request.ID, request.SupplierOrganizationID, payload, version, time.Now().UTC(), version-1)
	}
	if err != nil {
		return fmt.Errorf("persist credit aggregate: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("credit aggregate changed concurrently; retry the operation")
	}
	return tx.Commit(context.Background())
}

func syncNormalizedCredit(ctx context.Context, tx pgx.Tx, view View) error {
	return syncNormalizedCreditWithEvidence(ctx, tx, view, false)
}

func syncNormalizedCreditWithEvidence(ctx context.Context, tx pgx.Tx, view View, system bool) error {
	r := view.Request
	_, err := tx.Exec(ctx, `
		INSERT INTO app.credit_requests
		(invoice_document_id, fee_terms, id, supplier_organization_id, buyer_user_id, buyer_business_id, principal_kobo, currency, goods_description, invoice_reference, invoice_document_hash, due_date, grace_hours, collection_at, state, agreement_version_id, mandate_id, acceptance_id, release_id, receipt_id, obligation_id, created_by, created_at, updated_at, version)
		VALUES (NULLIF($25,'')::uuid,$24::jsonb,$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,NULLIF($8,''),NULLIF($9,''),$10::date,$11,$12,$13,NULLIF($14,'')::uuid,NULLIF($15,'')::uuid,NULLIF($16,'')::uuid,NULLIF($17,'')::uuid,NULLIF($18,'')::uuid,NULLIF($19,'')::uuid,$20::uuid,$21,$22,$23)
		ON CONFLICT (id) DO UPDATE SET principal_kobo=EXCLUDED.principal_kobo,goods_description=EXCLUDED.goods_description,invoice_reference=EXCLUDED.invoice_reference,invoice_document_hash=EXCLUDED.invoice_document_hash,invoice_document_id=EXCLUDED.invoice_document_id,due_date=EXCLUDED.due_date,grace_hours=EXCLUDED.grace_hours,collection_at=EXCLUDED.collection_at,state=EXCLUDED.state, agreement_version_id=EXCLUDED.agreement_version_id, mandate_id=EXCLUDED.mandate_id, acceptance_id=EXCLUDED.acceptance_id, release_id=EXCLUDED.release_id, receipt_id=EXCLUDED.receipt_id, obligation_id=EXCLUDED.obligation_id, updated_at=EXCLUDED.updated_at, version=EXCLUDED.version`,
		r.ID, r.SupplierOrganizationID, r.BuyerUserID, r.BuyerBusinessID, int64(r.PrincipalKobo), r.Currency, r.GoodsDescription, r.InvoiceReference, r.InvoiceDocumentHash, r.DueDate, r.GraceHours, r.CollectionAt, r.State, r.AgreementVersionID, r.MandateID, r.AcceptanceID, r.ReleaseID, r.ReceiptID, r.ObligationID, r.CreatedBy, r.CreatedAt, r.UpdatedAt, r.Version, feeTermsJSON(r.FeeTerms), r.InvoiceDocumentID)
	if err != nil {
		return fmt.Errorf("persist normalized credit request: %w", err)
	}
	if a := view.Agreement; a.ID != "" && !system {
		if _, err := tx.Exec(ctx, `INSERT INTO app.agreement_versions (id,credit_request_id,version,canonical_json,document_hash,terms_version,privacy_version,created_by,created_at,canonical_bytes) VALUES ($1::uuid,$2::uuid,$3,$4::jsonb,$5,$6,$7,$8::uuid,$9,$10) ON CONFLICT (id) DO NOTHING`, a.ID, a.CreditRequestID, a.Version, a.CanonicalJSON, a.DocumentHash, a.TermsVersion, a.PrivacyVersion, a.CreatedBy, a.CreatedAt, a.CanonicalBytes); err != nil {
			return fmt.Errorf("persist agreement version: %w", err)
		}
	}
	if a := view.Acceptance; a != nil && !system {
		if _, err := tx.Exec(ctx, `INSERT INTO app.agreement_acceptances (id,credit_request_id,agreement_version_id,accepting_user_id,person_id,business_id,acceptance_method,authentication_level,agreement_hash,mandate_provider_id,accepted_at) VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7,$8,$9,$10,$11) ON CONFLICT (id) DO NOTHING`, a.ID, a.CreditRequestID, a.AgreementVersionID, a.AcceptingUserID, a.PersonID, a.BusinessID, a.AcceptanceMethod, a.AuthenticationLevel, a.AgreementHash, a.MandateProviderID, a.AcceptedAt); err != nil {
			return fmt.Errorf("persist agreement acceptance: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key) VALUES('credit_request',$1,'notification.requested',jsonb_build_object('event','OBLIGATION_ACCEPTED','acceptance_id',$2::text),$3) ON CONFLICT(idempotency_key) DO NOTHING`, r.ID, a.ID, "acceptance-notification:"+a.ID); err != nil {
			return err
		}
	}
	if release := view.Release; release != nil && !system {
		if _, err := tx.Exec(ctx, `INSERT INTO app.goods_releases (id,credit_request_id,supplier_actor_id,delivery_method,notes,released_at) VALUES ($1::uuid,$2::uuid,$3::uuid,$4,NULLIF($5,''),$6) ON CONFLICT (id) DO NOTHING`, release.ID, release.CreditRequestID, release.SupplierActorID, release.DeliveryMethod, release.Notes, release.ReleasedAt); err != nil {
			return fmt.Errorf("persist goods release: %w", err)
		}
		// The buyer is told, in the same transaction that records the release,
		// that goods are out and that silence has a deadline. The idempotency
		// key is derived from the release so the delivery receipt for this
		// notice can be found again when deemed acceptance is considered.
		if _, err := tx.Exec(ctx, `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key) VALUES('credit_request',$1,'notification.requested',jsonb_build_object('event','GOODS_RELEASED','release_id',$2::text,'ends_at',$4::text,'amount_kobo',$5::bigint),$3) ON CONFLICT(idempotency_key) DO NOTHING`, r.ID, release.ID, "goods-release-notification:"+release.ID, release.DeemedAcceptedAt.Format(time.RFC3339), int64(r.PrincipalKobo)); err != nil {
			return fmt.Errorf("queue goods release notice: %w", err)
		}
	}
	for _, receipt := range view.Receipts {
		if system {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO app.receipt_confirmations (id,credit_request_id,buyer_user_id,state,issue_reason,received_at) VALUES ($1::uuid,$2::uuid,$3::uuid,$4,NULLIF($5,''),$6) ON CONFLICT (id) DO NOTHING`, receipt.ID, receipt.CreditRequestID, receipt.BuyerUserID, receipt.State, receipt.IssueReason, receipt.ReceivedAt); err != nil {
			return fmt.Errorf("persist receipt confirmation: %w", err)
		}
	}
	if obligation := view.Obligation; obligation != nil {
		if _, err := tx.Exec(ctx, `INSERT INTO app.obligations (id,credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at) VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,$8,$9,$10,$11,$12::uuid,$13) ON CONFLICT (id) DO UPDATE SET lifecycle_status=EXCLUDED.lifecycle_status,payment_status=EXCLUDED.payment_status,outstanding_kobo=EXCLUDED.outstanding_kobo`, obligation.ID, obligation.CreditRequestID, obligation.AgreementVersionID, obligation.SupplierOrganizationID, obligation.BuyerBusinessID, int64(obligation.PrincipalKobo), obligation.Currency, obligation.LifecycleStatus, obligation.PaymentStatus, int64(obligation.OutstandingKobo), int64(obligation.BaseFeeKobo), obligation.LedgerTransactionID, obligation.ActivatedAt); err != nil {
			return fmt.Errorf("persist obligation: %w", err)
		}

		if obligation.BaseFeeKobo > 0 {
			baseRate, _ := r.FeeTerms.Rates()
			if _, err := tx.Exec(ctx, `INSERT INTO app.fees(supplier_organization_id,obligation_id,fee_type,basis_amount_kobo,rate_basis_points,amount_kobo,currency,state,accrued_at) VALUES($1::uuid,$2::uuid,'base_service',$3,$4,$5,$6,'accrued',$7) ON CONFLICT(obligation_id,fee_type) WHERE fee_type='base_service' DO NOTHING`, obligation.SupplierOrganizationID, obligation.ID, int64(obligation.PrincipalKobo), baseRate, int64(obligation.BaseFeeKobo), obligation.Currency, obligation.ActivatedAt); err != nil {
				return fmt.Errorf("persist base fee: %w", err)
			}
		}
	}
	return nil
}

func (s *PostgresStore) persistByObligation(obligationID string) error {
	s.mu.RLock()
	var requestID string
	for id, request := range s.requests {
		if request.ObligationID == obligationID {
			requestID = id
			break
		}
	}
	s.mu.RUnlock()
	if requestID == "" {
		return ErrRequestNotFound
	}
	return s.persist(requestID)
}

// InvalidateObligation removes the process-local projection after another
// transactional repository (payments or disputes) updates its authoritative
// database state. The next read rehydrates the committed aggregate.
func (s *PostgresStore) InvalidateObligation(obligationID string) {
	s.mu.RLock()
	requestID := ""
	for _, obligation := range s.obligations {
		if obligation.ID == obligationID {
			requestID = obligation.CreditRequestID
			break
		}
	}
	s.mu.RUnlock()
	if requestID != "" {
		s.discard(requestID)
	}
}

func feeTermsJSON(f *ledger.FeeTerms) []byte { b, _ := json.Marshal(f); return b }

// AutoActivateMatured discovers eligible persisted sales, including on a cold worker.
// Buyer-originated evidence remains read-only; each activation records distinct
// immutable system evidence in the same transaction as its ledger and schedule.
var errDeemedNotEligible = errors.New("system acceptance is no longer eligible")

func (s *PostgresStore) AutoActivateMatured(ctx context.Context, asOf time.Time) ([]string, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("credit database is not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	seconds := max(int64(s.deemedAcceptanceNoticeWindow()/time.Second), int64(259200))
	var enabled bool
	var hours int64
	if err := s.pool.QueryRow(ctx, `SELECT COALESCE((SELECT value='true'::jsonb FROM app.platform_settings WHERE key='features.system_acceptance'),false),COALESCE((SELECT hours FROM app.system_acceptance_settings()),72)`).Scan(&enabled, &hours); err != nil {
		return nil, err
	}
	if !enabled {
		return []string{}, nil
	}
	if hours < 72 || hours > 720 {
		return nil, errors.New("automatic recognition waiting period is invalid")
	}
	seconds = max(seconds, hours*3600)
	rows, err := s.pool.Query(ctx, `SELECT request_id::text,buyer_id::text FROM app.deemed_acceptance_candidates($1)`, seconds)
	if err != nil {
		return nil, err
	}
	type candidate struct{ id, buyer string }
	candidates := []candidate{}
	for rows.Next() {
		var c candidate
		if err = rows.Scan(&c.id, &c.buyer); err != nil {
			rows.Close()
			return nil, err
		}
		candidates = append(candidates, c)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	activated := []string{}
	for _, c := range candidates {
		// Eligibility uses database time; a caller cannot advance the waiting period.
		_, _, err = s.recordReceiptTransaction(ctx, c.id, c.buyer, "confirmed", deemedAcceptanceReason, true)
		if errors.Is(err, errDeemedNotEligible) {
			continue
		}
		if err != nil {
			return activated, err
		}
		activated = append(activated, c.id)
	}
	return activated, nil
}

func (s *PostgresStore) checkPurchasePermission(r *CreditRequest, actor, action string) bool {
	if s.pool == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, actor); err != nil {
		return false
	}
	var allowed bool
	err = tx.QueryRow(ctx, `SELECT app.can_purchase($1::uuid,$2,$3)`, r.BuyerBusinessID, action, int64(r.PrincipalKobo)).Scan(&allowed)
	return err == nil && allowed
}

func (s *PostgresStore) acceptingPerson(actor string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, actor); err != nil {
		return "", err
	}
	var person string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM app.persons WHERE user_id=$1::uuid`, actor).Scan(&person); err != nil {
		return "", errors.New("verified acting person required")
	}
	return person, nil
}

// Fetch presentation capabilities in one database round trip. They are hints;
// every command independently locks and checks its current permission.
func (s *PostgresStore) readPurchaseActions(r *CreditRequest, actor string) []string {
	result := []string{}
	if s.pool == nil {
		return result
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return result
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, actor); err != nil {
		return result
	}
	if err = tx.QueryRow(ctx, `SELECT ARRAY(SELECT action FROM unnest(ARRAY['read','review','accept','receive','bank','payments','disputes']) WITH ORDINALITY AS permissions(action,position) WHERE app.can_purchase($1::uuid,action,$2) ORDER BY position)`, r.BuyerBusinessID, int64(r.PrincipalKobo)).Scan(&result); err != nil {
		return []string{}
	}
	return result
}
