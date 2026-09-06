<script lang="ts">
	import { onMount } from 'svelte';
	import { signOut } from '$lib/api/client';
	import AuthGate from '$lib/components/AuthGate.svelte';
	import CommandPalette from '$lib/components/CommandPalette.svelte';
	import ConnectivityBanner from '$lib/components/ConnectivityBanner.svelte';
	import PortalNav from '$lib/components/PortalNav.svelte';
	import { page } from '$app/state';
	let { children } = $props();
	let paletteOpen = $state(false);
	let searchReady = $state(false);
	onMount(() => { searchReady = true; });
	const links: [string, string][] = [
		['Overview', '/buyer'], ['Sales waiting for me', '/buyer/requests'], ['What I owe', '/buyer/obligations'],
		['My buying limits', '/buyer/trade-lines'], ['How I have paid before', '/buyer/history'], ['Changes to my payment days','/buyer/amendments'], ['Transfers I reported', '/buyer/payments'], ['Bank debit permission', '/buyer/mandates'],
		['Messages Kredit sent me', '/buyer/notifications'], ['What sellers may send me', '/buyer/permissions'],
		['My settings', '/buyer/settings'], ['Get help', '/legal/complaints']
	];
	const mobilePrimary: [string, string, string][] = [
		['Overview', '/buyer', 'home'],
		['Sales', '/buyer/requests', 'sales'],
		['I owe', '/buyer/obligations', 'owe'],
		['Limits', '/buyer/trade-lines', 'limits']
	];
	const mobileMore: [string, string, string][] = [
		['How I have paid before', '/buyer/history', 'Sales and payments'], ['Changes to my payment days','/buyer/amendments','Sales and payments'],
		['Transfers I reported', '/buyer/payments', 'Sales and payments'],
		['Bank debit permission', '/buyer/mandates', 'Sales and payments'],
		['Messages Kredit sent me', '/buyer/notifications', 'Messages and choices'],
		['What sellers may send me', '/buyer/permissions', 'Messages and choices'],
		['My settings', '/buyer/settings', 'Account and help'],
		['Get help', '/legal/complaints', 'Account and help']
	];
</script>

<svelte:head><title>Customer account — Kredit</title></svelte:head>
{#key page.url.pathname}<AuthGate area="customer account">
	<div class="buyer-shell">
		<ConnectivityBanner />
		<PortalNav label="Customer account" homeHref="/buyer" {links} {mobilePrimary} {mobileMore} onsearch={() => (paletteOpen = true)} onsignout={signOut} {searchReady} />
		<div class="portal-content"><div class="motion-scope product-route">{@render children()}</div></div>
	</div>
	<CommandPalette {links} bind:open={paletteOpen} />
</AuthGate>{/key}

<style>
	.buyer-shell { min-height: 100vh; }
	@media(max-width:760px){.portal-content{padding-bottom:5.5rem}}
</style>
