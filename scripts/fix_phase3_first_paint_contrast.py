from pathlib import Path

p = Path('web/src/routes/demo/+page.svelte')
text = p.read_text()
replacements = [
    ("\timport { onMount } from 'svelte';\n\n", ""),
    ("\tlet interactive = $state(false);\n", ""),
    ("\tonMount(() => { interactive = true; });\n", ""),
    ("<button disabled={!interactive} class:chosen={amount === option}", "<button class:chosen={amount === option}"),
    ("<button class=\"next-action\" disabled={!interactive} onclick={next}", "<button class=\"next-action\" onclick={next}"),
    (".amount-choice button:disabled{opacity:1;color:#555955;cursor:default}", ""),
    (".amount-choice button.chosen,.amount-choice button.chosen:disabled{border-color:#2738d6;background:#2738d6;color:#fff;opacity:1}", ".amount-choice button.chosen{border-color:#2738d6;background:#2738d6;color:#fff}"),
    (".next-action:disabled{opacity:1;color:#fff;cursor:default}", ""),
    ("@keyframes frame-in{from{transform:translateY(8px);opacity:.75}to{transform:none;opacity:1}}", "@keyframes frame-in{from{transform:translateY(8px)}to{transform:none}}"),
]
for old, new in replacements:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f'expected exactly one match for {old[:80]!r}, found {count}')
    text = text.replace(old, new, 1)
p.write_text(text)
