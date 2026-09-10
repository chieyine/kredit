<script lang="ts">
 import {feeDisclosure,validFeeTerms} from "$lib/fee-terms";
	import { onMount } from 'svelte';
	import { checkedJSON, LatestRequest, publicError, record, rows } from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';
	import { tradeLine, tradeStatement, drawdown as drawdownRecord } from '$lib/trade-line-records';
	import { formatKobo } from '$lib/money';
	import { readableDateTime } from '$lib/datetime';
	let statements: any[] = $state([]), error = $state(''), notice = $state(''), busy = $state(''), loading = $state(true);
	let issueReason: Record<string, string> = $state({});
	const money = formatKobo;
	const reads = new LatestRequest(), intents = new Map<string, MutationIntent>();
	const stateLabel = (state: string) => ({ PENDING_BUYER_CONFIRMATION: 'Please read this sale', BUYER_CONFIRMED: 'You said yes — the seller can now send the goods', GOODS_RELEASED: 'The seller sent the goods — did they reach you?', RECEIPT_ISSUE_REPORTED: 'Problem reported', ACTIVATED: 'Payment has started', CANCELLED: 'Cancelled', EXPIRED: 'Ended' })[state] ?? state;
	async function load() {
		const read = reads.begin(); loading = true; error = ''; statements = [];
		try {
			const lines = await checkedJSON('/api/v1/buyer/trade-lines', rows('trade_lines', tradeLine), {signal: read.signal});
			const result = [];
			for (let offset = 0; offset < lines.length; offset += 8) {
				if (!read.current()) return;
				result.push(...await Promise.all(lines.slice(offset, offset+8).map(line => checkedJSON(`/api/v1/buyer/trade-lines/${encodeURIComponent(line.id)}/statement`, value => {
					const statement = tradeStatement(value);
					if (statement.line.id !== line.id) throw new Error('Customer limit did not match');
					return statement;
				}, {signal: read.signal}))));
			}
			if (read.current()) statements = result;
		} catch (cause) { if (read.current()) error = publicError(cause, 'your customer limits'); }
		finally { if (read.current()) loading = false; }
	}
	async function command(path: string, body: unknown, drawdownID: string) {
		if (busy || loading) return;
		busy = drawdownID; error = ''; notice = '';
		try {
			if (!intents.has(path)) intents.set(path, new MutationIntent('buyer-limit-action', path));
			await intents.get(path)!.run(body, value => {
				const result = record(value), saved = drawdownRecord(result.drawdown), line = tradeLine(result.trade_line);
				if (saved.id !== drawdownID || saved.trade_line_id !== line.id || !path.includes(`/trade-lines/${encodeURIComponent(line.id)}/`)) throw new Error('Purchase update not confirmed');
				return saved;
			});
			notice = 'Your response has been saved.'; await load();
		} catch (cause) { error = cause instanceof Error ? cause.message : 'We could not confirm your response.'; }
		finally { busy = ''; }
	}
	onMount(() => { void load(); return () => reads.cancel(); });
</script>

