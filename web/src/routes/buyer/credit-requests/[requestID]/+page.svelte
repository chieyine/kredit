<script lang="ts">
  import { getContext, onMount } from 'svelte';
  import { page } from '$app/state';
  import { ACCOUNT_CONTEXT, type AccountContext } from '$lib/account-context';
  import { LatestRequest, readResource, record, rows, text, type Decoder, type Resource } from '$lib/api/reliable';
  import { MutationError, MutationIntent } from '$lib/api/mutation';
  import { saleView, paymentRow, dateLabel, timeLabel, type PaymentRow, type SaleView } from '$lib/records';
  import { feeDisclosure, validFeeTerms } from '$lib/fee-terms';
  import { acceptanceMessage, collectionBoundary, disputeEffectCopy, hostedAuthorizationURL } from '$lib/financial-copy';
  import { exactKobo, parseNaira } from '$lib/money';
  import { productLabel } from '$lib/product-language';
  import Money from '$lib/components/Money.svelte';
  import ResourceNotice from '$lib/components/ResourceNotice.svelte';
  const account = getContext<AccountContext>(ACCOUNT_CONTEXT);
  const requestID = $derived(page.params.requestID ?? '');
  const root = $derived(`/api/v1/buyer/credit-requests/${encodeURIComponent(requestID)}`);
  let resource = $state<Resource<SaleView>>({ state: 'loading', scope: '' });
  let paymentResource = $state<Resource<PaymentRow[]>>({ state: 'loading', scope: '' });
  let busy = $state(''), message = $state(''), actionError = $state(''), uncertain = $state('');
  let receiptIssue = $state(''), claimAmount = $state(''), claimReference = $state(''), paymentURL = $state('');
  let disputeAmount = $state(''), disputeReason = $state(''), disputeExplanation = $state(''), disputeEffect = $state('CONTESTED_ONLY');
  const intents = new Map<string, MutationIntent>();
  const reads = new LatestRequest();
  const view = $derived(resource.state === 'ready' ? resource.data : null);
  const bankURL = $derived(view?.mandate ? hostedAuthorizationURL(view.mandate.authorization_url, view.mandate.provider) : null);
  const accepted = $derived(view && ['BUYER_ACCEPTED', 'READY_TO_RELEASE', 'RECEIPT_CONFIRMATION_PENDING', 'ACTIVE', 'COMPLETED', 'PAID'].includes(view.request.state));
  const mayAccept = $derived(view?.request.state === 'BUYER_REVIEWING' && view.agreement?.id && /^[0-9a-f]{64}$/i.test(view.agreement.document_hash) && validFeeTerms(view.request.fee_terms) && !!view.request.supplier_legal_name.trim() && !!view.request.buyer_legal_name.trim() && !!view.request.goods_description.trim() && dateLabel(view.request.due_date) !== 'Date unavailable' && timeLabel(view.request.collection_at) !== 'Time unavailable');
  const mayPay = $derived(view?.obligation && (exactKobo(view.obligation.outstanding_kobo) ?? 0n) > 0n);
  function intent(action: string) {
    const key = `${requestID}:${action}`;
    let result = intents.get(key);
    if (!result) { result = new MutationIntent(account.userID, `${root}/${action}`); intents.set(key, result); }
    return result;
  }
  const blocked = (action: string) => Boolean(busy || (uncertain && uncertain !== action));
  async function load() {
    const id = requestID, request = reads.begin();
    resource = { state: 'loading', scope: id }; paymentResource = { state: 'loading', scope: id };
    const [sale, payments] = await Promise.all([
      readResource(id, `/api/v1/buyer/credit-requests/${encodeURIComponent(id)}`, saleView, request.signal, 'this sale'),
      readResource(id, `/api/v1/buyer/credit-requests/${encodeURIComponent(id)}/payments`, rows('payments', paymentRow), request.signal, 'payments on this sale')
    ]);
    if (!request.current() || requestID !== id) return;
    resource = sale; paymentResource = payments;
    if (sale.state === 'ready') {
      const state = sale.data.request.state;
      if ((uncertain === 'accept' && ['BUYER_ACCEPTED', 'READY_TO_RELEASE', 'ACTIVE', 'RECEIPT_CONFIRMATION_PENDING'].includes(state)) || (uncertain === 'decline' && state === 'DECLINED') || (uncertain === 'mandate' && sale.data.mandate)) { uncertain = ''; actionError = ''; }
      if (sale.data.mandate?.status === 'ACTIVE') { try { sessionStorage.removeItem(`kredit.bank-return.${account.userID}`); } catch { /* No stored return context is required for readiness. */ } }
    }
  }
  async function perform<T>(action: string, payload: unknown, decode: Decoder<T>, success: (result: T) => void) {
    if (blocked(action)) return;
    busy = action; actionError = ''; message = '';
    try {
      const result = await intent(action).run(payload, decode);
      uncertain = ''; success(result); await load();
    } catch (cause) {
      actionError = cause instanceof Error ? cause.message : 'We could not confirm this request.';
      if (cause instanceof MutationError && cause.outcome === 'unknown') uncertain = action;
    } finally { busy = ''; }
  }
  async function acceptSale() {
    if (!view || !mayAccept) return;
    await perform('accept', { agreement_version_id: view.agreement!.id, agreement_hash: view.agreement!.document_hash, mandate_provider_id: view.mandate?.provider_id ?? '' }, saleView, result => { message = acceptanceMessage(result.request.state); });
  }
  async function authorizeBank() {
    if (!accepted) return;
    await perform('mandate', undefined, saleView, result => {
      message = result.request.state === 'READY_TO_RELEASE' && result.mandate?.status === 'ACTIVE' ? 'Your bank permission is ready. The seller can arrange the goods.' : 'Bank permission is not ready yet. Complete the secure provider step below, then check the status.';
    });
  }
  function rememberReturn() {
    try { sessionStorage.setItem(`kredit.bank-return.${account.userID}`, JSON.stringify({ requestID, expiresAt: Date.now() + 3600000 })); } catch { /* The sale is still accessible from the buyer account. */ }
  }
  async function recordReceipt(state: 'confirmed' | 'issue_raised') {
    if (state === 'issue_raised' && !receiptIssue.trim()) { actionError = 'Describe the delivery problem first.'; return; }
    await perform('receipt', { state, ...(state === 'issue_raised' ? { issue_reason: receiptIssue.trim() } : {}) }, value => saleView(record(value).view), () => { message = state === 'confirmed' ? 'Receipt of the goods is recorded.' : 'Your delivery problem is recorded. The seller can review it.'; });
  }
  async function claimPayment() {
    const amount = parseNaira(claimAmount);
    if (amount <= 0 || !claimReference.trim()) { actionError = 'Enter the amount and transfer reference. Contact support if you cannot find the reference.'; return; }
    await perform('payment-claims', { amount_kobo: amount, paid_at: intent('payment-claims').createdAt, transfer_reference: claimReference.trim() }, value => record(record(value).payment_claim), () => { message = 'Transfer reported. The seller must confirm receipt before your balance changes.'; claimAmount = ''; claimReference = ''; });
  }
  async function createPaymentLink() {
    await perform('payment-link', undefined, value => {
      const result = record(value), raw = text(result.payment_url), url = new URL(raw, location.origin);
      if (url.username || url.password || !url.pathname.startsWith('/pay/') || (url.origin !== location.origin && url.origin !== 'https://kredit.com.ng')) throw new Error('The payment link could not be verified.');
      return url.href;
    }, url => { paymentURL = url; message = 'Payment page ready. Check the seller and amount before paying.'; });
  }
  async function openDispute(event: SubmitEvent) {
    event.preventDefault(); const amount = parseNaira(disputeAmount);
    if (amount <= 0 || !disputeReason.trim()) { actionError = 'Enter the amount in question and the reason for your report.'; return; }
    await perform('disputes', { disputed_amount_kobo: amount, reason: disputeReason.trim(), explanation: disputeExplanation.trim(), collection_effect: disputeEffect }, record, () => { message = `Problem reported. ${disputeEffectCopy(disputeEffect, amount)}`; });
  }
  onMount(() => { void load(); return () => reads.cancel(); });
