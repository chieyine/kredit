"""One-use source-bound patch; no main or deployment access."""
import hashlib
import json
from pathlib import Path
import subprocess

root = Path.cwd().resolve()
changes = json.loads((root / '.audit-polish.json').read_text())
allowed = {'.github/workflows/product-audit.yml', 'docs/testing/audit-2026-09-06-implementation.md', 'web/src/routes/app/credit/[id]/+page.svelte', 'web/src/routes/app/credit/new/+page.svelte', 'web/tests/audit-completion.spec.ts', 'web/tests/content-seo.spec.ts', 'web/tests/product-flows.spec.ts'}
assert len(changes) == len(allowed) and {c['path'] for c in changes} == allowed
prepared = []
for change in changes:
    path = root / change['path']
    assert path.resolve().is_relative_to(root) and not path.is_symlink()
    before = path.read_bytes()
    assert hashlib.sha256(before).hexdigest() == change['before'], f'Source changed: {path}'
    original = before.decode('utf-8')
    result = original
    for start, end, replacement in reversed(change['edits']):
        assert 0 <= start <= end <= len(original)
        result = result[:start] + replacement + result[end:]
    after = result.encode('utf-8')
    assert hashlib.sha256(after).hexdigest() == change['after'], f'Result mismatch: {path}'
    prepared.append((path, after))
for path, after in prepared:
    path.write_bytes(after)
    subprocess.run(['git', 'add', '--', str(path.relative_to(root))], check=True)
for path in ['.audit-polish.json', '.audit-polish.py', '.github/workflows/audit-polish.yml']:
    subprocess.run(['git', 'rm', '--', path], check=True)
print('Validated seven reviewed files and removed the one-use transport.')
