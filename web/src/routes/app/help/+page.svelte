<script lang="ts">
 import { onMount } from 'svelte';
 import { checkedJSON, LatestRequest, publicError, record, rows, text } from '$lib/api/reliable';
 import { MutationIntent } from '$lib/api/mutation';
 import { productLabel } from '$lib/product-language';
 type Organization = { id: string; legal_name: string; trading_name?: string };
 type Case = { id: string; state: string; created_at: string };
 let organizations = $state<Organization[]>([]), organizationID = $state(''), cases = $state<Case[]>([]);
 let message = $state(''), notice = $state(''), error = $state(''), busy = $state(false), loading = $state(true);
 const reads = new LatestRequest(), intents = new Map<string, MutationIntent>();
 function decodeCase(value: unknown): Case { const item = record(value); const result = { id: text(item.id), state: text(item.state), created_at: text(item.created_at) }; if (!result.id || !['OPEN','IN_PROGRESS','RESOLVED','CLOSED'].includes(result.state) || !Number.isFinite(Date.parse(result.created_at))) throw new Error('Incomplete case'); return result; }
 function intent(url: string) { let item = intents.get(url); if (!item) { item = new MutationIntent('support', url); intents.set(url, item); } return item; }
 async function loadCases() {
  const request = reads.begin(); loading = true; error = ''; cases = [];
  try {
   if (!organizationID) return;
   const result = await checkedJSON(`/api/v1/organizations/${encodeURIComponent(organizationID)}/support-cases`, rows('cases', decodeCase), { signal: request.signal });
   if (request.current()) cases = result;
  } catch (cause) { if (request.current()) error = publicError(cause, 'your help requests'); }
  finally { if (request.current()) loading = false; }
 }
 async function initialize() {
  const request = reads.begin(); loading = true; error = '';
  try {
   const result = await checkedJSON('/api/v1/organizations', rows('organizations', value => { const item = record(value); return { id: text(item.id), legal_name: text(item.legal_name), trading_name: item.trading_name === undefined ? '' : text(item.trading_name) }; }), { signal: request.signal });
   if (!request.current()) return;
   organizations = result; organizationID = result[0]?.id ?? ''; await loadCases();
  } catch (cause) { if (request.current()) { error = publicError(cause, 'your businesses'); loading = false; } }
 }
 async function send(event: SubmitEvent) {
  event.preventDefault(); if (busy || !organizationID || message.trim().length < 5) return;
  busy = true; error = ''; notice = '';
  const url = `/api/v1/organizations/${encodeURIComponent(organizationID)}/support-cases`;
  try {
   const item = await intent(url).run({ subject_type: 'business_account', subject_id: organizationID, message }, value => decodeCase(record(value).case));
   message = ''; notice = `Your message was saved. Help number: ${item.id}`; await loadCases();
  } catch (cause) { error = cause instanceof Error ? cause.message : 'We could not confirm your message. Retry with the same details.'; }
  finally { busy = false; }
 }
 async function updateCase(item: Case, state: string) {
  if (busy) return; busy = true; error = ''; notice = '';
  const url = `/api/v1/organizations/${encodeURIComponent(organizationID)}/support-cases/${encodeURIComponent(item.id)}`;
  try {
   await intent(url).run({ state, note: state === 'RESOLVED' ? 'Customer says the problem is solved.' : 'Customer closed this help request.' }, value => { const updated = decodeCase(record(value).case); if (updated.id !== item.id || updated.state !== state) throw new Error('Update not confirmed'); return updated; }, 'PATCH');
   notice = state === 'RESOLVED' ? 'Marked as solved. You can close it after checking everything.' : 'Help request closed.'; await loadCases();
  } catch (cause) { error = cause instanceof Error ? cause.message : 'We could not confirm this update.'; }
  finally { busy = false; }
 }
 onMount(() => { void initialize(); return () => reads.cancel(); });
</script>
<svelte:head><title>Get help — Kredit</title></svelte:head>
<main class="shell workspace help"><p class="eyebrow">Get help</p><h1>Talk to a real person.</h1><p class="lede">Tell us what happened in your own words. A real person will look at your Kredit records and reply to you safely.</p>
{#if notice}<p class="notice" role="status">{notice}</p>{/if}{#if error}<p class="error" role="alert">{error} <button type="button" disabled={busy || loading} onclick={() => organizations.length ? loadCases() : initialize()}>Try again</button></p>{/if}
<section class="card"><h2>Send a message</h2>{#if organizations.length>1}<label>Business<select disabled={busy || loading} bind:value={organizationID} onchange={loadCases}>{#each organizations as organization}<option value={organization.id}>{organization.trading_name||organization.legal_name}</option>{/each}</select></label>{/if}<form onsubmit={send}><label>What do you need help with?<textarea disabled={busy || loading} bind:value={message} maxlength="2000" rows="6" placeholder="For example: My customer paid, but I cannot see the payment." required></textarea></label><button class="primary" disabled={busy||loading||!organizationID||message.trim().length<5}>{busy?'Sending…':'Send to support'}</button></form><p><small>Never send your password, verification code, bank PIN or full card number.</small></p></section>
<section class="card"><h2>Your help requests</h2>{#if loading}<p role="status">Opening your help requests…</p>{:else if error}<p>Your help requests have not been verified. Try again above.</p>{:else if !organizationID}<p>Create a business account to send and track help requests, or use the complaint link below.</p>{:else if cases.length}<div class="cases">{#each cases as item}<article><strong>Help number {item.id}</strong><span>{productLabel(item.state)}</span><small>Sent {new Date(item.created_at).toLocaleDateString('en-NG')}</small>{#if item.state==='RESOLVED'}<button disabled={busy} onclick={()=>updateCase(item,'CLOSED')}>Close this request</button>{:else if item.state!=='CLOSED'}<button disabled={busy} onclick={()=>updateCase(item,'RESOLVED')}>My problem is solved</button>{/if}</article>{/each}</div>{:else}<p>You have not asked for help yet.</p>{/if}</section>
<p><a href="/legal/complaints">Make a formal complaint →</a></p></main>
<style>.help{max-width:54rem}.help h1{font-family:var(--font-serif);font-size:clamp(3rem,7vw,5rem);line-height:.95}.card,form,label,.cases{display:grid;gap:.75rem}textarea,select{box-sizing:border-box;width:100%;padding:.8rem;border:1px solid var(--color-border);font:inherit}.cases article{display:grid;grid-template-columns:1fr auto;gap:.35rem;padding:.9rem 0;border-bottom:1px solid var(--color-border)}.cases small{grid-column:1/-1}.cases button{grid-column:1/-1;width:max-content;padding:.55rem .7rem;border:1px solid var(--color-border);background:var(--color-surface);font:inherit;font-weight:750}.notice{padding:1rem;border-left:4px solid var(--color-positive);background:#eef8f2}.error{color:var(--color-destructive)}</style>
