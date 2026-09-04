package platformops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type CommandInput struct {
	PreflightVersion int64          `json:"-"`
	Type             string         `json:"command_type"`
	TargetType       string         `json:"target_type"`
	TargetID         string         `json:"target_id"`
	OrganizationID   string         `json:"organization_id,omitempty"`
	Reason           string         `json:"reason"`
	ExpectedVersion  int64          `json:"expected_version"`
	IdempotencyKey   string         `json:"-"`
	CorrelationID    string         `json:"-"`
	Scope            string         `json:"scope,omitempty"`
	ExpiresAt        time.Time      `json:"expires_at,omitempty"`
	Resolution       string         `json:"resolution,omitempty"`
	ExternalResult   map[string]any `json:"-"`
}

type Command struct {
	ID             string         `json:"id,omitempty"`
	Type           string         `json:"command_type"`
	TargetType     string         `json:"target_type"`
	TargetID       string         `json:"target_id"`
	Reason         string         `json:"reason,omitempty"`
	CurrentVersion int64          `json:"current_version"`
	Impact         map[string]any `json:"impact_preview"`
	State          string         `json:"state,omitempty"`
	Result         map[string]any `json:"result,omitempty"`
	CorrelationID  string         `json:"correlation_id,omitempty"`
	CreatedAt      time.Time      `json:"created_at,omitempty"`
}

var commandTypes = map[string]bool{
	"retry_job": true, "retry_webhook": true, "suspend_user": true, "restore_user": true,
	"suspend_organization": true, "restore_organization": true, "place_risk_hold": true, "lift_risk_hold": true,
	"request_reconciliation": true, "resolve_unknown_submission": true, "retry_collection": true, "cancel_collection": true,
}

type commandQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s *Store) PreflightCommand(ctx context.Context, in CommandInput) (Command, error) {
	if err := validateCommand(in, true); err != nil {
		return Command{}, err
	}
	preview, err := s.PreviewCommand(ctx, in)
	if err != nil {
		return Command{}, err
	}
	if preview.CurrentVersion != in.ExpectedVersion {
		return Command{}, errors.New("version conflict before provider operation")
	}
	return preview, nil
}
func (s *Store) PreviewCommand(ctx context.Context, in CommandInput) (Command, error) {
	return s.previewCommand(ctx, in, s.pool)
}
func (s *Store) previewCommand(ctx context.Context, in CommandInput, q commandQueryer) (Command, error) {
	if s == nil || s.pool == nil {
		return Command{}, errors.New("operations database is not configured")
	}
	if err := validateCommand(in, false); err != nil {
		return Command{}, err
	}
	version, state, err := s.targetVersion(ctx, q, in, false)
	if err != nil {
		return Command{}, err
	}
	impact := map[string]any{"target_state": state, "will_notify": true, "audit": "immutable", "financial_effect": "none unless explicitly stated"}
	switch in.Type {
	case "retry_job":
		impact["effect"] = "requeue one failed job using its existing idempotency boundary"
	case "retry_webhook":
		impact["effect"] = "requeue one verified failed webhook without changing its provider identity"
	case "suspend_user":
		impact["effect"] = "block sign-in and revoke active sessions"
	case "restore_user":
		impact["effect"] = "restore the status captured by the active suspension"
	case "suspend_organization":
		impact["effect"] = "block sensitive organization mutations"
	case "restore_organization":
		impact["effect"] = "restore the status captured by the active suspension"
	case "place_risk_hold":
		impact["effect"] = "block " + in.Scope + " actions until expiry or lift"
	case "lift_risk_hold":
		impact["effect"] = "remove this active risk hold"
	case "request_reconciliation":
		impact["effect"] = "open a tracked provider reconciliation case"
	case "resolve_unknown_submission":
		impact["effect"] = "query the provider and apply only its idempotent authoritative result"
	case "retry_collection":
		impact["effect"] = "submit one bounded retry; payment recognition remains provider-reference idempotent"
	case "cancel_collection":
		impact["effect"] = "ask the provider to cancel; no local cancellation occurs without provider confirmation"
	}
	return Command{Type: in.Type, TargetType: in.TargetType, TargetID: in.TargetID, CurrentVersion: version, Impact: impact, State: "PREVIEWED"}, nil
}

