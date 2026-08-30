<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';
	import Money from '$lib/components/Money.svelte';
	import { productLabel } from '$lib/product-language';
	const id = $derived(page.params.id);
	let organizationID = $state(''), view: any = $state(null), payments: any[] = $state([]), error = $state(''), busy = $state(false);
	let deliveryMethod = $state('supplier_delivery'), releaseNotes = $state(''), paymentAmount = $state(''), paymentReference = $state('');
	let draftPrincipal = $state(''), draftGoods = $state(''), draftDueDate = $state(''), draftCollectionAt = $state(''), draftGraceHours = $state(0);
	let disputeAmount=$state(''),disputeReason=$state(''),disputeExplanation=$state(''),disputeEffect=$state('CONTESTED_ONLY');
	async function load() {
		organizationID = page.url.searchParams.get('organization') ?? organizationID;
		if (!organizationID) { const organizations = await fetch('/api/v1/organizations', { credentials: 'include' }); if (organizations.ok) organizationID = (await organizations.json()).organizations?.[0]?.id ?? ''; }
		if (!organizationID) { error = 'Add your business before opening a sale.'; return; }
		const response = await fetch(`/api/v1/organizations/${organizationID}/credit-requests/${id}`, { credentials: 'include' });
		if (!response.ok) { error = 'We could not open this sale.'; return; }
		view = await response.json();
		if (view.request.state === 'DRAFT') { draftPrincipal = String(view.request.principal_kobo / 100); draftGoods = view.request.goods_description; draftDueDate = view.request.due_date; draftCollectionAt = new Date(view.request.collection_at).toISOString().slice(0,16); draftGraceHours = view.request.grace_hours; }
		if (view.obligation) { const paymentResponse = await fetch(`/api/v1/organizations/${organizationID}/credit-requests/${id}/payments`, { credentials: 'include' }); if (paymentResponse.ok) payments = (await paymentResponse.json()).payments ?? []; }
	}
	async function updateDraft() {
		busy = true; error = '';
		const principalKobo = Math.round(Number(draftPrincipal.replace(/,/g,'')) * 100);
		const response = await fetch(`/api/v1/organizations/${organizationID}/credit-requests/${id}`, { method: 'PATCH', credentials: 'include', headers: { 'Content-Type':'application/json', 'Idempotency-Key': idempotencyKey(), ...csrfHeaders() }, body: JSON.stringify({ expected_version: view.request.version, principal_kobo: principalKobo, goods_description: draftGoods, due_date: draftDueDate, grace_hours: Number(draftGraceHours), collection_at: new Date(draftCollectionAt).toISOString() }) });
		const result = await response.json().catch(() => ({})); busy = false;
		if (!response.ok) { error = result.detail ?? 'The draft could not be updated.'; return; }
		await load();
	}
	async function command(path: string, body: any = undefined) {
		busy = true; error = '';
		const response = await fetch(`/api/v1/organizations/${organizationID}/credit-requests/${id}/${path}`, { method: 'POST', credentials: 'include', headers: { ...(body ? {'Content-Type':'application/json'} : {}), 'Idempotency-Key': idempotencyKey(), ...csrfHeaders() }, body: body ? JSON.stringify(body) : undefined });
		const result = await response.json().catch(() => ({})); busy = false;
		if (!response.ok) { error = result.detail ?? 'The action could not be completed.'; return; }
		await load();
	}
	async function recordPayment() {
		const amount = Math.round(Number(paymentAmount.replace(/,/g,'')) * 100);
		if (amount <= 0) { error = 'Enter a valid payment amount.'; return; }
		await command('payments', { amount_kobo: amount, currency: 'NGN', source_type: 'supplier_recorded_transfer', provider_reference: paymentReference, paid_at: new Date().toISOString() });
		paymentAmount = ''; paymentReference = '';
	}
	async function openDispute(event:SubmitEvent){event.preventDefault();const amount=Math.round(Number(disputeAmount.replace(/,/g,''))*100);if(amount<=0||!disputeReason.trim()){error='Enter the contested amount and reason.';return}await command('disputes',{disputed_amount_kobo:amount,reason:disputeReason,explanation:disputeExplanation,collection_effect:disputeEffect});if(!error){disputeAmount='';disputeReason='';disputeExplanation=''}}
	onMount(load);
</script>

