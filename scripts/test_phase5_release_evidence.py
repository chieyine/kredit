import copy
import hashlib
import tempfile
import unittest
from datetime import datetime, timedelta, timezone
from pathlib import Path

from verify_release_evidence import GATES, template, validate


class ReleaseEvidenceTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.now = datetime(2026, 9, 6, tzinfo=timezone.utc)
        self.sha = 'a' * 40
        self.envhash = 'b' * 64
        self.manifest = template()
        self.manifest.update(candidate_sha=self.sha, environment='isolated-test', environment_sha256=self.envhash, prepared_by='fixture-author')
        for name, gate in self.manifest['gates'].items():
            path = self.root / (name + '.txt')
            path.write_text('Synthetic unit-test review, NOT actual approval: ' + name)
            gate.update(status='approved', reviewed_by='fixture-reviewer',
                        checked_at=(self.now - timedelta(days=1)).isoformat(),
                        reviewed_at=self.now.isoformat(), expires_at=(self.now + timedelta(days=30)).isoformat(),
                        path=path.name, sha256=hashlib.sha256(path.read_bytes()).hexdigest())

    def check(self, value):
        return validate(value, self.root, self.sha, 'isolated-test', self.envhash, self.now)

    def test_structurally_complete_synthetic_manifest(self):
        self.assertEqual([], self.check(self.manifest))

    def test_pending_template_is_not_approval(self):
        self.assertTrue(self.check(template()))

    def test_each_gate_is_required(self):
        for name in GATES:
            with self.subTest(gate=name):
                value = copy.deepcopy(self.manifest)
                del value['gates'][name]
                self.assertTrue(self.check(value))

    def test_scope_mismatches_fail(self):
        for key in ('candidate_sha', 'environment', 'environment_sha256'):
            value = copy.deepcopy(self.manifest)
            value[key] = 'wrong'
            self.assertTrue(self.check(value))

    def test_missing_pending_or_synthetic_approval_fails(self):
        for key, replacement in [('status', 'pending'), ('evidence_type', 'synthetic'), ('reviewed_by', ''), ('reviewed_by', 'fixture-author')]:
            value = copy.deepcopy(self.manifest)
            value['gates']['penetration_test'][key] = replacement
            self.assertTrue(self.check(value))

    def test_invalid_review_dates_fail(self):
        for key, replacement in [('checked_at', 'bad'), ('checked_at', '2026-09-06'),
                                 ('reviewed_at', (self.now + timedelta(days=1)).isoformat()),
                                 ('expires_at', self.now.isoformat()),
                                 ('checked_at', (self.now - timedelta(days=100)).isoformat())]:
            value = copy.deepcopy(self.manifest)
            value['gates']['restore_drill'][key] = replacement
            self.assertTrue(self.check(value))

    def test_open_or_accepted_high_risk_blocks(self):
        for severity in ('critical', 'high'):
            for status in ('open', 'accepted_risk'):
                value = copy.deepcopy(self.manifest)
                value['findings'] = [{'severity': severity, 'status': status, 'reference': 'fixture'}]
                self.assertTrue(self.check(value))

    def test_resolved_findings_do_not_block(self):
        self.manifest['findings'] = [{'severity': 'high', 'status': 'resolved', 'reference': 'fixture'}]
        self.assertEqual([], self.check(self.manifest))

    def test_explicit_finding_register_required(self):
        del self.manifest['findings']
        self.assertTrue(self.check(self.manifest))

    def test_changed_file_fails(self):
        (self.root / 'security_review.txt').write_text('tampered')
        self.assertTrue(self.check(self.manifest))

    def test_path_traversal_absolute_and_missing_files_fail(self):
        for path in ['../escape', '/etc/passwd', 'missing.txt', '']:
            value = copy.deepcopy(self.manifest)
            value['gates']['restore_drill']['path'] = path
            self.assertTrue(self.check(value))

    def test_symlink_escape_fails(self):
        with tempfile.TemporaryDirectory() as elsewhere:
            outside = Path(elsewhere) / 'private.txt'
            outside.write_text('not inside approved evidence root')
            (self.root / 'escape').symlink_to(outside)
            self.manifest['gates']['restore_drill']['path'] = 'escape'
            self.assertTrue(self.check(self.manifest))

    def test_wrong_schema_and_unknown_gate_fail(self):
        self.assertTrue(self.check({'schema_version': 2}))
        self.manifest['gates']['unknown'] = {}
        self.assertTrue(self.check(self.manifest))
