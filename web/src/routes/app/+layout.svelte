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
		['Dashboard', '/app/overview'], ['Find anything', '/app/search'], ['Finish setting up', '/app/onboarding'], ['Add a sale', '/app/credit/quick'], ['Customers', '/app/customers'],
		['Customer limits', '/app/trade-lines'], ['Payments received', '/app/payments'], ['Bank debits', '/app/collections'],
		['Problems', '/app/disputes'], ['Money overdue', '/app/overdue'], ['Reports', '/app/reports'],
		['Your staff', '/app/team'], ['Business activity', '/app/activity'], ['Messages we sent', '/app/notifications'],
		['Account safety', '/app/settings/security'], ['Message choices', '/app/settings/notifications'], ['Your information', '/app/settings/privacy'],
		['Kredit fees', '/app/settings/billing'], ['Your bank account', '/app/settings/settlement'],
		['Get help', '/app/help']
	];
	const mobilePrimary: [string, string, string][] = [
		['Dashboard', '/app/overview', 'home'],
		['Add sale', '/app/credit/quick', 'add'],
		['Customers', '/app/customers', 'customers'],
		['Payments', '/app/payments', 'payments']
	];
	const mobileMore: [string, string, string][] = [
		['Find anything', '/app/search', 'Sales and money'],
		['Bank debits', '/app/collections', 'Sales and money'],
		['Problems', '/app/disputes', 'Sales and money'],
		['Money overdue', '/app/overdue', 'Sales and money'],
		['Reports', '/app/reports', 'Sales and money'],
		['Customer limits', '/app/trade-lines', 'Sales and money'],
		['Your staff', '/app/team', 'Your business'],
		['Business activity', '/app/activity', 'Your business'],
		['Messages we sent', '/app/notifications', 'Your business'],
		['Settings', '/app/settings', 'Account and help'],
		['Get help', '/app/help', 'Account and help']
	];
</script>

<svelte:head><title>Your Kredit account</title></svelte:head>
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