<svelte:head><title>Sale {id} — Kredit</title></svelte:head>
<section class="page-shell">
	<a href="/app/overview">← Your sales</a>
	{#if error}<p class="error" role="alert">{error}</p>{/if}
	{#if !view}<p>Opening sale…</p>{:else}
		<p class="eyebrow">Sale · {productLabel(view.request.state)}</p><h1>{view.request.buyer_legal_name}</h1>
		<p class="muted">Sale number <strong>{id}</strong>. The sale, goods and payments stay together.</p>
		<section class="detail-grid"><article class="card"><h2>The sale</h2><dl><div><dt>Money to pay</dt><dd><Money amountKobo={view.request.principal_kobo} /></dd></div><div><dt>Goods</dt><dd>{view.request.goods_description}</dd></div><div><dt>Pay before</dt><dd>{view.request.due_date}</dd></div><div><dt>Bank debit after</dt><dd>{new Date(view.request.collection_at).toLocaleString()}</dd></div><div><dt>Extra time</dt><dd>{view.request.grace_hours} hours</dd></div></dl></article>
		<article class="card"><h2>What is happening?</h2><p><strong>{productLabel(view.request.state)}</strong></p>{#if view.agreement?.document_hash}<details><summary>Technical record</summary><p>Sale record code<br/><code>{view.agreement.document_hash}</code></p></details>{/if}{#if view.obligation}<p>Money left<br/><strong><Money amountKobo={view.obligation.outstanding_kobo} /></strong></p><p><a href={`/api/v1/organizations/${organizationID}/credit-requests/${id}/agreement-document`} target="_blank" rel="noreferrer">Print or save this sale →</a></p>{/if}</article></section>
		{#if view.request.state === 'DRAFT'}<section class="card action"><h2>Check before you send</h2><p>You can change this sale now. After you send it, the customer must see the same sale.</p><label>Money to pay (₦)<input bind:value={draftPrincipal} inputmode="decimal" /></label><label>Goods<textarea bind:value={draftGoods}></textarea></label><label>Payment day<input type="date" bind:value={draftDueDate} /></label><label>Bank debit day and time<input type="datetime-local" bind:value={draftCollectionAt} /></label><label>Extra hours<input type="number" min="0" max="720" bind:value={draftGraceHours} /></label><div class="button-row"><button disabled={busy} onclick={updateDraft}>Save for later</button><button class="primary" disabled={busy} onclick={() => command('send')}>Send to customer</button><button class="danger" disabled={busy} onclick={() => command('cancel')}>Delete this sale</button></div></section>{/if}
		{#if view.request.state === 'SENT' || view.request.state === 'BUYER_REVIEWING'}<section class="card action"><h2>Waiting for your customer</h2><p>You can cancel if you no longer want to make this sale. The old record will stay.</p><button class="danger" disabled={busy} onclick={() => command('cancel')}>Cancel this sale</button></section>{/if}
		{#if view.request.state === 'READY_TO_RELEASE'}<section class="card action"><h2>Did the goods leave?</h2><label>How did the customer get them?<select bind:value={deliveryMethod}><option value="supplier_delivery">We delivered</option><option value="buyer_collection">Customer picked up</option><option value="third_party_delivery">Another person delivered</option></select></label><label>Delivery note<textarea bind:value={releaseNotes}></textarea></label><button class="primary" disabled={busy} onclick={() => command('release',{delivery_method:deliveryMethod,notes:releaseNotes})}>Yes, the goods left</button></section>{/if}
		{#if view.obligation}<section class="card action"><h2>Add money you received</h2><label>Money received (₦)<input bind:value={paymentAmount} inputmode="decimal" /></label><label>Transfer or receipt number<input bind:value={paymentReference} /></label><button class="primary" disabled={busy} onclick={recordPayment}>Add this payment</button><h3>Past payments</h3>{#if payments.length}<ul>{#each payments as payment}<li><Money amountKobo={payment.amount_kobo} /> · {productLabel(payment.source_type)} · {productLabel(payment.state)}</li>{/each}</ul>{:else}<p>No payment yet.</p>{/if}</section><section class="card action"><h2>Report a problem</h2><p>Add the money affected and what happened.</p><form onsubmit={openDispute}><label>Money affected (₦)<input bind:value={disputeAmount} inputmode="decimal" required /></label><label>Short reason<input bind:value={disputeReason} required /></label><label>What happened?<textarea bind:value={disputeExplanation} rows="4"></textarea></label><label>What should happen to bank debit?<select bind:value={disputeEffect}><option value="CONTESTED_ONLY">Stop only the money affected</option><option value="FULL_BLOCK">Stop all bank debit</option><option value="NO_AUTOMATIC_BLOCK">Save the problem but do not stop debit</option></select></label><button class="primary" disabled={busy}>Report this problem</button></form></section>{/if}
	{/if}
	</section>
<style>.detail-grid{display:grid;grid-template-columns:2fr 1fr;gap:1rem;margin:1.5rem 0}.card{padding:1.5rem}dl div{display:flex;justify-content:space-between;gap:2rem;border-bottom:1px solid var(--color-border);padding:.7rem 0}dd{margin:0;text-align:right;font-weight:700}.action,.action form{display:grid;gap:.8rem;margin:1rem 0}.action form{margin:0}.action label{display:grid;gap:.35rem;font-weight:700}.action input,.action select,.action textarea{padding:.75rem;border:1px solid var(--color-border);border-radius:.7rem;font:inherit}.button-row{display:flex;flex-wrap:wrap;gap:.7rem}.danger{color:var(--color-destructive);border-color:var(--color-destructive);background:var(--color-surface)}code{overflow-wrap:anywhere}@media(max-width:720px){.detail-grid{grid-template-columns:1fr}}</style>
