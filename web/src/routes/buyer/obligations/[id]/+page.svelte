<script lang="ts">
	import { checkedJSON, LatestRequest, record, rows, text, publicError } from '$lib/api/reliable';
 import { kobo } from '$lib/records';
	import { page } from '$app/state';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';
	import { formatKobo } from '$lib/money';
	import { productLabel } from '$lib/product-language';

	let data: any = $state(null);
	let error = $state('');
	let notice = $state('');
	let busy = $state('');

 const reads=new LatestRequest(); const keys=new Map<string,string>();
 function decode(value:unknown){
  const data=record(value),view=record(data.view),request=record(view.request),obligation=record(view.obligation);
  text(request.id);text(request.goods_description);kobo(obligation.outstanding_kobo);text(obligation.payment_status);
  rows('schedule_items',value=>{const item=record(value);text(item.id);text(item.state);kobo(item.principal_due_kobo);kobo(item.allocated_kobo);for(const key of ['due_at','collection_at'])if(!Number.isFinite(Date.parse(text(item[key]))))throw new Error('Invalid payment date');return item;})(data);
  rows('collection_notices',value=>{const item=record(value);text(item.schedule_item_id);text(item.notification_id);if(typeof item.acknowledged!=='boolean')throw new Error('Invalid notice state');return item;})(data);
  rows('payments',value=>{const item=record(value);text(item.id);text(item.state);return item;})(data);
  rows('payment_claims',value=>{const item=record(value);text(item.id);text(item.state);return item;})(data);
  return data;
 }
 async function loadSale(id=page.params.id){
  const request=reads.begin();error='';data=null;notice='';
  try {const result=await checkedJSON(`/api/v1/buyer/obligations/${encodeURIComponent(id??'')}`,decode,{signal:request.signal});if(request.current())data=result;}
  catch(cause){if(request.current())error=publicError(cause,'this sale');}
 }
 $effect(()=>{const id=page.params.id;void loadSale(id);return()=>reads.cancel();});

	async function acknowledgeNotice(itemID: string, notificationID: string) {
		if (busy) return;
		busy = itemID;
  const saleID=page.params.id;
		error = '';
		notice = '';
		try {
			const response = await fetch(`/api/v1/buyer/schedule-items/${encodeURIComponent(itemID)}/collection-notice/acknowledge`, {
				method: 'POST', credentials: 'include', signal: AbortSignal.timeout(20000), redirect: 'error',
				body: JSON.stringify({notification_id: notificationID}),
    headers: { 'Content-Type': 'application/json', 'Idempotency-Key': keys.get(notificationID) ?? (()=>{const key=idempotencyKey();keys.set(notificationID,key);return key;})(), ...csrfHeaders() }
			});
			if (!response.ok) {
				const result = await response.json().catch(() => ({}));
				throw new Error(result.detail ?? 'We could not save your answer. Please try again.');
			}
			if(page.params.id!==saleID)return;
   keys.delete(notificationID);
   data.collection_notices=data.collection_notices.map((item:any)=>item.notification_id===notificationID?{...item,acknowledged:true}:item);
			notice = 'Saved. This only says you saw the notice. It does not mean you have paid, and you can still report a problem.';
		} catch (cause) {
			if(page.params.id===saleID)error = 'We could not confirm your answer. Make sure you received the notice, then retry the same action.';
		} finally {
			busy = '';
		}
	}

	const money = formatKobo;
</script>

<svelte:head><title>Money you owe — Kredit</title></svelte:head>

<main class="shell workspace">
	<p class="eyebrow">Money I owe</p>
	{#if data}
		<h1>{data.view.request.goods_description}</h1>
		{#if notice}<p class="notice" role="status">{notice}</p>{/if}
		{#if error}<p role="alert">{error}</p>{/if}
		<section class="summary">
			<article><span>Money left to pay</span><strong>{money(data.view.obligation.outstanding_kobo)}</strong></article>
			<article><span>Where this stands</span><strong>{productLabel(data.view.obligation.payment_status)}</strong></article>
			<article><span>Pay before</span><strong>{data.schedule_items.find((i:any)=>i.state!=='CANCELLED'&&i.principal_due_kobo>i.allocated_kobo)?new Date(data.schedule_items.find((i:any)=>i.state!=='CANCELLED'&&i.principal_due_kobo>i.allocated_kobo).due_at).toLocaleDateString('en-NG',{timeZone:'Africa/Lagos'}):'Nothing due right now'}</strong></article>
		</section>
		<h2>Your payment days</h2>
		<p><a href="/buyer/amendments">See any changes to your payment days, and what you agreed to →</a></p>
		{#if data.schedule_items.length}
			<div class="table"><table><thead><tr><th>Pay before</th><th>Money to pay</th><th>Money paid</th><th>Where it stands</th><th>Debit notice</th></tr></thead><tbody>
				{#each data.schedule_items as item}
     {@const debitNotice=data.collection_notices.find((n:any)=>n.schedule_item_id===item.id)}
					<tr><td>{new Date(item.due_at).toLocaleDateString('en-NG', { timeZone: 'Africa/Lagos' })}</td><td>{money(item.principal_due_kobo)}</td><td>{money(item.allocated_kobo)}</td><td>{productLabel(item.state)}</td><td>{#if debitNotice}<p>Debit date: {new Date(item.collection_at).toLocaleDateString('en-NG', {timeZone:'Africa/Lagos'})}. Up to {money(item.principal_due_kobo-item.allocated_kobo)} still due.</p>{#if debitNotice.acknowledged}<span>Notice acknowledged</span>{:else}<button class="secondary" disabled={Boolean(busy)} onclick={() => acknowledgeNotice(item.id,debitNotice.notification_id)}>{busy === item.id ? 'Saving…' : 'I have seen this notice'}</button>{/if}{:else if item.state === 'PAID' || item.state === 'CANCELLED'}Not needed{:else}No delivered debit notice yet{/if}</td></tr>
				{/each}
			</tbody></table></div>
		{:else}<p>You pay this sale once, all at one time.</p>{/if}
		<h2>What you have paid</h2>
		<p>{data.payments.filter((p:any)=>p.state==='recognized').length} {data.payments.filter((p:any)=>p.state==='recognized').length===1?'payment':'payments'} confirmed. {data.payment_claims.filter((p:any)=>['pending','expired'].includes(p.state)).length} still waiting for the seller to check their bank.</p>
		<a href={`/buyer/credit-requests/${encodeURIComponent(data.view.request.id)}`}>Pay this sale, or report a problem →</a>
	{:else if error}<h1>We could not open this sale.</h1><p role="alert">{error}</p><button type="button" onclick={()=>loadSale()}>Try again</button>
	{:else}<p>Opening this sale…</p>{/if}
</main>

<style>
	.summary{display:grid;grid-template-columns:repeat(3,1fr);gap:1rem}.summary article{display:grid;gap:.5rem;padding:1rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}.summary span,th{color:var(--color-muted)}.table{overflow-x:auto}table{width:100%;border-collapse:collapse;background:var(--color-surface)}th,td{padding:.8rem;text-align:left;border-bottom:1px solid var(--color-border)}.notice{padding:.8rem;border-radius:.75rem;background:#e6f7ed;color:var(--color-positive)}@media(max-width:700px){.summary{grid-template-columns:1fr}}
</style>
