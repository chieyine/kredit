<script lang="ts">
	import { page } from '$app/state';
	import ConsumerPurchase from '$lib/components/ConsumerPurchase.svelte';
</script>

<svelte:head><title>Consumer sale — Kredit</title></svelte:head>
{#key page.params.saleID + page.url.search}<ConsumerPurchase
		id={page.params.saleID ?? ''}
		organization={page.url.searchParams.get('organization') ?? ''}
	/>{/key}
