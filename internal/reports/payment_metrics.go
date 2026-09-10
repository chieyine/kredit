package reports

// The reporting cohort is defined by the final payment of fully repaid
// principal, not by any partial payment that happened inside the window.
const daysToPaymentSQL = `WITH paid AS (
 SELECT o.id,o.activated_at,max(p.paid_at) paid_at
 FROM app.obligations o JOIN app.payments p ON p.obligation_id=o.id AND p.state='recognized'
 WHERE o.payment_status='PAID' AND ($3='' OR o.supplier_organization_id=NULLIF($3,'')::uuid)
 GROUP BY o.id,o.activated_at,o.principal_kobo
 HAVING sum(p.amount_kobo)=o.principal_kobo AND max(p.paid_at)>=$1 AND max(p.paid_at)<$2
) SELECT COALESCE(avg(extract(epoch FROM (paid_at-activated_at))/86400),0)::float8 FROM paid`

// Each allocated instalment has its own agreed collection deadline. Comparing
// every payment to the first due date mislabels valid instalment repayments.
const onTimePaymentSQL = `WITH paid AS (
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
) SELECT CASE WHEN count(*)=0 THEN 0 ELSE 100.0*count(*) FILTER(WHERE on_time)/count(*) END::float8 FROM timely`

const failedCollectionRecoverySQL = `WITH failed AS (
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
 ELSE 100.0*(SELECT count(*) FROM recovered)/(SELECT count(*) FROM failed) END::float8`
