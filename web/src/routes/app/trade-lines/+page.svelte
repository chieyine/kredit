<script lang="ts">
 import { parseNaira, formatKobo } from '$lib/money';
 import { onMount } from 'svelte';
 import { checkedJSON, LatestRequest, publicError, record, rows } from '$lib/api/reliable';
 import { MutationIntent } from '$lib/api/mutation';
 import { organization, customer } from '$lib/records';
 import { tradeLine } from '$lib/trade-line-records';
 import { lagosISO } from '$lib/admin-client';
 import { productLabel } from '$lib/product-language';
 let organizations: any[] = $state([]), organizationID = $state(''), lines: any[] = $state([]), customers: any[] = $state([]), loading = $state(true), busy = $state(false), error = $state(''), loadError = $state(''), notice = $state('');
 let buyerUserID = $state(''), buyerBusinessID = $state(''), mandateID = $state(''), limit = $state(''), cadence = $state('monthly'), grace = $state(48), startAt = $state(''), endAt = $state(''), termsVersion = $state('trade-line-v1');
 let canCreateLimit = $state(false);
 const money = formatKobo, reads = new LatestRequest(), intents = new Map<string, MutationIntent>();
 async function load() {
  const request = reads.begin(); loading = true; error = ''; loadError = ''; lines = []; customers = []; canCreateLimit = false;
  try {
   if (!organizations.length) {
    const result = await checkedJSON('/api/v1/organizations', rows('organizations', organization), { signal: request.signal });
    if (!request.current()) return;
    organizations = result;
    const requested = new URLSearchParams(location.search).get('organization'); organizationID = result.find(item => item.id === requested)?.id ?? result[0]?.id ?? '';
   }
   if (!organizationID) return;
   const base = `/api/v1/organizations/${encodeURIComponent(organizationID)}`;
   const [newLines, newCustomers, enabled] = await Promise.all([
    checkedJSON(base + '/trade-lines', rows('trade_lines', tradeLine), { signal: request.signal }),
    checkedJSON(base + '/customers', rows('customers', customer), { signal: request.signal }),
    checkedJSON('/api/v1/platform/capabilities', value => { const features = record(record(value).features); if (typeof features.trade_lines !== 'boolean') throw new Error('Capabilities unavailable'); return features.trade_lines; }, { signal: request.signal })
   ]);
   if (request.current()) { lines = newLines; customers = newCustomers; canCreateLimit = enabled; }
  } catch (cause) { if (request.current()) { loadError = publicError(cause, 'customer limits and setup permissions'); error = loadError; } }
  finally { if (request.current()) loading = false; }
 }
 function selectCustomer() { buyerBusinessID = customers.find(item => item.buyer_user_id === buyerUserID)?.buyer_business_id ?? ''; }
 function switchBusiness() { buyerUserID = ''; buyerBusinessID = ''; mandateID = ''; notice = ''; void load(); }
 async function create(event: SubmitEvent) {
  event.preventDefault(); if (busy || loading || !canCreateLimit) return;
  busy = true; error = ''; notice = '';
  try {
   const amount = parseNaira(limit); if (amount <= 0) throw new Error('Enter a valid customer limit.');
   const starts = lagosISO(startAt), ends = lagosISO(endAt);
   if (ends <= starts) throw new Error('The end must be after the start.');
   if (!customers.some(item => item.buyer_user_id === buyerUserID && item.buyer_business_id === buyerBusinessID)) throw new Error('Choose a customer from this business.');
   const url = `/api/v1/organizations/${encodeURIComponent(organizationID)}/trade-lines`;
   let intent = intents.get(url); if (!intent) { intent = new MutationIntent('customer-limit', url); intents.set(url, intent); }
   const saved = await intent.run({ buyer_user_id: buyerUserID, buyer_business_id: buyerBusinessID, mandate_id: mandateID, approved_limit_kobo: amount, cadence, default_grace_hours: Number(grace), start_at: starts, end_at: ends, terms_version: termsVersion }, value => { const line = tradeLine(record(value).trade_line); if (line.supplier_organization_id !== organizationID || line.buyer_user_id !== buyerUserID || String(line.approved_limit_kobo) !== String(amount)) throw new Error('Limit not confirmed'); return line; });
   notice = `Customer limit saved: ${money(saved.approved_limit_kobo)}. Status: ${productLabel(saved.state)}.`;
   buyerUserID = ''; buyerBusinessID = ''; mandateID = ''; limit = ''; await load();
  } catch (cause) { error = cause instanceof Error ? cause.message : 'The limit has not been confirmed.'; }
  finally { busy = false; }
 }
 onMount(() => { void load(); return () => reads.cancel(); });
