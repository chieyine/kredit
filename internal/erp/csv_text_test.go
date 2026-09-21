package erp

import (
	"bytes"
	"encoding/csv"
	"testing"
)

func TestAuditSpreadsheetCSVNeutralizesTextButMachineCSVIsLossless(t *testing.T) {
	for _, value := range []string{"=1+1", "+SUM(A1)", "-1+1", "@SUM(A1)", " \t=1+1", "\r=1+1", "\n=1+1", "\ufeff=1+1", "＝1+1", "\"\n=1+1"} {
		pkg := ERPPackageV2{Invoices: []ERPInvoice{{Reference: value, CustomerName: value, AmountKobo: 12345, DueDate: value, State: value}}, CreditNotes: []ERPCreditNote{{Reason: value}}, JournalEntries: []ERPJournalEntry{{Description: value}}}
		safe, err := pkg.ToCSV()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := pkg.ToMachineCSV()
		if err != nil {
			t.Fatal(err)
		}
		safeRows, err := csv.NewReader(bytes.NewReader(safe)).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		rawRows, err := csv.NewReader(bytes.NewReader(raw)).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		if rawRows[1][1] != value || rawRows[1][2] != value || rawRows[2][6] != value || rawRows[3][6] != value {
			t.Fatal("machine export changed source text")
		}
		if safeRows[1][3] != "12345" || safeRows[1][4] != "0" {
			t.Fatal("spreadsheet export changed numeric money")
		}
		expected := "'" + value
		// A quote followed by a newline stays inside one quoted CSV cell.
		if value == "\"\n=1+1" {
			expected = value
		}
		for _, cell := range []string{safeRows[1][1], safeRows[1][2], safeRows[1][5], safeRows[1][6], safeRows[2][6], safeRows[3][6]} {
			if cell != expected {
				t.Fatalf("text column not encoded: %q", cell)
			}
		}
	}
	if spreadsheetText("Normal trader, Ltd") != "Normal trader, Ltd" {
		t.Fatal("ordinary text changed")
	}
}
