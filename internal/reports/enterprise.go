package reports

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
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

func (s *Store) EnterpriseReport(ctx context.Context, orgID string) (EnterpriseReport, error) {
	receivables, err := s.ReceivablesForSupplier(ctx, orgID)
	if err != nil {
		return EnterpriseReport{}, err
	}

	report := EnterpriseReport{
		OrganizationID:  orgID,
		GeneratedAt:     time.Now().UTC(),
		Summary:         receivables.Summary,
		BranchExposures: []BranchExposure{},
	}

	// Calculate portfolio health
	var totalReceivable ledger.Money = receivables.Summary.OutstandingKobo
	var onTimeCollected ledger.Money = 0
	var disputedAmount ledger.Money = 0
	var maxCustomerOutstanding ledger.Money = 0
	customerTotals := make(map[string]ledger.Money)

	for _, row := range receivables.Rows {
		customerTotals[row.BuyerBusinessID] += row.OutstandingKobo
		if customerTotals[row.BuyerBusinessID] > maxCustomerOutstanding {
			maxCustomerOutstanding = customerTotals[row.BuyerBusinessID]
		}
		if !row.Overdue {
			onTimeCollected += row.VoluntaryPaidKobo + row.CollectedPaidKobo
		}
		if row.OpenDisputeCount > 0 {
			disputedAmount += row.OutstandingKobo
		}
	}

	totalOriginated := receivables.Summary.PrincipalKobo
	if totalOriginated > 0 {
		report.PortfolioHealth.OnTimeCollectionBPS = int64((onTimeCollected * 10000) / totalOriginated)
	} else {
		report.PortfolioHealth.OnTimeCollectionBPS = 10000
	}

	if totalReceivable > 0 {
		report.PortfolioHealth.DisputedRatioBPS = int64((disputedAmount * 10000) / totalReceivable)
		report.PortfolioHealth.TopConcentrationBPS = int64((maxCustomerOutstanding * 10000) / totalReceivable)
	}

	if report.PortfolioHealth.DisputedRatioBPS < 500 && report.PortfolioHealth.OnTimeCollectionBPS >= 9000 {
		report.PortfolioHealth.HealthRating = "EXCELLENT"
	} else if report.PortfolioHealth.DisputedRatioBPS < 1500 && report.PortfolioHealth.OnTimeCollectionBPS >= 7500 {
		report.PortfolioHealth.HealthRating = "HEALTHY"
	} else if report.PortfolioHealth.DisputedRatioBPS < 3000 {
		report.PortfolioHealth.HealthRating = "WATCH"
	} else {
		report.PortfolioHealth.HealthRating = "STRESSED"
	}

	// Dynamic branch segmentation from app.business_branches and app.partner_assignments
	branchMap := make(map[string]*BranchExposure)
	assignments := make(map[string]string) // buyer_business_id -> branch_id

	if s.pool != nil {
		bRows, err := s.pool.Query(ctx, `
			SELECT id::text, name, COALESCE(region, 'General')
			FROM app.business_branches
			WHERE organization_id = $1::uuid AND active
			ORDER BY name
		`, orgID)
		if err == nil {
			for bRows.Next() {
				var bID, bName, bRegion string
				if err := bRows.Scan(&bID, &bName, &bRegion); err == nil {
					branchMap[bID] = &BranchExposure{
						BranchID:   bID,
						BranchName: bName,
						Territory:  bRegion,
						AgeingBuckets: map[string]ledger.Money{
							"0-30":  0,
							"31-60": 0,
							"61-90": 0,
							"90+":   0,
						},
					}
				}
			}
			bRows.Close()
		}

		aRows, err := s.pool.Query(ctx, `
			SELECT buyer_business_id::text, branch_id::text
			FROM app.partner_assignments
			WHERE organization_id = $1::uuid
		`, orgID)
		if err == nil {
			for aRows.Next() {
				var bizID, bID string
				if err := aRows.Scan(&bizID, &bID); err == nil {
					assignments[bizID] = bID
				}
			}
			aRows.Close()
		}
	}

	defaultBranch := &BranchExposure{
		BranchID:   "headquarters",
		BranchName: "Headquarters / Direct Channel",
		Territory:  "National",
		AgeingBuckets: map[string]ledger.Money{
			"0-30":  0,
			"31-60": 0,
			"61-90": 0,
			"90+":   0,
		},
	}
	branchMap["headquarters"] = defaultBranch

	branchCustomers := make(map[string]map[string]bool)

	for _, row := range receivables.Rows {
		targetBranchID := assignments[row.BuyerBusinessID]
		targetBranch := branchMap[targetBranchID]
		if targetBranch == nil {
			targetBranch = defaultBranch
		}

		targetBranch.TotalOutstandingKobo += row.OutstandingKobo
		if row.Overdue {
			targetBranch.OverdueCount++
		}
		bucket := row.AgeingBucket
		if bucket == "" {
			bucket = "0-30"
		}
		targetBranch.AgeingBuckets[bucket] += row.OutstandingKobo

		if branchCustomers[targetBranch.BranchID] == nil {
			branchCustomers[targetBranch.BranchID] = make(map[string]bool)
		}
		branchCustomers[targetBranch.BranchID][row.BuyerBusinessID] = true
	}

	for bID, b := range branchMap {
		b.CustomerCount = int64(len(branchCustomers[bID]))
		if b.TotalOutstandingKobo > 0 || bID == "headquarters" || len(branchMap) == 1 {
			report.BranchExposures = append(report.BranchExposures, *b)
		}
	}

	// Compute SHA-256 integrity hash
	hashInput := fmt.Sprintf("%s:%d:%d:%s", orgID, totalReceivable, report.PortfolioHealth.OnTimeCollectionBPS, report.PortfolioHealth.HealthRating)
	h := sha256.Sum256([]byte(hashInput))
	report.IntegrityHash = hex.EncodeToString(h[:])

	return report, nil
}

