package documents

import (
	"context"
	"os"
	"testing"
	"time"

	"kredit/internal/db"
	"kredit/internal/identifier"
)

// Uses only an isolated database and synthetic ObjectJanitor. No object-storage
// credentials or real deletions are involved in this retention regression.
func TestCleanupIdentityReservationsPreserveEvidence(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated database required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	root, err := db.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	worker, err := db.OpenAsRole(ctx, os.Getenv("RIVER_DATABASE_URL"), "kredit_worker")
	if err != nil {
		t.Fatal(err)
	}
	defer worker.Close()
	user := identifier.New()
	if _, err := root.Raw().Exec(ctx, `INSERT INTO app.users(id,normalized_email) VALUES($1,$2)`, user, "identity-cleanup-"+user+"@example.test"); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	old := now.Add(-14 * 24 * time.Hour)
	expired := now.Add(-10 * 24 * time.Hour)
	grace := now.Add(-6 * 24 * time.Hour)
	future := now.Add(time.Hour)
	key := func() string { return "identity/" + user + "/identity_evidence/" + identifier.New() }
	cases := []struct {
		name      string
		key       string
		metadata  bool
		expires   *time.Time
		completed *time.Time
		modified  time.Time
		remove    bool
	}{
		{name: "unreferenced identity object", key: key(), modified: old, remove: true},
		{name: "expired incomplete identity reservation", key: key(), metadata: true, expires: &expired, modified: old, remove: true},
		{name: "completed quarantined evidence", key: key(), metadata: true, expires: &expired, completed: &old, modified: old},
		{name: "incomplete reservation within grace period", key: key(), metadata: true, expires: &grace, modified: old},
		{name: "unexpired reservation", key: key(), metadata: true, expires: &future, modified: old},
		{name: "unknown reservation expiry", key: key(), metadata: true, modified: old},
		{name: "recent object without metadata", key: key(), modified: now},
		{name: "unknown object age", key: key()},
		{name: "unrelated object", key: "unrelated-storage-object", modified: old},
		{name: "malformed identity owner", key: "identity/------------------------------------/identity_evidence/" + identifier.New(), modified: old},
		{name: "malformed identity object id", key: "identity/" + user + "/identity_evidence/------------------------------------", modified: old},
		{name: "unexpected path segment", key: "identity/" + user + "/identity_evidence/nested/" + identifier.New(), modified: old},
		{name: "unsupported object prefix", key: "backups/" + user + "/identity_evidence/" + identifier.New(), modified: old},
	}
	objects := &cleanupFixture{MemoryObjectStore: NewMemoryObjectStore()}
	for _, tc := range cases {
		if tc.metadata {
			if _, err := root.Raw().Exec(ctx, `INSERT INTO app.documents(uploaded_by,purpose,object_key,file_name,content_type,size_bytes,sha256,scan_state,retention_class,upload_expires_at,upload_completed_at) VALUES($1,'identity_evidence',$2,'fixture.pdf','application/pdf',3,'','QUARANTINED','identity',$3,$4)`, user, tc.key, tc.expires, tc.completed); err != nil {
				t.Fatalf("%s: insert metadata: %v", tc.name, err)
			}
		}
		objects.candidates = append(objects.candidates, ObjectCandidate{Key: tc.key, ModifiedAt: tc.modified})
	}
	if err := NewPostgresStore(worker.Raw(), objects).CleanupOrphans(ctx); err != nil {
		t.Fatal(err)
	}
	removed := make(map[string]int)
	for _, key := range objects.removed {
		removed[key]++
	}
	for _, tc := range cases {
		want := 0
		if tc.remove {
			want = 1
		}
		if removed[tc.key] != want {
			t.Errorf("%s: deletions = %d, want %d", tc.name, removed[tc.key], want)
		}
		var records int
		if err := root.Raw().QueryRow(ctx, `SELECT count(*) FROM app.audit_events WHERE action='document.orphan_removed' AND resource_id=$1`, tc.key).Scan(&records); err != nil {
			t.Fatal(err)
		}
		if records != want {
			t.Errorf("%s: removal audit records = %d, want %d", tc.name, records, want)
		}
	}
	var publicExecute bool
	if err := root.Raw().QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid='app.document_object_is_orphan(text)'::regprocedure AND a.grantee=0 AND a.privilege_type='EXECUTE')`).Scan(&publicExecute); err != nil {
		t.Fatal(err)
	}
	if publicExecute {
		t.Error("orphan eligibility must not be executable by PUBLIC")
	}
}
