package erp

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"kredit/internal/ledger"
)

type ContractVersion string

const (
	ContractV1 ContractVersion = "v1"
	ContractV2 ContractVersion = "v2"
)

type ERPInvoice struct {
	Reference          string       `json:"reference"`
	CustomerIdentifier string       `json:"customer_identifier"`
	CustomerName       string       `json:"customer_name"`
	AmountKobo         ledger.Money `json:"amount_kobo"`
	DueDate            string       `json:"due_date"`
	State              string       `json:"state"`
	CreatedAt          time.Time    `json:"created_at"`
}

type ERPPayment struct {
	Reference           string       `json:"reference"`
	InvoiceReference    string       `json:"invoice_reference"`
	AmountKobo          ledger.Money `json:"amount_kobo"`
	PaymentMethod       string       `json:"payment_method"`
	PaidAt              time.Time    `json:"paid_at"`
	LedgerTransactionID string       `json:"ledger_transaction_id"`
}

type ERPCreditNote struct {
	Reference        string       `json:"reference"`
	InvoiceReference string       `json:"invoice_reference"`
	AmountKobo       ledger.Money `json:"amount_kobo"`
	Reason           string       `json:"reason"`
	ApprovedBy       string       `json:"approved_by"`
	ApprovedAt       time.Time    `json:"approved_at"`
}

type ERPJournalEntry struct {
	TransactionID string       `json:"transaction_id"`
	AccountCode   string       `json:"account_code"`
	AccountName   string       `json:"account_name"`
	DebitKobo     ledger.Money `json:"debit_kobo"`
	CreditKobo    ledger.Money `json:"credit_kobo"`
	Description   string       `json:"description"`
	Timestamp     time.Time    `json:"timestamp"`
}

type ERPPackageV2 struct {
	Version        ContractVersion   `json:"version"`
	OrganizationID string            `json:"organization_id"`
	ExportedAt     time.Time         `json:"exported_at"`
	PeriodStart    time.Time         `json:"period_start"`
	PeriodEnd      time.Time         `json:"period_end"`
	Invoices       []ERPInvoice      `json:"invoices"`
	Payments       []ERPPayment      `json:"payments"`
	CreditNotes    []ERPCreditNote   `json:"credit_notes"`
	JournalEntries []ERPJournalEntry `json:"journal_entries"`
	IntegrityHash  string            `json:"integrity_hash"`
}

func (pkg *ERPPackageV2) ComputeIntegrityHash() string {
	copyPkg := *pkg
	copyPkg.IntegrityHash = ""
	raw, _ := json.Marshal(copyPkg)
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}

// ToCSV produces a spreadsheet-oriented export. Machine consumers must opt in
// to ToMachineCSV to retain exact text without presentation escaping.
func (pkg *ERPPackageV2) ToCSV() ([]byte, error) { return pkg.ToSpreadsheetCSV() }

func (pkg *ERPPackageV2) ToSpreadsheetCSV() ([]byte, error) { return pkg.toCSV(true) }

// ToMachineCSV is lossless interchange data, not safe to open as a spreadsheet.
func (pkg *ERPPackageV2) ToMachineCSV() ([]byte, error) { return pkg.toCSV(false) }

func (pkg *ERPPackageV2) toCSV(spreadsheet bool) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	write := func(row []string) error {
		if spreadsheet {
			for _, index := range []int{0, 1, 2, 5, 6} {
				row[index] = spreadsheetText(row[index])
			}
		}
		return w.Write(row)
	}

	// Header row
	if err := write([]string{"record_type", "reference", "account_or_customer", "debit_kobo", "credit_kobo", "date", "notes"}); err != nil {
		return nil, err
	}

	for _, inv := range pkg.Invoices {
		if err := write([]string{
			"INVOICE",
			inv.Reference,
			inv.CustomerName,
			strconv.FormatInt(int64(inv.AmountKobo), 10),
			"0",
			inv.DueDate,
			inv.State,
		}); err != nil {
			return nil, err
		}
	}

	for _, p := range pkg.Payments {
		if err := write([]string{
			"PAYMENT",
			p.Reference,
			p.InvoiceReference,
			"0",
			strconv.FormatInt(int64(p.AmountKobo), 10),
			p.PaidAt.Format(time.RFC3339),
			p.PaymentMethod,
		}); err != nil {
			return nil, err
		}
	}

	for _, cn := range pkg.CreditNotes {
		if err := write([]string{
			"CREDIT_NOTE",
			cn.Reference,
			cn.InvoiceReference,
			"0",
			strconv.FormatInt(int64(cn.AmountKobo), 10),
			cn.ApprovedAt.Format(time.RFC3339),
			cn.Reason,
		}); err != nil {
			return nil, err
		}
	}

	for _, j := range pkg.JournalEntries {
		if err := write([]string{
			"JOURNAL",
			j.TransactionID,
			fmt.Sprintf("%s (%s)", j.AccountCode, j.AccountName),
			strconv.FormatInt(int64(j.DebitKobo), 10),
			strconv.FormatInt(int64(j.CreditKobo), 10),
			j.Timestamp.Format(time.RFC3339),
			j.Description,
		}); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}
