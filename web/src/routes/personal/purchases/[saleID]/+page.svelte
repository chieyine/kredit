<script lang="ts">
	import { page } from '$app/state';
	import ConsumerPurchase from '$lib/components/ConsumerPurchase.svelte';
</script>

<svelte:head><title>Your purchase — Kredit</title></svelte:head>
{#key page.params.saleID}<ConsumerPurchase id={page.params.saleID ?? ''} />{/key}
