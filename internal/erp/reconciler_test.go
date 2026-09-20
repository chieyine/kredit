package erp

import (
	"testing"
	"time"
)

func TestERPPackageSerializationAndReconciliation(t *testing.T) {
	now := time.Now().UTC()
	pkg := ERPPackageV2{
		Version:        ContractV2,
		OrganizationID: "org-supplier-100",
		ExportedAt:     now,
		PeriodStart:    now.Add(-30 * 24 * time.Hour),
		PeriodEnd:      now,
		Invoices: []ERPInvoice{
			{
				Reference:          "INV-001",
				CustomerIdentifier: "CUST-A",
				CustomerName:       "Alhaji Sani & Sons",
				AmountKobo:         25000000,
				DueDate:            "2026-10-15",
				State:              "ACTIVE",
				CreatedAt:          now,
			},
		},
		Payments: []ERPPayment{
			{
				Reference:           "PAY-001",
				InvoiceReference:    "INV-001",
				AmountKobo:          10000000,
				PaymentMethod:       "bank_transfer",
				PaidAt:              now,
				LedgerTransactionID: "tx-ledger-991",
			},
		},
		CreditNotes: []ERPCreditNote{
			{
				Reference:        "CN-001",
				InvoiceReference: "INV-001",
				AmountKobo:       1000000,
				Reason:           "Damaged bag allowance",
				ApprovedBy:       "user-finance-2",
				ApprovedAt:       now,
			},
		},
		JournalEntries: []ERPJournalEntry{
			{
				TransactionID: "tx-1",
				AccountCode:   "1200",
				AccountName:   "Accounts Receivable",
				DebitKobo:     25000000,
				CreditKobo:    0,
				Description:   "Invoice INV-001 issuance",
				Timestamp:     now,
			},
			{
				TransactionID: "tx-1",
				AccountCode:   "4000",
				AccountName:   "Sales Revenue",
				DebitKobo:     0,
				CreditKobo:    25000000,
				Description:   "Invoice INV-001 revenue recognition",
				Timestamp:     now,
			},
		},
	}

	// 1. Integrity hash computation
	h := pkg.ComputeIntegrityHash()
	if len(h) != 64 {
		t.Fatalf("expected 64-character SHA-256 hex string, got %s", h)
	}

	// 2. CSV serialization
	csvBytes, err := pkg.ToCSV()
	if err != nil {
		t.Fatalf("to CSV: %v", err)
	}
	if len(csvBytes) == 0 {
		t.Fatal("expected non-empty CSV output")
	}

	// 3. Reconcile matching records
	internals := []InternalRecord{
		{Reference: "INV-001", TransactionID: "tx-1", AmountKobo: 25000000, Date: now},
		{Reference: "PAY-001", TransactionID: "tx-2", AmountKobo: 10000000, Date: now},
	}
	externals := []ExternalRecord{
		{Reference: "INV-001", AmountKobo: 25000000, Date: now, Note: "ERP Invoice 001"},
		{Reference: "PAY-001", AmountKobo: 10000000, Date: now, Note: "ERP Payment 001"},
	}

	res, err := ReconcileExternal("org-supplier-100", pkg.PeriodStart, pkg.PeriodEnd, internals, externals)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if !res.Balanced || res.NetVarianceKobo != 0 || res.MatchedRecordCount != 2 || len(res.Discrepancies) != 0 {
		t.Fatalf("expected balanced reconciliation, got: %+v", res)
	}

	// 4. Reconcile with amount discrepancy
	externalsMismatched := []ExternalRecord{
		{Reference: "INV-001", AmountKobo: 24000000, Date: now, Note: "ERP Invoice 001 (Understated)"},
		{Reference: "PAY-001", AmountKobo: 10000000, Date: now, Note: "ERP Payment 001"},
	}
	resMismatch, err := ReconcileExternal("org-supplier-100", pkg.PeriodStart, pkg.PeriodEnd, internals, externalsMismatched)
	if err != nil {
		t.Fatalf("reconcile mismatch: %v", err)
	}
	if resMismatch.Balanced || resMismatch.NetVarianceKobo != 1000000 || len(resMismatch.Discrepancies) != 1 {
		t.Fatalf("expected 1 amount mismatch discrepancy, got: %+v", resMismatch)
	}
	if resMismatch.Discrepancies[0].Type != DiscrepancyAmountMismatch {
		t.Fatalf("expected DiscrepancyAmountMismatch, got %s", resMismatch.Discrepancies[0].Type)
	}
}
