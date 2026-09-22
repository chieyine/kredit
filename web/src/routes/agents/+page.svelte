<script lang="ts">
	import AuthGate from '$lib/components/AuthGate.svelte';
	import PortalNav from '$lib/components/PortalNav.svelte';
	import ConnectivityBanner from '$lib/components/ConnectivityBanner.svelte';
	import DSA from '$lib/components/DSA.svelte';
	import { signOut } from '$lib/api/client';
	const links: [string, string][] = [
		['Your rewards', '/agents'],
		['Account safety', '/account/security'],
		['Your information', '/account/privacy'],
		['Get help', '/workspace/help']
	];
</script>

<svelte:head><title>Field agent rewards — Kredit</title></svelte:head>
<AuthGate area="field agent account"
	><ConnectivityBanner /><PortalNav label="Field agent account" homeHref="/agents" {links} onsignout={signOut} />
	<div class="product-route"><DSA /></div></AuthGate
>
