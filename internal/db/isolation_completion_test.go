package db

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"os"
	"testing"
)

// Each table has a real row. An empty result cannot accidentally make the
// negative checks pass, and child rows must inherit their parent's boundary.
func TestRemainingTenantTablesRejectUnrelatedRuntimeIdentity(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated integration database required")
	}
	p, err := Open(t.Context(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	tx, err := p.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, e := tx.Exec(t.Context(), sql, args...); e != nil {
			t.Fatalf("%s: %v", sql, e)
		}
	}
	user, other, org, otherOrg, privacy, recovery, correction, request, agreement, transaction, obligation, schedule, dispute := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	notification := uuid.NewString()
	for _, id := range []string{user, other} {
		exec(`INSERT INTO app.users(id,normalized_email) VALUES($1::uuid,$2)`, id, id+"@isolation.test")
	}
	for _, id := range []string{org, otherOrg} {
		exec(`INSERT INTO app.organizations(id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Isolation fixture','limited_company','test','test')`, id)
	}
	exec(`INSERT INTO app.privacy_requests(id,requester_user_id,request_type) VALUES($1::uuid,$2::uuid,'ACCESS')`, privacy, user)
	exec(`INSERT INTO app.privacy_request_events(request_id,event_type,actor_reference) VALUES($1::uuid,'test','test')`, privacy)
	exec(`INSERT INTO app.privacy_exports(request_id,object_reference,content_sha256,payload,expires_at) VALUES($1::uuid,'test',repeat('a',64),'{}',now()+interval '1 day')`, privacy)
	exec(`INSERT INTO app.processing_restrictions(user_id,privacy_request_id,scope,reason) VALUES($1::uuid,$2::uuid,'optional','test')`, user, privacy)
	exec(`INSERT INTO app.account_recovery_codes(user_id,code_hash,code_hint,expires_at) VALUES($1::uuid,'test','1234',now()+interval '1 day')`, user)
	exec(`INSERT INTO app.account_recovery_requests(id,target_user_id,requested_channel,request_fingerprint) VALUES($1::uuid,$2::uuid,'email','test')`, recovery, user)
	exec(`INSERT INTO app.account_recovery_evidence(request_id,factor_type,evidence_hash) VALUES($1::uuid,'recovery_code','test')`, recovery)
	exec(`INSERT INTO app.account_recovery_events(request_id,event_type,actor_reference) VALUES($1::uuid,'test','test')`, recovery)
	exec(`INSERT INTO app.correction_requests(id,organization_id,subject_type,subject_id,requested_by,reason,state) VALUES($1::uuid,$2::uuid,'obligation',$1::uuid,$3::uuid,'test','OPEN')`, correction, org, user)
	exec(`INSERT INTO app.correction_decisions(request_id,reviewer_id,outcome,reason) VALUES($1::uuid,$2::uuid,'REJECTED','test')`, correction, other)
	exec(`INSERT INTO app.notification_preferences(recipient_id) VALUES($1::uuid)`, user)
	exec(`INSERT INTO app.notifications(id,recipient_id,channel,template,template_version,event_reference,state,body,scheduled_at) VALUES($3::uuid,$1::uuid,'email','test','v1',$2,'scheduled','test',now())`, user, uuid.NewString(), notification)
	exec(`INSERT INTO app.credit_requests(id,supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,gen_random_uuid(),3000,'test',current_date+30,now()+interval '30 days','ACTIVE',$3::uuid)`, request, org, user)
	exec(`INSERT INTO app.agreement_versions(id,credit_request_id,version,canonical_json,document_hash,terms_version,privacy_version,created_by) VALUES($1::uuid,$2::uuid,1,'{}',$1,'v1','v1',$3::uuid)`, agreement, request, user)
	exec(`INSERT INTO ledger.transactions(id,event_type,reference_type,reference_id,idempotency_key,effective_at) VALUES($1::uuid,'test','credit_request',$2::uuid,$1,now())`, transaction, request)
	exec(`INSERT INTO app.obligations(id,credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at) SELECT $1::uuid,$2::uuid,$3::uuid,$4::uuid,buyer_business_id,3000,'NGN','ACTIVE','CURRENT',3000,0,$5::uuid,now() FROM app.credit_requests WHERE id=$2::uuid`, obligation, request, agreement, org, transaction)
	exec(`INSERT INTO app.repayment_schedules(id,obligation_id,schedule_type,timezone,allocation_policy,cadence,grace_hours,status) VALUES($1::uuid,$2::uuid,'equal','Africa/Lagos','due_date_order','custom',0,'ACTIVE')`, schedule, obligation)
	exec(`INSERT INTO app.schedule_items(schedule_id,sequence,principal_due_kobo,due_at,grace_hours,collection_at,state) VALUES($1::uuid,1,3000,now(),0,now(),'OPEN')`, schedule)
	exec(`INSERT INTO app.disputes(id,obligation_id,supplier_organization_id,buyer_user_id,opened_by,total_disputed_kobo,remaining_disputed_kobo,reason,state,collection_effect) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$4::uuid,100,100,'test','OPEN','CONTESTED_ONLY')`, dispute, obligation, org, user)
	exec(`SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, user, org)
	exec(`INSERT INTO app.dispute_evidence(dispute_id,submitted_by,statement) VALUES($1::uuid,$2::uuid,'test')`, dispute, user)
	exec(`INSERT INTO app.dispute_decisions(dispute_id,reviewer_id,outcome,valid_principal_kobo,adjustment_kobo,remaining_disputed_kobo,reason) VALUES($1::uuid,$2::uuid,'REJECTED',3000,0,0,'test')`, dispute, other)
	rows := []struct{ table, key, id string }{
		{"privacy_requests", "id", privacy}, {"privacy_request_events", "request_id", privacy}, {"privacy_exports", "request_id", privacy}, {"processing_restrictions", "user_id", user},
		{"account_recovery_codes", "user_id", user}, {"account_recovery_requests", "id", recovery}, {"account_recovery_evidence", "request_id", recovery}, {"account_recovery_events", "request_id", recovery},
		{"correction_requests", "id", correction}, {"correction_decisions", "request_id", correction}, {"notification_preferences", "recipient_id", user}, {"notifications", "id", notification},
		{"repayment_schedules", "id", schedule}, {"schedule_items", "schedule_id", schedule}, {"disputes", "id", dispute}, {"dispute_evidence", "dispute_id", dispute}, {"dispute_decisions", "dispute_id", dispute},
	}
	for _, role := range []string{"kredit_app", "kredit_worker"} {
		exec(`SET LOCAL ROLE ` + role)
		for _, subject := range []struct {
			user, org string
			want      int
		}{{user, org, 1}, {other, otherOrg, 0}, {"", "", 0}} {
			exec(`SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, subject.user, subject.org)
			for _, row := range rows {
				var count int
				if e := tx.QueryRow(t.Context(), `SELECT count(*) FROM app.`+row.table+` WHERE `+row.key+`=$1::uuid`, row.id).Scan(&count); e != nil || count != subject.want {
					t.Fatalf("%s %s subject=%q count=%d want=%d: %v", role, row.table, subject.user, count, subject.want, e)
				}
				if subject.want == 0 {
					exec(`SAVEPOINT forbidden_update`)
					tag, updateErr := tx.Exec(t.Context(), `UPDATE app.`+row.table+` SET `+row.key+`=`+row.key+` WHERE `+row.key+`=$1::uuid`, row.id)
					var denied *pgconn.PgError
					if updateErr != nil && (!errors.As(updateErr, &denied) || denied.Code != "42501") {
						t.Fatalf("unexpected update failure on %s: %v", row.table, updateErr)
					}
					if updateErr == nil && tag.RowsAffected() != 0 {
						t.Fatalf("%s changed another subject's %s", role, row.table)
					}
					exec(`ROLLBACK TO SAVEPOINT forbidden_update`)
					exec(`RELEASE SAVEPOINT forbidden_update`)
				}
			}
		}
		exec(`RESET ROLE`)
	}
	exec(`INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_admin',$1::uuid,'Synthetic operations reviewer')`, other)
	exec(`SET LOCAL ROLE kredit_app`)
	exec(`SELECT set_config('app.current_user_id',$1,true)`, other)
	var open int
	if err := tx.QueryRow(t.Context(), `SELECT open_disputes FROM app.isolation_operations_counts()`).Scan(&open); err != nil || open < 1 {
		t.Fatalf("authorized aggregate unavailable: %d %v", open, err)
	}
	var reference string
	if err := tx.QueryRow(t.Context(), `SELECT id::text FROM app.dispute_reference_lookup($1)`, dispute).Scan(&reference); err != nil || reference != dispute {
		t.Fatalf("authorized exact lookup unavailable: %s %v", reference, err)
	}
	exec(`RESET ROLE`)
	exec(`UPDATE app.platform_role_assignments SET revoked_at=now() WHERE user_id=$1::uuid`, other)
	exec(`SET LOCAL ROLE kredit_app`)
	if err := tx.QueryRow(t.Context(), `SELECT count(*) FROM app.isolation_operations_counts()`).Scan(&open); err != nil || open != 0 {
		t.Fatalf("revoked reviewer retained aggregate access: %d %v", open, err)
	}
}
