package reports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"time"
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
	result := PilotScorecard{GeneratedAt: s.source.Now(), From: from.UTC(), To: to.UTC(), OrganizationID: organizationID, SourceOfTruth: "authoritative domain tables; analytics events are reconciliation evidence only", RefreshMode: "live query", FreshnessStatus: "live", Funnel: map[string]int64{}, ReconciliationOK: true}
	org := organizationID
	var latest *time.Time
	orgHash := ""
	if org != "" {
		digest := sha256.Sum256([]byte(org))
		orgHash = hex.EncodeToString(digest[:])
	}
	if err := s.pool.QueryRow(ctx, `SELECT max(recorded_at) FROM app.analytics_events WHERE occurred_at >= $1 AND occurred_at < $2 AND ($3='' OR organization_id_hash=$3)`, from, to, orgHash).Scan(&latest); err != nil {
		return PilotScorecard{}, err
	}
	result.LatestEventAt = latest
	if err := s.pool.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE metadata->>'answer'='yes'),count(*) FILTER(WHERE metadata->>'answer'='partly'),count(*) FILTER(WHERE metadata->>'answer'='no'),count(*) FILTER(WHERE metadata->>'area'='seller'),count(*) FILTER(WHERE metadata->>'area'='buyer') FROM app.analytics_events WHERE name='feedback.clarity_submitted' AND occurred_at >= $1 AND occurred_at < $2 AND ($3='' OR organization_id_hash=$3)`, from, to, orgHash).Scan(&result.Feedback.Total, &result.Feedback.Yes, &result.Feedback.Partly, &result.Feedback.No, &result.Feedback.Seller, &result.Feedback.Buyer); err != nil {
		return PilotScorecard{}, fmt.Errorf("scorecard feedback: %w", err)
	}
	if result.Feedback.Total > 0 {
		result.Feedback.ClearPercent = 100 * float64(result.Feedback.Yes) / float64(result.Feedback.Total)
	}

	queries := []struct {
		set                                       *([]Metric)
		key, label, unit, definition, source, sql string
	}{
		{&result.KPIs, "gross_trade_credit_volume", "Gross trade credit activated", "kobo", "Sum of principal for obligations activated in the window.", "app.obligations", `SELECT COALESCE(sum(principal_kobo),0)::float8 FROM app.obligations WHERE activated_at >= $1 AND activated_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)`},
		{&result.KPIs, "active_suppliers", "Active suppliers", "organizations", "Distinct supplier organisations with an obligation activated in the window.", "app.obligations", `SELECT count(DISTINCT supplier_organization_id)::float8 FROM app.obligations WHERE activated_at >= $1 AND activated_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)`},
		{&result.KPIs, "time_to_first_accepted_sale", "Time to first accepted credit sale", "hours", "Average hours from a supplier's first onboarding record to its first accepted agreement.", "app.supplier_onboarding_profiles + app.credit_requests + app.agreement_acceptances", `WITH firsts AS (SELECT c.supplier_organization_id,min(a.accepted_at) accepted_at FROM app.agreement_acceptances a JOIN app.credit_requests c ON c.id=a.credit_request_id WHERE a.accepted_at >= $1 AND a.accepted_at < $2 AND ($3='' OR c.supplier_organization_id=NULLIF($3,'')::uuid) GROUP BY c.supplier_organization_id) SELECT COALESCE(avg(extract(epoch FROM (f.accepted_at-p.created_at))/3600),0)::float8 FROM firsts f JOIN app.supplier_onboarding_profiles p ON p.organization_id=f.supplier_organization_id`},
		{&result.Drivers, "sent_to_acceptance", "Sent-to-acceptance conversion", "percent", "Accepted agreements divided by credit.sent events in the window.", "app.analytics_events (transition evidence)", `SELECT CASE WHEN count(*) FILTER(WHERE name='credit.sent')=0 THEN 0 ELSE 100.0*count(*) FILTER(WHERE name='credit.accepted')/count(*) FILTER(WHERE name='credit.sent') END::float8 FROM app.analytics_events WHERE occurred_at >= $1 AND occurred_at < $2 AND ($3='' OR organization_id_hash=encode(digest($3,'sha256'),'hex'))`},
		{&result.Drivers, "invitation_to_verification", "Invitation-to-verification conversion", "percent", "Invitations accepted by a user who owns a verified business, divided by invitations created in the window.", "app.buyer_invitations + app.businesses", `SELECT CASE WHEN count(*)=0 THEN 0 ELSE 100.0*count(*) FILTER(WHERE EXISTS(SELECT 1 FROM app.businesses b WHERE b.owner_user_id=i.accepted_by_user_id AND b.status='verified'))/count(*) END::float8 FROM app.buyer_invitations i WHERE i.created_at >= $1 AND i.created_at < $2 AND ($3='' OR i.organization_id=NULLIF($3,'')::uuid)`},
		{&result.Drivers, "acceptance_to_release", "Acceptance to goods release", "hours", "Average elapsed hours from immutable acceptance to goods release for the same request.", "app.agreement_acceptances + app.goods_releases", `SELECT COALESCE(avg(extract(epoch FROM (g.released_at-a.accepted_at))/3600),0)::float8 FROM app.agreement_acceptances a JOIN app.goods_releases g USING(credit_request_id) JOIN app.credit_requests c ON c.id=a.credit_request_id WHERE g.released_at >= $1 AND g.released_at < $2 AND ($3='' OR c.supplier_organization_id=NULLIF($3,'')::uuid)`},
		{&result.Drivers, "release_to_receipt", "Release to receipt confirmation", "hours", "Average elapsed hours from goods release to buyer receipt response.", "app.goods_releases + app.receipt_confirmations", `SELECT COALESCE(avg(extract(epoch FROM (r.received_at-g.released_at))/3600),0)::float8 FROM app.goods_releases g JOIN app.receipt_confirmations r USING(credit_request_id) JOIN app.credit_requests c ON c.id=g.credit_request_id WHERE r.received_at >= $1 AND r.received_at < $2 AND ($3='' OR c.supplier_organization_id=NULLIF($3,'')::uuid)`},
		{&result.Drivers, "days_to_payment", "Days to payment", "days", "Average days from obligation activation to the final recognised payment timestamp for paid obligations.", "app.obligations + app.payments", `WITH paid AS (SELECT o.id,o.activated_at,max(p.paid_at) paid_at FROM app.obligations o JOIN app.payments p ON p.obligation_id=o.id AND p.state='recognized' WHERE o.payment_status='PAID' AND p.paid_at >= $1 AND p.paid_at < $2 AND ($3='' OR o.supplier_organization_id=NULLIF($3,'')::uuid) GROUP BY o.id,o.activated_at) SELECT COALESCE(avg(extract(epoch FROM (paid_at-activated_at))/86400),0)::float8 FROM paid`},
		{&result.Drivers, "repeat_sale_rate", "Repeat-sale rate", "percent", "Supplier-buyer relationships with at least two activated obligations divided by relationships with any activated obligation.", "app.obligations", `WITH pairs AS (SELECT supplier_organization_id,buyer_business_id,count(*) n FROM app.obligations WHERE activated_at >= $1 AND activated_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid) GROUP BY supplier_organization_id,buyer_business_id) SELECT CASE WHEN count(*)=0 THEN 0 ELSE 100.0*count(*) FILTER(WHERE n>1)/count(*) END::float8 FROM pairs`},
		{&result.Drivers, "trade_line_utilization", "Trade-line utilisation", "percent", "Current exposure plus pending reservations divided by approved limit on active trade lines.", "app.trade_lines", `SELECT CASE WHEN COALESCE(sum(approved_limit_kobo),0)=0 THEN 0 ELSE 100.0*sum(current_exposure_kobo+reserved_pending_kobo)/sum(approved_limit_kobo) END::float8 FROM app.trade_lines WHERE state='ACTIVE' AND $1::timestamptz < $2::timestamptz AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)`},
		{&result.Drivers, "supplier_retention", "Retained active suppliers", "percent", "Suppliers with activated obligations in both the selected window and the immediately preceding equal window, divided by active suppliers in the preceding window.", "app.obligations", `WITH previous AS (SELECT DISTINCT supplier_organization_id FROM app.obligations WHERE activated_at >= $1::timestamptz-($2::timestamptz-$1::timestamptz) AND activated_at < $1 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)), current_window AS (SELECT DISTINCT supplier_organization_id FROM app.obligations WHERE activated_at >= $1 AND activated_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)) SELECT CASE WHEN (SELECT count(*) FROM previous)=0 THEN 0 ELSE 100.0*(SELECT count(*) FROM previous p JOIN current_window c USING(supplier_organization_id))/(SELECT count(*) FROM previous) END::float8`},
		{&result.Guardrails, "on_time_payment_rate", "On-time payment rate", "percent", "Paid obligations whose last recognised payment was no later than contractual due date plus grace, divided by paid obligations.", "app.credit_requests + app.obligations + app.payments", `WITH paid AS (SELECT o.id,c.due_date,c.grace_hours,max(p.paid_at) paid_at FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id JOIN app.payments p ON p.obligation_id=o.id AND p.state='recognized' WHERE o.payment_status='PAID' AND p.paid_at >= $1 AND p.paid_at < $2 AND ($3='' OR o.supplier_organization_id=NULLIF($3,'')::uuid) GROUP BY o.id,c.due_date,c.grace_hours) SELECT CASE WHEN count(*)=0 THEN 0 ELSE 100.0*count(*) FILTER(WHERE paid_at <= due_date::timestamptz + make_interval(hours=>grace_hours))/count(*) END::float8 FROM paid`},
		{&result.Guardrails, "failed_collection_recovery", "Failed-collection recovery", "percent", "Obligations with a later successful collection after a failed attempt, divided by obligations with a failed attempt.", "app.collection_attempts + app.obligations", `WITH failed AS (SELECT DISTINCT ca.obligation_id FROM app.collection_attempts ca JOIN app.obligations o ON o.id=ca.obligation_id WHERE ca.state='FAILED' AND ca.requested_at >= $1 AND ca.requested_at < $2 AND ($3='' OR o.supplier_organization_id=NULLIF($3,'')::uuid)), recovered AS (SELECT DISTINCT f.obligation_id FROM failed f WHERE EXISTS(SELECT 1 FROM app.collection_attempts ca WHERE ca.obligation_id=f.obligation_id AND ca.state IN('SUCCEEDED','PARTIAL') AND ca.attempt_number>1)) SELECT CASE WHEN (SELECT count(*) FROM failed)=0 THEN 0 ELSE 100.0*(SELECT count(*) FROM recovered)/(SELECT count(*) FROM failed) END::float8`},
		{&result.Guardrails, "dispute_rate", "Dispute rate", "percent", "Obligations with a dispute opened in the window divided by obligations activated in the window.", "app.disputes + app.obligations", `SELECT CASE WHEN count(DISTINCT o.id)=0 THEN 0 ELSE 100.0*count(DISTINCT d.obligation_id)/count(DISTINCT o.id) END::float8 FROM app.obligations o LEFT JOIN app.disputes d ON d.obligation_id=o.id AND d.opened_at >= $1 AND d.opened_at < $2 WHERE o.activated_at >= $1 AND o.activated_at < $2 AND ($3='' OR o.supplier_organization_id=NULLIF($3,'')::uuid)`},
		{&result.Guardrails, "receipt_issue_rate", "Issue-at-receipt rate", "percent", "Receipt responses marked issue_raised divided by all receipt responses.", "app.receipt_confirmations + app.credit_requests", `SELECT CASE WHEN count(*)=0 THEN 0 ELSE 100.0*count(*) FILTER(WHERE r.state='issue_raised')/count(*) END::float8 FROM app.receipt_confirmations r JOIN app.credit_requests c ON c.id=r.credit_request_id WHERE r.received_at >= $1 AND r.received_at < $2 AND ($3='' OR c.supplier_organization_id=NULLIF($3,'')::uuid)`},
		{&result.Guardrails, "provider_reliability", "Collection provider reliability", "percent", "Successful or partial final collection attempts divided by all final attempts.", "app.collection_attempts + app.obligations", `SELECT CASE WHEN count(*)=0 THEN 100 ELSE 100.0*count(*) FILTER(WHERE ca.state IN('SUCCEEDED','PARTIAL'))/count(*) END::float8 FROM app.collection_attempts ca JOIN app.obligations o ON o.id=ca.obligation_id WHERE ca.final_at >= $1 AND ca.final_at < $2 AND ca.state IN('SUCCEEDED','PARTIAL','FAILED','CANCELLED') AND ($3='' OR o.supplier_organization_id=NULLIF($3,'')::uuid)`},
		{&result.Guardrails, "recognized_loss_rate", "Recognised loss rate", "percent", "Write-off amount recorded in the window divided by principal of the affected obligations.", "app.operation_actions + app.obligations", `WITH losses AS (SELECT resource_id,COALESCE(sum((metadata->>'amount_kobo')::bigint),0) amount FROM app.operation_actions WHERE action='write_off' AND created_at >= $1 AND created_at < $2 AND ($3='' OR organization_id=NULLIF($3,'')::uuid) GROUP BY resource_id), exposure AS (SELECT COALESCE(sum(o.principal_kobo),0) principal FROM app.obligations o WHERE EXISTS(SELECT 1 FROM losses l WHERE l.resource_id=o.id)) SELECT CASE WHEN exposure.principal=0 THEN 0 ELSE 100.0*(SELECT COALESCE(sum(amount),0) FROM losses)/exposure.principal END::float8 FROM exposure`},
		{&result.Guardrails, "support_intervention_rate", "Support intervention rate", "cases_per_100_active_suppliers", "Support cases opened per 100 suppliers that activated an obligation in the selected window.", "app.support_cases + app.obligations", `WITH active AS (SELECT count(DISTINCT supplier_organization_id) n FROM app.obligations WHERE activated_at >= $1 AND activated_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)), cases AS (SELECT count(*) n FROM app.support_cases WHERE created_at >= $1 AND created_at < $2 AND ($3='' OR organization_id=NULLIF($3,'')::uuid)) SELECT CASE WHEN active.n=0 THEN 0 ELSE 100.0*cases.n/active.n END::float8 FROM active,cases`},
		{&result.Guardrails, "accessibility_defects", "Open accessibility defects", "cases", "Accessibility defect cases opened by the end of the window and still open or in progress.", "app.support_cases", `SELECT count(*)::float8 FROM app.support_cases WHERE subject_type='accessibility_defect' AND created_at < $2 AND state IN('OPEN','IN_PROGRESS') AND $1::timestamptz < $2::timestamptz AND ($3='' OR organization_id=NULLIF($3,'')::uuid)`},
	}
	for _, q := range queries {
		var value float64
		if err := s.pool.QueryRow(ctx, q.sql, from, to, org).Scan(&value); err != nil {
			return PilotScorecard{}, fmt.Errorf("scorecard metric %s: %w", q.key, err)
		}
		*q.set = append(*q.set, Metric{Key: q.key, Label: q.label, Value: value, Unit: q.unit, Definition: q.definition, Source: q.source, TargetStatus: "baseline_required"})
	}

	rows, err := s.pool.Query(ctx, `SELECT name,count(*) FROM app.analytics_events WHERE occurred_at >= $1 AND occurred_at < $2 AND ($3='' OR organization_id_hash=$3) GROUP BY name`, from, to, orgHash)
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

	reconciliationSQL := `WITH expected(event,n) AS (
		SELECT 'customer.invited',count(*) FROM app.buyer_invitations WHERE created_at >= $1 AND created_at < $2 AND ($3='' OR organization_id=NULLIF($3,'')::uuid) UNION ALL
		SELECT 'credit.drafted',count(*) FROM app.credit_requests WHERE created_at >= $1 AND created_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid) UNION ALL
		SELECT 'credit.accepted',count(*) FROM app.agreement_acceptances a JOIN app.credit_requests c ON c.id=a.credit_request_id WHERE a.accepted_at >= $1 AND a.accepted_at < $2 AND ($3='' OR c.supplier_organization_id=NULLIF($3,'')::uuid) UNION ALL
		SELECT 'goods.released',count(*) FROM app.goods_releases g JOIN app.credit_requests c ON c.id=g.credit_request_id WHERE g.released_at >= $1 AND g.released_at < $2 AND ($3='' OR c.supplier_organization_id=NULLIF($3,'')::uuid) UNION ALL
		SELECT 'receipt.confirmed',count(*) FROM app.receipt_confirmations r JOIN app.credit_requests c ON c.id=r.credit_request_id WHERE r.state='confirmed' AND r.received_at >= $1 AND r.received_at < $2 AND ($3='' OR c.supplier_organization_id=NULLIF($3,'')::uuid) UNION ALL
		SELECT 'obligation.activated',count(*) FROM app.obligations WHERE activated_at >= $1 AND activated_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid) UNION ALL
		SELECT 'payment.confirmed',count(*) FROM app.payments WHERE state='recognized' AND recognized_at >= $1 AND recognized_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid) UNION ALL
		SELECT 'trade_line.created',count(*) FROM app.trade_lines WHERE created_at >= $1 AND created_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid) UNION ALL
		SELECT 'dispute.opened',count(*) FROM app.disputes WHERE opened_at >= $1 AND opened_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)
	), observed AS (SELECT name,count(*) n FROM app.analytics_events WHERE occurred_at >= $1 AND occurred_at < $2 AND ($3='' OR organization_id_hash=encode(digest($3,'sha256'),'hex')) GROUP BY name)
	SELECT e.event,e.n,COALESCE(o.n,0),e.n-COALESCE(o.n,0) FROM expected e LEFT JOIN observed o ON o.name=e.event ORDER BY e.event`
	rows, err = s.pool.Query(ctx, reconciliationSQL, from, to, org)
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
	return result, nil
}
