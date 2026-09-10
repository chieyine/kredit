<script lang="ts">
 import {feeDisclosure} from "$lib/fee-terms";
 import { parseNaira, formatKobo, nairaInput } from '$lib/money';
 import { checkedJSON, LatestRequest, publicError, record, rows, RequestError } from '$lib/api/reliable';
 import { MutationIntent } from '$lib/api/mutation';
 import { organization } from '$lib/records';
 import { tradeLine, drawdown, tradeStatement } from '$lib/trade-line-records';
 import { lagosISO, localTime } from '$lib/admin-client';
	import { page } from '$app/state';
	import { onMount, untrack } from 'svelte';


	let organizations: any[] = $state([]), organizationID = $state(''), statement: any = $state(null);
	let error = $state(''), notice = $state(''), busy = $state(''), limit = $state('');
	let principal = $state(''), goods = $state(''), invoiceReference = $state(''), invoiceDocumentHash = $state('');
	let dueDate = $state(''), collectionAt = $state('');
	let deliveryMethod: Record<string, string> = $state({}), releaseEvidence: Record<string, string> = $state({});
	const money = formatKobo;
 let loading = $state(true), pauseReason = $state('');
 let capabilityError = $state(''), capabilityLoading = $state(true);
 type PendingCommand = {path:string;body:unknown;key:string;method:string;lineID:string};
 let pendingCommand = $state<PendingCommand | null>(null);
 let routeGeneration = 0;
 const reads = new LatestRequest(), intents = new Map<string, MutationIntent>();
	const stateLabel = (state: string) => ({ PENDING_BUYER_CONFIRMATION: 'Waiting for the customer to agree', BUYER_CONFIRMED: 'Customer agreed — you can send the goods', GOODS_RELEASED: 'Goods sent — waiting for them to confirm', RECEIPT_ISSUE_REPORTED: 'Customer reported a problem', ACTIVATED: 'Payment has started', CANCELLED: 'Cancelled', EXPIRED: 'Expired', ACTIVE: 'Active' })[state] ?? state;

 async function load(lineID = page.params.id ?? '') {
  const request = reads.begin(); loading = true; error = ''; statement = null;
  try {
   if (!organizations.length) {
    const result = await checkedJSON('/api/v1/organizations', rows('organizations', organization), { signal: request.signal });
    if (!request.current()) return; organizations = result;
   }
   const requested = new URLSearchParams(location.search).get('organization');
   const candidates = [...organizations].sort((a,b) => Number(b.id === requested) - Number(a.id === requested));
   for (const org of candidates) {
    try {
     const result = await checkedJSON(`/api/v1/organizations/${encodeURIComponent(org.id)}/trade-lines/${encodeURIComponent(lineID)}/statement`, tradeStatement, { signal: request.signal });
     if (!request.current()) return;
     if (result.line.id !== lineID || result.line.supplier_organization_id !== org.id) throw new Error('Wrong customer limit');
     organizationID = org.id; statement = result; if (!pendingCommand) limit = nairaInput(result.line.approved_limit_kobo); return;
    } catch (cause) { if (!(cause instanceof RequestError && [403,404].includes(cause.status))) throw cause; }
   }
   throw new Error('Customer limit unavailable');
  } catch (cause) { if (request.current()) error = publicError(cause, 'this customer limit'); }
  finally { if (request.current()) loading = false; }
 }
 async function command(path: string, body: unknown, key: string, method = 'POST') {
  if (busy || loading || !statement) return false;
  const lineID = statement.line.id, generation = routeGeneration;
  if (lineID !== page.params.id) return false;
  if (pendingCommand) {
   if (pendingCommand.path !== path || pendingCommand.lineID !== lineID) return false;
   body = pendingCommand.body; method = pendingCommand.method; key = pendingCommand.key;
  }
  const submitted = {path,body:body === undefined ? undefined : JSON.parse(JSON.stringify(body)),key,method,lineID};
  busy = key; error = ''; notice = '';
  try {
   let intent = intents.get(path); if (!intent) { intent = new MutationIntent('customer-limit-command', path); intents.set(path, intent); }
   await intent.run(body, value => {
    const result = record(value), line = tradeLine(result.trade_line);
    if (line.id !== lineID) throw new Error('Limit update not confirmed');
    if (path.includes('/drawdowns')) {
     const sale = drawdown(result.drawdown);
     if (sale.trade_line_id !== lineID || (key !== 'reserve' && sale.id !== key)) throw new Error('Sale update not confirmed');
    }
    return result;
   }, method);
   if (generation !== routeGeneration || lineID !== page.params.id) return false;
   pendingCommand = null; notice = 'Saved.'; await load(lineID); return true;
  } catch (cause) {
   if (generation === routeGeneration && lineID === page.params.id) {
    pendingCommand = intents.get(path)?.unresolved ? submitted : null;
    error = cause instanceof Error ? cause.message : 'The result has not been confirmed.';
   }
   return false;
  } finally { if (generation === routeGeneration && lineID === page.params.id) busy = ''; }
 }

	async function reserve(event: SubmitEvent) {
		event.preventDefault();
		const principalKobo = parseNaira(principal);
		if (principalKobo <= 0 || !goods || !dueDate || !collectionAt) { error = 'Please enter the money, the goods, the payment day and the bank debit time.'; return; }
		let collection: string; try { collection = lagosISO(collectionAt); } catch { error = 'Enter a valid Lagos collection date and time.'; return; }
		const saved = await command(`/api/v1/organizations/${organizationID}/trade-lines/${page.params.id}/drawdowns`, { principal_kobo: principalKobo, goods_description: goods, invoice_reference: invoiceReference, invoice_document_hash: invoiceDocumentHash, due_date: dueDate, collection_at: collection }, 'reserve');
		if (saved) { principal = ''; goods = ''; invoiceReference = ''; invoiceDocumentHash = ''; }
	}

 async function reduce() {
  if (busy || !statement) return;
  const amount = parseNaira(limit);
  if (amount < 0) { error = 'Enter a valid new limit.'; return; }
  await command(`/api/v1/organizations/${organizationID}/trade-lines/${page.params.id}`, { expected_version: statement.line.version, approved_limit_kobo: amount }, 'limit', 'PATCH');
 }

	let canAddDrawdown = $state(false);
 const capabilities = new LatestRequest();
 async function loadCapabilities() {
  const request = capabilities.begin(); capabilityLoading = true; capabilityError = '';
  try {
   const enabled = await checkedJSON('/api/v1/platform/capabilities', value => {
    const flag = record(record(value).features).drawdowns;
    if (typeof flag !== 'boolean') throw new Error('Customer-limit availability was incomplete.');
    return flag;
   }, {signal:request.signal});
   if (request.current()) canAddDrawdown = enabled;
  } catch (cause) { if (request.current()) capabilityError = publicError(cause, 'sale availability'); }
  finally { if (request.current()) capabilityLoading = false; }
 }
 $effect(() => {
  const lineID = page.params.id;
  untrack(() => {
   routeGeneration++; statement = null; organizationID = ''; busy = ''; pendingCommand = null; notice = '';
   principal = ''; goods = ''; invoiceReference = ''; invoiceDocumentHash = ''; dueDate = ''; collectionAt = ''; pauseReason = ''; deliveryMethod = {}; releaseEvidence = {};
   void load(lineID);
  });
  return () => reads.cancel();
 });
 onMount(() => { void loadCapabilities(); return () => capabilities.cancel(); });
