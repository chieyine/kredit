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
      message=response.ok?(state==='confirmed'?'Thank you. We have put it on record that the goods reached you.':'Your problem has been reported. The seller will see it.'):result.detail??'We could not save your answer. Please try again.';
      if(response.ok)await load();
    } catch {message='We could not reach Kredit. Check your network and try again.'} finally {receiptBusy=false}
  }
  let claimAmount = $state('');
  let claimReference = $state('');
  let paymentURL = $state('');
  let disputeAmount = $state(''), disputeReason = $state(''), disputeExplanation = $state(''), disputeEffect = $state('CONTESTED_ONLY');
  const requestID = $derived(page.params.requestID);
  async function load() { const response = await fetch(`/api/v1/buyer/credit-requests/${requestID}`); if (response.ok) view = await response.json(); const paymentResponse = await fetch(`/api/v1/buyer/credit-requests/${requestID}/payments`); if (paymentResponse.ok) payments = (await paymentResponse.json()).payments; }
  async function authorizeAndAccept() {
    message = 'Setting up your bank debit permission…';
    if (!mandateKey) mandateKey = idempotencyKey();
    const mandateResponse = await fetch(`/api/v1/buyer/credit-requests/${requestID}/mandate`, { method: 'POST', headers: { 'Idempotency-Key': mandateKey, ...csrfHeaders() } });
    if (!mandateResponse.ok) { message = 'We could not set up bank debit. Nothing was agreed. Please try again.'; return; }
    const mandated = await mandateResponse.json();
    if (!acceptKey) acceptKey = idempotencyKey();
    const response = await fetch(`/api/v1/buyer/credit-requests/${requestID}/accept`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'Idempotency-Key': acceptKey, ...csrfHeaders() }, body: JSON.stringify({ agreement_version_id: mandated.agreement.id, agreement_hash: mandated.agreement.document_hash, mandate_provider_id: mandated.mandate.provider_id }) });
    if (response.ok) { mandateKey = ''; acceptKey = ''; }
    message = response.ok ? 'Done. You have agreed, and the seller can send your goods now.' : 'We could not save your answer. Please try again.'; await load();
  }
  async function decline() {
    message = 'Declining request…';
    const response = await fetch(`/api/v1/buyer/credit-requests/${requestID}/decline`, { method: 'POST', headers: { 'Idempotency-Key': idempotencyKey(), ...csrfHeaders() } });
    const result = await response.json().catch(() => ({}));
    message = response.ok ? 'You said no. This sale will not go ahead.' : (result.detail ?? 'We could not save your answer. Please try again.');
    await load();
  }
  async function claimPayment() {
    const amountKobo = parseNaira(claimAmount);
    if(amountKobo<=0 || !claimReference.trim()){message='Please enter how much you paid and the transfer reference number.';return;}
    const response=await fetch(`/api/v1/buyer/credit-requests/${requestID}/payment-claims`,{method:'POST',credentials:'include',headers:{'Content-Type':'application/json','Idempotency-Key':idempotencyKey(),...csrfHeaders()},body:JSON.stringify({amount_kobo:amountKobo,paid_at:new Date().toISOString(),transfer_reference:claimReference})});
    const result=await response.json().catch(()=>({})); message=response.ok?'We have told the seller. They will check their bank and confirm it.':(result.detail??'We could not send this. Please try again.'); if(response.ok){claimAmount='';claimReference='';}
  }
  async function createPaymentLink(){const response=await fetch(`/api/v1/buyer/credit-requests/${requestID}/payment-link`,{method:'POST',credentials:'include',headers:{'Idempotency-Key':idempotencyKey(),...csrfHeaders()}});const result=await response.json().catch(()=>({}));if(response.ok)paymentURL=result.payment_url;else message=result.detail??'We could not create the payment link. Please try again.';}
  async function openDispute(event:SubmitEvent){event.preventDefault();const disputed_amount_kobo=parseNaira(disputeAmount);if(disputed_amount_kobo<=0||!disputeReason.trim()){message='Please enter how much money is affected and what went wrong.';return}const response=await fetch(`/api/v1/buyer/credit-requests/${requestID}/disputes`,{method:'POST',credentials:'include',headers:{'Content-Type':'application/json','Idempotency-Key':idempotencyKey(),...csrfHeaders()},body:JSON.stringify({disputed_amount_kobo,reason:disputeReason,explanation:disputeExplanation,collection_effect:disputeEffect})});const result=await response.json().catch(()=>({}));if(!response.ok){message=result.detail??'We could not report this problem. Please try again.';return}message='Reported. From now on, bank debit follows the choice you made above. But if a debit was already sent to your bank, that one may still go through.';disputeAmount='';disputeReason='';disputeExplanation='';await load()}
  onMount(load);
