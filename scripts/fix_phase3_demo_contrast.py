from pathlib import Path

p = Path('web/src/routes/demo/+page.svelte')
text = p.read_text()
repls = {
    ".amount-choice button{min-height:2.25rem;padding:.35rem .55rem;border:1px solid #bcb7ad;background:transparent;font:inherit;font-size:.68rem;font-weight:750;cursor:pointer}":
    ".amount-choice button{min-height:2.25rem;padding:.35rem .55rem;border:1px solid #bcb7ad;background:transparent;color:#555955;font:inherit;font-size:.68rem;font-weight:750;cursor:pointer}.amount-choice button:disabled{opacity:1;color:#555955;cursor:default}",
    ".amount-choice button.chosen{border-color:#2738d6;background:#2738d6;color:#fff}":
    ".amount-choice button.chosen,.amount-choice button.chosen:disabled{border-color:#2738d6;background:#2738d6;color:#fff;opacity:1}",
    ".deal-head small,.deal-facts small,.balance-card small{display:block;color:#6e716c;font-size:.62rem;font-weight:800;letter-spacing:.08em;text-transform:uppercase}":
    ".deal-head small,.deal-facts small,.balance-card small{display:block;color:#555955;font-size:.62rem;font-weight:800;letter-spacing:.08em;text-transform:uppercase}",
    ".next-action{display:flex;align-items:center;justify-content:space-between;width:100%;min-height:3.4rem;margin-top:1rem;padding:.7rem 1rem;border:0;background:#2738d6;color:#fff;font:inherit;font-size:.8rem;font-weight:850;cursor:pointer}":
    ".next-action{display:flex;align-items:center;justify-content:space-between;width:100%;min-height:3.4rem;margin-top:1rem;padding:.7rem 1rem;border:0;background:#2738d6;color:#fff;font:inherit;font-size:.8rem;font-weight:850;cursor:pointer}.next-action:disabled{opacity:1;color:#fff;cursor:default}"
}
for old,new in repls.items():
    if text.count(old) != 1:
        raise SystemExit(f'expected exactly one CSS match, got {text.count(old)} for {old[:60]}')
    text = text.replace(old,new,1)
p.write_text(text)
