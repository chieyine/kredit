#!/usr/bin/env python3
"""Negative and preservation tests for the explicit field inventory."""
import copy
import csv
import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent
SPEC = importlib.util.spec_from_file_location('inventory', ROOT / 'scripts/data-inventory.py')
inventory = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(inventory)
BASE = ROOT / 'docs/compliance/data-inventory.tsv'
ADDITIONS = ROOT / 'docs/compliance/data-inventory-additions.json'


class InventoryTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.manifest = json.loads(ADDITIONS.read_text())

    def rendered(self, manifest=None):
        path = self.root / 'additions.json'
        path.write_text(json.dumps(self.manifest if manifest is None else manifest))
        return inventory.read_inventory(BASE, path)

    def schema(self, keys):
        path = self.root / 'schema.tsv'
        path.write_text(''.join('\t'.join(key) + '\n' for key in keys))
        return path

    def test_preserves_every_original_record_and_names_all_additions(self):
        raw = BASE.read_bytes()
        self.assertEqual(hashlib.sha1(b'blob ' + str(len(raw)).encode() + b'\0' + raw).hexdigest(), '138de0efb499b10296b4f0ea5f873206c427f554')
        with BASE.open(newline='') as source:
            original = list(csv.DictReader(source, delimiter='\t'))
        rows = self.rendered()
        lookup = {inventory.key_of(row): row for row in rows}
        self.assertEqual(len(original), 1424)
        self.assertEqual(len(rows), 1636)
        for row in original:
            self.assertEqual(row, lookup[inventory.key_of(row)])
        self.assertEqual(sum(row['classification'] == 'restricted_pending_review' for row in rows), 212)
        self.assertEqual(rows, self.rendered())

    def test_live_schema_cannot_silently_add_or_remove_fields(self):
        rows = self.rendered()
        keys = [inventory.key_of(row) for row in rows]
        inventory.check_schema(rows, self.schema(reversed(keys)))
        for invalid in [keys[:-1], keys + [('app', 'unknown_table', 'unknown_field')], [], keys + [keys[0]]]:
            with self.assertRaises(ValueError):
                inventory.check_schema(rows, self.schema(invalid))

    def test_duplicates_cannot_replace_an_existing_record(self):
        duplicate = copy.deepcopy(self.manifest)
        duplicate['tables'][0]['fields'].append(duplicate['tables'][0]['fields'][0])
        with self.assertRaises(ValueError):
            self.rendered(duplicate)
        duplicate = copy.deepcopy(self.manifest)
        duplicate['tables'].append({'schema':'app','table':'account_recovery_codes','purpose':'test','fields':['id']})
        with self.assertRaises(ValueError):
            self.rendered(duplicate)
        with self.assertRaises(ValueError):
            json.loads('{"field":1,"field":2}', object_pairs_hook=inventory.unique_json)

    def test_empty_cells_control_characters_and_false_approval_are_rejected(self):
        for value in ['', ' ', 'one\ttwo', 'one\ntwo', 42, None]:
            changed = copy.deepcopy(self.manifest)
            changed['defaults']['retention'] = value
            with self.assertRaises(ValueError):
                self.rendered(changed)
        changed = copy.deepcopy(self.manifest)
        changed['review_status'] = 'approved'
        with self.assertRaises(ValueError):
            self.rendered(changed)
        changed = copy.deepcopy(self.manifest)
        changed['tables'][0]['fields'].append('invalid.field')
        with self.assertRaises(ValueError):
            self.rendered(changed)
        changed = copy.deepcopy(self.manifest)
        del changed['defaults']['protection']
        with self.assertRaises(ValueError):
            self.rendered(changed)


if __name__ == '__main__':
    unittest.main()
