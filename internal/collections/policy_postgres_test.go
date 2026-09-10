package collections

import (
	"encoding/json"
	"testing"
	"time"

	"kredit/internal/businesspolicy"

	"github.com/google/uuid"
)

func TestEligibilityReflectsAdminPauseBeforeFirstAttempt(t *testing.T) {
	f := financialFixture(t)
	tx, err := f.pool.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(f.ctx) }()
	policy, err := businesspolicy.ReadTx(f.ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	policy.Values.CollectionsEnabled = false
	policy.Values.MaxRetries = 1
	encoded, err := json.Marshal(policy.Values)
	if err != nil {
		t.Fatal(err)
	}
	var checker string
	if err = tx.QueryRow(f.ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, uuid.NewString()+"@collection-policy.test").Scan(&checker); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(f.ctx, `INSERT INTO app.business_policy_changes(id,base_revision,values,proposed_by,reason,effective_at,state,decided_by,decided_at) VALUES($1::uuid,$2,$3::jsonb,$4::uuid,'Historical collection pause fixture',now()-interval '1 minute','approved',$5::uuid,now()-interval '2 minutes')`, uuid.NewString(), policy.Revision, encoded, f.user, checker); err != nil {
		t.Fatal(err)
	}
	engine := f.engine(NewMockProvider("policy-check"))
	engine.pool = tx
	eligibility, err := engine.EligibilityContext(f.ctx, f.id, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if eligibility.Eligible {
		t.Fatal("admin-paused collections appeared eligible")
	}
	found := false
	for _, reason := range eligibility.Reasons {
		if reason == "feature_disabled" {
			found = true
		}
	}
	if !found {
		t.Fatalf("admin pause missing from reasons: %+v", eligibility)
	}
	items, err := engine.ReadAttemptsContext(f.ctx, f.id)
	if err != nil || len(items) != 0 {
		t.Fatalf("first-attempt history=%+v err=%v", items, err)
	}
}
