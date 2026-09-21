#!/usr/bin/env bash
set -euo pipefail

# A mutable tag is not a pin. `actions/checkout@v4` and `golang:1.26.8-alpine`
# both resolve to whatever the publisher last pushed, so a compromised or
# simply changed upstream lands in a build that signs financial artefacts
# without any change to this repository.
#
# The repository already knows this: the audit workflows pin actions by commit
# SHA, and ci.yml checksum-verifies the golangci-lint binary it downloads -
# with a comment explaining why. That rigour just was not applied everywhere.
#
# This gate is a ratchet. Everything still unpinned is listed in the baseline;
# anything not in the baseline fails, and an entry that has been pinned must be
# removed from the baseline. The list can only shrink.
#
# To resolve a pin (needs network):
#   action: gh api repos/<owner>/<repo>/git/ref/tags/<tag> --jq .object.sha
#   image:  docker buildx imagetools inspect <image>:<tag> --format '{{.Manifest.Digest}}'

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
baseline="docs/compliance/supply-chain-unpinned-baseline.txt"
[[ -s "$baseline" ]] || { printf 'Baseline is missing: %s\n' "$baseline" >&2; exit 1; }

actual="$(mktemp)"; expected="$(mktemp)"
trap 'rm -f "$actual" "$expected"' EXIT

{
  # GitHub Actions referenced by anything other than a 40-character commit SHA.
  grep -rhoE '^\s*(-\s*)?uses:\s*[^ ]+' .github/workflows/*.yml 2>/dev/null \
    | sed -E 's/^\s*(-\s*)?uses:\s*//' \
    | grep -vE '@[0-9a-f]{40}$' \
    | sed 's/^/action /' || true
  # Container base images referenced by tag rather than digest.
  grep -rhoE '^FROM\s+[^ ]+' infra/containers/Dockerfile.* 2>/dev/null \
    | sed -E 's/^FROM\s+//' \
    | grep -vE '@sha256:[0-9a-f]{64}$' \
    | grep -vE '^(build|runtime-base|build-simulator)$' \
    | sed 's/^/image /' || true
} | sort -u > "$actual"

{ grep -vE '^\s*(#|$)' "$baseline" || true; } | sort -u > "$expected"

if ! diff -u "$expected" "$actual" > /dev/null; then
  printf 'Supply-chain pinning changed.\n\n' >&2
  printf '+ lines are newly unpinned references. Pin them by commit SHA or image digest.\n' >&2
  printf -- '- lines are references now pinned; remove them from %s.\n\n' "$baseline" >&2
  diff -u "$expected" "$actual" >&2 || true
  exit 1
fi
printf 'Supply-chain pinning matches the baseline (%s unpinned reference(s) remaining).\n' "$(grep -c . "$actual" || true)"
