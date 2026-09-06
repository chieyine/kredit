import hashlib
import tempfile
import unittest
from pathlib import Path

from verify_backup import verify


class BackupTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.path = Path(self.temp.name) / 'backup.dump'
        self.path.write_bytes(b'synthetic backup fixture')
        self.sidecar = Path(str(self.path) + '.sha256')
        self.digest = hashlib.sha256(self.path.read_bytes()).hexdigest()

    def test_requested_bytes_pass(self):
        self.sidecar.write_text(self.digest + '  backup.dump\n')
        verify(self.path)

    def test_missing_checksum_fails(self):
        with self.assertRaises(ValueError):
            verify(self.path)

    def test_tampered_bytes_fail(self):
        self.sidecar.write_text(self.digest + '  backup.dump\n')
        self.path.write_bytes(b'changed')
        with self.assertRaises(ValueError):
            verify(self.path)

    def test_sidecar_cannot_redirect_verification(self):
        other = self.path.parent / 'innocent.dump'
        other.write_bytes(b'other valid file')
        self.sidecar.write_text(hashlib.sha256(other.read_bytes()).hexdigest() + '  ' + str(other) + '\n')
        with self.assertRaises(ValueError):
            verify(self.path)

    def test_multiple_or_invalid_records_fail(self):
        for text in ['', 'not-a-hash', self.digest + '\n' + self.digest]:
            with self.subTest(text=text):
                self.sidecar.write_text(text)
                with self.assertRaises(ValueError):
                    verify(self.path)
