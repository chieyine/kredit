-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_offer_policy() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE p jsonb; cap bigint;
BEGIN
 IF TG_OP='UPDATE' THEN
  IF NEW.fee_terms IS DISTINCT FROM OLD.fee_terms THEN RAISE EXCEPTION 'offer fee terms are immutable'; END IF;
  IF NEW.principal_kobo=OLD.principal_kobo THEN RETURN NEW; END IF;
  IF TG_TABLE_NAME='credit_requests' AND OLD.state<>'DRAFT' THEN RAISE EXCEPTION 'accepted principal cannot be amended'; END IF;
 ELSE
  -- Upsert replays retain the original terms and are not new offers.
  IF TG_TABLE_NAME='credit_requests' AND EXISTS(SELECT 1 FROM app.credit_requests WHERE id=NEW.id) THEN RETURN NEW; END IF;
  IF TG_TABLE_NAME='drawdowns' AND EXISTS(SELECT 1 FROM app.drawdowns WHERE id=NEW.id) THEN RETURN NEW; END IF;
 END IF;
 IF NEW.fee_terms IS NOT NULL AND NEW.fee_terms<>'null'::jsonb AND
  (jsonb_typeof(NEW.fee_terms)<>'object' OR NOT(NEW.fee_terms ?& ARRAY['base_bps','collection_bps','policy_revision']) OR
   (NEW.fee_terms->>'base_bps')::bigint NOT BETWEEN 0 AND 1000 OR (NEW.fee_terms->>'collection_bps')::bigint NOT BETWEEN 0 AND 1000) THEN RAISE EXCEPTION 'invalid fee terms'; END IF;
 -- A drawdown already contains the immutable offer. Creating its normalized
 -- credit request at activation is recognition of that offer, not a new quote.
 IF TG_TABLE_NAME='credit_requests' AND TG_OP='INSERT' AND EXISTS(SELECT 1 FROM app.drawdowns WHERE id=NEW.id) THEN
  IF NOT EXISTS(SELECT 1 FROM app.drawdowns d JOIN app.trade_lines l ON l.id=d.trade_line_id WHERE d.id=NEW.id AND d.principal_kobo=NEW.principal_kobo AND COALESCE(d.fee_terms,'null'::jsonb)=COALESCE(NEW.fee_terms,'null'::jsonb) AND l.supplier_organization_id=NEW.supplier_organization_id AND l.buyer_user_id=NEW.buyer_user_id AND l.buyer_business_id=NEW.buyer_business_id) THEN RAISE EXCEPTION 'drawdown recognition must preserve the recorded offer'; END IF;
  RETURN NEW;
 END IF;
 p:=app.business_policy();cap:=COALESCE((p->>'max_principal_kobo')::bigint,0);
 IF cap>0 AND NEW.principal_kobo>cap THEN RAISE EXCEPTION 'principal exceeds current business policy'; END IF;
 IF TG_TABLE_NAME='drawdowns' AND TG_OP='INSERT' THEN
  PERFORM pg_advisory_xact_lock(hashtextextended(NEW.trade_line_id::text,612));
  cap:=COALESCE((p->>'max_drawdowns_per_day')::bigint,0);
  IF cap>0 AND (SELECT count(*) FROM app.drawdowns WHERE trade_line_id=NEW.trade_line_id AND (created_at AT TIME ZONE 'Africa/Lagos')::date=(now() AT TIME ZONE 'Africa/Lagos')::date)>=cap THEN RAISE EXCEPTION 'daily drawdown limit reached'; END IF;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_exposure_policy() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE cap bigint; buyer uuid; exclude_drawdown uuid; added bigint; exposure numeric;
