<script lang="ts">
 import { page } from '$app/state';
  import AuthGate from '$lib/components/AuthGate.svelte';
  import PortalNav from '$lib/components/PortalNav.svelte';
  import { signOut } from '$lib/api/client';
  let { children } = $props();
  const links: [string,string][] = [['My purchases','/personal/purchases'],['Business workspace','/workspace/today'],['Account settings','/account'],['Help','/legal/complaints']];
</script>
{#key page.url.pathname}
<AuthGate area="personal purchases">
  <PortalNav label="Personal purchases" homeHref="/personal/purchases" {links} onsignout={signOut} />
  {@render children()}
</AuthGate>

{/key}
