<script lang="ts">
 import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
 import { onMount } from 'svelte';
 import { checkedJSON, LatestRequest, publicError, record, rows, text } from '$lib/api/reliable';
 import { MutationIntent } from '$lib/api/mutation';
 import { kobo } from '$lib/records';
 import { localTime } from '$lib/admin-client';
 import { exactKobo, formatKobo } from '$lib/money';
 let changes:any[]=$state([]),busy=$state(false),loading=$state(true),error=$state(''),message=$state(''),consents:Record<string,boolean>=$state({}),offset=$state(0),more=$state(false);
 const reads = new LatestRequest(), intents = new Map<string, MutationIntent>();
 function changeRecord(value: unknown) {
  const change = record(value);
  for (const key of ['id','obligation_id','state','reason','expires_at']) if (!text(change[key])) throw new Error('Incomplete date change');
  if (!Number.isFinite(Date.parse(String(change.expires_at))) || !Array.isArray(change.items) || !Array.isArray(change.dates)) throw new Error('Incomplete date evidence');
  const items = new Map<string, Record<string, unknown>>();
  for (const value of change.items) {
   const item = record(value), id = text(item.id);
   if (!id || items.has(id) || !Number.isFinite(Date.parse(text(item.due_at)))) throw new Error('Invalid original payment day');
   kobo(item.principal_due_kobo); kobo(item.allocated_kobo); items.set(id, item);
  }
  const seen = new Set<string>();
  const dates = change.dates.map(value => {
   const date = record(value), id = text(date.item_id), item = items.get(id);
   if (!item || seen.has(id) || !Number.isFinite(Date.parse(text(date.due_at)))) throw new Error('Invalid proposed payment day');
   seen.add(id);
   const unpaid = exactKobo(kobo(item.principal_due_kobo))! - exactKobo(kobo(item.allocated_kobo))!;
   if (unpaid < 0n) throw new Error('Invalid unpaid balance');
   return {...date, unpaid_kobo: unpaid, old_due_at: item.due_at};
  });
  if (!dates.length) throw new Error('Missing proposed dates');
  return {...change, dates};
 }
 async function load() {
  const read = reads.begin(); loading=true;error='';changes=[];consents={};
  try {
   const result = await checkedJSON(`/api/v1/buyer/amendments?offset=${offset}`, rows('changes', changeRecord), {signal: read.signal});
   if (read.current()) { changes=result.slice(0,100);more=result.length>100; }
  } catch(cause) { if(read.current())error=publicError(cause,'your proposed payment dates'); }
  finally { if(read.current())loading=false; }
 }
 async function decide(change:any,action:'accept'|'reject') {
  if (busy || loading || change.state!=='awaiting_buyer' || (action==='accept'&&!consents[change.id])) return;
  busy=true;error='';message='';
  const path=`/api/v1/buyer/amendments/${encodeURIComponent(change.id)}/decision`;
  try {
   if(!intents.has(path))intents.set(path,new MutationIntent('buyer-payment-date-decision',path));
   await intents.get(path)!.run({action,reason:action==='accept'?'I accept the exact dates shown on this page.':'I do not accept the new dates.'},value=>{
    if(record(value).recorded!==true)throw new Error('Decision not confirmed');return true;
   });
   message=action==='accept'?'Your acceptance was confirmed. The new payment days now apply.':'Your rejection was confirmed. The old payment days still stand.';
   await load();
  }catch(cause){error=cause instanceof Error?cause.message:'We could not confirm your decision.';}
  finally{busy=false;}
 }
 onMount(()=>{void load();return()=>reads.cancel();});
</script>
<svelte:head><title>Changes to your payment days — Kredit</title></svelte:head>
<main class="shell workspace"><h1>Changes to your payment days</h1><VerifyIdentity/><p>Read every new date before you decide. Until you accept, your old payment days still stand. The money you owe and the agreed fee do not change.</p>{#if error}<p role="alert">{error}</p>{/if}{#if message}<p role="status">{message}</p>{/if}{#each changes as c}<article><h2>{c.state.replaceAll('_',' ')}</h2><p>{c.reason}</p><p>Reference: {c.obligation_id} · Expires {localTime(c.expires_at)}</p><table><thead><tr><th>Money still unpaid</th><th>Old date</th><th>New date being asked for</th></tr></thead><tbody>{#each c.dates as d}<tr><td>{formatKobo(d.unpaid_kobo)}</td><td>{localTime(d.old_due_at)}</td><td>{localTime(d.due_at)}</td></tr>{/each}</tbody></table>{#if c.state==='awaiting_buyer'}<label><input type="checkbox" bind:checked={consents[c.id]} disabled={busy||loading}/> I have read these exact dates and I agree to them.</label><button disabled={busy||loading||!consents[c.id]} onclick={()=>decide(c,'accept')}>Yes, I accept these dates</button><button disabled={busy||loading} onclick={()=>decide(c,'reject')}>No, keep my old dates</button>{/if}</article>{:else}<p>{loading?'Loading date changes…':error?'Date changes could not be loaded.':'No repayment date changes to review.'}</p>{/each}<button disabled={busy||loading||!offset} onclick={()=>{offset=Math.max(0,offset-100);load()}}>Previous</button><button disabled={busy||loading||!more} onclick={()=>{offset+=100;load()}}>Next</button>{#if error}<button disabled={busy||loading} onclick={load}>Reload date changes</button>{/if}</main>
<style>article{padding:1.2rem;border:1px solid #ccc;margin:1rem 0;overflow:auto}td,th{padding:.6rem;text-align:left;border-bottom:1px solid #ccc}label{display:block;padding:1rem 0}button{padding:.7rem;margin:.5rem .5rem .5rem 0}p{overflow-wrap:anywhere}</style>
