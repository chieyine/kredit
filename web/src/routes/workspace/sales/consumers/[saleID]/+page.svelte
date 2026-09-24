<script lang="ts">
	import { page } from '$app/state';
	import ConsumerPurchase from '$lib/components/ConsumerPurchase.svelte';
	import { checkedJSON, publicError, rows } from '$lib/api/reliable';
	import { organization } from '$lib/records';
	// This is always the seller's view. Without a business in the link the
	// component would read it as the customer's own purchase and refuse, so
	// fall back to the account's first business, as the sale page does.
	const requested = $derived(page.url.searchParams.get('organization') ?? '');
	let fallback = $state(''),
		error = $state('');
	const organizationID = $derived(requested || fallback);
	$effect(() => {
		if (requested || fallback) return;
		checkedJSON('/api/v1/organizations', rows('organizations', organization))
			.then((items) => {
				fallback = items[0]?.id ?? '';
				if (!fallback) error = 'Add your business first, then open this sale.';
			})
			.catch((cause) => (error = publicError(cause, 'your business')));
	});
</script>

<svelte:head><title>Consumer sale — Kredit</title></svelte:head>
{#if organizationID}
	{#key page.params.saleID + organizationID}<ConsumerPurchase
			id={page.params.saleID ?? ''}
			organization={organizationID}
		/>{/key}
{:else if error}<main class="shell workspace"><p class="error" role="alert">{error}</p></main>
{:else}<main class="shell workspace"><p role="status">Opening this sale…</p></main>{/if}
