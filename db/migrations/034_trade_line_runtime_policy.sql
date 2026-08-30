-- +goose Up
ALTER TABLE app.trade_lines ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS trade_line_runtime_access ON app.trade_lines;
CREATE POLICY trade_line_runtime_access ON app.trade_lines USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
ALTER TABLE app.drawdowns ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS drawdown_runtime_access ON app.drawdowns;
CREATE POLICY drawdown_runtime_access ON app.drawdowns USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
ALTER TABLE app.drawdown_reservations ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS drawdown_reservation_runtime_access ON app.drawdown_reservations;
CREATE POLICY drawdown_reservation_runtime_access ON app.drawdown_reservations USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
-- +goose Down
DROP POLICY IF EXISTS drawdown_reservation_runtime_access ON app.drawdown_reservations;
DROP POLICY IF EXISTS drawdown_runtime_access ON app.drawdowns;
DROP POLICY IF EXISTS trade_line_runtime_access ON app.trade_lines;
