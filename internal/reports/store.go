package reports

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"kredit/internal/credit"
	"kredit/internal/disputes"
	"kredit/internal/identifier"
	"kredit/internal/ledger"
	"kredit/internal/payments"
	"kredit/internal/schedules"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Source is deliberately read-only. Reports are projections of authoritative
// credit, payment, schedule, dispute and ledger state; they do not maintain a
// second balance.
type Source struct {
	SupplierViews func(string) []credit.View
	BuyerViews    func(string) []credit.View
	Payments      func(string) []payments.Payment
	Schedule      func(string) (schedules.Schedule, []schedules.Item, error)
	Disputes      func(string) []disputes.Dispute
	Now           func() time.Time
}

type Summary struct {
	ObligationCount    int64        `json:"obligation_count"`
	PrincipalKobo      ledger.Money `json:"principal_kobo"`
	OutstandingKobo    ledger.Money `json:"outstanding_kobo"`
	DueTodayKobo       ledger.Money `json:"due_today_kobo"`
	DueThisWeekKobo    ledger.Money `json:"due_this_week_kobo"`
	OverdueKobo        ledger.Money `json:"overdue_kobo"`
	VoluntaryPaidKobo  ledger.Money `json:"voluntary_paid_kobo"`
	CollectedPaidKobo  ledger.Money `json:"collected_paid_kobo"`
	BaseFeesKobo       ledger.Money `json:"base_fees_kobo"`
	CollectionFeesKobo ledger.Money `json:"collection_fees_kobo"`
	OpenDisputeCount   int64        `json:"open_dispute_count"`
	MandateIssueCount  int64        `json:"mandate_issue_count"`
}

type ObligationRow struct {
	ObligationID               string       `json:"obligation_id"`
	CreditRequestID            string       `json:"credit_request_id"`
	BuyerBusinessID            string       `json:"buyer_business_id"`
	BuyerName                  string       `json:"buyer_name"`
	PrincipalKobo              ledger.Money `json:"principal_kobo"`
	OutstandingKobo            ledger.Money `json:"outstanding_kobo"`
	BaseFeeKobo                ledger.Money `json:"base_fee_kobo"`
	DueDate                    string       `json:"due_date"`
	NextDueAt                  *time.Time   `json:"next_due_at,omitempty"`
	PaymentStatus              string       `json:"payment_status"`
	AgeingBucket               string       `json:"ageing_bucket"`
	Overdue                    bool         `json:"overdue"`
	LatePayment                bool         `json:"late_payment"`
	DaysLate                   int64        `json:"days_late"`
	VoluntaryPaidKobo          ledger.Money `json:"voluntary_paid_kobo"`
	CollectedPaidKobo          ledger.Money `json:"collected_paid_kobo"`
	CollectionFeesKobo         ledger.Money `json:"collection_fees_kobo"`
	OpenDisputeCount           int64        `json:"open_dispute_count"`
	MandateIssue               bool         `json:"mandate_issue"`
	MandateCancelledWhileOwing bool         `json:"mandate_cancelled_while_owing"`
	ActivatedAt                time.Time    `json:"activated_at"`
}

type Receivables struct {
	GeneratedAt time.Time       `json:"generated_at"`
	Currency    string          `json:"currency"`
	Summary     Summary         `json:"summary"`
	Rows        []ObligationRow `json:"rows"`
}

type Ageing struct {
	GeneratedAt time.Time               `json:"generated_at"`
	Currency    string                  `json:"currency"`
	Buckets     map[string]ledger.Money `json:"buckets"`
	Rows        []ObligationRow         `json:"rows"`
}

type Fees struct {
	GeneratedAt        time.Time       `json:"generated_at"`
	Currency           string          `json:"currency"`
	BaseFeesKobo       ledger.Money    `json:"base_fees_kobo"`
	CollectionFeesKobo ledger.Money    `json:"collection_fees_kobo"`
	TotalFeesKobo      ledger.Money    `json:"total_fees_kobo"`
	ByObligation       []ObligationFee `json:"by_obligation"`
}

type ObligationFee struct {
	ObligationID       string       `json:"obligation_id"`
	BuyerName          string       `json:"buyer_name"`
	BaseFeeKobo        ledger.Money `json:"base_fee_kobo"`
	CollectionFeesKobo ledger.Money `json:"collection_fees_kobo"`
	TotalFeesKobo      ledger.Money `json:"total_fees_kobo"`
}

