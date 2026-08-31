<script lang="ts">
	import { onMount } from 'svelte';
	import { api, csrfHeaders, idempotencyKey } from '$lib/api/client';
	import type { paths } from '$lib/api/generated/schema';
	import Money from '$lib/components/Money.svelte';
	import { productLabel } from '$lib/product-language';
	type BuyerHistory = paths['/buyer/history']['get']['responses'][200]['content']['application/json'];
	let history: BuyerHistory | null = $state(null);
	let error = $state('');
	let selectedID=$state(''),reason=$state(''),evidence=$state(''),notice=$state(''),busy=$state(false);
	async function askForCorrection(event:SubmitEvent){event.preventDefault();busy=true;error='';notice='';const response=await fetch('/api/v1/buyer/history/corrections',{method:'POST',credentials:'include',headers:{'Content-Type':'application/json','Idempotency-Key':idempotencyKey(),...csrfHeaders()},body:JSON.stringify({subject_type:'obligation',subject_id:selectedID,source_event_id:'',reason,evidence:evidence.split('\n').map(item=>item.trim()).filter(Boolean)})});const data=await response.json().catch(()=>({}));busy=false;if(!response.ok){error=data.detail??'We could not send your correction request.';return}reason='';evidence='';notice='Your correction request was sent to the seller.'}
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
		<section class="correction"><h2>Is a record wrong?</h2><p>Ask the seller to check a wrong amount, payment or sale detail. Tell them exactly what should change.</p>{#if notice}<p class="notice" role="status">{notice}</p>{/if}<form onsubmit={askForCorrection}><label>Which sale?<select bind:value={selectedID} required><option value="">Choose a sale</option>{#each history.obligations as item}<option value={item.obligation_id}>{item.buyer_name} · {new Date(String(item.due_date)).toLocaleDateString('en-NG')}</option>{/each}</select></label><label>What is wrong?<textarea bind:value={reason} rows="4" required placeholder="For example: I paid ₦50,000 on 28 August, but it is not showing."></textarea></label><label>Proof or reference numbers <small>one on each line, if you have any</small><textarea bind:value={evidence} rows="3"></textarea></label><button disabled={busy||!selectedID||!reason.trim()}>{busy?'Sending…':'Ask for a correction'}</button></form></section>
	{:else if error}<p class="error" role="alert">{error}</p>{:else}<p>Loading your history…</p>{/if}
</main>

<style>
	.eyebrow { color: #2738d6; font-weight: 700; text-transform: uppercase; letter-spacing: .08em; font-size: .78rem; }
	h1 { font-size: clamp(2.5rem, 7vw, 5rem); line-height: 1; letter-spacing: -.055em; max-width: 10ch; }
	.intro { max-width: 42rem; color: var(--color-muted); }
	.grid { display: grid; grid-template-columns: repeat(4, minmax(0,1fr)); gap: 1rem; margin: 2.5rem 0; }
	article { display: grid; gap: .75rem; padding: 1.25rem; border: 1px solid var(--color-border); border-radius: 1rem; background: var(--color-surface); } article span { color: var(--color-muted); } article strong { font-size: 1.35rem; }
	h2 { margin-top: 2.5rem; }.table-wrap { overflow-x: auto; } table { width: 100%; border-collapse: collapse; background: var(--color-surface); } th, td { text-align: left; padding: .9rem; border-bottom: 1px solid var(--color-border); } th { color: var(--color-muted); font-size: .8rem; text-transform: uppercase; }.correction{max-width:42rem;margin-top:2rem;padding:1.2rem;border:1px solid var(--color-border);background:var(--color-surface)}.correction h2{margin-top:0}.correction form,.correction label{display:grid;gap:.4rem}.correction form{gap:.8rem}.correction select,.correction textarea{box-sizing:border-box;width:100%;padding:.7rem;border:1px solid var(--color-border);font:inherit}.correction button{width:max-content;padding:.7rem .9rem}.notice{padding:.7rem;border-left:4px solid var(--color-positive)}
	.error { color: #b42318; } @media (max-width: 760px) { .grid { grid-template-columns: repeat(2, minmax(0,1fr)); } }
</style>
