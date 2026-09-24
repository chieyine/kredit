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
	// A record page belongs to the tab whose address is the longest match, so
	// an order counts as "Offers" and a sale as "Business sales".
	const active = $derived(
		section?.links
			.map(([, href]) => href)
			.filter((href) => page.url.pathname === href || page.url.pathname.startsWith(href + '/'))
			.sort((a, b) => b.length - a.length)[0]
	);
	// "page" only for the page itself; a record inside a tab marks it "true".
	const current = (href: string) =>
		href === page.url.pathname ? ('page' as const) : href === active ? ('true' as const) : undefined;
	let navigation: HTMLElement | undefined = $state();
	// On a phone the tabs scroll sideways; bring the current one into view so
	// the reader can see where they are without swiping to find it.
	$effect(() => {
		void page.url.pathname;
		const tab = navigation?.querySelector<HTMLElement>('[aria-current]');
		if (!navigation || !tab || navigation.scrollWidth <= navigation.clientWidth) return;
		navigation.scrollLeft = tab.offsetLeft - (navigation.clientWidth - tab.offsetWidth) / 2;
	});
</script>

{#if section}<nav class="section-navigation shell" aria-label={section.label} bind:this={navigation}>
		{#each section.links as [label, href], i (i)}<a href={workspaceHref(href, page.url)} aria-current={current(href)}
				>{label}</a
			>{/each}
	</nav>{/if}

<style>
	.section-navigation {
		/* positioned so a tab's offsetLeft is measured from the bar itself */
		position: relative;
		display: flex;
		gap: 0.4rem;
		overflow-x: auto;
		padding-block: 0.8rem;
		border-bottom: 1px solid var(--color-border);
		scrollbar-width: thin;
	}
	/* With room to spare every tab stays visible; only a phone scrolls them. */
	@media (min-width: 761px) {
		.section-navigation {
			flex-wrap: wrap;
			overflow-x: visible;
		}
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
	.section-navigation a[aria-current] {
		color: var(--color-foreground);
		background: transparent;
		font-weight: 700;
		box-shadow: inset 0 -2px 0 var(--color-accent);
	}
	.section-navigation a:hover {
		color: var(--color-foreground);
	}
</style>
