package config

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kredit/internal/platformsettings"
)

// WatchConnections reports this specific process boot and gracefully stops it
// when saved settings must be applied. Restarting is the supervisor's job.
func WatchConnections(ctx context.Context, applied Config, settings platformsettings.Service, pool *pgxpool.Pool, process string, restart func()) {
	if pool == nil || settings == nil || restart == nil || (process != "api" && process != "worker") {
		return
	}
	if applied.AdminConnectionVersions == nil {
		applied.AdminConnectionVersions = map[string]int{}
	}
	bootID, err := uuid.NewRandom()
	if err != nil {
		slog.Error("runtime boot identity unavailable", "process", process, "error", err)
		restart()
		return
	}
	instance := strings.TrimSpace(os.Getenv("RUNTIME_INSTANCE_ID"))
	if instance == "" {
		instance, _ = os.Hostname()
	}
	instance = boundedRuntimeLabel(instance, "unnamed")
	// Container builds use a source archive without .git. The release image
	// supplies its revision explicitly; native Git builds retain VCS metadata.
	revision := boundedRuntimeLabel(os.Getenv("APP_REVISION"), "unknown")
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				revision = boundedRuntimeLabel(setting.Value, "unknown")
			}
		}
	}
	version := boundedRuntimeLabel(applied.Version, "unknown")
	versions, err := json.Marshal(applied.AdminConnectionVersions)
	if err != nil {
		slog.Error("runtime configuration versions could not be encoded", "process", process, "error", err)
		restart()
		return
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if _, err := pool.Exec(cleanupCtx, `UPDATE app.runtime_process_instances SET state='stopped',updated_at=clock_timestamp() WHERE boot_id=$1::uuid AND process=$2`, bootID.String(), process); err != nil {
			slog.Error("runtime stop status could not be recorded", "process", process, "error", err)
		}
	}()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	var lastPrune time.Time
	for {
		if ctx.Err() != nil {
			return
		}
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		next, err := ApplyStoredConnections(checkCtx, applied, settings, "", nil)
		status := "current"
		changed := err == nil && !reflect.DeepEqual(next.AdminConnectionVersions, applied.AdminConnectionVersions)
		if err != nil {
			status = "saved_configuration_unavailable"
		} else if changed {
			status = "restart_required"
			if applied.AdminConfigAutoApply {
				status = "applying"
			}
		}
		_, writeErr := pool.Exec(checkCtx, `
			INSERT INTO app.runtime_process_instances(boot_id,process,instance_name,version,revision,versions,state,updated_at)
			VALUES($1::uuid,$2,$3,$4,$5,$6::jsonb,$7,clock_timestamp())
			ON CONFLICT(boot_id) DO UPDATE SET versions=EXCLUDED.versions,state=EXCLUDED.state,updated_at=EXCLUDED.updated_at`,
			bootID.String(), process, instance, version, revision, versions, status)
		cancel()
		if writeErr != nil && ctx.Err() == nil {
			slog.Error("runtime instance heartbeat failed", "process", process, "error", writeErr)
		}
		if changed && applied.AdminConfigAutoApply && writeErr == nil {
			restart()
			return
		}
		// Old boot records are diagnostic data, not permanent audit evidence.
		// Bound each cleanup and let each runtime role prune only its own kind.
		if time.Since(lastPrune) >= time.Hour && ctx.Err() == nil {
			pruneCtx, pruneCancel := context.WithTimeout(ctx, 2*time.Second)
			_, pruneErr := pool.Exec(pruneCtx, `DELETE FROM app.runtime_process_instances WHERE boot_id IN (SELECT boot_id FROM app.runtime_process_instances WHERE process=$1 AND updated_at<now()-interval '7 days' ORDER BY updated_at LIMIT 1000)`, process)
			pruneCancel()
			lastPrune = time.Now()
			if pruneErr != nil && ctx.Err() == nil {
				slog.Error("runtime instance history cleanup failed", "process", process, "error", pruneErr)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func boundedRuntimeLabel(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	runes := []rune(value)
	if len(runes) > 128 {
		return string(runes[:128])
	}
	return value
}