<svelte:head><title>Your customer limits — Kredit</title></svelte:head>
<main class="shell workspace">
	<p class="eyebrow">Your limits</p><h1>How much you can take now and pay later.</h1><p class="lede">Read every sale before the seller sends anything. Payment only starts after you confirm the goods reached you.</p>
	{#if error}<p class="error" role="alert">{error}</p>{/if}{#if notice}<p class="notice" role="status">{notice}</p>{/if}
	{#if loading}<p role="status">Opening your customer limits…</p>{:else if statements.length}{#each statements as statement}<section class="line">
		<header><div><span>You can still take</span><strong>{money(statement.line.available_limit_kobo)}</strong></div><div><span>You already owe</span><strong>{money(statement.line.current_exposure_kobo)}</strong></div><div><span>Waiting for the goods</span><strong>{money(statement.line.reserved_pending_kobo)}</strong></div></header>
		{#if statement.drawdowns.length}<div class="drawdowns">{#each statement.drawdowns as drawdown}<article>
			<div class="title"><strong>{money(drawdown.principal_kobo)}</strong><span class="status">{stateLabel(drawdown.state)}</span></div><h2>{drawdown.goods_description}</h2>
			<dl><dt>Pay before</dt><dd>{drawdown.due_date}</dd><dt>Kredit may debit your bank after</dt><dd>{readableDateTime(drawdown.collection_at)}</dd><dt>Extra time before that</dt><dd>{drawdown.grace_hours} hours</dd><dt>Invoice</dt><dd>{drawdown.invoice_reference || 'No number'}</dd><dt>Kredit fee (paid by the seller)</dt><dd>{feeDisclosure(drawdown.fee_terms)}</dd></dl>
			{#if drawdown.legal_versions}<p>Documents for this sale: <a href={`/legal/terms?version=${encodeURIComponent(drawdown.legal_versions.terms_version)}`}>Terms</a> · <a href={`/legal/privacy?version=${encodeURIComponent(drawdown.legal_versions.privacy_version)}`}>Privacy notice</a></p>{/if}<details class="hash"><summary>Agreement reference</summary><code>{drawdown.agreement_hash}</code></details><a href={`/api/v1/buyer/trade-lines/${encodeURIComponent(statement.line.id)}/drawdowns/${encodeURIComponent(drawdown.id)}/agreement-document`} target="_blank" rel="noreferrer">Print or save a copy of this sale →</a>
			{#if drawdown.state === 'PENDING_BUYER_CONFIRMATION'}<p>Read the goods, the money and the dates. Nothing is owed yet.</p><button class="primary" disabled={busy !== '' || loading || !validFeeTerms(drawdown.fee_terms)} onclick={() => command(`/api/v1/buyer/trade-lines/${encodeURIComponent(statement.line.id)}/drawdowns/${encodeURIComponent(drawdown.id)}/confirm`, { agreement_hash: drawdown.agreement_hash }, drawdown.id)}>Yes to this {money(drawdown.principal_kobo)} sale</button>{/if}
			{#if ['PENDING_BUYER_CONFIRMATION', 'BUYER_CONFIRMED'].includes(drawdown.state)}<button class="danger" disabled={busy !== '' || loading} onclick={() => command(`/api/v1/buyer/trade-lines/${encodeURIComponent(statement.line.id)}/drawdowns/${encodeURIComponent(drawdown.id)}/cancel`, {}, drawdown.id)}>Cancel this sale</button>{/if}
			{#if drawdown.state === 'GOODS_RELEASED'}<div class="receipt"><p>Seller's delivery note: {drawdown.delivery_method}{drawdown.release_evidence_reference ? ` · ${drawdown.release_evidence_reference}` : ''}</p><button class="primary" disabled={busy !== '' || loading} onclick={() => command(`/api/v1/buyer/trade-lines/${encodeURIComponent(statement.line.id)}/drawdowns/${encodeURIComponent(drawdown.id)}/receipt`, { state: 'no_issue' }, drawdown.id)}>Yes, I got the goods</button><label>What is wrong?<textarea bind:value={issueReason[drawdown.id]} rows="3" disabled={busy !== '' || loading}></textarea></label><button class="danger" disabled={busy !== '' || loading || !issueReason[drawdown.id]?.trim()} onclick={() => command(`/api/v1/buyer/trade-lines/${encodeURIComponent(statement.line.id)}/drawdowns/${encodeURIComponent(drawdown.id)}/receipt`, { state: 'issue_reported', issue_reason: issueReason[drawdown.id] }, drawdown.id)}>Report the problem</button></div>{/if}
			{#if drawdown.receipt_state === 'issue_reported'}<p class="error">Problem reported: {drawdown.receipt_issue_reason}</p>{/if}{#if drawdown.obligation_id}<a href={`/buyer/obligations/${encodeURIComponent(drawdown.obligation_id)}`}>Open payment details →</a>{/if}
		</article>{/each}</div>{:else}<p>You have not used this limit yet.</p>{/if}
	</section>{/each}{:else if !error}<section class="empty"><h2>You have no buying limit yet</h2><p>One will show here when a seller gives you one.</p></section>{/if}
{#if error}<button disabled={busy !== '' || loading} onclick={load}>Reload customer limits</button>{/if}
</main>

<style>
	.line{margin:1.5rem 0;padding:1.25rem;border:1px solid var(--color-border);border-radius:1.25rem;background:var(--color-surface)}.line>header{display:grid;grid-template-columns:repeat(3,1fr);gap:1rem;padding-bottom:1rem;border-bottom:1px solid var(--color-border)}.line>header div{display:grid;gap:.35rem}.line>header span,dt{color:var(--color-muted)}.line>header strong{font-size:1.35rem}.drawdowns{display:grid;gap:1rem;margin-top:1rem}.drawdowns article{padding:1rem;border-radius:1rem;background:var(--color-surface-muted)}.title{display:flex;justify-content:space-between;gap:1rem}.title>strong{font-size:1.4rem}dl{display:grid;grid-template-columns:max-content 1fr;gap:.4rem 1rem}dd{margin:0}.hash{overflow-wrap:anywhere;color:var(--color-muted)}.receipt{display:grid;gap:.75rem;padding-top:.75rem;border-top:1px solid var(--color-border)}.receipt label{display:grid;gap:.35rem}.receipt textarea{padding:.7rem;border:1px solid var(--color-border);border-radius:.5rem;background:var(--color-surface);color:inherit}button{width:max-content;padding:.7rem 1rem;border:0;border-radius:999px;font-weight:700}.danger{color:var(--color-destructive);background:transparent;border:1px solid currentColor}.error{color:var(--color-destructive)}.notice{color:var(--color-positive)}@media(max-width:620px){.line>header{grid-template-columns:1fr}dl{grid-template-columns:1fr}dt{margin-top:.4rem}}
</style>
