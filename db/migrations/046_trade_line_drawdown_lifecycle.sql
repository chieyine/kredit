-- +goose Up

-- Existing ACTIVATED rows without obligations cannot be safely guessed into
-- financial records. Fail closed so an operator must reconcile them before
-- applying this migration. Pre-production demo data is recreated by the seed.
-- +goose StatementBegin
DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM app.drawdowns WHERE state = 'ACTIVATED') THEN
		RAISE EXCEPTION 'cannot migrate legacy activated drawdowns without immutable lifecycle evidence; reconcile or recreate pre-production fixtures first';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE app.drawdowns
    ADD COLUMN invoice_document_hash text,
    ADD COLUMN due_date date,
    ADD COLUMN collection_at timestamptz,
    ADD COLUMN grace_hours integer,
    ADD COLUMN terms_version text,
    ADD COLUMN agreement_hash text,
    ADD COLUMN release_actor_id uuid REFERENCES app.users(id),
    ADD COLUMN delivery_method text,
    ADD COLUMN release_notes text,
    ADD COLUMN release_evidence_reference text,
    ADD COLUMN released_at timestamptz,
    ADD COLUMN receipt_state text,
    ADD COLUMN receipt_actor_id uuid REFERENCES app.users(id),
    ADD COLUMN receipt_issue_reason text,
    ADD COLUMN receipt_dispute_id uuid,
    ADD COLUMN receipt_at timestamptz;

UPDATE app.drawdowns d
SET due_date = (d.created_at AT TIME ZONE 'Africa/Lagos')::date + 30,
    collection_at = d.created_at + INTERVAL '30 days' + make_interval(hours => l.default_grace_hours),
    grace_hours = l.default_grace_hours,
    terms_version = l.terms_version,
    agreement_hash = ''
FROM app.trade_lines l
WHERE l.id = d.trade_line_id;

-- Pending legacy drawdowns never captured all immutable Wave 1 terms. Cancel
-- them safely instead of inventing acceptance evidence, then rebuild capacity.
UPDATE app.drawdowns SET state = 'CANCELLED' WHERE state <> 'ACTIVATED';
UPDATE app.drawdown_reservations r
SET state = 'RELEASED'
WHERE EXISTS (SELECT 1 FROM app.drawdowns d WHERE d.id = r.drawdown_id AND d.state = 'CANCELLED');
UPDATE app.trade_lines l
SET reserved_pending_kobo = COALESCE(v.reserved_kobo, 0),
    available_limit_kobo = GREATEST(0, l.approved_limit_kobo - l.current_exposure_kobo - COALESCE(v.reserved_kobo, 0))
FROM (
    SELECT line.id AS trade_line_id, SUM(r.amount_kobo) FILTER (WHERE r.state IN ('PENDING','CONFIRMED')) AS reserved_kobo
    FROM app.trade_lines line
    LEFT JOIN app.drawdown_reservations r ON r.trade_line_id = line.id
    GROUP BY line.id
) v
WHERE l.id = v.trade_line_id;

ALTER TABLE app.drawdowns
    ALTER COLUMN due_date SET NOT NULL,
    ALTER COLUMN collection_at SET NOT NULL,
    ALTER COLUMN grace_hours SET NOT NULL,
    ALTER COLUMN terms_version SET NOT NULL,
    ALTER COLUMN agreement_hash SET NOT NULL,
    ADD CONSTRAINT drawdowns_grace_hours_check CHECK (grace_hours BETWEEN 0 AND 720),
	ADD CONSTRAINT drawdowns_agreement_hash_check CHECK (state = 'CANCELLED' OR agreement_hash ~ '^[0-9a-f]{64}$'),
    ADD CONSTRAINT drawdowns_collection_after_due_check CHECK (collection_at >= due_date::timestamptz),
    ADD CONSTRAINT drawdowns_receipt_state_check CHECK (receipt_state IS NULL OR receipt_state IN ('no_issue','issue_reported')),
    ADD CONSTRAINT drawdowns_issue_reason_check CHECK (receipt_state <> 'issue_reported' OR (NULLIF(btrim(receipt_issue_reason),'') IS NOT NULL AND receipt_dispute_id IS NOT NULL)),
    ADD CONSTRAINT drawdowns_release_evidence_check CHECK (state NOT IN ('GOODS_RELEASED','RECEIPT_ISSUE_REPORTED','ACTIVATED') OR (release_actor_id IS NOT NULL AND NULLIF(btrim(delivery_method),'') IS NOT NULL AND released_at IS NOT NULL)),
    ADD CONSTRAINT drawdowns_activation_evidence_check CHECK (state <> 'ACTIVATED' OR (obligation_id IS NOT NULL AND receipt_state = 'no_issue' AND receipt_actor_id IS NOT NULL AND receipt_at IS NOT NULL));

