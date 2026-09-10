package db

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRelationshipChoicesBelongToBuyerAndRemainAppendOnly(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("isolated integration database required")
	}
	database, err := Open(t.Context(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	tx, err := database.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	buyer, supplier, org, business := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(t.Context(), sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{buyer, supplier} {
		exec(`INSERT INTO app.users(id,normalized_email) VALUES($1::uuid,$2)`, id, id+"@example.test")
	}
	exec(`INSERT INTO app.organizations(id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Synthetic consent seller','limited_company','Test address','Test')`, org)
	exec(`INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active')`, org, supplier)
	exec(`INSERT INTO app.businesses(id,owner_user_id,legal_name,business_type,business_address,industry,status) VALUES($1::uuid,$2::uuid,'Synthetic consent buyer','limited_company','Test address','Test','verified')`, business, buyer)
	exec(`INSERT INTO app.trade_relationships(supplier_organization_id,buyer_business_id,status) VALUES($1::uuid,$2::uuid,'active')`, org, business)
	exec(`SET LOCAL ROLE kredit_app`)
	exec(`SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, buyer, org)
	insert := `INSERT INTO app.relationship_consents(buyer_user_id,supplier_organization_id,consent_type,version,evidence_hash,granted) VALUES($1::uuid,$2::uuid,'payment_reminders','v1','synthetic evidence',true)`
	exec(insert, buyer, org)
	denied := func(sql string, args ...any) {
		t.Helper()
		exec(`SAVEPOINT forbidden_consent_change`)
		_, err := tx.Exec(t.Context(), sql, args...)
		var pgerr *pgconn.PgError
		if !errors.As(err, &pgerr) || pgerr.Code != "42501" {
			t.Fatalf("expected permission denial, got %v", err)
		}
		exec(`ROLLBACK TO SAVEPOINT forbidden_consent_change`)
		exec(`RELEASE SAVEPOINT forbidden_consent_change`)
	}
	denied(`UPDATE app.relationship_consents SET granted=false WHERE buyer_user_id=$1::uuid`, buyer)
	denied(`DELETE FROM app.relationship_consents WHERE buyer_user_id=$1::uuid`, buyer)
	exec(`SELECT set_config('app.current_user_id',$1,true)`, supplier)
	var count int
	if err := tx.QueryRow(t.Context(), `SELECT count(*) FROM app.relationship_consents WHERE buyer_user_id=$1::uuid AND supplier_organization_id=$2::uuid`, buyer, org).Scan(&count); err != nil || count != 1 {
		t.Fatalf("supplier should read buyer choice: count=%d err=%v", count, err)
	}
	denied(insert, buyer, org)
	exec(`RESET ROLE`)
	exec(`SET LOCAL ROLE kredit_worker`)
	exec(`SELECT set_config('app.current_user_id',$1,true)`, buyer)
	denied(insert, buyer, org)
}