</script>
<svelte:head><title>Customer limits — Kredit</title></svelte:head>
<main class="shell workspace"><header><p class="eyebrow">Customer limits</p><h1>Let a good customer keep buying.</h1><p class="lede">Set the most one customer may owe you at any time. Every new sale eats into that amount, and paying back frees it up again.</p></header>
	{#if organizations.length>1}<label class="org">Business<select bind:value={organizationID} disabled={busy || loading} onchange={switchBusiness}>{#each organizations as org}<option value={org.id}>{org.trading_name||org.legal_name}</option>{/each}</select></label>{/if}
	{#if error}<p class="error" role="alert">{error} <button onclick={load}>Refresh</button></p>{/if}{#if notice}<p class="notice" role="status">{notice}</p>{/if}
	{#if loading}<p role="status">Checking customer limits…</p>{:else if !organizationID}<p>Add a business before creating customer limits.</p>{:else if loadError}<p>Setup permissions could not be confirmed. Refresh to try again.</p>{:else if canCreateLimit}<section class="card"><h2>Give a customer a limit</h2><p>Your customer must give bank debit permission first.</p><form onsubmit={create}><label>Customer<select disabled={busy} bind:value={buyerUserID} onchange={selectCustomer} required><option value="">Choose a customer</option>{#each customers as customer}<option value={customer.buyer_user_id}>{customer.legal_name||customer.trading_name}</option>{/each}</select></label><label>Customer number<input disabled={busy} bind:value={buyerBusinessID} required /></label><label>Bank debit setup number<input disabled={busy} bind:value={mandateID} required /></label><label>Most they may owe you at once (₦)<input disabled={busy} bind:value={limit} inputmode="decimal" required /></label><label>How often they buy<select disabled={busy} bind:value={cadence}><option value="weekly">Every week</option><option value="monthly">Every month</option><option value="quarterly">Every three months</option></select></label><label>Extra hours you give them to pay<input disabled={busy} bind:value={grace} type="number" min="0" max="720" required /></label><label>Start day and time (Lagos)<input disabled={busy} bind:value={startAt} type="datetime-local" required /></label><label>End day and time (Lagos)<input disabled={busy} bind:value={endAt} type="datetime-local" required /></label><details class="wide"><summary>Setup details</summary><label>Rules number<input disabled={busy} bind:value={termsVersion} required /></label></details><button class="primary wide" disabled={busy||!buyerUserID||!mandateID||!limit||!startAt||!endAt}>{busy?'Saving…':(parseNaira(limit) > 0 ? `Give a ${money(parseNaira(limit))} limit` : 'Save customer limit')}</button></form></section>{:else}<section class="card"><h2>Give a customer a limit</h2><p>Customer limits are switched off for this account, so there is nothing to set up here. You can still record each sale on its own, and any limits already agreed stay listed below.</p><a class="primary" href="/app/credit/new">Record a sale</a></section>{/if}
	<section><h2>Customer limits</h2>{#if loading}<p role="status">Opening your limits…</p>{:else if loadError}<p>Customer limits could not be verified.</p>{:else if lines.length}<div class="lines">{#each lines as line}<article><header><strong>{money(line.available_limit_kobo)} still available</strong><span class="status">{productLabel(line.state)}</span></header><dl><dt>Full limit</dt><dd>{money(line.approved_limit_kobo)}</dd><dt>Money owed now</dt><dd>{money(line.current_exposure_kobo)}</dd><dt>Waiting</dt><dd>{money(line.reserved_pending_kobo)}</dd></dl><a href={`/app/trade-lines/${encodeURIComponent(line.id)}?organization=${encodeURIComponent(organizationID)}`}>Open this limit →</a></article>{/each}</div>{:else}<div class="empty-state"><h3>You have not set any limit yet</h3><p>You can set one once your customer gives bank debit permission.</p></div>{/if}</section>
</main>
<style>h1{font-size:clamp(2.5rem,6vw,4.7rem);line-height:1}.org{display:grid;gap:.3rem;max-width:28rem}.card{margin:1.5rem 0;padding:1.25rem}.card form{display:grid;grid-template-columns:repeat(3,1fr);gap:.8rem}.card label{display:grid;gap:.3rem;font-weight:700}input,select,button{box-sizing:border-box;width:100%;padding:.7rem;border:1px solid var(--color-border);border-radius:.6rem;background:var(--color-surface);color:inherit;font:inherit}.wide{grid-column:1/-1}.notice{padding:1rem;border-left:4px solid var(--color-positive);background:var(--color-surface-muted)}.lines{display:grid;grid-template-columns:repeat(2,1fr);gap:1rem}.lines article{padding:1.2rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}.lines header{display:flex;justify-content:space-between;gap:1rem}.lines dl{display:grid;grid-template-columns:1fr 1fr;gap:.4rem}.lines dd{text-align:right;margin:0}.lines a{color:var(--color-primary);font-weight:700}@media(max-width:760px){.card form,.lines{grid-template-columns:1fr}.wide{grid-column:auto}}</style>
