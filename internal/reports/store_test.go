package reports

import (
	"errors"
	"strings"
	"testing"
	"time"

	"kredit/internal/credit"
	"kredit/internal/disputes"
	"kredit/internal/mandates"
	"kredit/internal/payments"
	"kredit/internal/schedules"
)

func TestReceivablesAndExportAreDerivedFromPayments(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	views := []credit.View{{Request: credit.CreditRequest{ID: "request-1", SupplierOrganizationID: "org-1", BuyerBusinessID: "buyer-1", BuyerLegalName: "Buyer", DueDate: "2026-08-10", GraceHours: 0}, Obligation: &credit.Obligation{ID: "obligation-1", CreditRequestID: "request-1", BuyerBusinessID: "buyer-1", PrincipalKobo: 1000, OutstandingKobo: 600, BaseFeeKobo: 5, PaymentStatus: "PARTIALLY_PAID", ActivatedAt: now.Add(-30 * 24 * time.Hour)}, Mandate: &mandates.Mandate{Status: mandates.Active}}}
	s := NewStore(Source{SupplierViews: func(string) []credit.View { return views }, Payments: func(string) []payments.Payment {
		return []payments.Payment{{ObligationID: "obligation-1", AmountKobo: 400, SourceType: payments.SourceVoluntary, State: payments.StateRecognized}}
	}, Schedule: func(string) (schedules.Schedule, []schedules.Item, error) {
		return schedules.Schedule{}, nil, errors.New("missing")
	}, Disputes: func(string) []disputes.Dispute { return nil }, Now: func() time.Time { return now }})
	report := s.ReceivablesForSupplier("org-1")
	if report.Summary.OutstandingKobo != 600 || report.Summary.VoluntaryPaidKobo != 400 {
		t.Fatalf("unexpected summary: %#v", report.Summary)
	}
	data, err := s.ExportReceivablesCSV("org-1")
	if err != nil || !strings.Contains(string(data), "obligation_id") || !strings.Contains(string(data), "obligation-1") {
		t.Fatalf("unexpected export: %v %s", err, data)
	}
}

func TestHistoryDoesNotEmitNonFiniteAverage(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	views := []credit.View{{Request: credit.CreditRequest{BuyerUserID: "buyer-1", BuyerLegalName: "Buyer", DueDate: "2026-08-10"}, Obligation: &credit.Obligation{ID: "obligation-2", PrincipalKobo: 500, OutstandingKobo: 0, PaymentStatus: "PAID", ActivatedAt: now.Add(-time.Hour)}}}
	s := NewStore(Source{BuyerViews: func(string) []credit.View { return views }, Now: func() time.Time { return now }})
	h := s.HistoryForBuyer("buyer-1")
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
