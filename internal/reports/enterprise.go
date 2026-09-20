package reports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"sort"
	"strconv"
	"time"

	"kredit/internal/ledger"
)

type BranchExposure struct {
	BranchID             string                  `json:"branch_id"`
	BranchName           string                  `json:"branch_name"`
	Territory            string                  `json:"territory"`
	TotalOutstandingKobo ledger.Money            `json:"total_outstanding_kobo"`
	AgeingBuckets        map[string]ledger.Money `json:"ageing_buckets"`
	CustomerCount        int64                   `json:"customer_count"`
	OverdueCount         int64                   `json:"overdue_count"`
}

type PortfolioHealth struct {
	OnTimeCollectionBPS int64  `json:"on_time_collection_bps"`
	DisputedRatioBPS    int64  `json:"disputed_ratio_bps"`
	TopConcentrationBPS int64  `json:"top_concentration_bps"`
	HealthRating        string `json:"health_rating"`
}

type EnterpriseReport struct {
	OrganizationID  string           `json:"organization_id"`
	GeneratedAt     time.Time        `json:"generated_at"`
	Summary         Summary          `json:"summary"`
	PortfolioHealth PortfolioHealth  `json:"portfolio_health"`
	BranchExposures []BranchExposure `json:"branch_exposures"`
	IntegrityHash   string           `json:"integrity_hash"`
}

// EnterpriseReport uses one tenant-scoped snapshot for both money and branch
// assignments. Report failure must not silently invent a headquarters balance.
func (s *Store) EnterpriseReport(ctx context.Context, orgID string) (EnterpriseReport, error) {
	snapshot, branches, assignments, err := s.enterpriseSnapshot(ctx, orgID)
	if err != nil {
		return EnterpriseReport{}, err
	}
	receivables, err := snapshot.ReceivablesForSupplier(ctx, orgID)
	if err != nil {
		return EnterpriseReport{}, err
	}
	buckets := make(map[string]map[string]ledger.Money, len(receivables.Rows))
	for _, row := range receivables.Rows {
		schedule, items, e := snapshot.source.Schedule(row.ObligationID)
		if e != nil {
			return EnterpriseReport{}, e
		}
		buckets[row.ObligationID], e = enterpriseAgeing(row.OutstandingKobo, items, schedule.GraceHours, receivables.GeneratedAt)
		if e != nil {
			return EnterpriseReport{}, e
		}
	}
	return buildEnterpriseReport(orgID, receivables, branches, assignments, buckets)
}

func newBranchExposure(id, name, territory string) BranchExposure {
	return BranchExposure{BranchID: id, BranchName: name, Territory: territory,
		AgeingBuckets: map[string]ledger.Money{"0-30": 0, "31-60": 0, "61-90": 0, "90+": 0}}
}

