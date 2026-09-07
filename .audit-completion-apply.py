"""One-use transport of the locally type-checked audit fixes, never a production action."""
import base64
import hashlib
import json
from pathlib import Path
import subprocess
import zlib

root = Path.cwd().resolve()
encoded = (root / '.audit-completion.b64').read_text().strip()
# Restore the two transport omissions; the cryptographic digest below is authoritative.
encoded = encoded.replace('3myi8L4', '3myi8iL4').replace('Y7VcrPciZO', 'Y7VcrPJlPciZO')
raw = zlib.decompress(base64.b64decode(encoded, validate=True))
assert hashlib.sha256(raw).hexdigest() == 'da6b60d66a79fbb77718a205843c01c016d9666356682f0af4263153cfcb0f67', 'Transport digest mismatch; no source was changed'
changes = json.loads(raw)
assert len(changes) == 23
prepared = []
for change in changes:
    name = change['path']
    assert name == 'internal/notifications/store_test.go' or name == 'web/playwright.config.ts' or name.startswith(('web/src/', 'web/tests/'))
    path = root / name
    assert path.resolve().is_relative_to(root) and not path.is_symlink()
    if change['before'] is None:
        assert not path.exists(), f'New path already exists: {name}'
        original = ''
    else:
        data = path.read_bytes()
        assert hashlib.sha256(data).hexdigest() == change['before'], f'Source changed since review: {name}'
        original = data.decode('utf-8')
    result = original
    for start, end, replacement in reversed(change['edits']):
        assert 0 <= start <= end <= len(original)
        result = result[:start] + replacement + result[end:]
    data = result.encode('utf-8')
    assert hashlib.sha256(data).hexdigest() == change['after'], f'Patch result mismatch: {name}'
    prepared.append((path, data))
# Only write after validating the entire batch against the reviewed source.
for path, data in prepared:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(data)
    subprocess.run(['git', 'add', '--', str(path.relative_to(root))], check=True)
for name in ['.audit-completion.b64', '.audit-completion-apply.py', '.github/workflows/audit-completion-apply.yml', '.github/workflows/audit-source-snapshot.yml']:
    subprocess.run(['git', 'rm', '--', name], check=True)
print(f'Applied and verified {len(prepared)} source files; temporary transport removed.')
