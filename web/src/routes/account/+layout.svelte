<script lang="ts">
	import { page } from '$app/state';
	import AuthGate from '$lib/components/AuthGate.svelte';
	import PortalNav from '$lib/components/PortalNav.svelte';
	import { signOut } from '$lib/api/client';
	let { children } = $props();
	const links: [string, string][] = [
		['Account settings', '/account'],
		['Business workspace', '/workspace/today'],
		['Personal purchases', '/personal/purchases']
	];
</script>

{#key page.url.pathname}
	<AuthGate area="account settings"
		><PortalNav label="Your account" homeHref="/start" {links} onsignout={signOut} />{@render children()}</AuthGate
	>
{/key}
