package reports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"

	"kredit/internal/db"
)

type Metric struct {
	Key          string  `json:"key"`
	Label        string  `json:"label"`
	Value        float64 `json:"value"`
	Unit         string  `json:"unit"`
	Definition   string  `json:"definition"`
	Source       string  `json:"source"`
	TargetStatus string  `json:"target_status"`
}

type Reconciliation struct {
	Event       string `json:"event"`
	SourceCount int64  `json:"source_count"`
	EventCount  int64  `json:"event_count"`
	Difference  int64  `json:"difference"`
	Tolerance   int64  `json:"tolerance"`
	Status      string `json:"status"`
}

type FeedbackSummary struct {
	Total        int64   `json:"total"`
	Yes          int64   `json:"yes"`
	Partly       int64   `json:"partly"`
	No           int64   `json:"no"`
	Seller       int64   `json:"seller"`
	Buyer        int64   `json:"buyer"`
	ClearPercent float64 `json:"clear_percent"`
}

type PilotScorecard struct {
	GeneratedAt      time.Time        `json:"generated_at"`
	From             time.Time        `json:"from"`
	To               time.Time        `json:"to"`
	OrganizationID   string           `json:"organization_id,omitempty"`
	SourceOfTruth    string           `json:"source_of_truth"`
	RefreshMode      string           `json:"refresh_mode"`
	LatestEventAt    *time.Time       `json:"latest_event_at,omitempty"`
	FreshnessStatus  string           `json:"freshness_status"`
	KPIs             []Metric         `json:"kpis"`
	Drivers          []Metric         `json:"drivers"`
	Guardrails       []Metric         `json:"guardrails"`
	Funnel           map[string]int64 `json:"funnel"`
	Feedback         FeedbackSummary  `json:"feedback"`
	Reconciliation   []Reconciliation `json:"reconciliation"`
	ReconciliationOK bool             `json:"reconciliation_ok"`
}

