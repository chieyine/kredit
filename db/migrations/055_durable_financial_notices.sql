-- +goose Up
-- Capture notification intent in the financial transaction, including writes
-- performed by workers or administrative recovery rather than HTTP handlers.
-- +goose StatementBegin
CREATE FUNCTION app.enqueue_financial_notice(notice_key text, template_name text, recipient uuid, organization uuid, amount bigint, reference_id uuid) RETURNS void LANGUAGE sql AS $$
 INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key)
 VALUES('financial_notice',reference_id::text,'notification.requested',jsonb_build_object('notification',jsonb_build_object(
 'ID',notice_key,'Type',template_name,'RecipientID',COALESCE(recipient::text,''),'OrganizationID',COALESCE(organization::text,''),
 'Priority','critical','AmountKobo',amount,'Currency','NGN','Reference',reference_id::text,'NextAction','Review your financial activity in Kredit')),'notice:'||notice_key||':'||COALESCE(recipient::text,organization::text))
 ON CONFLICT(idempotency_key) DO NOTHING;
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION app.capture_financial_notice() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE buyer uuid; organization uuid; template_name text; notice_key text;
BEGIN
 IF TG_OP='UPDATE' AND NEW.state IS NOT DISTINCT FROM OLD.state THEN RETURN NEW; END IF;
 IF TG_TABLE_NAME='payments' THEN
  IF NEW.state='recognized' THEN
   PERFORM app.enqueue_financial_notice('payment-recorded:'||NEW.id,'PaymentRecorded',NEW.buyer_user_id,NULL,NEW.amount_kobo,NEW.id);
  ELSIF TG_OP='UPDATE' AND NEW.state='reversed' THEN
   PERFORM app.enqueue_financial_notice('payment-reversed:'||NEW.id,'PaymentReversed',NEW.buyer_user_id,NULL,NEW.amount_kobo,NEW.id);
  END IF;
 ELSIF TG_TABLE_NAME='payment_claims' THEN
  IF NEW.state='pending' THEN
   PERFORM app.enqueue_financial_notice('payment-claim:'||NEW.id,'BuyerPaymentClaimed',NULL,NEW.supplier_organization_id,NEW.amount_kobo,NEW.id);
  ELSE
   PERFORM app.enqueue_financial_notice('payment-claim-decision:'||NEW.id,'PaymentClaimDecision',NEW.buyer_user_id,NULL,NEW.amount_kobo,NEW.id);
  END IF;
 ELSIF TG_TABLE_NAME='disputes' THEN
  PERFORM app.enqueue_financial_notice('dispute:'||NEW.id||':'||NEW.state,'DisputeUpdated',NEW.buyer_user_id,NULL,NEW.remaining_disputed_kobo,NEW.id);
  PERFORM app.enqueue_financial_notice('dispute:'||NEW.id||':'||NEW.state,'DisputeUpdated',NULL,NEW.supplier_organization_id,NEW.remaining_disputed_kobo,NEW.id);
 ELSIF TG_TABLE_NAME='drawdowns' THEN
  SELECT buyer_user_id,supplier_organization_id INTO STRICT buyer,organization FROM app.trade_lines WHERE id=NEW.trade_line_id;
  CASE NEW.state
   WHEN 'PENDING_BUYER_CONFIRMATION' THEN
    PERFORM app.enqueue_financial_notice('drawdown-confirmation-required:'||NEW.id,'TradeLineDrawdownConfirmationRequired',buyer,NULL,NEW.principal_kobo,NEW.id);
   WHEN 'BUYER_CONFIRMED' THEN
    PERFORM app.enqueue_financial_notice('drawdown-confirmed:'||NEW.id,'TradeLineDrawdownConfirmed',NULL,organization,NEW.principal_kobo,NEW.id);
    PERFORM app.enqueue_financial_notice('drawdown-safe-to-release:'||NEW.id,'TradeLineDrawdownSafeToRelease',NULL,organization,NEW.principal_kobo,NEW.id);
   WHEN 'GOODS_RELEASED' THEN
    PERFORM app.enqueue_financial_notice('drawdown-released:'||NEW.id,'TradeLineDrawdownGoodsReleased',buyer,NULL,NEW.principal_kobo,NEW.id);
    PERFORM app.enqueue_financial_notice('drawdown-receipt-required:'||NEW.id,'TradeLineDrawdownReceiptRequired',buyer,NULL,NEW.principal_kobo,NEW.id);
   WHEN 'ACTIVATED' THEN
    PERFORM app.enqueue_financial_notice('drawdown-receipt-confirmed:'||NEW.id,'TradeLineDrawdownReceiptConfirmed',NULL,organization,NEW.principal_kobo,NEW.id);
   WHEN 'RECEIPT_ISSUE_REPORTED' THEN
    PERFORM app.enqueue_financial_notice('drawdown-receipt-issue:'||NEW.id,'TradeLineDrawdownReceiptIssueReported',NULL,organization,NEW.principal_kobo,NEW.id);
   WHEN 'CANCELLED', 'EXPIRED' THEN
    PERFORM app.enqueue_financial_notice('drawdown-cancelled:'||NEW.id,'TradeLineDrawdownCancelled',buyer,NULL,NEW.principal_kobo,NEW.id);
    PERFORM app.enqueue_financial_notice('drawdown-cancelled:'||NEW.id,'TradeLineDrawdownCancelled',NULL,organization,NEW.principal_kobo,NEW.id);
   ELSE NULL;
  END CASE;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER payment_notice AFTER INSERT OR UPDATE ON app.payments FOR EACH ROW EXECUTE FUNCTION app.capture_financial_notice();
CREATE TRIGGER claim_notice AFTER INSERT OR UPDATE ON app.payment_claims FOR EACH ROW EXECUTE FUNCTION app.capture_financial_notice();
CREATE TRIGGER dispute_notice AFTER INSERT OR UPDATE ON app.disputes FOR EACH ROW EXECUTE FUNCTION app.capture_financial_notice();
CREATE TRIGGER drawdown_notice AFTER INSERT OR UPDATE ON app.drawdowns FOR EACH ROW EXECUTE FUNCTION app.capture_financial_notice();
-- +goose Down
DROP TRIGGER drawdown_notice ON app.drawdowns;
DROP TRIGGER dispute_notice ON app.disputes;
DROP TRIGGER claim_notice ON app.payment_claims;
DROP TRIGGER payment_notice ON app.payments;
DROP FUNCTION app.capture_financial_notice();
DROP FUNCTION app.enqueue_financial_notice(text,text,uuid,uuid,bigint,uuid);
