-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.capture_financial_notice() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE buyer uuid; organization uuid; template_name text; notice_key text;
BEGIN
 IF TG_OP='UPDATE' AND NEW.state IS NOT DISTINCT FROM OLD.state AND (to_jsonb(NEW)->'remaining_disputed_kobo') IS NOT DISTINCT FROM (to_jsonb(OLD)->'remaining_disputed_kobo') THEN RETURN NEW; END IF;
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
  PERFORM app.enqueue_financial_notice('dispute:'||NEW.id||':'||NEW.state||':'||NEW.remaining_disputed_kobo,'DisputeUpdated',NEW.buyer_user_id,NULL,NEW.remaining_disputed_kobo,NEW.id);
  PERFORM app.enqueue_financial_notice('dispute:'||NEW.id||':'||NEW.state||':'||NEW.remaining_disputed_kobo,'DisputeUpdated',NULL,NEW.supplier_organization_id,NEW.remaining_disputed_kobo,NEW.id);
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
-- +goose StatementBegin
CREATE FUNCTION app.capture_adjustment_notice() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE buyer uuid;
BEGIN
 IF NEW.action NOT IN ('write_off','fee_waiver') OR NEW.resource_type<>'obligation' THEN RETURN NEW; END IF;
 SELECT c.buyer_user_id INTO STRICT buyer FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id WHERE o.id=NEW.resource_id;
 PERFORM app.enqueue_financial_notice('adjustment:'||NEW.id,'FinancialAdjustmentRecorded',buyer,NULL,(NEW.metadata->>'amount_kobo')::bigint,NEW.resource_id);
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER adjustment_notice AFTER INSERT ON app.operation_actions FOR EACH ROW EXECUTE FUNCTION app.capture_adjustment_notice();
CREATE TRIGGER financial_adjustment_actions_immutable BEFORE UPDATE OR DELETE ON app.operation_actions FOR EACH ROW EXECUTE FUNCTION app.reject_operations_command_mutation();
-- +goose Down
DROP TRIGGER financial_adjustment_actions_immutable ON app.operation_actions;
DROP TRIGGER adjustment_notice ON app.operation_actions;
DROP FUNCTION app.capture_adjustment_notice();
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.capture_financial_notice() RETURNS trigger LANGUAGE plpgsql AS $$
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