func (s *Store) PilotScorecard(ctx context.Context, from, to time.Time, organizationID string) (PilotScorecard, error) {
	if s.pool == nil {
		return PilotScorecard{}, errors.New("pilot scorecard requires the authoritative database")
	}
	if from.IsZero() || to.IsZero() || !from.Before(to) || to.Sub(from) > 366*24*time.Hour {
		return PilotScorecard{}, errors.New("scorecard window must be positive and no longer than 366 days")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return PilotScorecard{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = db.SetTenantContext(ctx, tx); err != nil {
		return PilotScorecard{}, err
	}
	result := PilotScorecard{GeneratedAt: s.source.Now(), From: from.UTC(), To: to.UTC(), OrganizationID: organizationID, SourceOfTruth: "authoritative domain tables; analytics events are reconciliation evidence only", RefreshMode: "live query", FreshnessStatus: "live", Funnel: map[string]int64{}, ReconciliationOK: true}
	org := organizationID
	var latest *time.Time
	orgHash := ""
	if org != "" {
		digest := sha256.Sum256([]byte(org))
		orgHash = hex.EncodeToString(digest[:])
	}
	if err := tx.QueryRow(ctx, `SELECT max(recorded_at) FROM app.analytics_events WHERE occurred_at >= $1 AND occurred_at < $2 AND ($3='' OR organization_id_hash=$3)`, from, to, orgHash).Scan(&latest); err != nil {
		return PilotScorecard{}, err
	}
	result.LatestEventAt = latest
	if err := tx.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE metadata->>'answer'='yes'),count(*) FILTER(WHERE metadata->>'answer'='partly'),count(*) FILTER(WHERE metadata->>'answer'='no'),count(*) FILTER(WHERE metadata->>'area'='seller'),count(*) FILTER(WHERE metadata->>'area'='buyer') FROM app.analytics_events WHERE name='feedback.clarity_submitted' AND occurred_at >= $1 AND occurred_at < $2 AND ($3='' OR organization_id_hash=$3)`, from, to, orgHash).Scan(&result.Feedback.Total, &result.Feedback.Yes, &result.Feedback.Partly, &result.Feedback.No, &result.Feedback.Seller, &result.Feedback.Buyer); err != nil {
		return PilotScorecard{}, fmt.Errorf("scorecard feedback: %w", err)
	}
	if result.Feedback.Total > 0 {
		result.Feedback.ClearPercent = 100 * float64(result.Feedback.Yes) / float64(result.Feedback.Total)
	}

	queries := []struct {
		set                                  *([]Metric)
		key, label, unit, definition, source string
	}{
		{&result.KPIs, "gross_trade_credit_volume", "Gross trade credit activated", "kobo", "Sum of principal for obligations activated in the window.", "app.obligations"},
		{&result.KPIs, "active_suppliers", "Active suppliers", "organizations", "Distinct supplier organisations with an obligation activated in the window.", "app.obligations"},
		{&result.KPIs, "time_to_first_accepted_sale", "Time to first accepted credit sale", "hours", "Average hours from a supplier's first onboarding record to its first accepted agreement.", "app.supplier_onboarding_profiles + app.credit_requests + app.agreement_acceptances"},
		{&result.Drivers, "sent_to_acceptance", "Sent-to-acceptance conversion", "percent", "Accepted agreements divided by credit.sent events in the window.", "app.analytics_events (transition evidence)"},
		{&result.Drivers, "invitation_to_verification", "Invitation-to-verification conversion", "percent", "Invitations accepted by a user who owns a verified business, divided by invitations created in the window.", "app.buyer_invitations + app.businesses"},
		{&result.Drivers, "acceptance_to_release", "Acceptance to goods release", "hours", "Average elapsed hours from immutable acceptance to goods release for the same request.", "app.agreement_acceptances + app.goods_releases"},
		{&result.Drivers, "release_to_receipt", "Release to receipt confirmation", "hours", "Average elapsed hours from goods release to buyer receipt response.", "app.goods_releases + app.receipt_confirmations"},
		{&result.Drivers, "days_to_payment", "Days to payment", "days", "Average days from activation to the final recognised payment, for principal fully repaid within the window.", "app.obligations + app.payments"},
		{&result.Drivers, "repeat_sale_rate", "Repeat-sale rate", "percent", "Supplier-buyer relationships with at least two activated obligations divided by relationships with any activated obligation.", "app.obligations"},
		{&result.Drivers, "trade_line_utilization", "Trade-line utilisation", "percent", "Current exposure plus pending reservations divided by approved limit on active trade lines.", "app.trade_lines"},
		{&result.Drivers, "supplier_retention", "Retained active suppliers", "percent", "Suppliers with activated obligations in both the selected window and the immediately preceding equal window, divided by active suppliers in the preceding window.", "app.obligations"},
		{&result.Guardrails, "on_time_payment_rate", "On-time payment rate", "percent", "Fully repaid obligations completed in the window whose allocated payments met every agreed instalment deadline.", "app.obligations + app.payments + app.payment_allocations + app.schedule_items"},
		{&result.Guardrails, "failed_collection_recovery", "Failed-collection recovery", "percent", "Obligations with a failed attempt in the window and a later positive collection completed before the window ends, divided by obligations with a failed attempt.", "app.collection_attempts + app.obligations"},
		{&result.Guardrails, "dispute_rate", "Dispute rate", "percent", "Obligations with a dispute opened in the window divided by obligations activated in the window.", "app.disputes + app.obligations"},
		{&result.Guardrails, "receipt_issue_rate", "Issue-at-receipt rate", "percent", "Receipt responses marked issue_raised divided by all receipt responses.", "app.receipt_confirmations + app.credit_requests"},
		{&result.Guardrails, "provider_reliability", "Collection provider reliability", "percent", "Successful or partial final collection attempts divided by all final attempts.", "app.collection_attempts + app.obligations"},
		{&result.Guardrails, "recognized_loss_rate", "Recognised loss rate", "percent", "Write-off amount recorded in the window divided by principal of the affected obligations.", "app.operation_actions + app.obligations"},
		{&result.Guardrails, "support_intervention_rate", "Support intervention rate", "cases_per_100_active_suppliers", "Support cases opened per 100 suppliers that activated an obligation in the selected window.", "app.support_cases + app.obligations"},
		{&result.Guardrails, "accessibility_defects", "Open accessibility defects", "cases", "Accessibility defect cases opened by the end of the window and still open or in progress.", "app.support_cases"},
		// The buyer is asked for full verification and a variable-amount debit
		// authorisation in exchange for goods they previously received on a
		// handshake. This is where the funnel breaks if the buyer proposition is
		// wrong, so it is measured on its own rather than folded into
		// sent-to-acceptance conversion.
		{&result.Drivers, "mandate_authorization_dropoff", "Mandate authorisation drop-off", "percent", "Mandates created in the window that never reached an active authorisation, divided by mandates created.", "app.mandates + app.credit_requests"},
		// Kredit earns its collection uplift only on money it collects, so the
		// pricing quietly rewards buyers drifting to the collection date. This
		// metric exists so that drift is visible rather than inferred: a falling
		// voluntary share is the signal that reminder timing or payment friction
		// has moved in Kredit's favour and against the buyer's.
		{&result.Guardrails, "voluntary_payment_share", "Voluntary payment share", "percent", "Recognised payment value that arrived without a Kredit collection, divided by all recognised payment value excluding adjustments.", "app.payments"},
		// Deemed acceptance is the only path where silence creates a collectable
		// obligation. A rising share means buyers are not answering, which is a
		// wrongful-debit risk signal long before it becomes a dispute.
		{&result.Guardrails, "deemed_acceptance_share", "Activations from buyer silence", "percent", "Confirmed receipts recorded by deemed acceptance, divided by all confirmed receipts in the window.", "app.receipt_confirmations + app.credit_requests"},
		// Kredit earns at most one hundred basis points on activated principal.
		// Every workflow counted here consumes human time against that margin, so
		// the ratio is the cost-to-serve signal. Minutes per touch is not
		// invented here; it is a sampled input the pilot supplies, and this
		// metric is the multiplier it applies to.
		{&result.Guardrails, "manual_touches_per_obligation", "Manual touches per activated obligation", "touches", "Support cases, operations actions, correction requests, and disputes recorded in the window, divided by obligations activated in the window.", "app.support_cases + app.operation_actions + app.correction_requests + app.disputes + app.obligations"},
	}
	for _, q := range queries {
		var value *float64
		if err := tx.QueryRow(ctx, `SELECT app.pilot_metric($1,$2,$3,$4)`, from, to, org, q.key).Scan(&value); err != nil {
			return PilotScorecard{}, fmt.Errorf("scorecard metric %s: %w", q.key, err)
		}
		metric := Metric{Key: q.key, Label: q.label, Unit: q.unit, Definition: q.definition, Source: q.source, TargetStatus: "no_data"}
		if value != nil {
			metric.Value = *value
			metric.TargetStatus = "baseline_required"
		}
		*q.set = append(*q.set, metric)
	}

	rows, err := tx.Query(ctx, `SELECT name,count(*) FROM app.analytics_events WHERE occurred_at >= $1 AND occurred_at < $2 AND ($3='' OR organization_id_hash=$3) GROUP BY name`, from, to, orgHash)
	if err != nil {
		return PilotScorecard{}, err
	}
	for rows.Next() {
		var name string
		var count int64
		if err := rows.Scan(&name, &count); err != nil {
			rows.Close()
			return PilotScorecard{}, err
		}
		result.Funnel[name] = count
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return PilotScorecard{}, err
	}
	rows.Close()

	reconciliationSQL := `SELECT * FROM app.pilot_reconciliation($1,$2,$3)`
	rows, err = tx.Query(ctx, reconciliationSQL, from, to, org)
	if err != nil {
		return PilotScorecard{}, err
	}
	for rows.Next() {
		var item Reconciliation
		if err := rows.Scan(&item.Event, &item.SourceCount, &item.EventCount, &item.Difference); err != nil {
			rows.Close()
			return PilotScorecard{}, err
		}
		item.Tolerance = 0
		item.Status = "reconciled"
		if item.Difference != 0 {
			item.Status = "mismatch"
			result.ReconciliationOK = false
		}
		result.Reconciliation = append(result.Reconciliation, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return PilotScorecard{}, err
	}
	rows.Close()
	sort.Slice(result.KPIs, func(i, j int) bool { return result.KPIs[i].Key < result.KPIs[j].Key })
	if err := tx.Commit(ctx); err != nil {
		return PilotScorecard{}, err
	}
	return result, nil
}
