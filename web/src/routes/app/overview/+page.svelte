<script lang="ts">
  import { getContext, onMount } from 'svelte';
  import { ACCOUNT_CONTEXT, type AccountContext } from '$lib/account-context';
  import { checkedJSON, LatestRequest, readResource, record, rows, type Resource } from '$lib/api/reliable';
  import { MutationIntent } from '$lib/api/mutation';
  import { organization, saleView, paymentRow, workRow, receivables, type Organization, type SaleView, type PaymentRow, type WorkRow, type Receivables } from '$lib/records';
  import { attentionItems } from '$lib/attention';
  import { exactKobo } from '$lib/money';
  import { productLabel } from '$lib/product-language';
  import Money from '$lib/components/Money.svelte';
  import ResourceNotice from '$lib/components/ResourceNotice.svelte';
  import FeedbackPrompt from '$lib/components/FeedbackPrompt.svelte';
  const account = getContext<AccountContext>(ACCOUNT_CONTEXT);
  const pending = <T,>(scope = ''): Resource<T> => ({ state: 'loading', scope });
  let businesses = $state<Resource<Organization[]>>(pending());
  let organizationID = $state('');
  let sales = $state<Resource<SaleView[]>>(pending());
  let payments = $state<Resource<PaymentRow[]>>(pending());
  let overdue = $state<Resource<WorkRow[]>>(pending());
  let claims = $state<Resource<WorkRow[]>>(pending());
  let disputes = $state<Resource<WorkRow[]>>(pending());
  let summary = $state<Resource<Receivables>>(pending());
  let visibleCount = $state(5), legalName = $state(''), tradingName = $state(''), businessType = $state('unregistered_business'), address = $state(''), industry = $state(''), createBusy = $state(false), createError = $state('');
  let creation: MutationIntent | null = null;
  const businessRequest = new LatestRequest(), dashboardRequest = new LatestRequest();
  const organizations = $derived(businesses.state === 'ready' ? businesses.data : []);
  const currentBusiness = $derived(organizations.find(item => item.id === organizationID));
  const allChecked = $derived([sales, payments, overdue, claims, disputes, summary].every(item => item.state === 'ready' && item.scope === organizationID));
  const attention = $derived(attentionItems(organizationID, sales.state === 'ready' ? sales.data : [], claims.state === 'ready' ? claims.data : [], overdue.state === 'ready' ? overdue.data : [], disputes.state === 'ready' ? disputes.data : []));
  const scopeQuery = $derived(`?organization=${encodeURIComponent(organizationID)}`);
  async function loadRequests() {
    const scope = organizationID;
    const request = dashboardRequest.begin(); visibleCount = 5;
    sales = pending(scope); payments = pending(scope); overdue = pending(scope); claims = pending(scope); disputes = pending(scope); summary = pending(scope);
    if (!scope) return;
    const root = `/api/v1/organizations/${encodeURIComponent(scope)}`;
    const results = await Promise.all([
      readResource(scope, `${root}/credit-requests`, rows('requests', saleView), request.signal, 'your sales'),
      readResource(scope, `${root}/payments`, rows('payments', paymentRow), request.signal, 'your payments'),
      readResource(scope, `${root}/overdue`, rows('overdue', workRow), request.signal, 'overdue sales'),
      readResource(scope, `${root}/payment-claims`, rows('payment_claims', workRow), request.signal, 'reported transfers'),
      readResource(scope, `${root}/disputes`, rows('disputes', workRow), request.signal, 'reported problems'),
      readResource(scope, `${root}/reports/receivables`, receivables, request.signal, 'your balance')
    ]);
    if (!request.current() || organizationID !== scope) return;
    [sales, payments, overdue, claims, disputes, summary] = results;
  }
  async function load() {
    const request = businessRequest.begin(); businesses = pending(account.userID);
    const result = await readResource(account.userID, '/api/v1/organizations', rows('organizations', organization), request.signal, 'your businesses');
    if (!request.current()) return;
    businesses = result;
    if (result.state === 'ready') {
      const requested = new URLSearchParams(location.search).get('organization');
      organizationID = result.data.find(item => item.id === (requested || organizationID))?.id ?? result.data[0]?.id ?? '';
      if (organizationID) await loadRequests();
    }
  }
  async function createOrganization() {
    if (createBusy) return; createBusy = true; createError = '';
    try {
      creation ??= new MutationIntent(account.userID, '/api/v1/organizations');
      await creation.run({ legal_name: legalName.trim(), trading_name: tradingName.trim(), business_type: businessType, business_address: address.trim(), industry: industry.trim(), timezone: 'Africa/Lagos', currency: 'NGN' }, record);
      await load();
    } catch (cause) { createError = cause instanceof Error ? cause.message : 'We could not confirm your business details.'; }
    finally { createBusy = false; }
  }
  onMount(() => { void load(); return () => { businessRequest.cancel(); dashboardRequest.cancel(); }; });
