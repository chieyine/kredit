<script lang="ts">
 import {feeDisclosure} from "$lib/fee-terms";
 import { parseNaira } from '$lib/money';
  import { page } from '$app/state';
  import { onMount } from 'svelte';
  import { csrfHeaders, idempotencyKey } from '$lib/api/client';
  import Money from '$lib/components/Money.svelte';
  import { productLabel } from '$lib/product-language';
  let view: any = $state(null);
  let payments: any[] = $state([]);
  let message = $state('');
  let mandateKey = $state('');
  let acceptKey = $state('');
  let receiptKey = $state('');
  let receiptIssue = $state(''), receiptBusy = $state(false);
  async function recordReceipt(state: 'confirmed' | 'issue_raised') {
    if(receiptBusy || (state==='issue_raised'&&!receiptIssue.trim()))return;
    receiptBusy=true; message='';
    try {
      const response=await fetch(`/api/v1/buyer/credit-requests/${requestID}/receipt`,{method:'POST',credentials:'include',headers:{'Content-Type':'application/json','Idempotency-Key':idempotencyKey(),...csrfHeaders()},body:JSON.stringify({state,issue_reason:state==='issue_raised'?receiptIssue:undefined})});
      const result=await response.json().catch(()=>({}));
      message=response.ok?(state==='confirmed'?'Receipt confirmed.':'Your receipt problem was reported.'):result.detail??'We could not save your answer.';
      if(response.ok)await load();
    } catch {message='We could not reach Kredit. Please check your connection.'} finally {receiptBusy=false}
  }
  let claimAmount = $state('');
  let claimReference = $state('');
  let paymentURL = $state('');
  let disputeAmount = $state(''), disputeReason = $state(''), disputeExplanation = $state(''), disputeEffect = $state('CONTESTED_ONLY');
  const requestID = $derived(page.params.requestID);
  async function load() { const response = await fetch(`/api/v1/buyer/credit-requests/${requestID}`); if (response.ok) view = await response.json(); const paymentResponse = await fetch(`/api/v1/buyer/credit-requests/${requestID}/payments`); if (paymentResponse.ok) payments = (await paymentResponse.json()).payments; }
  async function authorizeAndAccept() {
    message = 'Setting up bank debit…';
    if (!mandateKey) mandateKey = idempotencyKey();
    const mandateResponse = await fetch(`/api/v1/buyer/credit-requests/${requestID}/mandate`, { method: 'POST', headers: { 'Idempotency-Key': mandateKey, ...csrfHeaders() } });
    if (!mandateResponse.ok) { message = 'We could not set up bank debit.'; return; }
    const mandated = await mandateResponse.json();
    if (!acceptKey) acceptKey = idempotencyKey();
    const response = await fetch(`/api/v1/buyer/credit-requests/${requestID}/accept`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'Idempotency-Key': acceptKey, ...csrfHeaders() }, body: JSON.stringify({ agreement_version_id: mandated.agreement.id, agreement_hash: mandated.agreement.document_hash, mandate_provider_id: mandated.mandate.provider_id }) });
    if (response.ok) { mandateKey = ''; acceptKey = ''; }
    message = response.ok ? 'You said yes. The seller can now send the goods.' : 'We could not save your answer.'; await load();
  }
  async function decline() {
    message = 'Declining request…';
    const response = await fetch(`/api/v1/buyer/credit-requests/${requestID}/decline`, { method: 'POST', headers: { 'Idempotency-Key': idempotencyKey(), ...csrfHeaders() } });
    const result = await response.json().catch(() => ({}));
    message = response.ok ? 'You said no. This sale will not start.' : (result.detail ?? 'We could not save your answer.');
    await load();
  }
  async function claimPayment() {
    const amountKobo = parseNaira(claimAmount);
    if(amountKobo<=0 || !claimReference.trim()){message='Enter the amount and transfer reference.';return;}
    const response=await fetch(`/api/v1/buyer/credit-requests/${requestID}/payment-claims`,{method:'POST',credentials:'include',headers:{'Content-Type':'application/json','Idempotency-Key':idempotencyKey(),...csrfHeaders()},body:JSON.stringify({amount_kobo:amountKobo,paid_at:new Date().toISOString(),transfer_reference:claimReference})});
    const result=await response.json().catch(()=>({})); message=response.ok?'We told the seller. They will check their bank account.':(result.detail??'We could not send this payment.'); if(response.ok){claimAmount='';claimReference='';}
  }
  async function createPaymentLink(){const response=await fetch(`/api/v1/buyer/credit-requests/${requestID}/payment-link`,{method:'POST',credentials:'include',headers:{'Idempotency-Key':idempotencyKey(),...csrfHeaders()}});const result=await response.json().catch(()=>({}));if(response.ok)paymentURL=result.payment_url;else message=result.detail??'Payment link could not be created.';}
  async function openDispute(event:SubmitEvent){event.preventDefault();const disputed_amount_kobo=parseNaira(disputeAmount);if(disputed_amount_kobo<=0||!disputeReason.trim()){message='Enter the money affected and what went wrong.';return}const response=await fetch(`/api/v1/buyer/credit-requests/${requestID}/disputes`,{method:'POST',credentials:'include',headers:{'Content-Type':'application/json','Idempotency-Key':idempotencyKey(),...csrfHeaders()},body:JSON.stringify({disputed_amount_kobo,reason:disputeReason,explanation:disputeExplanation,collection_effect:disputeEffect})});const result=await response.json().catch(()=>({}));if(!response.ok){message=result.detail??'We could not report this problem.';return}message='Your problem was reported. Future debit requests follow the collection option you selected. A debit already submitted may still complete.';disputeAmount='';disputeReason='';disputeExplanation='';await load()}
  onMount(load);