ALTER TABLE app.drawdowns DROP CONSTRAINT drawdowns_state_check;
ALTER TABLE app.drawdowns
    ADD CONSTRAINT drawdowns_state_check CHECK (state IN ('PENDING_BUYER_CONFIRMATION','BUYER_CONFIRMED','GOODS_RELEASED','RECEIPT_ISSUE_REPORTED','ACTIVATED','CANCELLED','EXPIRED'));

ALTER TABLE app.drawdown_reservations DROP CONSTRAINT drawdown_reservations_state_check;
ALTER TABLE app.drawdown_reservations
    ADD CONSTRAINT drawdown_reservations_state_check CHECK (state IN ('PENDING','CONFIRMED','RELEASED_TO_SUPPLIER','CONVERTED','EXPIRED','RELEASED'));

CREATE UNIQUE INDEX drawdowns_one_obligation_idx
    ON app.drawdowns(obligation_id)
    WHERE obligation_id IS NOT NULL;

CREATE TABLE app.drawdown_receipt_disputes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    drawdown_id uuid NOT NULL UNIQUE REFERENCES app.drawdowns(id) ON DELETE CASCADE,
    supplier_organization_id uuid NOT NULL REFERENCES app.organizations(id),
    buyer_user_id uuid NOT NULL REFERENCES app.users(id),
    state text NOT NULL CHECK (state IN ('OPEN','RESOLVED','CANCELLED')),
    reason text NOT NULL CHECK (NULLIF(btrim(reason),'') IS NOT NULL),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE app.drawdowns
    ADD CONSTRAINT drawdowns_receipt_dispute_fk FOREIGN KEY (receipt_dispute_id) REFERENCES app.drawdown_receipt_disputes(id) DEFERRABLE INITIALLY DEFERRED;

-- +goose Down
DROP INDEX IF EXISTS app.drawdowns_one_obligation_idx;
ALTER TABLE app.drawdowns DROP CONSTRAINT IF EXISTS drawdowns_receipt_dispute_fk;
DROP TABLE IF EXISTS app.drawdown_receipt_disputes;
UPDATE app.drawdowns SET state = 'BUYER_CONFIRMED' WHERE state IN ('GOODS_RELEASED','RECEIPT_ISSUE_REPORTED');
UPDATE app.drawdowns SET state = 'CANCELLED' WHERE state = 'EXPIRED';
UPDATE app.drawdown_reservations SET state = 'CONFIRMED' WHERE state = 'RELEASED_TO_SUPPLIER';
ALTER TABLE app.drawdown_reservations DROP CONSTRAINT IF EXISTS drawdown_reservations_state_check;
ALTER TABLE app.drawdown_reservations
    ADD CONSTRAINT drawdown_reservations_state_check CHECK (state IN ('PENDING','CONFIRMED','CONVERTED','EXPIRED','RELEASED'));
ALTER TABLE app.drawdowns
    DROP CONSTRAINT IF EXISTS drawdowns_state_check,
	DROP CONSTRAINT IF EXISTS drawdowns_grace_hours_check,
	DROP CONSTRAINT IF EXISTS drawdowns_agreement_hash_check,
    DROP CONSTRAINT IF EXISTS drawdowns_collection_after_due_check,
    DROP CONSTRAINT IF EXISTS drawdowns_receipt_state_check,
    DROP CONSTRAINT IF EXISTS drawdowns_issue_reason_check,
    DROP CONSTRAINT IF EXISTS drawdowns_release_evidence_check,
    DROP CONSTRAINT IF EXISTS drawdowns_activation_evidence_check;
ALTER TABLE app.drawdowns
    ADD CONSTRAINT drawdowns_state_check CHECK (state IN ('PENDING_BUYER_CONFIRMATION','BUYER_CONFIRMED','ACTIVATED','CANCELLED'));
ALTER TABLE app.drawdowns
    DROP COLUMN IF EXISTS invoice_document_hash,
    DROP COLUMN IF EXISTS due_date,
    DROP COLUMN IF EXISTS collection_at,
    DROP COLUMN IF EXISTS grace_hours,
    DROP COLUMN IF EXISTS terms_version,
    DROP COLUMN IF EXISTS agreement_hash,
    DROP COLUMN IF EXISTS release_actor_id,
    DROP COLUMN IF EXISTS delivery_method,
    DROP COLUMN IF EXISTS release_notes,
    DROP COLUMN IF EXISTS release_evidence_reference,
    DROP COLUMN IF EXISTS released_at,
    DROP COLUMN IF EXISTS receipt_state,
    DROP COLUMN IF EXISTS receipt_actor_id,
    DROP COLUMN IF EXISTS receipt_issue_reason,
    DROP COLUMN IF EXISTS receipt_dispute_id,
    DROP COLUMN IF EXISTS receipt_at;
