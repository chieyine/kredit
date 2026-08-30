package collections

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresEngine struct {
	pool *pgxpool.Pool
	base *Engine
}

var _ Service = (*PostgresEngine)(nil)

func NewPostgresEngine(pool *pgxpool.Pool, base *Engine) *PostgresEngine {
	if base == nil {
		base = NewEngine(nil, nil, nil, nil)
	}
	return &PostgresEngine{pool: pool, base: base}
}
func (e *PostgresEngine) ProviderStatus() ProviderStatus { return e.base.ProviderStatus() }
func (e *PostgresEngine) SetFeatureEnabled(value bool)   { e.base.SetFeatureEnabled(value) }
func (e *PostgresEngine) SetMaxRetries(value int)        { e.base.SetMaxRetries(value) }
func (e *PostgresEngine) Eligibility(id string, now time.Time) (Eligibility, error) {
	return e.base.Eligibility(id, now)
}
func (e *PostgresEngine) Start(ctx context.Context, id, key string, now time.Time) (Attempt, error) {
	var result Attempt
	err := e.mutate(ctx, id, func(local *Engine) error { var err error; result, err = local.Start(ctx, id, key, now); return err })
	return result, err
}
func (e *PostgresEngine) ProcessWebhook(ctx context.Context, event Webhook) (Attempt, error) {
	id, err := e.obligationForExternal(ctx, event.ExternalReference)
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
func (e *PostgresEngine) Reconcile(ctx context.Context, attemptID string) (Attempt, error) {
	id, err := e.obligationForAttempt(ctx, attemptID)
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
	id, err := e.obligationForAttempt(ctx, attemptID)
	if err != nil {
		return Attempt{}, err
	}
	var result Attempt
	err = e.mutate(ctx, id, func(local *Engine) error {
		var operationErr error
		result, operationErr = local.Retry(ctx, attemptID, now)
		return operationErr
	})
	return result, err
}
func (e *PostgresEngine) Cancel(ctx context.Context, attemptID string) (Attempt, error) {
	id, err := e.obligationForAttempt(ctx, attemptID)
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
	id, err := e.obligationForAttempt(context.Background(), attemptID)
	if err != nil {
		return Attempt{}, false
	}
	local, err := e.load(context.Background(), id)
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
		local.byKey[item.IdempotencyKey] = ""
	}
	for _, value := range state.Attempts {
		item := value
		local.attempts[item.ID] = &item
		local.byExternal[item.ExternalReference] = item.ID
		if reservation := local.reservations[item.ReservationID]; reservation != nil {
			local.byKey[reservation.IdempotencyKey] = item.ID
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
	// Serialize mutations without locking the obligation row. Creating the
	// FK-backed snapshot before calling the payment repository takes a key-share
	// lock on app.obligations; the payment transaction then needs FOR UPDATE on
	// that same row and deadlocks against its caller. The advisory transaction
	// lock is scoped to this aggregate and does not interfere with the atomic
	// payment transaction.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 73421))`, id); err != nil {
		return err
	}
	var payload []byte
	err = tx.QueryRow(ctx, `SELECT aggregate FROM app.collection_aggregate_snapshots WHERE obligation_id=$1::uuid`, id).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		payload = nil
	} else if err != nil {
		return err
	}
	local := e.fresh()
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
	var payload []byte
	if err := e.pool.QueryRow(ctx, `SELECT aggregate FROM app.collection_aggregate_snapshots WHERE obligation_id=$1::uuid`, id).Scan(&payload); err != nil {
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
func (e *PostgresEngine) obligationForAttempt(ctx context.Context, id string) (string, error) {
	var obligation string
	err := e.pool.QueryRow(ctx, `SELECT obligation_id::text FROM app.collection_attempt_index WHERE attempt_id=$1::uuid`, id).Scan(&obligation)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("collection attempt not found")
	}
	return obligation, err
}
func (e *PostgresEngine) obligationForExternal(ctx context.Context, reference string) (string, error) {
	var obligation string
	err := e.pool.QueryRow(ctx, `SELECT obligation_id::text FROM app.collection_attempt_index WHERE external_reference=$1`, reference).Scan(&obligation)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("collection attempt not found")
	}
	return obligation, err
}
func syncCollectionTx(ctx context.Context, tx pgx.Tx, state persistedCollection) error {
	for _, r := range state.Reservations {
		if _, err := tx.Exec(ctx, `INSERT INTO app.collection_reservations(id,obligation_id,schedule_item_id,outstanding_snapshot_version,reserved_amount_kobo,state,expires_at,idempotency_key,created_at) VALUES($1::uuid,$2::uuid,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,$9) ON CONFLICT(id) DO UPDATE SET state=EXCLUDED.state`, r.ID, r.ObligationID, r.ScheduleItemID, r.OutstandingVersion, int64(r.ReservedAmountKobo), r.State, r.ExpiresAt, r.IdempotencyKey, r.CreatedAt); err != nil {
			return err
		}
	}
	for _, a := range state.Attempts {
		if _, err := tx.Exec(ctx, `INSERT INTO app.collection_attempts(id,reservation_id,obligation_id,provider,provider_collection_id,external_reference,requested_amount_kobo,succeeded_amount_kobo,state,attempt_number,retry_classification,failure_code,requested_at,final_at,settlement_state,settlement_reference) VALUES($1::uuid,$2::uuid,$3::uuid,$4,NULLIF($5,''),$6,$7,$8,$9,$10,NULLIF($11,''),NULLIF($12,''),$13,$14,NULLIF($15,''),NULLIF($16,'')) ON CONFLICT(id) DO UPDATE SET provider_collection_id=EXCLUDED.provider_collection_id,succeeded_amount_kobo=EXCLUDED.succeeded_amount_kobo,state=EXCLUDED.state,attempt_number=EXCLUDED.attempt_number,retry_classification=EXCLUDED.retry_classification,failure_code=EXCLUDED.failure_code,final_at=EXCLUDED.final_at,settlement_state=EXCLUDED.settlement_state,settlement_reference=EXCLUDED.settlement_reference`, a.ID, a.ReservationID, a.ObligationID, a.Provider, a.ProviderCollectionID, a.ExternalReference, int64(a.RequestedAmountKobo), int64(a.SucceededAmountKobo), a.State, a.AttemptNumber, a.RetryClassification, a.FailureCode, a.RequestedAt, nullableCollectionTime(a.FinalAt), a.SettlementState, a.SettlementReference); err != nil {
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
