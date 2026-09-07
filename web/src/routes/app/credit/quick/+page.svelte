<script lang="ts">
  import { getContext, onMount, tick } from 'svelte';
  import { goto } from '$app/navigation';
  import { ACCOUNT_CONTEXT, type AccountContext } from '$lib/account-context';
  import { checkedJSON, csrfHeader, LatestRequest, readResource, record, rows, text, type Resource } from '$lib/api/reliable';
  import { MutationIntent } from '$lib/api/mutation';
  import { customer, organization, dateLabel, timeLabel, type Customer, type Organization } from '$lib/records';
  import { deleteDraft, readDraft, saveDraft } from '$lib/sale-drafts';
  import { parseNaira, verbalizeNaira } from '$lib/money';
  import { collectionBoundary } from '$lib/financial-copy';
  import Money from '$lib/components/Money.svelte';
  import ResourceNotice from '$lib/components/ResourceNotice.svelte';
  const account = getContext<AccountContext>(ACCOUNT_CONTEXT);
  let organizations = $state<Resource<Organization[]>>({ state: 'loading', scope: '' });
  let customers = $state<Resource<Customer[]>>({ state: 'loading', scope: '' });
  let organizationID = $state(''), selectedBuyer = $state(''), goods = $state(''), principal = $state(''), dueDate = $state('');
  let step = $state(1), busy = $state(false), error = $state(''), recoveredDraft = $state(false), draftReady = $state(false), keepDraft = $state(false), draftWarning = $state('');
  let timing = $state<{ dueDate: string; collectionAt: string; organizationID: string } | null>(null);
  let heading = $state<HTMLHeadingElement>();
  let creation: MutationIntent | null = null;
  const reads = new LatestRequest(), businessReads = new LatestRequest();
  const amount = $derived(parseNaira(principal));
  const amountWords = $derived(verbalizeNaira(amount));
  const customerKey = (item: Customer) => `${item.buyer_user_id}:${item.buyer_business_id}`;
  const buyer = $derived(customers.state === 'ready' ? customers.data.find(item => customerKey(item) === selectedBuyer) : undefined);
  function restoreForBusiness() {
    draftReady = false; keepDraft = false; draftWarning = ''; recoveredDraft = false; goods = ''; principal = ''; dueDate = ''; timing = null; error = ''; creation = null;
    try {
      const saved = readDraft(account.userID, organizationID, sessionStorage);
      if (saved) { keepDraft = true; goods = saved.goods; principal = saved.principal; dueDate = saved.dueDate; recoveredDraft = true; }
    } catch { draftWarning = 'This browser cannot keep a draft. Keep this page open until you finish.'; }
    draftReady = true;
  }
  $effect(() => {
    if (!draftReady || !organizationID) return;
    try {
      if (!keepDraft || (!goods && !principal && !dueDate)) deleteDraft(account.userID, organizationID, sessionStorage);
      else if (!saveDraft(account.userID, organizationID, { goods, principal, dueDate }, sessionStorage)) draftWarning = 'Your draft could not be kept on this device. Keep this page open until you finish.';
    } catch { draftWarning = 'Your draft could not be kept on this device. Keep this page open until you finish.'; }
  });
  async function loadCustomers(resetDraft = false) {
    const scope = organizationID, request = reads.begin();
    customers = { state: 'loading', scope }; selectedBuyer = ''; step = 1;
    if (resetDraft) restoreForBusiness();
    if (!scope) return;
    const result = await readResource(scope, `/api/v1/organizations/${encodeURIComponent(scope)}/customers`, rows('customers', customer), request.signal, 'your customers');
    if (request.current() && organizationID === scope) customers = result;
  }
  async function focusStep() { await tick(); heading?.focus(); }
  async function next() {
    if (busy) return; error = '';
    if (step === 1 && !buyer) { error = 'Choose the customer for this sale.'; return; }
    if (step === 2 && (!goods.trim() || goods.trim().length > 5000 || amount <= 0)) { error = 'Enter the goods and a valid amount, with no more than two decimal places.'; return; }
    if (step === 3) {
      if (!/^\d{4}-\d{2}-\d{2}$/.test(dueDate)) { error = 'Choose the agreed payment date.'; return; }
      busy = true; timing = null;
      try {
        const scope = organizationID, date = dueDate;
        const result = await checkedJSON(`/api/v1/organizations/${encodeURIComponent(scope)}/credit-terms/preview`, value => {
          const result = record(value), collectionAt = text(result.collection_at);
          if (result.due_date !== date || result.grace_hours !== 24 || result.timezone !== 'Africa/Lagos' || result.timing_mode !== 'lagos_end_of_day' || !Number.isFinite(Date.parse(collectionAt))) throw new Error('The payment date could not be checked.');
          return { dueDate: date, collectionAt, organizationID: scope };
        }, { method: 'POST', headers: { 'Content-Type': 'application/json', ...csrfHeader() }, body: JSON.stringify({ due_date: date, grace_hours: 24 }) });
        if (organizationID !== scope || dueDate !== date) return;
        timing = result;
      } catch { error = 'We could not verify the payment date. Try again before saving.'; return; }
      finally { busy = false; }
    }
    step = Math.min(4, step + 1); await focusStep();
  }
  async function back() { if (busy) return; error = ''; step = Math.max(1, step - 1); await focusStep(); }
  async function submit() {
    if (busy) return; error = ''; busy = true;
    try {
      if (!buyer || !organizationID || amount <= 0 || !goods.trim() || timing?.dueDate !== dueDate || timing.organizationID !== organizationID) throw new Error('Go back and check the customer, amount and payment date.');
      creation ??= new MutationIntent(`${account.userID}:${organizationID}`, `/api/v1/organizations/${encodeURIComponent(organizationID)}/credit-requests`);
      const id = await creation.run({ buyer_user_id: buyer.buyer_user_id, buyer_business_id: buyer.buyer_business_id, buyer_legal_name: buyer.legal_name, buyer_trading_name: buyer.trading_name, principal_kobo: amount, goods_description: goods.trim(), invoice_reference: '', invoice_document_hash: '', due_date: dueDate, grace_hours: 24, collection_at: timing.collectionAt, timing_mode: 'lagos_end_of_day', schedule_type: 'one_time', schedule_count: 1, schedule_cadence: 'custom', month_end_policy: 'last_day', custom_schedule_items: [] }, value => text(record(record(value).request).id));
      draftReady = false;
      try { deleteDraft(account.userID, organizationID, sessionStorage); } catch { /* Server result is authoritative. */ }
      await goto(`/app/credit/${encodeURIComponent(id)}?organization=${encodeURIComponent(organizationID)}`);
    } catch (cause) { error = cause instanceof Error ? cause.message : 'We could not confirm that this sale was saved.'; }
    finally { busy = false; }
  }
  async function load() {
    const request = businessReads.begin();
    organizations = { state: 'loading', scope: account.userID };
    const result = await readResource(account.userID, '/api/v1/organizations', rows('organizations', organization), request.signal, 'your businesses');
    if (!request.current()) return;
    organizations = result;
    if (result.state !== 'ready') return;
    const params = new URLSearchParams(location.search);
    organizationID = result.data.find(item => item.id === params.get('organization'))?.id ?? result.data[0]?.id ?? '';
    if (!organizationID) return;
    restoreForBusiness();
    await loadCustomers();
    if (!request.current()) return;
    if (params.has('goods')) goods = (params.get('goods') ?? '').slice(0, 5000);
    if (params.has('amount')) principal = (params.get('amount') ?? '').slice(0, 40);
    if (customers.state === 'ready') {
      const matches = customers.data.filter(item => item.buyer_user_id === params.get('customer'));
      if (matches.length === 1) selectedBuyer = customerKey(matches[0]);
    }
  }
  onMount(() => { void load(); return () => { reads.cancel(); businessReads.cancel(); }; });
