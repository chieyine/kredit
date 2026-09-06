#!/usr/bin/env python3
from pathlib import Path

root = Path(__file__).resolve().parents[1]
store = root / 'internal/reports/store.go'
text = store.read_text(encoding='utf-8')
old = 'func (s *Store) Track(name, subjectID, purpose string, metadata map[string]string) (AnalyticsEvent, error) {\n'
new = '''func (s *Store) Track(name, subjectID, purpose string, metadata map[string]string) (AnalyticsEvent, error) {\n\treturn s.TrackContext(context.Background(), name, subjectID, purpose, metadata)\n}\n\nfunc (s *Store) TrackContext(ctx context.Context, name, subjectID, purpose string, metadata map[string]string) (AnalyticsEvent, error) {\n'''
if old not in text:
    raise SystemExit('Track signature not found')
text = text.replace(old, new, 1)
needle = 's.pool.QueryRow(context.Background(), `INSERT INTO app.analytics_events'
if needle not in text:
    raise SystemExit('analytics QueryRow background context not found')
text = text.replace(needle, 's.pool.QueryRow(ctx, `INSERT INTO app.analytics_events', 1)
old = 'func (s *Store) ListAnalytics() []AnalyticsEvent {\n'
new = '''func (s *Store) ListAnalytics() []AnalyticsEvent {\n\treturn s.ListAnalyticsContext(context.Background())\n}\n\nfunc (s *Store) ListAnalyticsContext(ctx context.Context) []AnalyticsEvent {\n'''
if old not in text:
    raise SystemExit('ListAnalytics signature not found')
text = text.replace(old, new, 1)
needle = 's.pool.Query(context.Background(), `SELECT id::text,name,subject_id_hash,purpose,occurred_at,recorded_at,schema_version,deduplication_key,COALESCE(organization_id_hash,\'\'),source,metadata FROM app.analytics_events ORDER BY occurred_at`)'
if needle not in text:
    raise SystemExit('analytics Query background context not found')
text = text.replace(needle, 's.pool.Query(ctx, `SELECT id::text,name,subject_id_hash,purpose,occurred_at,recorded_at,schema_version,deduplication_key,COALESCE(organization_id_hash,\'\'),source,metadata FROM app.analytics_events ORDER BY occurred_at`)', 1)
store.write_text(text, encoding='utf-8')

for path in (root / 'internal/web').glob('*.go'):
    if path.name.endswith('_test.go'):
        continue
    original = path.read_text(encoding='utf-8')
    updated = original.replace('s.runtime.Reports.Track(', 's.runtime.Reports.TrackContext(r.Context(), ')
    if updated != original:
        path.write_text(updated, encoding='utf-8')

# Lock the request-aware analytics boundary into the context audit.
audit = root / 'scripts/phase6-context-audit.py'
a = audit.read_text(encoding='utf-8')
insert = '''\n# Analytics persistence reached from HTTP must use the request-aware API.\nfor path in sorted(WEB.glob("*.go")):\n    if path.name.endswith("_test.go"):\n        continue\n    body = path.read_text(encoding="utf-8")\n    if "Reports.Track(" in body:\n        issues.append(f"{path.relative_to(ROOT)}: legacy analytics Track bypasses request context")\n\n'''
a = a.replace('\nif issues:\n', insert + 'if issues:\n', 1)
audit.write_text(a, encoding='utf-8')
