-- +goose Up
-- Buyer acknowledgement is independent evidence. A worker may consume it,
-- but must never manufacture or rewrite it using a buyer transaction context.
DROP POLICY collection_notice_ack_buyer ON app.collection_notice_acknowledgements;
CREATE POLICY collection_notice_ack_buyer_read
ON app.collection_notice_acknowledgements FOR SELECT
USING (current_user = 'kredit_app' AND buyer_user_id = app.current_user_id());
CREATE POLICY collection_notice_ack_buyer_insert
ON app.collection_notice_acknowledgements FOR INSERT
WITH CHECK (current_user = 'kredit_app' AND buyer_user_id = app.current_user_id());

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_worker') THEN
        REVOKE INSERT, UPDATE, DELETE ON app.collection_notice_acknowledgements FROM kredit_worker;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        REVOKE UPDATE, DELETE ON app.collection_notice_acknowledgements FROM kredit_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- Preserve the evidence boundary during rollback. Removing this boundary is
-- never necessary for older application versions, which only insert/read it.
SELECT 1;
