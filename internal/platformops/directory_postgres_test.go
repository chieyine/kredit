package platformops

import (
	"os"
	"testing"

	"kredit/internal/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUserDirectoryIncludesCurrentControlVersion(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("isolated integration database required")
	}
	pool, err := pgxpool.New(t.Context(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var id string
	var version int64
	if err := pool.QueryRow(t.Context(), `SELECT id::text,version FROM app.users ORDER BY version DESC,id LIMIT 1`).Scan(&id, &version); err != nil {
		t.Fatal(err)
	}
	actor := uuid.NewString()
	if _, err := pool.Exec(t.Context(), `INSERT INTO app.users(id,normalized_email) VALUES($1::uuid,$2)`, actor, actor+"@directory.test"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_admin',$1::uuid,'Directory verification fixture')`, actor); err != nil {
		t.Fatal(err)
	}
	users, err := NewStore(pool).Users(db.WithTenantContext(t.Context(), actor, ""), id, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].ID != id || users[0].Version != version || version < 1 {
		t.Fatalf("directory lost current user version: %+v, want %d", users, version)
	}
}
