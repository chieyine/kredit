-- +goose Up
-- Fixed metric queries only. No caller-controlled SQL or unrestricted source rows.
-- +goose StatementBegin
CREATE FUNCTION app.pilot_metric(timestamptz,timestamptz,text,text) RETURNS double precision
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE result double precision;
BEGIN
IF NOT app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']) THEN RAISE EXCEPTION 'scorecard authority required'; END IF;
 IF $1 IS NULL OR $2 IS NULL OR $1 >= $2 OR $2-$1>interval '366 days' THEN RAISE EXCEPTION 'invalid scorecard window'; END IF;
 CASE $4
 WHEN 'gross_trade_credit_volume' THEN SELECT * INTO result FROM (SELECT COALESCE(sum(principal_kobo),0)::float8 FROM app.obligations WHERE activated_at >= $1 AND activated_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)) metric_value;
 WHEN 'active_suppliers' THEN SELECT * INTO result FROM (SELECT count(DISTINCT supplier_organization_id)::float8 FROM app.obligations WHERE activated_at >= $1 AND activated_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)) metric_value;
 WHEN 'time_to_first_accepted_sale' THEN SELECT * INTO result FROM (WITH firsts AS (SELECT c.supplier_organization_id,min(a.accepted_at) accepted_at FROM app.agreement_acceptances a JOIN app.credit_requests c ON c.id=a.credit_request_id WHERE a.accepted_at >= $1 AND a.accepted_at < $2 AND ($3='' OR c.supplier_organization_id=NULLIF($3,'')::uuid) GROUP BY c.supplier_organization_id) SELECT COALESCE(avg(extract(epoch FROM (f.accepted_at-p.created_at))/3600),0)::float8 FROM firsts f JOIN app.supplier_onboarding_profiles p ON p.organization_id=f.supplier_organization_id) metric_value;
 WHEN 'sent_to_acceptance' THEN SELECT * INTO result FROM (SELECT CASE WHEN count(*) FILTER(WHERE name='credit.sent')=0 THEN 0 ELSE 100.0*count(*) FILTER(WHERE name='credit.accepted')/count(*) FILTER(WHERE name='credit.sent') END::float8 FROM app.analytics_events WHERE occurred_at >= $1 AND occurred_at < $2 AND ($3='' OR organization_id_hash=encode(public.digest($3,'sha256'),'hex'))) metric_value;
 WHEN 'invitation_to_verification' THEN SELECT * INTO result FROM (SELECT CASE WHEN count(*)=0 THEN 0 ELSE 100.0*count(*) FILTER(WHERE EXISTS(SELECT 1 FROM app.businesses b WHERE b.owner_user_id=i.accepted_by_user_id AND b.status='verified'))/count(*) END::float8 FROM app.buyer_invitations i WHERE i.created_at >= $1 AND i.created_at < $2 AND ($3='' OR i.organization_id=NULLIF($3,'')::uuid)) metric_value;
 WHEN 'acceptance_to_release' THEN SELECT * INTO result FROM (SELECT COALESCE(avg(extract(epoch FROM (g.released_at-a.accepted_at))/3600),0)::float8 FROM app.agreement_acceptances a JOIN app.goods_releases g USING(credit_request_id) JOIN app.credit_requests c ON c.id=a.credit_request_id WHERE g.released_at >= $1 AND g.released_at < $2 AND ($3='' OR c.supplier_organization_id=NULLIF($3,'')::uuid)) metric_value;
 WHEN 'release_to_receipt' THEN SELECT * INTO result FROM (SELECT COALESCE(avg(extract(epoch FROM (r.received_at-g.released_at))/3600),0)::float8 FROM app.goods_releases g JOIN app.receipt_confirmations r USING(credit_request_id) JOIN app.credit_requests c ON c.id=g.credit_request_id WHERE r.received_at >= $1 AND r.received_at < $2 AND ($3='' OR c.supplier_organization_id=NULLIF($3,'')::uuid)) metric_value;
 WHEN 'repeat_sale_rate' THEN SELECT * INTO result FROM (WITH pairs AS (SELECT supplier_organization_id,buyer_business_id,count(*) n FROM app.obligations WHERE activated_at >= $1 AND activated_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid) GROUP BY supplier_organization_id,buyer_business_id) SELECT CASE WHEN count(*)=0 THEN 0 ELSE 100.0*count(*) FILTER(WHERE n>1)/count(*) END::float8 FROM pairs) metric_value;
 WHEN 'trade_line_utilization' THEN SELECT * INTO result FROM (SELECT CASE WHEN COALESCE(sum(approved_limit_kobo),0)=0 THEN 0 ELSE 100.0*sum(current_exposure_kobo+reserved_pending_kobo)/sum(approved_limit_kobo) END::float8 FROM app.trade_lines WHERE state='ACTIVE' AND $1::timestamptz < $2::timestamptz AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)) metric_value;
 WHEN 'supplier_retention' THEN SELECT * INTO result FROM (WITH previous AS (SELECT DISTINCT supplier_organization_id FROM app.obligations WHERE activated_at >= $1::timestamptz-($2::timestamptz-$1::timestamptz) AND activated_at < $1 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)), current_window AS (SELECT DISTINCT supplier_organization_id FROM app.obligations WHERE activated_at >= $1 AND activated_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)) SELECT CASE WHEN (SELECT count(*) FROM previous)=0 THEN 0 ELSE 100.0*(SELECT count(*) FROM previous p JOIN current_window c USING(supplier_organization_id))/(SELECT count(*) FROM previous) END::float8) metric_value;
 WHEN 'on_time_payment_rate' THEN SELECT * INTO result FROM (WITH paid AS (
 SELECT o.id,o.credit_request_id,o.principal_kobo
 FROM app.obligations o JOIN app.payments p ON p.obligation_id=o.id AND p.state='recognized'
 WHERE o.payment_status='PAID' AND ($3='' OR o.supplier_organization_id=NULLIF($3,'')::uuid)
 GROUP BY o.id,o.credit_request_id,o.principal_kobo
 HAVING sum(p.amount_kobo)=o.principal_kobo AND max(p.paid_at)>=$1 AND max(p.paid_at)<$2
), timely AS (
 SELECT o.id,COALESCE(bool_and(pa.payment_id IS NOT NULL AND p.paid_at<=COALESCE(i.collection_at,c.collection_at)),false)
  AND COALESCE(sum(pa.amount_kobo),0)=o.principal_kobo AS on_time
 FROM paid o JOIN app.credit_requests c ON c.id=o.credit_request_id
 JOIN app.payments p ON p.obligation_id=o.id AND p.state='recognized'
 LEFT JOIN app.payment_allocations pa ON pa.payment_id=p.id
 LEFT JOIN app.schedule_items i ON i.id=pa.schedule_item_id
 GROUP BY o.id,o.principal_kobo
) SELECT CASE WHEN count(*)=0 THEN 0 ELSE 100.0*count(*) FILTER(WHERE on_time)/count(*) END::float8 FROM timely) metric_value;
 WHEN 'failed_collection_recovery' THEN SELECT * INTO result FROM (WITH failed AS (
 SELECT ca.obligation_id,min(COALESCE(ca.final_at,ca.requested_at)) failed_at
 FROM app.collection_attempts ca JOIN app.obligations o ON o.id=ca.obligation_id
 WHERE ca.state='FAILED' AND ca.requested_at>=$1 AND ca.requested_at<$2
  AND ($3='' OR o.supplier_organization_id=NULLIF($3,'')::uuid)
 GROUP BY ca.obligation_id
), recovered AS (
 SELECT f.obligation_id FROM failed f WHERE EXISTS(
  SELECT 1 FROM app.collection_attempts ca WHERE ca.obligation_id=f.obligation_id
   AND ca.state IN('SUCCEEDED','PARTIAL') AND ca.succeeded_amount_kobo>0
   AND ca.requested_at>=f.failed_at AND ca.final_at<$2
 )
) SELECT CASE WHEN (SELECT count(*) FROM failed)=0 THEN 0
 ELSE 100.0*(SELECT count(*) FROM recovered)/(SELECT count(*) FROM failed) END::float8) metric_value;
 WHEN 'dispute_rate' THEN SELECT * INTO result FROM (SELECT CASE WHEN count(DISTINCT o.id)=0 THEN 0 ELSE 100.0*count(DISTINCT d.obligation_id)/count(DISTINCT o.id) END::float8 FROM app.obligations o LEFT JOIN app.disputes d ON d.obligation_id=o.id AND d.opened_at >= $1 AND d.opened_at < $2 WHERE o.activated_at >= $1 AND o.activated_at < $2 AND ($3='' OR o.supplier_organization_id=NULLIF($3,'')::uuid)) metric_value;
 WHEN 'receipt_issue_rate' THEN SELECT * INTO result FROM (SELECT CASE WHEN count(*)=0 THEN 0 ELSE 100.0*count(*) FILTER(WHERE r.state='issue_raised')/count(*) END::float8 FROM app.receipt_confirmations r JOIN app.credit_requests c ON c.id=r.credit_request_id WHERE r.received_at >= $1 AND r.received_at < $2 AND ($3='' OR c.supplier_organization_id=NULLIF($3,'')::uuid)) metric_value;
 WHEN 'provider_reliability' THEN SELECT * INTO result FROM (SELECT CASE WHEN count(*)=0 THEN NULL ELSE 100.0*count(*) FILTER(WHERE ca.state IN('SUCCEEDED','PARTIAL'))/count(*) END::float8 FROM app.collection_attempts ca JOIN app.obligations o ON o.id=ca.obligation_id WHERE ca.final_at >= $1 AND ca.final_at < $2 AND ca.state IN('SUCCEEDED','PARTIAL','FAILED','CANCELLED') AND ($3='' OR o.supplier_organization_id=NULLIF($3,'')::uuid)) metric_value;
 WHEN 'recognized_loss_rate' THEN SELECT * INTO result FROM (WITH losses AS (SELECT resource_id,COALESCE(sum((metadata->>'amount_kobo')::bigint),0) amount FROM app.operation_actions WHERE action='write_off' AND created_at >= $1 AND created_at < $2 AND ($3='' OR organization_id=NULLIF($3,'')::uuid) GROUP BY resource_id), exposure AS (SELECT COALESCE(sum(o.principal_kobo),0) principal FROM app.obligations o WHERE EXISTS(SELECT 1 FROM losses l WHERE l.resource_id=o.id)) SELECT CASE WHEN exposure.principal=0 THEN 0 ELSE 100.0*(SELECT COALESCE(sum(amount),0) FROM losses)/exposure.principal END::float8 FROM exposure) metric_value;
 WHEN 'support_intervention_rate' THEN SELECT * INTO result FROM (WITH active AS (SELECT count(DISTINCT supplier_organization_id) n FROM app.obligations WHERE activated_at >= $1 AND activated_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)), cases AS (SELECT count(*) n FROM app.support_cases WHERE created_at >= $1 AND created_at < $2 AND ($3='' OR organization_id=NULLIF($3,'')::uuid)) SELECT CASE WHEN active.n=0 THEN 0 ELSE 100.0*cases.n/active.n END::float8 FROM active,cases) metric_value;
 WHEN 'accessibility_defects' THEN SELECT * INTO result FROM (SELECT count(*)::float8 FROM app.support_cases WHERE subject_type='accessibility_defect' AND created_at < $2 AND state IN('OPEN','IN_PROGRESS') AND $1::timestamptz < $2::timestamptz AND ($3='' OR organization_id=NULLIF($3,'')::uuid)) metric_value;
 WHEN 'mandate_authorization_dropoff' THEN SELECT * INTO result FROM (SELECT CASE WHEN count(*)=0 THEN 0 ELSE 100.0*count(*) FILTER(WHERE m.activated_at IS NULL AND m.status<>'ACTIVE')/count(*) END::float8 FROM app.mandates m JOIN app.credit_requests c ON c.id=m.credit_request_id WHERE m.created_at >= $1 AND m.created_at < $2 AND ($3='' OR c.supplier_organization_id=NULLIF($3,'')::uuid)) metric_value;
 WHEN 'voluntary_payment_share' THEN SELECT * INTO result FROM (SELECT CASE WHEN COALESCE(sum(amount_kobo) FILTER(WHERE source_type<>'adjustment'),0)=0 THEN 0 ELSE 100.0*COALESCE(sum(amount_kobo) FILTER(WHERE source_type NOT IN ('kredit_collection','collected','adjustment')),0)/sum(amount_kobo) FILTER(WHERE source_type<>'adjustment') END::float8 FROM app.payments WHERE state='recognized' AND recognized_at >= $1 AND recognized_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)) metric_value;
 WHEN 'deemed_acceptance_share' THEN SELECT * INTO result FROM (SELECT CASE WHEN count(*)=0 THEN 0 ELSE 100.0*count(*) FILTER(WHERE rc.issue_reason='deemed_acceptance_auto_activated')/count(*) END::float8 FROM app.receipt_confirmations rc JOIN app.credit_requests c ON c.id=rc.credit_request_id WHERE rc.state='confirmed' AND rc.received_at >= $1 AND rc.received_at < $2 AND ($3='' OR c.supplier_organization_id=NULLIF($3,'')::uuid)) metric_value;
 WHEN 'manual_touches_per_obligation' THEN SELECT * INTO result FROM (WITH activated AS (SELECT count(*) n FROM app.obligations WHERE activated_at >= $1 AND activated_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)), touches AS (SELECT (SELECT count(*) FROM app.support_cases WHERE created_at >= $1 AND created_at < $2 AND ($3='' OR organization_id=NULLIF($3,'')::uuid)) + (SELECT count(*) FROM app.operation_actions WHERE created_at >= $1 AND created_at < $2 AND ($3='' OR organization_id=NULLIF($3,'')::uuid)) + (SELECT count(*) FROM app.correction_requests WHERE created_at >= $1 AND created_at < $2 AND ($3='' OR organization_id=NULLIF($3,'')::uuid)) + (SELECT count(*) FROM app.disputes WHERE opened_at >= $1 AND opened_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)) n) SELECT CASE WHEN activated.n=0 THEN 0 ELSE touches.n::float8/activated.n END::float8 FROM activated,touches) metric_value;
 ELSE RAISE EXCEPTION 'unknown scorecard metric';
 END CASE;
 RETURN result;
