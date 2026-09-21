package buyers

import (
	"context"
	"testing"
)

func TestTermsImportLifecycle(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryTermsImportStore()

	uploader := "user-sales-1"
	reviewer := "user-finance-1"
	orgID := "org-distributor-1"

	rows := []TermsImportRowInput{
		{
			CustomerName:            "Mega Wholesale Stores Ltd",
			ContactEmail:            "purchases@megawholesale.ng",
			ContactPhone:            "08031234567",
			ProposedCreditLimitKobo: 50000000,
			ProposedGraceHours:      48,
			OpeningBalanceKobo:      12500000,
			OpeningBalanceReference: "LEGACY-INV-4412",
		},
		{
			CustomerName:            "Kano City Provisions",
			ContactEmail:            "kano.provisions@gmail.com",
			ContactPhone:            "08098765432",
			ProposedCreditLimitKobo: 20000000,
			ProposedGraceHours:      24,
			OpeningBalanceKobo:      0,
			OpeningBalanceReference: "",
		},
	}

	// 1. Stage batch
	batch, err := store.StageTermsBatch(ctx, uploader, orgID, "", rows)
	if err != nil {
		t.Fatalf("stage terms batch: %v", err)
	}
	if batch.State != "staged" || batch.TotalRows != 2 || batch.ValidRows != 2 {
		t.Fatalf("unexpected batch state: %+v", batch)
	}

	// 2. Uploader self-approval must fail under maker-checker dual control
	if _, err := store.ReviewTermsBatch(ctx, uploader, orgID, batch.ID, "approved"); err != ErrTermsDualControl {
		t.Fatalf("expected ErrTermsDualControl on self-approval, got: %v", err)
	}

	// 3. Independent reviewer approves
	approved, err := store.ReviewTermsBatch(ctx, reviewer, orgID, batch.ID, "approved")
	if err != nil {
		t.Fatalf("review terms batch: %v", err)
	}
	if approved.State != "approved" || approved.ApprovedBy != reviewer || approved.ApprovedAt == nil {
		t.Fatalf("expected approved batch, got %+v", approved)
	}

	// 4. Retrieve batch and verify rows
	b, loadedRows, err := store.GetTermsBatch(ctx, uploader, orgID, batch.ID)
	if err != nil {
		t.Fatalf("get terms batch: %v", err)
	}
	if b.ID != batch.ID || len(loadedRows) != 2 {
		t.Fatalf("expected 2 loaded rows, got %d", len(loadedRows))
	}
	if loadedRows[0].OpeningBalanceKobo != 12500000 {
		t.Fatalf("expected 12500000 kobo opening balance, got %d", loadedRows[0].OpeningBalanceKobo)
	}
}
