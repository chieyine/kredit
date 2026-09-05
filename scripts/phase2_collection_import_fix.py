from pathlib import Path

p = Path("internal/web/collection_jobs.go")
text = p.read_text()
if '"kredit/internal/collections"' not in text:
    anchor = '"kredit/internal/config"\n'
    if text.count(anchor) != 1:
        raise SystemExit(f"collection import anchor expected 1 match, found {text.count(anchor)}")
    text = text.replace(anchor, '"kredit/internal/collections"\n\t"kredit/internal/config"\n', 1)
p.write_text(text)
