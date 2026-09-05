package collections

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"kredit/internal/businesspolicy"
	"kredit/internal/db"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresEngine struct {
	noticeMinimum time.Duration
	pool          *pgxpool.Pool
	base          *Engine
}

var _ Service = (*PostgresEngine)(nil)

func NewPostgresEngine(pool *pgxpool.Pool, base *Engine) *PostgresEngine {
	if base == nil {
		base = NewEngine(nil, nil, nil, nil)
	}
	return &PostgresEngine{pool: pool, base: base}
}
func (e *PostgresEngine) RequirePriorNotice(delay time.Duration) { e.noticeMinimum = delay }
func (e *PostgresEngine) ProviderStatus() ProviderStatus         { return e.base.ProviderStatus() }
func (e *PostgresEngine) SetFeatureEnabled(value bool)           { e.base.SetFeatureEnabled(value) }
func (e *PostgresEngine) SetMaxRetries(value int)                { e.base.SetMaxRetries(value) }
func (e *PostgresEngine) Eligibility(id string, now time.Time) (Eligibility, error) {
	return e.EligibilityContext(context.Background(), id, now)
}
func (e *PostgresEngine) EligibilityContext(ctx context.Context, id string, now time.Time) (Eligibility, error) {
	local, err := e.load(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return e.base.Eligibility(id, now)
	}
	if err != nil {
		return Eligibility{}, err
	}
	return local.Eligibility(id, now)
}

// reserveProvider captures a debit intent without doing network I/O. The
// reservation and reference must commit before Submit can reach a bank.
type reserveProvider struct {
	Provider
	request *Request
}

