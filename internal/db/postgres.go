package db

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	if databaseURL == "" {
		return nil, errors.New("database URL is required")
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	config.MaxConns = 20
	config.MinConns = 2
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
	setRuntimeDefault(config.ConnConfig.RuntimeParams, "statement_timeout", "30000")
	setRuntimeDefault(config.ConnConfig.RuntimeParams, "lock_timeout", "5000")
	setRuntimeDefault(config.ConnConfig.RuntimeParams, "idle_in_transaction_session_timeout", "30000")
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
	var appOutbox, ledgerTransactions, riverJobs string
	if err := p.inner.QueryRow(ctx, `SELECT COALESCE(to_regclass('app.outbox_events')::text,''), COALESCE(to_regclass('ledger.transactions')::text,''), COALESCE(to_regclass('jobs.river_job')::text,'')`).Scan(&appOutbox, &ledgerTransactions, &riverJobs); err != nil {
		return fmt.Errorf("check postgres schema: %w", err)
	}
	if appOutbox == "" || ledgerTransactions == "" || riverJobs == "" {
		return fmt.Errorf("database migrations are incomplete (outbox=%t ledger=%t river=%t)", appOutbox != "", ledgerTransactions != "", riverJobs != "")
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
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
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
