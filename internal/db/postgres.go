package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"kredit/internal/platform/txcleanup"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is the single PostgreSQL boundary used by API and worker processes.
// Domain packages should receive a transaction from this boundary rather than
// opening their own connections or mutating shared in-memory state.
type Pool struct {
	inner *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Pool, error) {
	return open(ctx, databaseURL, "")
}

// OpenAsRole opens a pool whose PostgreSQL session is constrained to the
// supplied NOLOGIN runtime role. The role is set as a startup parameter so no
// query can run with the more privileged login role before SET ROLE executes.
func OpenAsRole(ctx context.Context, databaseURL, runtimeRole string) (*Pool, error) {
	if runtimeRole == "" {
		return nil, errors.New("runtime database role is required")
	}
	return open(ctx, databaseURL, runtimeRole)
}

func open(ctx context.Context, databaseURL, runtimeRole string) (*Pool, error) {
	if databaseURL == "" {
		return nil, errors.New("database URL is required")
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	maxConns := int32(20)
	if val := os.Getenv("DATABASE_MAX_CONNS"); val != "" {
		n, err := strconv.ParseInt(val, 10, 32)
		if err != nil || n <= 0 {
			return nil, errors.New("DATABASE_MAX_CONNS must be a positive 32-bit integer")
		}
		maxConns = int32(n)
	}
	minConns := min(int32(2), maxConns)
	if val := os.Getenv("DATABASE_MIN_CONNS"); val != "" {
		n, err := strconv.ParseInt(val, 10, 32)
		if err != nil || n < 0 {
			return nil, errors.New("DATABASE_MIN_CONNS must be a non-negative 32-bit integer")
		}
		minConns = int32(n)
	}
	if minConns > maxConns {
		return nil, errors.New("DATABASE_MIN_CONNS cannot exceed DATABASE_MAX_CONNS")
	}
	config.MaxConns = maxConns
	config.MinConns = minConns
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 30 * time.Second
	config.ConnConfig.ConnectTimeout = 10 * time.Second
	// Repository interfaces created before request-scoped contexts were added
	// still contain a few background-context calls. Database-side limits keep
	// those calls from holding a connection or lock indefinitely.
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = make(map[string]string)
	}
	if runtimeRole != "" {
		// The process-selected role wins over any role supplied in the URL.
		// Component entrypoints pass constants, not deployment input.
		config.ConnConfig.RuntimeParams["role"] = runtimeRole
	}
	if err := configureRuntimeTimeouts(config.ConnConfig.RuntimeParams); err != nil {
		return nil, err
	}
	// Verify every physical connection, including replacements after startup.
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		verifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := verifyRuntimeTimeouts(verifyCtx, conn, config.ConnConfig.RuntimeParams); err != nil {
			return err
		}
		if runtimeRole != "" {
			return verifyRuntimeRole(verifyCtx, conn, runtimeRole)
		}
		return nil
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Pool{inner: pool}, nil
}

func setRuntimeDefault(values map[string]string, name, value string) {
	if _, configured := values[name]; !configured {
		values[name] = value
	}
}

func (p *Pool) Close() {
	if p != nil && p.inner != nil {
		p.inner.Close()
	}
}

// Raw exposes the configured pgx pool to infrastructure integrations such as
// River. Application/domain packages should prefer Pool's transaction helpers.
func (p *Pool) Raw() *pgxpool.Pool {
	if p == nil {
		return nil
	}
	return p.inner
}

func (p *Pool) Ping(ctx context.Context) error {
	if p == nil || p.inner == nil {
		return errors.New("postgres pool is not configured")
	}
	return p.inner.Ping(ctx)
}

// CheckSchema fails startup when the application database has not been
// migrated to the durable runtime contract. Checking concrete objects catches
// a healthy-but-empty PostgreSQL instance before it can serve requests.
func (p *Pool) CheckSchema(ctx context.Context) error {
	if p == nil || p.inner == nil {
		return errors.New("postgres pool is not configured")
	}
	var appOutbox, ledgerTransactions, riverJobs, runtimeInstances string
	if err := p.inner.QueryRow(ctx, `SELECT COALESCE(to_regclass('app.outbox_events')::text,''), COALESCE(to_regclass('ledger.transactions')::text,''), COALESCE(to_regclass('jobs.river_job')::text,''), COALESCE(to_regclass('app.runtime_process_instances')::text,'')`).Scan(&appOutbox, &ledgerTransactions, &riverJobs, &runtimeInstances); err != nil {
		return fmt.Errorf("check postgres schema: %w", err)
	}
	if appOutbox == "" || ledgerTransactions == "" || riverJobs == "" || runtimeInstances == "" {
		return fmt.Errorf("database migrations are incomplete (outbox=%t ledger=%t river=%t runtime_instances=%t)", appOutbox != "", ledgerTransactions != "", riverJobs != "", runtimeInstances != "")
	}
	return nil
}

func (p *Pool) Exec(ctx context.Context, sql string, args ...any) error {
	if p == nil || p.inner == nil {
		return errors.New("postgres pool is not configured")
	}
	_, err := p.inner.Exec(ctx, sql, args...)
	return err
}

func (p *Pool) Begin(ctx context.Context) (pgx.Tx, error) {
	if p == nil || p.inner == nil {
		return nil, errors.New("postgres pool is not configured")
	}
	return p.inner.Begin(ctx)
}

func (p *Pool) WithTx(ctx context.Context, fn func(pgx.Tx) error) (err error) {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer txcleanup.Finish(ctx, tx, &err)
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// WithTenantTx establishes the request identity on the transaction-local
// settings consumed by PostgreSQL RLS policies. SET LOCAL ensures pooled
// connections cannot retain a previous tenant's context.
func (p *Pool) WithTenantTx(ctx context.Context, userID, organizationID string, fn func(pgx.Tx) error) error {
	return p.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT set_config('app.current_user_id', $1, true), set_config('app.current_organization_id', $2, true)`, userID, organizationID); err != nil {
			return fmt.Errorf("set tenant context: %w", err)
		}
		return fn(tx)
	})
}
