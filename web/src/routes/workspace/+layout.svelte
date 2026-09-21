<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { signOut } from '$lib/api/client';
	import AuthGate from '$lib/components/AuthGate.svelte';
	import PortalNav from '$lib/components/PortalNav.svelte';
	import CommandPalette from '$lib/components/CommandPalette.svelte';
	import ConnectivityBanner from '$lib/components/ConnectivityBanner.svelte';
	import BankReturnNotice from '$lib/components/BankReturnNotice.svelte';
	import WorkspaceSectionNav from '$lib/components/WorkspaceSectionNav.svelte';
	import {
		workspaceLinks,
		workspacePrimary,
		workspaceMore,
		workspaceHref,
		workspaceSections
	} from '$lib/workspace-navigation';
	let { children } = $props();
	let searchReady = $state(false),
		paletteOpen = $state(false);
	const links = $derived(
		workspaceLinks.map(([label, href]) => [label, workspaceHref(href, page.url)] as [string, string])
	);
	const primary = $derived(
		workspacePrimary.map(
			([label, href, icon]) => [label, workspaceHref(href, page.url), icon] as [string, string, string]
		)
	);
	const more = $derived(
		workspaceMore.map(
			([label, href, group]) => [label, workspaceHref(href, page.url), group] as [string, string, string]
		)
	);
	const searchLinks = $derived([
		...new Map(
			[...workspaceLinks, ...workspaceSections.flatMap((section) => section.links)].map(([label, href]) => [
				href,
				[label, workspaceHref(href, page.url)] as [string, string]
			])
		).values()
	]);
	onMount(() => {
		searchReady = true;
	});
</script>

<svelte:head><title>Business workspace — Kredit</title></svelte:head>
{#key page.url.pathname}
	<AuthGate area="business workspace">
		<div class="workspace-shell">
			<ConnectivityBanner />
			<PortalNav
				label="Business workspace"
				homeHref={workspaceHref('/workspace/today', page.url)}
				{links}
				mobilePrimary={primary}
				mobileMore={more}
				onsearch={() => (paletteOpen = true)}
				onsignout={signOut}
				{searchReady}
			/>
			<WorkspaceSectionNav />
			<div class="portal-content">
				<BankReturnNotice />{#key page.url.pathname + page.url.search}{@render children()}{/key}
			</div>
		</div>
		<CommandPalette links={searchLinks} bind:open={paletteOpen} />
	</AuthGate>
{/key}

<style>
	.workspace-shell {
		min-height: 100svh;
	}
	@media (max-width: 760px) {
		.portal-content {
			padding-bottom: 5.5rem;
		}
	}
</style>
