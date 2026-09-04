<script lang="ts">
 import { onMount } from 'svelte';
 import { csrfHeaders, idempotencyKey } from '$lib/api/client';
 type Review = {id:string;kind:string;target_id:string;expected:string;actual:string;owner_id:string|null;history:{id:string;action:string;reason:string;occurred_at:string}[]};
 let cases:Review[]=$state([]), loading=$state(true),busy=$state(''),error=$state('');
 let reasons:Record<string,string>=$state({});
 const names:Record<string,string>={provider_reversal:'Bank debit reversal',settlement_without_payment:'Settlement without a payment record',ledger:'Journal totals',balance:'Outstanding balance',schedule:'Payment schedule',collection_payment:'Bank debit and payment',settlement:'Supplier settlement',settlement_missing:'Missing settlement evidence'};
 function money(value:string){try{const n=BigInt(value),a=n<0n?-n:n;return `${n<0n?'-':''}₦${(a/100n).toLocaleString('en-NG')}.${(a%100n).toString().padStart(2,'0')}`}catch{return 'Unavailable'}}
 async function load(){loading=true;error='';try{const response=await fetch('/api/v1/ops/financial-reconciliation',{credentials:'include'});if(!response.ok)throw new Error('Financial reviews could not be loaded.');cases=(await response.json()).cases;}catch(e){error=e instanceof Error?e.message:'Financial reviews could not be loaded.'}finally{loading=false}}
 async function decide(item:Review,action:string){busy=item.id;error='';try{const response=await fetch(`/api/v1/ops/financial-reconciliation/${item.id}/decision`,{method:'POST',credentials:'include',headers:{'Content-Type':'application/json','Idempotency-Key':idempotencyKey(),...csrfHeaders()},body:JSON.stringify({action,reason:reasons[item.id]||''})});const body=await response.json();if(!response.ok)throw new Error(body.detail||'The review could not be updated.');await load()}catch(e){error=e instanceof Error?e.message:'The review could not be updated.'}finally{busy=''}}
 onMount(load);
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
 <details><summary>Review history</summary>{#each item.history as event (event.id)}<p>{new Date(event.occurred_at).toLocaleString()} · {event.action.toLowerCase()}: {event.reason}</p>{/each}</details>
 <label for={`reason-${item.id}`}>Investigation notes</label><textarea id={`reason-${item.id}`} bind:value={reasons[item.id]} minlength="8" maxlength="2000" placeholder="Explain what you checked and the outcome"></textarea>
 <div><button disabled={!!busy||(reasons[item.id]||'').trim().length<8} onclick={()=>decide(item,'claim')}>Claim review</button><button disabled={!!busy||!item.owner_id||(reasons[item.id]||'').trim().length<8} onclick={()=>decide(item,'resolve')}>Close resolved review</button></div>
 </article>{/each}
</main>
<style>article{padding:1.5rem;border:1px solid var(--color-border);border-radius:1rem;margin:1.5rem 0}label,textarea{display:block;margin:.75rem 0}textarea{width:100%;min-height:6rem}button{padding:.7rem 1rem;margin:.25rem;border:1px solid var(--color-border);border-radius:.5rem}code{overflow-wrap:anywhere}</style>
