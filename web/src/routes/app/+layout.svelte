<script lang="ts">
	import { signOut } from '$lib/api/client';
	import { onMount } from 'svelte';
	import CommandPalette from '$lib/components/CommandPalette.svelte';
	import AuthGate from '$lib/components/AuthGate.svelte';
	import PortalNav from '$lib/components/PortalNav.svelte';
	import { page } from '$app/state';
	let { children } = $props();
	let paletteOpen = $state(false);
	let ready = $state(false);
	onMount(() => { ready = true; });
	const links: [string, string][] = [
		['Home', '/app/overview'], ['Finish setup', '/app/onboarding'], ['Add a sale', '/app/credit/new'], ['Customers', '/app/customers'],
		['Customer limits', '/app/trade-lines'], ['Money received', '/app/payments'], ['Late payments', '/app/collections'],
		['Problems', '/app/disputes'], ['Money overdue', '/app/overdue'], ['Reports', '/app/reports'],
		['Your staff', '/app/team'], ['Account safety', '/app/settings/security'],
		['Messages', '/app/settings/notifications'], ['Your details', '/app/settings/privacy'],
		['Kredit fees', '/app/settings/billing'], ['Bank account', '/app/settings/settlement'],
		['Get help', '/legal/complaints']
	];
	const mobilePrimary: [string, string, string][] = [
		['Home', '/app/overview', 'home'],
		['Add sale', '/app/credit/new', 'add'],
		['Customers', '/app/customers', 'customers'],
		['Payments', '/app/payments', 'payments']
	];
	const mobileMore: [string, string][] = [
		['Money owed', '/app/collections'],
		['Problems', '/app/disputes'],
		['Reports', '/app/reports'],
		['Customer limits', '/app/trade-lines'],
		['Your staff', '/app/team'],
		['Settings', '/app/settings'],
		['Get help', '/legal/complaints']
	];
</script>

<svelte:head><title>Workspace — Kredit</title></svelte:head>
{#if page.url.pathname === '/app'}
	{@render children()}
{:else}
	{#key page.url.pathname}<AuthGate area="seller account">
		<div class="app-shell">
			<PortalNav label="Seller account" homeHref="/app/overview" {links} {mobilePrimary} {mobileMore} onsearch={() => (paletteOpen = true)} onsignout={signOut} searchReady={ready} />
			<div class="portal-content">{#key page.url.pathname}<div class="motion-scope product-route">{@render children()}</div>{/key}</div>
		</div>
		<CommandPalette {links} bind:open={paletteOpen} />
	</AuthGate>{/key}
{/if}

<style>
	.app-shell { min-height: 100vh; }
	@media(max-width:760px){.portal-content{padding-bottom:5.5rem}}
</style>
