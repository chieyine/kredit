<script lang="ts">
 import { page } from '$app/state';
 import { formatKobo } from '$lib/money';
 import { organization, kobo } from '$lib/records';
 import { checkedJSON, LatestRequest, record, rows, text, publicError, RequestError } from '$lib/api/reliable';
 import { productLabel } from '$lib/product-language';
 import ShareActions from '$lib/components/ShareActions.svelte';
 import { readLocal, writeLocal } from '$lib/product-tools';
 let history = $state<Record<string,any>|null>(null), statement=$state<Record<string,any>|null>(null);
 let error=$state(''),organizationID=$state(''),note=$state(''),noteMessage=$state('');
 const requests=new LatestRequest(), money=formatKobo;
 function decodeHistory(value:unknown){
  const row=record(value);kobo(row.current_active_principal_kobo);
  for(const key of ['active_obligations','completed_obligations'])if(!Number.isSafeInteger(row[key])||Number(row[key])<0)throw new Error('Invalid sale count');
  if(typeof row.on_time_percentage!=='number'||!Number.isFinite(row.on_time_percentage)||row.on_time_percentage<0||row.on_time_percentage>100)throw new Error('Invalid payment history');
  return row;
 }
 function decodeStatement(value:unknown){
  const row=record(value);text(row.buyer_id);
  rows('obligations',value=>{const sale=record(value);text(sale.credit_request_id);text(sale.payment_status);kobo(sale.outstanding_kobo);return sale;})(row);
  return row;
 }
 async function loadCustomer(customerID=page.params.id,selected=page.url.searchParams.get('organization')){
  const request=requests.begin();error='';history=null;statement=null;organizationID='';note='';noteMessage='';
  try{
   const organizations=await checkedJSON('/api/v1/organizations',rows('organizations',organization),{signal:request.signal});
   const candidates=selected?organizations.filter(org=>org.id===selected):organizations;
   for(const org of candidates){
    const base=`/api/v1/organizations/${encodeURIComponent(org.id)}/customers/${encodeURIComponent(customerID??'')}`;
    const results=await Promise.allSettled([checkedJSON(`${base}/history`,decodeHistory,{signal:request.signal}),checkedJSON(`${base}/statement`,decodeStatement,{signal:request.signal})]);
    if(!request.current())return;
    const unavailable=results.find(result=>result.status==='rejected'&&!(result.reason instanceof RequestError&&result.reason.status===404));
    if(unavailable?.status==='rejected')throw unavailable.reason;
    if(results[0].status==='rejected'||results[1].status==='rejected')continue;
    if(results[1].value.buyer_id!==customerID)throw new Error('Wrong customer statement');
    organizationID=org.id;history=results[0].value;statement=results[1].value;
    const saved=readLocal<unknown>(`kredit:customer-note:${org.id}:${customerID}`,'');note=typeof saved==='string'?saved:'';
    return;
   }
   throw new RequestError('We could not find this customer in your business.',404);
  }catch(cause){if(request.current())error=publicError(cause,'this customer');}
 }
 $effect(()=>{void loadCustomer(page.params.id,page.url.searchParams.get('organization'));return()=>requests.cancel();});
 function customerName(){return statement?.obligations?.find((item:any)=>item.buyer_name)?.buyer_name||'This customer';}
 function saveNote(){
  if(!organizationID||!history)return;
  noteMessage=writeLocal(`kredit:customer-note:${organizationID}:${page.params.id}`,note.trim())?'Saved on this phone. It is cleared when you sign out.':'This phone could not save the note. Copy it somewhere safe before leaving.';
 }
</script>
<svelte:head><title>Customer — Kredit</title></svelte:head>
<main class="shell workspace"><a href="/app/customers">← Customers</a><p class="eyebrow">Customer</p><h1>{customerName()}</h1><p class="lede">You only see sales this customer made with you. No other seller can see your records, and you cannot see theirs.</p>
	{#if error}<p class="error" role="alert">{error} <button type="button" onclick={()=>loadCustomer()}>Try again</button></p>{:else if !history}<p>Opening this customer…</p>{:else}<div class="customer-actions"><a class="primary-link" href={`/app/credit/new?customer=${encodeURIComponent(page.params.id ?? '')}&organization=${encodeURIComponent(organizationID)}`}>Sell to them again</a><button type="button" onclick={()=>window.print()}>Print this page</button></div><ShareActions title="Kredit customer statement" text={`Kredit statement: ${money(history.current_active_principal_kobo)} is still owed across ${history.active_obligations ?? 0} open sale(s). ${history.completed_obligations ?? 0} sale(s) fully paid.`}/><section class="stats"><article><span>Sales still open</span><strong>{history.active_obligations ?? 0}</strong></article><article><span>Money they still owe you</span><strong>{money(history.current_active_principal_kobo)}</strong></article><article><span>Sales fully paid</span><strong>{history.completed_obligations ?? 0}</strong></article><article><span>Paid on time</span><strong>{history.completed_obligations ? `${Number(history.on_time_percentage ?? 0).toFixed(0)}%` : 'No record yet'}</strong></article></section><section class="card"><h2>Your private note</h2><p>Directions to their shop, their usual order, anything you need to remember. The note stays on this phone, Kredit never receives it, and signing out removes it.</p><label>Note about this customer<textarea bind:value={note} rows="3" placeholder="For example: delivers to the second shop on Mondays"></textarea></label><button type="button" onclick={saveNote}>Save note</button>{#if noteMessage}<small role="status">{noteMessage}</small>{/if}</section><section class="card"><h2>Sales with you</h2>{#if statement?.obligations?.length}<div class="table">{#each statement.obligations as obligation}<a href={`/app/credit/${encodeURIComponent(obligation.credit_request_id)}?organization=${encodeURIComponent(organizationID)}`}><span>{obligation.buyer_name || 'Credit sale'}</span><strong>{money(obligation.outstanding_kobo)}</strong><small>{productLabel(obligation.payment_status)}</small></a>{/each}</div>{:else}<p>This customer has no open sale with you right now.</p>{/if}</section>{/if}
</main>
<style>.customer-actions{display:flex;flex-wrap:wrap;gap:.7rem;margin-top:1.25rem}.customer-actions a,.customer-actions button,.card button{padding:.7rem .9rem;border:1px solid var(--color-foreground);background:var(--color-surface);color:inherit;font:inherit;font-weight:750;text-decoration:none}.customer-actions .primary-link{border-color:var(--color-primary);background:var(--color-primary);color:#fff}.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(11rem,1fr));gap:1rem;margin:2rem 0}.stats article,.card{padding:1.2rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}.card{margin-top:1rem}.card label{display:grid;gap:.4rem;font-weight:750}.card textarea{box-sizing:border-box;width:100%;margin:.4rem 0;padding:.75rem;border:1px solid var(--color-border);font:inherit}.stats span{display:block;color:var(--color-muted)}.stats strong{display:block;font-size:1.45rem;margin-top:.4rem}.table{display:grid}.table a{display:grid;grid-template-columns:1fr auto auto;gap:1rem;padding:.9rem 0;border-bottom:1px solid var(--color-border);color:inherit;text-decoration:none}.table small{color:var(--color-muted)}.error{color:#b42318}@media print{.customer-actions,:global(.share-actions){display:none!important}}@media(max-width:600px){.table a{grid-template-columns:1fr auto}.table small{grid-column:1/-1}}</style>
