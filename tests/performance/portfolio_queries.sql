-- Run only against an isolated performance database populated with at least
-- 100,000 synthetic obligations. The first guard deliberately fails smaller
-- datasets so a smoke fixture cannot be mislabeled as a portfolio test.
DO $$
BEGIN
  IF (SELECT count(*) FROM app.obligations) < 100000 THEN
    RAISE EXCEPTION 'performance dataset requires at least 100000 obligations';
  END IF;
END $$;

EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
SELECT supplier_organization_id, payment_status, sum(outstanding_kobo)
FROM app.obligations
GROUP BY supplier_organization_id, payment_status;

EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
SELECT i.id, i.collection_at, i.principal_due_kobo - i.allocated_kobo AS due_kobo
FROM app.schedule_items i
WHERE i.collection_at <= now() AND i.state IN ('OPEN','IN_GRACE','OVERDUE','PARTIALLY_PAID')
ORDER BY i.collection_at
LIMIT 10000;
