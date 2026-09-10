<script lang="ts">
  import { onMount, type Snippet } from 'svelte';
  let { title, description, version, effectiveDate, sections, children }:
    { title: string; description: string; version: string; effectiveDate: string; sections: {id:string; title:string}[]; children: Snippet } = $props();
  let active = $state('');
  let mobile: HTMLDetailsElement;
  onMount(() => {
    const observer = new IntersectionObserver(entries => {
      for (const entry of entries) if (entry.isIntersecting) active = entry.target.id;
    }, { rootMargin: '-100px 0px -60% 0px' });
    sections.forEach(section => { const el = document.getElementById(section.id); if (el) observer.observe(el); });
    return () => observer.disconnect();
  });
</script>
{#snippet links()}
  {#each sections as section}<a href={`#${section.id}`} aria-current={active === section.id ? 'location' : undefined} onclick={() => { if (mobile) mobile.open = false; }}>{section.title}</a>{/each}
{/snippet}
<main class="shell document">
  <header><p class="eyebrow">Kredit · Legal</p><h1>{title}</h1><p class="description">{description}</p><p class="metadata">Effective {effectiveDate} <span aria-hidden="true">·</span> {version}</p></header>
  <div class="document-grid">
    <aside><nav class="desktop-contents" aria-label="On this page"><strong>On this page</strong>{@render links()}</nav><details class="mobile-contents" bind:this={mobile}><summary>On this page</summary><nav aria-label="Document sections">{@render links()}</nav></details></aside>
    <article>{@render children()}</article>
  </div>
</main>
<style>
.document{max-width:1160px;padding-block:3.5rem 6rem}.document header{max-width:760px;margin-bottom:3rem}.document h1{font-size:clamp(2.3rem,5vw,4rem);font-family:var(--font-serif);font-weight:500;letter-spacing:-.035em;line-height:1.1;margin:.7rem 0 1.2rem}.description{font-size:1.1rem;line-height:1.65;color:var(--color-muted);max-width:65ch}.metadata{font-size:.85rem;color:var(--color-muted);display:flex;gap:.7rem;flex-wrap:wrap;margin-top:1.5rem}.document-grid{display:grid;grid-template-columns:240px minmax(0,1fr);gap:4rem;border-top:1px solid var(--color-border);padding-top:2rem}.desktop-contents{position:sticky;top:7rem;max-height:calc(100dvh - 9rem);overflow:auto;display:grid;gap:.15rem}.desktop-contents strong{font-size:.85rem;margin-bottom:.7rem}nav a{display:block;padding:.55rem .7rem;border-left:2px solid transparent;color:var(--color-muted);font-size:.88rem;line-height:1.45;text-decoration:none}nav a:hover,nav a[aria-current]{color:var(--color-primary);border-left-color:var(--color-primary);background:#f3f4fc}article{min-width:0;max-width:72ch}article :global(section){scroll-margin-top:7rem;margin:0 0 2.8rem}article :global(h2){font:650 1.4rem/1.35 var(--font-sans,Arial,sans-serif);letter-spacing:-.015em;margin:0 0 1rem}article :global(h3){font-size:1.05rem;margin-top:1.6rem}article :global(p),article :global(li){font-size:1rem;line-height:1.8;color:var(--color-foreground)}article :global(li){margin:.5rem 0}article :global(.detail-grid),article :global(.purpose-list){display:block}article :global(.document-actions){display:flex;flex-wrap:wrap;gap:1rem;border-top:1px solid var(--color-border);padding-top:1.5rem}.mobile-contents{display:none}@media(max-width:800px){.document{padding-block:2rem 4rem}.document header{margin-bottom:1.5rem}.document-grid{display:block;padding-top:0}.desktop-contents{display:none}.mobile-contents{display:block;margin-bottom:2rem;border-bottom:1px solid var(--color-border)}summary{min-height:3rem;display:flex;align-items:center;cursor:pointer;font-weight:650}summary::after{content:'+';margin-left:auto}details[open] summary::after{content:'−'}nav a{min-height:44px;box-sizing:border-box;display:flex;align-items:center}.document-grid article{max-width:100%}}@media print{.document{max-width:none;padding:0}.document-grid{display:block}.document-grid aside{display:none}article{max-width:none}article :global(section){break-inside:auto}article :global(.document-actions){display:none}}
</style>
