package reports

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"kredit/internal/credit"
	"kredit/internal/disputes"
	"kredit/internal/ledger"
	"kredit/internal/mandates"
	"kredit/internal/payments"
	"kredit/internal/schedules"
)

func TestReceivablesAndExportAreDerivedFromPayments(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	views := []credit.View{{Request: credit.CreditRequest{ID: "request-1", SupplierOrganizationID: "org-1", BuyerBusinessID: "buyer-1", BuyerLegalName: "Buyer", DueDate: "2026-08-10", GraceHours: 0}, Obligation: &credit.Obligation{ID: "obligation-1", CreditRequestID: "request-1", BuyerBusinessID: "buyer-1", PrincipalKobo: 1000, OutstandingKobo: 600, BaseFeeKobo: 5, PaymentStatus: "PARTIALLY_PAID", ActivatedAt: now.Add(-30 * 24 * time.Hour)}, Mandate: &mandates.Mandate{Status: mandates.Active}}}
	s := NewStore(Source{SupplierViews: func(string) []credit.View { return views }, Payments: func(string) ([]payments.Payment, error) {
		return []payments.Payment{{ObligationID: "obligation-1", AmountKobo: 400, SourceType: payments.SourceVoluntary, State: payments.StateRecognized}}, nil
	}, Schedule: func(string) (schedules.Schedule, []schedules.Item, error) {
		return schedules.Schedule{}, nil, errors.New("missing")
	}, Disputes: func(string) []disputes.Dispute { return nil }, Now: func() time.Time { return now }})
	report, readErr := s.ReceivablesForSupplier(context.Background(), "org-1")
	if readErr != nil {
		t.Fatal(readErr)
	}
	if report.Summary.OutstandingKobo != 600 || report.Summary.VoluntaryPaidKobo != 400 {
		t.Fatalf("unexpected summary: %#v", report.Summary)
	}
	data, err := s.ExportReceivablesCSV(context.Background(), "org-1")
	if err != nil || !strings.Contains(string(data), "obligation_id") || !strings.Contains(string(data), "obligation-1") {
		t.Fatalf("unexpected export: %v %s", err, data)
	}
}

func TestHistoryDoesNotEmitNonFiniteAverage(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	views := []credit.View{{Request: credit.CreditRequest{BuyerUserID: "buyer-1", BuyerLegalName: "Buyer", DueDate: "2026-08-10"}, Obligation: &credit.Obligation{ID: "obligation-2", PrincipalKobo: 500, OutstandingKobo: 0, PaymentStatus: "PAID", ActivatedAt: now.Add(-time.Hour)}}}
	s := NewStore(Source{BuyerViews: func(string) []credit.View { return views }, Payments: func(string) ([]payments.Payment, error) {
		return []payments.Payment{{AmountKobo: 500, State: payments.StateRecognized, SourceType: payments.SourceVoluntary}}, nil
	}, Now: func() time.Time { return now }})
	h, readErr := s.HistoryForBuyer(context.Background(), "buyer-1")
	if readErr != nil {
		t.Fatal(readErr)
	}
	if h.OnTimePercentage != 100 || h.AverageDaysLate != 0 {
		t.Fatalf("unexpected finite history metrics: %#v", h)
	}
}

func TestAnalyticsRejectsSensitiveMetadataAndUnversionedNames(t *testing.T) {
	s := NewStore(Source{})
	if _, err := s.Track("viewed", "subject", "product_improvement", nil); err == nil {
		t.Fatal("expected an unversioned event name to be rejected")
	}
	if _, err := s.Track("credit.viewed", "subject", "product_improvement", map[string]string{"phone": "+2348000000000"}); err == nil {
		t.Fatal("expected direct identity metadata to be rejected")
	}
	event, err := s.Track("credit.viewed", "subject", "product_improvement", map[string]string{"surface": "buyer"})
	if err != nil || event.SubjectID == "subject" || event.SchemaVersion != 1 || event.DeduplicationKey == "" {
		t.Fatalf("privacy-minimised event was not created: event=%+v err=%v", event, err)
	}
}

func TestForgivenBalanceDoesNotBecomeOnTimeRepaymentHistory(t *testing.T) {
	views := []credit.View{{Obligation: &credit.Obligation{ID: "forgiven", PrincipalKobo: 1000, PaymentStatus: "PAID"}}}
	s := NewStore(Source{BuyerViews: func(string) []credit.View { return views }})
	h, readErr := s.HistoryForBuyer(context.Background(), "buyer")
	if readErr != nil {
		t.Fatal(readErr)
	}
	if h.CompletedObligations != 0 || h.OnTimeCount != 0 {
		t.Fatal("forgiveness reported as proven repayment")
	}
}

func TestCSVNeutralizesSpreadsheetFormulas(t *testing.T) {
	for _, name := range []string{"=SUM(1,1)", " +cmd", "\t@SUM(1,1)", "-1+2"} {
		if !strings.HasPrefix(csvText(name), "'") {
			t.Fatal("formula exported: " + name)
		}
	}
}

func TestFeeReportSubtractsApprovedWaivers(t *testing.T) {
	store := NewStore(Source{SupplierViews: func(string) []credit.View {
		return []credit.View{{Obligation: &credit.Obligation{ID: "debt", BaseFeeKobo: 500}}}
	}, FeeWaivers: func(string) map[string]ledger.Money { return map[string]ledger.Money{"debt": 200} }})
	fees, err := store.FeesForSupplier(context.Background(), "supplier")
	if err != nil || fees.TotalFeesKobo != 300 || fees.WaivedFeesKobo != 200 || fees.ByObligation[0].TotalFeesKobo != 300 {
		t.Fatalf("fees=%+v %v", fees, err)
	}
}

