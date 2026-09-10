package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"kredit/internal/platformops"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSaleRiskHoldsRespectPartyScopeExpiryAndOutage(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("isolated integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var actor string
	if err = pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, "hold-"+uuid.NewString()+"@example.test").Scan(&actor); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := pool.Exec(ctx, `DELETE FROM app.risk_holds WHERE created_by=$1::uuid`, actor); err != nil {
			t.Error(err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM app.users WHERE id=$1::uuid`, actor); err != nil {
			t.Error(err)
		}
	}()
	supplier, buyer := uuid.NewString(), uuid.NewString()
	server := &Server{runtime: &Runtime{PlatformOps: platformops.NewStore(pool)}}
	assert := func(scope string, allowed bool, status int) {
		t.Helper()
		rec := httptest.NewRecorder()
		ok := server.requireSaleRiskClear(rec, httptest.NewRequest("POST", "/", nil), supplier, buyer, scope)
		if ok != allowed || (!ok && rec.Code != status) {
			t.Fatalf("scope %s: allowed=%v status=%d body=%s", scope, ok, rec.Code, rec.Body.String())
		}
	}
	var hold string
	if err = pool.QueryRow(ctx, `INSERT INTO app.risk_holds(target_type,target_id,scope,reason,expires_at,created_by) VALUES('buyer',$1::uuid,'credit','Synthetic safety review',$2,$3::uuid) RETURNING id::text`, buyer, time.Now().Add(time.Hour), actor).Scan(&hold); err != nil {
		t.Fatal(err)
	}
	assert("credit", false, http.StatusLocked)
	assert("release", true, 0)
	if _, err = pool.Exec(ctx, `UPDATE app.risk_holds SET created_at=now()-interval '2 hours',expires_at=now()-interval '1 second' WHERE id=$1::uuid`, hold); err != nil {
		t.Fatal(err)
	}
	assert("credit", true, 0)
	if _, err = pool.Exec(ctx, `INSERT INTO app.risk_holds(target_type,target_id,scope,reason,expires_at,created_by) VALUES('supplier',$1::uuid,'all_sensitive','Synthetic safety review',$2,$3::uuid)`, supplier, time.Now().Add(time.Hour), actor); err != nil {
		t.Fatal(err)
	}
	assert("credit", false, http.StatusLocked)
	assert("release", false, http.StatusLocked)
	closed, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	closed.Close()
	server.runtime.PlatformOps = platformops.NewStore(closed)
	assert("credit", false, http.StatusServiceUnavailable)
}
