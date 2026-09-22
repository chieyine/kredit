<script lang="ts">
	import { page } from '$app/state';
	import { workspaceHref, workspaceSections } from '$lib/workspace-navigation';
	const section = $derived(
		workspaceSections.find(
			(item) =>
				page.url.pathname === item.root ||
				page.url.pathname.startsWith(item.root + '/') ||
				item.links.some(([, href]) => page.url.pathname === href)
		)
	);
</script>

{#if section}<nav class="section-navigation shell" aria-label={section.label}>
		{#each section.links as [label, href]}<a
				href={workspaceHref(href, page.url)}
				aria-current={page.url.pathname === href ? 'page' : undefined}>{label}</a
			>{/each}
	</nav>{/if}

<style>
	.section-navigation {
		display: flex;
		gap: 0.4rem;
		overflow-x: auto;
		padding-block: 0.8rem;
		border-bottom: 1px solid var(--color-border);
		scrollbar-width: thin;
	}
	.section-navigation a {
		display: inline-flex;
		align-items: center;
		min-height: 2.75rem;
		padding: 0.4rem 0.8rem;
		white-space: nowrap;
		text-decoration: none;
		color: var(--color-muted);
		font-size: 0.9rem;
	}
	.section-navigation a[aria-current='page'] {
		color: var(--color-foreground);
		background: transparent;
		font-weight: 700;
		box-shadow: inset 0 -2px 0 var(--color-accent);
	}
	.section-navigation a:hover {
		color: var(--color-foreground);
	}
</style>
