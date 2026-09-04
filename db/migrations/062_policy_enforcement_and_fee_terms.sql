-- +goose Up
ALTER TABLE app.credit_requests ADD COLUMN fee_terms jsonb;
ALTER TABLE app.drawdowns ADD COLUMN fee_terms jsonb;
-- +goose StatementBegin
CREATE FUNCTION app.guard_offer_policy() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
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
CREATE TRIGGER credit_offer_policy BEFORE INSERT OR UPDATE ON app.credit_requests FOR EACH ROW EXECUTE FUNCTION app.guard_offer_policy();
CREATE TRIGGER drawdown_offer_policy BEFORE INSERT OR UPDATE ON app.drawdowns FOR EACH ROW EXECUTE FUNCTION app.guard_offer_policy();
-- +goose StatementBegin
CREATE FUNCTION app.guard_business_count_policy() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE p jsonb; cap bigint; existing bigint; industry_allowed boolean;
BEGIN
 IF TG_TABLE_NAME='organizations' AND EXISTS(SELECT 1 FROM app.organizations WHERE id=NEW.id) THEN RETURN NEW; END IF;
 IF TG_TABLE_NAME='businesses' AND EXISTS(SELECT 1 FROM app.businesses WHERE id=NEW.id) THEN RETURN NEW; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended(TG_TABLE_NAME,613));
 p:=app.business_policy();
 IF trim(COALESCE(p->>'allowed_industries',''))<>'' THEN
  SELECT EXISTS(SELECT 1 FROM unnest(string_to_array(p->>'allowed_industries',',')) i WHERE lower(trim(i))=lower(trim(NEW.industry))) INTO industry_allowed;
  IF NOT industry_allowed THEN RAISE EXCEPTION 'industry is outside current business policy'; END IF;
 END IF;
 IF TG_TABLE_NAME='organizations' THEN cap:=COALESCE((p->>'max_suppliers')::bigint,0);SELECT count(*) INTO existing FROM app.organizations;
 ELSE cap:=COALESCE((p->>'max_buyers')::bigint,0);SELECT count(*) INTO existing FROM app.businesses; END IF;
 IF cap>0 AND existing>=cap THEN RAISE EXCEPTION 'business count limit reached'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER supplier_count_policy BEFORE INSERT ON app.organizations FOR EACH ROW EXECUTE FUNCTION app.guard_business_count_policy();
CREATE TRIGGER buyer_count_policy BEFORE INSERT ON app.businesses FOR EACH ROW EXECUTE FUNCTION app.guard_business_count_policy();
-- +goose StatementBegin
CREATE FUNCTION app.guard_exposure_policy() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
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
CREATE TRIGGER obligation_exposure_policy BEFORE INSERT ON app.obligations FOR EACH ROW EXECUTE FUNCTION app.guard_exposure_policy();
CREATE TRIGGER reservation_exposure_policy BEFORE INSERT ON app.drawdown_reservations FOR EACH ROW EXECUTE FUNCTION app.guard_exposure_policy();
-- +goose StatementBegin
CREATE FUNCTION app.guard_collection_policy() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.state='PROCESSING' AND NOT EXISTS(SELECT 1 FROM app.collection_reservations WHERE id=NEW.id) AND COALESCE((app.business_policy()->>'collections_enabled')::boolean,true)=false THEN RAISE EXCEPTION 'new collections are paused by business policy'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER collection_business_policy BEFORE INSERT ON app.collection_reservations FOR EACH ROW EXECUTE FUNCTION app.guard_collection_policy();
-- +goose StatementBegin
CREATE FUNCTION app.guard_payment_claim_policy() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM app.payment_claims WHERE id=NEW.id) AND COALESCE((app.business_policy()->>'payment_claims')::boolean,true)=false THEN RAISE EXCEPTION 'new payment claims are paused by business policy'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER payment_claim_business_policy BEFORE INSERT ON app.payment_claims FOR EACH ROW EXECUTE FUNCTION app.guard_payment_claim_policy();
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM app.credit_requests WHERE fee_terms->>'base_bps'<>'50' OR fee_terms->>'collection_bps'<>'50') OR EXISTS(SELECT 1 FROM app.drawdowns WHERE fee_terms->>'base_bps'<>'50' OR fee_terms->>'collection_bps'<>'50') THEN RAISE EXCEPTION 'custom fee terms must be retained; use a forward migration'; END IF;
END $$;
-- +goose StatementEnd
DROP TRIGGER payment_claim_business_policy ON app.payment_claims;
DROP FUNCTION app.guard_payment_claim_policy();
DROP TRIGGER collection_business_policy ON app.collection_reservations;
DROP FUNCTION app.guard_collection_policy();
DROP TRIGGER obligation_exposure_policy ON app.obligations;
DROP TRIGGER reservation_exposure_policy ON app.drawdown_reservations;
DROP FUNCTION app.guard_exposure_policy();
DROP TRIGGER supplier_count_policy ON app.organizations;
DROP TRIGGER buyer_count_policy ON app.businesses;
DROP FUNCTION app.guard_business_count_policy();
DROP TRIGGER credit_offer_policy ON app.credit_requests;
DROP TRIGGER drawdown_offer_policy ON app.drawdowns;
DROP FUNCTION app.guard_offer_policy();
ALTER TABLE app.credit_requests DROP COLUMN fee_terms;
ALTER TABLE app.drawdowns DROP COLUMN fee_terms;
