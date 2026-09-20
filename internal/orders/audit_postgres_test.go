package orders

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"kredit/internal/buyers"
	"kredit/internal/db"
	"kredit/internal/ledger"
	"kredit/internal/schedules"
)

// These fixtures use a privileged connection only for setup/assertions. Every
// operation under test uses APP_DATABASE_URL and the real non-owner login.
// Never run this fixture outside an explicitly named, disposable *_test DB.
type auditFixture struct {
	admin                                                       *pgxpool.Pool
	runtime                                                     *db.Pool
	org, buyerOrg, owner, sales, finance, other, buyer, profile string
	order                                                       [2]string
	obligation                                                  [2]string
	ctx                                                         context.Context
}

func newAuditFixture(t *testing.T) *auditFixture {
	t.Helper()
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated PostgreSQL integration required")
	}
	if os.Getenv("DATABASE_URL") == "" || os.Getenv("APP_DATABASE_URL") == "" {
		t.Fatal("both fixture owner and restricted application login are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	admin, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	var database string
	if err = admin.QueryRow(ctx, `SELECT current_database()`).Scan(&database); err != nil || !strings.HasSuffix(database, "_test") {
		t.Fatalf("refusing non-test database fixture: %v", err)
	}
	runtime, err := db.OpenAsRole(ctx, os.Getenv("APP_DATABASE_URL"), "kredit_app")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(runtime.Close)
	f := &auditFixture{admin: admin, runtime: runtime, ctx: ctx, org: uuid.NewString(), owner: uuid.NewString(), sales: uuid.NewString(), finance: uuid.NewString(), other: uuid.NewString(), buyer: uuid.NewString(), profile: uuid.NewString()}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := admin.Exec(ctx, q, args...); e != nil {
			t.Fatalf("fixture: %v", e)
		}
	}
	for _, id := range []string{f.owner, f.sales, f.finance, f.other, f.buyer} {
		exec(`INSERT INTO app.users(id,normalized_email) VALUES($1::uuid,$2)`, id, id+"@audit-regression.test")
	}
	exec(`INSERT INTO app.organizations(id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Audit supplier','limited_company','synthetic','test')`, f.org)
	for _, member := range []struct{ id, role string }{{f.owner, "owner"}, {f.sales, "sales"}, {f.finance, "finance"}} {
		exec(`INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,$3,'active')`, f.org, member.id, member.role)
	}
	exec(`INSERT INTO app.businesses(id,owner_user_id,legal_name,business_type,business_address,industry) VALUES($1::uuid,$2::uuid,'Audit buyer','limited_company','synthetic','test')`, f.profile, f.buyer)
	if err = admin.QueryRow(ctx, `SELECT organization_id::text FROM app.businesses WHERE id=$1::uuid`, f.profile).Scan(&f.buyerOrg); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO app.trade_relationships(supplier_organization_id,buyer_business_id,status) VALUES($1::uuid,$2::uuid,'active')`, f.org, f.profile)
	for i := range f.order {
		f.order[i] = uuid.NewString()
		agreement, activation := uuid.NewString(), uuid.NewString()
		f.obligation[i] = uuid.NewString()
		exec(`INSERT INTO app.credit_requests(id,supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,10000,'synthetic cartons',current_date+7,now()+interval '7 days','ACTIVE',$5::uuid)`, f.order[i], f.org, f.buyer, f.profile, f.owner)
		exec(`INSERT INTO app.agreement_versions(id,credit_request_id,version,canonical_json,document_hash,terms_version,privacy_version,created_by) VALUES($1::uuid,$2::uuid,1,'{}',$3,'v1','v1',$4::uuid)`, agreement, f.order[i], "audit-"+agreement, f.owner)
		exec(`INSERT INTO ledger.transactions(id,event_type,reference_type,reference_id,idempotency_key,effective_at) VALUES($1::uuid,'audit_fixture','credit_request',$2,$3,now())`, activation, f.order[i], "audit-fixture-"+f.order[i])
		exec(`INSERT INTO app.obligations(id,credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,10000,'NGN','ACTIVE','UNPAID',10000,0,$6::uuid,now())`, f.obligation[i], f.order[i], agreement, f.org, f.profile, activation)
		payload, e := json.Marshal(map[string]any{"request": map[string]any{"id": f.order[i], "version": 1}, "obligation": map[string]any{"id": f.obligation[i], "outstanding_kobo": 10000, "payment_status": "UNPAID"}})
		if e != nil {
			t.Fatal(e)
		}
		exec(`INSERT INTO app.credit_aggregate_snapshots(credit_request_id,supplier_organization_id,buyer_user_id,aggregate,version) VALUES($1,$2,$3,$4,1)`, f.order[i], f.org, f.buyer, payload)
		if _, _, e = schedules.NewPostgresStore(admin).CreateDefault(f.obligation[i], 10000, time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02"), time.Now().UTC().AddDate(0, 0, 7), 0); e != nil {
			t.Fatal(e)
		}
	}
	// The owning test database is destroyed by CI; retain immutable synthetic
	// evidence rather than disabling safety triggers for fixture cleanup.
	return f
}
func (f *auditFixture) as(user string) context.Context {
	return db.WithTenantContext(f.ctx, user, f.org)
}
func (f *auditFixture) store() *PostgresStore {
	return NewPostgresStore(f.runtime.Raw(), ledger.NewPostgresStore(f.runtime.Raw()))
}
func (f *auditFixture) scalar(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := f.admin.QueryRow(f.ctx, q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestAuditPostgresTermsPermissionsAndReviewOnly(t *testing.T) {
	f := newAuditFixture(t)
	s := buyers.NewPostgresTermsImportStore(f.runtime.Raw(), ledger.NewPostgresStore(f.runtime.Raw()))
	rows := []buyers.TermsImportRowInput{{CustomerName: "Customer A", OpeningBalanceKobo: 500000, OpeningBalanceReference: "LEGACY-1", ProposedGraceHours: 24}}
	first, err := s.StageTermsBatch(f.as(f.sales), f.sales, f.org, "", rows)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.StageTermsBatch(f.as(f.sales), f.sales, f.org, "", rows)
	if err != nil || again.ID != first.ID {
		t.Fatalf("stage retry: %+v %v", again, err)
	}
	rows[0].CustomerName = "Customer B"
	different, err := s.StageTermsBatch(f.as(f.sales), f.sales, f.org, "", rows)
	if err != nil || different.ID == first.ID {
		t.Fatalf("same-row-count source collision: %v", err)
	}
	list, err := s.ListTermsBatches(f.as(f.finance), f.finance, f.org)
	if err != nil || len(list) != 2 {
		t.Fatalf("tenant-scoped read missing real rows: %d %v", len(list), err)
	}
	for _, actor := range []string{f.sales, f.other} {
		if _, err = s.ReviewTermsBatch(f.as(actor), actor, f.org, first.ID, "approved"); err == nil {
			t.Fatal("sales/nonmember approved import")
		}
	}
	if _, err = s.ListTermsBatches(f.as(f.other), f.other, f.org); err == nil {
		t.Fatal("nonmember read imports")
	}
	// Reject a direct SQL attempt as sales too, not only the HTTP/service layer.
	scoped := &db.ScopedDatabase{Pool: f.runtime.Raw()}
	if _, err = scoped.Exec(f.as(f.sales), `UPDATE app.partner_terms_import_batches SET state='approved',approved_by=$2::uuid,approved_at=now() WHERE id=$1::uuid`, first.ID, f.sales); err == nil {
		t.Fatal("database allowed sales/self approval")
	}
	approved, err := s.ReviewTermsBatch(f.as(f.finance), f.finance, f.org, first.ID, "approved")
	if err != nil {
		t.Fatal(err)
	}
	if approved.FinancialApplication != "not_applied" {
		t.Fatalf("review presented as applied: %+v", approved)
	}
	same, err := s.ReviewTermsBatch(f.as(f.finance), f.finance, f.org, first.ID, "approved")
	if err != nil || !same.ApprovedAt.Equal(*approved.ApprovedAt) {
		t.Fatalf("review retry: %v", err)
	}
	_, stored, err := s.GetTermsBatch(f.as(f.finance), f.finance, f.org, first.ID)
	if err != nil || len(stored) != 1 || stored[0].ValidationStatus != "valid" {
		t.Fatalf("validation/application mixed: %+v %v", stored, err)
	}
	if n := f.scalar(t, `SELECT count(*) FROM ledger.transactions WHERE reference_id=$1`, stored[0].ID); n != 0 {
		t.Fatal("import row became a financial obligation")
	}
	if _, err = scoped.Exec(f.as(f.owner), `UPDATE app.partner_terms_import_rows SET opening_balance_kobo=1 WHERE batch_id=$1::uuid`, first.ID); err == nil {
		t.Fatal("approved source rows were mutable")
	}
	if _, _, err = s.GetTermsBatch(db.WithTenantContext(f.ctx, f.finance, f.buyerOrg), f.finance, f.org, first.ID); err == nil {
		t.Fatal("explicit tenant overrode authenticated tenant")
	}
}

func TestAuditPostgresShipmentRelationsConcurrencyAndReceipts(t *testing.T) {
	f := newAuditFixture(t)
	s := f.store()
	for _, order := range f.order {
		if err := s.CreateLineItems(f.as(f.sales), order, []LineItem{{Description: "cartons", Quantity: 10, UnitPriceKobo: 1000}, {Description: "other cartons", Quantity: 10, UnitPriceKobo: 1000}}); err != nil {
			t.Fatal(err)
		}
	}
	a, err := s.ListLineItems(f.as(f.sales), f.order[0])
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.ListLineItems(f.as(f.sales), f.order[1])
	if err != nil {
		t.Fatal(err)
	}
	dispatch := func(items []ShipmentItem) (Shipment, error) {
		return s.DispatchShipment(f.as(f.sales), DispatchInput{OrderID: f.order[0], SupplierOrganizationID: f.org, DispatchedBy: f.sales, Items: items})
	}
	for _, items := range [][]ShipmentItem{
		{{LineItemID: b[0].ID, Quantity: 1}},
		{{LineItemID: a[0].ID, Quantity: 2}, {LineItemID: a[1].ID, Quantity: 11}},
	} {
		if _, err = dispatch(items); err == nil {
			t.Fatal("wrong-order/partial dispatch accepted")
		}
		if n := f.scalar(t, `SELECT count(*) FROM app.order_shipments WHERE order_id=$1::uuid`, f.order[0]); n != 0 {
			t.Fatal("failed dispatch retained header")
		}
		if n := f.scalar(t, `SELECT sum(fulfilled_quantity) FROM app.order_line_items WHERE order_id=$1::uuid`, f.order[0]); n != 0 {
			t.Fatal("failed dispatch incremented quantity")
		}
	}
	var wg sync.WaitGroup
	results := make(chan Shipment, 2)
	failures := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ship, e := dispatch([]ShipmentItem{{LineItemID: a[0].ID, Quantity: 6}})
			if e != nil {
				failures <- e
			} else {
				results <- ship
			}
		}()
	}
	wg.Wait()
	close(results)
	close(failures)
	if len(results) != 1 || len(failures) != 1 {
		t.Fatalf("concurrent over-fulfilment: successes=%d failures=%d", len(results), len(failures))
	}
	ship := <-results
	if n := f.scalar(t, `SELECT fulfilled_quantity FROM app.order_line_items WHERE id=$1::uuid`, a[0].ID); n != 6 {
		t.Fatalf("quantity was not applied exactly once: %d", n)
	}
	buyerCtx := db.WithTenantContext(f.ctx, f.buyer, "")
	in := DeliveryInput{ShipmentID: ship.ID, OrderID: f.order[1], ReceivedBy: f.buyer, SignedProof: "six cartons received"}
	if _, err = s.RecordDeliveryReceipt(buyerCtx, in); err == nil {
		t.Fatal("receipt bound to different order")
	}
	in.OrderID = f.order[0]
	in.ReceivedBy = f.sales
	if _, err = s.RecordDeliveryReceipt(f.as(f.sales), in); err == nil {
		t.Fatal("seller signed buyer receipt")
	}
	in.ReceivedBy = f.buyer
	rec, err := s.RecordDeliveryReceipt(buyerCtx, in)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.RecordDeliveryReceipt(buyerCtx, in)
	if err != nil || again.ID != rec.ID {
		t.Fatalf("receipt replay: %+v %v", again, err)
	}
	in.SignedProof = "changed claim"
	if _, err = s.RecordDeliveryReceipt(buyerCtx, in); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflicting receipt accepted: %v", err)
	}
	if n := f.scalar(t, `SELECT count(*) FROM app.order_shipments WHERE id=$1::uuid AND status='delivered'`, ship.ID); n != 1 {
		t.Fatal("receipt did not transition shipment")
	}
	scoped := &db.ScopedDatabase{Pool: f.runtime.Raw()}
	if _, err = scoped.Exec(f.as(f.owner), `DELETE FROM app.order_delivery_receipts WHERE id=$1::uuid`, rec.ID); err == nil {
		t.Fatal("receipt can be deleted")
	}
	if _, err = scoped.Exec(buyerCtx, `UPDATE app.order_shipments SET carrier='changed' WHERE id=$1::uuid`, ship.ID); err == nil {
		t.Fatal("receipt permission grants general shipment edits")
	}
}

var errAuditInjected = errors.New("injected failure after journal posting")

type auditFailingLedger struct{ *ledger.PostgresStore }

func (l auditFailingLedger) PostAdjustmentTx(ctx context.Context, tx pgx.Tx, id string, amount ledger.Money, kind string, at time.Time, key string) (ledger.Transaction, error) {
	posted, err := l.PostgresStore.PostAdjustmentTx(ctx, tx, id, amount, kind, at, key)
	if err != nil {
		return posted, err
	}
	return posted, errAuditInjected
}
func TestAuditPostgresCreditNoteBindsOrderAndRollsBackFinancialAggregate(t *testing.T) {
	f := newAuditFixture(t)
	s := f.store()
	in := CreditNoteInput{OrderID: f.order[0], ObligationID: f.obligation[1], SupplierOrganizationID: f.org, IssuedBy: f.sales, AmountKobo: 2500, Reason: "verified shortage"}
	if _, err := s.CreateCreditNote(f.as(f.sales), in); err == nil {
		t.Fatal("credit note targeted another order's obligation")
	}
	in.ObligationID = ""
	note, err := s.CreateCreditNote(f.as(f.sales), in)
	if err != nil {
		t.Fatal(err)
	}
	if note.ObligationID != f.obligation[0] {
		t.Fatal("canonical obligation was not resolved")
	}
	failing := NewPostgresStore(f.runtime.Raw(), auditFailingLedger{ledger.NewPostgresStore(f.runtime.Raw())})
	if err = failing.ApproveCreditNote(f.as(f.finance), note.ID, f.finance); !errors.Is(err, errAuditInjected) {
		t.Fatalf("failure injection did not reach journal: %v", err)
	}
	if n := f.scalar(t, `SELECT count(*) FROM app.order_credit_notes WHERE id=$1::uuid AND status='draft'`, note.ID); n != 1 {
		t.Fatal("failed approval committed its status")
	}
	if n := f.scalar(t, `SELECT count(*) FROM ledger.transactions WHERE reference_id=$1`, note.ID); n != 0 {
		t.Fatal("failed approval left a journal entry")
	}
	if n := f.scalar(t, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, f.obligation[0]); n != 10000 {
		t.Fatal("failed approval changed balance")
	}
	if err = s.ApproveCreditNote(f.as(f.sales), note.ID, f.sales); err == nil {
		t.Fatal("sales/self approval allowed")
	}
	if err = s.ApproveCreditNote(f.as(f.finance), note.ID, f.finance); err != nil {
		t.Fatal(err)
	}
	if err = s.ApproveCreditNote(f.as(f.finance), note.ID, f.finance); err != nil {
		t.Fatalf("exact approval replay: %v", err)
	}
	for _, query := range []string{
		`SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`,
		`SELECT sum(i.principal_due_kobo-i.allocated_kobo) FROM app.schedule_items i JOIN app.repayment_schedules s ON s.id=i.schedule_id WHERE s.obligation_id=$1::uuid AND i.state<>'CANCELLED'`,
		`SELECT (aggregate#>>'{obligation,outstanding_kobo}')::bigint FROM app.credit_aggregate_snapshots WHERE aggregate#>>'{obligation,id}'=$1`,
	} {
		if n := f.scalar(t, query, f.obligation[0]); n != 7500 {
			t.Fatalf("financial aggregate disagrees: %d", n)
		}
	}
	if n := f.scalar(t, `SELECT count(*) FROM ledger.transactions WHERE reference_id=$1`, note.ID); n != 1 {
		t.Fatal("approval replay posted twice")
	}
	if n := f.scalar(t, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, f.obligation[1]); n != 10000 {
		t.Fatal("another order's balance changed")
	}
}

