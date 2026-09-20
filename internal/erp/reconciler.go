package erp

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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

// ReconcileExternal compares external ERP entries against internal ledger postings
func ReconcileExternal(orgID string, periodStart, periodEnd time.Time, internal []InternalRecord, external []ExternalRecord) (ReconciliationResult, error) {
	if orgID == "" {
		return ReconciliationResult{}, errors.New("organization ID is required")
	}

	result := ReconciliationResult{
		OrganizationID: orgID,
		ReconciledAt:   time.Now().UTC(),
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		Discrepancies:  []Discrepancy{},
		Balanced:       true,
	}

	extMap := make(map[string]ExternalRecord, len(external))
	for _, ext := range external {
		extMap[ext.Reference] = ext
		result.TotalExternalKobo += ext.AmountKobo
	}

	intMap := make(map[string]InternalRecord, len(internal))
	for _, in := range internal {
		intMap[in.Reference] = in
		result.TotalInternalKobo += in.AmountKobo
	}

	// Check all internal records against external
	for ref, in := range intMap {
		ext, exists := extMap[ref]
		if !exists {
			result.Balanced = false
			result.Discrepancies = append(result.Discrepancies, Discrepancy{
				Type:               DiscrepancyMissingExternal,
				Reference:          ref,
				ExpectedAmountKobo: in.AmountKobo,
				ActualAmountKobo:   0,
				DifferenceKobo:     in.AmountKobo,
				Description:        fmt.Sprintf("Internal ledger transaction %s is missing from external ERP", in.TransactionID),
			})
			continue
		}

		if in.AmountKobo != ext.AmountKobo {
			result.Balanced = false
			diff := in.AmountKobo - ext.AmountKobo
			result.Discrepancies = append(result.Discrepancies, Discrepancy{
				Type:               DiscrepancyAmountMismatch,
				Reference:          ref,
				ExpectedAmountKobo: in.AmountKobo,
				ActualAmountKobo:   ext.AmountKobo,
				DifferenceKobo:     diff,
				Description:        fmt.Sprintf("Amount mismatch: internal %d kobo vs external %d kobo", in.AmountKobo, ext.AmountKobo),
			})
		} else {
			result.MatchedRecordCount++
		}
	}

	// Check external records missing from internal
	for ref, ext := range extMap {
		if _, exists := intMap[ref]; !exists {
			result.Balanced = false
			result.Discrepancies = append(result.Discrepancies, Discrepancy{
				Type:               DiscrepancyMissingInternal,
				Reference:          ref,
				ExpectedAmountKobo: 0,
				ActualAmountKobo:   ext.AmountKobo,
				DifferenceKobo:     -ext.AmountKobo,
				Description:        fmt.Sprintf("External ERP entry %s not found in internal double-entry ledger", ref),
			})
		}
	}

	result.NetVarianceKobo = result.TotalInternalKobo - result.TotalExternalKobo
	if result.NetVarianceKobo != 0 || len(result.Discrepancies) > 0 {
		result.Balanced = false
	}

	hashInput := fmt.Sprintf("%s:%d:%d:%d:%d", orgID, result.TotalInternalKobo, result.TotalExternalKobo, result.MatchedRecordCount, len(result.Discrepancies))
	h := sha256.Sum256([]byte(hashInput))
	result.ReportHash = hex.EncodeToString(h[:])

	return result, nil
}
