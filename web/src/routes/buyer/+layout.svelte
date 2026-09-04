<script lang="ts">
	import { signOut } from '$lib/api/client';
	import AuthGate from '$lib/components/AuthGate.svelte';
	import PortalNav from '$lib/components/PortalNav.svelte';
	import { page } from '$app/state';
	let { children } = $props();
	const links: [string, string][] = [
		['Home', '/buyer'], ['Sales to check', '/buyer/requests'], ['Money I owe', '/buyer/obligations'],
		['My limits', '/buyer/trade-lines'], ['My history', '/buyer/history'], ['Date changes','/buyer/amendments'], ['Transfers I reported', '/buyer/payments'], ['Bank debit', '/buyer/mandates'],
		['Message history', '/buyer/notifications'], ['Seller permissions', '/buyer/permissions'],
		['My settings', '/buyer/settings'], ['Get help', '/legal/complaints']
	];
	const mobilePrimary: [string, string, string][] = [
		['Home', '/buyer', 'home'],
		['Sales', '/buyer/requests', 'sales'],
		['I owe', '/buyer/obligations', 'owe'],
		['Limits', '/buyer/trade-lines', 'limits']
	];
	const mobileMore: [string, string, string][] = [
		['My history', '/buyer/history', 'Sales and payments'], ['Date changes','/buyer/amendments','Sales and payments'],
		['Transfers I reported', '/buyer/payments', 'Sales and payments'],
		['Bank debit', '/buyer/mandates', 'Sales and payments'],
		['Message history', '/buyer/notifications', 'Messages and choices'],
		['Seller permissions', '/buyer/permissions', 'Messages and choices'],
		['My settings', '/buyer/settings', 'Account and help'],
		['Get help', '/legal/complaints', 'Account and help']
	];
</script>

<svelte:head><title>Buyer portal — Kredit</title></svelte:head>
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