END $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.pilot_metric(timestamptz,timestamptz,text,text) FROM PUBLIC;
-- +goose StatementBegin
CREATE FUNCTION app.pilot_reconciliation(timestamptz,timestamptz,text)
RETURNS TABLE(event text,source_count bigint,event_count bigint,difference bigint)
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
BEGIN
IF NOT app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']) THEN RAISE EXCEPTION 'scorecard authority required'; END IF;
 IF $1 IS NULL OR $2 IS NULL OR $1 >= $2 OR $2-$1>interval '366 days' THEN RAISE EXCEPTION 'invalid scorecard window'; END IF;
 RETURN QUERY WITH expected(event,n) AS (
		SELECT 'customer.invited',count(*) FROM app.buyer_invitations WHERE created_at >= $1 AND created_at < $2 AND ($3='' OR organization_id=NULLIF($3,'')::uuid) UNION ALL
		SELECT 'credit.drafted',count(*) FROM app.credit_requests WHERE created_at >= $1 AND created_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid) UNION ALL
		SELECT 'credit.accepted',count(*) FROM app.agreement_acceptances a JOIN app.credit_requests c ON c.id=a.credit_request_id WHERE a.accepted_at >= $1 AND a.accepted_at < $2 AND ($3='' OR c.supplier_organization_id=NULLIF($3,'')::uuid) UNION ALL
		SELECT 'goods.released',count(*) FROM app.goods_releases g JOIN app.credit_requests c ON c.id=g.credit_request_id WHERE g.released_at >= $1 AND g.released_at < $2 AND ($3='' OR c.supplier_organization_id=NULLIF($3,'')::uuid) UNION ALL
		SELECT 'receipt.confirmed',count(*) FROM app.receipt_confirmations r JOIN app.credit_requests c ON c.id=r.credit_request_id WHERE r.state='confirmed' AND r.received_at >= $1 AND r.received_at < $2 AND ($3='' OR c.supplier_organization_id=NULLIF($3,'')::uuid) UNION ALL
		SELECT 'obligation.activated',count(*) FROM app.obligations WHERE activated_at >= $1 AND activated_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid) UNION ALL
		SELECT 'payment.confirmed',count(*) FROM app.payments WHERE state='recognized' AND recognized_at >= $1 AND recognized_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid) UNION ALL
		SELECT 'trade_line.created',count(*) FROM app.trade_lines WHERE created_at >= $1 AND created_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid) UNION ALL
		SELECT 'dispute.opened',count(*) FROM app.disputes WHERE opened_at >= $1 AND opened_at < $2 AND ($3='' OR supplier_organization_id=NULLIF($3,'')::uuid)
	), observed AS (SELECT name,count(*) n FROM app.analytics_events WHERE occurred_at >= $1 AND occurred_at < $2 AND ($3='' OR organization_id_hash=encode(public.digest($3,'sha256'),'hex')) GROUP BY name)
	SELECT e.event,e.n,COALESCE(o.n,0),e.n-COALESCE(o.n,0) FROM expected e LEFT JOIN observed o ON o.name=e.event ORDER BY e.event;
END $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.pilot_reconciliation(timestamptz,timestamptz,text) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
 GRANT EXECUTE ON FUNCTION app.pilot_metric(timestamptz,timestamptz,text,text),app.pilot_reconciliation(timestamptz,timestamptz,text) TO kredit_app;
END IF; END $$;
-- +goose StatementEnd
-- +goose Down
DROP FUNCTION app.pilot_metric(timestamptz,timestamptz,text,text),app.pilot_reconciliation(timestamptz,timestamptz,text);
