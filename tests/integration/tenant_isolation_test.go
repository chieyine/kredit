//go:build integration

package integration

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	tenantA        = "00000000-0000-7000-8000-000000000010"
	tenantB        = "00000000-0000-7000-8000-0000000000b0"
	requestA       = "00000000-0000-7000-8000-000000000051"
	requestB       = "00000000-0000-7000-8000-0000000000b1"
	agreementA     = "00000000-0000-7000-8000-000000000061"
	agreementB     = "00000000-0000-7000-8000-0000000000b2"
	obligationA    = "00000000-0000-7000-8000-000000000071"
	obligationB    = "00000000-0000-7000-8000-0000000000b3"
	ledgerTxnB     = "00000000-0000-7000-8000-0000000000b4"
	appPassword    = "kredit-app-development-only"
	workerPassword = "kredit-worker-development-only"
)

func TestFinancialRowsAreTenantBoundAtDatabaseLayer(t *testing.T) {
	ctx := context.Background()
	admin := integrationPool(t, os.Getenv("DATABASE_URL"), "", "")
	defer admin.Close()
	seedSecondTenant(t, ctx, admin)

	app := integrationPool(t, os.Getenv("DATABASE_URL"), "kredit_app_login", appPassword)
	defer app.Close()

	t.Run("no tenant context fails closed", func(t *testing.T) {
		tx := beginRoleTx(t, ctx, app, "kredit_app")
		defer tx.Rollback(ctx)
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM app.obligations`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("expected no supplier obligations without tenant context, got %d", count)
		}
	})

	t.Run("tenant A sees A and not B", func(t *testing.T) {
		tx := beginTenantTx(t, ctx, app, "kredit_app", tenantA)
		defer tx.Rollback(ctx)
		assertVisible(t, ctx, tx, obligationA, true)
		assertVisible(t, ctx, tx, obligationB, false)

		command, err := tx.Exec(ctx, `UPDATE app.obligations SET outstanding_kobo=outstanding_kobo WHERE id=$1::uuid`, obligationB)
		if err != nil {
			t.Fatal(err)
		}
		if command.RowsAffected() != 0 {
			t.Fatalf("cross-tenant update changed %d rows", command.RowsAffected())
		}
	})

	t.Run("tenant B sees B and not A", func(t *testing.T) {
		tx := beginTenantTx(t, ctx, app, "kredit_app", tenantB)
		defer tx.Rollback(ctx)
		assertVisible(t, ctx, tx, obligationB, true)
		assertVisible(t, ctx, tx, obligationA, false)
	})
}

func TestWorkerIsTenantBoundAndCannotForgeBuyerEvidence(t *testing.T) {
	ctx := context.Background()
	admin := integrationPool(t, os.Getenv("DATABASE_URL"), "", "")
	defer admin.Close()
	seedSecondTenant(t, ctx, admin)

	worker := integrationPool(t, os.Getenv("DATABASE_URL"), "kredit_worker_login", workerPassword)
	defer worker.Close()

	tx := beginTenantTx(t, ctx, worker, "kredit_worker", tenantA)
	defer tx.Rollback(ctx)
	assertVisible(t, ctx, tx, obligationA, true)
	assertVisible(t, ctx, tx, obligationB, false)

	_, err := tx.Exec(ctx, `UPDATE app.agreement_acceptances SET accepted_at=accepted_at WHERE credit_request_id=$1::uuid`, requestA)
	if err == nil {
		t.Fatal("worker unexpectedly mutated buyer-originated agreement acceptance evidence")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Fatalf("expected worker evidence guard permission error 42501, got %v", err)
	}
}

func integrationPool(t *testing.T, rawURL, user, password string) *pgxpool.Pool {
	t.Helper()
	if rawURL == "" {
		t.Fatal("DATABASE_URL is required")
	}
	config, err := pgxpool.ParseConfig(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	if user != "" {
		config.ConnConfig.User = user
		config.ConnConfig.Password = password
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Fatalf("connect as %s: %v", user, err)
	}
	return pool
}

func beginRoleTx(t *testing.T, ctx context.Context, pool *pgxpool.Pool, role string) pgx.Tx {
	t.Helper()
	if role != "kredit_app" && role != "kredit_worker" {
		t.Fatalf("unsupported runtime role %q", role)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE "+role); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	return tx
}

func beginTenantTx(t *testing.T, ctx context.Context, pool *pgxpool.Pool, role, organizationID string) pgx.Tx {
	t.Helper()
	tx := beginRoleTx(t, ctx, pool, role)
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_user_id','',true), set_config('app.current_organization_id',$1,true)`, organizationID); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	return tx
}

func assertVisible(t *testing.T, ctx context.Context, tx pgx.Tx, obligationID string, expected bool) {
	t.Helper()
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM app.obligations WHERE id=$1::uuid`, obligationID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if (count == 1) != expected {
		t.Fatalf("obligation %s visibility=%d, expected visible=%v", obligationID, count, expected)
	}
}

func seedSecondTenant(t *testing.T, ctx context.Context, admin *pgxpool.Pool) {
	t.Helper()
	statements := []struct {
		name string
		sql  string
		args []any
	}{
		{
			name: "organization",
			sql: `INSERT INTO app.organizations
				(id,legal_name,trading_name,business_type,registration_info,business_address,industry,status)
			VALUES ($1::uuid,'Phase 2 Tenant B Ltd','Tenant B','limited_company','{"number":"PHASE2-B"}'::jsonb,'Lagos, Nigeria','pharmaceuticals','verified')
			ON CONFLICT (id) DO NOTHING`,
			args: []any{tenantB},
		},
		{
			name: "credit request",
			sql: `INSERT INTO app.credit_requests
			SELECT (jsonb_populate_record(NULL::app.credit_requests,
				to_jsonb(source_row) || jsonb_build_object(
					'id',$1::text,
					'supplier_organization_id',$2::text,
					'invoice_reference','PHASE2-TENANT-B',
					'state','DRAFT',
					'agreement_version_id',NULL,
					'acceptance_id',NULL,
					'release_id',NULL,
					'receipt_id',NULL,
					'obligation_id',NULL
				))).*
			FROM app.credit_requests source_row
			WHERE source_row.id=$3::uuid
			ON CONFLICT (id) DO NOTHING`,
			args: []any{requestB, tenantB, requestA},
		},
		{
			name: "agreement",
			sql: `INSERT INTO app.agreement_versions
			SELECT (jsonb_populate_record(NULL::app.agreement_versions,
				to_jsonb(source_row) || jsonb_build_object(
					'id',$1::text,
					'credit_request_id',$2::text
				))).*
			FROM app.agreement_versions source_row
			WHERE source_row.id=$3::uuid
			ON CONFLICT (id) DO NOTHING`,
			args: []any{agreementB, requestB, agreementA},
		},
		{
			name: "obligation",
			sql: `INSERT INTO app.obligations
			SELECT (jsonb_populate_record(NULL::app.obligations,
				to_jsonb(source_row) || jsonb_build_object(
					'id',$1::text,
					'credit_request_id',$2::text,
					'agreement_version_id',$3::text,
					'supplier_organization_id',$4::text,
					'ledger_transaction_id',$5::text
				))).*
			FROM app.obligations source_row
			WHERE source_row.id=$6::uuid
			ON CONFLICT (id) DO NOTHING`,
			args: []any{obligationB, requestB, agreementB, tenantB, ledgerTxnB, obligationA},
		},
	}
	for _, statement := range statements {
		if _, err := admin.Exec(ctx, statement.sql, statement.args...); err != nil {
			t.Fatalf("seed tenant B %s fixture: %v", statement.name, err)
		}
	}
}
