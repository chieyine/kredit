-- +goose Up
ALTER TABLE app.fee_debits ADD COLUMN review_required boolean NOT NULL DEFAULT false;
-- +goose StatementBegin
CREATE FUNCTION app.fee_notice_scope(account_name text,external_reference text,mandate text) RETURNS TABLE(organization_id uuid,authorization_id uuid,debit_id uuid) LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT a.organization_id,a.id,d.id FROM app.fee_authorizations a LEFT JOIN app.fee_debits d ON d.authorization_id=a.id AND 'fee-debit-'||d.id::text=external_reference
 WHERE a.provider=account_name AND ((external_reference LIKE 'fee-debit-%' AND d.id IS NOT NULL) OR (external_reference NOT LIKE 'fee-debit-%' AND a.mandate_reference=mandate AND mandate<>''))
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.fee_notice_scope(text,text,text) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN GRANT EXECUTE ON FUNCTION app.fee_notice_scope(text,text,text) TO kredit_worker;END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT EXECUTE ON FUNCTION app.fee_notice_scope(text,text,text) TO kredit_app;END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
DROP FUNCTION app.fee_notice_scope(text,text,text);
