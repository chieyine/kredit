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
	FeeWaivers    func(string) map[string]ledger.Money
	SupplierViews func(string) []credit.View
	BuyerViews    func(string) []credit.View
	Payments      func(string) ([]payments.Payment, error)
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
	dueTodayKobo, dueWeekKobo, overdueKobo ledger.Money
	ageingAmounts                          map[string]ledger.Money
	ObligationID                           string       `json:"obligation_id"`
	CreditRequestID                        string       `json:"credit_request_id"`
	BuyerBusinessID                        string       `json:"buyer_business_id"`
	BuyerName                              string       `json:"buyer_name"`
	PrincipalKobo                          ledger.Money `json:"principal_kobo"`
	OutstandingKobo                        ledger.Money `json:"outstanding_kobo"`
	BaseFeeKobo                            ledger.Money `json:"base_fee_kobo"`
	DueDate                                string       `json:"due_date"`
	NextDueAt                              *time.Time   `json:"next_due_at,omitempty"`
	PaymentStatus                          string       `json:"payment_status"`
	AgeingBucket                           string       `json:"ageing_bucket"`
	Overdue                                bool         `json:"overdue"`
	LatePayment                            bool         `json:"late_payment"`
	DaysLate                               int64        `json:"days_late"`
	VoluntaryPaidKobo                      ledger.Money `json:"voluntary_paid_kobo"`
	CollectedPaidKobo                      ledger.Money `json:"collected_paid_kobo"`
	CollectionFeesKobo                     ledger.Money `json:"collection_fees_kobo"`
	OpenDisputeCount                       int64        `json:"open_dispute_count"`
	MandateIssue                           bool         `json:"mandate_issue"`
	MandateCancelledWhileOwing             bool         `json:"mandate_cancelled_while_owing"`
	ActivatedAt                            time.Time    `json:"activated_at"`
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
	WaivedFeesKobo     ledger.Money    `json:"waived_fees_kobo"`
	GeneratedAt        time.Time       `json:"generated_at"`
	Currency           string          `json:"currency"`
	BaseFeesKobo       ledger.Money    `json:"base_fees_kobo"`
	CollectionFeesKobo ledger.Money    `json:"collection_fees_kobo"`
	TotalFeesKobo      ledger.Money    `json:"total_fees_kobo"`
	ByObligation       []ObligationFee `json:"by_obligation"`
}

