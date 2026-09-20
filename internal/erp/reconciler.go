package erp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"kredit/internal/ledger"
)

type DiscrepancyType string

const (
	DiscrepancyMissingExternal DiscrepancyType = "missing_external"
	DiscrepancyMissingInternal DiscrepancyType = "missing_internal"
	DiscrepancyAmountMismatch  DiscrepancyType = "amount_mismatch"
	DiscrepancyTimingDrift     DiscrepancyType = "timing_drift"
)

type Discrepancy struct {
	Type               DiscrepancyType `json:"type"`
	Reference          string          `json:"reference"`
	ExpectedAmountKobo ledger.Money    `json:"expected_amount_kobo"`
	ActualAmountKobo   ledger.Money    `json:"actual_amount_kobo"`
	DifferenceKobo     ledger.Money    `json:"difference_kobo"`
	Description        string          `json:"description"`
}

type ReconciliationResult struct {
	OrganizationID     string        `json:"organization_id"`
	ReconciledAt       time.Time     `json:"reconciled_at"`
	PeriodStart        time.Time     `json:"period_start"`
	PeriodEnd          time.Time     `json:"period_end"`
	TotalInternalKobo  ledger.Money  `json:"total_internal_kobo"`
	TotalExternalKobo  ledger.Money  `json:"total_external_kobo"`
	NetVarianceKobo    ledger.Money  `json:"net_variance_kobo"`
	MatchedRecordCount int           `json:"matched_record_count"`
	Discrepancies      []Discrepancy `json:"discrepancies"`
	Balanced           bool          `json:"balanced"`
	ReportHash         string        `json:"report_hash"`
}

type ExternalRecord struct {
	Reference  string       `json:"reference"`
	AmountKobo ledger.Money `json:"amount_kobo"`
	Date       time.Time    `json:"date"`
	Note       string       `json:"note"`
}

type InternalRecord struct {
	Reference     string       `json:"reference"`
	TransactionID string       `json:"transaction_id"`
	AmountKobo    ledger.Money `json:"amount_kobo"`
	Date          time.Time    `json:"date"`
}

// MaxReconciliationRecords bounds both report size and untrusted input work.
const MaxReconciliationRecords = 10000

var ErrInvalidRecords = errors.New("invalid reconciliation records")

