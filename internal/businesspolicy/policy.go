// Package businesspolicy owns versioned operating policies. Deployment secrets,
// provider certification and accounting invariants are deliberately not policies.
package businesspolicy

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"kredit/internal/config"
)

type Values struct {
	CollectionsEnabled  bool   `json:"collections_enabled"`
	AutomaticCollection bool   `json:"automatic_collection"`
	AutomaticRetry      bool   `json:"automatic_retry"`
	PaymentClaims       bool   `json:"payment_claims"`
	NoticeHours         int64  `json:"notice_hours"`
	MaxRetries          int64  `json:"max_retries"`
	MaxSuppliers        int64  `json:"max_suppliers"`
	MaxBuyers           int64  `json:"max_buyers"`
	MaxPrincipal        int64  `json:"max_principal_kobo"`
	MaxExposure         int64  `json:"max_exposure_kobo"`
	MaxDrawdowns        int64  `json:"max_drawdowns_per_day"`
	EnhancedReview      int64  `json:"enhanced_review_kobo"`
	CorrectionThreshold int64  `json:"correction_threshold_kobo"`
	BaseFeeBPS          int64  `json:"base_fee_bps"`
	CollectionFeeBPS    int64  `json:"collection_fee_bps"`
	AllowedIndustries   string `json:"allowed_industries"`
	UpcomingNoticeDays  int64  `json:"upcoming_notice_days"`
	MandateNoticeDays   int64  `json:"mandate_notice_days"`
}

type Field struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Group string `json:"group"`
	Kind  string `json:"kind"`
	Min   int64  `json:"min"`
	Max   int64  `json:"max"`
	Help  string `json:"help"`
}

