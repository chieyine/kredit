package usercontrol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Explicit column lists prevent a future credential or internal-review column
// from silently entering a customer's download. Every section is subject-bound;
// company-wide records require the separate authorized business report flow.
func privacyExportPayload(ctx context.Context, tx pgx.Tx, requestID, userID string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	result := map[string]any{"request_id": requestID, "generated_at": time.Now().UTC(), "format_version": 2, "scope": "Personal account records and buyer-side transactions. Business-wide reports and uploaded file contents are obtained through their separately authorized download routes. Authentication secrets, live access links and internal security diagnostics are excluded."}
	var profile json.RawMessage
	if err := tx.QueryRow(ctx, `SELECT to_jsonb(p) FROM (SELECT id,normalized_email,normalized_phone,display_name,status,last_authenticated_at,created_at,version FROM app.users WHERE id=$1::uuid) p`, userID).Scan(&profile); err != nil {
		return nil, err
	}
	result["profile"] = profile
	sections := []struct{ name, query string }{
		{"memberships", `SELECT id,organization_id,role,status,invited_at,accepted_at,created_at FROM app.memberships WHERE user_id=$1::uuid ORDER BY id`},
		{"persons", `SELECT id,full_name,status,created_at FROM app.persons WHERE user_id=$1::uuid ORDER BY id`},
		{"owned_businesses", `SELECT id,organization_id,legal_name,trading_name,business_type,business_address,industry,status,created_at FROM app.businesses WHERE owner_user_id=$1::uuid ORDER BY id`},
		{"seller_consents", `SELECT id,supplier_organization_id,consent_type,version,granted,created_at FROM app.relationship_consents WHERE buyer_user_id=$1::uuid ORDER BY created_at,id`},
		{"notification_preferences", `SELECT preferred_channel,fallback_channel,opted_out,quiet_start_hour,quiet_end_hour,timezone,payment_reminders_enabled,product_updates_enabled,updated_at,version FROM app.notification_preferences WHERE recipient_id=$1::uuid`},
		{"notification_history", `SELECT id,channel,template,template_version,event_reference,state,scheduled_at,sent_at,delivered_at,read_at,failed_at,supplier_organization_id,priority FROM app.notifications WHERE recipient_id=$1::uuid ORDER BY scheduled_at,id`},
		{"privacy_requests", `SELECT id,organization_id,request_type,state,identity_verified_at,due_at,details,decision_reason,retention_outcome,legal_hold_applies,completion_reason,completed_at,created_at FROM app.privacy_requests WHERE requester_user_id=$1::uuid ORDER BY created_at,id`},
		{"processing_restrictions", `SELECT id,privacy_request_id,scope,reason,active,created_at,lifted_at FROM app.processing_restrictions WHERE user_id=$1::uuid ORDER BY created_at,id`},
		{"credit_requests", `SELECT id,supplier_organization_id,buyer_business_id,principal_kobo,currency,goods_description,invoice_reference,due_date,grace_hours,collection_at,state,created_at,updated_at,fee_terms FROM app.credit_requests WHERE buyer_user_id=$1::uuid ORDER BY created_at,id`},
		{"obligations", `SELECT o.id,o.credit_request_id,o.supplier_organization_id,o.principal_kobo,o.currency,o.lifecycle_status,o.payment_status,o.outstanding_kobo,o.base_fee_kobo,o.activated_at FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id WHERE c.buyer_user_id=$1::uuid ORDER BY o.activated_at,o.id`},
		{"payments", `SELECT id,obligation_id,supplier_organization_id,source_type,amount_kobo,currency,state,paid_at,recognized_at,reversal_of,collection_fee_kobo FROM app.payments WHERE buyer_user_id=$1::uuid ORDER BY paid_at,id`},
		{"payment_claims", `SELECT id,obligation_id,supplier_organization_id,amount_kobo,currency,paid_at,source_account_masked,transfer_reference,state,hold_expires_at,review_reason,payment_id,created_at,reviewed_at FROM app.payment_claims WHERE buyer_user_id=$1::uuid ORDER BY created_at,id`},
		{"system_acceptances", `SELECT id,credit_request_id,supplier_organization_id,release_id,notification_id,minimum_seconds,recorded_at FROM app.system_acceptances WHERE buyer_user_id=$1::uuid ORDER BY recorded_at,id`},
		{"agreement_acceptances", `SELECT id,credit_request_id,agreement_version_id,business_id,acceptance_method,agreement_hash,accepted_at FROM app.agreement_acceptances WHERE accepting_user_id=$1::uuid ORDER BY accepted_at,id`},
		{"trade_lines", `SELECT id,supplier_organization_id,buyer_business_id,approved_limit_kobo,current_exposure_kobo,reserved_pending_kobo,available_limit_kobo,cadence,start_at,end_at,state,terms_version,created_at,updated_at FROM app.trade_lines WHERE buyer_user_id=$1::uuid ORDER BY created_at,id`},
		{"drawdowns", `SELECT d.id,d.trade_line_id,d.principal_kobo,d.goods_description,d.invoice_reference,d.state,d.obligation_id,d.buyer_confirmed_at,d.activated_at,d.due_date,d.collection_at,d.agreement_hash,d.receipt_state,d.receipt_issue_reason,d.fee_terms,d.legal_versions FROM app.drawdowns d JOIN app.trade_lines l ON l.id=d.trade_line_id WHERE l.buyer_user_id=$1::uuid ORDER BY d.created_at,d.id`},
		{"bank_permissions", `SELECT m.id,m.provider,m.mandate_type,m.amount_ceiling_kobo,m.starts_at,m.ends_at,m.state,m.accepted_disclosure_version,m.created_at,m.supplier_organization_id FROM app.payment_mandates m WHERE (m.buyer_subject_type='person' AND EXISTS(SELECT 1 FROM app.persons p WHERE p.id=m.buyer_subject_id AND p.user_id=$1::uuid)) OR (m.buyer_subject_type='business' AND EXISTS(SELECT 1 FROM app.businesses b WHERE b.id=m.buyer_subject_id AND b.owner_user_id=$1::uuid)) ORDER BY m.created_at,m.id`},
		{"disputes", `SELECT id,obligation_id,supplier_organization_id,total_disputed_kobo,remaining_disputed_kobo,reason,explanation,state,collection_effect,opened_at,resolved_at FROM app.disputes WHERE buyer_user_id=$1::uuid ORDER BY opened_at,id`},
		{"uploaded_files", `SELECT id,organization_id,purpose,file_name,content_type,size_bytes,sha256,scan_state,created_at,upload_completed_at FROM app.documents WHERE uploaded_by=$1::uuid ORDER BY created_at,id`},
		{"support_cases", `SELECT id,organization_id,subject_type,subject_id,state,created_at,updated_at FROM app.support_cases WHERE opened_by=$1::uuid ORDER BY created_at,id`},
		{"support_history", `SELECT e.id,e.case_id,e.action,e.note,e.created_at FROM app.support_case_events e JOIN app.support_cases c ON c.id=e.case_id WHERE c.opened_by=$1::uuid ORDER BY e.created_at,e.id`},
		{"correction_requests", `SELECT id,organization_id,subject_type,subject_id,source_event_id,reason,evidence,state,created_at,updated_at FROM app.correction_requests WHERE requested_by=$1::uuid ORDER BY created_at,id`},
		{"correction_decisions", `SELECT d.id,d.request_id,d.outcome,d.reason,d.correction_event_id,d.decided_at FROM app.correction_decisions d JOIN app.correction_requests r ON r.id=d.request_id WHERE r.requested_by=$1::uuid ORDER BY d.decided_at,d.id`},
	}
	total := len(profile)
	for _, section := range sections {
		var rows json.RawMessage
		if err := tx.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(personal_row)),'[]'::jsonb) FROM (`+section.query+`) personal_row`, userID).Scan(&rows); err != nil {
			return nil, fmt.Errorf("prepare %s export: %w", section.name, err)
		}
		total += len(rows)
		if total > 10*1024*1024 {
			return nil, errors.New("this export needs assisted delivery because it exceeds the interactive download size")
		}
		result[section.name] = rows
	}
	return json.Marshal(result)
}
