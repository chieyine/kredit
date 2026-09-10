<script lang="ts">
	import { onMount } from 'svelte';
	import { checkedJSON, LatestRequest, publicError, record, text } from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';
	import { kobo } from '$lib/records';
	import Money from '$lib/components/Money.svelte';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import { readableDate } from '$lib/datetime';

	type Sale = {
		obligation_id: string;
		supplier_name?: string;
		principal_kobo?: number | string;
		outstanding_kobo?: number | string;
		due_date?: string;
		payment_status?: string;
	};
	type History = {
		active_obligations: number;
		completed_obligations: number;
		dispute_count: number;
		on_time_count?: number;
		on_time_percentage?: number;
		obligations: Sale[];
 corrections?: {id:string;subject_id:string;reason:string;state:string;decisions:{id:string;reason:string;outcome:string;decided_at:string}[]}[];
	};

	let history = $state<History | null>(null);
	let error = $state('');
	let selectedID = $state(''),
		reason = $state(''),
		evidence = $state(''),
		notice = $state(''),
		busy = $state(false);

	function saleLabel(sale: Sale) {
		const seller = sale.supplier_name?.trim() || 'Seller';
		return sale.due_date ? `${seller} · due ${readableDate(sale.due_date)}` : seller;
	}

	const reads = new LatestRequest();
	let correctionIntent: MutationIntent;
	function decodeHistory(value: unknown): History {
		const result = record(value);
		for (const key of ['active_obligations', 'completed_obligations', 'dispute_count', 'on_time_count']) {
			if (!Number.isSafeInteger(result[key]) || Number(result[key]) < 0) throw new Error('Incomplete history totals');
		}
		if (!Array.isArray(result.obligations)) throw new Error('Incomplete sales history');
		const obligations = result.obligations.map(value => {
			const sale = record(value);
			if (!text(sale.obligation_id)) throw new Error('Missing sale reference');
			kobo(sale.principal_kobo); kobo(sale.outstanding_kobo);
			for (const key of ['supplier_name', 'due_date', 'payment_status']) text(sale[key]);
			return sale as Sale;
		});
		const corrections = result.corrections === undefined ? [] : result.corrections;
  if(!Array.isArray(corrections))throw new Error('Incomplete correction history');
  for(const value of corrections){const c=record(value);for(const key of ['id','subject_id','reason','state'])text(c[key]);if(!Array.isArray(c.decisions))throw new Error('Incomplete correction decision');for(const value of c.decisions){const d=record(value);for(const key of ['id','reason','outcome','decided_at'])text(d[key]);if(!Number.isFinite(Date.parse(String(d.decided_at))))throw new Error('Invalid decision date');}}
  return { ...result, obligations,corrections } as History;
	}
	async function load() {
		const read = reads.begin(); error = ''; history = null;
		try {
			const result = await checkedJSON('/api/v1/buyer/history', decodeHistory, { signal: read.signal });
			if (read.current()) history = result;
		} catch (cause) { if (read.current()) error = publicError(cause, 'your trade history'); }
	}
	async function askForCorrection(event: SubmitEvent) {
		event.preventDefault();
		if (busy || !history?.obligations.some(sale => sale.obligation_id === selectedID) || reason.trim().length < 8) return;
		busy = true; error = ''; notice = '';
		try {
			correctionIntent ??= new MutationIntent('buyer-history-correction', '/api/v1/buyer/history/corrections');
			await correctionIntent.run({ subject_type: 'obligation', subject_id: selectedID, source_event_id: '', reason: reason.trim(), evidence: evidence.split('\n').map(item => item.trim()).filter(Boolean) }, value => {
				const correction = record(record(value).correction);
				if (!text(correction.id) || correction.subject_id !== selectedID || correction.state !== 'OPEN') throw new Error('Correction not confirmed');
				return correction;
			});
			reason = ''; evidence = ''; notice = 'Your correction request has been saved for review.'; await load();
		} catch (cause) { error = cause instanceof Error ? cause.message : 'We could not confirm your correction request.'; }
		finally { busy = false; }
	}
	onMount(() => { void load(); return () => reads.cancel(); });
</script>

<svelte:head><title>Your trade history — Kredit</title></svelte:head>

