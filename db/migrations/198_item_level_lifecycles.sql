-- +goose Up
-- Track 4: Independent item-level order, shipment, partial delivery, invoice, and approved credit-note lifecycles.

CREATE TABLE IF NOT EXISTS app.order_line_items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id uuid NOT NULL REFERENCES app.credit_requests(id) ON DELETE CASCADE,
  sku text NOT NULL DEFAULT '',
  description text NOT NULL,
  unit_price_kobo bigint NOT NULL CHECK(unit_price_kobo >= 0),
  quantity bigint NOT NULL CHECK(quantity > 0),
  fulfilled_quantity bigint NOT NULL DEFAULT 0 CHECK(fulfilled_quantity >= 0 AND fulfilled_quantity <= quantity),
  returned_quantity bigint NOT NULL DEFAULT 0 CHECK(returned_quantity >= 0 AND returned_quantity <= fulfilled_quantity),
  total_kobo bigint NOT NULL CHECK(total_kobo = unit_price_kobo * quantity),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS app.order_shipments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id uuid NOT NULL REFERENCES app.credit_requests(id) ON DELETE CASCADE,
  supplier_organization_id uuid NOT NULL REFERENCES app.organizations(id),
  tracking_reference text NOT NULL DEFAULT '',
  carrier text NOT NULL DEFAULT '',
  dispatched_by uuid NOT NULL REFERENCES app.users(id),
  dispatched_at timestamptz NOT NULL DEFAULT now(),
  status text NOT NULL DEFAULT 'in_transit' CHECK(status IN ('in_transit', 'delivered', 'cancelled'))
);

CREATE TABLE IF NOT EXISTS app.order_shipment_items (
  shipment_id uuid NOT NULL REFERENCES app.order_shipments(id) ON DELETE CASCADE,
  line_item_id uuid NOT NULL REFERENCES app.order_line_items(id) ON DELETE CASCADE,
  quantity bigint NOT NULL CHECK(quantity > 0),
  PRIMARY KEY (shipment_id, line_item_id)
);

CREATE TABLE IF NOT EXISTS app.order_delivery_receipts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  shipment_id uuid NOT NULL REFERENCES app.order_shipments(id) ON DELETE CASCADE,
  order_id uuid NOT NULL REFERENCES app.credit_requests(id) ON DELETE CASCADE,
  received_by uuid NOT NULL REFERENCES app.users(id),
  received_at timestamptz NOT NULL DEFAULT now(),
  condition_notes text NOT NULL DEFAULT '',
  signed_proof_hash text NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS app.order_credit_notes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id uuid NOT NULL REFERENCES app.credit_requests(id) ON DELETE CASCADE,
  obligation_id uuid REFERENCES app.obligations(id),
  supplier_organization_id uuid NOT NULL REFERENCES app.organizations(id),
  amount_kobo bigint NOT NULL CHECK(amount_kobo > 0),
  reason text NOT NULL,
  issued_by uuid NOT NULL REFERENCES app.users(id),
  approved_by uuid REFERENCES app.users(id),
  status text NOT NULL DEFAULT 'draft' CHECK(status IN ('draft', 'approved', 'applied', 'void')),
  created_at timestamptz NOT NULL DEFAULT now(),
  approved_at timestamptz,
  CHECK((status <> 'approved') OR (approved_by IS NOT NULL AND approved_by <> issued_by AND approved_at IS NOT NULL))
);

ALTER TABLE app.order_line_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.order_line_items FORCE ROW LEVEL SECURITY;
ALTER TABLE app.order_shipments ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.order_shipments FORCE ROW LEVEL SECURITY;
ALTER TABLE app.order_shipment_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.order_shipment_items FORCE ROW LEVEL SECURITY;
ALTER TABLE app.order_delivery_receipts ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.order_delivery_receipts FORCE ROW LEVEL SECURITY;
ALTER TABLE app.order_credit_notes ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.order_credit_notes FORCE ROW LEVEL SECURITY;

CREATE POLICY line_items_read ON app.order_line_items FOR SELECT USING (
  EXISTS(SELECT 1 FROM app.credit_requests cr WHERE cr.id = order_line_items.order_id)
);
CREATE POLICY line_items_write ON app.order_line_items FOR ALL USING (
  EXISTS(SELECT 1 FROM app.credit_requests cr WHERE cr.id = order_line_items.order_id AND cr.supplier_organization_id = app.current_organization_id())
);

CREATE POLICY shipments_read ON app.order_shipments FOR SELECT USING (
  supplier_organization_id = app.current_organization_id() OR EXISTS(SELECT 1 FROM app.credit_requests cr WHERE cr.id = order_shipments.order_id AND (cr.buyer_user_id = app.current_user_id() OR app.can_purchase(cr.buyer_business_id, 'read')))
);
CREATE POLICY shipments_write ON app.order_shipments FOR ALL USING (
  supplier_organization_id = app.current_organization_id()
);

CREATE POLICY shipment_items_read ON app.order_shipment_items FOR SELECT USING (
  EXISTS(SELECT 1 FROM app.order_shipments s WHERE s.id = order_shipment_items.shipment_id)
);
CREATE POLICY shipment_items_write ON app.order_shipment_items FOR ALL USING (
  EXISTS(SELECT 1 FROM app.order_shipments s WHERE s.id = order_shipment_items.shipment_id AND s.supplier_organization_id = app.current_organization_id())
);

CREATE POLICY delivery_receipts_read ON app.order_delivery_receipts FOR SELECT USING (
  EXISTS(SELECT 1 FROM app.credit_requests cr WHERE cr.id = order_delivery_receipts.order_id)
);
CREATE POLICY delivery_receipts_write ON app.order_delivery_receipts FOR ALL USING (
  EXISTS(SELECT 1 FROM app.credit_requests cr WHERE cr.id = order_delivery_receipts.order_id AND (cr.buyer_user_id = app.current_user_id() OR app.can_purchase(cr.buyer_business_id, 'receive')))
);

CREATE POLICY credit_notes_read ON app.order_credit_notes FOR SELECT USING (
  supplier_organization_id = app.current_organization_id() OR EXISTS(SELECT 1 FROM app.credit_requests cr WHERE cr.id = order_credit_notes.order_id AND (cr.buyer_user_id = app.current_user_id() OR app.can_purchase(cr.buyer_business_id, 'read')))
);
CREATE POLICY credit_notes_write ON app.order_credit_notes FOR ALL USING (
  supplier_organization_id = app.current_organization_id()
);

GRANT SELECT, INSERT, UPDATE, DELETE ON app.order_line_items, app.order_shipments, app.order_shipment_items, app.order_delivery_receipts, app.order_credit_notes TO kredit_app, kredit_worker;

-- +goose Down
DROP TABLE IF EXISTS app.order_credit_notes;
DROP TABLE IF EXISTS app.order_delivery_receipts;
DROP TABLE IF EXISTS app.order_shipment_items;
DROP TABLE IF EXISTS app.order_shipments;
DROP TABLE IF EXISTS app.order_line_items;