BEGIN
 cap:=COALESCE((app.business_policy()->>'max_exposure_kobo')::bigint,0);
 IF TG_TABLE_NAME='goods_releases' THEN
  IF EXISTS(SELECT 1 FROM app.goods_releases WHERE id=NEW.id) THEN RETURN NEW; END IF;
  -- A trade-line drawdown was already reserved before goods release. Its
  -- normalized release row is written only when receipt activates the debt.
  IF EXISTS(SELECT 1 FROM app.drawdown_reservations WHERE drawdown_id=NEW.credit_request_id AND state IN('RELEASED_TO_SUPPLIER','CONVERTED')) THEN RETURN NEW; END IF;
  SELECT buyer_business_id,principal_kobo INTO buyer,added FROM app.credit_requests WHERE id=NEW.credit_request_id;
  exclude_drawdown:=NEW.credit_request_id;
 ELSIF TG_TABLE_NAME='obligations' THEN
  IF EXISTS(SELECT 1 FROM app.obligations WHERE id=NEW.id) THEN RETURN NEW; END IF;
  -- Never suppress recognition of an existing physical-goods commitment.
  IF EXISTS(SELECT 1 FROM app.goods_releases WHERE credit_request_id=NEW.credit_request_id) THEN
   PERFORM pg_advisory_xact_lock(hashtextextended(NEW.buyer_business_id::text,614));
   RETURN NEW;
  END IF;
  buyer:=NEW.buyer_business_id;exclude_drawdown:=NEW.credit_request_id;added:=NEW.outstanding_kobo;
 ELSE
  IF EXISTS(SELECT 1 FROM app.drawdown_reservations WHERE id=NEW.id) OR NEW.state NOT IN('PENDING','CONFIRMED','RELEASED_TO_SUPPLIER') THEN RETURN NEW; END IF;
  SELECT buyer_business_id INTO buyer FROM app.trade_lines WHERE id=NEW.trade_line_id;
  exclude_drawdown:=NEW.drawdown_id;added:=NEW.amount_kobo;
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended(buyer::text,614));
 IF cap<=0 THEN RETURN NEW; END IF;
 SELECT COALESCE(sum(outstanding_kobo),0) INTO exposure FROM app.obligations WHERE buyer_business_id=buyer AND outstanding_kobo>0;
 SELECT exposure+COALESCE(sum(r.amount_kobo),0) INTO exposure FROM app.drawdown_reservations r JOIN app.trade_lines l ON l.id=r.trade_line_id WHERE l.buyer_business_id=buyer AND r.drawdown_id<>exclude_drawdown AND r.state IN('PENDING','CONFIRMED','RELEASED_TO_SUPPLIER');
 -- Released ordinary credit also consumes headroom before receipt recognition.
 -- Do not double-count trade-line reservations or already activated obligations.
 SELECT exposure+COALESCE(sum(c.principal_kobo),0) INTO exposure FROM app.credit_requests c
 WHERE c.buyer_business_id=buyer AND c.id<>exclude_drawdown AND EXISTS(SELECT 1 FROM app.goods_releases g WHERE g.credit_request_id=c.id)
 AND NOT EXISTS(SELECT 1 FROM app.obligations o WHERE o.credit_request_id=c.id)
 AND NOT EXISTS(SELECT 1 FROM app.drawdowns d WHERE d.id=c.id);
 IF exposure+added>cap THEN RAISE EXCEPTION 'buyer exposure exceeds current business policy'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER goods_release_exposure_policy BEFORE INSERT ON app.goods_releases FOR EACH ROW EXECUTE FUNCTION app.guard_exposure_policy();
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM app.business_policy_changes) OR COALESCE((app.business_policy()->>'max_exposure_kobo')::bigint,0)>0 OR COALESCE((app.business_policy()->>'max_principal_kobo')::bigint,0)>0 THEN RAISE EXCEPTION 'active commitment policies require a forward migration'; END IF;
END $$;
-- +goose StatementEnd
DROP TRIGGER goods_release_exposure_policy ON app.goods_releases;
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_offer_policy() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE p jsonb; cap bigint;
BEGIN
 IF TG_OP='UPDATE' THEN
  IF NEW.fee_terms IS DISTINCT FROM OLD.fee_terms THEN RAISE EXCEPTION 'offer fee terms are immutable'; END IF;
  IF NEW.principal_kobo=OLD.principal_kobo THEN RETURN NEW; END IF;
  IF TG_TABLE_NAME='credit_requests' AND OLD.state<>'DRAFT' THEN RAISE EXCEPTION 'accepted principal cannot be amended'; END IF;
 ELSE
  -- Upsert replays retain the original terms and are not new offers.
  IF TG_TABLE_NAME='credit_requests' AND EXISTS(SELECT 1 FROM app.credit_requests WHERE id=NEW.id) THEN RETURN NEW; END IF;
  IF TG_TABLE_NAME='drawdowns' AND EXISTS(SELECT 1 FROM app.drawdowns WHERE id=NEW.id) THEN RETURN NEW; END IF;
 END IF;
 IF NEW.fee_terms IS NOT NULL AND NEW.fee_terms<>'null'::jsonb AND
  (jsonb_typeof(NEW.fee_terms)<>'object' OR NOT(NEW.fee_terms ?& ARRAY['base_bps','collection_bps','policy_revision']) OR
   (NEW.fee_terms->>'base_bps')::bigint NOT BETWEEN 0 AND 1000 OR (NEW.fee_terms->>'collection_bps')::bigint NOT BETWEEN 0 AND 1000) THEN RAISE EXCEPTION 'invalid fee terms'; END IF;
 p:=app.business_policy();cap:=COALESCE((p->>'max_principal_kobo')::bigint,0);
 IF cap>0 AND NEW.principal_kobo>cap THEN RAISE EXCEPTION 'principal exceeds current business policy'; END IF;
 IF TG_TABLE_NAME='drawdowns' AND TG_OP='INSERT' THEN
  PERFORM pg_advisory_xact_lock(hashtextextended(NEW.trade_line_id::text,612));
  cap:=COALESCE((p->>'max_drawdowns_per_day')::bigint,0);
  IF cap>0 AND (SELECT count(*) FROM app.drawdowns WHERE trade_line_id=NEW.trade_line_id AND (created_at AT TIME ZONE 'Africa/Lagos')::date=(now() AT TIME ZONE 'Africa/Lagos')::date)>=cap THEN RAISE EXCEPTION 'daily drawdown limit reached'; END IF;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_exposure_policy() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE cap bigint; buyer uuid; exclude_drawdown uuid; added bigint; exposure numeric;