func validateCommand(in CommandInput, execute bool) error {
	if !commandTypes[in.Type] {
		return errors.New("unsupported operations command")
	}
	if strings.TrimSpace(in.TargetType) == "" || strings.TrimSpace(in.TargetID) == "" {
		return errors.New("target type and target id are required")
	}
	if execute && len(strings.TrimSpace(in.Reason)) < 8 {
		return errors.New("structured reason must be at least 8 characters")
	}
	if execute && in.ExpectedVersion < 1 {
		return errors.New("current expected version is required")
	}
	if execute && strings.TrimSpace(in.IdempotencyKey) == "" {
		return errors.New("Idempotency-Key is required")
	}
	if in.Type == "place_risk_hold" {
		if in.TargetType != "buyer" && in.TargetType != "supplier" {
			return errors.New("risk hold target must be buyer or supplier")
		}
		if in.Scope != "credit" && in.Scope != "release" && in.Scope != "collection" && in.Scope != "settlement" && in.Scope != "all_sensitive" {
			return errors.New("valid risk hold scope is required")
		}
		if !in.ExpiresAt.After(time.Now().UTC()) {
			return errors.New("risk hold expiry must be in the future")
		}
	}
	return nil
}

func (s *Store) targetVersion(ctx context.Context, q commandQueryer, in CommandInput, lock bool) (int64, string, error) {
	suffix := ""
	if lock {
		suffix = " FOR UPDATE"
	}
	var version int64
	var state string
	var row pgx.Row
	switch in.Type {
	case "retry_job":
		row = q.QueryRow(ctx, `SELECT attempt+1,state FROM jobs.river_job WHERE id=$1`+suffix, in.TargetID)
	case "retry_webhook":
		row = q.QueryRow(ctx, `SELECT attempts+1,state FROM app.provider_webhook_inbox WHERE provider=$1 AND event_id=$2`+suffix, in.TargetType, in.TargetID)
	case "suspend_user":
		row = q.QueryRow(ctx, `SELECT version,status FROM app.users WHERE id=$1::uuid`+suffix, in.TargetID)
	case "restore_user":
		row = q.QueryRow(ctx, `SELECT version,'suspended' FROM app.platform_suspensions WHERE id=$1::uuid AND target_type='user' AND lifted_at IS NULL`+suffix, in.TargetID)
	case "suspend_organization":
		row = q.QueryRow(ctx, `SELECT version,status FROM app.organizations WHERE id=$1::uuid`+suffix, in.TargetID)
	case "restore_organization":
		row = q.QueryRow(ctx, `SELECT version,'suspended' FROM app.platform_suspensions WHERE id=$1::uuid AND target_type='organization' AND lifted_at IS NULL`+suffix, in.TargetID)
	case "place_risk_hold":
		version, state = 1, "not_held"
		return version, state, nil
	case "lift_risk_hold":
		row = q.QueryRow(ctx, `SELECT version,'active' FROM app.risk_holds WHERE id=$1::uuid AND lifted_at IS NULL AND expires_at>now()`+suffix, in.TargetID)
	case "request_reconciliation":
		version, state = 1, "unrequested"
		return version, state, nil
	case "resolve_unknown_submission", "retry_collection", "cancel_collection":
		row = q.QueryRow(ctx, `SELECT attempt_number,state FROM app.collection_attempts WHERE id=$1::uuid`+suffix, in.TargetID)
	}
	if err := row.Scan(&version, &state); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", errors.New("operations target was not found")
		}
		return 0, "", err
	}
	return version, state, nil
}

