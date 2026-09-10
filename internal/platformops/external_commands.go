package platformops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"kredit/internal/access"

	"github.com/jackc/pgx/v5"
)

func ExternalCommand(kind string) bool {
	return kind == "retry_collection" || kind == "cancel_collection" || kind == "resolve_unknown_submission"
}

// BeginExternalCommand commits intent before a provider can act. Only the caller
// that creates the record receives permission to invoke the provider.
func (s *Store) BeginExternalCommand(ctx context.Context, actor string, in CommandInput) (Command, bool, error) {
	if s == nil || s.pool == nil || !ExternalCommand(in.Type) {
		return Command{}, false, errors.New("external operations service is unavailable")
	}
	if err := validateCommand(in, true); err != nil {
		return Command{}, false, err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Command{}, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "external-operation:"+in.TargetID); err != nil {
		return Command{}, false, err
	}
	if err = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionOperateCollections); err != nil {
		return Command{}, false, err
	}
	encoded, err := json.Marshal(in)
	if err != nil {
		return Command{}, false, err
	}
	digest := sha256.Sum256(encoded)
	requestHash := hex.EncodeToString(digest[:])
	var id, savedHash string
	err = tx.QueryRow(ctx, `SELECT id::text,request_hash FROM app.operations_commands WHERE requested_by=$1::uuid AND idempotency_key=$2`, actor, in.IdempotencyKey).Scan(&id, &savedHash)
	if err == nil {
		if savedHash != requestHash {
			return Command{}, false, errors.New("idempotency key belongs to a different command")
		}
		return Command{ID: id, State: "PREVIEWED"}, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Command{}, false, err
	}
	var unfinished bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.operations_commands c WHERE c.target_id=$1 AND c.command_type=$2 AND ($2<>'resolve_unknown_submission' OR c.created_at>now()-interval '5 minutes') AND NOT EXISTS(SELECT 1 FROM app.operations_command_events e WHERE e.command_id=c.id AND e.state='APPLIED'))`, in.TargetID, in.Type).Scan(&unfinished); err != nil {
		return Command{}, false, err
	}
	if unfinished {
		return Command{}, false, errors.New("a previous operation needs reconciliation before another attempt")
	}
	preview, err := s.previewCommand(ctx, in, tx)
	if err != nil {
		return Command{}, false, err
	}
	if preview.CurrentVersion != in.ExpectedVersion {
		return Command{}, false, errors.New("version conflict before provider operation")
	}
	impact, err := json.Marshal(preview.Impact)
	if err != nil {
		return Command{}, false, err
	}
	err = tx.QueryRow(ctx, `INSERT INTO app.operations_commands(command_type,target_type,target_id,organization_id,requested_by,reason,expected_version,idempotency_key,impact_preview,correlation_id,request_hash) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5::uuid,$6,$7,$8,$9,$10,$11) RETURNING id::text,created_at`, in.Type, in.TargetType, in.TargetID, in.OrganizationID, actor, in.Reason, in.ExpectedVersion, in.IdempotencyKey, impact, in.CorrelationID, requestHash).Scan(&preview.ID, &preview.CreatedAt)
	if err != nil {
		return Command{}, false, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.operations_command_events(command_id,state,result) VALUES($1::uuid,'PREVIEWED','{"provider_result":"pending"}')`, preview.ID); err != nil {
		return Command{}, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Command{}, false, err
	}
	preview.State, preview.Reason, preview.CorrelationID = "PREVIEWED", in.Reason, in.CorrelationID
	return preview, true, nil
}

// FinishExternalCommand stores evidence after the provider call, even if the
// original HTTP request disconnected. It cannot create a command retroactively.
func (s *Store) FinishExternalCommand(ctx context.Context, actor string, in CommandInput) (Command, error) {
	if s == nil || s.pool == nil || !ExternalCommand(in.Type) || len(in.ExternalResult) == 0 {
		return Command{}, errors.New("recorded intent and provider evidence are required")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Command{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "external-operation:"+in.TargetID); err != nil {
		return Command{}, err
	}
	encoded, err := json.Marshal(in)
	if err != nil {
		return Command{}, err
	}
	digest := sha256.Sum256(encoded)
	var command Command
	var savedHash string
	err = tx.QueryRow(ctx, `SELECT c.id::text,c.command_type,c.target_type,c.target_id,c.reason,c.expected_version,c.impact_preview,e.state,e.result,c.correlation_id,c.created_at,c.request_hash FROM app.operations_commands c JOIN LATERAL (SELECT state,result FROM app.operations_command_events WHERE command_id=c.id ORDER BY occurred_at DESC,id DESC LIMIT 1)e ON true WHERE c.requested_by=$1::uuid AND c.idempotency_key=$2`, actor, in.IdempotencyKey).Scan(&command.ID, &command.Type, &command.TargetType, &command.TargetID, &command.Reason, &command.CurrentVersion, &command.Impact, &command.State, &command.Result, &command.CorrelationID, &command.CreatedAt, &savedHash)
	if err != nil {
		return Command{}, err
	}
	if savedHash != hex.EncodeToString(digest[:]) {
		return Command{}, errors.New("provider result does not match recorded intent")
	}
	if command.State == "APPLIED" {
		return command, nil
	}
	result, err := json.Marshal(in.ExternalResult)
	if err != nil {
		return Command{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.operations_command_events(command_id,state,result) VALUES($1::uuid,'APPLIED',$2::jsonb)`, command.ID, result); err != nil {
		return Command{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Command{}, err
	}
	command.State, command.Result = "APPLIED", in.ExternalResult
	return command, nil
}

func CommandPermission(commandType string) (access.Permission, bool) {
	switch commandType {
	case "retry_job", "retry_webhook", "retry_document_scan":
		return access.PermissionOperateJobs, true
	case "request_reconciliation", "resolve_unknown_submission", "retry_collection", "cancel_collection":
		return access.PermissionOperateCollections, true
	case "suspend_user", "restore_user", "suspend_organization", "restore_organization":
		return access.PermissionSuspendAccounts, true
	case "place_risk_hold", "lift_risk_hold":
		return access.PermissionManageRiskHold, true
	default:
		return "", false
	}
}