BEGIN
 cap:=COALESCE((app.business_policy()->>'max_exposure_kobo')::bigint,0);
 IF cap<=0 THEN RETURN NEW; END IF;
 IF TG_TABLE_NAME='obligations' THEN
  IF EXISTS(SELECT 1 FROM app.obligations WHERE id=NEW.id) THEN RETURN NEW; END IF;
  buyer:=NEW.buyer_business_id;exclude_drawdown:=NEW.credit_request_id;added:=NEW.outstanding_kobo;
 ELSE
  IF EXISTS(SELECT 1 FROM app.drawdown_reservations WHERE id=NEW.id) OR NEW.state NOT IN('PENDING','CONFIRMED','RELEASED_TO_SUPPLIER') THEN RETURN NEW; END IF;
  SELECT buyer_business_id INTO buyer FROM app.trade_lines WHERE id=NEW.trade_line_id;
  exclude_drawdown:=NEW.drawdown_id;added:=NEW.amount_kobo;
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended(buyer::text,614));
 SELECT COALESCE(sum(outstanding_kobo),0) INTO exposure FROM app.obligations WHERE buyer_business_id=buyer AND outstanding_kobo>0;
 SELECT exposure+COALESCE(sum(r.amount_kobo),0) INTO exposure FROM app.drawdown_reservations r JOIN app.trade_lines l ON l.id=r.trade_line_id WHERE l.buyer_business_id=buyer AND r.drawdown_id<>exclude_drawdown AND r.state IN('PENDING','CONFIRMED','RELEASED_TO_SUPPLIER');
 IF exposure+added>cap THEN RAISE EXCEPTION 'buyer exposure exceeds current business policy'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
