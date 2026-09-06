import hashlib
import unittest

from recovery_fingerprint import quote, row_bytes


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
