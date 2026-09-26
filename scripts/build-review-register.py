#!/usr/bin/env python3
"""Regenerate the file-review register from facts, not from a template.

The register this replaces recorded one identical sentence against all 1,421
files, zero findings, and a toolchain version the repository does not pin. A
register whose every entry is "verified correct" cannot distinguish a clean
file from a broken one, so it is evidence of nothing.

This generator records only what is true: the file's real size and digest, the
depth at which this audit actually examined it, and the findings that cite it.
"not-reviewed" is a legitimate, expected outcome and is written as such.
"""
from __future__ import annotations
import argparse, hashlib, json, subprocess, sys, csv
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

# Reuse review depth only when explicit evidence names this exact file digest.
# Path prefixes cannot prove that a newly added or edited file was inspected.
DEFAULT_EVIDENCE = ROOT / "docs/launch-audit-2026-09-21/file-review-index.json"

# Findings that cite each path. Written by hand from the audit report so the
# register points back at the evidence rather than asserting a verdict.
FINDINGS: dict[str, list[str]] = {}
def cite(fid: str, *paths: str) -> None:
    for p in paths:
        FINDINGS.setdefault(p, []).append(fid)

cite("A2-001", "docs/launch-audit-2026-09-21/file-review-index.json",
     "docs/launch-audit-2026-09-21/README.md", "docs/launch-audit-2026-09-21/FILE-BY-FILE.md",
     "docs/launch-audit-2026-09-21/FINDINGS.md", "docs/launch-audit-2026-09-21/file-review.csv")
cite("A2-002", "internal/credit/store.go", "internal/credit/postgres.go",
     "internal/web/financial_reads.go", "internal/web/runtime.go",
     "internal/paymentclaims/postgres.go", "internal/audit/store.go",
     "internal/support/store.go", "internal/organizations/postgres.go")
cite("A2-003", "internal/payments/postgres.go", "internal/operations/postgres.go",
     "internal/disputes/postgres.go", "internal/db/schedule_adjustment.go",
     "internal/ledger/postgres.go")
cite("A2-004", "internal/disputes/postgres.go")
cite("A2-005", "internal/ledger/postgres.go", "internal/payments/postgres.go",
     "internal/db/postgres.go", "internal/auth/postgres.go", "internal/organizations/postgres.go")
cite("A2-006", "scripts/rls-policy-shape-check.sh", "docs/compliance/rls-permissive-baseline.txt",
     "db/migrations/009_milestone8_disputes_operations.sql",
     "db/migrations/029_runtime_domain_repository_policies.sql",
     "db/migrations/037_dispute_operation_runtime_policy.sql",
     "db/migrations/052_sweep_collection_safety.sql",
     "db/migrations/056_financial_reconciliation.sql")
cite("A2-007", "internal/web/mono_handlers.go")
cite("A2-008", "internal/credit/postgres.go")
cite("A2-009", "internal/web/runtime.go")
cite("A2-010", "internal/auth/postgres.go", "internal/buyers/postgres.go",
     "internal/notifications/store.go", "internal/platformsettings/crypto.go")
cite("A2-011", "internal/ledger/reconcile.go")
cite("A2-012", "internal/outbox/store.go")
cite("A2-013", "internal/outbox/dispatcher.go")
cite("A2-014", "cmd/worker/main.go")
cite("A2-015", "internal/web/server.go", "internal/auth/postgres.go")
cite("A2-016", "internal/web/auth_handlers.go")
cite("A2-017", "internal/web/document_handlers.go", "internal/access/roles.go")
cite("A2-018", "internal/config/config.go", "internal/config/admin_connections.go")
cite("A2-019", "internal/publictoken/token.go")
cite("A2-020", "web/package.json")
cite("A2-022", *[f".github/workflows/{n}.yml" for n in (
     "audit-closeout-diagnostics", "audit-document-retention", "audit-format-diagnostic",
     "audit-journal-replay", "audit-offline-validation", "audit-rollback-cleanup",
     "audit-source-review", "audit-source-snapshot")])
