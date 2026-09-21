package buyers

import (
	"context"
	"errors"
	"kredit/internal/ledger"
	"testing"
	"time"
)

type rejectImportPosting struct {
	ledger.Service
	calls int
}

func (s *rejectImportPosting) PostActivation(string, ledger.Money, time.Time, string) (ledger.Transaction, error) {
	s.calls++
	return ledger.Transaction{}, errors.New("an unlinked staging row is not an obligation")
}

func TestAuditTermsCanonicalContentAndExactReplay(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryTermsImportStore()
	rows := []TermsImportRowInput{{CustomerName: "buyer A", OpeningBalanceKobo: 1000, ProposedGraceHours: 24}}
	first, err := store.StageTermsBatch(ctx, "sales", "org", "", rows)
	if err != nil {
		t.Fatal(err)
	}
	again, err := store.StageTermsBatch(ctx, "sales", "org", "", rows)
	if err != nil || first.ID != again.ID {
		t.Fatalf("exact replay: %+v %v", again, err)
	}
	rows[0].CustomerName = "buyer B"
	second, err := store.StageTermsBatch(ctx, "sales", "org", "", rows)
	if err != nil || first.SourceHash == second.SourceHash || first.ID == second.ID {
		t.Fatalf("different same-length import collided: %+v %v", second, err)
	}
	if _, err = store.StageTermsBatch(ctx, "sales", "org", first.SourceHash, rows); !errors.Is(err, ErrTermsInvalidRow) {
		t.Fatalf("forged content fingerprint: %v", err)
	}
	if _, _, err = store.GetTermsBatch(ctx, "sales", "another-org", first.ID); !errors.Is(err, ErrTermsBatchNotFound) {
		t.Fatalf("cross-organization batch: %v", err)
	}
}
func TestAuditTermsApprovalDoesNotInventDebtOrApplication(t *testing.T) {
	ctx := context.Background()
	journal := &rejectImportPosting{Service: ledger.NewStore()}
	store := NewMemoryTermsImportStore(journal)
	batch, err := store.StageTermsBatch(ctx, "sales", "org", "", []TermsImportRowInput{{CustomerName: "buyer", OpeningBalanceKobo: 50000}})
	if err != nil {
		t.Fatal(err)
	}
	approved, err := store.ReviewTermsBatch(ctx, "finance", "org", batch.ID, "approved")
	if err != nil {
		t.Fatal(err)
	}
	if journal.calls != 0 || approved.FinancialApplication != "not_applied" {
		t.Fatalf("review created financial effect: %+v calls=%d", approved, journal.calls)
	}
	_, rows, err := store.GetTermsBatch(ctx, "finance", "org", batch.ID)
	if err != nil || rows[0].ValidationStatus != "valid" {
		t.Fatalf("validation conflated with application: %+v %v", rows, err)
	}
	again, err := store.ReviewTermsBatch(ctx, "finance", "org", batch.ID, "approved")
	if err != nil || again.ApprovedAt == nil || !again.ApprovedAt.Equal(*approved.ApprovedAt) {
		t.Fatalf("review retry failed: %v", err)
	}
	*approved.ApprovedAt = time.Time{}
	restored, _, err := store.GetTermsBatch(ctx, "finance", "org", batch.ID)
	if err != nil || restored.ApprovedAt.IsZero() {
		t.Fatal("returned timestamp aliases stored approval")
	}
}
func TestAuditTermsRejectsInvalidFinancialRange(t *testing.T) {
	for _, row := range []TermsImportRowInput{{OpeningBalanceKobo: -1}, {ProposedCreditLimitKobo: 9007199254740992}, {ProposedGraceHours: 721}} {
		if _, err := TermsSourceHash([]TermsImportRowInput{row}); !errors.Is(err, ErrTermsInvalidRow) {
			t.Fatalf("invalid row accepted: %+v %v", row, err)
		}
	}
}
