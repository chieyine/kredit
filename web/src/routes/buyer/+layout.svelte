<script lang="ts">
	import { signOut } from '$lib/api/client';
	import AuthGate from '$lib/components/AuthGate.svelte';
	import PortalNav from '$lib/components/PortalNav.svelte';
	import { page } from '$app/state';
	let { children } = $props();
	const links: [string, string][] = [
		['Overview', '/buyer'], ['Sales to review', '/buyer/requests'], ['What I owe', '/buyer/obligations'],
		['My credit limits', '/buyer/trade-lines'], ['Payment history', '/buyer/history'], ['Payment-date changes','/buyer/amendments'], ['Transfers I reported', '/buyer/payments'], ['Bank debit permission', '/buyer/mandates'],
		['Message history', '/buyer/notifications'], ['Seller access', '/buyer/permissions'],
		['Account settings', '/buyer/settings'], ['Help & complaints', '/legal/complaints']
	];
	const mobilePrimary: [string, string, string][] = [
		['Overview', '/buyer', 'home'],
		['Sales', '/buyer/requests', 'sales'],
		['I owe', '/buyer/obligations', 'owe'],
		['Limits', '/buyer/trade-lines', 'limits']
	];
	const mobileMore: [string, string, string][] = [
		['Payment history', '/buyer/history', 'Sales and payments'], ['Payment-date changes','/buyer/amendments','Sales and payments'],
		['Transfers I reported', '/buyer/payments', 'Sales and payments'],
		['Bank debit permission', '/buyer/mandates', 'Sales and payments'],
		['Message history', '/buyer/notifications', 'Messages and choices'],
		['Seller access', '/buyer/permissions', 'Messages and choices'],
		['Account settings', '/buyer/settings', 'Account and help'],
		['Help & complaints', '/legal/complaints', 'Account and help']
	];
</script>

<svelte:head><title>Customer account — Kredit</title></svelte:head>
{#key page.url.pathname}<AuthGate area="customer account">
	<div class="buyer-shell">
		<PortalNav label="Customer account" homeHref="/buyer" {links} {mobilePrimary} {mobileMore} onsignout={signOut} />
		<div class="portal-content"><div class="motion-scope product-route">{@render children()}</div></div>
	</div>
</AuthGate>{/key}

<style>
	.buyer-shell { min-height: 100vh; }
	@media(max-width:760px){.portal-content{padding-bottom:5.5rem}}
</style>