// ReconcileExternal compares unique signed movements, not outstanding balances.
// Both interval endpoints are inclusive. Reference is a journal transaction ID
// for the HTTP ledger comparison. Ambiguous duplicates are never overwritten.
func ReconcileExternal(orgID string, periodStart, periodEnd time.Time, internal []InternalRecord, external []ExternalRecord) (ReconciliationResult, error) {
	invalid := func(message string) (ReconciliationResult, error) {
		return ReconciliationResult{}, fmt.Errorf("%w: %s", ErrInvalidRecords, message)
	}
	if strings.TrimSpace(orgID) == "" || periodStart.IsZero() || periodEnd.IsZero() || periodEnd.Before(periodStart) {
		return invalid("organization and a valid period are required")
	}
	if len(internal) > MaxReconciliationRecords || len(external) > MaxReconciliationRecords {
		return invalid("too many records; select a smaller period")
	}
	// Copy before sorting: callers retain ownership of their input slices.
	ins := append([]InternalRecord{}, internal...)
	exts := append([]ExternalRecord{}, external...)
	sort.Slice(ins, func(i, j int) bool { return ins[i].Reference < ins[j].Reference })
	sort.Slice(exts, func(i, j int) bool { return exts[i].Reference < exts[j].Reference })
	valid := func(ref string, at time.Time) bool {
		return strings.TrimSpace(ref) != "" && len(ref) <= 200 && !at.IsZero() && !at.Before(periodStart) && !at.After(periodEnd)
	}
	result := ReconciliationResult{
		OrganizationID: orgID, ReconciledAt: time.Now().UTC(),
		PeriodStart: periodStart.UTC(), PeriodEnd: periodEnd.UTC(),
		Discrepancies: []Discrepancy{}, Balanced: true,
	}
	intMap := make(map[string]InternalRecord, len(ins))
	extMap := make(map[string]ExternalRecord, len(exts))
	var err error
	for i := range ins {
		in := &ins[i]
		if !valid(in.Reference, in.Date) {
			return invalid("internal reference or date is invalid")
		}
		if _, exists := intMap[in.Reference]; exists {
			return invalid("duplicate internal reference")
		}
		in.Date = in.Date.UTC()
		intMap[in.Reference] = *in
		result.TotalInternalKobo, err = ledger.CheckedAdd(result.TotalInternalKobo, in.AmountKobo)
		if err != nil {
			return invalid("internal total overflows money range")
		}
	}
	for i := range exts {
		ext := &exts[i]
		if !valid(ext.Reference, ext.Date) || len(ext.Note) > 2000 {
			return invalid("external reference, date or note is invalid")
		}
		if _, exists := extMap[ext.Reference]; exists {
			return invalid("duplicate external reference")
		}
		ext.Date = ext.Date.UTC()
		extMap[ext.Reference] = *ext
		result.TotalExternalKobo, err = ledger.CheckedAdd(result.TotalExternalKobo, ext.AmountKobo)
		if err != nil {
			return invalid("external total overflows money range")
		}
	}
	for _, in := range ins {
		ext, exists := extMap[in.Reference]
		if !exists {
			result.Discrepancies = append(result.Discrepancies, Discrepancy{
				Type: DiscrepancyMissingExternal, Reference: in.Reference,
				ExpectedAmountKobo: in.AmountKobo, DifferenceKobo: in.AmountKobo,
				Description: "Internal movement is missing from external ERP",
			})
			continue
		}
		if in.AmountKobo == ext.AmountKobo {
			result.MatchedRecordCount++
			continue
		}
		difference, err := checkedDifference(in.AmountKobo, ext.AmountKobo)
		if err != nil {
			return invalid("record variance overflows money range")
		}
		result.Discrepancies = append(result.Discrepancies, Discrepancy{
			Type: DiscrepancyAmountMismatch, Reference: in.Reference,
			ExpectedAmountKobo: in.AmountKobo, ActualAmountKobo: ext.AmountKobo,
			DifferenceKobo: difference, Description: "Signed movement amounts differ",
		})
	}
	for _, ext := range exts {
		if _, exists := intMap[ext.Reference]; exists {
			continue
		}
		difference, err := checkedDifference(0, ext.AmountKobo)
		if err != nil {
			return invalid("missing-record variance overflows money range")
		}
		result.Discrepancies = append(result.Discrepancies, Discrepancy{
			Type: DiscrepancyMissingInternal, Reference: ext.Reference,
			ActualAmountKobo: ext.AmountKobo, DifferenceKobo: difference,
			Description: "External movement is missing from internal ledger",
		})
	}
	result.NetVarianceKobo, err = checkedDifference(result.TotalInternalKobo, result.TotalExternalKobo)
	if err != nil {
		return invalid("net variance overflows money range")
	}
	result.Balanced = result.NetVarianceKobo == 0 && len(result.Discrepancies) == 0
	sort.Slice(result.Discrepancies, func(i, j int) bool {
		return result.Discrepancies[i].Reference < result.Discrepancies[j].Reference
	})
	// Bind the actual period, identities, amounts, dates, notes and findings.
	// Generation time is excluded so the same evidence has a reproducible hash.
	stable := result
	stable.ReconciledAt = time.Time{}
	raw, err := json.Marshal(struct {
		Version  string               `json:"version"`
		Report   ReconciliationResult `json:"report"`
		Internal []InternalRecord     `json:"internal"`
		External []ExternalRecord     `json:"external"`
	}{"erp-reconciliation-v2", stable, ins, exts})
	if err != nil {
		return invalid("record dates cannot be encoded")
	}
	h := sha256.Sum256(raw)
	result.ReportHash = hex.EncodeToString(h[:])
	return result, nil
}

func checkedDifference(left, right ledger.Money) (ledger.Money, error) {
	if (right > 0 && left < math.MinInt64+right) || (right < 0 && left > math.MaxInt64+right) {
		return 0, ErrInvalidRecords
	}
	return left - right, nil
}