type History struct {
	GeneratedAt                  time.Time       `json:"generated_at"`
	VerifiedSince                *time.Time      `json:"verified_since,omitempty"`
	ActiveObligations            int64           `json:"active_obligations"`
	CompletedObligations         int64           `json:"completed_obligations"`
	TotalCompletedPrincipalKobo  ledger.Money    `json:"total_completed_principal_kobo"`
	CurrentActivePrincipalKobo   ledger.Money    `json:"current_active_principal_kobo"`
	LargestCompletedAmountKobo   ledger.Money    `json:"largest_completed_amount_kobo"`
	OnTimeCount                  int64           `json:"on_time_count"`
	OnTimePercentage             float64         `json:"on_time_percentage"`
	LatePaymentCount             int64           `json:"late_payment_count"`
	AverageDaysLate              float64         `json:"average_days_late"`
	UnresolvedOverdueObligations int64           `json:"unresolved_overdue_obligations"`
	DisputeCount                 int64           `json:"dispute_count"`
	MandateCancellationsOwing    int64           `json:"mandate_cancellations_while_owing"`
	Obligations                  []ObligationRow `json:"obligations"`
	Shareable                    bool            `json:"shareable"`
	Score                        *float64        `json:"score"` // always nil: factual history is not a score
}

type Statement struct {
	GeneratedAt time.Time          `json:"generated_at"`
	Currency    string             `json:"currency"`
	BuyerID     string             `json:"buyer_id"`
	Rows        []ObligationRow    `json:"obligations"`
	Payments    []payments.Payment `json:"payments"`
}

type Store struct {
	source Source
	pool   *pgxpool.Pool
	mu     sync.RWMutex
	events []AnalyticsEvent
}

func NewPostgresStore(pool *pgxpool.Pool, source Source) *Store {
	store := NewStore(source)
	store.pool = pool
	return store
}

