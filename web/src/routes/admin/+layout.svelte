<script lang="ts">
	import PortalNav from '$lib/components/PortalNav.svelte';
	import AuthGate from '$lib/components/AuthGate.svelte';
	import CommandPalette from '$lib/components/CommandPalette.svelte';
	import { signOut } from '$lib/api/client';
	import { page } from '$app/state';
	let { children } = $props();
	let paletteOpen=$state(false);
	const links: [string,string][] = [['Overview','/admin'],['Users','/admin/users'],['Businesses','/admin/organizations'],['Money','/admin/money'],['Support cases','/admin/cases'],['Disputes','/admin/disputes'],['Application evidence','/admin/analytics'],['Find a reference','/admin/search'],['Jobs','/admin/jobs'],['Provider events','/admin/provider-events'],['Diagnostics','/admin/diagnostics'],['Account recovery','/admin/recovery'],['Privacy requests','/admin/privacy'],['Protected controls','/admin/controls'],['Admin team','/admin/team'],['Audit history','/admin/audit']];
	const mobilePrimary:[string,string,string][]=[['Overview','/admin','home'],['Users','/admin/users','customers'],['Businesses','/admin/organizations','sales'],['Money','/admin/money','payments']];
	const mobileMore:[string,string,string][]=[
		['Support cases','/admin/cases','Customer support'],['Disputes','/admin/disputes','Customer support'],['Account recovery','/admin/recovery','Customer support'],['Privacy requests','/admin/privacy','Customer support'],
		['Application evidence','/admin/analytics','Operations'],['Find a reference','/admin/search','Operations'],['Jobs','/admin/jobs','Operations'],['Provider events','/admin/provider-events','Operations'],['Diagnostics','/admin/diagnostics','Operations'],
		['Protected controls','/admin/controls','Access and control'],['Admin team','/admin/team','Access and control'],['Audit history','/admin/audit','Access and control']
	];
</script>
<svelte:head><title>Kredit admin</title></svelte:head>
{#key page.url.pathname}<AuthGate area="admin account"><div class="admin-shell"><PortalNav label="Admin account" homeHref="/admin" {links} {mobilePrimary} {mobileMore} dark onsearch={()=>paletteOpen=true} onsignout={signOut}/><div class="admin-content">{@render children()}</div></div><CommandPalette {links} bind:open={paletteOpen}/></AuthGate>{/key}
<style>.admin-shell{min-height:100vh;background:#f1eee6}.admin-content{min-height:calc(100vh - 4rem);padding-bottom:4rem}@media(max-width:760px){.admin-content{padding-bottom:6rem}}</style>