func TestInstalmentTimelinessUsesEachAcceptedCollectionDate(t *testing.T) {
	first := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	second := first.AddDate(0, 1, 0)
	items := []schedules.Item{{Sequence: 1, PrincipalDueKobo: 500, CollectionAt: first}, {Sequence: 2, PrincipalDueKobo: 500, CollectionAt: second}}
	records := []payments.Payment{{AmountKobo: 500, State: payments.StateRecognized, PaidAt: first, RecognizedAt: first}, {AmountKobo: 500, State: payments.StateRecognized, PaidAt: second, RecognizedAt: second}}
	if late, days := paymentTimeliness(records, items, first); late || days != 0 {
		t.Fatalf("on-time instalment marked late: %v %d", late, days)
	}
	records[0].PaidAt = first.Add(48 * time.Hour)
	if late, days := paymentTimeliness(records, items, first); !late || days != 2 {
		t.Fatalf("late first instalment hidden: %v %d", late, days)
	}
}

func TestSummaryRejectsOverflowInsteadOfNegativeBalance(t *testing.T) {
	views := []credit.View{{Obligation: &credit.Obligation{ID: "a", PrincipalKobo: ledger.Money(1<<63 - 1), OutstandingKobo: ledger.Money(1<<63 - 1)}}, {Obligation: &credit.Obligation{ID: "b", PrincipalKobo: 1, OutstandingKobo: 1}}}
	store := NewStore(Source{SupplierViews: func(string) []credit.View { return views }})
	if _, err := store.ReceivablesForSupplier(context.Background(), "org"); err == nil {
		t.Fatal("overflow produced a successful receivables report")
	}
}

func TestPaymentReadFailureDoesNotBecomeAnEmptyHistory(t *testing.T) {
	sentinel := errors.New("payment database unavailable")
	s := NewStore(Source{SupplierViews: func(string) []credit.View {
		return []credit.View{{Obligation: &credit.Obligation{ID: "o", PrincipalKobo: 100, OutstandingKobo: 100}}}
	}, Payments: func(string) ([]payments.Payment, error) { return nil, sentinel }})
	if _, err := s.ReceivablesForSupplier(context.Background(), "org"); !errors.Is(err, sentinel) {
		t.Fatalf("payment failure hidden: %v", err)
	}
}
func TestReportTotalsUseOnlyOutstandingInstalmentsInEachPeriod(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	items := []schedules.Item{
		{Sequence: 4, PrincipalDueKobo: 100, DueAt: now.AddDate(0, 0, 2), CollectionAt: now.AddDate(0, 0, 3)},
		{Sequence: 1, PrincipalDueKobo: 100, AllocatedKobo: 40, DueAt: now.AddDate(0, 0, -10), CollectionAt: now.AddDate(0, 0, -9)},
		{Sequence: 2, PrincipalDueKobo: 100, DueAt: now, CollectionAt: now.AddDate(0, 0, 1)},
		{Sequence: 3, PrincipalDueKobo: 100, DueAt: now.AddDate(0, 1, 0), CollectionAt: now.AddDate(0, 1, 1)},
	}
	s := NewStore(Source{Now: func() time.Time { return now }, SupplierViews: func(string) []credit.View {
		return []credit.View{{Request: credit.CreditRequest{DueDate: "2026-08-24"}, Obligation: &credit.Obligation{ID: "o", PrincipalKobo: 400, OutstandingKobo: 360}}}
	}, Schedule: func(string) (schedules.Schedule, []schedules.Item, error) { return schedules.Schedule{}, items, nil }})
	report, err := s.ReceivablesForSupplier(context.Background(), "org")
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.OverdueKobo != 60 || report.Summary.DueTodayKobo != 100 || report.Summary.DueThisWeekKobo != 100 {
		t.Fatalf("incorrect instalment totals: %+v", report.Summary)
	}
	ageing, err := s.AgeingForSupplier(context.Background(), "org")
	if err != nil {
		t.Fatal(err)
	}
	if ageing.Buckets["8_30"] != 60 || ageing.Buckets["current"] != 300 {
		t.Fatalf("ageing=%+v", ageing)
	}
}

func TestCancelledReportDoesNotReadFinancialSource(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store := NewStore(Source{SupplierViews: func(string) []credit.View { t.Fatal("cancelled report reached source"); return nil }, BuyerViews: func(string) []credit.View { t.Fatal("cancelled report reached source"); return nil }})
	for _, read := range []func() error{
		func() error { _, err := store.ReceivablesForSupplier(ctx, "org"); return err },
		func() error { _, err := store.AgeingForSupplier(ctx, "org"); return err },
		func() error { _, err := store.FeesForSupplier(ctx, "org"); return err },
		func() error { _, err := store.HistoryForBuyer(ctx, "buyer"); return err },
		func() error { _, err := store.HistoryForSupplierBuyer(ctx, "org", "buyer"); return err },
		func() error { _, err := store.CustomerStatement(ctx, "org", "buyer"); return err },
		func() error { _, err := store.ExportReceivablesCSV(ctx, "org"); return err },
	} {
		if err := read(); !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v; want cancellation", err)
		}
	}
}