</script>

<svelte:head><title>Check this sale before you agree — Kredit</title></svelte:head>
<main>
  <h1>Read this sale before you say yes</h1>
  <p class="lede">Once you agree, nobody can change any of this. So if something here is wrong, do not accept it. Call the seller first and sort it out.</p>
  {#if view}
    <section class="terms"><h2>What you are agreeing to</h2>{#if view.obligation}<p>This is what you agreed to at the start. <a href={`/buyer/obligations/${view.obligation.id}`}>See where your payment days stand today</a>.</p>{/if}<dl><dt>Seller</dt><dd>{view.request.supplier_legal_name}</dd><dt>Customer</dt><dd>{view.request.buyer_legal_name}</dd><dt>Money to pay</dt><dd><Money amountKobo={view.request.principal_kobo} /></dd><dt>Goods</dt><dd>{view.request.goods_description}</dd><dt>{view.obligation?'First payment day you agreed':'First payment day'}</dt><dd>{view.request.due_date}</dd><dt>{view.obligation?'Bank debit day you agreed':'Kredit may debit your bank after'}</dt><dd>{new Date(view.request.collection_at).toLocaleString('en-NG',{timeZone:'Africa/Lagos',timeZoneName:'short'})}</dd><dt>Extra time before that</dt><dd>{view.request.grace_hours} hours</dd><dt>How you will pay</dt><dd>{view.request.schedule_type==='equal'?`${view.request.schedule_count} ${view.request.schedule_cadence} payments`:view.request.schedule_type==='custom'?'Different amounts on different days':'One payment, all at once'}</dd><dt>Kredit fee (paid by the seller)</dt><dd>{feeDisclosure(view.request.fee_terms)}</dd><dt>Sale number</dt><dd>{view.request.id}</dd></dl>{#if view.request.custom_schedule_items?.length}<h3>Payment days</h3><ol>{#each view.request.custom_schedule_items as item}<li><Money amountKobo={item.amount_kobo} /> by {item.due_date}</li>{/each}</ol>{/if}<details><summary>Technical record (for reference)</summary><p>This code proves these exact terms were never changed after you agreed.<br/><code>{view.agreement.document_hash}</code></p></details></section>
    {#if view.obligation || payments.length}<h2>What you have paid so far</h2>
    {#if payments.length === 0}<p>Nothing paid on this sale yet.</p>{:else}<ul>{#each payments as payment}<li><Money amountKobo={payment.amount_kobo} /> · {productLabel(payment.source_type)} · {productLabel(payment.state)}</li>{/each}</ul>{/if}{/if}
    {#if view.request.state === 'BUYER_REVIEWING'}<div class="actions"><button onclick={authorizeAndAccept}>Yes, I agree to pay <Money amountKobo={view.request.principal_kobo} /></button><button class="secondary" onclick={decline}>No, I do not agree to this</button></div>{/if}
    {#if view.request.state === 'RECEIPT_CONFIRMATION_PENDING'}<section class="claim"><h2>Did the goods reach you?</h2><p>Answer honestly. If something is missing, broken or wrong, say it now. Proving it next month is much harder.</p><button disabled={receiptBusy} onclick={()=>recordReceipt('confirmed')}>Yes, I got the goods</button><label>Or tell us what is wrong<textarea bind:value={receiptIssue} rows="3" placeholder="For example: 5 cartons were missing"></textarea></label><button disabled={receiptBusy||!receiptIssue.trim()} onclick={()=>recordReceipt('issue_raised')}>Report this problem</button></section>{/if}
    {#if view.obligation}<p><a href={`/api/v1/buyer/credit-requests/${requestID}/agreement-document`} target="_blank" rel="noreferrer">Print or save a copy of this sale →</a></p><section class="claim"><h2>Pay this sale</h2><p>We will make you a safe link for exactly what you still owe.</p><button onclick={createPaymentLink}>Make me a payment link</button>{#if paymentURL}<a href={paymentURL}>Open the payment page →</a>{/if}<h3>Already sent the money by transfer?</h3><p>Tell the seller. They will look in their bank, find it and confirm it. Your balance drops the moment they do.</p><label>How much did you send? (₦)<input bind:value={claimAmount} inputmode="decimal" /></label><label>The transfer reference number<input bind:value={claimReference} /></label><button onclick={claimPayment}>Tell the seller I have paid</button></section><details class="claim problem"><summary><strong>Report a problem</strong> <span class="hint">goods or amount wrong</span></summary><p>Tell us how much money it affects and what went wrong. We keep it on record, so nobody can pretend it never happened.</p><form onsubmit={openDispute}><label>How much money does this affect? (₦)<input bind:value={disputeAmount} inputmode="decimal" required /></label><label>In one line, what went wrong?<input bind:value={disputeReason} required placeholder="For example: 5 cartons were missing" /></label><label>Now tell us the whole story<textarea bind:value={disputeExplanation} rows="4"></textarea></label><label>While we sort this out, what should happen to bank debit?<select bind:value={disputeEffect}><option value="CONTESTED_ONLY">Hold only the money in question</option><option value="FULL_BLOCK">Hold all bank debit on this sale</option><option value="NO_AUTOMATIC_BLOCK">Record the problem, but let payment continue</option></select></label><button>Report this problem</button></form></details>{/if}
    {#if message}<p role="status">{message}</p>{/if}
  {:else}<p>Opening this sale…</p>{/if}
</main>
<style>details.problem>summary{cursor:pointer;list-style:none;display:flex;align-items:center;justify-content:space-between;gap:1rem;min-height:2.75rem;font-size:1rem}details.problem>summary::-webkit-details-marker{display:none}details.problem>summary::after{content:"+";font-weight:800;color:#2738d6}details.problem[open]>summary::after{content:"\2013"}details.problem>summary .hint{font-size:.78rem;font-weight:600;color:#5f645f}.terms{max-width:48rem;padding:1.25rem;border:1px solid var(--color-border);border-radius:1rem}.terms dl{display:grid;grid-template-columns:minmax(10rem,1fr) 2fr;gap:.7rem 1rem}.terms dt{font-weight:700}.terms dd{margin:0}.actions{display:flex;flex-wrap:wrap;gap:.75rem;margin:1rem 0}.secondary{background:var(--color-surface);color:var(--color-destructive);border:1px solid var(--color-destructive)}.claim{display:grid;gap:.75rem;max-width:32rem;margin-top:2rem;padding:1rem;border:1px solid var(--color-border);border-radius:1rem}.claim form,.claim label{display:grid;gap:.5rem}.claim form{gap:.75rem}.claim input,.claim textarea,.claim select{box-sizing:border-box;width:100%;padding:.7rem;border:1px solid var(--color-border);border-radius:.5rem;background:var(--color-surface);color:inherit}@media(max-width:600px){.terms dl{grid-template-columns:1fr}.terms dt{margin-top:.5rem}}</style>