</script>

<svelte:head><title>Customer limit — Kredit</title></svelte:head>
<main class="shell workspace">
	<a href="/app/trade-lines">← Customer limits</a><p class="eyebrow">Customer limit</p>
	{#if statement}
		<h1>{money(statement.line.available_limit_kobo)} available.</h1>
		<section class="summary"><article><span>Full limit</span><strong>{money(statement.line.approved_limit_kobo)}</strong></article><article><span>Customer already owes</span><strong>{money(statement.line.current_exposure_kobo)}</strong></article><article><span>Waiting for customer</span><strong>{money(statement.line.reserved_pending_kobo)}</strong></article><article><span>Now</span><strong>{stateLabel(statement.line.state)}</strong></article></section>
		{#if error}<p class="error" role="alert">{error}</p>{/if}{#if notice}<p class="notice" role="status">{notice}</p>{/if}
  {#if pendingCommand}<section class="card"><p>The last change has not been confirmed. Retry its original details before making another change.</p><button disabled={!!busy || loading} onclick={()=>{if(pendingCommand)void command(pendingCommand.path,pendingCommand.body,pendingCommand.key,pendingCommand.method)}}>Retry the original change</button></section>{/if}
		<section class="card compact"><h2>Pause this limit</h2><p>While a limit is paused, the customer cannot use it for a new sale. Sales already running are not affected.</p>{#if statement.line.state === 'ACTIVE'}<label>Why are you pausing this limit?<input disabled={!!busy || !!pendingCommand} bind:value={pauseReason} minlength="8" maxlength="2000" /></label><button class="danger" disabled={!!busy || !!pendingCommand || pauseReason.trim().length < 8} onclick={()=>command(`/api/v1/organizations/${organizationID}/trade-lines/${statement.line.id}/suspend`,{reason:pauseReason},'suspend')}>Pause this limit</button>{:else if statement.line.state === 'SUSPENDED'}<button class="primary" disabled={!!busy || !!pendingCommand || loading} onclick={()=>command(`/api/v1/organizations/${organizationID}/trade-lines/${statement.line.id}/resume`,{},'resume')}>Use this limit again</button>{/if}</section>
		{#if capabilityLoading}<section class="card"><p role="status">Checking whether sales from this limit are available…</p></section>
  {:else if capabilityError}<section class="card"><p role="alert">{capabilityError}</p><button onclick={loadCapabilities}>Check sale availability again</button></section>
  {:else if canAddDrawdown}
		<section class="card"><h2>Add a sale from this limit</h2><p>Your customer sees the goods, the money and the payment day before you hand anything over.</p>
			<form onsubmit={reserve} class="form-grid"><label>Money to pay (₦)<input disabled={!!busy || !!pendingCommand} bind:value={principal} inputmode="decimal" required /></label><label>What are they buying?<textarea disabled={!!busy || !!pendingCommand} bind:value={goods} rows="3" required></textarea></label><label>Invoice number <small>optional</small><input disabled={!!busy || !!pendingCommand} bind:value={invoiceReference} /></label><label>Pay before<input disabled={!!busy || !!pendingCommand} type="date" bind:value={dueDate} required /></label><label>If unpaid, Kredit may debit their bank after (Lagos time)<input disabled={!!busy || !!pendingCommand} type="datetime-local" bind:value={collectionAt} required /></label><button class="primary wide" disabled={!!busy || !!pendingCommand || loading}>{busy === 'reserve' ? 'Saving…' : `Add ${money(parseNaira(principal))} sale`}</button></form>
		</section>
		{:else}
		<section class="card"><h2>Add a sale from this limit</h2><p>Selling from a customer limit is switched off for this account, so there is nothing to fill in here. You can still record a normal sale, and the balance below stays up to date.</p><a class="primary" href="/app/credit/new">Record a sale</a></section>
		{/if}
		<section class="card compact"><h2>Lower the limit</h2><p>You can only take back what the customer has not used yet. To raise a limit, the customer must agree first.</p><label>New limit (₦)<input disabled={!!busy || !!pendingCommand} bind:value={limit} inputmode="decimal" /></label><button disabled={!!busy || !!pendingCommand || loading} onclick={reduce}>Change limit to {money(parseNaira(limit))}</button></section>
		<h2>Sales using this limit</h2>
		{#if statement.drawdowns.length}<div class="drawdowns">{#each statement.drawdowns as drawdown}<article class="drawdown">
			<header><strong>{money(drawdown.principal_kobo)}</strong><span class="status">{stateLabel(drawdown.state)}</span></header>
			<dl><dt>Goods</dt><dd>{drawdown.goods_description}<p>{feeDisclosure(drawdown.fee_terms)}</p></dd><dt>Pay before</dt><dd>{drawdown.due_date}</dd><dt>Bank debit after</dt><dd>{localTime(drawdown.collection_at)}</dd><dt>Extra time before that</dt><dd>{drawdown.grace_hours} hours</dd><dt>Invoice number</dt><dd>{drawdown.invoice_reference || 'None'}</dd></dl>
			<details class="hash"><summary>Technical record (for reference)</summary><code>{drawdown.agreement_hash}</code></details><a href={`/api/v1/organizations/${organizationID}/trade-lines/${statement.line.id}/drawdowns/${drawdown.id}/agreement-document`} target="_blank" rel="noreferrer">Print or save a copy of this sale →</a>
			{#if drawdown.state === 'BUYER_CONFIRMED'}<div class="action"><label>How will they get the goods?<input disabled={!!busy || !!pendingCommand} bind:value={deliveryMethod[drawdown.id]} placeholder="Delivery or pickup" /></label><label>Delivery or receipt number<input disabled={!!busy || !!pendingCommand} bind:value={releaseEvidence[drawdown.id]} placeholder="Optional" /></label><button class="primary" disabled={!!busy || !!pendingCommand || loading} onclick={() => command(`/api/v1/organizations/${organizationID}/trade-lines/${statement.line.id}/drawdowns/${drawdown.id}/release`, { delivery_method: deliveryMethod[drawdown.id], evidence_reference: releaseEvidence[drawdown.id] }, drawdown.id)}>The goods have left</button></div>{/if}
			{#if ['PENDING_BUYER_CONFIRMATION', 'BUYER_CONFIRMED'].includes(drawdown.state)}<button class="danger" disabled={!!busy || !!pendingCommand || loading} onclick={() => command(`/api/v1/organizations/${organizationID}/trade-lines/${statement.line.id}/drawdowns/${drawdown.id}/cancel`, {}, drawdown.id)}>Cancel this sale</button>{/if}
			{#if drawdown.release_actor_id}<p>How the goods left: {drawdown.delivery_method}{drawdown.release_evidence_reference ? ` · ${drawdown.release_evidence_reference}` : ''}</p>{/if}{#if drawdown.receipt_state === 'issue_reported'}<p class="error">Customer reported: {drawdown.receipt_issue_reason}</p>{/if}
		</article>{/each}</div>{:else}<p>No sale has used this limit yet.</p>{/if}
	{:else if error}<h1>We could not open this limit.</h1><p role="alert">{error}</p><button type="button" onclick={()=>load()}>Try again</button>{:else}<p role="status">Opening customer limit…</p>{/if}
</main>

<style>
	.summary{display:grid;grid-template-columns:repeat(4,1fr);gap:1rem}.summary article,.card,.drawdown{padding:1rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}.summary article{display:grid;gap:.4rem}.summary span{color:var(--color-muted)}.card{margin:1.5rem 0}.compact{max-width:32rem}.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:1rem}.form-grid label,.action label,.compact label{display:grid;gap:.35rem}.form-grid input,.form-grid textarea,.action input,.compact input{padding:.7rem;border:1px solid var(--color-border);border-radius:.5rem;background:var(--color-surface);color:inherit}.wide{grid-column:1/-1}.drawdowns{display:grid;gap:1rem}.drawdown{gap:1rem}.drawdown header{display:flex;justify-content:space-between;gap:1rem;align-items:center}.drawdown dl{display:grid;grid-template-columns:max-content 1fr;gap:.4rem 1rem;margin:0}.drawdown dt{color:var(--color-muted)}.drawdown dd{margin:0}.hash{overflow-wrap:anywhere;color:var(--color-muted)}.action{display:grid;gap:.75rem;padding-top:.75rem;border-top:1px solid var(--color-border)}button{padding:.7rem 1rem;border:0;border-radius:999px;font-weight:700}.danger{color:var(--color-destructive);background:transparent;border:1px solid currentColor}.error{color:var(--color-destructive)}.notice{color:var(--color-positive)}@media(max-width:760px){.summary,.form-grid{grid-template-columns:repeat(2,1fr)}}@media(max-width:520px){.summary,.form-grid{grid-template-columns:1fr}}
</style>
