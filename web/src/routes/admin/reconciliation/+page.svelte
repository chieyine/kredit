<script lang="ts">
 import { onMount } from 'svelte';
 import { localTime } from '$lib/admin-client';
 import { checkedJSON, record, rows, text, publicError, LatestRequest } from '$lib/api/reliable';
 import { MutationIntent } from '$lib/api/mutation';
 type Review = {id:string;kind:string;target_id:string;expected:string;actual:string;owner_id:string|null;history:{id:string;action:string;reason:string;occurred_at:string}[]};
 let cases:Review[]=$state([]), loading=$state(true),busy=$state(''),error=$state('');
 let reasons:Record<string,string>=$state({});
 const requests = new LatestRequest();
 const intents = new Map<string, MutationIntent>();
 const names:Record<string,string>={provider_reversal:'Bank debit reversal',settlement_without_payment:'Settlement without a payment record',ledger:'Journal totals',balance:'Outstanding balance',schedule:'Payment schedule',collection_payment:'Bank debit and payment',settlement:'Supplier settlement',settlement_missing:'Missing settlement evidence'};
 function money(value:string){try{const n=BigInt(value),a=n<0n?-n:n;return `${n<0n?'-':''}₦${(a/100n).toLocaleString('en-NG')}.${(a%100n).toString().padStart(2,'0')}`}catch{return 'Unavailable'}}
 function review(value:unknown):Review {
  const item=record(value);
  for(const key of ['id','kind','target_id','expected','actual']) text(item[key]);
  if(!/^-?\d+$/.test(item.expected as string)||!(/^-?\d+$/.test(item.actual as string))) throw new Error('Financial amounts were incomplete');
  if(item.owner_id!==null) text(item.owner_id);
  rows('history',value=>{const event=record(value);for(const key of ['id','action','reason','occurred_at'])text(event[key]);return event;})(item);
  return item as Review;
 }
 async function load(){
  const request=requests.begin();loading=true;error='';
  try{const result=await checkedJSON('/api/v1/ops/financial-reconciliation',rows('cases',review),{signal:request.signal});if(request.current())cases=result;}
  catch(cause){if(request.current()){cases=[];error=publicError(cause,'financial reviews');}}
  finally{if(request.current())loading=false;}
 }
 async function decide(item:Review,action:string){
  if(busy)return;busy=item.id;error='';
  try{
   const url=`/api/v1/ops/financial-reconciliation/${encodeURIComponent(item.id)}/decision`;
   let intent=intents.get(item.id);if(!intent){intent=new MutationIntent('financial-review',url);intents.set(item.id,intent);}
   await intent.run({action,reason:reasons[item.id]||''},value=>{if(record(value).status!=='applied')throw new Error('Review was not confirmed');return true;});
   await load();
  }catch(cause){error=cause instanceof Error?cause.message:'The review could not be updated.';}finally{busy='';}
 }
 onMount(()=>{void load();return()=>requests.cancel();});
</script>
<svelte:head><title>Financial reviews — Kredit</title></svelte:head>
<main class="shell workspace">
 <p class="eyebrow">Operations / Financial reviews</p><h1>Resolve financial differences.</h1>
 <p>Claim a review, investigate the underlying records, and record the outcome. A review can close only after the records agree.</p>
 <button onclick={load} disabled={loading||!!busy}>Refresh reviews</button>
 {#if error}<p role="alert" class="error">{error}</p>{/if}
 {#if loading}<p>Loading financial reviews…</p>{:else if !error && cases.length===0}<p>No open financial reviews.</p>{/if}
 {#each cases as item (item.id)}
 <article><h2>{names[item.kind]||'Financial difference'}</h2><p>Reference: <code>{item.target_id}</code></p>
 <p>Expected: <strong>{money(item.expected)}</strong> · Recorded: <strong>{money(item.actual)}</strong></p>
 <p>{item.owner_id?'Assigned to a reviewer':'Awaiting a reviewer'}</p>
 <details><summary>Review history</summary>{#each item.history as event (event.id)}<p>{localTime(event.occurred_at)} · {event.action.toLowerCase()}: {event.reason}</p>{/each}</details>
 <label for={`reason-${item.id}`}>Investigation notes</label><textarea disabled={!!busy} id={`reason-${item.id}`} bind:value={reasons[item.id]} minlength="8" maxlength="2000" placeholder="Explain what you checked and the outcome"></textarea>
 <div><button disabled={!!busy||(reasons[item.id]||'').trim().length<8} onclick={()=>decide(item,'claim')}>Take this review</button><button disabled={!!busy||!item.owner_id||(reasons[item.id]||'').trim().length<8} onclick={()=>decide(item,'resolve')}>Close this review</button></div>
 </article>{/each}
</main>
<style>article{padding:1.5rem;border:1px solid var(--color-border);border-radius:1rem;margin:1.5rem 0}label,textarea{display:block;margin:.75rem 0}textarea{width:100%;min-height:6rem}button{padding:.7rem 1rem;margin:.25rem;border:1px solid var(--color-border);border-radius:.5rem}code{overflow-wrap:anywhere}</style>
