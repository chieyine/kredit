<script lang="ts">
 import { onMount } from 'svelte';
 import { checkedJSON, LatestRequest, publicError, record, rows, text } from '$lib/api/reliable';
 import { MutationIntent } from '$lib/api/mutation';
 import { productLabel } from '$lib/product-language';
 import Money from '$lib/components/Money.svelte';
 let organizations: any[] = $state([]), organizationID = $state(''), events: any[] = $state([]), actions: any[] = $state([]), corrections: any[] = $state([]), provider: any = $state(null), readiness: any = $state(null);
 let loading = $state(true), error = $state(''), notice = $state(''), busy = $state(''), decisionReasons = $state<Record<string,string>>({}), unavailable: string[] = $state([]);
 const reads = new LatestRequest(), intents = new Map<string, MutationIntent>();
 const names = ['audit-events','operations','corrections','provider-status','onboarding'];
 const labels = ['Business activity','Corrections you made','Correction requests','Payment provider','Setup status'];
 function item(value: unknown, fields: string[]) { const row = record(value); for (const field of fields) text(row[field]); return row; }
 function decode(index: number, value: unknown): any {
  if (index === 0) return rows('events', v => item(v, ['action','created_at']))(value);
  if (index === 1) return rows('actions', v => { const row = item(v, ['id','action_type','reason','created_at']); if (!(typeof row.amount_kobo === 'string' && /^-?\d+$/.test(row.amount_kobo)) && !(typeof row.amount_kobo === 'number' && Number.isSafeInteger(row.amount_kobo))) throw new Error('Invalid amount'); return row; })(value);
  if (index === 2) return rows('corrections', v => item(v, ['id','subject_type','state','reason','created_at']))(value);
  const result = index === 4 ? record(record(value).readiness) : record(value);
  if (index === 3) { text(result.name); if (typeof result.feature_enabled !== 'boolean' || typeof record(result.health).healthy !== 'boolean') throw new Error('Incomplete provider status'); }
  else if (typeof result.ready !== 'boolean' || typeof result.state !== 'string') throw new Error('Incomplete readiness');
  return result;
 }
 async function load() {
  const request = reads.begin(); loading = true; error = ''; unavailable = []; events = []; actions = []; corrections = []; provider = null; readiness = null;
  try {
   if (!organizationID) return;
   const result = await Promise.allSettled(names.map((name, index) => checkedJSON(`/api/v1/organizations/${encodeURIComponent(organizationID)}/${name}`, value => decode(index, value), { signal: request.signal })));
   if (!request.current()) return;
   unavailable = result.flatMap((entry, index) => entry.status === 'rejected' ? [labels[index]] : []);
   const values = result.map(entry => entry.status === 'fulfilled' ? entry.value : null);
   [events, actions, corrections] = values.slice(0,3).map(value => value ?? []); provider = values[3]; readiness = values[4];
  } finally { if (request.current()) loading = false; }
 }
 async function initialize() {
  const request = reads.begin(); loading = true; error = '';
  try {
   const result = await checkedJSON('/api/v1/organizations', rows('organizations', value => item(value, ['id','legal_name'])), { signal: request.signal });
   if (!request.current()) return;
   organizations = result; organizationID = String(result[0]?.id ?? ''); await load();
  } catch (cause) { if (request.current()) { error = publicError(cause, 'your businesses'); loading = false; } }
 }
 async function decide(item: any, outcome: string) {
  if (busy) return;
  const reason = (decisionReasons[item.id] ?? '').trim();
  if (outcome !== 'UNDER_REVIEW' && reason.length < 8) { error = 'Please explain this decision in at least 8 characters.'; return; }
  busy = item.id; error = ''; notice = '';
  const url = `/api/v1/organizations/${encodeURIComponent(organizationID)}/corrections/${encodeURIComponent(item.id)}/decide`;
  let intent = intents.get(url); if (!intent) { intent = new MutationIntent('correction-decision', url); intents.set(url, intent); }
  try {
   await intent.run({ outcome, reason }, value => { const result = record(value); const correction = record(result.correction); if (text(correction.id) !== item.id || text(correction.state) !== outcome) throw new Error('Decision not confirmed'); return result; });
   delete decisionReasons[item.id]; notice = 'Your decision was saved.'; await load();
  } catch (cause) { error = cause instanceof Error ? cause.message : 'We could not confirm this decision.'; }
  finally { busy = ''; }
 }
 onMount(() => { void initialize(); return () => reads.cancel(); });
