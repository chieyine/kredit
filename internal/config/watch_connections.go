package config

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"kredit/internal/platformsettings"
)

// WatchConnections lets supervised API/worker processes gracefully restart to
// adopt validated owner settings. No shell commands or credentials leave the
// process. The production supervisor restarts the stopped service.
func WatchConnections(ctx context.Context, applied Config, settings platformsettings.Service, pool *pgxpool.Pool, process string, restart func()) {
	if pool == nil || settings == nil || restart == nil || (process != "api" && process != "worker") {
		return
	}
	if applied.AdminConnectionVersions == nil {
		applied.AdminConnectionVersions = map[string]int{}
	}
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	versions, _ := json.Marshal(applied.AdminConnectionVersions)
	for {
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
		_, writeErr := pool.Exec(checkCtx, `INSERT INTO app.runtime_process_status(process,versions,state,updated_at) VALUES($1,$2::jsonb,$3,now()) ON CONFLICT(process) DO UPDATE SET versions=EXCLUDED.versions,state=EXCLUDED.state,updated_at=now()`, process, versions, status)
		cancel()
		if changed && applied.AdminConfigAutoApply && writeErr == nil {
			restart()
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
