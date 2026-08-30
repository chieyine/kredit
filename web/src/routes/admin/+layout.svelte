<script lang="ts">
	import PortalNav from '$lib/components/PortalNav.svelte';
	import AuthGate from '$lib/components/AuthGate.svelte';
	import { page } from '$app/state';
	let { children } = $props();
	const links: [string,string][] = [['Overview','/admin'],['Pilot scorecard','/admin/analytics'],['Search','/admin/search'],['Jobs','/admin/jobs'],['Provider events','/admin/provider-events'],['Controls','/admin/controls'],['Diagnostics','/admin/diagnostics'],['Recovery','/admin/recovery'],['Privacy','/admin/privacy'],['Audit','/admin/audit']];
</script>
<svelte:head><title>Operations — Kredit</title></svelte:head>
{#key page.url.pathname}<AuthGate area="operations account"><div class="admin-shell"><PortalNav label="Operations navigation" homeHref="/admin" {links} dark /><main>{@render children()}</main></div></AuthGate>{/key}
<style>.admin-shell{min-height:100vh}</style>
