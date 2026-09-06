#!/usr/bin/env python3
"""One-time, hash-bound source patch transport. Data is JSON, never executable code."""
import base64
import hashlib
import json
import lzma
from pathlib import Path

root = Path.cwd().resolve()
encoded = ''.join((root / f'scripts/audit_patch/part-{i}.b64').read_text(encoding='ascii') for i in range(1, 7))
assert len(encoded) == 70176, 'Incomplete source transport'
compressed = base64.b64decode(encoded, validate=True)
assert hashlib.sha256(compressed).hexdigest() == '6db08c8df649c9f40cdf3eb7d927d2a26d7cbf432885d75c99f890676a6cdc1b', 'Source transport hash mismatch'
decoder = lzma.LZMADecompressor(memlimit=256 << 20)
payload = decoder.decompress(compressed, max_length=1 << 20)
assert decoder.eof and not decoder.unused_data, 'Invalid or oversized transport'
operations = json.loads(payload)
assert isinstance(operations, list) and len(operations) == 59
prepared = []
seen = set()
for operation in operations:
    name = operation['path']
    path = Path(name)
    assert name not in seen and not path.is_absolute() and '..' not in path.parts
    assert name in ('CHANGELOG.md', 'IMPLEMENTATION_STATUS.md') or path.parts[0] in ('web', 'internal', 'api', 'scripts', 'docs')
    seen.add(name)
    target = root / path
    assert target.resolve().is_relative_to(root) and not target.is_symlink()
    before = target.read_bytes() if target.exists() else None
    assert (hashlib.sha256(before).hexdigest() if before is not None else None) == operation['before'], f'Base source changed: {name}'
    if operation['after'] is None:
        after = None
    elif 'content' in operation:
        after = operation['content'].encode('utf-8')
    else:
        assert before is not None
        value = before.decode('utf-8')
        previous = len(value) + 1
        for start, end, replacement in reversed(operation['changes']):
            assert isinstance(start, int) and isinstance(end, int) and 0 <= start <= end < previous
            value = value[:start] + replacement + value[end:]
            previous = start + 1
        after = value.encode('utf-8')
    assert (hashlib.sha256(after).hexdigest() if after is not None else None) == operation['after'], f'Result hash mismatch: {name}'
    prepared.append((name, target, after))
# Validate every transformation before modifying any source file.
for name, target, after in prepared:
    if after is None:
        target.unlink()
    else:
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(after)
(root / '.tmp').mkdir(exist_ok=True)
(root / '.tmp/audit-changed-paths.txt').write_text(''.join(name + '\n' for name, _, _ in prepared), encoding='utf-8')
print(f'Applied {len(prepared)} source transformations; every input and result hash verified.')
