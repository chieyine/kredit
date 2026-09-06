<script lang="ts">
	import { signOut } from '$lib/api/client';
	import { onMount } from 'svelte';
	import CommandPalette from '$lib/components/CommandPalette.svelte';
	import ConnectivityBanner from '$lib/components/ConnectivityBanner.svelte';
	import AuthGate from '$lib/components/AuthGate.svelte';
	import PortalNav from '$lib/components/PortalNav.svelte';
	import { page } from '$app/state';
	let { children } = $props();
	let paletteOpen = $state(false);
	let ready = $state(false);
	onMount(() => { ready = true; });
	const links: [string, string][] = [
		['Dashboard', '/app/overview'], ['Search records', '/app/search'], ['Complete setup', '/app/onboarding'], ['Add credit sale', '/app/credit/quick'], ['Customers', '/app/customers'],
		['Customer limits', '/app/trade-lines'], ['Payments received', '/app/payments'], ['Collections', '/app/collections'],
		['Disputes', '/app/disputes'], ['Overdue balances', '/app/overdue'], ['Reports', '/app/reports'],
		['Team access', '/app/team'], ['Activity log', '/app/activity'], ['Message history', '/app/notifications'],
		['Account security', '/app/settings/security'], ['Notification settings', '/app/settings/notifications'], ['Privacy & data', '/app/settings/privacy'],
		['Fees & billing', '/app/settings/billing'], ['Settlement account', '/app/settings/settlement'],
		['Help & support', '/app/help']
	];
	const mobilePrimary: [string, string, string][] = [
		['Dashboard', '/app/overview', 'home'],
		['Add sale', '/app/credit/quick', 'add'],
		['Customers', '/app/customers', 'customers'],
		['Payments', '/app/payments', 'payments']
	];
	const mobileMore: [string, string, string][] = [
		['Search records', '/app/search', 'Sales and money'],
		['Collections', '/app/collections', 'Sales and money'],
		['Disputes', '/app/disputes', 'Sales and money'],
		['Overdue balances', '/app/overdue', 'Sales and money'],
		['Reports', '/app/reports', 'Sales and money'],
		['Customer limits', '/app/trade-lines', 'Sales and money'],
		['Team access', '/app/team', 'Your business'],
		['Activity log', '/app/activity', 'Your business'],
		['Message history', '/app/notifications', 'Your business'],
		['Settings', '/app/settings', 'Account and help'],
		['Help & support', '/app/help', 'Account and help']
	];
</script>

<svelte:head><title>Seller workspace — Kredit</title></svelte:head>
{#if page.url.pathname === '/app'}
	{@render children()}
{:else}
	{#key page.url.pathname}<AuthGate area="seller account">
		<div class="app-shell">
			<ConnectivityBanner />
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
