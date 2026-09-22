package db

import (
	"context"

	"kredit/internal/platform/txcleanup"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ScopedDatabase gives legacy single-statement repositories the same
// transaction-local identity boundary as repositories that explicitly begin
// tenant transactions. It never derives identity from a target row.
type ScopedDatabase struct{ Pool *pgxpool.Pool }

func (p *ScopedDatabase) Begin(ctx context.Context) (pgx.Tx, error) {
	return p.BeginTx(ctx, pgx.TxOptions{})
}

func (p *ScopedDatabase) BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	tx, err := p.Pool.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	identity, _ := TenantFromContext(ctx)
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, identity.UserID, identity.OrganizationID); err != nil {
		return nil, RollbackFailure(ctx, tx, err)
	}
	return tx, nil
}

func (p *ScopedDatabase) Exec(ctx context.Context, sql string, args ...any) (tag pgconn.CommandTag, err error) {
	tx, err := p.Begin(ctx)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	defer txcleanup.Finish(ctx, tx, &err)
	tag, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		return tag, err
	}
	return tag, tx.Commit(ctx)
}

func (p *ScopedDatabase) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	tx, err := p.Begin(ctx)
	if err != nil {
		return &scopedRow{err: err}
	}
	return &scopedRow{row: tx.QueryRow(ctx, sql, args...), tx: tx, ctx: ctx}
}

type scopedRow struct {
	row pgx.Row
	tx  pgx.Tx
	ctx context.Context
	err error
}

func (r *scopedRow) Scan(dest ...any) (err error) {
	if r.err != nil {
		return r.err
	}
	defer txcleanup.Finish(r.ctx, r.tx, &err)
	if err := r.row.Scan(dest...); err != nil {
		return err
	}
	return r.tx.Commit(r.ctx)
}

func (p *ScopedDatabase) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	tx, err := p.Begin(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, RollbackFailure(ctx, tx, err)
	}
	return &scopedRows{Rows: rows, tx: tx, ctx: ctx}, nil
}

type scopedRows struct {
	pgx.Rows
	tx     pgx.Tx
	ctx    context.Context
	closed bool
	err    error
}

func (r *scopedRows) Close() {
	if r.closed {
		return
	}
	r.closed = true
	defer txcleanup.Finish(r.ctx, r.tx, &r.err)
	r.Rows.Close()
	if r.err = r.Rows.Err(); r.err != nil {
		return
	}
	r.err = r.tx.Commit(r.ctx)
}
func (r *scopedRows) Next() bool {
	if r.Rows.Next() {
		return true
	}
	r.Close()
	return false
}
func (r *scopedRows) Err() error {
	if r.err != nil {
		return r.err
	}
	return r.Rows.Err()
}
