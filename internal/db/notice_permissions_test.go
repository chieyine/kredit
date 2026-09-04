package db

import (
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestNoticeAcknowledgementRuntimePermissions(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("KREDIT_INTEGRATION=1 and DATABASE_URL are required")
	}
	pool, err := Open(t.Context(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, role := range []string{"kredit_app", "kredit_worker"} {
		t.Run(role, func(t *testing.T) {
			tx, err := pool.Begin(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(t.Context()) }()
			if _, err = tx.Exec(t.Context(), "SET LOCAL ROLE "+role); err != nil {
				t.Fatal(err)
			}
			var canRead, canInsert, canUpdate, canDelete bool
			if err = tx.QueryRow(t.Context(), `SELECT has_table_privilege(current_user,'app.collection_notice_acknowledgements','SELECT'),has_table_privilege(current_user,'app.collection_notice_acknowledgements','INSERT'),has_table_privilege(current_user,'app.collection_notice_acknowledgements','UPDATE'),has_table_privilege(current_user,'app.collection_notice_acknowledgements','DELETE')`).Scan(&canRead, &canInsert, &canUpdate, &canDelete); err != nil {
				t.Fatal(err)
			}
			if !canRead || canInsert != (role == "kredit_app") || canUpdate || canDelete {
				t.Fatalf("unsafe acknowledgement privileges: read=%v insert=%v update=%v delete=%v", canRead, canInsert, canUpdate, canDelete)
			}
			if role == "kredit_worker" {
				if _, err = tx.Exec(t.Context(), `SELECT set_config('app.current_user_id','00000000-0000-0000-0000-000000000001',true)`); err != nil {
					t.Fatal(err)
				}
				_, err = tx.Exec(t.Context(), `INSERT INTO app.collection_notice_acknowledgements(schedule_item_id,buyer_user_id,notification_id,receipt_channel,receipt_event_id) VALUES('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001','email','forged')`)
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
					t.Fatalf("worker must be denied before it can fabricate acknowledgement evidence: %v", err)
				}
			}
		})
	}
}
