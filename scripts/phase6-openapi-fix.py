#!/usr/bin/env python3
"""Repair legacy OpenAPI YAML constructs rejected by a real OpenAPI 3.1 parser.

The previous structural grep never parsed YAML, so comma-containing unquoted values
inside one-line flow mappings survived unnoticed. The contract also retained two
OpenAPI 3.0-style `nullable: true` fields even though the document declares 3.1.0.
This script performs only those narrow, deterministic repairs; it does not alter
paths, methods, operationIds, parameters, response codes, or business semantics.
"""
from __future__ import annotations

from pathlib import Path
import json
import re

path = Path('api/openapi.yaml')
text = path.read_text(encoding='utf-8')
lines = text.splitlines()

# Match only a complete one-line response flow mapping whose body contains exactly
# one description key. Nested/multi-key flow mappings are deliberately ignored.
pat = re.compile(r"^(?P<prefix>\s*'[^']+'\s*:\s*\{\s*description\s*:\s*)(?P<value>[^{}]*?)(?P<suffix>\s*\}\s*)$")
quoted = 0
for i, line in enumerate(lines):
    m = pat.match(line)
    if not m:
        continue
    value = m.group('value').strip()
    if not value or value.startswith(("'", '"')):
        continue
    lines[i] = f"{m.group('prefix')}{json.dumps(value, ensure_ascii=False)}{m.group('suffix')}"
    quoted += 1

text = '\n'.join(lines) + '\n'

# OpenAPI 3.1 uses JSON Schema null unions rather than the OpenAPI 3.0 `nullable`
# keyword. Keep the same API meaning while making the schema structurally valid.
nullable_replacements = {
    "verified_since: {type: string, format: date-time, nullable: true, description: Omitted until an authoritative identity-verification date is available; agreement activation is not verification.}":
        "verified_since: {type: [string, 'null'], format: date-time, description: Omitted until an authoritative identity-verification date is available; agreement activation is not verification.}",
    "score: {type: number, nullable: true}":
        "score: {type: [number, 'null']}",
}
nullable_fixed = 0
for old, new in nullable_replacements.items():
    if old in text:
        text = text.replace(old, new)
        nullable_fixed += 1

if quoted == 0 and nullable_fixed == 0:
    raise SystemExit('no legacy OpenAPI constructs required repair')

path.write_text(text, encoding='utf-8')
print(f'quoted_response_descriptions={quoted}')
print(f'openapi31_nullable_repairs={nullable_fixed}')
