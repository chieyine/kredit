"""Synthetic validator tests, not provider certification evidence."""
import copy
import hashlib
import tempfile
import unittest
from pathlib import Path
from verify_provider_evidence import CONTRACTS, validate


class EvidenceTests(unittest.TestCase):
    def setUp(self):
        self.folder = tempfile.TemporaryDirectory()
        self.addCleanup(self.folder.cleanup)
        self.root = Path(self.folder.name)
        data = b"synthetic validator fixture, not a provider response"
        (self.root / "case.json").write_bytes(data)
        self.pack = {
            "schema_version": 1, "provider": "mono-sweep", "environment": "mono-sandbox",
            "adapter_commit": "a" * 40, "completed_at": "2020-01-01T00:00:00Z",
            "sweep_access_confirmed": True, "partial_sweep_access_confirmed": True,
            "human_review_complete": True, "reviewer": "test-reviewer",
            "provider_confirmation_reference": "restricted-test-reference",
            "contract_confirmations": {key: True for key in CONTRACTS},
            "scenarios": [{"id": i, "status": "passed", "source": "actual_mono_sandbox",
                           "assertions_verified": True, "expected": "test assertion",
                           "actual": "test assertion", "evidence": [{"path": "case.json", "sha256": hashlib.sha256(data).hexdigest()}]}
                          for i in range(1, 22)],
        }

    def test_complete_manifest(self):
        self.assertEqual(validate(self.pack, self.root), [])

    def test_incomplete_and_synthetic_evidence_rejected(self):
        for mutate in (
            lambda p: p.update(human_review_complete=False),
            lambda p: p.update(environment="local-fixture"),
            lambda p: p.update(completed_at="2999-01-01T00:00:00Z"),
            lambda p: p["scenarios"].pop(),
            lambda p: p["scenarios"][0].update(status="pending"),
            lambda p: p["scenarios"][0].update(source="synthetic"),
            lambda p: p["scenarios"][0].update(id=2),
            lambda p: p["scenarios"][0].update(assertions_verified=False),
        ):
            pack = copy.deepcopy(self.pack)
            mutate(pack)
            self.assertTrue(validate(pack, self.root))

    def test_hash_and_path_guard(self):
        for path, digest in (("../outside", "a" * 64), ("case.json", "a" * 64), ("/etc/passwd", "a" * 64)):
            pack = copy.deepcopy(self.pack)
            pack["scenarios"][0]["evidence"] = [{"path": path, "sha256": digest}]
            self.assertTrue(validate(pack, self.root))

    def test_malformed_manifest(self):
        for pack in ([], None, {"scenarios": [None]}, {"scenarios": "invalid"}):
            self.assertTrue(validate(pack, self.root))


if __name__ == "__main__":
    unittest.main()
