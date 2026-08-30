<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';

	let organizations: any[] = $state([]), organizationID = $state(''), statement: any = $state(null);
	let error = $state(''), notice = $state(''), busy = $state(''), limit = $state('');
	let principal = $state(''), goods = $state(''), invoiceReference = $state(''), invoiceDocumentHash = $state('');
	let dueDate = $state(''), collectionAt = $state('');
	let deliveryMethod: Record<string, string> = $state({}), releaseEvidence: Record<string, string> = $state({});
	const money = (value: number) => `₦${(Number(value || 0) / 100).toLocaleString('en-NG', { minimumFractionDigits: 2 })}`;
	const stateLabel = (state: string) => ({ PENDING_BUYER_CONFIRMATION: 'Waiting for buyer confirmation', BUYER_CONFIRMED: 'Buyer confirmed — safe to release', GOODS_RELEASED: 'Released — waiting for receipt', RECEIPT_ISSUE_REPORTED: 'Buyer reported an issue', ACTIVATED: 'Active obligation', CANCELLED: 'Cancelled', EXPIRED: 'Expired', ACTIVE: 'Active' })[state] ?? state;

	async function load() {
		error = '';
		for (const org of organizations) {
			const response = await fetch(`/api/v1/organizations/${org.id}/trade-lines/${page.params.id}/statement`, { credentials: 'include' });
			if (response.ok) { organizationID = org.id; statement = await response.json(); limit = String(statement.line.approved_limit_kobo / 100); return; }
		}
		error = 'We could not find this customer limit.';
	}

	async function command(path: string, body: unknown, key: string) {
		busy = key; error = ''; notice = '';
		try {
			const response = await fetch(path, { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json', 'Idempotency-Key': idempotencyKey(), ...csrfHeaders() }, body: JSON.stringify(body) });
			const result = await response.json().catch(() => ({}));
			if (!response.ok) throw new Error(result.detail ?? 'The action could not be completed.');
			notice = 'Your change was saved.'; await load();
		} catch (cause) { error = cause instanceof Error ? cause.message : 'The action could not be completed.'; }
		finally { busy = ''; }
	}

	async function reserve(event: SubmitEvent) {
		event.preventDefault();
		const principalKobo = Math.round(Number(principal.replace(/,/g, '')) * 100);
		if (principalKobo <= 0 || !goods || !dueDate || !collectionAt) { error = 'Enter an amount, goods, due date and collection time.'; return; }
		await command(`/api/v1/organizations/${organizationID}/trade-lines/${page.params.id}/drawdowns`, { principal_kobo: principalKobo, goods_description: goods, invoice_reference: invoiceReference, invoice_document_hash: invoiceDocumentHash, due_date: dueDate, collection_at: new Date(collectionAt).toISOString() }, 'reserve');
		if (!error) { principal = ''; goods = ''; invoiceReference = ''; invoiceDocumentHash = ''; }
	}

	async function reduce() {
		busy = 'limit'; error = '';
		const approved_limit_kobo = Math.round(Number(limit.replace(/,/g, '')) * 100);
		const response = await fetch(`/api/v1/organizations/${organizationID}/trade-lines/${page.params.id}`, { method: 'PATCH', credentials: 'include', headers: { 'Content-Type': 'application/json', 'Idempotency-Key': idempotencyKey(), ...csrfHeaders() }, body: JSON.stringify({ expected_version: statement.line.version, approved_limit_kobo }) });
		const result = await response.json().catch(() => ({})); busy = '';
		if (!response.ok) { error = result.detail ?? 'Limit could not be reduced.'; return; }
		await load();
	}

	onMount(async () => { const response = await fetch('/api/v1/organizations', { credentials: 'include' }); if (response.ok) { organizations = (await response.json()).organizations ?? []; await load(); } });
</script>

<svelte:head><title>Customer limit — Kredit</title></svelte:head>
<main class="shell workspace">
	<a href="/app/trade-lines">← Customer limits</a><p class="eyebrow">Customer limit</p>
	{#if statement}
		<h1>{money(statement.line.available_limit_kobo)} available.</h1>
		<section class="summary"><article><span>Full limit</span><strong>{money(statement.line.approved_limit_kobo)}</strong></article><article><span>Customer already owes</span><strong>{money(statement.line.current_exposure_kobo)}</strong></article><article><span>Waiting for customer</span><strong>{money(statement.line.reserved_pending_kobo)}</strong></article><article><span>Now</span><strong>{stateLabel(statement.line.state)}</strong></article></section>
		{#if error}<p class="error" role="alert">{error}</p>{/if}{#if notice}<p class="notice" role="status">{notice}</p>{/if}
		<section class="card compact"><h2>Pause this limit</h2><p>The customer cannot use a paused limit for a new sale. Existing sales and payments stay safe.</p>{#if statement.line.state === 'ACTIVE'}<button class="danger" disabled={busy==='suspend'} onclick={()=>command(`/api/v1/organizations/${organizationID}/trade-lines/${statement.line.id}/suspend`,{},'suspend')}>Pause this limit</button>{:else if statement.line.state === 'SUSPENDED'}<button class="primary" disabled={busy==='resume'} onclick={()=>command(`/api/v1/organizations/${organizationID}/trade-lines/${statement.line.id}/resume`,{},'resume')}>Use this limit again</button>{/if}</section>
		<section class="card"><h2>Add a sale from this limit</h2><p>Your customer will see the goods, money and payment day before you give them the goods.</p>
			<form onsubmit={reserve} class="form-grid"><label>Money to pay (₦)<input bind:value={principal} inputmode="decimal" required /></label><label>What are they buying?<textarea bind:value={goods} rows="3" required></textarea></label><label>Invoice number <small>optional</small><input bind:value={invoiceReference} /></label><label>Pay before<input type="date" bind:value={dueDate} required /></label><label>Bank debit may start after<input type="datetime-local" bind:value={collectionAt} required /></label><button class="primary wide" disabled={busy === 'reserve'}>{busy === 'reserve' ? 'Saving…' : `Add ${money(Math.round(Number(principal.replace(/,/g, '')) * 100))} sale`}</button></form>
		</section>
		<section class="card compact"><h2>Lower the limit</h2><p>You can lower only the part the customer has not used. The customer must agree before you raise it.</p><label>New limit (₦)<input bind:value={limit} inputmode="decimal" /></label><button disabled={busy === 'limit'} onclick={reduce}>Change limit to {money(Math.round(Number(limit.replace(/,/g, '')) * 100))}</button></section>
		<h2>Sales using this limit</h2>
		{#if statement.drawdowns.length}<div class="drawdowns">{#each statement.drawdowns as drawdown}<article class="drawdown">
			<header><strong>{money(drawdown.principal_kobo)}</strong><span class="status">{stateLabel(drawdown.state)}</span></header>
			<dl><dt>Goods</dt><dd>{drawdown.goods_description}</dd><dt>Pay before</dt><dd>{drawdown.due_date}</dd><dt>Bank debit after</dt><dd>{new Date(drawdown.collection_at).toLocaleString('en-NG')}</dd><dt>Extra time</dt><dd>{drawdown.grace_hours} hours</dd><dt>Invoice number</dt><dd>{drawdown.invoice_reference || 'None'}</dd></dl>
			<details class="hash"><summary>Technical record</summary><code>{drawdown.agreement_hash}</code></details><a href={`/api/v1/organizations/${organizationID}/trade-lines/${statement.line.id}/drawdowns/${drawdown.id}/agreement-document`} target="_blank" rel="noreferrer">Print or save this sale →</a>
			{#if drawdown.state === 'BUYER_CONFIRMED'}<div class="action"><label>How will they get the goods?<input bind:value={deliveryMethod[drawdown.id]} placeholder="Delivery or pickup" /></label><label>Delivery or receipt number<input bind:value={releaseEvidence[drawdown.id]} placeholder="Optional" /></label><button class="primary" disabled={busy === drawdown.id} onclick={() => command(`/api/v1/organizations/${organizationID}/trade-lines/${statement.line.id}/drawdowns/${drawdown.id}/release`, { delivery_method: deliveryMethod[drawdown.id], evidence_reference: releaseEvidence[drawdown.id] }, drawdown.id)}>The goods have left</button></div>{/if}
			{#if ['PENDING_BUYER_CONFIRMATION', 'BUYER_CONFIRMED'].includes(drawdown.state)}<button class="danger" disabled={busy === drawdown.id} onclick={() => command(`/api/v1/organizations/${organizationID}/trade-lines/${statement.line.id}/drawdowns/${drawdown.id}/cancel`, {}, drawdown.id)}>Cancel this sale</button>{/if}
			{#if drawdown.release_actor_id}<p>How the goods left: {drawdown.delivery_method}{drawdown.release_evidence_reference ? ` · ${drawdown.release_evidence_reference}` : ''}</p>{/if}{#if drawdown.receipt_state === 'issue_reported'}<p class="error">Customer's problem: {drawdown.receipt_issue_reason}</p>{/if}
		</article>{/each}</div>{:else}<p>No sale has used this limit yet.</p>{/if}
	{:else if error}<h1>We could not open this limit.</h1><p role="alert">{error}</p>{:else}<p>Opening customer limit…</p>{/if}
</main>

<style>
	.summary{display:grid;grid-template-columns:repeat(4,1fr);gap:1rem}.summary article,.card,.drawdown{padding:1rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}.summary article{display:grid;gap:.4rem}.summary span{color:var(--color-muted)}.card{margin:1.5rem 0}.compact{max-width:32rem}.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:1rem}.form-grid label,.action label,.compact label{display:grid;gap:.35rem}.form-grid input,.form-grid textarea,.action input,.compact input{padding:.7rem;border:1px solid var(--color-border);border-radius:.5rem;background:var(--color-surface);color:inherit}.wide{grid-column:1/-1}.drawdowns{display:grid;gap:1rem}.drawdown{gap:1rem}.drawdown header{display:flex;justify-content:space-between;gap:1rem;align-items:center}.drawdown dl{display:grid;grid-template-columns:max-content 1fr;gap:.4rem 1rem;margin:0}.drawdown dt{color:var(--color-muted)}.drawdown dd{margin:0}.hash{overflow-wrap:anywhere;color:var(--color-muted)}.action{display:grid;gap:.75rem;padding-top:.75rem;border-top:1px solid var(--color-border)}button{padding:.7rem 1rem;border:0;border-radius:999px;font-weight:700}.danger{color:var(--color-destructive);background:transparent;border:1px solid currentColor}.error{color:var(--color-destructive)}.notice{color:var(--color-positive)}@media(max-width:760px){.summary,.form-grid{grid-template-columns:repeat(2,1fr)}}@media(max-width:520px){.summary,.form-grid{grid-template-columns:1fr}}
</style>