func Catalog() []Field {
	return []Field{
		{"collections_enabled", "Allow new collections", "Collections", "boolean", 0, 0, "Pausing does not stop reconciliation or reverse debits already submitted. Provider deployment approval is still required."},
		{"automatic_collection", "Collect due payments automatically", "Collections", "boolean", 0, 0, "Workers recheck this before starting each collection."},
		{"automatic_retry", "Retry eligible failed collections automatically", "Collections", "boolean", 0, 0, "Requires automatic collection. Final failures are never retried."},
		{"payment_claims", "Allow new buyer payment claims", "Collections", "boolean", 0, 0, "Existing claims can still be reviewed."},
		{"notice_hours", "Minimum delivered notice (hours)", "Collections", "number", 1, 720, "Applies to new debits; deployment notice requirements remain a minimum."},
		{"max_retries", "Maximum attempts per collection", "Collections", "number", 1, 10, "Includes the first attempt. Lowering this does not erase previous attempts."},
		{"max_suppliers", "Maximum supplier businesses", "Limits", "number", 0, 1000000000, "Zero means no additional admin cap. Deployment caps still apply."},
		{"max_buyers", "Maximum buyer businesses", "Limits", "number", 0, 1000000000, "Zero means no additional admin cap. Deployment caps still apply."},
		{"max_principal_kobo", "Maximum new principal (kobo)", "Limits", "money", 0, 9000000000000000, "100 kobo = ₦1. Zero means no additional admin cap."},
		{"max_exposure_kobo", "Maximum exposure per buyer (kobo)", "Limits", "money", 0, 9000000000000000, "Outstanding principal, released goods and pending trade-line reservations. Existing balances are preserved."},
		{"max_drawdowns_per_day", "Maximum drawdowns per line per day", "Limits", "number", 0, 1000000, "Counted by Lagos calendar day; cancelled drawdowns still count."},
		{"enhanced_review_kobo", "Enhanced review flag threshold (kobo)", "Limits", "money", 0, 9000000000000000, "Flags new credit requests for enhanced review; zero adds no admin threshold. This flag is not an approval decision."},
		{"correction_threshold_kobo", "Correction approval threshold (kobo)", "Limits", "money", 1, 1000000, "Cumulative write-offs and waivers reaching this amount are blocked pending the separate transaction approval workflow."},
		{"base_fee_bps", "Supplier base fee (basis points)", "Fees", "number", 0, 1000, "100 basis points = 1%. Recorded on new offers; existing offers retain their rates."},
		{"collection_fee_bps", "Successful collection fee (basis points)", "Fees", "number", 0, 1000, "Only successful eligible collections incur this fee, using the offer's recorded rate."},
		{"allowed_industries", "Allowed industries for new businesses", "Limits", "text", 0, 0, "Comma-separated exact industry names. Blank adds no restriction; deployment restrictions still apply."},
		{"upcoming_notice_days", "Upcoming payment reminder (days)", "Notices", "number", 1, 30, "How far ahead to enqueue upcoming payment reminders."},
		{"mandate_notice_days", "Mandate expiry reminder (days)", "Notices", "number", 1, 90, "How far ahead to enqueue mandate expiry reminders."},
	}
}
func Defaults(c config.Config) Values {
	h := c.CollectionNoticeMinHours
	if h < 1 {
		h = 24
	}
	retries := c.PilotMaxCollectionRetries
	if retries < 1 {
		retries = 3
	}
	return Values{CollectionsEnabled: true, AutomaticCollection: c.AutomaticCollectionEnabled, AutomaticRetry: c.AutomaticRetryEnabled, PaymentClaims: c.OffPlatformPaymentClaims, NoticeHours: h, MaxRetries: retries, MaxSuppliers: c.PilotMaxSupplierOrganizations, MaxBuyers: c.PilotMaxBuyerBusinesses, MaxPrincipal: c.PilotMaxPrincipalKobo, MaxExposure: c.PilotMaxActiveExposureKobo, MaxDrawdowns: c.PilotMaxDrawdownsPerLineDay, EnhancedReview: c.PilotEnhancedReviewKobo, CorrectionThreshold: 1000000, BaseFeeBPS: 50, CollectionFeeBPS: 50, AllowedIndustries: c.PilotAllowedIndustries, UpcomingNoticeDays: 3, MandateNoticeDays: 7}
}
func (v Values) Validate() error {
	data, _ := json.Marshal(v)
	var fields map[string]any
	_ = json.Unmarshal(data, &fields)
	for _, f := range Catalog() {
		if f.Kind == "number" || f.Kind == "money" {
			n := fields[f.Key].(float64)
			if n < float64(f.Min) || n > float64(f.Max) {
				return fmt.Errorf("%s must be between %d and %d", f.Label, f.Min, f.Max)
			}
		}
	}
	if v.AutomaticRetry && !v.AutomaticCollection {
		return errors.New("automatic retry requires automatic collection")
	}
	if len(v.AllowedIndustries) > 2000 {
		return errors.New("industry list is too long")
	}
	for _, part := range strings.Split(v.AllowedIndustries, ",") {
		if len(strings.TrimSpace(part)) > 100 {
			return errors.New("industry name is too long")
		}
	}
	return nil
}
func (v Values) ValidateDeployment(c config.Config) error {
	if err := v.Validate(); err != nil {
		return err
	}
	for _, p := range []struct {
		name       string
		value, cap int64
	}{{"supplier limit", v.MaxSuppliers, c.PilotMaxSupplierOrganizations}, {"buyer limit", v.MaxBuyers, c.PilotMaxBuyerBusinesses}, {"principal limit", v.MaxPrincipal, c.PilotMaxPrincipalKobo}, {"exposure limit", v.MaxExposure, c.PilotMaxActiveExposureKobo}, {"drawdown limit", v.MaxDrawdowns, c.PilotMaxDrawdownsPerLineDay}, {"attempt limit", v.MaxRetries, c.PilotMaxCollectionRetries}} {
		if p.cap > 0 && (p.value == 0 || p.value > p.cap) {
			return fmt.Errorf("%s exceeds the deployment approval ceiling (%d)", p.name, p.cap)
		}
	}
	if (c.RealCollections || c.MonoSweepEnabled) && v.NoticeHours < c.CollectionNoticeMinHours {
		return errors.New("notice period is below the deployment requirement")
	}
	if c.PilotEnhancedReviewKobo > 0 && (v.EnhancedReview == 0 || v.EnhancedReview > c.PilotEnhancedReviewKobo) {
		return errors.New("enhanced review threshold exceeds deployment approval")
	}
	if strings.TrimSpace(c.PilotAllowedIndustries) != "" {
		if strings.TrimSpace(v.AllowedIndustries) == "" {
			return errors.New("deployment requires an industry allowlist")
		}
		for _, industry := range strings.Split(v.AllowedIndustries, ",") {
			if !AllowsIndustry(c.PilotAllowedIndustries, industry) {
				return errors.New("industry is outside deployment approval")
			}
		}
	}
	return nil
}
func AllowsIndustry(list, industry string) bool {
	if strings.TrimSpace(list) == "" {
		return true
	}
	for _, s := range strings.Split(list, ",") {
		if strings.EqualFold(strings.TrimSpace(s), strings.TrimSpace(industry)) {
			return true
		}
	}
	return false
}
func Changed(a, b Values) bool { return !reflect.DeepEqual(a, b) }

// A proposal is a complete reviewed snapshot. Missing/null values must not
// silently turn switches off or remove numeric limits during decoding.
func (v *Values) UnmarshalJSON(data []byte) error {
	type plain Values
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for _, f := range Catalog() {
		b, ok := raw[f.Key]
		if !ok || string(b) == "null" {
			return fmt.Errorf("%s is required", f.Label)
		}
	}
	if len(raw) != len(Catalog()) {
		return errors.New("unknown business policy setting")
	}
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*v = Values(decoded)
	return nil
}
