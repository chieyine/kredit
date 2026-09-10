<script lang="ts">
	import PortalNav from '$lib/components/PortalNav.svelte';
	import AuthGate from '$lib/components/AuthGate.svelte';
	import CommandPalette from '$lib/components/CommandPalette.svelte';
	import ConnectivityBanner from '$lib/components/ConnectivityBanner.svelte';
	import { signOut } from '$lib/api/client';
	import { page } from '$app/state';
	let { children } = $props();
	let paletteOpen=$state(false);
	const links: [string,string][] = [['Overview','/admin'],['Approval queue','/admin/inbox'],['Financial approvals','/admin/approvals'],['Change history','/admin/history'],['Needs attention','/admin/attention'],['Users','/admin/users'],['Businesses','/admin/organizations'],['Money movement','/admin/money'],['Support cases','/admin/cases'],['Disputes','/admin/disputes'],['Analytics evidence','/admin/analytics'],['Find a reference','/admin/search'],['Background jobs','/admin/jobs'],['Provider events','/admin/provider-events'],['Reconciliation reviews','/admin/reconciliation'],['System diagnostics','/admin/diagnostics'],['Mono integration','/admin/mono'],['Registration recovery','/admin/customer-registrations'],['Account recovery','/admin/recovery'],['Privacy requests','/admin/privacy'],['Website content','/admin/website'],['Platform settings','/admin/platform-settings'],['Business settings','/admin/settings'],['Protected controls','/admin/controls'],['Admin access','/admin/team'],['Audit trail','/admin/audit']];
	const mobilePrimary:[string,string,string][]=[['Overview','/admin','home'],['Users','/admin/users','customers'],['Businesses','/admin/organizations','sales'],['Money','/admin/money','payments']];
	const mobileMore:[string,string,string][]=[
		['Approval queue','/admin/inbox','Operations'],['Financial approvals','/admin/approvals','Operations'],['Change history','/admin/history','Operations'],['Needs attention','/admin/attention','Operations'],['Support cases','/admin/cases','Customer support'],['Disputes','/admin/disputes','Customer support'],['Account recovery','/admin/recovery','Customer support'],['Privacy requests','/admin/privacy','Customer support'],
		['Analytics evidence','/admin/analytics','Operations'],['Find a reference','/admin/search','Operations'],['Background jobs','/admin/jobs','Operations'],['Provider events','/admin/provider-events','Operations'],['Reconciliation reviews','/admin/reconciliation','Operations'],['System diagnostics','/admin/diagnostics','Operations'],['Mono integration','/admin/mono','Operations'],['Registration recovery','/admin/customer-registrations','Operations'],
		['Website content','/admin/website','Access and control'],['Platform settings','/admin/platform-settings','Access and control'],['Business settings','/admin/settings','Access and control'],['Protected controls','/admin/controls','Access and control'],['Admin access','/admin/team','Access and control'],['Audit trail','/admin/audit','Access and control']
	];
</script>
<svelte:head><title>Kredit admin</title></svelte:head>
{#key page.url.pathname}<AuthGate area="admin account"><div class="admin-shell"><ConnectivityBanner/><PortalNav label="Admin account" homeHref="/admin" {links} {mobilePrimary} {mobileMore} dark onsearch={()=>paletteOpen=true} onsignout={signOut}/><div class="admin-content">{@render children()}</div></div><CommandPalette {links} bind:open={paletteOpen}/></AuthGate>{/key}
<style>.admin-shell{min-height:100vh;background:#f1eee6}.admin-content{min-height:calc(100vh - 4rem);padding-bottom:4rem}@media(max-width:760px){.admin-content{padding-bottom:6rem}}</style>