<main class="shell">
	<p class="eyebrow">Your history</p>
	<h1>Your own record, in your own hands</h1>
	<p class="intro">
		This page shows your real sales and real payments. There is no secret score, and no number
		deciding your worth.
	</p>

	{#if history}
		<section class="grid">
			<article><span>Sales fully paid</span><strong>{history.completed_obligations}</strong></article>
			<article><span>Paid on time</span><strong>{history.on_time_count ?? 0} of {history.completed_obligations}</strong></article>
			<article><span>Sales still open</span><strong>{history.active_obligations}</strong></article>
			<article><span>Open problems</span><strong>{history.dispute_count}</strong></article>
		</section>

		<h2>Your sales</h2>
		{#if history.obligations.length}
			<div class="table-wrap">
				<table>
					<thead>
						<tr><th>Seller</th><th>Value of goods</th><th>Left to pay</th><th>Pay before</th><th>Now</th></tr>
					</thead>
					<tbody>
						{#each history.obligations as sale (sale.obligation_id)}
							<tr>
								<td>{sale.supplier_name?.trim() || 'Seller'}</td>
								<td><Money amountKobo={sale.principal_kobo ?? null} /></td>
								<td><Money amountKobo={sale.outstanding_kobo ?? null} /></td>
								<td>{readableDate(sale.due_date) || '—'}</td>
								<td>{#if sale.payment_status}<StatusPill status={sale.payment_status} />{/if}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{:else}
			<p class="none">You have no sales on record yet.</p>
		{/if}

  {#if history.corrections?.length}<section class="correction"><h2>Your correction requests</h2><p>Review decisions stay alongside the original records. Any change to money owed is recorded separately through the payment or financial-change process.</p>{#each history.corrections as item}<article><h3>{item.reason}</h3><p><StatusPill status={item.state}/></p>{#each item.decisions as decision}<p><strong>{decision.outcome==='APPROVED'?'Approved correction note':'Review decision'}</strong> · {readableDate(decision.decided_at)}</p><p>{decision.reason}</p>{/each}</article>{/each}</section>{/if}
		<section class="correction">
			<h2>Is something here wrong?</h2>
			<p>Ask the seller to check it. Say exactly what is wrong and what it should be.</p>
			{#if notice}<p class="notice" role="status">{notice}</p>{/if}
			{#if error}<p class="error" role="alert">{error}</p>{/if}
			<form onsubmit={askForCorrection}>
				<label>
					Which sale?
					<select bind:value={selectedID} required disabled={busy}>
						<option value="">Choose a sale</option>
						{#each history.obligations as sale (sale.obligation_id)}
							<option value={sale.obligation_id}>{saleLabel(sale)}</option>
						{/each}
					</select>
				</label>
				<label>
					What is wrong?
					<textarea
						bind:value={reason}
						rows="4"
						disabled={busy}
						minlength="8"
						required
						placeholder="For example: I paid ₦50,000 on 28 August, but it is not showing."
					></textarea>
				</label>
				<label>
					Proof or reference numbers <small>one on each line, if you have any</small>
					<textarea bind:value={evidence} rows="3" disabled={busy}></textarea>
				</label>
				<button disabled={busy || !selectedID || reason.trim().length < 8}>
					{busy ? 'Sending…' : 'Ask for a correction'}
				</button>
			</form>
		</section>
	{:else if error}
		<p class="error" role="alert">{error}</p><button onclick={load}>Try again</button>
	{:else}
		<p>Opening your history…</p>
	{/if}
</main>

<style>
	h1 { font-size: clamp(2.5rem, 7vw, 5rem); line-height: 1; letter-spacing: -.055em; max-width: 10ch; }
	.intro { max-width: 42rem; color: var(--color-muted); }
	.grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 1rem; margin: 2.5rem 0; }
	article { display: grid; gap: .75rem; padding: 1.25rem; border: 1px solid var(--color-border); background: var(--color-surface); }
	article span { color: var(--color-muted); }
	article strong { font-size: 1.35rem; font-variant-numeric: tabular-nums; }
	h2 { margin-top: 2.5rem; }
	.table-wrap { overflow-x: auto; }
	table { width: 100%; border-collapse: collapse; background: var(--color-surface); }
	th, td { text-align: left; padding: .9rem; border-bottom: 1px solid var(--color-border); }
	th { color: var(--color-muted); font-size: .8rem; text-transform: uppercase; }
	.none { color: var(--color-muted); }
	.correction { max-width: 42rem; margin-top: 2rem; padding: 1.2rem; border: 1px solid var(--color-border); background: var(--color-surface); }
	.correction h2 { margin-top: 0; }
	.correction form, .correction label { display: grid; gap: .4rem; }
	.correction form { gap: .8rem; }
	.correction select, .correction textarea { box-sizing: border-box; width: 100%; padding: .7rem; border: 1px solid var(--color-border); font: inherit; }
	.correction button { width: max-content; padding: .7rem .9rem; }
	.notice { padding: .7rem; border-left: 4px solid var(--color-positive); }
	.error { color: var(--color-destructive); }
	@media (max-width: 760px) { .grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