func (s *Store) ExecuteCommand(ctx context.Context, actorID string, in CommandInput) (Command, error) {
	if err := validateCommand(in, true); err != nil {
		return Command{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Command{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	encodedIntent, err := json.Marshal(in)
	if err != nil {
		return Command{}, err
	}
	digest := sha256.Sum256(encodedIntent)
	requestHash := hex.EncodeToString(digest[:])
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "operation:"+actorID+":"+in.IdempotencyKey); err != nil {
		return Command{}, err
	}
	var storedHash string
	var existing Command
	err = tx.QueryRow(ctx, `SELECT c.id::text,c.command_type,c.target_type,c.target_id,c.expected_version,c.impact_preview,e.state,e.result,c.correlation_id,c.created_at,COALESCE(c.request_hash,'') FROM app.operations_commands c JOIN LATERAL (SELECT state,result FROM app.operations_command_events WHERE command_id=c.id ORDER BY occurred_at DESC LIMIT 1)e ON true WHERE c.requested_by=$1::uuid AND c.idempotency_key=$2`, actorID, in.IdempotencyKey).Scan(&existing.ID, &existing.Type, &existing.TargetType, &existing.TargetID, &existing.CurrentVersion, &existing.Impact, &existing.State, &existing.Result, &existing.CorrelationID, &existing.CreatedAt, &storedHash)
	if err == nil {
		if storedHash != requestHash {
			return Command{}, errors.New("idempotency key belongs to a different or unverifiable command")
		}
		if err = tx.Commit(ctx); err != nil {
			return Command{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Command{}, err
	}
	// Lock through the transaction directly (targetVersion uses the pool), then
	// enforce optimistic concurrency in each mutation below as the final guard.
	preview, err := s.previewCommand(ctx, in, tx)
	if err != nil {
		return Command{}, err
	}
	if in.PreflightVersion == in.ExpectedVersion && len(in.ExternalResult) > 0 {
		preview.CurrentVersion = in.ExpectedVersion
	}
	if preview.CurrentVersion != in.ExpectedVersion {
		return Command{}, fmt.Errorf("version conflict: current version is %d", preview.CurrentVersion)
	}
	impactJSON, _ := json.Marshal(preview.Impact)
	result := map[string]any{}
	var commandID string
	err = tx.QueryRow(ctx, `INSERT INTO app.operations_commands(command_type,target_type,target_id,organization_id,requested_by,reason,expected_version,idempotency_key,impact_preview,correlation_id,request_hash) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5::uuid,$6,$7,$8,$9,$10,$11) RETURNING id::text,created_at`, in.Type, in.TargetType, in.TargetID, in.OrganizationID, actorID, strings.TrimSpace(in.Reason), in.ExpectedVersion, in.IdempotencyKey, impactJSON, in.CorrelationID, requestHash).Scan(&commandID, &preview.CreatedAt)
	if err != nil {
		return Command{}, err
	}
	switch in.Type {
	case "retry_job":
		tag, e := tx.Exec(ctx, `UPDATE jobs.river_job SET state='available',scheduled_at=now(),finalized_at=NULL WHERE id=$1::bigint AND attempt+1=$2 AND state IN ('retryable','discarded','cancelled')`, in.TargetID, in.ExpectedVersion)
		err = e
		if tag.RowsAffected() != 1 && err == nil {
			err = errors.New("job is no longer retryable")
		}
	case "retry_webhook":
		tag, e := tx.Exec(ctx, `UPDATE app.provider_webhook_inbox SET state='received',last_error=NULL,processed_at=NULL WHERE provider=$1 AND event_id=$2 AND attempts+1=$3 AND state='failed'`, in.TargetType, in.TargetID, in.ExpectedVersion)
		err = e
		if tag.RowsAffected() != 1 && err == nil {
			err = errors.New("webhook is no longer retryable")
		}
		if err == nil {
			_, _ = tx.Exec(ctx, `UPDATE jobs.river_job SET state='available',scheduled_at=now(),finalized_at=NULL WHERE kind='kredit.provider_webhook' AND encoded_args->>'provider'=$1 AND encoded_args->>'event_id'=$2 AND state IN ('retryable','discarded','cancelled')`, in.TargetType, in.TargetID)
		}
	case "suspend_user":
		var old string
		err = tx.QueryRow(ctx, `UPDATE app.users SET status='suspended',version=version+1 WHERE id=$1::uuid AND version=$2 AND status='active' RETURNING 'active'`, in.TargetID, in.ExpectedVersion).Scan(&old)
		if err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO app.platform_suspensions(target_type,target_id,previous_status,reason,created_by) VALUES('user',$1::uuid,$2,$3,$4::uuid)`, in.TargetID, old, in.Reason, actorID)
		}
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE app.sessions SET revoked_at=now() WHERE user_id=$1::uuid AND revoked_at IS NULL`, in.TargetID)
		}
		result["affected_target_id"] = in.TargetID
	case "restore_user":
		var affected string
		affected, err = restoreSuspension(ctx, tx, "user", in.TargetID, actorID, in.Reason, in.ExpectedVersion)
		result["affected_target_id"] = affected
	case "suspend_organization":
		var old string
		err = tx.QueryRow(ctx, `SELECT status FROM app.organizations WHERE id=$1::uuid AND version=$2 AND status<>'suspended' FOR UPDATE`, in.TargetID, in.ExpectedVersion).Scan(&old)
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE app.organizations SET status='suspended',version=version+1,updated_at=now() WHERE id=$1::uuid`, in.TargetID)
		}
		if err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO app.platform_suspensions(target_type,target_id,previous_status,reason,created_by) VALUES('organization',$1::uuid,$2,$3,$4::uuid)`, in.TargetID, old, in.Reason, actorID)
		}
		result["affected_target_id"] = in.TargetID
	case "restore_organization":
		var affected string
		affected, err = restoreSuspension(ctx, tx, "organization", in.TargetID, actorID, in.Reason, in.ExpectedVersion)
		result["affected_target_id"] = affected
	case "place_risk_hold":
		var id string
		err = tx.QueryRow(ctx, `INSERT INTO app.risk_holds(target_type,target_id,scope,reason,expires_at,created_by) VALUES($1,$2::uuid,$3,$4,$5,$6::uuid) RETURNING id::text`, in.TargetType, in.TargetID, in.Scope, in.Reason, in.ExpiresAt, actorID).Scan(&id)
		result["hold_id"] = id
		result["affected_target_id"] = in.TargetID
	case "lift_risk_hold":
		var affected string
		err = tx.QueryRow(ctx, `UPDATE app.risk_holds SET lifted_by=$1::uuid,lifted_reason=$2,lifted_at=now(),version=version+1 WHERE id=$3::uuid AND version=$4 AND lifted_at IS NULL RETURNING target_id::text`, actorID, in.Reason, in.TargetID, in.ExpectedVersion).Scan(&affected)
		result["affected_target_id"] = affected
	case "request_reconciliation":
		var id string
		err = tx.QueryRow(ctx, `INSERT INTO app.reconciliation_cases(provider,operation,target_type,target_id,reason,requested_by) VALUES($1,$2,$3,$4,$5,$6::uuid) RETURNING id::text`, in.TargetType, "provider_reconciliation", in.TargetType, in.TargetID, in.Reason, actorID).Scan(&id)
		result["case_id"] = id
	case "resolve_unknown_submission":
		if len(in.ExternalResult) == 0 {
			err = errors.New("verified provider reconciliation result is required")
		} else {
			result = in.ExternalResult
		}
	case "retry_collection", "cancel_collection":
		// Provider interaction is deliberately performed by the collection service
		// before this immutable command is committed; its result is captured here.
		if len(in.ExternalResult) == 0 {
			err = errors.New("verified provider result is required")
		} else {
			result = in.ExternalResult
		}
	}
	if err != nil {
		return Command{}, err
	}
	resultJSON, _ := json.Marshal(result)
	if _, err = tx.Exec(ctx, `INSERT INTO app.operations_command_events(command_id,state,result) VALUES($1::uuid,'APPLIED',$2)`, commandID, resultJSON); err != nil {
		return Command{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Command{}, err
	}
	preview.ID, preview.Reason, preview.State, preview.Result, preview.CorrelationID = commandID, in.Reason, "APPLIED", result, in.CorrelationID
	return preview, nil
}

func restoreSuspension(ctx context.Context, tx pgx.Tx, targetType, id, actor, reason string, version int64) (string, error) {
	var targetID, previous string
	if err := tx.QueryRow(ctx, `SELECT target_id::text,previous_status FROM app.platform_suspensions WHERE id=$1::uuid AND target_type=$2 AND version=$3 AND lifted_at IS NULL FOR UPDATE`, id, targetType, version).Scan(&targetID, &previous); err != nil {
		return "", err
	}
	var err error
	if targetType == "user" {
		_, err = tx.Exec(ctx, `UPDATE app.users SET status=$1,version=version+1 WHERE id=$2::uuid AND status='suspended'`, previous, targetID)
	} else {
		_, err = tx.Exec(ctx, `UPDATE app.organizations SET status=$1,version=version+1,updated_at=now() WHERE id=$2::uuid AND status='suspended'`, previous, targetID)
	}
	if err == nil {
		_, err = tx.Exec(ctx, `UPDATE app.platform_suspensions SET lifted_by=$1::uuid,lifted_reason=$2,lifted_at=now(),version=version+1 WHERE id=$3::uuid`, actor, reason, id)
	}
	return targetID, err
}

type Diagnostics struct {
	WindowMinutes int              `json:"window_minutes"`
	Provider      []map[string]any `json:"provider"`
	Queues        []map[string]any `json:"queues"`
	Integrity     map[string]int64 `json:"integrity"`
	CorrelationID string           `json:"correlation_id"`
}

func (s *Store) Diagnostics(ctx context.Context, windowMinutes int, correlationID string) (Diagnostics, error) {
	if windowMinutes < 5 || windowMinutes > 1440 {
		windowMinutes = 60
	}
	d := Diagnostics{WindowMinutes: windowMinutes, Provider: []map[string]any{}, Queues: []map[string]any{}, Integrity: map[string]int64{}, CorrelationID: redactCorrelation(correlationID)}
	rows, err := s.pool.Query(ctx, `WITH ordered AS (SELECT provider,state,received_at,processed_at,duplicate_count,provider_sequence,lag(provider_sequence) OVER(PARTITION BY provider ORDER BY received_at,event_id) previous_sequence FROM app.provider_webhook_inbox WHERE received_at>=now()-make_interval(mins=>$1)) SELECT provider,count(*)::bigint,count(*) FILTER(WHERE state='failed')::bigint,count(*) FILTER(WHERE state='processing' AND received_at<now()-interval '2 minutes')::bigint,COALESCE(extract(epoch from now()-(min(received_at) FILTER(WHERE state IN('received','processing','failed')))),0)::bigint,COALESCE(avg(extract(epoch from processed_at-received_at)) FILTER(WHERE processed_at IS NOT NULL),0)::double precision,COALESCE(sum(duplicate_count),0)::bigint,count(*) FILTER(WHERE provider_sequence IS NOT NULL AND previous_sequence IS NOT NULL AND provider_sequence<previous_sequence)::bigint FROM ordered GROUP BY provider`, windowMinutes)
	if err != nil {
		return d, err
	}
	for rows.Next() {
		var p string
		var total, failed, timeout, oldest, duplicates, outOfOrder int64
		var latency float64
		if err = rows.Scan(&p, &total, &failed, &timeout, &oldest, &latency, &duplicates, &outOfOrder); err != nil {
			rows.Close()
			return d, err
		}
		d.Provider = append(d.Provider, map[string]any{"provider": p, "operation": "webhook", "total": total, "errors": failed, "timeouts": timeout, "average_latency_seconds": latency, "duplicates": duplicates, "out_of_order": outOfOrder, "oldest_unprocessed_seconds": oldest})
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return d, err
	}
	rows.Close()
	rows, err = s.pool.Query(ctx, `SELECT queue,count(*)::bigint,COALESCE(extract(epoch from now()-min(scheduled_at)),0)::bigint FROM jobs.river_job WHERE state IN('available','pending','retryable','running','scheduled') GROUP BY queue ORDER BY queue`)
	if err != nil {
		return d, err
	}
	for rows.Next() {
		var q string
		var count, age int64
		if err = rows.Scan(&q, &count, &age); err != nil {
			rows.Close()
			return d, err
		}
		d.Queues = append(d.Queues, map[string]any{"queue": q, "count": count, "oldest_seconds": age})
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return d, err
	}
	rows.Close()
	keys := []string{"dead_letters", "unknown_submissions", "reconciliation_overdue", "ledger_drift", "report_drift", "notification_backlog", "scanner_backlog", "mandate_mismatch", "settlement_mismatch"}
	query := `SELECT (SELECT count(*) FROM app.job_dead_letters),(SELECT count(*) FROM app.collection_attempts WHERE state='UNKNOWN'),(SELECT count(*) FROM app.reconciliation_cases WHERE state IN('REQUESTED','IN_PROGRESS') AND created_at<now()-interval '30 minutes'),(SELECT count(*) FROM (SELECT transaction_id FROM ledger.postings GROUP BY transaction_id HAVING sum(debit_kobo)<>sum(credit_kobo)) drift),(SELECT count(*) FROM app.financial_discrepancies WHERE kind='balance'),(SELECT count(*) FROM app.notifications WHERE state IN('scheduled','failed')),(SELECT count(*) FROM app.documents WHERE scan_state IN('PENDING','QUARANTINED')),(SELECT count(*) FROM app.trade_lines WHERE state='ACTIVE' AND (mandate_id IS NULL OR mandate_active=false)),(SELECT count(*) FROM app.financial_discrepancies WHERE kind IN ('settlement','settlement_missing','provider_reversal','settlement_without_payment'))`
	values := make([]int64, len(keys))
	args := make([]any, len(keys))
	for i := range values {
		args[i] = &values[i]
	}
	if err = s.pool.QueryRow(ctx, query).Scan(args...); err != nil {
		return d, err
	}
	for i, k := range keys {
		d.Integrity[k] = values[i]
	}
	return d, nil
}

func redactCorrelation(value string) string {
	value = strings.TrimSpace(value)
	if len(value) < 8 {
		return "redacted"
	}
	return value[:4] + "…" + value[len(value)-4:]
}

func (s *Store) ActiveHold(ctx context.Context, targetType, targetID, scope string) (bool, error) {
	if s == nil || s.pool == nil {
		return false, nil
	}
	var blocked bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.risk_holds WHERE target_type=$1 AND target_id=$2::uuid AND lifted_at IS NULL AND expires_at>now() AND scope IN ($3,'all_sensitive'))`, targetType, targetID, scope).Scan(&blocked)
	return blocked, err
}