func (p *reserveProvider) Submit(_ context.Context, request Request) (Response, error) {
	p.request = &request
	return Response{State: ProviderPending}, nil
}
func (p *reserveProvider) Capabilities() Capabilities {
	if provider, ok := p.Provider.(CapabilityProvider); ok {
		return provider.Capabilities()
	}
	return Capabilities{}
}
func (p *reserveProvider) Sign(event Webhook) string {
	if signer, ok := p.Provider.(WebhookSigner); ok {
		return signer.Sign(event)
	}
	return ""
}
func (e *PostgresEngine) Start(ctx context.Context, id, key string, now time.Time) (Attempt, error) {
	return e.submitPrepared(ctx, id, func(local *Engine) (Attempt, error) { return local.Start(ctx, id, key, now) })
}
func (e *PostgresEngine) submitPrepared(ctx context.Context, id string, prepare func(*Engine) (Attempt, error)) (Attempt, error) {
	var result Attempt
	var request *Request
	err := e.mutate(ctx, id, func(local *Engine) error {
		proxy := &reserveProvider{Provider: local.provider}
		local.provider = proxy
		var err error
		result, err = prepare(local)
		request = proxy.request
		return err
	})
	if err != nil || request == nil {
		return result, err
	}
	// No resubmission after a crash here: reconciliation uses the saved request.
	response, submitErr := e.base.provider.Submit(ctx, *request)
	if submitErr != nil {
		response = Response{State: ProviderTimeout}
	}
	event := Webhook{EventID: "provider-submission:" + result.ID, ExternalReference: result.ExternalReference,
		ProviderCollectionID: response.ProviderCollectionID, State: response.State,
		SucceededAmountKobo: response.SucceededAmountKobo, FailureCode: response.FailureCode,
		Retryable: response.Retryable, SettlementState: response.SettlementState, SettlementReference: response.SettlementReference}
	if signer, ok := e.base.provider.(WebhookSigner); ok {
		event.Signature = signer.Sign(event)
	}
	return e.ProcessWebhook(ctx, event)
}
func (e *PostgresEngine) ProcessWebhook(ctx context.Context, event Webhook) (Attempt, error) {
	id, organizationID, err := e.obligationForExternal(ctx, event.ExternalReference)
	if err != nil {
		return Attempt{}, err
	}
	ctx, err = e.scopeTenantContext(ctx, organizationID)
	if err != nil {
		return Attempt{}, err
	}
	var result Attempt
	err = e.mutate(ctx, id, func(local *Engine) error {
		var operationErr error
		result, operationErr = local.ProcessWebhook(ctx, event)
		return operationErr
	})
	return result, err
}
func (e *PostgresEngine) SignalWebhook(ctx context.Context, event Webhook) (Attempt, error) {
	id, organizationID, err := e.obligationForExternal(ctx, event.ExternalReference)
	if err != nil {
		return Attempt{}, err
	}
	ctx, err = e.scopeTenantContext(ctx, organizationID)
	if err != nil {
		return Attempt{}, err
	}
	var result Attempt
	err = e.mutate(ctx, id, func(local *Engine) error {
		var operationErr error
		result, operationErr = local.SignalWebhook(ctx, event)
		return operationErr
	})
	return result, err
}
func (e *PostgresEngine) Reconcile(ctx context.Context, attemptID string) (Attempt, error) {
	id, organizationID, err := e.obligationForAttempt(ctx, attemptID)
	if err != nil {
		return Attempt{}, err
	}
	ctx, err = e.scopeTenantContext(ctx, organizationID)
	if err != nil {
		return Attempt{}, err
	}
	var result Attempt
	err = e.mutate(ctx, id, func(local *Engine) error {
		var operationErr error
		result, operationErr = local.Reconcile(ctx, attemptID)
		return operationErr
	})
	return result, err
}
func (e *PostgresEngine) Retry(ctx context.Context, attemptID string, now time.Time) (Attempt, error) {
	id, organizationID, err := e.obligationForAttempt(ctx, attemptID)
	if err != nil {
		return Attempt{}, err
	}
	ctx, err = e.scopeTenantContext(ctx, organizationID)
	if err != nil {
		return Attempt{}, err
	}
	return e.submitPrepared(ctx, id, func(local *Engine) (Attempt, error) { return local.Retry(ctx, attemptID, now) })
}
func (e *PostgresEngine) Cancel(ctx context.Context, attemptID string) (Attempt, error) {
	id, organizationID, err := e.obligationForAttempt(ctx, attemptID)
	if err != nil {
		return Attempt{}, err
	}
	ctx, err = e.scopeTenantContext(ctx, organizationID)
	if err != nil {
		return Attempt{}, err
	}
	var result Attempt
	err = e.mutate(ctx, id, func(local *Engine) error {
		var operationErr error
		result, operationErr = local.Cancel(ctx, attemptID)
		return operationErr
	})
	return result, err
}
func (e *PostgresEngine) GetAttempt(attemptID string) (Attempt, bool) {
	return e.GetAttemptContext(context.Background(), attemptID)
}
func (e *PostgresEngine) GetAttemptContext(ctx context.Context, attemptID string) (Attempt, bool) {
	id, organizationID, err := e.obligationForAttempt(ctx, attemptID)
	if err != nil {
		return Attempt{}, false
	}
	ctx, err = e.scopeTenantContext(ctx, organizationID)
	if err != nil {
		return Attempt{}, false
	}
	local, err := e.load(ctx, id)
	if err != nil {
		return Attempt{}, false
	}
	return local.GetAttempt(attemptID)
}
func (e *PostgresEngine) ListAttempts(id string) []Attempt {
	local, err := e.load(context.Background(), id)
	if err != nil {
		return []Attempt{}
	}
	return local.ListAttempts(id)
}

type persistedCollection struct {
	Reservations []CollectionReservation `json:"reservations"`
	Attempts     []Attempt               `json:"attempts"`
	Events       []string                `json:"events"`
}