</script>

<svelte:head><title>Check this sale — Kredit</title></svelte:head>
<main>
  <h1>Check this sale before you say yes</h1>
  {#if view}
    <section class="terms"><h2>The sale</h2>{#if view.obligation}<p>These are your original agreement terms. <a href={`/buyer/obligations/${view.obligation.id}`}>View your current repayment schedule</a>.</p>{/if}<dl><dt>Seller</dt><dd>{view.request.supplier_legal_name}</dd><dt>Customer</dt><dd>{view.request.buyer_legal_name}</dd><dt>Money to pay</dt><dd><Money amountKobo={view.request.principal_kobo} /></dd><dt>Goods</dt><dd>{view.request.goods_description}</dd><dt>{view.obligation?'Original first payment day':'First payment day'}</dt><dd>{view.request.due_date}</dd><dt>{view.obligation?'Original bank debit date':'Bank debit after'}</dt><dd>{new Date(view.request.collection_at).toLocaleString('en-NG',{timeZone:'Africa/Lagos',timeZoneName:'short'})}</dd><dt>Extra time</dt><dd>{view.request.grace_hours} hours</dd><dt>How to pay</dt><dd>{view.request.schedule_type==='equal'?`${view.request.schedule_count} ${view.request.schedule_cadence} payments`:view.request.schedule_type==='custom'?'Different payment amounts':'One payment'}</dd><dt>Kredit fee</dt><dd>{feeDisclosure(view.request.fee_terms)}</dd><dt>Sale number</dt><dd>{view.request.id}</dd></dl>{#if view.request.custom_schedule_items?.length}<h3>Payment days</h3><ol>{#each view.request.custom_schedule_items as item}<li><Money amountKobo={item.amount_kobo} /> by {item.due_date}</li>{/each}</ol>{/if}<details><summary>Technical record</summary><p>Sale record code<br/><code>{view.agreement.document_hash}</code></p></details></section>
    <h2>Money already paid</h2>
    {#if payments.length === 0}<p>No payment yet.</p>{:else}<ul>{#each payments as payment}<li><Money amountKobo={payment.amount_kobo} /> · {productLabel(payment.source_type)} · {productLabel(payment.state)}</li>{/each}</ul>{/if}
    {#if view.request.state === 'BUYER_REVIEWING'}<div class="actions"><button onclick={authorizeAndAccept}>Yes, I agree to pay <Money amountKobo={view.request.principal_kobo} /></button><button class="secondary" onclick={decline}>No, I do not agree</button></div>{/if}
    {#if view.request.state === 'RECEIPT_CONFIRMATION_PENDING'}<section class="claim"><h2>Did the goods arrive?</h2><button disabled={receiptBusy} onclick={()=>recordReceipt('confirmed')}>Yes, I got the goods</button><label>What is wrong?<textarea bind:value={receiptIssue} rows="3"></textarea></label><button disabled={receiptBusy||!receiptIssue.trim()} onclick={()=>recordReceipt('issue_raised')}>Report a receipt problem</button></section>{/if}
    {#if view.obligation}<p><a href={`/api/v1/buyer/credit-requests/${requestID}/agreement-document`} target="_blank" rel="noreferrer">Print or save this sale →</a></p><section class="claim"><h2>Pay now</h2><p>Make a safe link for the exact money left to pay.</p><button onclick={createPaymentLink}>Make payment link</button>{#if paymentURL}<a href={paymentURL}>Open payment page →</a>{/if}<h3>Already made a transfer?</h3><label>Money you paid (₦)<input bind:value={claimAmount} inputmode="decimal" /></label><label>Transfer number<input bind:value={claimReference} /></label><button onclick={claimPayment}>Tell the seller I paid</button></section><section class="claim"><h2>Report a problem</h2><p>Tell us how much money is affected and what went wrong.</p><form onsubmit={openDispute}><label>Money affected (₦)<input bind:value={disputeAmount} inputmode="decimal" required /></label><label>Short reason<input bind:value={disputeReason} required /></label><label>What happened?<textarea bind:value={disputeExplanation} rows="4"></textarea></label><label>What should happen to bank debit?<select bind:value={disputeEffect}><option value="CONTESTED_ONLY">Pause only this money</option><option value="FULL_BLOCK">Pause all bank debit</option><option value="NO_AUTOMATIC_BLOCK">Save the problem but keep payment running</option></select></label><button>Report this problem</button></form></section>{/if}
    {#if message}<p role="status">{message}</p>{/if}
  {:else}<p>Loading agreement…</p>{/if}
</main>
<style>.terms{max-width:48rem;padding:1.25rem;border:1px solid var(--color-border);border-radius:1rem}.terms dl{display:grid;grid-template-columns:minmax(10rem,1fr) 2fr;gap:.7rem 1rem}.terms dt{font-weight:700}.terms dd{margin:0}.actions{display:flex;flex-wrap:wrap;gap:.75rem;margin:1rem 0}.secondary{background:var(--color-surface);color:var(--color-destructive);border:1px solid var(--color-destructive)}.claim{display:grid;gap:.75rem;max-width:32rem;margin-top:2rem;padding:1rem;border:1px solid var(--color-border);border-radius:1rem}.claim form,.claim label{display:grid;gap:.5rem}.claim form{gap:.75rem}.claim input,.claim textarea,.claim select{box-sizing:border-box;width:100%;padding:.7rem;border:1px solid var(--color-border);border-radius:.5rem;background:var(--color-surface);color:inherit}@media(max-width:600px){.terms dl{grid-template-columns:1fr}.terms dt{margin-top:.5rem}}</style>
