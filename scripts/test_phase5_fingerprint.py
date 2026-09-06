import hashlib
import unittest
from unittest.mock import patch

from recovery_fingerprint import connection_environment, quote, row_bytes


class FingerprintTests(unittest.TestCase):
    def test_quote_identifier(self):
        self.assertEqual('"safe"', quote('safe'))
        self.assertEqual('"a""b"', quote('a"b'))

    def test_numeric_precision_is_not_rounded(self):
        first = '{"table":"fixture","row":{"amount":9007199254740992.001}}\n'
        second = '{"table":"fixture","row":{"amount":9007199254740992.002}}\n'
        self.assertNotEqual(hashlib.sha256(row_bytes(first)).digest(), hashlib.sha256(row_bytes(second)).digest())
        self.assertEqual(first.encode(), row_bytes(first))

    def test_unicode_and_embedded_escapes_preserved(self):
        line = '{"table":"fixture","row":{"text":"Naira ₦\\nnext"}}\n'
        self.assertEqual(line.encode('utf-8'), row_bytes(line))

    def test_uri_becomes_explicit_libpq_parameters(self):
        with patch.dict('os.environ', {'PGHOST': 'wrong', 'PGSERVICE': 'wrong', 'PGOPTIONS': 'unsafe'}):
            env = connection_environment('postgres://test%40user:p%40ss@127.0.0.1:5432/fixture?sslmode=require')
        self.assertEqual('test@user', env['PGUSER'])
        self.assertEqual('p@ss', env['PGPASSWORD'])
        self.assertEqual('fixture', env['PGDATABASE'])
        self.assertEqual('127.0.0.1', env['PGHOST'])
        self.assertEqual('require', env['PGSSLMODE'])
        self.assertNotIn('PGSERVICE', env)
        self.assertNotIn('PGOPTIONS', env)

    def test_ambiguous_or_unsupported_connections_fail(self):
        for raw in ['', 'postgres://localhost/fixture', 'postgres://user@localhost/',
                    'postgres://user@localhost/fixture?sslmode=require&sslmode=disable',
                    'postgres://user@localhost/fixture?options=unsafe']:
            with self.subTest(raw=raw), self.assertRaises(ValueError):
                connection_environment(raw)