func (e *PostgresEngine) fresh() *Engine {
	e.base.mu.Lock()
	defer e.base.mu.Unlock()
	return &Engine{provider: e.base.provider, payments: e.base.payments, snapshot: e.base.snapshot, due: e.base.due, reservations: map[string]*CollectionReservation{}, attempts: map[string]*Attempt{}, byKey: map[string]string{}, byExternal: map[string]string{}, events: map[string]bool{}, now: e.base.now, featureEnabled: e.base.featureEnabled, reservationTTL: e.base.reservationTTL, maxRetries: e.base.maxRetries}
}
func installCollection(local *Engine, state persistedCollection) {
	for _, value := range state.Reservations {
		item := value
		local.reservations[item.ID] = &item
		local.byKey[item.ObligationID+"\x00"+item.IdempotencyKey] = ""
	}
	for _, value := range state.Attempts {
		item := value
		local.attempts[item.ID] = &item
		local.byExternal[item.ExternalReference] = item.ID
		if reservation := local.reservations[item.ReservationID]; reservation != nil {
			local.byKey[reservation.ObligationID+"\x00"+reservation.IdempotencyKey] = item.ID
		}
	}
	for _, id := range state.Events {
		local.events[id] = true
	}
}
func snapshotCollection(local *Engine) persistedCollection {
	local.mu.Lock()
	defer local.mu.Unlock()
	state := persistedCollection{Reservations: []CollectionReservation{}, Attempts: []Attempt{}, Events: []string{}}
	for _, value := range local.reservations {
		state.Reservations = append(state.Reservations, *value)
	}
	for _, value := range local.attempts {
		state.Attempts = append(state.Attempts, *value)
	}
	for id := range local.events {
		state.Events = append(state.Events, id)
	}
	return state
}
func (e *PostgresEngine) mutate(ctx context.Context, id string, operation func(*Engine) error) error {
	if e == nil || e.pool == nil {
		return errors.New("collection database is not configured")
	}
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := db.SetObligationContext(ctx, tx, id); err != nil {
		return err
	}
	policy, err := businesspolicy.ReadTx(ctx, tx)
	if err != nil {
		return err
	}
	if e.noticeMinimum > 0 {
		minimum := e.noticeMinimum
		if policy.Initialized {
			minimum = max(minimum, time.Duration(policy.Values.NoticeHours)*time.Hour)
		}
		if _, err := tx.Exec(ctx, `SELECT set_config('app.collection_notice_min_seconds',$1,true)`, fmt.Sprint(int64(minimum/time.Second))); err != nil {
			return err
		}
	}
	// Serialize mutations without locking the obligation row. Creating the
	// FK-backed snapshot before calling the payment repository takes a key-share
	// lock on app.obligations; the payment transaction then needs FOR UPDATE on
	// that same row and deadlocks against its caller. The advisory transaction
	// lock is scoped to this aggregate and does not interfere with the atomic
	// payment transaction.
	var locked bool
	if err := tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended($1,73421))`, id).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		return errors.New("collection operation is busy; retry with the same key")
	}
	var payload []byte
	err = tx.QueryRow(ctx, `SELECT aggregate FROM app.collection_aggregate_snapshots WHERE obligation_id=$1::uuid`, id).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		payload = nil
	} else if err != nil {
		return err
	}
	local := e.fresh()
	if policy.Initialized {
		local.featureEnabled = local.featureEnabled && policy.Values.CollectionsEnabled
		local.maxRetries = int(policy.Values.MaxRetries)
	}
	if len(payload) > 0 && string(payload) != "{}" {
		var state persistedCollection
		if err := json.Unmarshal(payload, &state); err != nil {
			return err
		}
		installCollection(local, state)
	}
	if err := operation(local); err != nil {
		return err
	}
	state := snapshotCollection(local)
	encoded, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO app.collection_aggregate_snapshots(obligation_id,aggregate) VALUES($1::uuid,$2::jsonb) ON CONFLICT(obligation_id) DO UPDATE SET aggregate=EXCLUDED.aggregate,version=app.collection_aggregate_snapshots.version+1,updated_at=now()`, id, encoded); err != nil {
		return err
	}
	if err := syncCollectionTx(ctx, tx, state); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (e *PostgresEngine) load(ctx context.Context, id string) (*Engine, error) {
	if e == nil || e.pool == nil {
		return nil, errors.New("collection database is not configured")
	}
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := db.SetObligationContext(ctx, tx, id); err != nil {
		return nil, err
	}
	var payload []byte
	if err := tx.QueryRow(ctx, `SELECT aggregate FROM app.collection_aggregate_snapshots WHERE obligation_id=$1::uuid`, id).Scan(&payload); err != nil {
		return nil, err
	}
	local := e.fresh()
	var state persistedCollection
	if err := json.Unmarshal(payload, &state); err != nil {
		return nil, err
	}
	installCollection(local, state)
	return local, nil
}
func (e *PostgresEngine) scopeTenantContext(ctx context.Context, organizationID string) (context.Context, error) {
	if existing, ok := db.TenantFromContext(ctx); ok && existing.OrganizationID != "" {
		if existing.OrganizationID != organizationID {
			return nil, errors.New("collection resource is outside the authorized tenant")
		}
		return ctx, nil
	}
	return db.WithTenantContext(ctx, "", organizationID), nil
}
func (e *PostgresEngine) obligationForAttempt(ctx context.Context, id string) (string, string, error) {
	var obligation, organizationID string
	err := e.pool.QueryRow(ctx, `SELECT obligation_id::text,organization_id::text FROM app.collection_identity_by_attempt($1::uuid)`, id).Scan(&obligation, &organizationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", errors.New("collection attempt not found")
	}
	return obligation, organizationID, err
}
func (e *PostgresEngine) obligationForExternal(ctx context.Context, reference string) (string, string, error) {
	var obligation, organizationID string
	err := e.pool.QueryRow(ctx, `SELECT obligation_id::text,organization_id::text FROM app.collection_identity_by_external($1)`, reference).Scan(&obligation, &organizationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", errors.New("collection attempt not found")
	}
	return obligation, organizationID, err
}
func syncCollectionTx(ctx context.Context, tx pgx.Tx, state persistedCollection) error {
	for _, r := range state.Reservations {
		if _, err := tx.Exec(ctx, `INSERT INTO app.collection_reservations(id,obligation_id,schedule_item_id,outstanding_snapshot_version,reserved_amount_kobo,state,expires_at,idempotency_key,created_at) VALUES($1::uuid,$2::uuid,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,$9) ON CONFLICT(id) DO UPDATE SET state=EXCLUDED.state`, r.ID, r.ObligationID, r.ScheduleItemID, r.OutstandingVersion, int64(r.ReservedAmountKobo), r.State, r.ExpiresAt, "collection:"+r.ID, r.CreatedAt); err != nil {
			return err
		}
	}
	for _, a := range state.Attempts {
		// Bind a new request to the exact mandate whose capacity was reserved.
		// Existing attempts retain that binding even after mandate renewal.
		var mismatched bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.collection_reservations r JOIN app.payment_mandates m ON m.id=r.mandate_id WHERE r.id=$1::uuid AND (m.provider<>$2 OR m.provider_mandate_id<>$3) AND NOT EXISTS(SELECT 1 FROM app.collection_attempts WHERE id=$4::uuid))`, a.ReservationID, a.Provider, a.MandateReference, a.ID).Scan(&mismatched); err != nil {
			return err
		}
		if mismatched {
			return errors.New("collection mandate changed before reservation; refresh eligibility")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO app.collection_attempts(id,reservation_id,obligation_id,provider,provider_collection_id,external_reference,requested_amount_kobo,succeeded_amount_kobo,state,attempt_number,retry_classification,failure_code,requested_at,final_at,settlement_state,settlement_reference) VALUES($1::uuid,$2::uuid,$3::uuid,$4,NULLIF($5,''),$6,$7,$8,$9,$10,NULLIF($11,''),NULLIF($12,''),$13,$14,NULLIF($15,''),NULLIF($16,'')) ON CONFLICT(id) DO UPDATE SET provider_collection_id=EXCLUDED.provider_collection_id,succeeded_amount_kobo=EXCLUDED.succeeded_amount_kobo,state=EXCLUDED.state,attempt_number=EXCLUDED.attempt_number,retry_classification=EXCLUDED.retry_classification,failure_code=EXCLUDED.failure_code,final_at=EXCLUDED.final_at,settlement_state=EXCLUDED.settlement_state,settlement_reference=EXCLUDED.settlement_reference`, a.ID, a.ReservationID, a.ObligationID, a.Provider, a.ProviderCollectionID, a.ExternalReference, int64(a.RequestedAmountKobo), int64(a.SucceededAmountKobo), a.State, a.AttemptNumber, a.RetryClassification, a.FailureCode, a.RequestedAt, nullableCollectionTime(a.FinalAt), a.SettlementState, a.SettlementReference); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE app.collection_attempts SET next_retry_at=$2 WHERE id=$1::uuid`, a.ID, nullableCollectionTime(a.NextRetryAt)); err != nil {
			return err
		}
		eventType := map[string]string{AttemptPending: "COLLECTION_INITIATED", AttemptSubmitted: "COLLECTION_INITIATED", AttemptUnknown: "COLLECTION_UNCERTAIN", AttemptSucceeded: "COLLECTION_SUCCESSFUL", AttemptPartial: "PARTIAL_COLLECTION", AttemptFailed: "COLLECTION_FAILED", AttemptCancelled: "COLLECTION_CANCELLED"}[a.State]
		key := fmt.Sprintf("attempt:%s:%s:%d", a.ID, eventType, a.SucceededAmountKobo)
		if _, err := tx.Exec(ctx, `INSERT INTO app.collection_events(attempt_id,obligation_id,event_type,amount_kobo,correlation_id,idempotency_key) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6) ON CONFLICT(idempotency_key) DO NOTHING`, a.ID, a.ObligationID, eventType, a.SucceededAmountKobo, a.ExternalReference, key); err != nil {
			return err
		}
		if !a.NextRetryAt.IsZero() {
			payload, _ := json.Marshal(map[string]any{"event": "RETRY_SCHEDULED", "attempt_id": a.ID, "next_retry_at": a.NextRetryAt})
			if _, err := tx.Exec(ctx, `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key) VALUES('collection',$1,'notification.requested',$2::jsonb,$3) ON CONFLICT(idempotency_key) DO NOTHING`, a.ObligationID, payload, "retry-scheduled:"+a.ID); err != nil {
				return err
			}
		}
		noticeAmount := a.SucceededAmountKobo
		if noticeAmount == 0 {
			noticeAmount = a.RequestedAmountKobo
		}
		payload, _ := json.Marshal(map[string]any{"attempt_id": a.ID, "obligation_id": a.ObligationID, "event": eventType, "amount_kobo": noticeAmount, "next_retry_at": a.NextRetryAt})
		if _, err := tx.Exec(ctx, `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key) VALUES('collection',$1,'notification.requested',$2::jsonb,$3) ON CONFLICT(idempotency_key) DO NOTHING`, a.ObligationID, payload, "notify:"+key); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO app.collection_attempt_index(attempt_id,external_reference,obligation_id) VALUES($1::uuid,$2,$3::uuid) ON CONFLICT(attempt_id) DO NOTHING`, a.ID, a.ExternalReference, a.ObligationID); err != nil {
			return err
		}
	}
	return nil
}
func nullableCollectionTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

func (e *PostgresEngine) ReadAttempts(id string) ([]Attempt, error) {
	return e.ReadAttemptsContext(context.Background(), id)
}
func (e *PostgresEngine) ReadAttemptsContext(ctx context.Context, id string) ([]Attempt, error) {
	local, err := e.load(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return []Attempt{}, nil
	}
	if err != nil {
		return nil, err
	}
	return local.ListAttempts(id), nil
}
