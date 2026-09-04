<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';
	import { formatKobo } from '$lib/money';
	import { productLabel } from '$lib/product-language';

	let data: any = $state(null);
	let error = $state('');
	let notice = $state('');
	let busy = $state('');

	async function loadSale() {
		error = '';
		try {
			const response = await fetch(`/api/v1/buyer/obligations/${page.params.id}`, { credentials: 'include' });
			const result = await response.json().catch(() => ({}));
			if (!response.ok) throw new Error(result.detail ?? 'We could not open this sale.');
			data = result;
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'We could not open this sale. Check your connection and try again.';
		}
	}
	onMount(() => { void loadSale(); });

	async function acknowledgeNotice(itemID: string) {
		if (busy) return;
		busy = itemID;
		error = '';
		notice = '';
		try {
			const response = await fetch(`/api/v1/buyer/schedule-items/${itemID}/collection-notice/acknowledge`, {
				method: 'POST', credentials: 'include',
				headers: { 'Idempotency-Key': idempotencyKey(), ...csrfHeaders() }
			});
			if (!response.ok) {
				const result = await response.json().catch(() => ({}));
				throw new Error(result.detail ?? 'We could not save your acknowledgement.');
			}
			notice = 'Notice acknowledged. This does not confirm a payment or waive your right to report a problem.';
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'We could not confirm the result. Check your connection and try again.';
		} finally {
			busy = '';
		}
	}

	const money = formatKobo;
</script>

<svelte:head><title>Money you owe — Kredit</title></svelte:head>

<main class="shell workspace">
	<p class="eyebrow">Money I owe</p>
	{#if data}
		<h1>{data.view.request.goods_description}</h1>
		{#if notice}<p class="notice" role="status">{notice}</p>{/if}
		{#if error}<p role="alert">{error}</p>{/if}
		<section class="summary">
			<article><span>Money left</span><strong>{money(data.view.obligation.outstanding_kobo)}</strong></article>
			<article><span>Now</span><strong>{productLabel(data.view.obligation.payment_status)}</strong></article>
			<article><span>Pay before</span><strong>{data.schedule_items.find((i:any)=>i.state!=='CANCELLED'&&i.principal_due_kobo>i.allocated_kobo)?new Date(data.schedule_items.find((i:any)=>i.state!=='CANCELLED'&&i.principal_due_kobo>i.allocated_kobo).due_at).toLocaleDateString('en-NG',{timeZone:'Africa/Lagos'}):'No scheduled payment due'}</strong></article>
		</section>
		<h2>When to pay</h2>
		<p><a href="/buyer/amendments">Review repayment date changes and your acceptance history →</a></p>
		{#if data.schedule_items.length}
			<div class="table"><table><thead><tr><th>Pay before</th><th>Money to pay</th><th>Money paid</th><th>Now</th><th>Debit notice</th></tr></thead><tbody>
				{#each data.schedule_items as item}
					<tr><td>{new Date(item.due_at).toLocaleDateString('en-NG', { timeZone: 'Africa/Lagos' })}</td><td>{money(item.principal_due_kobo)}</td><td>{money(item.allocated_kobo)}</td><td>{productLabel(item.state)}</td><td>{#if item.state !== 'PAID' && item.state !== 'CANCELLED' && new Date(item.collection_at).getTime() <= Date.now()}<button class="secondary" disabled={Boolean(busy)} onclick={() => acknowledgeNotice(item.id)}>{busy === item.id ? 'Saving…' : 'Confirm notice received'}</button>{:else}Not needed{/if}</td></tr>
				{/each}
			</tbody></table></div>
		{:else}<p>You will pay once.</p>{/if}
		<h2>Your payments</h2>
		<p>{data.payments.length} {data.payments.length===1?'payment':'payments'} saved. {data.payment_claims.length} waiting for the seller to check.</p>
		<a href={`/buyer/credit-requests/${data.view.request.id}`}>Pay or tell us about a problem →</a>
	{:else if error}<h1>We could not open this sale.</h1><p role="alert">{error}</p><button type="button" onclick={loadSale}>Try again</button>
	{:else}<p>Opening your sale…</p>{/if}
</main>

<style>
	.summary{display:grid;grid-template-columns:repeat(3,1fr);gap:1rem}.summary article{display:grid;gap:.5rem;padding:1rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}.summary span,th{color:var(--color-muted)}.table{overflow-x:auto}table{width:100%;border-collapse:collapse;background:var(--color-surface)}th,td{padding:.8rem;text-align:left;border-bottom:1px solid var(--color-border)}.notice{padding:.8rem;border-radius:.75rem;background:#e6f7ed;color:var(--color-positive)}@media(max-width:700px){.summary{grid-template-columns:1fr}}
</style>
