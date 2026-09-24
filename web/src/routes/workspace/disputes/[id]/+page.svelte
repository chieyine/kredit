<script lang="ts">
	import { page } from '$app/state';
	import DisputeDetail from '$lib/components/DisputeDetail.svelte';
	import { checkedJSON, publicError, rows } from '$lib/api/reliable';
	import { organization } from '$lib/records';
	// A link from a notice or a bookmark may not name the business. Fall back to
	// the account's first business, as the sale page does, rather than refusing.
	const requested = $derived(page.url.searchParams.get('organization') ?? '');
	let fallback = $state(''),
		error = $state(''),
		loading = $state(false);
	const organizationID = $derived(requested || fallback);
	$effect(() => {
		if (requested || fallback) return;
		loading = true;
		checkedJSON('/api/v1/organizations', rows('organizations', organization))
			.then((items) => {
				fallback = items[0]?.id ?? '';
				if (!fallback) error = 'Add your business first, then open this problem.';
			})
			.catch((cause) => (error = publicError(cause, 'your business')))
			.finally(() => (loading = false));
	});
</script>

<svelte:head><title>Problem details — Kredit</title></svelte:head>
<main class="shell workspace">
	<p class="eyebrow">Reported problem</p>
	<h1>A problem your customer reported</h1>
	<p class="lede">
		Add your side and any proof below. If the customer is right, reduce what they owe with a credit note from the sale’s
		“Deliveries and credit notes” page. Kredit support reviews the problem and records the decision.
	</p>
	{#if organizationID}<DisputeDetail
			endpoint={`/api/v1/organizations/${encodeURIComponent(organizationID)}/disputes/${encodeURIComponent(page.params.id ?? '')}`}
			backHref={`/workspace/disputes?organization=${encodeURIComponent(organizationID)}`}
		/>{:else if loading}<p role="status">Opening this problem…</p>{:else if error}<p class="error" role="alert">
			{error}
		</p>{/if}
</main>
