package organizations

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestConcurrentOrganizationCreationHonorsLimit(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var owner string
	if err = pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("organization-cap-%d@example.test", time.Now().UnixNano())).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err = pool.QueryRow(ctx, `SELECT app.organization_count()`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := NewPostgresStore(pool, "isolated-limit-test")
			s.SetOrganizationLimit(count + 1)
			<-start
			_, _, err := s.Create(owner, CreateInput{LegalName: "Cap fixture", BusinessType: "limited_company", BusinessAddress: "Lagos", Industry: "retail"})
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	accepted := 0
	for err := range results {
		if err == nil {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatalf("concurrent cap allowed %d organizations", accepted)
	}
	var initialized int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM app.supplier_onboarding_profiles p JOIN app.memberships m ON m.organization_id=p.organization_id JOIN app.supplier_onboarding_revisions r ON r.organization_id=p.organization_id AND r.change_type='profile.created' WHERE m.user_id=$1::uuid AND m.role='owner' AND p.owner_email_verified_at IS NOT NULL`, owner).Scan(&initialized); err != nil || initialized != 1 {
		t.Fatalf("business did not commit its initial profile and revision: count=%d error=%v", initialized, err)
	}
	var after int64
	if err = pool.QueryRow(ctx, `SELECT app.organization_count()`).Scan(&after); err != nil || after != count+1 {
		t.Fatalf("unexpected persisted count %d: %v", after, err)
	}
}
