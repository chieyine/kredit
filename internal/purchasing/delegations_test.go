package purchasing

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDelegationsRequireCurrentOwnerAndExpire(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated database required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	owner, staff, other, profile := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := admin.Exec(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	for _, u := range []string{owner, staff, other} {
		exec(`INSERT INTO app.users(id,normalized_email) VALUES($1::uuid,$2)`, u, u+"@delegation.test")
	}
	exec(`INSERT INTO app.businesses(id,owner_user_id,legal_name,business_type,business_address,industry) VALUES($1::uuid,$2::uuid,'Delegation fixture','limited_company','Lagos','food')`, profile, owner)
	exec(`INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'sales','active')`, profile, staff)
	cfg, _ := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	s := Store{Pool: pool}
	g := Grant{UserID: staff, Actions: []string{"read", "accept", "receive", "review"}, CeilingKobo: 10000, ExpiresAt: time.Now().UTC().Add(time.Hour).Truncate(time.Second)}
	if _, err = s.Save(ctx, staff, profile, g); !errors.Is(err, ErrAuthority) {
		t.Fatalf("staff granted themselves access: %v", err)
	}
	saved, err := s.Save(ctx, owner, profile, g)
	if err != nil {
		t.Fatal(err)
	}
	directory, e := s.Read(ctx, owner, profile)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, item := range directory.Grants {
		if item.UserID == staff {
			found = item.Active && item.Version == saved.Version
		}
	}
	if !found {
		t.Fatal("saved active grant missing from directory")
	}
	replay, err := s.Save(ctx, owner, profile, g)
	if err != nil || replay.Version != saved.Version {
		t.Fatalf("replay: %#v %v", replay, err)
	}
	allowed := func(action string, amount int64) bool {
		t.Helper()
		tx, e := pool.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, e = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, staff); e != nil {
			t.Fatal(e)
		}
		var yes bool
		if e = tx.QueryRow(ctx, `SELECT app.can_purchase($1::uuid,$2,$3)`, profile, action, amount).Scan(&yes); e != nil {
			t.Fatal(e)
		}
		return yes
	}
	if !allowed("read", 0) || !allowed("accept", 10000) || allowed("accept", 10001) || allowed("bank", 0) {
		t.Fatal("permission or ceiling ignored")
	}
	g.CeilingKobo = 20000
	if _, err = s.Save(ctx, owner, profile, g); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale grant accepted: %v", err)
	}
	g.UserID = other
	g.Version = 0
	if _, err = s.Save(ctx, owner, profile, g); err == nil {
		t.Fatal("outsider granted access")
	}
	exec(`UPDATE app.memberships SET status='removed' WHERE organization_id=$1::uuid AND user_id=$2::uuid`, profile, staff)
	if allowed("read", 0) {
		t.Fatal("removed staff retained access")
	}
	exec(`UPDATE app.memberships SET status='active' WHERE organization_id=$1::uuid AND user_id=$2::uuid`, profile, staff)
	if allowed("read", 0) {
		t.Fatal("restoring membership revived an old grant")
	}
	saved.Actions = []string{}
	saved.ExpiresAt = time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	if _, err = s.Save(ctx, owner, profile, saved); err != nil {
		t.Fatal(err)
	}
	if allowed("read", 0) {
		t.Fatal("revoked permission retained access")
	}
	// A limited grant cannot revive a former owner's legacy financial authority.
	exec(`INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active')`, profile, other)
	exec(`UPDATE app.memberships SET role='finance' WHERE organization_id=$1::uuid AND user_id=$2::uuid AND status='active'`, profile, owner)
	if _, err = s.Save(ctx, other, profile, Grant{UserID: owner, Actions: []string{"read"}, CeilingKobo: 0, ExpiresAt: time.Now().Add(time.Hour)}); err == nil {
		t.Fatal("former owner received an unusable grant")
	}
	tx, e := pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, e = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, owner); e != nil {
		t.Fatal(e)
	}
	var restored bool
	if e = tx.QueryRow(ctx, `SELECT app.can_purchase($1::uuid)`, profile).Scan(&restored); e != nil || restored {
		t.Fatalf("former owner restored legacy authority: %v %v", restored, e)
	}

}
