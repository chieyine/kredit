<script lang="ts">
	import PortalNav from '$lib/components/PortalNav.svelte';
	import AuthGate from '$lib/components/AuthGate.svelte';
	import CommandPalette from '$lib/components/CommandPalette.svelte';
	import ConnectivityBanner from '$lib/components/ConnectivityBanner.svelte';
	import { signOut } from '$lib/api/client';
	import { page } from '$app/state';
	let { children } = $props();
	let paletteOpen = $state(false);
	const links: [string, string][] = [
		['Overview', '/admin'],
		['Engineering tools', '/admin/tools'],
		['Launch setup', '/admin/setup'],
		['Approval queue', '/admin/inbox'],
		['Financial approvals', '/admin/approvals'],
		['Change history', '/admin/history'],
		['Needs attention', '/admin/attention'],
		['Users', '/admin/users'],
		['Businesses', '/admin/organizations'],
		['Money movement', '/admin/money'],
		['Support cases', '/admin/cases'],
		['Disputes', '/admin/disputes'],
		['Analytics evidence', '/admin/analytics'],
		['Find a reference', '/admin/search'],
		['Background jobs', '/admin/jobs'],
		['Provider events', '/admin/provider-events'],
		['Reconciliation reviews', '/admin/reconciliation'],
		['Provider work', '/admin/provider-work'],
		['Message recovery', '/admin/message-submissions'],
		['Business bank review', '/admin/settlement-review'],
		['Seller settlements', '/admin/seller-settlements'],
		['Fee billing', '/admin/billing'],
		['Field agents', '/admin/agents'],
		['Consumer purchases', '/admin/consumer-sales'],
		['System diagnostics', '/admin/diagnostics'],
		['Mono integration', '/admin/mono'],
		['Registration recovery', '/admin/customer-registrations'],
		['Verification recovery', '/admin/verification-requests'],
		['Mandate recovery', '/admin/mandate-authorizations'],
		['Account recovery', '/admin/recovery'],
		['Privacy requests', '/admin/privacy'],
		['Website content', '/admin/website'],
		['Platform settings', '/admin/platform-settings'],
		['Business settings', '/admin/settings'],
		['Protected controls', '/admin/controls'],
		['Admin access', '/admin/team'],
		['Audit trail', '/admin/audit']
	];
	// Five things you check daily. Everything else sits under "Engineering tools".
	const mobilePrimary: [string, string, string][] = [
		['Needs attention', '/admin/attention', 'home'],
		['Businesses', '/admin/organizations', 'customers'],
		['Money', '/admin/money', 'payments'],
		['Problems', '/admin/disputes', 'sales'],
		['Settings', '/admin/platform-settings', 'limits']
	];
	const mobileMore: [string, string, string][] = [
		['Overview', '/admin', 'Needs attention'],
		['Approval queue', '/admin/inbox', 'Needs attention'],
		['Financial approvals', '/admin/approvals', 'Needs attention'],
		['Verification requests', '/admin/verification-requests', 'Needs attention'],
		['Users', '/admin/users', 'Businesses'],
		['Customer registrations', '/admin/customer-registrations', 'Businesses'],
		['Consumer purchases', '/admin/consumer-sales', 'Businesses'],
		['Reconciliation', '/admin/reconciliation', 'Money'],
		['Seller settlements', '/admin/seller-settlements', 'Money'],
		['Settlement review', '/admin/settlement-review', 'Money'],
		['Support cases', '/admin/cases', 'Problems'],
		['Admin team', '/admin/team', 'Settings'],
		['Engineering tools', '/admin/tools', 'Settings']
	];
</script>

<svelte:head><title>Kredit admin</title></svelte:head>
{#key page.url.pathname}<AuthGate area="admin account"
		><div class="admin-shell">
			<ConnectivityBanner /><PortalNav
				label="Admin account"
				homeHref="/admin"
				{links}
				{mobilePrimary}
				{mobileMore}
				dark
				onsearch={() => (paletteOpen = true)}
				onsignout={signOut}
			/>
			<div class="admin-content">{@render children()}</div>
		</div>
		<CommandPalette {links} bind:open={paletteOpen} /></AuthGate
	>{/key}

<style>
	.admin-shell {
		min-height: 100vh;
		background: var(--color-background);
	}
	.admin-content {
		min-height: calc(100vh - 4rem);
		padding-bottom: 4rem;
	}
	@media (max-width: 760px) {
		.admin-content {
			padding-bottom: 6rem;
		}
	}
</style>