</script>
<svelte:head><title>Business activity — Kredit</title></svelte:head>
<main class="shell workspace activity"><p class="eyebrow">Business activity</p><h1>Every important change, in one place.</h1><p class="lede">Corrections, money changes, who did what, and whether bank debit is working right now.</p>{#if organizations.length>1}<label class="business">Business<select disabled={!!busy || loading} bind:value={organizationID} onchange={load}>{#each organizations as item}<option value={item.id}>{item.trading_name||item.legal_name}</option>{/each}</select></label>{/if}{#if error}<p class="error" role="alert">{error} <button type="button" disabled={!!busy || loading} onclick={() => organizations.length ? load() : initialize()}>Try again</button></p>{/if}{#if unavailable.length}<p class="unavailable" role="status">We could not check {unavailable.join(', ').toLowerCase()}. What is shown below is everything else. <button type="button" onclick={load}>Check again</button></p>{/if}{#if notice}<p class="notice" role="status">{notice}</p>{/if}{#if loading}<p role="status">Opening your business activity…</p>{:else if !organizationID}<p>Create a business account to see its activity.</p>{:else}
	<section class="status-grid"><article><span>Is bank debit working?</span><strong>{!provider?'Unavailable':provider.feature_enabled&&provider.health.healthy?'Working':'Needs checking'}</strong><small>{provider?.name||'Provider status could not be checked'}</small></article><article><span>Is your account ready?</span><strong>{!readiness?'Unavailable':readiness.ready?'Ready':productLabel(readiness.state,'Check setup')}</strong><a href="/app/onboarding">Finish setting up →</a></article></section>
	<section class="card"><h2>Customers asking you to fix something</h2><p>A customer can ask you to correct a wrong amount, payment or sale detail. Their request lands here. Approval adds your reviewed note to their history; record any money adjustment through the protected financial-change process.</p>{#if unavailable.includes('Correction requests')}<p>Correction requests could not be loaded.</p>{:else if corrections.length}<div class="corrections">{#each corrections as item}<article><div><strong>{productLabel(item.subject_type)} correction</strong><span>{productLabel(item.state)}</span></div><p>{item.reason}</p><small>Requested {new Date(item.created_at).toLocaleString('en-NG')}</small>{#if ['OPEN','UNDER_REVIEW'].includes(item.state)}<label>Why are you deciding this?<textarea disabled={!!busy} bind:value={decisionReasons[item.id]} rows="2"></textarea></label><div class="buttons">{#if item.state==='OPEN'}<button disabled={!!busy} onclick={()=>decide(item,'UNDER_REVIEW')}>I am looking into it</button>{/if}<button disabled={!!busy} onclick={()=>decide(item,'APPROVED')}>Approve correction note</button><button class="secondary" disabled={!!busy} onclick={()=>decide(item,'REJECTED')}>No, it is correct as it is</button></div>{/if}</article>{/each}</div>{:else}<p>No customer has asked you to fix anything.</p>{/if}</section>
	<section class="columns"><article class="card"><h2>Money written off or reduced</h2>{#if unavailable.includes('Corrections you made')}<p>Money changes could not be loaded.</p>{:else if actions.length}<ul>{#each actions as item}<li><strong>{productLabel(item.action_type)}</strong> · <Money amountKobo={item.amount_kobo}/><small>{item.reason} · {new Date(item.created_at).toLocaleString('en-NG')}</small></li>{/each}</ul>{:else}<p>You have not written off any money or changed any fee.</p>{/if}</article><article class="card"><h2>Who did what</h2>{#if unavailable.includes('Business activity')}<p>Business activity could not be loaded.</p>{:else if events.length}<ul>{#each events.slice(0,30) as item}<li><strong>{productLabel(item.action)}</strong><small>{new Date(item.created_at).toLocaleString('en-NG')}</small></li>{/each}</ul>{:else}<p>Nothing to show yet.</p>{/if}</article></section>{/if}</main>
<style>.unavailable{padding:.85rem 1rem;border-left:3px solid var(--color-warning);background:#fff4e5;line-height:1.6}.unavailable button{margin-left:.5rem;padding:.4rem .7rem;border:1px solid currentColor;background:transparent;color:inherit;font:inherit}.activity{max-width:68rem}.business{display:grid;gap:.35rem;max-width:22rem;margin:1rem 0}.business select,textarea{padding:.7rem;border:1px solid var(--color-border);font:inherit}.status-grid,.columns{display:grid;grid-template-columns:1fr 1fr;gap:1rem;margin:1.5rem 0}.status-grid article,.card{padding:1.2rem;border:1px solid var(--color-border);background:var(--color-surface)}.status-grid span,.status-grid strong,.status-grid small{display:block}.status-grid strong{margin:.4rem 0;font-size:1.35rem}.corrections,.card ul{display:grid;gap:.7rem;padding:0;list-style:none}.corrections>article,.card li{padding:.8rem;border-left:4px solid var(--color-primary);background:var(--color-surface-muted)}.corrections>article>div{display:flex;justify-content:space-between;gap:1rem}.corrections label{display:grid;gap:.35rem}.buttons{display:flex;flex-wrap:wrap;gap:.5rem;margin-top:.6rem}.buttons button{padding:.6rem;border:1px solid var(--color-primary);background:var(--color-primary);color:#fff;font:inherit;font-weight:750}.buttons .secondary{border-color:var(--color-foreground);background:var(--color-surface);color:var(--color-foreground)}.card small{display:block;margin-top:.3rem;color:var(--color-muted)}.notice{padding:.8rem;border-left:4px solid var(--color-positive)}.error{color:var(--color-destructive)}@media(max-width:720px){.status-grid,.columns{grid-template-columns:1fr}}</style>
