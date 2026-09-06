#!/usr/bin/env bash
set -euo pipefail
: "${RELEASE_EVIDENCE_MANIFEST:?a protected reviewed manifest is required}"
: "${RELEASE_EVIDENCE_ROOT:?a protected evidence directory is required}"
: "${RELEASE_CANDIDATE_SHA:?the exact reviewed commit is required}"
: "${RELEASE_TARGET_ENVIRONMENT:?the intended environment is required}"
: "${RELEASE_ENVIRONMENT_SHA256:?the reviewed non-secret environment/configuration fingerprint is required}"
actual_sha="$(git rev-parse HEAD)"
[[ "$actual_sha" == "$RELEASE_CANDIDATE_SHA" ]] || { printf 'release commit does not match checked-out code\n' >&2; exit 1; }
[[ -z "$(git status --porcelain --untracked-files=normal)" ]] || { printf 'release checkout must be clean\n' >&2; exit 1; }
python3 scripts/verify_release_evidence.py "$RELEASE_EVIDENCE_MANIFEST" \
  --evidence-root "$RELEASE_EVIDENCE_ROOT" --commit "$actual_sha" \
  --environment "$RELEASE_TARGET_ENVIRONMENT" --environment-sha256 "$RELEASE_ENVIRONMENT_SHA256"
