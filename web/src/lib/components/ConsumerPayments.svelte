<script lang="ts">
	import { LatestRequest, publicError, type Resource } from '$lib/api/reliable';
	import { readPurchaseSummaries, type PurchaseSummary } from '$lib/purchase-summary';
	import Money from './Money.svelte';
	import ResourceNotice from './ResourceNotice.svelte';
	let { organizationID }: { organizationID: string } = $props();
	let purchases = $state<Resource<PurchaseSummary[]>>({ state: 'loading', scope: '' });
	const reads = new LatestRequest();
	const items = $derived(purchases.state === 'ready' ? purchases.data : []);
	const received = $derived(items.reduce((sum, sale) => sum + BigInt(sale.paid_kobo) - BigInt(sale.refunded_kobo), 0n));
	const reported = $derived(items.filter((sale) => sale.pending_claim_count > 0));
	async function load(scope: string) {
		const request = reads.begin();
		purchases = { state: 'loading', scope };
		try {
			const data = await readPurchaseSummaries(scope, request.signal);
			if (request.current()) purchases = { state: 'ready', scope, data, checkedAt: new Date().toISOString() };
		} catch (cause) {
			if (request.current())
				purchases = { state: 'error', scope, status: 0, message: publicError(cause, 'payments from individuals') };
		}
	}
	$effect(() => {
		if (organizationID) void load(organizationID);
		return () => reads.cancel();
	});
</script>

<section class="personal-payments" aria-labelledby="personal-payments-title">
	<h2 id="personal-payments-title">Payments from individuals</h2>
	<ResourceNotice resource={purchases} label="Personal purchase payments" retry={() => load(organizationID)} />
	{#if purchases.state === 'ready'}
		<p>Confirmed receipts after reversals and refunds: <strong><Money amountKobo={received} /></strong></p>
		{#if reported.length}
			<p>Check your bank before confirming these reported payments.</p>
			<ul>
				{#each reported as sale (sale.id)}
					<li>
						<a
							href={`/workspace/sales/consumers/${encodeURIComponent(sale.id)}?organization=${encodeURIComponent(organizationID)}`}
						>
							{sale.customer_name || sale.target} · {sale.terms.item} · {sale.pending_claim_count} to check
						</a>
					</li>
				{/each}
			</ul>
		{:else}<p>No personal purchase payments are waiting for confirmation.</p>{/if}
	{/if}
	<a href={`/workspace/sales/consumers?organization=${encodeURIComponent(organizationID)}`}
		>Open personal sales and payment records →</a
	>
</section>

<style>
	.personal-payments {
		margin-block: 2rem;
		padding-block: 1.5rem;
		border-top: 1px solid var(--color-border);
	}
	h2 {
		font-size: 1.3rem;
	}
	p {
		line-height: 1.6;
	}
	a {
		display: inline-flex;
		align-items: center;
		min-height: 44px;
	}
</style>
