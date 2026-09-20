package credit

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPurchasingRevocationBlocksCachedOwnerAndSerializesWrites(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated integration database required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	buyer, seller, successor, workspace, supplier, profile, requestID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, e := admin.Exec(ctx, sql, args...); e != nil {
			t.Fatal(e)
		}
	}
	for _, id := range []string{buyer, seller, successor} {
		exec(`INSERT INTO app.users(id,normalized_email) VALUES($1::uuid,$2)`, id, id+"@authority.test")
	}
	for _, id := range []string{workspace, supplier} {
		exec(`INSERT INTO app.organizations(id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Authority fixture','limited_company','Lagos','food')`, id)
	}
	for _, owner := range []string{buyer, successor} {
		exec(`INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active')`, workspace, owner)
	}
	exec(`INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active')`, supplier, seller)
	exec(`INSERT INTO app.businesses(id,organization_id,owner_user_id,legal_name,business_type,business_address,industry,status) VALUES($1::uuid,$2::uuid,$3::uuid,'Authority fixture','limited_company','Lagos','food','verified')`, profile, workspace, buyer)
	exec(`INSERT INTO app.credit_requests(id,supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,10000,'Goods',current_date+7,now()+interval '7 days','SENT',$5::uuid)`, requestID, supplier, buyer, profile, seller)
	payload, err := json.Marshal(View{Request: CreditRequest{ID: requestID, SupplierOrganizationID: supplier, BuyerUserID: buyer, BuyerBusinessID: profile, CreatedBy: seller, State: Sent, Version: 1}})
	if err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO app.credit_aggregate_snapshots(credit_request_id,supplier_organization_id,buyer_user_id,aggregate,version) VALUES($1,$2,$3,$4,1)`, requestID, supplier, buyer, payload)
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	runtime, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	repo := NewPostgresStore(runtime, nil)
	if _, err = repo.GetForBuyer(requestID, buyer); err != nil {
		t.Fatalf("active owner cannot read: %v", err)
	}
	writer, err := runtime.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Rollback(ctx)
	if _, err = writer.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, buyer, supplier); err != nil {
		t.Fatal(err)
	}
	if tag, e := writer.Exec(ctx, `UPDATE app.credit_requests SET goods_description=goods_description WHERE id=$1::uuid`, requestID); e != nil || tag.RowsAffected() != 1 {
		t.Fatalf("authorized write failed: %v", e)
	}
	revoker, err := admin.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = revoker.Exec(ctx, `SET LOCAL lock_timeout='100ms'`); err != nil {
		t.Fatal(err)
	}
	_, err = revoker.Exec(ctx, `UPDATE app.memberships SET status='removed' WHERE organization_id=$1::uuid AND user_id=$2::uuid`, workspace, buyer)
	var lockError *pgconn.PgError
	if !errors.As(err, &lockError) || lockError.Code != "55P03" {
		t.Fatalf("removal overtook an authorized write: %v", err)
	}
	_ = revoker.Rollback(ctx)
	if err = writer.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	exec(`UPDATE app.memberships SET status='removed' WHERE organization_id=$1::uuid AND user_id=$2::uuid`, workspace, buyer)
	if _, err = repo.GetForBuyer(requestID, buyer); err == nil {
		t.Fatal("cached purchasing record exposed after removal")
	}
	if _, err = repo.Review(requestID, buyer); err == nil {
		t.Fatal("removed owner reviewed a cached purchase")
	}
	if _, err = repo.GetForSupplier(requestID, supplier); err != nil {
		t.Fatalf("removal hid the supplier's receivable: %v", err)
	}
	for _, role := range []string{"kredit_app", "kredit_worker"} {
		tx, e := admin.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = tx.Exec(ctx, `SET LOCAL ROLE `+role); e != nil {
			t.Fatal(e)
		}
		if _, e = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, buyer, supplier); e != nil {
			t.Fatal(e)
		}
		for _, table := range []string{"credit_requests", "credit_aggregate_snapshots"} {
			var count int
			field := "id"
			if table == "credit_aggregate_snapshots" {
				field = "credit_request_id"
			}
			if e = tx.QueryRow(ctx, `SELECT count(*) FROM app.`+table+` WHERE `+field+`::text=$1`, requestID).Scan(&count); e != nil {
				t.Fatal(e)
			}
			want := 0
			if role == "kredit_worker" {
				want = 1
			}
			if count != want {
				t.Fatalf("%s %s visible=%d want=%d", role, table, count, want)
			}
		}
		_ = tx.Rollback(ctx)
	}
	var status string
	if err = admin.QueryRow(ctx, `SELECT status FROM app.users WHERE id=$1::uuid`, buyer).Scan(&status); err != nil || status != "active" {
		t.Fatalf("business removal changed personal account: %s %v", status, err)
	}
}