</script>
<svelte:head><title>Business overview — Kredit</title></svelte:head>
<main class="shell workspace account-home">
  <header class="task-heading"><div><p class="eyebrow">Your business</p><h1>{currentBusiness?.trading_name || currentBusiness?.legal_name || 'Business overview'}</h1><p class="lede">Your sales, payments and next steps.</p></div>{#if organizationID}<a class="primary" href={`/app/credit/quick${scopeQuery}`}>Add a sale</a>{/if}</header>
  <ResourceNotice resource={businesses} label="Businesses" retry={load} />
  {#if businesses.state === 'ready' && !organizations.length}
    <section class="card onboarding"><h2>Add your business</h2><p>Start with the name your customers know. Verification is required before you can send a sale or use payment services.</p>
      <form class="form-grid" onsubmit={event => { event.preventDefault(); void createOrganization(); }}>
        <label>Your name, or registered business name<input bind:value={legalName} autocomplete="organization" required disabled={createBusy} /></label><label>Trading name <small>if different</small><input bind:value={tradingName} disabled={createBusy} /></label>
        <label>Business type<select bind:value={businessType} disabled={createBusy}><option value="unregistered_business">Not registered yet</option><option value="registered_business">Business name registered with CAC</option><option value="sole_proprietor">Sole proprietor</option><option value="limited_company">Limited company</option><option value="partnership">Partnership</option></select></label>
        <label>What do you sell?<input bind:value={industry} placeholder="Food, medicines, building materials…" required disabled={createBusy} /></label><label class="wide">Business address<textarea bind:value={address} placeholder="Shop number, street, area, town and state" required disabled={createBusy}></textarea></label>
        {#if createError}<p class="error wide" role="alert">{createError}</p>{/if}<button class="primary wide" disabled={createBusy}>{createBusy ? 'Saving…' : 'Add my business'}</button>
      </form>
    </section>
  {:else if currentBusiness}
    <div class="toolbar"><label>Business<select bind:value={organizationID} onchange={loadRequests}>{#each organizations as org}<option value={org.id}>{org.trading_name || org.legal_name}</option>{/each}</select></label><button type="button" onclick={loadRequests}>Refresh</button><a href="/app/onboarding">Business setup</a></div>
    <section class="balance-card" aria-label="What you are owed" aria-busy={summary.state === 'loading'}>
      <p>Outstanding balance</p>
      {#if summary.state === 'ready'}
        <strong class="balance"><Money amountKobo={summary.data.outstanding_kobo} /></strong><p class="balance-caption">Across {summary.data.obligation_count} active sale{summary.data.obligation_count === 1 ? '' : 's'}</p>
        {#if (exactKobo(summary.data.overdue_kobo) ?? 0n) > 0n}<a class="late-strip" href={`/app/overdue${scopeQuery}`}><span>Overdue</span><strong><Money amountKobo={summary.data.overdue_kobo} /></strong><span aria-hidden="true">→</span></a>
        {:else}<p class="checked-state">No overdue balance in this checked summary.</p>{/if}
      {:else}<ResourceNotice resource={summary} label="Balance" retry={loadRequests} />{/if}
    </section>
    <section class="attention-section" aria-labelledby="attention-heading">
      <header class="section-heading"><div><h2 id="attention-heading">Needs attention</h2><p>{allChecked ? `${attention.length} item${attention.length === 1 ? '' : 's'} to review` : 'Some records have not been checked yet.'}</p></div><a href={`/app/credit${scopeQuery}`}>View sales</a></header>
      {#each [{ resource: sales, label: 'Sales' }, { resource: payments, label: 'Payments' }, { resource: overdue, label: 'Overdue sales' }, { resource: claims, label: 'Reported transfers' }, { resource: disputes, label: 'Reported problems' }] as item}<ResourceNotice resource={item.resource} label={item.label} retry={loadRequests} />{/each}
      {#if attention.length}<div class="action-list">{#each attention.slice(0, visibleCount) as item (item.id)}<article><div><h3>{item.title}</h3><p>{item.detail}</p></div><a href={item.href}>{item.action} <span aria-hidden="true">→</span></a></article>{/each}</div>{#if attention.length > visibleCount}<button class="secondary show-more" type="button" onclick={() => visibleCount += 10}>Show more · {attention.length - visibleCount} remaining</button>{/if}
      {:else if allChecked}<div class="empty-state"><h3>Nothing needs an action right now.</h3><p>Your sales, reported payments and problems were checked successfully.</p></div>{/if}
    </section>
    {#if sales.state === 'ready'}<section class="recent-sales"><header class="section-heading"><h2>Recent sales</h2><a href={`/app/credit${scopeQuery}`}>All sales</a></header>{#if !sales.data.length}<div class="empty-state"><h3>No sales yet</h3><p>Add the goods, amount and payment date for your first customer.</p><a class="primary" href={`/app/credit/quick${scopeQuery}`}>Add my first sale</a></div>{:else}<div class="record-list">{#each sales.data.slice(0, 5) as view (view.request.id)}<a class="record-row" href={`/app/credit/${encodeURIComponent(view.request.id)}${scopeQuery}`}><span><strong>{view.request.buyer_legal_name}</strong><small>{productLabel(view.request.state)}</small></span><strong><Money amountKobo={view.obligation?.outstanding_kobo ?? view.request.principal_kobo} /></strong></a>{/each}</div>{/if}</section>{/if}
    <FeedbackPrompt area="seller" {organizationID} />
  {/if}
</main>
<style>
  .account-home{max-width:66rem}.task-heading{display:flex;align-items:center;justify-content:space-between;gap:1.5rem;padding-block:1rem 1.5rem}.task-heading h1{margin:.25rem 0;font-family:inherit;font-size:clamp(1.8rem,4vw,2.4rem);line-height:1.2;letter-spacing:-.035em}.task-heading .lede{font-size:1rem;margin:.5rem 0 0}.toolbar{display:flex;align-items:end;flex-wrap:wrap;gap:1rem;margin-bottom:1.5rem}.toolbar label{display:grid;gap:.4rem;flex:1;max-width:24rem}.toolbar select{width:100%;font:inherit;padding:.7rem;background:var(--color-surface);border:1px solid var(--color-border);border-radius:.35rem}.toolbar button{padding:.7rem 1rem;border:1px solid var(--color-border);background:var(--color-surface);border-radius:.35rem}.toolbar a{display:flex;align-items:center;color:var(--color-primary)}.balance-card{padding:1.5rem;background:#17181b;color:#fffdf8;border-radius:.65rem}.balance-card>p{margin:0;color:#d7d4cc}.balance{display:block;margin:.5rem 0;font-size:clamp(2.1rem,6vw,3.5rem);line-height:1.2;letter-spacing:-.03em;font-variant-numeric:tabular-nums;overflow-wrap:anywhere}.balance-caption{font-size:.9rem}.late-strip{display:flex;gap:1rem;align-items:center;flex-wrap:wrap;margin:1.5rem -.5rem -.5rem;padding:1rem;border-radius:.35rem;background:#fff0e8;color:#702c18;text-decoration:none}.late-strip strong{margin-left:auto}.checked-state{margin-top:1rem!important;font-size:.9rem}.attention-section,.recent-sales{margin-block:2rem}.section-heading{display:flex;align-items:baseline;justify-content:space-between;gap:1rem;margin-bottom:1rem}.section-heading h2{font-size:1.25rem;margin:0}.section-heading p{font-size:.9rem;color:var(--color-muted);margin:.4rem 0 0}.section-heading a{color:var(--color-primary)}.action-list{border:1px solid var(--color-border);border-radius:.5rem;background:var(--color-surface);overflow:hidden}.action-list article{display:flex;align-items:center;justify-content:space-between;gap:1.5rem;padding:1rem 1.25rem;border-bottom:1px solid var(--color-border)}.action-list article:last-child{border:0}.action-list h3{font-family:inherit;font-size:1rem;margin:0}.action-list p{font-size:.9rem;line-height:1.5;color:var(--color-muted);margin:.35rem 0 0}.action-list a{display:flex;align-items:center;gap:.5rem;color:var(--color-primary);font-weight:650;white-space:nowrap}.show-more{margin-top:1rem;padding:.75rem 1rem;background:transparent;border:1px solid var(--color-border);border-radius:.35rem}.record-list{border-top:1px solid var(--color-border)}.record-row{display:flex;align-items:center;justify-content:space-between;gap:1rem;padding:1rem .25rem;border-bottom:1px solid var(--color-border);text-decoration:none}.record-row span{display:grid;gap:.4rem}.record-row small{color:var(--color-muted)}.record-row>strong{font-variant-numeric:tabular-nums;text-align:right}.onboarding{padding:1.5rem;max-width:48rem;margin:auto}.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:1rem}.form-grid label{display:grid;gap:.45rem}.form-grid input,.form-grid select,.form-grid textarea{box-sizing:border-box;width:100%;font:inherit;padding:.8rem;border:1px solid var(--color-border);border-radius:.35rem;background:var(--color-surface)}.wide{grid-column:1/-1}.empty-state{padding:1.5rem;border:1px dashed var(--color-border);border-radius:.5rem}.empty-state h3{margin:0;font-size:1rem}.empty-state p{line-height:1.6;color:var(--color-muted)}@media(max-width:560px){.task-heading{align-items:start;flex-wrap:wrap}.action-list article{align-items:start;flex-direction:column;gap:.4rem}.action-list a{min-height:2.75rem}.form-grid{grid-template-columns:1fr}.record-row{align-items:start}.record-row>strong{font-size:.95rem}.section-heading{flex-wrap:wrap}}
</style>
