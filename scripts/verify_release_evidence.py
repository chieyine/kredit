#!/usr/bin/env python3
"""Validate readiness evidence completeness, scope, freshness and file hashes.

This is NOT an authenticity verifier, legal opinion or permission to deploy.
Run only against reviewed documents in protected storage. Never commit actual
customer/provider evidence. Human deployment approval remains mandatory.
"""
import argparse
import hashlib
import json
import re
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path

GATES = {
    'security_review': 'documented_review',
    'api_authorization': 'documented_review',
    'penetration_test': 'independent_assessment',
    'legal_terms': 'legal_approval',
    'mandate_consent': 'legal_approval',
    'privacy_dpia': 'privacy_approval',
    'provider_certification': 'actual_provider',
    'restore_drill': 'target_environment',
    'load_test': 'target_environment',
    'monitoring_drill': 'target_environment',
    'incident_drill': 'target_environment',
    'manual_accessibility': 'manual_assessment',
    'support_training': 'documented_review',
    'launch_approval': 'documented_review',
}
MAX_AGE = timedelta(days=90)
MAX_BYTES = 20 * 1024 * 1024


def template() -> dict:
    return {'schema_version': 1, 'candidate_sha': '', 'environment': '',
            'environment_sha256': '', 'prepared_by': '', 'findings': [],
            'gates': {name: {'status': 'pending', 'evidence_type': kind,
                             'reviewed_by': '', 'checked_at': '', 'reviewed_at': '',
                             'expires_at': '', 'path': '', 'sha256': ''}
                      for name, kind in GATES.items()}}


def timestamp(value: object) -> datetime:
    if not isinstance(value, str):
        raise ValueError('timestamp must be a string')
    result = datetime.fromisoformat(value.replace('Z', '+00:00'))
    if result.tzinfo is None:
        raise ValueError('timezone is required')
    return result.astimezone(timezone.utc)


def validate(manifest: dict, root: Path, commit: str, environment: str,
             environment_sha256: str, now: datetime | None = None) -> list[str]:
    errors = []
    now = now or datetime.now(timezone.utc)
    if not isinstance(manifest, dict) or manifest.get('schema_version') != 1:
        return ['unsupported release-evidence manifest']
    if re.fullmatch('[0-9a-f]{40}', commit or '') is None or manifest.get('candidate_sha') != commit:
        errors.append('candidate commit is missing or mismatched')
    if not environment or manifest.get('environment') != environment:
        errors.append('target environment is missing or mismatched')
    if re.fullmatch('[0-9a-f]{64}', environment_sha256 or '') is None or manifest.get('environment_sha256') != environment_sha256:
        errors.append('target configuration fingerprint is missing or mismatched')
    preparer = manifest.get('prepared_by')
    if not isinstance(preparer, str) or not preparer.strip():
        errors.append('evidence preparer is required')
    findings = manifest.get('findings')
    if not isinstance(findings, list):
        errors.append('explicit finding register is required')
    else:
        for item in findings:
            if not isinstance(item, dict) or item.get('severity') not in ('critical', 'high', 'medium', 'low') or item.get('status') not in ('open', 'resolved', 'accepted_risk') or not isinstance(item.get('reference'), str) or not item['reference'].strip():
                errors.append('invalid finding record')
            elif item['severity'] in ('critical', 'high') and item['status'] != 'resolved':
                errors.append('unresolved critical/high finding blocks release')
    gates = manifest.get('gates')
    if not isinstance(gates, dict) or set(gates) != set(GATES):
        return errors + ['all required readiness gates, and no unknown gates, must be present']
    root = root.resolve(strict=True)
    if not root.is_dir():
        return errors + ['protected evidence root must be a directory']
    for name, kind in GATES.items():
        gate = gates[name]
        if not isinstance(gate, dict):
            errors.append(name + ': invalid gate record')
            continue
        if gate.get('status') != 'approved' or gate.get('evidence_type') != kind:
            errors.append(name + ': approval or required evidence type is missing')
        reviewer = gate.get('reviewed_by')
        if not isinstance(reviewer, str) or not reviewer.strip():
            errors.append(name + ': human reviewer is missing')
        elif reviewer.strip().casefold() == str(preparer).strip().casefold():
            errors.append(name + ': reviewer must differ from preparer')
        try:
            checked, reviewed, expires = (timestamp(gate[key]) for key in ('checked_at', 'reviewed_at', 'expires_at'))
            if not checked <= reviewed <= now < expires or expires - checked > MAX_AGE or now - checked > MAX_AGE:
                raise ValueError('stale/future evidence')
        except (ValueError, KeyError, TypeError):
            errors.append(name + ': invalid, expired or future-dated review')
        try:
            relative = Path(gate['path'])
            if relative.is_absolute() or '..' in relative.parts:
                raise ValueError('evidence path escapes the root')
            path = (root / relative).resolve(strict=True)
            if not path.is_relative_to(root) or not path.is_file() or not 0 < path.stat().st_size <= MAX_BYTES:
                raise ValueError('invalid evidence file')
            expected = gate['sha256']
            if not isinstance(expected, str) or re.fullmatch('[0-9a-f]{64}', expected) is None:
                raise ValueError('invalid digest')
            digest = hashlib.sha256()
            with path.open('rb') as stream:
                for chunk in iter(lambda: stream.read(1024 * 1024), b''):
                    digest.update(chunk)
            if digest.hexdigest() != expected:
                raise ValueError('digest mismatch')
        except (OSError, ValueError, KeyError, TypeError):
            errors.append(name + ': missing, unsafe, oversized or changed evidence file')
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('manifest', type=Path, nargs='?')
    parser.add_argument('--write-template', action='store_true')
    parser.add_argument('--evidence-root', type=Path)
    parser.add_argument('--commit')
    parser.add_argument('--environment')
    parser.add_argument('--environment-sha256')
    args = parser.parse_args()
    if args.write_template:
        print(json.dumps(template(), indent=2))
        return 0
    if args.manifest is None or args.evidence_root is None:
        raise ValueError('manifest and protected evidence root are required')
    raw = args.manifest.read_bytes()
    if len(raw) > 1024 * 1024:
        raise ValueError('manifest is too large')
    errors = validate(json.loads(raw), args.evidence_root, args.commit, args.environment, args.environment_sha256)
    if errors:
        print('Release evidence REJECTED:\n' + '\n'.join(errors), file=sys.stderr)
        return 1
    print('Evidence completeness/scope/hashes passed. Authenticity and deployment require human approval.')
    return 0


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, ValueError, KeyError, TypeError):
        print('Release evidence could not be verified; no approval was granted.', file=sys.stderr)
        sys.exit(1)