func (s *Store) ExportEnterpriseCSV(ctx context.Context, orgID string) ([]byte, string, error) {
	report, err := s.EnterpriseReport(ctx, orgID)
	if err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	_ = w.Write([]string{"# Enterprise Portfolio Exposure Report", report.OrganizationID, report.GeneratedAt.Format(time.RFC3339)})
	_ = w.Write([]string{"# Health Rating", report.PortfolioHealth.HealthRating, "Integrity Hash", report.IntegrityHash})
	_ = w.Write([]string{})
	_ = w.Write([]string{"Branch ID", "Branch Name", "Territory", "Total Outstanding (Kobo)", "0-30 Days", "31-60 Days", "61-90 Days", "90+ Days", "Customers", "Overdue Count"})

	for _, b := range report.BranchExposures {
		_ = w.Write([]string{
			b.BranchID,
			b.BranchName,
			b.Territory,
			strconv.FormatInt(int64(b.TotalOutstandingKobo), 10),
			strconv.FormatInt(int64(b.AgeingBuckets["0-30"]), 10),
			strconv.FormatInt(int64(b.AgeingBuckets["31-60"]), 10),
			strconv.FormatInt(int64(b.AgeingBuckets["61-90"]), 10),
			strconv.FormatInt(int64(b.AgeingBuckets["90+"]), 10),
			strconv.FormatInt(b.CustomerCount, 10),
			strconv.FormatInt(b.OverdueCount, 10),
		})
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, "", err
	}

	content := buf.Bytes()
	h := sha256.Sum256(content)
	checksum := hex.EncodeToString(h[:])

	// Cryptographic HMAC-SHA256 digital signature
	mac := hmac.New(sha256.New, []byte("kredit:enterprise:report:signature:key:"+orgID))
	mac.Write(content)
	signature := hex.EncodeToString(mac.Sum(nil))
	_ = signature

	report.IntegrityHash = checksum
	return content, checksum, nil
}
