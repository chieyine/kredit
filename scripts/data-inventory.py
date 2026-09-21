#!/usr/bin/env python3
"""Render the explicit inventory records; never infer fields from a live schema.

The original TSV remains unchanged. The reviewed additions are maintained in
JSON to avoid repeating unverified control claims across hundreds of fields.
Schema comparison validates coverage only, not legal or security approval.
"""
from __future__ import annotations

import argparse
import csv
import json
from pathlib import Path
import re
import sys
from typing import Any

COLUMNS = 'schema table field classification subject source purpose lawful_basis readers writers protection retention deletion_hold_behavior processor location_transfer owner'.split()
KEYS = COLUMNS[:3]
IDENTIFIER = re.compile(r'[a-z_][a-z0-9_]*\Z')
ROOT = Path(__file__).resolve().parent.parent


def cell(value: Any) -> str:
    if not isinstance(value, str) or not value.strip() or any(ord(c) < 32 or ord(c) == 127 for c in value):
        raise ValueError('inventory cells must be nonempty, single-line text')
    return value


def key_of(row: dict[str, str]) -> tuple[str, str, str]:
    values = tuple(row[name] for name in KEYS)
    if any(IDENTIFIER.fullmatch(value) is None for value in values):
        raise ValueError('invalid inventory identifier')
    return values


def unique_json(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError('duplicate JSON property: ' + key)
        result[key] = value
    return result


def read_inventory(base: Path, additions: Path) -> list[dict[str, str]]:
    records: dict[tuple[str, str, str], dict[str, str]] = {}

    def append(row: dict[str, Any]) -> None:
        if set(row) != set(COLUMNS):
            raise ValueError('inventory record must have exactly the sixteen declared columns')
        validated = {name: cell(row[name]) for name in COLUMNS}
        key = key_of(validated)
        if key in records:
            raise ValueError('duplicate inventory field: ' + '.'.join(key))
        records[key] = validated

    with base.open(encoding='utf-8', newline='') as source:
        reader = csv.DictReader(source, delimiter='\t')
        if reader.fieldnames != COLUMNS:
            raise ValueError('unexpected base inventory header')
        for row in reader:
            append(row)
    data = json.loads(additions.read_text(encoding='utf-8'), object_pairs_hook=unique_json)
    if not isinstance(data, dict) or set(data) != {'version', 'review_status', 'source', 'defaults', 'tables'}:
        raise ValueError('invalid addition manifest structure')
    if type(data['version']) is not int or data['version'] != 1:
        raise ValueError('unsupported inventory manifest version')
    if data['review_status'] != 'pending_control_and_legal_review':
        raise ValueError('this provisional manifest must not imply completed review')
    cell(data['source'])
    defaults = data['defaults']
    if not isinstance(defaults, dict) or set(defaults) != set(COLUMNS) - set(KEYS) - {'purpose'}:
        raise ValueError('incomplete inventory defaults')
    defaults = {name: cell(value) for name, value in defaults.items()}
    if not isinstance(data['tables'], list) or not data['tables']:
        raise ValueError('explicit table/field records are required')
    for table in data['tables']:
        if not isinstance(table, dict) or set(table) != {'schema', 'table', 'purpose', 'fields'}:
            raise ValueError('invalid table inventory record')
        if not isinstance(table['fields'], list) or not table['fields']:
            raise ValueError('explicit nonempty fields are required')
        for field in table['fields']:
            append({**defaults, 'schema': table['schema'], 'table': table['table'], 'field': field, 'purpose': table['purpose']})
    return [records[key] for key in sorted(records)]


def check_schema(rows: list[dict[str, str]], schema: Path) -> None:
    actual = set()
    with schema.open(encoding='utf-8', newline='') as source:
        for values in csv.reader(source, delimiter='\t'):
            if len(values) not in (3, 4):
                raise ValueError('schema evidence must contain three identifiers and optional type')
            key = key_of(dict(zip(KEYS, values[:3])))
            if key in actual:
                raise ValueError('duplicate field in schema evidence: ' + '.'.join(key))
            actual.add(key)
    if not actual:
        raise ValueError('empty schema evidence is not coverage')
    recorded = {key_of(row) for row in rows}
    missing, extra = actual - recorded, recorded - actual
    if missing or extra:
        detail = []
        for label, items in [('missing', missing), ('extra', extra)]:
            detail.extend(label + ': ' + '.'.join(key) for key in sorted(items))
        raise ValueError('inventory/schema mismatch\n' + '\n'.join(detail))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base', type=Path, default=ROOT / 'docs/compliance/data-inventory.tsv')
    parser.add_argument('--additions', type=Path, default=ROOT / 'docs/compliance/data-inventory-additions.json')
    parser.add_argument('--check-schema', type=Path)
    args = parser.parse_args()
    try:
        rows = read_inventory(args.base, args.additions)
        if args.check_schema:
            check_schema(rows, args.check_schema)
            pending = sum(row['classification'] == 'restricted_pending_review' for row in rows)
            print(f'Inventory covers all {len(rows)} schema fields; {pending} additions still require control/legal review.')
        else:
            writer = csv.DictWriter(sys.stdout, fieldnames=COLUMNS, delimiter='\t', lineterminator='\n')
            writer.writeheader()
            writer.writerows(rows)
        return 0
    except (OSError, ValueError, TypeError, KeyError, csv.Error) as error:
        print('Inventory validation failed: ' + str(error), file=sys.stderr)
        return 1


if __name__ == '__main__':
    raise SystemExit(main())