type ObligationFee struct {
	WaivedFeesKobo     ledger.Money `json:"waived_fees_kobo"`
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
		source.Payments = func(string) ([]payments.Payment, error) { return nil, nil }
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

func (s *Store) ReceivablesForSupplier(ctx context.Context, orgID string) (Receivables, error) {
	if err := ctx.Err(); err != nil {
		return Receivables{}, err
	}
	if s.pool != nil {
		snapshot, err := s.financialSnapshot(ctx, orgID, "")
		if err != nil {
			return Receivables{}, err
		}
		return snapshot.ReceivablesForSupplier(ctx, orgID)
	}
	rows, err := s.rows(s.source.SupplierViews(orgID))
	if err != nil {
		return Receivables{}, err
	}
	result := Receivables{GeneratedAt: s.source.Now(), Currency: "NGN", Rows: rows}
	for _, row := range rows {
		if err := result.Summary.addAt(row, result.GeneratedAt); err != nil {
			return Receivables{}, err
		}
	}
	return result, nil
}

func (s *Store) AgeingForSupplier(ctx context.Context, orgID string) (Ageing, error) {
	if err := ctx.Err(); err != nil {
		return Ageing{}, err
	}
	if s.pool != nil {
		snapshot, err := s.financialSnapshot(ctx, orgID, "")
		if err != nil {
			return Ageing{}, err
		}
		return snapshot.AgeingForSupplier(ctx, orgID)
	}
	receivables, err := s.ReceivablesForSupplier(ctx, orgID)
	if err != nil {
		return Ageing{}, err
	}
	buckets := map[string]ledger.Money{"current": 0, "1_7": 0, "8_30": 0, "31_60": 0, "61_plus": 0, "paid": 0}
	for _, row := range receivables.Rows {
		for bucket, amount := range row.ageingAmounts {
			buckets[bucket], err = ledger.CheckedAdd(buckets[bucket], amount)
			if err != nil {
				return Ageing{}, err
			}
		}
	}
	return Ageing{GeneratedAt: receivables.GeneratedAt, Currency: receivables.Currency, Buckets: buckets, Rows: receivables.Rows}, nil
}

func (s *Store) FeesForSupplier(ctx context.Context, orgID string) (Fees, error) {
	if err := ctx.Err(); err != nil {
		return Fees{}, err
	}
	if s.pool != nil {
		snapshot, err := s.financialSnapshot(ctx, orgID, "")
		if err != nil {
			return Fees{}, err
		}
		return snapshot.FeesForSupplier(ctx, orgID)
	}
	rows, err := s.rows(s.source.SupplierViews(orgID))
	if err != nil {
		return Fees{}, err
	}
	waivers := map[string]ledger.Money{}
	if s.source.FeeWaivers != nil {
		waivers = s.source.FeeWaivers(orgID)
	}
	result := Fees{GeneratedAt: s.source.Now(), Currency: "NGN", ByObligation: []ObligationFee{}}
	for _, row := range rows {
		fee := ObligationFee{ObligationID: row.ObligationID, BuyerName: row.BuyerName, BaseFeeKobo: row.BaseFeeKobo, CollectionFeesKobo: row.CollectionFeesKobo, WaivedFeesKobo: waivers[row.ObligationID]}
		var err error
		fee.TotalFeesKobo, err = ledger.CheckedAdd(fee.BaseFeeKobo, fee.CollectionFeesKobo)
		if err != nil {
			return Fees{}, err
		}
		if fee.WaivedFeesKobo < 0 || fee.WaivedFeesKobo > fee.TotalFeesKobo {
			return Fees{}, errors.New("fee waivers exceed accrued fees")
		}
		fee.TotalFeesKobo, err = ledger.CheckedAdd(fee.TotalFeesKobo, -fee.WaivedFeesKobo)
		if err != nil {
			return Fees{}, err
		}
		for _, pair := range []struct {
			total  *ledger.Money
			amount ledger.Money
		}{{&result.BaseFeesKobo, fee.BaseFeeKobo}, {&result.CollectionFeesKobo, fee.CollectionFeesKobo}, {&result.WaivedFeesKobo, fee.WaivedFeesKobo}, {&result.TotalFeesKobo, fee.TotalFeesKobo}} {
			*pair.total, err = ledger.CheckedAdd(*pair.total, pair.amount)
			if err != nil {
				return Fees{}, err
			}
		}
		result.ByObligation = append(result.ByObligation, fee)
	}
	return result, nil
}

func (s *Store) HistoryForBuyer(ctx context.Context, buyerID string) (History, error) {
	if err := ctx.Err(); err != nil {
		return History{}, err
	}
	if s.pool != nil {
		snapshot, err := s.financialSnapshot(ctx, "", buyerID)
		if err != nil {
			return History{}, err
		}
		return snapshot.HistoryForBuyer(ctx, buyerID)
	}
	return s.historyFromViews(s.source.BuyerViews(buyerID))
}

func (s *Store) HistoryForSupplierBuyer(ctx context.Context, orgID, buyerID string) (History, error) {
	if err := ctx.Err(); err != nil {
		return History{}, err
	}
	if s.pool != nil {
		snapshot, err := s.financialSnapshot(ctx, orgID, buyerID)
		if err != nil {
			return History{}, err
		}
		return snapshot.HistoryForSupplierBuyer(ctx, orgID, buyerID)
	}
	views := []credit.View{}
	for _, view := range s.source.SupplierViews(orgID) {
		if view.Request.BuyerUserID == buyerID {
			views = append(views, view)
		}
	}
	return s.historyFromViews(views)
}

func (s *Store) CustomerStatement(ctx context.Context, orgID, buyerID string) (Statement, error) {
	if err := ctx.Err(); err != nil {
		return Statement{}, err
	}
	if s.pool != nil {
		snapshot, err := s.financialSnapshot(ctx, orgID, buyerID)
		if err != nil {
			return Statement{}, err
		}
		return snapshot.CustomerStatement(ctx, orgID, buyerID)
	}
	rows := []credit.View{}
	for _, view := range s.source.SupplierViews(orgID) {
		if view.Request.BuyerUserID == buyerID {
			rows = append(rows, view)
		}
	}
	reportRows, err := s.rows(rows)
	if err != nil {
		return Statement{}, err
	}
	result := Statement{GeneratedAt: s.source.Now(), Currency: "NGN", BuyerID: buyerID, Rows: reportRows, Payments: []payments.Payment{}}
	for _, row := range result.Rows {
		records, err := s.source.Payments(row.ObligationID)
		if err != nil {
			return Statement{}, err
		}
		result.Payments = append(result.Payments, records...)
	}
	return result, nil
}

func (s *Store) historyFromViews(views []credit.View) (History, error) {
	rows, err := s.rows(views)
	if err != nil {
		return History{}, err
	}
	h := History{GeneratedAt: s.source.Now(), Obligations: rows, Shareable: false}
	var lateDays int64
	for _, row := range rows {
		if row.OutstandingKobo > 0 {
			h.ActiveObligations++
			h.CurrentActivePrincipalKobo, err = ledger.CheckedAdd(h.CurrentActivePrincipalKobo, row.OutstandingKobo)
			if err != nil {
				return History{}, err
			}
		}
		if row.PaymentStatus == "PAID" && row.VoluntaryPaidKobo <= row.PrincipalKobo && row.CollectedPaidKobo == row.PrincipalKobo-row.VoluntaryPaidKobo {
			h.CompletedObligations++
			h.TotalCompletedPrincipalKobo, err = ledger.CheckedAdd(h.TotalCompletedPrincipalKobo, row.PrincipalKobo)
			if err != nil {
				return History{}, err
			}
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

	}
	if h.CompletedObligations > 0 {
		h.OnTimePercentage = float64(h.OnTimeCount) / float64(h.CompletedObligations) * 100
		if h.LatePaymentCount > 0 {
			h.AverageDaysLate = float64(lateDays) / float64(h.LatePaymentCount)
		}
	}
	return h, nil
}

func (s *Store) ExportReceivablesCSV(ctx context.Context, orgID string) ([]byte, error) {
	r, err := s.ReceivablesForSupplier(ctx, orgID)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	if err := w.Write([]string{"obligation_id", "buyer_name", "principal_kobo", "outstanding_kobo", "due_date", "ageing_bucket", "payment_status", "voluntary_paid_kobo", "collected_paid_kobo", "collection_fees_kobo"}); err != nil {
		return nil, err
	}
	for _, row := range r.Rows {
		if err := w.Write([]string{row.ObligationID, csvText(row.BuyerName), strconv.FormatInt(int64(row.PrincipalKobo), 10), strconv.FormatInt(int64(row.OutstandingKobo), 10), row.DueDate, row.AgeingBucket, row.PaymentStatus, strconv.FormatInt(int64(row.VoluntaryPaidKobo), 10), strconv.FormatInt(int64(row.CollectedPaidKobo), 10), strconv.FormatInt(int64(row.CollectionFeesKobo), 10)}); err != nil {
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
			isLetter := r >= 'a' && r <= 'z'
			isSuffix := i > 0 && (r == '_' || (r >= '0' && r <= '9'))
			if !isLetter && !isSuffix {
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

func (s *Store) rows(views []credit.View) ([]ObligationRow, error) {
	rows := []ObligationRow{}
	now := s.source.Now()
	for _, v := range views {
		if v.Obligation == nil {
			continue
		}
		o := v.Obligation
		row := ObligationRow{ObligationID: o.ID, CreditRequestID: o.CreditRequestID, BuyerBusinessID: o.BuyerBusinessID, BuyerName: v.Request.BuyerLegalName, PrincipalKobo: o.PrincipalKobo, OutstandingKobo: o.OutstandingKobo, BaseFeeKobo: o.BaseFeeKobo, DueDate: v.Request.DueDate, PaymentStatus: o.PaymentStatus, ActivatedAt: o.ActivatedAt}
		var due time.Time
		var scheduleItems []schedules.Item
		if _, items, err := s.source.Schedule(o.ID); err == nil {
			scheduleItems = items
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
		if err := row.summarizeInstalments(scheduleItems, due, v.Request.GraceHours, now); err != nil {
			return nil, err
		}
		allPayments, err := s.source.Payments(o.ID)
		if err != nil {
			return nil, err
		}
		for _, p := range allPayments {
			if p.State != payments.StateRecognized {
				continue
			}
			if p.SourceType == payments.SourceCollected {
				row.CollectedPaidKobo, err = ledger.CheckedAdd(row.CollectedPaidKobo, p.AmountKobo)
				if err != nil {
					return nil, err
				}
			} else {
				row.VoluntaryPaidKobo, err = ledger.CheckedAdd(row.VoluntaryPaidKobo, p.AmountKobo)
				if err != nil {
					return nil, err
				}
			}
			row.CollectionFeesKobo, err = ledger.CheckedAdd(row.CollectionFeesKobo, p.CollectionFeeKobo)
			if err != nil {
				return nil, err
			}

		}
		fallbackDeadline := due
		if !due.IsZero() {
			fallbackDeadline = due.Add(time.Duration(v.Request.GraceHours) * time.Hour)
		}
		row.LatePayment, row.DaysLate = paymentTimeliness(allPayments, scheduleItems, fallbackDeadline)
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
	return rows, nil
}

func (s *Summary) addAt(row ObligationRow, now time.Time) error {
	var err error
	s.ObligationCount++
	s.PrincipalKobo, err = ledger.CheckedAdd(s.PrincipalKobo, row.PrincipalKobo)
	if err != nil {
		return err
	}
	s.OutstandingKobo, err = ledger.CheckedAdd(s.OutstandingKobo, row.OutstandingKobo)
	if err != nil {
		return err
	}
	s.VoluntaryPaidKobo, err = ledger.CheckedAdd(s.VoluntaryPaidKobo, row.VoluntaryPaidKobo)
	if err != nil {
		return err
	}
	s.CollectedPaidKobo, err = ledger.CheckedAdd(s.CollectedPaidKobo, row.CollectedPaidKobo)
	if err != nil {
		return err
	}
	s.BaseFeesKobo, err = ledger.CheckedAdd(s.BaseFeesKobo, row.BaseFeeKobo)
	if err != nil {
		return err
	}
	s.CollectionFeesKobo, err = ledger.CheckedAdd(s.CollectionFeesKobo, row.CollectionFeesKobo)
	if err != nil {
		return err
	}
	s.OverdueKobo, err = ledger.CheckedAdd(s.OverdueKobo, row.overdueKobo)
	if err != nil {
		return err
	}
	s.DueTodayKobo, err = ledger.CheckedAdd(s.DueTodayKobo, row.dueTodayKobo)
	if err != nil {
		return err
	}
	s.DueThisWeekKobo, err = ledger.CheckedAdd(s.DueThisWeekKobo, row.dueWeekKobo)
	if err != nil {
		return err
	}
	if row.OpenDisputeCount > 0 {
		s.OpenDisputeCount++
	}
	if row.MandateIssue {
		s.MandateIssueCount++
	}
	return nil
}

// CSV quoting does not prevent spreadsheet applications interpreting formulas.
func csvText(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsAny(trimmed[:1], "=+-@") {
		return "'" + value
	}
	return value
}

// Replay oldest-first allocations in recognition order against each instalment's
// collection date. A later instalment must not be judged against the first due date.
func paymentTimeliness(records []payments.Payment, items []schedules.Item, fallback time.Time) (bool, int64) {
	records = append([]payments.Payment(nil), records...)
	items = append([]schedules.Item(nil), items...)
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].RecognizedAt.Equal(records[j].RecognizedAt) {
			return records[i].ID < records[j].ID
		}
		return records[i].RecognizedAt.Before(records[j].RecognizedAt)
	})
	sort.SliceStable(items, func(i, j int) bool { return items[i].Sequence < items[j].Sequence })
	index := 0
	var used ledger.Money
	late := false
	var days int64
	check := func(paid, deadline time.Time) {
		if !deadline.IsZero() && paid.After(deadline) {
			late = true
			if d := int64(paid.Sub(deadline).Hours() / 24); d > days {
				days = d
			}
		}
	}
	for _, payment := range records {
		if payment.State != payments.StateRecognized {
			continue
		}
		remaining := payment.AmountKobo
		if len(items) == 0 {
			check(payment.PaidAt, fallback)
			continue
		}
		for remaining > 0 && index < len(items) {
			item := items[index]
			if item.State == schedules.ItemCancelled || used >= item.PrincipalDueKobo {
				index++
				used = 0
				continue
			}
			take := item.PrincipalDueKobo - used
			if take > remaining {
				take = remaining
			}
			check(payment.PaidAt, item.CollectionAt)
			used += take
			remaining -= take
		}
	}
	return late, days
}

func (row *ObligationRow) summarizeInstalments(items []schedules.Item, fallback time.Time, graceHours int, now time.Time) error {
	row.ageingAmounts = map[string]ledger.Money{}
	row.AgeingBucket = "paid"
	if row.OutstandingKobo == 0 {
		return nil
	}
	if len(items) == 0 {
		items = []schedules.Item{{PrincipalDueKobo: row.OutstandingKobo, DueAt: fallback, CollectionAt: fallback.Add(time.Duration(graceHours) * time.Hour)}}
	}
	loc, err := time.LoadLocation("Africa/Lagos")
	if err != nil {
		return err
	}
	today := now.In(loc)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)
	remaining := row.OutstandingKobo
	for _, item := range items {
		if item.State == schedules.ItemPaid || item.State == schedules.ItemCancelled {
			continue
		}
		amount := min(remaining, item.PrincipalDueKobo-item.AllocatedKobo)
		if amount <= 0 {
			continue
		}
		remaining -= amount
		deadline := item.CollectionAt
		if deadline.IsZero() {
			deadline = item.DueAt.Add(time.Duration(graceHours) * time.Hour)
		}
		bucket := "current"
		if !deadline.IsZero() && !now.Before(deadline) {
			days := int(now.Sub(deadline) / (24 * time.Hour))
			bucket = "61_plus"
			if days <= 7 {
				bucket = "1_7"
			} else if days <= 30 {
				bucket = "8_30"
			} else if days <= 60 {
				bucket = "31_60"
			}
			row.overdueKobo, err = ledger.CheckedAdd(row.overdueKobo, amount)
			if err != nil {
				return err
			}
			row.Overdue = true
		}
		if row.AgeingBucket == "paid" || (row.AgeingBucket == "current" && bucket != "current") {
			row.AgeingBucket = bucket
		}
		row.ageingAmounts[bucket], err = ledger.CheckedAdd(row.ageingAmounts[bucket], amount)
		if err != nil {
			return err
		}
		due := item.DueAt.In(loc)
		day := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, loc)
		if day.Equal(today) {
			row.dueTodayKobo, err = ledger.CheckedAdd(row.dueTodayKobo, amount)
			if err != nil {
				return err
			}
		}
		if day.After(today) && day.Before(today.AddDate(0, 0, 8)) {
			row.dueWeekKobo, err = ledger.CheckedAdd(row.dueWeekKobo, amount)
			if err != nil {
				return err
			}
		}
	}
	if remaining > 0 {
		return errors.New("unpaid schedule does not cover the outstanding balance")
	}
	return nil
}
