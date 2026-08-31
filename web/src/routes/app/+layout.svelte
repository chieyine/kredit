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
		['Your staff', '/app/team'], ['Business activity', '/app/activity'], ['Message history', '/app/notifications'],
		['Account safety', '/app/settings/security'], ['Message choices', '/app/settings/notifications'], ['Your details', '/app/settings/privacy'],
		['Kredit fees', '/app/settings/billing'], ['Bank account', '/app/settings/settlement'],
		['Get help', '/app/help']
	];
	const mobilePrimary: [string, string, string][] = [
		['Home', '/app/overview', 'home'],
		['Add sale', '/app/credit/new', 'add'],
		['Customers', '/app/customers', 'customers'],
		['Payments', '/app/payments', 'payments']
	];
	const mobileMore: [string, string, string][] = [
		['Money owed', '/app/collections', 'Sales and money'],
		['Problems', '/app/disputes', 'Sales and money'],
		['Reports', '/app/reports', 'Sales and money'],
		['Customer limits', '/app/trade-lines', 'Sales and money'],
		['Your staff', '/app/team', 'Your business'],
		['Business activity', '/app/activity', 'Your business'],
		['Message history', '/app/notifications', 'Your business'],
		['Settings', '/app/settings', 'Account and help'],
		['Get help', '/app/help', 'Account and help']
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