type AnalyticsEvent struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	SubjectID          string         `json:"subject_id"`
	Purpose            string         `json:"purpose"`
	At                 time.Time      `json:"at"`
	RecordedAt         time.Time      `json:"recorded_at"`
	SchemaVersion      int            `json:"schema_version"`
	DeduplicationKey   string         `json:"deduplication_key"`
	OrganizationIDHash string         `json:"organization_id_hash,omitempty"`
	Source             string         `json:"source"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

func NewStore(source Source) *Store {
	if source.Now == nil {
		source.Now = func() time.Time { return time.Now().UTC() }
	}
	if source.SupplierViews == nil {
		source.SupplierViews = func(string) []credit.View { return nil }
	}
	if source.BuyerViews == nil {
		source.BuyerViews = func(string) []credit.View { return nil }
	}
	if source.Payments == nil {
		source.Payments = func(string) []payments.Payment { return nil }
	}
	if source.Schedule == nil {
		source.Schedule = func(string) (schedules.Schedule, []schedules.Item, error) {
			return schedules.Schedule{}, nil, errors.New("schedule unavailable")
		}
	}
	if source.Disputes == nil {
		source.Disputes = func(string) []disputes.Dispute { return nil }
	}
	return &Store{source: source, events: []AnalyticsEvent{}}
}

func (s *Store) ReceivablesForSupplier(orgID string) Receivables {
	rows := s.rows(s.source.SupplierViews(orgID))
	result := Receivables{GeneratedAt: s.source.Now(), Currency: "NGN", Rows: rows}
	for _, row := range rows {
		result.Summary.addAt(row, result.GeneratedAt)
	}
	return result
}

func (s *Store) AgeingForSupplier(orgID string) Ageing {
	receivables := s.ReceivablesForSupplier(orgID)
	buckets := map[string]ledger.Money{"current": 0, "1_7": 0, "8_30": 0, "31_60": 0, "61_plus": 0, "paid": 0}
	for _, row := range receivables.Rows {
		buckets[row.AgeingBucket] += row.OutstandingKobo
	}
	return Ageing{GeneratedAt: receivables.GeneratedAt, Currency: receivables.Currency, Buckets: buckets, Rows: receivables.Rows}
}

func (s *Store) FeesForSupplier(orgID string) Fees {
	rows := s.rows(s.source.SupplierViews(orgID))
	result := Fees{GeneratedAt: s.source.Now(), Currency: "NGN", ByObligation: []ObligationFee{}}
	for _, row := range rows {
		fee := ObligationFee{ObligationID: row.ObligationID, BuyerName: row.BuyerName, BaseFeeKobo: row.BaseFeeKobo, CollectionFeesKobo: row.CollectionFeesKobo, TotalFeesKobo: row.BaseFeeKobo + row.CollectionFeesKobo}
		result.BaseFeesKobo += fee.BaseFeeKobo
		result.CollectionFeesKobo += fee.CollectionFeesKobo
		result.ByObligation = append(result.ByObligation, fee)
	}
	result.TotalFeesKobo = result.BaseFeesKobo + result.CollectionFeesKobo
	return result
}

func (s *Store) HistoryForBuyer(buyerID string) History {
	return s.historyFromViews(s.source.BuyerViews(buyerID))
}

func (s *Store) HistoryForSupplierBuyer(orgID, buyerID string) History {
	views := []credit.View{}
	for _, view := range s.source.SupplierViews(orgID) {
		if view.Request.BuyerUserID == buyerID {
			views = append(views, view)
		}
	}
	return s.historyFromViews(views)
}

func (s *Store) CustomerStatement(orgID, buyerID string) Statement {
	rows := []credit.View{}
	for _, view := range s.source.SupplierViews(orgID) {
		if view.Request.BuyerUserID == buyerID {
			rows = append(rows, view)
		}
	}
	result := Statement{GeneratedAt: s.source.Now(), Currency: "NGN", BuyerID: buyerID, Rows: s.rows(rows), Payments: []payments.Payment{}}
	for _, row := range result.Rows {
		result.Payments = append(result.Payments, s.source.Payments(row.ObligationID)...)
	}
	return result
}

func (s *Store) historyFromViews(views []credit.View) History {
	rows := s.rows(views)
	h := History{GeneratedAt: s.source.Now(), Obligations: rows, Shareable: false}
	var lateDays int64
	for _, row := range rows {
		if row.OutstandingKobo > 0 {
			h.ActiveObligations++
			h.CurrentActivePrincipalKobo += row.OutstandingKobo
		}
		if row.PaymentStatus == "PAID" {
			h.CompletedObligations++
			h.TotalCompletedPrincipalKobo += row.PrincipalKobo
			if row.PrincipalKobo > h.LargestCompletedAmountKobo {
				h.LargestCompletedAmountKobo = row.PrincipalKobo
			}
			if row.LatePayment {
				h.LatePaymentCount++
				lateDays += row.DaysLate
			} else {
				h.OnTimeCount++
			}
		}
		if row.Overdue && row.OutstandingKobo > 0 {
			h.UnresolvedOverdueObligations++
		}
		if row.MandateCancelledWhileOwing {
			h.MandateCancellationsOwing++
		}
		for _, d := range s.source.Disputes(row.ObligationID) {
			h.DisputeCount++
			_ = d
		}
		if h.VerifiedSince == nil || row.ActivatedAt.Before(*h.VerifiedSince) {
			t := row.ActivatedAt
			h.VerifiedSince = &t
		}
	}
	if h.CompletedObligations > 0 {
		h.OnTimePercentage = float64(h.OnTimeCount) / float64(h.CompletedObligations) * 100
		if h.LatePaymentCount > 0 {
			h.AverageDaysLate = float64(lateDays) / float64(h.LatePaymentCount)
		}
	}
	return h
}

func (s *Store) ExportReceivablesCSV(orgID string) ([]byte, error) {
	r := s.ReceivablesForSupplier(orgID)
	var b strings.Builder
	w := csv.NewWriter(&b)
	if err := w.Write([]string{"obligation_id", "buyer_name", "principal_kobo", "outstanding_kobo", "due_date", "ageing_bucket", "payment_status", "voluntary_paid_kobo", "collected_paid_kobo", "collection_fees_kobo"}); err != nil {
		return nil, err
	}
	for _, row := range r.Rows {
		if err := w.Write([]string{row.ObligationID, row.BuyerName, strconv.FormatInt(int64(row.PrincipalKobo), 10), strconv.FormatInt(int64(row.OutstandingKobo), 10), row.DueDate, row.AgeingBucket, row.PaymentStatus, strconv.FormatInt(int64(row.VoluntaryPaidKobo), 10), strconv.FormatInt(int64(row.CollectedPaidKobo), 10), strconv.FormatInt(int64(row.CollectionFeesKobo), 10)}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}

func (s *Store) Track(name, subjectID, purpose string, metadata map[string]string) (AnalyticsEvent, error) {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(purpose) == "" {
		return AnalyticsEvent{}, errors.New("analytics name and purpose are required")
	}
	if !validAnalyticsName(name) {
		return AnalyticsEvent{}, errors.New("analytics name must be a versioned dotted event name")
	}
	if err := validateAnalyticsMetadata(metadata); err != nil {
		return AnalyticsEvent{}, err
	}
	digest := sha256.Sum256([]byte(subjectID))
	now := s.source.Now()
	dedupeDigest := sha256.Sum256([]byte(name + "\x00" + subjectID + "\x00" + now.UTC().Format(time.RFC3339Nano)))
	e := AnalyticsEvent{ID: fmt.Sprintf("analytics-%d", now.UnixNano()), Name: name, SubjectID: hex.EncodeToString(digest[:]), Purpose: purpose, At: now, RecordedAt: now, SchemaVersion: 1, DeduplicationKey: "application:" + hex.EncodeToString(dedupeDigest[:]), Source: "application", Metadata: analyticsMetadata(metadata)}
	if s.pool != nil {
		e.ID = identifier.New()
		encoded, err := json.Marshal(metadata)
		if err != nil {
			return AnalyticsEvent{}, err
		}
		if err := s.pool.QueryRow(context.Background(), `INSERT INTO app.analytics_events(id,name,subject_id_hash,purpose,metadata,occurred_at,schema_version,deduplication_key,source) VALUES($1::uuid,$2,$3,$4,$5::jsonb,$6,$7,$8,$9) ON CONFLICT(deduplication_key) DO UPDATE SET deduplication_key=EXCLUDED.deduplication_key RETURNING occurred_at,recorded_at`, e.ID, e.Name, e.SubjectID, e.Purpose, encoded, e.At, e.SchemaVersion, e.DeduplicationKey, e.Source).Scan(&e.At, &e.RecordedAt); err != nil {
			return AnalyticsEvent{}, err
		}
		return e, nil
	}
	s.mu.Lock()
	s.events = append(s.events, e)
	s.mu.Unlock()
	return e, nil
}

func (s *Store) ListAnalytics() []AnalyticsEvent {
	if s.pool != nil {
		rows, err := s.pool.Query(context.Background(), `SELECT id::text,name,subject_id_hash,purpose,occurred_at,recorded_at,schema_version,deduplication_key,COALESCE(organization_id_hash,''),source,metadata FROM app.analytics_events ORDER BY occurred_at`)
		if err != nil {
			return []AnalyticsEvent{}
		}
		defer rows.Close()
		out := []AnalyticsEvent{}
		for rows.Next() {
			var event AnalyticsEvent
			var encoded []byte
			if err := rows.Scan(&event.ID, &event.Name, &event.SubjectID, &event.Purpose, &event.At, &event.RecordedAt, &event.SchemaVersion, &event.DeduplicationKey, &event.OrganizationIDHash, &event.Source, &encoded); err != nil {
				return []AnalyticsEvent{}
			}
			if err := json.Unmarshal(encoded, &event.Metadata); err != nil {
				return []AnalyticsEvent{}
			}
			out = append(out, event)
		}
		if rows.Err() != nil {
			return []AnalyticsEvent{}
		}
		return out
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]AnalyticsEvent(nil), s.events...)
}

var forbiddenAnalyticsMetadata = map[string]struct{}{
	"phone": {}, "email": {}, "bvn": {}, "nin": {}, "invoice_reference": {},
	"goods_description": {}, "bank_account": {}, "provider_token": {}, "reason": {},
	"notes": {}, "statement": {}, "body": {}, "name": {}, "address": {},
}

func validAnalyticsName(name string) bool {
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for i, r := range part {
			if !((r >= 'a' && r <= 'z') || (i > 0 && (r == '_' || (r >= '0' && r <= '9')))) {
				return false
			}
		}
	}
	return true
}

func validateAnalyticsMetadata(metadata map[string]string) error {
	if len(metadata) > 16 {
		return errors.New("analytics metadata exceeds the 16-field limit")
	}
	for key, value := range metadata {
		if _, forbidden := forbiddenAnalyticsMetadata[strings.ToLower(strings.TrimSpace(key))]; forbidden {
			return fmt.Errorf("analytics metadata field %q is not permitted", key)
		}
		if len(value) > 256 {
			return fmt.Errorf("analytics metadata field %q exceeds the value limit", key)
		}
	}
	return nil
}

func analyticsMetadata(metadata map[string]string) map[string]any {
	if metadata == nil {
		return nil
	}
	result := make(map[string]any, len(metadata))
	for key, value := range metadata {
		result[key] = value
	}
	return result
}

func (s *Store) rows(views []credit.View) []ObligationRow {
	rows := []ObligationRow{}
	now := s.source.Now()
	for _, v := range views {
		if v.Obligation == nil {
			continue
		}
		o := v.Obligation
		row := ObligationRow{ObligationID: o.ID, CreditRequestID: o.CreditRequestID, BuyerBusinessID: o.BuyerBusinessID, BuyerName: v.Request.BuyerLegalName, PrincipalKobo: o.PrincipalKobo, OutstandingKobo: o.OutstandingKobo, BaseFeeKobo: o.BaseFeeKobo, DueDate: v.Request.DueDate, PaymentStatus: o.PaymentStatus, ActivatedAt: o.ActivatedAt}
		var due time.Time
		if _, items, err := s.source.Schedule(o.ID); err == nil {
			for i := range items {
				if items[i].State != schedules.ItemPaid && items[i].State != schedules.ItemCancelled {
					d := items[i].DueAt
					if due.IsZero() || d.Before(due) {
						due = d
					}
				}
			}
		}
		if due.IsZero() {
			due, _ = time.Parse("2006-01-02", v.Request.DueDate)
			if !due.IsZero() {
				due = due.UTC()
			}
		}
		if !due.IsZero() {
			row.NextDueAt = &due
		}
		if due.IsZero() || o.OutstandingKobo == 0 {
			row.AgeingBucket = "paid"
		} else {
			grace := time.Duration(v.Request.GraceHours) * time.Hour
			overdueAt := due.Add(grace)
			row.Overdue = now.After(overdueAt)
			if !row.Overdue {
				row.AgeingBucket = "current"
			} else {
				days := int(now.Sub(overdueAt).Hours() / 24)
				switch {
				case days <= 7:
					row.AgeingBucket = "1_7"
				case days <= 30:
					row.AgeingBucket = "8_30"
				case days <= 60:
					row.AgeingBucket = "31_60"
				default:
					row.AgeingBucket = "61_plus"
				}
			}
		}
		for _, p := range s.source.Payments(o.ID) {
			if p.State != payments.StateRecognized {
				continue
			}
			if p.SourceType == payments.SourceCollected {
				row.CollectedPaidKobo += p.AmountKobo
			} else {
				row.VoluntaryPaidKobo += p.AmountKobo
			}
			row.CollectionFeesKobo += p.CollectionFeeKobo
			if !due.IsZero() && p.PaidAt.After(due.Add(time.Duration(v.Request.GraceHours)*time.Hour)) {
				row.LatePayment = true
				days := int64(p.PaidAt.Sub(due.Add(time.Duration(v.Request.GraceHours)*time.Hour)).Hours() / 24)
				if days > row.DaysLate {
					row.DaysLate = days
				}
			}
		}
		for _, d := range s.source.Disputes(o.ID) {
			if d.State == disputes.StateOpen || d.State == disputes.StateUnderReview || d.State == disputes.StatePartiallyResolved {
				row.OpenDisputeCount++
			}
		}
		row.MandateIssue = o.OutstandingKobo > 0 && (v.Mandate == nil || v.Mandate.Status != "ACTIVE")
		row.MandateCancelledWhileOwing = o.OutstandingKobo > 0 && v.Mandate != nil && v.Mandate.Status == "CANCELLED"
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ActivatedAt.Before(rows[j].ActivatedAt) })
	return rows
}

func (s *Summary) addAt(row ObligationRow, now time.Time) {
	s.ObligationCount++
	s.PrincipalKobo += row.PrincipalKobo
	s.OutstandingKobo += row.OutstandingKobo
	s.VoluntaryPaidKobo += row.VoluntaryPaidKobo
	s.CollectedPaidKobo += row.CollectedPaidKobo
	s.BaseFeesKobo += row.BaseFeeKobo
	s.CollectionFeesKobo += row.CollectionFeesKobo
	if row.Overdue {
		s.OverdueKobo += row.OutstandingKobo
	}
	if row.AgeingBucket == "current" {
		if row.NextDueAt != nil {
			d := row.NextDueAt.Truncate(24 * time.Hour)
			today := now.UTC().Truncate(24 * time.Hour)
			if d.Equal(today) {
				s.DueTodayKobo += row.OutstandingKobo
			}
			if d.After(today) && d.Before(today.Add(8*24*time.Hour)) {
				s.DueThisWeekKobo += row.OutstandingKobo
			}
		}
	}
	if row.OpenDisputeCount > 0 {
		s.OpenDisputeCount++
	}
	if row.MandateIssue {
		s.MandateIssueCount++
	}
}