func buildEnterpriseReport(orgID string, receivables Receivables, branches []BranchExposure, assignments map[string]string, buckets map[string]map[string]ledger.Money) (EnterpriseReport, error) {
	report := EnterpriseReport{OrganizationID: orgID, GeneratedAt: receivables.GeneratedAt, Summary: receivables.Summary, BranchExposures: []BranchExposure{}}
	branchMap := map[string]BranchExposure{"headquarters": newBranchExposure("headquarters", "Headquarters / Direct Channel", "National")}
	for _, b := range branches {
		if b.BranchID == "" || b.BranchID == "headquarters" {
			return EnterpriseReport{}, errors.New("invalid branch identity")
		}
		if _, exists := branchMap[b.BranchID]; exists {
			return EnterpriseReport{}, errors.New("duplicate branch identity")
		}
		branchMap[b.BranchID] = newBranchExposure(b.BranchID, b.BranchName, b.Territory)
	}
	customerTotals := map[string]ledger.Money{}
	branchCustomers := map[string]map[string]bool{}
	var onTimeCollected, disputedAmount, maxCustomerOutstanding, totalOutstanding ledger.Money
	for _, row := range receivables.Rows {
		if row.OutstandingKobo < 0 || row.VoluntaryPaidKobo < 0 || row.CollectedPaidKobo < 0 {
			return EnterpriseReport{}, errors.New("invalid portfolio amounts")
		}
		var err error
		customerTotals[row.BuyerBusinessID], err = ledger.CheckedAdd(customerTotals[row.BuyerBusinessID], row.OutstandingKobo)
		if err != nil {
			return EnterpriseReport{}, err
		}
		maxCustomerOutstanding = max(maxCustomerOutstanding, customerTotals[row.BuyerBusinessID])
		// Preserve the published originated-principal denominator. This is a
		// conservative collected share, not proof that every payment was timely.
		// Fully paid but late obligations must never count as on-time collections.
		if !row.Overdue && !row.LatePayment {
			paid, e := ledger.CheckedAdd(row.VoluntaryPaidKobo, row.CollectedPaidKobo)
			if e != nil {
				return EnterpriseReport{}, e
			}
			onTimeCollected, err = ledger.CheckedAdd(onTimeCollected, paid)
			if err != nil {
				return EnterpriseReport{}, err
			}
		}
		if row.OpenDisputeCount > 0 {
			disputedAmount, err = ledger.CheckedAdd(disputedAmount, row.OutstandingKobo)
			if err != nil {
				return EnterpriseReport{}, err
			}
		}
		id := assignments[row.BuyerBusinessID]
		if id == "" {
			id = "headquarters"
		}
		branch, exists := branchMap[id]
		if !exists {
			return EnterpriseReport{}, errors.New("assigned branch is unavailable")
		}
		branch.TotalOutstandingKobo, err = ledger.CheckedAdd(branch.TotalOutstandingKobo, row.OutstandingKobo)
		if err != nil {
			return EnterpriseReport{}, err
		}
		if row.Overdue {
			branch.OverdueCount++
		}
		var classified ledger.Money
		for bucket, amount := range buckets[row.ObligationID] {
			if _, exists := branch.AgeingBuckets[bucket]; !exists || amount < 0 {
				return EnterpriseReport{}, errors.New("invalid enterprise ageing bucket")
			}
			classified, err = ledger.CheckedAdd(classified, amount)
			if err != nil {
				return EnterpriseReport{}, err
			}
			branch.AgeingBuckets[bucket], err = ledger.CheckedAdd(branch.AgeingBuckets[bucket], amount)
			if err != nil {
				return EnterpriseReport{}, err
			}
		}
		if classified != row.OutstandingKobo {
			return EnterpriseReport{}, errors.New("ageing does not cover the outstanding balance")
		}
		if branchCustomers[id] == nil {
			branchCustomers[id] = map[string]bool{}
		}
		branchCustomers[id][row.BuyerBusinessID] = true
		branchMap[id] = branch
		totalOutstanding, err = ledger.CheckedAdd(totalOutstanding, row.OutstandingKobo)
		if err != nil {
			return EnterpriseReport{}, err
		}
	}
	if totalOutstanding != report.Summary.OutstandingKobo {
		return EnterpriseReport{}, errors.New("portfolio summary does not match its rows")
	}
	var err error
	report.PortfolioHealth.OnTimeCollectionBPS, err = portfolioBPS(onTimeCollected, report.Summary.PrincipalKobo)
	if err != nil {
		return EnterpriseReport{}, err
	}
	report.PortfolioHealth.DisputedRatioBPS, err = portfolioBPS(disputedAmount, totalOutstanding)
	if err != nil {
		return EnterpriseReport{}, err
	}
	report.PortfolioHealth.TopConcentrationBPS, err = portfolioBPS(maxCustomerOutstanding, totalOutstanding)
	if err != nil {
		return EnterpriseReport{}, err
	}
	switch {
	case report.Summary.PrincipalKobo == 0:
		report.PortfolioHealth.HealthRating = "NO_DATA"
	case report.PortfolioHealth.DisputedRatioBPS < 500 && report.PortfolioHealth.OnTimeCollectionBPS >= 9000:
		report.PortfolioHealth.HealthRating = "EXCELLENT"
	case report.PortfolioHealth.DisputedRatioBPS < 1500 && report.PortfolioHealth.OnTimeCollectionBPS >= 7500:
		report.PortfolioHealth.HealthRating = "HEALTHY"
	case report.PortfolioHealth.DisputedRatioBPS < 3000:
		report.PortfolioHealth.HealthRating = "WATCH"
	default:
		report.PortfolioHealth.HealthRating = "STRESSED"
	}
	for id, b := range branchMap {
		b.CustomerCount = int64(len(branchCustomers[id]))
		if b.TotalOutstandingKobo > 0 || id == "headquarters" {
			report.BranchExposures = append(report.BranchExposures, b)
		}
	}
	sort.Slice(report.BranchExposures, func(i, j int) bool { return report.BranchExposures[i].BranchID < report.BranchExposures[j].BranchID })
	stable := report
	stable.GeneratedAt = time.Time{}
	data, err := json.Marshal(stable)
	if err != nil {
		return EnterpriseReport{}, err
	}
	h := sha256.Sum256(data)
	report.IntegrityHash = hex.EncodeToString(h[:])
	return report, nil
}

// Ratios must not multiply an int64 money value by 10,000 before division.
func portfolioBPS(numerator, denominator ledger.Money) (int64, error) {
	if numerator < 0 || denominator < 0 || numerator > denominator {
		return 0, errors.New("invalid portfolio ratio")
	}
	if denominator == 0 {
		return 0, nil
	}
	n := big.NewInt(int64(numerator))
	n.Mul(n, big.NewInt(10000))
	n.Quo(n, big.NewInt(int64(denominator)))
	return n.Int64(), nil
}

func (s *Store) ExportEnterpriseCSV(ctx context.Context, orgID string) ([]byte, string, error) {
	report, err := s.EnterpriseReport(ctx, orgID)
	if err != nil {
		return nil, "", err
	}
	return enterpriseCSV(report)
}

func enterpriseCSV(report EnterpriseReport) ([]byte, string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	rows := [][]string{
		{"# Enterprise Portfolio Exposure Report", csvText(report.OrganizationID), report.GeneratedAt.Format(time.RFC3339)},
		{"# Health Rating", report.PortfolioHealth.HealthRating, "Integrity Hash", report.IntegrityHash},
		{},
		{"Branch ID", "Branch Name", "Territory", "Total Outstanding (Kobo)", "Current / 0-30 Days", "31-60 Days", "61-90 Days", "Over 90 Days", "Customers", "Overdue Count"},
	}
	for _, b := range report.BranchExposures {
		rows = append(rows, []string{csvText(b.BranchID), csvText(b.BranchName), csvText(b.Territory),
			strconv.FormatInt(int64(b.TotalOutstandingKobo), 10), strconv.FormatInt(int64(b.AgeingBuckets["0-30"]), 10),
			strconv.FormatInt(int64(b.AgeingBuckets["31-60"]), 10), strconv.FormatInt(int64(b.AgeingBuckets["61-90"]), 10),
			strconv.FormatInt(int64(b.AgeingBuckets["90+"]), 10), strconv.FormatInt(b.CustomerCount, 10), strconv.FormatInt(b.OverdueCount, 10)})
	}
	if err := w.WriteAll(rows); err != nil {
		return nil, "", err
	}
	content := buf.Bytes()
	h := sha256.Sum256(content)
	// An unkeyed checksum is not an authenticated signature.
	return content, hex.EncodeToString(h[:]), nil
}