func TestAuditPostgresChildRowsRespectBranchAndRevokedMembership(t *testing.T) {
	f := newAuditFixture(t)
	s := f.store()
	if err := s.CreateLineItems(f.as(f.owner), f.order[0], []LineItem{{Description: "cartons", Quantity: 10, UnitPriceKobo: 1000}}); err != nil {
		t.Fatal(err)
	}
	lines, err := s.ListLineItems(f.as(f.owner), f.order[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DispatchShipment(f.as(f.owner), DispatchInput{OrderID: f.order[0], SupplierOrganizationID: f.org, DispatchedBy: f.owner, Items: []ShipmentItem{{LineItemID: lines[0].ID, Quantity: 1}}}); err != nil {
		t.Fatal(err)
	}
	note, err := s.CreateCreditNote(f.as(f.owner), CreditNoteInput{OrderID: f.order[0], SupplierOrganizationID: f.org, IssuedBy: f.owner, AmountKobo: 100, Reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	// A restricted staff member with no assigned branches must not inherit
	// organization-wide rights to existing shipment/note rows.
	tx, err := f.admin.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(f.ctx)
	if _, err = tx.Exec(f.ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, f.owner, f.org); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(f.ctx, `INSERT INTO app.member_branch_scopes(organization_id,user_id,membership_id,membership_version,mode,branch_ids,updated_by) SELECT organization_id,user_id,id,purchasing_authority_version,'branches','{}',$3::uuid FROM app.memberships WHERE organization_id=$1::uuid AND user_id=$2::uuid`, f.org, f.sales, f.owner); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	ships, err := s.ListShipments(f.as(f.sales), f.order[0])
	if err != nil || len(ships) != 0 {
		t.Fatalf("branch shipment leak: %+v %v", ships, err)
	}
	notes, err := s.ListCreditNotes(f.as(f.sales), f.order[0])
	if err != nil || len(notes) != 0 {
		t.Fatalf("branch credit-note leak: %+v %v", notes, err)
	}
	if _, err = s.CreateCreditNote(f.as(f.sales), CreditNoteInput{OrderID: f.order[0], SupplierOrganizationID: f.org, IssuedBy: f.sales, AmountKobo: 100, Reason: "test"}); err == nil {
		t.Fatal("branch staff changed another branch's order")
	}
	if err = s.ApproveCreditNote(f.as(f.sales), note.ID, f.sales); err == nil {
		t.Fatal("branch staff reviewed unavailable note")
	}
	if _, err = f.admin.Exec(f.ctx, `UPDATE app.memberships SET status='removed' WHERE organization_id=$1::uuid AND user_id=$2::uuid`, f.org, f.finance); err != nil {
		t.Fatal(err)
	}
	if err = s.ApproveCreditNote(f.as(f.finance), note.ID, f.finance); err == nil {
		t.Fatal("revoked reviewer retained authority")
	}
}
