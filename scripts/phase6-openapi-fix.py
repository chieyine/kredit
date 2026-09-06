#!/usr/bin/env python3
"""Repair legacy OpenAPI YAML flow descriptions that a real parser rejects.

The previous structural grep never parsed YAML, so comma-containing unquoted values
inside one-line flow mappings survived unnoticed. This script only rewrites the
simple one-property response maps whose sole key is `description`; it does not alter
paths, methods, schemas, operationIds, parameters or response codes.
"""
from __future__ import annotations

from pathlib import Path
import re

path = Path('api/openapi.yaml')
lines = path.read_text(encoding='utf-8').splitlines()

# Match only a complete one-line response flow mapping whose body contains exactly
# one description key. Nested/multi-key flow mappings are deliberately ignored.
pat = re.compile(r"^(?P<prefix>\s*'[^']+'\s*:\s*\{\s*description\s*:\s*)(?P<value>[^{}]*?)(?P<suffix>\s*\}\s*)$")
changed = 0
for i, line in enumerate(lines):
    m = pat.match(line)
    if not m:
        continue
    value = m.group('value').strip()
    if not value or value.startswith(("'", '"')):
        continue
    # Quote as JSON/YAML double-quoted scalar using Python's JSON encoder to escape
    # backslashes and quotes safely.
    import json
    lines[i] = f"{m.group('prefix')}{json.dumps(value, ensure_ascii=False)}{m.group('suffix')}"
    changed += 1

if changed == 0:
    raise SystemExit('no legacy response descriptions required repair')
path.write_text('\n'.join(lines) + '\n', encoding='utf-8')
print(f'quoted_response_descriptions={changed}')