cite("A2-023", ".github/workflows/ci.yml", "infra/containers/Dockerfile.api",
     "infra/containers/Dockerfile.web", "infra/environments/versions.tf")
cite("A2-024", "infra/environments/versions.tf")
cite("A2-025", "infra/postgres/roles.sql")
cite("A2-026", "infra/environments/Caddyfile.prod")
cite("A2-027", "internal/platform/logging/sanitize.go", "internal/platform/logging/logging.go")
cite("A2-028", "internal/web/http_helpers.go")
cite("A2-029", "web/src/routes/admin/platform-settings/+page.svelte")
cite("A2-030", "web/src/lib/components/Deck.svelte")


def depth(path: str, digest: str, evidence: dict[str, dict]) -> str:
    prior = evidence.get(path, {})
    reviewed = prior.get("review_depth", "not-reviewed")
    if prior.get("sha256") == digest and reviewed in {"read-in-full", "swept"}:
        return reviewed
    return "not-reviewed"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--review-evidence", type=Path, default=DEFAULT_EVIDENCE)
    args = parser.parse_args()
    evidence = {}
    if args.review_evidence.is_file():
        source = json.loads(args.review_evidence.read_text(encoding="utf-8"))
        evidence = {entry["file"]: entry for entry in source.get("entries", [])}
    files = subprocess.run(["git", "ls-files", "-z"], cwd=ROOT, capture_output=True, check=True)
    paths = [p for p in files.stdout.decode().split("\0") if p]
    head = subprocess.run(["git", "rev-parse", "HEAD"], cwd=ROOT,
                          capture_output=True, text=True, check=True).stdout.strip()
    entries = []
    for rel in sorted(paths):
        fp = ROOT / rel
        if not fp.is_file():
            continue
        raw = fp.read_bytes()
        try:
            lines = raw.decode("utf-8").count("\n") + (0 if raw.endswith(b"\n") or not raw else 1)
        except UnicodeDecodeError:
            lines = None
        entries.append({
            "file": rel,
            "bytes": len(raw),
            "lines": lines,
            "sha256": hashlib.sha256(raw).hexdigest(),
            "review_depth": depth(rel, hashlib.sha256(raw).hexdigest(), evidence),
            "findings": sorted(set(FINDINGS.get(rel, []))),
        })
    counts: dict[str, int] = {}
    for e in entries:
        counts[e["review_depth"]] = counts.get(e["review_depth"], 0) + 1
    register = {
        "schema": "kredit.file-review/2",
        "commit": head,
        "generator": "scripts/build-review-register.py",
        "audit": "A2 — independent code audit, 21 September 2026",
        "caveat": ("review_depth records how far this audit looked, not a verdict on the "
                   "file. 'not-reviewed' means exactly that. No entry asserts correctness."),
        "counts": {"files": len(entries), **{k: counts[k] for k in sorted(counts)},
                   "files_with_findings": sum(1 for e in entries if e["findings"]),
                   "distinct_findings": len({f for e in entries for f in e["findings"]})},
        "entries": entries,
    }
    out = ROOT / "docs/launch-audit-2026-09-21/file-review-index.json"
    out.write_text(json.dumps(register, indent=1) + "\n")
    csv_out = ROOT / "docs/launch-audit-2026-09-21/file-review.csv"
    with csv_out.open("w", newline="") as fh:
        w = csv.writer(fh)
        w.writerow(["file", "bytes", "lines", "sha256", "review_depth", "findings"])
        for e in entries:
            w.writerow([e["file"], e["bytes"], e["lines"], e["sha256"],
                        e["review_depth"], " ".join(e["findings"])])
    print(json.dumps(register["counts"], indent=1))
    return 0


if __name__ == "__main__":
    sys.exit(main())
