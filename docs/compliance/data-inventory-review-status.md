# Field inventory: structural coverage and review status

The canonical inventory is rendered from the original `data-inventory.tsv` plus the explicit records in `data-inventory-additions.json`:

```sh
python3 scripts/data-inventory.py > /tmp/kredit-data-inventory.tsv
```

The original 1,424 records are preserved byte-for-byte. The manifest records the 212 missing fields observed in the retained synthetic PostgreSQL schema for workflow 35571022032. It does not copy field names dynamically from a live database. The checker compares the complete resolved key set with the actual schema and fails for missing fields, extra fields, duplicates, malformed rows or incomplete metadata. Ordering is presentation, not a reason to omit or duplicate a record.

## No compliance sign-off is implied

The additions deliberately use `restricted_pending_review`. Access roles, effective row policies, field encryption, retention periods, lawful bases, accountable owners, external processors and transfer locations require review. This inventory is not authorization to delete financial or identity evidence, and does not certify those controls as implemented. The original entries retain their previous review status; preserving them does not constitute a fresh approval.

Before release, assign the accountable owners, confirm each control against code and deployment, and record approved classifications and retention/hold rules. Moving a field to a completed register must remove its provisional counterpart in the same reviewed change; duplicates are rejected rather than silently overwritten.

The parser and its negative tests run before the real-schema comparison. Missing or newly added schema fields must be reviewed and explicitly recorded; never regenerate an apparently complete inventory by inventing control approvals.
