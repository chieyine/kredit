import unittest
from unittest.mock import patch

from phase5_load import percentile, run, valid_body, validate_origin


class LoadBaselineTests(unittest.TestCase):
    def test_explicit_loopback_only(self):
        self.assertEqual('http://127.0.0.1:8080', validate_origin('http://127.0.0.1:8080/'))
        for url in ['https://kredit.ng', 'http://localhost:8080', 'http://127.0.0.1', 'http://127.0.0.1:8080/api', 'http://user:password@127.0.0.1:8080', 'http://127.0.0.1:8080?secret=1', 'http://127.0.0.1:8080#fragment']:
            with self.subTest(url=url), self.assertRaises(ValueError):
                validate_origin(url)

    def test_percentiles_are_nearest_rank(self):
        self.assertEqual(95, percentile(list(range(1, 101)), .95))
        self.assertEqual(99, percentile(list(range(1, 101)), .99))
        self.assertEqual(4, percentile([1, 2, 3, 4], .99))
        with self.assertRaises(ValueError):
            percentile([], .95)

    def test_error_or_empty_buyer_payload_is_not_success(self):
        self.assertFalse(valid_body('supplier_payments', {'error': 'unavailable'}))
        self.assertFalse(valid_body('buyer_requests', {'requests': []}))
        self.assertTrue(valid_body('buyer_requests', {'requests': [{'id': 'fixture'}]}))
        self.assertTrue(valid_body('supplier_payments', {'payments': []}))

    def test_no_environment_acknowledgement_no_network(self):
        with patch.dict('os.environ', {}, clear=True), patch('phase5_load.login') as login:
            with self.assertRaises(ValueError):
                run('http://127.0.0.1:8080', 64, 4, 1000, 2000)
            login.assert_not_called()
