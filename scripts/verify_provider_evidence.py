"""Validate a restricted Mono sandbox evidence pack; never enable production."""
import argparse
import hashlib
import json
import re
from datetime import datetime, timezone
from pathlib import Path

REQUIRED = set(range(1, 22))
CONTRACTS = {"retrieve_debit_path", "partial_result_shape", "reference_identity", "mandate_validity", "cancellation_semantics"}


def validate(pack: dict, root: Path) -> list[str]:
    errors = []
    if not isinstance(pack, dict):
        return ["Evidence manifest must be an object."]
    if pack.get("schema_version") != 1 or pack.get("provider") != "mono-sweep":
        errors.append("Unsupported evidence schema/provider.")
    if pack.get("environment") != "mono-sandbox":
        errors.append("Actual isolated Mono sandbox evidence is required.")
    if not re.fullmatch(r"[0-9a-f]{40}", str(pack.get("adapter_commit", ""))):
        errors.append("Pin the adapter commit SHA.")
    for flag in ("sweep_access_confirmed", "partial_sweep_access_confirmed", "human_review_complete"):
        if pack.get(flag) is not True:
            errors.append(f"Required confirmation missing: {flag}.")
    if not pack.get("reviewer") or not pack.get("provider_confirmation_reference"):
        errors.append("Reviewer and restricted provider confirmation reference are required.")
    try:
        stamp = datetime.fromisoformat(str(pack.get("completed_at", "")).replace("Z", "+00:00"))
        if stamp.tzinfo is None or stamp > datetime.now(timezone.utc):
            raise ValueError
    except ValueError:
        errors.append("A real, non-future, timezone-aware completion date is required.")
    confirmed = pack.get("contract_confirmations", {})
    if not isinstance(confirmed, dict) or any(confirmed.get(key) is not True for key in CONTRACTS):
        errors.append("Provider contract questions are not all confirmed.")
    scenarios = pack.get("scenarios")
    if not isinstance(scenarios, list):
        return errors + ["Scenario evidence must be a list."]
    ids = []
    root = root.resolve()
    for case in scenarios:
        if not isinstance(case, dict) or type(case.get("id")) is not int:
            errors.append("Invalid scenario entry.")
            continue
        number = case["id"]
        ids.append(number)
        prefix = f"Scenario {number}"
        if case.get("status") != "passed" or case.get("source") != "actual_mono_sandbox":
            errors.append(prefix + ": actual sandbox pass evidence is missing.")
        if case.get("assertions_verified") is not True:
            errors.append(prefix + ": expected/actual assertions have not been reviewed.")
        if not case.get("expected") or not case.get("actual"):
            errors.append(prefix + ": record expected and actual outcomes.")
        refs = case.get("evidence")
        if not isinstance(refs, list) or not refs:
            errors.append(prefix + ": restricted evidence files are required.")
            continue
        for evidence in refs:
            try:
                relative = Path(evidence["path"])
                digest = evidence["sha256"]
                path = (root / relative).resolve()
                if relative.is_absolute() or not path.is_relative_to(root) or not path.is_file():
                    raise ValueError
                if not re.fullmatch(r"[0-9a-f]{64}", str(digest)):
                    raise ValueError
                if hashlib.sha256(path.read_bytes()).hexdigest() != digest:
                    raise ValueError
            except (OSError, ValueError, TypeError, KeyError):
                errors.append(prefix + ": missing, unsafe or hash-mismatched evidence file.")
    if set(ids) != REQUIRED or len(ids) != 21:
        errors.append("Exactly one result for each of the 21 required scenarios is needed.")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("manifest", type=Path)
    parser.add_argument("--evidence-dir", required=True, type=Path)
    args = parser.parse_args()
    try:
        pack = json.loads(args.manifest.read_text(encoding="utf-8"))
    except (OSError, ValueError):
        print("BLOCKED: evidence manifest is missing or invalid JSON.")
        return 2
    errors = validate(pack, args.evidence_dir)
    for error in errors:
        print("BLOCKED: " + error)
    if errors:
        return 2
    print("Evidence manifest and file hashes are complete. Human provenance review, provider approval and all release gates remain required. No production setting was changed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
