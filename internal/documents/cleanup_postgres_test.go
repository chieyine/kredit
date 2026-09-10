package documents

import (
	"context"
	"os"
	"testing"
	"time"

	"kredit/internal/db"
	"kredit/internal/identifier"
)

type cleanupFixture struct {
	*MemoryObjectStore
	candidates []ObjectCandidate
	removed    []string
}

func (f *cleanupFixture) ListObjects(context.Context, string) ([]ObjectCandidate, string, error) {
	return f.candidates, "", nil
}
func (f *cleanupFixture) DeleteObject(_ context.Context, key string) error {
	f.removed = append(f.removed, key)
	return nil
}
func TestCleanupPreservesCompletedAndRecentObjects(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated database required")
	}
	ctx := context.Background()
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
	user, org := identifier.New(), identifier.New()
	if _, err = root.Raw().Exec(ctx, `INSERT INTO app.users(id,normalized_email) VALUES($1,$2)`, user, "cleanup-"+user+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if _, err = root.Raw().Exec(ctx, `INSERT INTO app.organizations(id,legal_name,business_type,business_address,industry) VALUES($1,'Cleanup fixture','limited_company','Lagos','retail')`, org); err != nil {
		t.Fatal(err)
	}
	orphan := org + "/invoice/" + identifier.New()
	complete := org + "/invoice/" + identifier.New()
	expired := org + "/invoice/" + identifier.New()
	recent := org + "/invoice/" + identifier.New()
	for _, key := range []string{complete, expired} {
		if _, err = root.Raw().Exec(ctx, `INSERT INTO app.documents(organization_id,uploaded_by,purpose,object_key,file_name,content_type,size_bytes,sha256,scan_state,retention_class,upload_expires_at,upload_completed_at) VALUES($1,$2,'invoice',$3,'fixture.pdf','application/pdf',3,'','QUARANTINED','financial',now()-interval '10 days',CASE WHEN $4 THEN now()-interval '12 days' END)`, org, user, key, key == complete); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-14 * 24 * time.Hour)
	objects := &cleanupFixture{MemoryObjectStore: NewMemoryObjectStore(), candidates: []ObjectCandidate{{orphan, old}, {complete, old}, {expired, old}, {recent, time.Now()}, {"unrelated-storage-object", old}}}
	if err = NewPostgresStore(worker.Raw(), objects).CleanupOrphans(ctx); err != nil {
		t.Fatal(err)
	}
	if len(objects.removed) != 2 || objects.removed[0] != orphan || objects.removed[1] != expired {
		t.Fatalf("incorrect cleanup selection: %v", objects.removed)
	}
}
