<script lang="ts">
 import {feeDisclosure} from "$lib/fee-terms";
	import { onMount } from 'svelte';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';
	let statements: any[] = $state([]), error = $state(''), notice = $state(''), busy = $state('');
	let issueReason: Record<string, string> = $state({});
	const money = (value: number) => `₦${(Number(value || 0) / 100).toLocaleString('en-NG', { minimumFractionDigits: 2 })}`;
	const stateLabel = (state: string) => ({ PENDING_BUYER_CONFIRMATION: 'Please check this sale', BUYER_CONFIRMED: 'You said yes — seller may send goods', GOODS_RELEASED: 'Seller sent goods — did you get them?', RECEIPT_ISSUE_REPORTED: 'Problem reported', ACTIVATED: 'Payment has started', CANCELLED: 'Cancelled', EXPIRED: 'Ended' })[state] ?? state;
	async function load() {
		error = '';
		const response = await fetch('/api/v1/buyer/trade-lines', { credentials: 'include' });
		if (!response.ok) { error = 'We could not open your customer limits.'; return; }
		const lines = (await response.json()).trade_lines ?? [];
		statements = (await Promise.all(lines.map(async (line: any) => { const result = await fetch(`/api/v1/buyer/trade-lines/${line.id}/statement`, { credentials: 'include' }); return result.ok ? result.json() : null; }))).filter(Boolean);
	}
	async function command(path: string, body: unknown, drawdownID: string) {
		busy = drawdownID; error = ''; notice = '';
		try {
			const response = await fetch(path, { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json', 'Idempotency-Key': idempotencyKey(), ...csrfHeaders() }, body: JSON.stringify(body) });
			const result = await response.json().catch(() => ({}));
			if (!response.ok) throw new Error(result.detail ?? 'The action could not be completed.');
			notice = 'Your response was recorded.'; await load();
		} catch (cause) { error = cause instanceof Error ? cause.message : 'The action could not be completed.'; }
		finally { busy = ''; }
	}
	onMount(load);
</script>

<svelte:head><title>Your customer limits — Kredit</title></svelte:head>
<main class="shell workspace">
	<p class="eyebrow">Your limits</p><h1>See how much you can take and pay later.</h1><p class="lede">Check every sale before the seller sends goods. Payment starts only after you say the goods arrived.</p>
	{#if error}<p class="error" role="alert">{error}</p>{/if}{#if notice}<p class="notice" role="status">{notice}</p>{/if}
	{#if statements.length}{#each statements as statement}<section class="line">
		<header><div><span>You can still use</span><strong>{money(statement.line.available_limit_kobo)}</strong></div><div><span>You already owe</span><strong>{money(statement.line.current_exposure_kobo)}</strong></div><div><span>Waiting for goods</span><strong>{money(statement.line.reserved_pending_kobo)}</strong></div></header>
		{#if statement.drawdowns.length}<div class="drawdowns">{#each statement.drawdowns as drawdown}<article>
			<div class="title"><strong>{money(drawdown.principal_kobo)}</strong><span class="status">{stateLabel(drawdown.state)}</span></div><h2>{drawdown.goods_description}</h2>
			<dl><dt>Pay before</dt><dd>{drawdown.due_date}</dd><dt>Bank debit after</dt><dd>{new Date(drawdown.collection_at).toLocaleString('en-NG')}</dd><dt>Extra time</dt><dd>{drawdown.grace_hours} hours</dd><dt>Invoice</dt><dd>{drawdown.invoice_reference || 'No number'}</dd><dt>Kredit fee</dt><dd>{feeDisclosure(drawdown.fee_terms)}</dd></dl>
			<details class="hash"><summary>Technical record</summary><code>{drawdown.agreement_hash}</code></details><a href={`/api/v1/buyer/trade-lines/${statement.line.id}/drawdowns/${drawdown.id}/agreement-document`} target="_blank" rel="noreferrer">Print or save this sale →</a>
			{#if drawdown.state === 'PENDING_BUYER_CONFIRMATION'}<p>Check the goods, money and dates. Payment does not start yet.</p><button class="primary" disabled={busy === drawdown.id} onclick={() => command(`/api/v1/buyer/trade-lines/${statement.line.id}/drawdowns/${drawdown.id}/confirm`, { agreement_hash: drawdown.agreement_hash }, drawdown.id)}>Yes to this {money(drawdown.principal_kobo)} sale</button>{/if}
			{#if ['PENDING_BUYER_CONFIRMATION', 'BUYER_CONFIRMED'].includes(drawdown.state)}<button class="danger" disabled={busy === drawdown.id} onclick={() => command(`/api/v1/buyer/trade-lines/${statement.line.id}/drawdowns/${drawdown.id}/cancel`, {}, drawdown.id)}>Cancel this sale</button>{/if}
			{#if drawdown.state === 'GOODS_RELEASED'}<div class="receipt"><p>Seller's delivery note: {drawdown.delivery_method}{drawdown.release_evidence_reference ? ` · ${drawdown.release_evidence_reference}` : ''}</p><button class="primary" disabled={busy === drawdown.id} onclick={() => command(`/api/v1/buyer/trade-lines/${statement.line.id}/drawdowns/${drawdown.id}/receipt`, { state: 'no_issue' }, drawdown.id)}>Yes, I got the goods</button><label>What is wrong?<textarea bind:value={issueReason[drawdown.id]} rows="3"></textarea></label><button class="danger" disabled={busy === drawdown.id || !issueReason[drawdown.id]?.trim()} onclick={() => command(`/api/v1/buyer/trade-lines/${statement.line.id}/drawdowns/${drawdown.id}/receipt`, { state: 'issue_reported', issue_reason: issueReason[drawdown.id] }, drawdown.id)}>Report the problem</button></div>{/if}
			{#if drawdown.receipt_state === 'issue_reported'}<p class="error">Problem reported: {drawdown.receipt_issue_reason}</p>{/if}{#if drawdown.obligation_id}<a href={`/buyer/obligations/${drawdown.obligation_id}`}>Open payment details →</a>{/if}
		</article>{/each}</div>{:else}<p>No sale has used this limit.</p>{/if}
	</section>{/each}{:else if !error}<section class="empty"><h2>No customer limit</h2><p>A limit will show here when a seller gives you one.</p></section>{/if}
</main>

<style>
	.line{margin:1.5rem 0;padding:1.25rem;border:1px solid var(--color-border);border-radius:1.25rem;background:var(--color-surface)}.line>header{display:grid;grid-template-columns:repeat(3,1fr);gap:1rem;padding-bottom:1rem;border-bottom:1px solid var(--color-border)}.line>header div{display:grid;gap:.35rem}.line>header span,dt{color:var(--color-muted)}.line>header strong{font-size:1.35rem}.drawdowns{display:grid;gap:1rem;margin-top:1rem}.drawdowns article{padding:1rem;border-radius:1rem;background:var(--color-surface-muted)}.title{display:flex;justify-content:space-between;gap:1rem}.title>strong{font-size:1.4rem}dl{display:grid;grid-template-columns:max-content 1fr;gap:.4rem 1rem}dd{margin:0}.hash{overflow-wrap:anywhere;color:var(--color-muted)}.receipt{display:grid;gap:.75rem;padding-top:.75rem;border-top:1px solid var(--color-border)}.receipt label{display:grid;gap:.35rem}.receipt textarea{padding:.7rem;border:1px solid var(--color-border);border-radius:.5rem;background:var(--color-surface);color:inherit}button{width:max-content;padding:.7rem 1rem;border:0;border-radius:999px;font-weight:700}.danger{color:var(--color-destructive);background:transparent;border:1px solid currentColor}.error{color:var(--color-destructive)}.notice{color:var(--color-positive)}@media(max-width:620px){.line>header{grid-template-columns:1fr}dl{grid-template-columns:1fr}dt{margin-top:.4rem}}
</style>
