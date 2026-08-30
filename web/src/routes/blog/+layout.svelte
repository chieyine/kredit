<script lang="ts">
	let { children } = $props();
</script>

<main class="shell prose-page">{@render children()}</main>

<style>
	main :global(h1) { max-width: 20ch; }
	main :global(article) { max-width: 46rem; }
	main :global(article h2) { letter-spacing: -0.03em; margin-top: 2.5rem; font-size: 1.6rem; }
	main :global(article p), main :global(article li) { color: var(--color-muted); line-height: 1.7; }
	main :global(article strong), main :global(article li::marker) { color: var(--color-foreground); }
	main :global(.back) { display: inline-block; margin-bottom: 1.5rem; color: var(--color-primary); font-weight: 700; text-decoration: none; }
	main :global(ul), main :global(ol) { padding-left: 1.25rem; }
</style>
