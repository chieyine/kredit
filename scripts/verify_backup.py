#!/usr/bin/env python3
"""Verify the requested backup bytes, never a filename supplied by a sidecar."""
import hashlib
import re
import sys
from pathlib import Path


def verify(path: Path) -> None:
    sidecar = Path(str(path) + '.sha256')
    if not path.is_file():
        raise ValueError('backup file is required')
    if not sidecar.is_file():
        raise ValueError('backup checksum file is required')
    lines = [line for line in sidecar.read_text(encoding='utf-8').splitlines() if line.strip()]
    if len(lines) != 1:
        raise ValueError('exactly one backup checksum is required')
    expected = lines[0].split()[0]
    if re.fullmatch(r'[a-fA-F0-9]{64}', expected) is None:
        raise ValueError('backup checksum is invalid')
    digest = hashlib.sha256()
    with path.open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(chunk)
    if digest.hexdigest() != expected.lower():
        raise ValueError('backup checksum mismatch')


if __name__ == '__main__':
    try:
        if len(sys.argv) != 2:
            raise ValueError('usage: verify_backup.py BACKUP_FILE')
        verify(Path(sys.argv[1]))
    except (OSError, ValueError, UnicodeError) as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
    print('requested_backup_checksum=passed')
