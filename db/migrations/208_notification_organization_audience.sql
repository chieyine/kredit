-- +goose Up
-- Buyer notices can also carry a supplier organization, so organization_id
-- alone is not an authority requirement. Preserve the broadcast audience.
ALTER TABLE app.notifications ADD COLUMN organization_audience boolean NOT NULL DEFAULT false;

-- Existing broadcasts were expanded using event_id + ':' + member.user_id.
-- Restrict the backfill to the known broadcast templates so buyer notices
-- retain their independent purchasing authority.
UPDATE app.notifications SET organization_audience=true
WHERE supplier_organization_id IS NOT NULL
  AND right(event_reference,37)=':' || recipient_id::text
  AND template IN (
    'SupplierSensitiveSettingChanged','SupplierVerificationOutcome','SupplierPilotReady',
    'FeeInvoiceIssued','FeeInvoicePaymentRecorded','FeeInvoiceRefundRecorded',
    'SellerSettlementRecorded','BuyerPaymentClaimed','MandateCancelled','ConsumerPurchaseUpdated','DisputeUpdated',
    'TradeLineDrawdownConfirmed','TradeLineDrawdownSafeToRelease','TradeLineDrawdownActivated',
    'TradeLineDrawdownReceiptIssueReported','TradeLineDrawdownReceiptConfirmed','TradeLineDrawdownCancelled'
  );

-- The delivery/recovery paths may update transport state, but cannot change
-- the audience boundary under an already recorded message identity.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.preserve_notification_intent() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.event_fingerprint IS DISTINCT FROM OLD.event_fingerprint
     OR NEW.organization_audience IS DISTINCT FROM OLD.organization_audience THEN
    RAISE EXCEPTION 'notification intent is immutable';
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose Down
-- Do not erase the audience boundary of messages already queued for delivery.
SELECT 1;
