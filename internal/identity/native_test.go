package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"kredit/internal/documents"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNativeLookupPermissionsConsentAndEvidence(t *testing.T) {
	dsn := os.Getenv("NATIVE_IDENTITY_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires disposable identity test database")
	}
	ctx := context.Background()
	owner, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	var user, other string
	for _, target := range []*string{&user, &other} {
		if err = owner.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, uuid.NewString()+"@native-check.test").Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	app, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	calls := 0
	remote := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("mono-sec-key") != "synthetic-secret" {
			t.Error("wrong authentication header")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v3/lookup/phone/initiate":
			var in map[string]string
			_ = json.NewDecoder(r.Body).Decode(&in)
			if in["phone_number"] != "08012345678" {
				t.Error("wrong initiation body")
			}
			_, _ = fmt.Fprint(w, `{"status":"successful","data":{"reference":"synthetic-otp","expires_in_seconds":300}}`)
		case "/v3/lookup/phone/verify":
			var in map[string]string
			_ = json.NewDecoder(r.Body).Decode(&in)
			if in["reference"] != "synthetic-otp" || in["otp"] != "123456" {
				t.Error("wrong OTP body")
			}
			_, _ = fmt.Fprint(w, `{"status":"successful","data":{"nin":"12345678901","first_name":"Ada","last_name":"Example"}}`)
		case "/v3/lookup/cac":
			if r.URL.Query().Get("search") != "RC12345" || r.URL.Query().Get("exact") != "true" {
				t.Error("wrong CAC query")
			}
			_, _ = fmt.Fprint(w, `{"status":"successful","data":[{"rc_number":"12345","approved_name":"Example Limited","active":true,"registration_approved":true}]}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer remote.Close()
	p, err := NewNativeLookup(app, "mono-test-account", remote.URL, "synthetic-secret")
	if err != nil {
		t.Fatal(err)
	}
	p.client = remote.Client()
	actor := WithActor(ctx, user)
	session, err := p.CreatePersonVerification(actor, PersonVerificationInput{SubjectID: uuid.NewString(), FullName: "Ada Example"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.GetVerification(WithActor(ctx, other), session.ProviderID); err == nil {
		t.Fatal("another user read identity evidence")
	}
	if err = p.Act(actor, session.ProviderID, NativeAction{Action: "phone_start", Value: "08012345678", ExpectedVersion: 1}); err == nil || calls != 0 {
		t.Fatal("lookup without consent")
	}
	if err = p.Act(actor, session.ProviderID, NativeAction{Action: "phone_start", Value: "08012345678", ExpectedVersion: 1, ConsentVersion: NativeConsentVersion}); err != nil {
		t.Fatal(err)
	}
	if err = p.Act(actor, session.ProviderID, NativeAction{Action: "phone_start", Value: "08012345678", ExpectedVersion: 3, ConsentVersion: NativeConsentVersion}); err == nil || calls != 1 {
		t.Fatal("unexpired OTP was sent twice")
	}
	if err = p.Act(actor, session.ProviderID, NativeAction{Action: "phone_verify", Value: "123456", ExpectedVersion: 3}); err != nil {
		t.Fatal(err)
	}
	verified, err := p.GetVerification(actor, session.ProviderID)
	if err != nil || verified.State != "verified" || verified.VerificationLevel != 2 {
		t.Fatalf("identity evidence: %+v %v", verified, err)
	}
	encoded, _ := json.Marshal(verified.SafeResult)
	if strings.Contains(string(encoded), "12345678901") {
		t.Fatal("NIN persisted")
	}
	business, err := p.CreateBusinessVerification(actor, BusinessVerificationInput{SubjectID: uuid.NewString(), LegalName: "Example Limited", RequireReview: true})
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Act(actor, business.ProviderID, NativeAction{Action: "business_lookup", Value: "RC12345", ExpectedVersion: 1, ConsentVersion: NativeConsentVersion}); err != nil {
		t.Fatal(err)
	}
	check, err := p.GetVerification(actor, business.ProviderID)
	if err != nil || check.State != "review" {
		t.Fatal("supplier KYB bypassed evidence review", err)
	}

	docs := documents.NewPostgresStore(app, documents.NewMemoryObjectStore())
	doc, err := docs.Add(ctx, "", user, "identity_"+business.ProviderID, "authority.pdf", "application/pdf", "identity_evidence", 3, bytes.NewReader([]byte("pdf")))
	if err != nil {
		t.Fatal("identity evidence upload", err)
	}
	if err = p.AttachEvidence(actor, business.ProviderID, doc.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = docs.ReadForTenant(ctx, doc.ID, other, ""); err == nil {
		t.Fatal("another user read private evidence")
	}
	if _, err = docs.SignedDownloadForTenant(ctx, doc.ID, user, "", 1); err == nil {
		t.Fatal("unscanned evidence was downloadable")
	}
	if err = p.AttachEvidence(WithActor(ctx, other), business.ProviderID, doc.ID); err == nil {
		t.Fatal("another user attached evidence")
	}
	if err = p.Act(actor, business.ProviderID, NativeAction{Action: "approve", ExpectedVersion: 3, Evidence: "Customer cannot approve its own supplier verification"}); err == nil {
		t.Fatal("customer approved own KYB")
	}
	if _, err = owner.Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_owner',$1::uuid,'Synthetic identity appeal review')`, user); err != nil {
		t.Fatal(err)
	}
	var version int64
	if err = owner.QueryRow(ctx, `SELECT version FROM app.native_identity_sessions WHERE id=$1::uuid`, business.ProviderID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err = p.Act(actor, business.ProviderID, NativeAction{Action: "reject", ExpectedVersion: version, Evidence: "Synthetic rejected authority evidence for appeal check"}); err != nil {
		t.Fatal(err)
	}
	if err = p.Act(actor, business.ProviderID, NativeAction{Action: "retry", ExpectedVersion: version + 1, Evidence: "Synthetic appeal reviewed; fresh evidence required"}); err != nil {
		t.Fatal(err)
	}
	var history int
	if err = owner.QueryRow(ctx, `SELECT count(*) FROM app.native_identity_history WHERE session_id=$1::uuid AND previous_decision->>'state'='failed'`, business.ProviderID).Scan(&history); err != nil || history != 1 {
		t.Fatal("appeal lost original decision", history, err)
	}
	check, err = p.GetVerification(actor, business.ProviderID)
	if err != nil || check.State != "pending" {
		t.Fatal("appeal auto-verified identity", check, err)
	}
}

func TestIdentityRouterNeverFallsBack(t *testing.T) {
	active := NewMockProvider()
	old := NewUnavailableProvider("saved account cannot be reached")
	router, err := NewRouter(active, old)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = GetFrom(context.Background(), router, "missing-account", "reference"); err == nil {
		t.Fatal("unknown account fell back")
	}
	if _, err = GetRouted(context.Background(), router, "legacy-reference"); err == nil {
		t.Fatal("unrouted case used current account")
	}
	result, err := active.CreateBusinessVerification(context.Background(), BusinessVerificationInput{SubjectID: "business", LegalName: "Business"})
	if err != nil {
		t.Fatal(err)
	}
	ref := RoutedReference(active.Name(), result.ProviderID)
	v, err := GetRouted(context.Background(), router, ref)
	if err != nil || v.ProviderID != ref || v.SubjectID != "business" {
		t.Fatal("saved route failed", err)
	}
}