</script>
<svelte:head><title>Add a sale — Kredit</title></svelte:head>
<main class="shell quick-sale">
  <header class="task-heading"><div><p class="eyebrow">New credit sale</p><h1>Add a sale</h1><p class="lede">Choose the customer, goods, amount and payment date.</p></div><a href={`/app/credit/new?advanced=1&organization=${encodeURIComponent(organizationID)}`}>Instalments or an invoice? Use the full form</a></header>
  <ResourceNotice resource={organizations} label="Businesses" retry={load} />
  {#if organizations.state === 'ready' && !organizations.data.length}<div class="empty-state"><h2>Add your business first</h2><a class="primary" href="/app/overview">Add business details</a></div>
  {:else if organizationID}
    {#if recoveredDraft}<div class="inline-notice" role="status"><p>Your draft for this business was restored. Choose the customer again.</p><button type="button" disabled={busy} onclick={() => { keepDraft = false; goods = ''; principal = ''; dueDate = ''; timing = null; recoveredDraft = false; }}>Start fresh</button></div>{/if}
    <ol class="steps" aria-label="Sale steps">{#each ['Customer', 'Goods & amount', 'Payment date', 'Review'] as label, index}<li aria-current={step === index + 1 ? 'step' : undefined}><span aria-hidden="true">{index + 1}</span>{label}</li>{/each}</ol>
    {#if error}<p class="error" role="alert">{error}</p>{/if}
    <section class="sale-card" aria-busy={busy}>
      <h2 bind:this={heading} tabindex="-1">{['Who is buying?', 'What are you supplying?', 'When will they pay?', 'Check your sale'][step - 1]}</h2>
      {#if step === 1}
        {#if organizations.state === 'ready' && organizations.data.length > 1}<label>Your business<select bind:value={organizationID} disabled={busy} onchange={() => loadCustomers(true)}>{#each organizations.data as org}<option value={org.id}>{org.trading_name || org.legal_name}</option>{/each}</select></label>{/if}
        <ResourceNotice resource={customers} label="Customers" retry={() => void loadCustomers()} />
        {#if customers.state === 'ready'}{#if customers.data.length}<label>Customer<select bind:value={selectedBuyer} disabled={busy}><option value="">Choose a customer</option>{#each customers.data as item}<option value={customerKey(item)}>{item.trading_name || item.legal_name}</option>{/each}</select></label>{#if buyer}<div class="inline-notice"><strong>{buyer.trading_name || buyer.legal_name}</strong>{#if buyer.trading_name && buyer.trading_name !== buyer.legal_name}<p>Contracting party: {buyer.legal_name}</p>{/if}{#if buyer.overdue}<p>This customer has an overdue record. Review it before offering more credit.</p>{/if}</div>{/if}{:else}<div class="empty-state"><h3>No customers yet</h3><p>Invite a customer to confirm their own details.</p></div>{/if}<a href={`/app/customers/new?organization=${encodeURIComponent(organizationID)}`}>Add a customer</a>{/if}
      {:else if step === 2}
        <label>Goods and quantity<textarea bind:value={goods} rows="4" maxlength="5000" placeholder="For example: 40 cartons of 5L cooking oil" disabled={busy}></textarea></label>
        <label>Sale amount (₦)<input bind:value={principal} inputmode="decimal" maxlength="40" placeholder="120,000.00" disabled={busy} />{#if amountWords}<small>{amountWords}</small>{/if}</label>
      {:else if step === 3}
        <label>Agreed payment date<input type="date" bind:value={dueDate} disabled={busy} /></label>
        <div class="inline-notice"><strong>Payment timing</strong><p>The day ends at 11:59 pm Nigerian time. This sale allows 24 extra hours before a bank debit can be considered.</p><p>{collectionBoundary}</p></div>
      {:else}
        <dl class="sale-summary"><div><dt>Customer</dt><dd>{buyer?.trading_name || buyer?.legal_name}</dd></div>{#if buyer?.trading_name && buyer.trading_name !== buyer.legal_name}<div><dt>Contracting party</dt><dd>{buyer.legal_name}</dd></div>{/if}<div><dt>Goods</dt><dd>{goods}</dd></div><div><dt>Sale amount</dt><dd class="amount"><Money amountKobo={amount} /></dd></div><div><dt>Pay by</dt><dd>{dateLabel(dueDate)}</dd></div><div><dt>Bank debit may be considered from</dt><dd>{timing ? timeLabel(timing.collectionAt) : 'Not yet verified'}</dd></div></dl>
        <p class="field-help">This saves a draft. Review the complete terms and fees on the next screen before sending it. Your customer must accept the agreement and complete the required bank permission.</p>
      {/if}
    </section>
    <footer class="form-actions"><div>{#if step > 1}<button class="secondary" type="button" disabled={busy} onclick={back}>Back</button>{/if}</div>{#if step < 4}<button class="primary" type="button" onclick={next} disabled={busy || (step === 1 && !buyer)}>{busy ? 'Checking date…' : 'Continue'}</button>{:else}<button class="primary" type="button" onclick={submit} disabled={busy || !timing}>{busy ? 'Saving…' : 'Save draft sale'}</button>{/if}</footer>
    <label class="draft-choice"><input type="checkbox" bind:checked={keepDraft} />Keep this draft on this device for up to 12 hours</label><p class="field-help">Draft saving is off until you choose it. Avoid using it on a shared device. The selected customer is never saved in a browser draft.</p>{#if draftWarning}<p role="status">{draftWarning}</p>{/if}
  {/if}
</main>
<style>
  .quick-sale{max-width:48rem;padding-bottom:3rem}.task-heading{display:flex;align-items:start;justify-content:space-between;gap:2rem;margin-block:1rem 1.5rem}.task-heading h1{font-family:inherit;font-size:2rem;letter-spacing:-.03em;line-height:1.2;margin:.35rem 0}.task-heading .lede{font-size:1rem}.task-heading>a{max-width:14rem;color:var(--color-primary);font-size:.9rem;line-height:1.5}.steps{list-style:none;display:grid;grid-template-columns:repeat(4,1fr);padding:0;margin:1.5rem 0;border-bottom:1px solid var(--color-border)}.steps li{display:flex;align-items:center;gap:.4rem;padding:.7rem .3rem;color:var(--color-muted);font-size:.85rem;border-bottom:3px solid transparent}.steps li[aria-current]{border-color:var(--color-primary);color:var(--color-primary);font-weight:750}.steps span{display:grid;place-items:center;flex-shrink:0;width:1.6rem;height:1.6rem;border:1px solid currentColor;border-radius:50%}.sale-card{display:grid;gap:1.25rem;padding:clamp(1.25rem,4vw,2rem);border:1px solid var(--color-border);border-radius:.6rem;background:var(--color-surface);min-height:18rem;align-content:start}.sale-card h2{font-size:1.3rem;margin:0;line-height:1.3}.sale-card label{display:grid;gap:.5rem;font-weight:650}.sale-card input,.sale-card textarea,.sale-card select{box-sizing:border-box;width:100%;font:inherit;padding:.8rem;border:1px solid #9b9c96;border-radius:.35rem;background:#fff}.sale-card small,.field-help{font-size:.9rem;color:var(--color-muted);line-height:1.6;font-weight:400}.sale-card a{color:var(--color-primary)}.inline-notice{padding:1rem;border-left:3px solid var(--color-primary);background:#eef0ff;line-height:1.6}.inline-notice p{margin:.35rem 0}.inline-notice button{background:transparent;border:1px solid var(--color-primary);padding:.5rem .8rem;color:var(--color-primary)}.form-actions{display:flex;justify-content:space-between;gap:1rem;padding:1.25rem 0}.secondary{padding:.75rem 1rem;border:1px solid var(--color-border);border-radius:.35rem;background:var(--color-surface)}.sale-summary{display:grid;gap:1rem;margin:0}.sale-summary>div{display:grid;grid-template-columns:1fr 1.5fr;gap:1rem;border-bottom:1px solid var(--color-border);padding-bottom:1rem}.sale-summary dt{color:var(--color-muted)}.sale-summary dd{margin:0;overflow-wrap:anywhere;font-weight:650}.sale-summary .amount{font-size:1.5rem;font-variant-numeric:tabular-nums}.draft-choice{display:flex;gap:.6rem;align-items:center;font-size:.9rem}.draft-choice input{width:1.25rem;height:1.25rem;min-height:0;accent-color:var(--color-primary)}.error{padding:1rem;background:#fff1ed;border:1px solid #cc9c8c;color:#8c2f16;border-radius:.35rem;line-height:1.6}@media(max-width:560px){.task-heading{display:block}.task-heading>a{display:block;max-width:none}.steps li{flex-direction:column;text-align:center;gap:.5rem;font-size:.75rem}.sale-summary>div{grid-template-columns:1fr;gap:.4rem}.form-actions .primary{flex:1;max-width:17rem}}
</style>