</script>
<svelte:head><title>Review your sale — Kredit</title></svelte:head>
<main class="shell buyer-sale">
  <header class="task-heading"><div><p class="eyebrow">Your credit sale</p><h1>{accepted ? 'Your sale record' : 'Review this sale'}</h1><p>Check the goods, amount and payment date before agreeing.</p></div><a href="/legal/complaints">Get help</a></header>
  <ResourceNotice resource={resource} label="Sale" retry={load} />
  {#if actionError}<div class="action-error" role="alert"><p>{actionError}</p><button type="button" onclick={load} disabled={!!busy}>Check current record</button><a href="/legal/complaints">Contact support</a></div>{/if}
  {#if message}<p class="inline-notice" role="status">{message}</p>{/if}
  {#if view}
    <p class="state-label">{productLabel(view.request.state)}</p>
    <section class="terms" aria-labelledby="terms-heading"><h2 id="terms-heading">{accepted ? 'Agreed sale details' : 'What you are agreeing to'}</h2>
      <dl class="sale-summary"><div><dt>Seller</dt><dd>{view.request.supplier_legal_name || 'Seller name unavailable'}</dd></div><div><dt>Customer</dt><dd>{view.request.buyer_legal_name}</dd></div><div><dt>Goods</dt><dd>{view.request.goods_description}</dd></div><div><dt>Sale amount</dt><dd class="amount"><Money amountKobo={view.request.principal_kobo} /></dd></div><div><dt>First payment date</dt><dd>{dateLabel(view.request.due_date)}</dd></div><div><dt>Bank debit may be considered from</dt><dd>{timeLabel(view.request.collection_at)}</dd></div><div><dt>Extra time</dt><dd>{view.request.grace_hours} hours</dd></div><div><dt>Payment arrangement</dt><dd>{view.request.schedule_type === 'equal' ? `${view.request.schedule_count} ${view.request.schedule_cadence} payments` : view.request.schedule_type === 'custom' ? 'Amounts and dates below' : 'One payment'}</dd></div></dl>
      {#if view.request.custom_schedule_items.length}<h3>Payment schedule</h3><ol>{#each view.request.custom_schedule_items as item}<li><Money amountKobo={item.amount_kobo} /> by {dateLabel(item.due_date)}</li>{/each}</ol>{/if}
      <p class="fee-note">{feeDisclosure(view.request.fee_terms)}</p><p class="field-help">{collectionBoundary}</p>
      {#if view.agreement?.document_hash}<details><summary>Sale reference and record details</summary><p>Sale: <code>{view.request.id}</code></p><p>Agreement fingerprint:</p><code class="fingerprint">{view.agreement.document_hash}</code><p>This identifies the recorded version of the agreement.</p></details>{/if}
    </section>
    {#if view.request.state === 'BUYER_REVIEWING'}
      <section class="consent-panel"><h2>Your decision</h2><p>Accepting records your agreement to this sale. Bank-debit permission is a separate step. Ask the seller to correct anything that is wrong before you accept.</p>
        {#if !mayAccept}<p role="alert">The complete agreement or fees could not be verified. Refresh before accepting.</p>{/if}
        <div class="actions"><button class="primary" disabled={!mayAccept || blocked('accept')} onclick={acceptSale}>{busy === 'accept' ? 'Recording your decision…' : 'Accept sale for'} {#if busy !== 'accept'}<Money amountKobo={view.request.principal_kobo} />{/if}</button><button class="secondary" disabled={blocked('decline')} onclick={() => perform('decline', undefined, saleView, () => { message = 'You declined this sale.'; })}>Decline sale</button></div>
      </section>
    {/if}
    {#if accepted && !['PAID', 'COMPLETED'].includes(view.request.state)}
      <section class="bank-panel" aria-labelledby="bank-heading"><h2 id="bank-heading">Bank-debit permission</h2>
        {#if view.mandate?.status === 'ACTIVE'}<p class="permission-state">Permission active</p>{#if view.request.state === 'READY_TO_RELEASE'}<p>The seller can now arrange the goods.</p>{:else}<p>Permission is active. Any debit still depends on the agreed date, amount and payment checks.</p>{/if}
        {:else}<p>The sale is accepted, but bank permission is not ready. The seller must not release goods until Kredit confirms readiness.</p><p>{view.mandate ? `Permission status: ${productLabel(view.mandate.status)}` : 'No bank permission is attached yet.'}</p>
          {#if bankURL}<p>You will continue on Mono’s secure authorisation page. Read the accounts, limit and duration shown there before giving permission.</p><a class="primary" href={bankURL} rel="noreferrer noopener" onclick={rememberReturn}>Continue securely with Mono <span aria-hidden="true">↗</span></a><p class="field-help">Returning from the provider is not confirmation. Use “Check permission status” after completing that step.</p>
          {:else if !view.mandate || ['CANCELLED', 'EXPIRED', 'FAILED'].includes(view.mandate.status)}<button class="primary" disabled={blocked('mandate')} onclick={authorizeBank}>{busy === 'mandate' ? 'Preparing secure permission…' : 'Set up bank permission'}</button>
          {:else}<p>The provider link is not available. Check the status or contact support; do not create another permission request blindly.</p>{/if}
          <button class="secondary" type="button" disabled={!!busy} onclick={load}>Check permission status</button>
        {/if}<p class="field-help">Do not share your PIN, bank password or sign-in code with the seller or Kredit support.</p>
      </section>
    {/if}
    {#if view.request.state === 'RECEIPT_CONFIRMATION_PENDING'}<section class="delivery-panel"><h2>Did you receive all the goods?</h2><p>Check the quantity and condition against this sale.</p><button class="primary" disabled={blocked('receipt')} onclick={() => recordReceipt('confirmed')}>Yes, I received all the goods</button><label>Report a delivery problem<textarea bind:value={receiptIssue} rows="3" maxlength="2000" placeholder="For example: 5 cartons were missing" disabled={!!busy}></textarea></label><button class="secondary" disabled={blocked('receipt') || !receiptIssue.trim()} onclick={() => recordReceipt('issue_raised')}>Report delivery problem</button></section>{/if}
    {#if view.obligation}
      <section class="payments-panel"><div class="section-heading"><h2>Payments on this sale</h2><a href={`/buyer/obligations/${encodeURIComponent(view.obligation.id)}`}>View schedule</a></div><p class="balance-label">Left to pay</p><p class="outstanding"><Money amountKobo={view.obligation.outstanding_kobo} /></p>
        <ResourceNotice resource={paymentResource} label="Payments" retry={load} />
        {#if paymentResource.state === 'ready'}{#if !paymentResource.data.length}<p>No confirmed payments are recorded on this sale.</p>{:else}<ul class="payment-list">{#each paymentResource.data as payment (payment.id)}<li><strong><Money amountKobo={payment.amount_kobo} /></strong><span>{productLabel(payment.state)} · {productLabel(payment.source_type)}</span></li>{/each}</ul>{/if}{/if}
        <a href={`/api/v1/buyer/credit-requests/${encodeURIComponent(requestID)}/agreement-document`} target="_blank" rel="noreferrer">Print or save the agreement</a>
      </section>
      {#if mayPay}<section class="payment-options"><h2>Make or report a payment</h2><p>Use a payment page, or tell the seller about a transfer you already made.</p><button class="primary" disabled={blocked('payment-link')} onclick={createPaymentLink}>Get payment page</button>{#if paymentURL}<a href={paymentURL}>Open payment page</a>{/if}
        <details><summary>Already paid by bank transfer?</summary><p>A transfer report is not a payment confirmation. Your balance changes only after the money is verified.</p><label>Amount transferred (₦)<input bind:value={claimAmount} inputmode="decimal" disabled={!!busy} /></label><label>Transfer reference<input bind:value={claimReference} maxlength="256" disabled={!!busy} /></label><p class="field-help">Find this on your bank receipt. Contact support if the reference is unavailable, or a different person sent the payment and you need help matching it.</p><button class="secondary" disabled={blocked('payment-claims')} onclick={claimPayment}>Report my transfer</button></details>
      </section>{/if}
      <details class="dispute-panel"><summary>Report a problem with this sale</summary><p>Your report and its outcome stay with the sale.</p><form onsubmit={openDispute}><label>Amount in question (₦)<input bind:value={disputeAmount} inputmode="decimal" required disabled={!!busy} /></label><label>What went wrong?<input bind:value={disputeReason} maxlength="200" placeholder="For example: 5 cartons were missing" required disabled={!!busy} /></label><label>Details<textarea bind:value={disputeExplanation} maxlength="5000" rows="4" disabled={!!busy}></textarea></label><label>What should happen to new bank debits?<select bind:value={disputeEffect} disabled={!!busy}><option value="CONTESTED_ONLY">Hold the amount in question</option><option value="FULL_BLOCK">Hold all bank debits on this sale</option><option value="NO_AUTOMATIC_BLOCK">Keep eligible payments running</option></select></label><p class="inline-notice">{disputeEffectCopy(disputeEffect, parseNaira(disputeAmount) > 0 ? parseNaira(disputeAmount) : undefined)}</p><button class="primary" disabled={blocked('disputes')}>Report problem</button></form></details>
    {/if}
  {/if}
</main>
<style>
  .buyer-sale{max-width:48rem;padding-bottom:3rem}.task-heading,.section-heading{display:flex;align-items:baseline;justify-content:space-between;gap:1rem;margin:1rem 0 1.5rem}.task-heading h1{font-family:inherit;font-size:2rem;line-height:1.2;letter-spacing:-.03em;margin:.35rem 0}.task-heading p{color:var(--color-muted);line-height:1.6}.task-heading>a{white-space:nowrap}.buyer-sale a{color:var(--color-primary)}.buyer-sale a.primary{color:#fff}.buyer-sale h2{font-size:1.25rem;margin:0 0 1rem;line-height:1.3}.buyer-sale h3{font-size:1.05rem}.buyer-sale p,.buyer-sale li{line-height:1.6}.terms,.consent-panel,.bank-panel,.delivery-panel,.payments-panel,.payment-options,.dispute-panel{border:1px solid var(--color-border);background:var(--color-surface);border-radius:.6rem;padding:clamp(1rem,4vw,1.5rem);margin:1.25rem 0}.consent-panel,.bank-panel{border-top:3px solid var(--color-primary)}.state-label{display:inline-block;background:#eceefc;color:#25339c;padding:.35rem .75rem;border-radius:.3rem;font-size:.9rem;font-weight:650;margin:0}.sale-summary{display:grid;gap:.8rem;margin:0}.sale-summary>div{display:grid;grid-template-columns:1fr 1.4fr;gap:1rem;padding-bottom:.8rem;border-bottom:1px solid var(--color-border)}.sale-summary dt{color:var(--color-muted);line-height:1.5}.sale-summary dd{margin:0;overflow-wrap:anywhere;line-height:1.5;font-weight:600}.sale-summary .amount{font-size:1.7rem;font-variant-numeric:tabular-nums}.fee-note,.field-help{font-size:.9rem;color:var(--color-muted);line-height:1.6}.actions{display:flex;flex-wrap:wrap;gap:.75rem}.primary,.secondary{min-height:3rem;padding:.75rem 1rem;border-radius:.35rem;font:inherit;line-height:1.5;cursor:pointer}.secondary{border:1px solid var(--color-border);background:transparent;color:var(--color-foreground)}.bank-panel>.secondary{display:block;margin-top:1rem}.permission-state{color:#135d3f;font-weight:750}.buyer-sale label{display:grid;gap:.5rem;margin:1rem 0;font-weight:650}.buyer-sale input,.buyer-sale textarea,.buyer-sale select{box-sizing:border-box;width:100%;padding:.75rem;border:1px solid #9b9c96;border-radius:.35rem;background:#fff;font:inherit}.buyer-sale summary{display:flex;align-items:center;min-height:2.75rem;cursor:pointer;font-weight:700;line-height:1.5}.buyer-sale summary::after{content:'+';margin-left:auto;padding-left:1rem}.buyer-sale details[open]>summary::after{content:'−'}.fingerprint{display:block;overflow-wrap:anywhere;white-space:normal}.inline-notice{background:#eef0ff;border-left:3px solid var(--color-primary);padding:1rem;line-height:1.6}.action-error{padding:1rem;border:1px solid #cba783;border-radius:.4rem;background:#fff3e0;color:#603f16}.action-error button{border:1px solid currentColor;background:transparent;padding:.6rem .8rem;color:inherit}.action-error a{display:inline-block;margin:.5rem}.balance-label{color:var(--color-muted);margin-bottom:.25rem}.outstanding{font-size:2rem;font-weight:750;margin:.25rem 0 1rem;font-variant-numeric:tabular-nums}.payment-list{list-style:none;padding:0}.payment-list li{display:flex;justify-content:space-between;flex-wrap:wrap;gap:.5rem;border-bottom:1px solid var(--color-border);padding:.8rem 0}.payment-list span{font-size:.9rem;color:var(--color-muted)}@media(max-width:560px){.sale-summary>div{grid-template-columns:1fr;gap:.3rem}.task-heading{flex-wrap:wrap}.actions>*{width:100%}.section-heading{flex-wrap:wrap}}
</style>
