<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import type { paths } from '$lib/api/generated/schema';
	import Money from '$lib/components/Money.svelte';
	import { productLabel } from '$lib/product-language';
	type BuyerHistory = paths['/buyer/history']['get']['responses'][200]['content']['application/json'];
	let history: BuyerHistory | null = null;
	let error = '';
	onMount(async () => {
		const { data } = await api.GET('/buyer/history');
		if (!data) { error = 'Sign in to view your factual trade history.'; return; }
		history = data;
	});
</script>

<svelte:head><title>Factual trade history — Kredit</title></svelte:head>
<main class="shell">
	<p class="eyebrow">Your history</p><h1>See how you paid before</h1>
	<p class="intro">This page shows your sales and payments. Kredit does not give you a secret score.</p>
	{#if history}
		<section class="grid">
			<article><span>Sales fully paid</span><strong>{history.completed_obligations}</strong></article>
			<article><span>Paid by due date</span><strong>{history.on_time_count ?? 0} ({(history.on_time_percentage ?? 0).toFixed(0)}%)</strong></article>
			<article><span>Sales still open</span><strong>{history.active_obligations}</strong></article>
			<article><span>Open problems</span><strong>{history.dispute_count}</strong></article>
		</section>
		<h2>Your sales</h2>
		<div class="table-wrap"><table><thead><tr><th>Seller</th><th>Money</th><th>Now</th><th>Pay before</th></tr></thead><tbody>
			{#each history.obligations as item}<tr><td>{item.buyer_name}</td><td><Money amountKobo={Number(item.principal_kobo)} /></td><td>{productLabel(item.payment_status)}</td><td>{new Date(String(item.due_date)).toLocaleDateString('en-NG')}</td></tr>{/each}
		</tbody></table></div>
	{:else if error}<p class="error" role="alert">{error}</p>{:else}<p>Loading your history…</p>{/if}
</main>

<style>
	.eyebrow { color: #2738d6; font-weight: 700; text-transform: uppercase; letter-spacing: .08em; font-size: .78rem; }
	h1 { font-size: clamp(2.5rem, 7vw, 5rem); line-height: 1; letter-spacing: -.055em; max-width: 10ch; }
	.intro { max-width: 42rem; color: var(--color-muted); }
	.grid { display: grid; grid-template-columns: repeat(4, minmax(0,1fr)); gap: 1rem; margin: 2.5rem 0; }
	article { display: grid; gap: .75rem; padding: 1.25rem; border: 1px solid var(--color-border); border-radius: 1rem; background: var(--color-surface); } article span { color: var(--color-muted); } article strong { font-size: 1.35rem; }
	h2 { margin-top: 2.5rem; }.table-wrap { overflow-x: auto; } table { width: 100%; border-collapse: collapse; background: var(--color-surface); } th, td { text-align: left; padding: .9rem; border-bottom: 1px solid var(--color-border); } th { color: var(--color-muted); font-size: .8rem; text-transform: uppercase; }
	.error { color: #b42318; } @media (max-width: 760px) { .grid { grid-template-columns: repeat(2, minmax(0,1fr)); } }
</style>
