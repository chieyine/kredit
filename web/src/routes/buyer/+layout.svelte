<script lang="ts">
	import { signOut } from '$lib/api/client';
	import AuthGate from '$lib/components/AuthGate.svelte';
	import PortalNav from '$lib/components/PortalNav.svelte';
	import { page } from '$app/state';
	let { children } = $props();
	const links: [string, string][] = [
		['Home', '/buyer'], ['Sales to check', '/buyer/requests'], ['Money I owe', '/buyer/obligations'],
		['My limits', '/buyer/trade-lines'], ['My history', '/buyer/history'], ['Bank debit', '/buyer/mandates'],
		['My settings', '/buyer/settings'], ['Get help', '/legal/complaints']
	];
	const mobilePrimary: [string, string, string][] = [
		['Home', '/buyer', 'home'],
		['Sales', '/buyer/requests', 'sales'],
		['I owe', '/buyer/obligations', 'owe'],
		['Limits', '/buyer/trade-lines', 'limits']
	];
	const mobileMore: [string, string][] = [
		['My history', '/buyer/history'],
		['Bank debit', '/buyer/mandates'],
		['My settings', '/buyer/settings'],
		['Get help', '/legal/complaints']
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
