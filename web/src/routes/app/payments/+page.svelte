<script lang="ts">
  import { getContext, onMount } from 'svelte';
  import { sumKobo } from '$lib/money';
  import { ACCOUNT_CONTEXT, type AccountContext } from '$lib/account-context';
  import { checkedJSON, LatestRequest, publicError, record, rows, text } from '$lib/api/reliable';
  import { MutationIntent } from '$lib/api/mutation';
  import { kobo, organization, timeLabel, type Organization } from '$lib/records';
  import type { KoboValue } from '$lib/money';
  import Money from '$lib/components/Money.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import PaymentReview from '$lib/components/PaymentReview.svelte';
  type Payment = { id: string; payment_id: string; buyer_legal_name: string; description: string; reference: string; amount_kobo: KoboValue; state: string; source_type: string; paid_at: string };
  type Claim = { id: string; state: string; amount_kobo: KoboValue; transfer_reference: string; paid_at: string; hold_expires_at: string };
  const account = getContext<AccountContext>(ACCOUNT_CONTEXT);
  const optional = (value: unknown) => typeof value === 'string' ? value : '';
  const payment = (value: unknown): Payment => { const row = record(value); return { id: text(row.id), payment_id: optional(row.payment_id), buyer_legal_name: text(row.buyer_legal_name), description: optional(row.description), reference: optional(row.reference), amount_kobo: kobo(row.amount_kobo), state: text(row.state), source_type: text(row.source_type), paid_at: text(row.paid_at) }; };
  const claimRow = (value: unknown): Claim => { const row = record(value); return { id: text(row.id), state: text(row.state), amount_kobo: kobo(row.amount_kobo), transfer_reference: text(row.transfer_reference), paid_at: text(row.paid_at), hold_expires_at: optional(row.hold_expires_at) }; };
  let organizations = $state<Organization[]>([]), payments = $state<Payment[]>([]), claims = $state<Claim[]>([]);
  let organizationID = $state(''), error = $state(''), busy = $state(''), query = $state(''), status = $state('all'), reviewError = $state('');
  let loading = $state(true), review = $state<{ organizationID: string; claim: Claim; decision: 'confirmed' | 'rejected' } | null>(null);
  const reads = new LatestRequest(), businesses = new LatestRequest();
  const intents = new Map<string, MutationIntent>();
  const pendingClaims = $derived(claims.filter(claim => claim.state === 'pending'));
  const pendingTotal = $derived(sumKobo(pendingClaims.map(claim => claim.amount_kobo)));
  const receivedTotal = $derived(sumKobo(payments.filter(payment => payment.state === 'recognized').map(payment => payment.amount_kobo)));
  const visiblePayments = $derived(payments.filter(payment => `${payment.buyer_legal_name} ${payment.description} ${payment.reference}`.toLowerCase().includes(query.trim().toLowerCase()) && (status === 'all' || payment.state === status)));
  const date = (value: string) => value ? timeLabel(value) : 'Not available';
  const source = (value: string) => ({ integrated_voluntary: 'Paid online', supplier_recorded_transfer: 'Bank transfer', buyer_payment_claim: 'Customer reported transfer', cash_recorded: 'Cash', kredit_collection: 'Collected by Kredit', adjustment: 'Account correction' } as Record<string, string>)[value] ?? 'Payment';
  const stateLabel = (value: string) => ({ recognized: 'Received', reversed: 'Reversed', pending: 'Awaiting confirmation', confirmed: 'Received', rejected: 'Not received', expired: 'Expired' } as Record<string, string>)[value] ?? 'Status unavailable';
  async function load() {
    const scope = organizationID, request = reads.begin(); loading = true; error = ''; payments = []; claims = [];
    if (!scope) { loading = false; return; }
    try {
      const root = `/api/v1/organizations/${encodeURIComponent(scope)}`;
      const [newPayments, newClaims] = await Promise.all([
        checkedJSON(`${root}/payments`, rows('payments', payment), { signal: request.signal }),
        checkedJSON(`${root}/payment-claims`, rows('payment_claims', claimRow), { signal: request.signal })
      ]);
      if (!request.current() || organizationID !== scope) return;
      payments = newPayments; claims = newClaims;
    } catch (cause) { if (request.current() && organizationID === scope) error = publicError(cause, 'your payments'); }
    finally { if (request.current() && organizationID === scope) loading = false; }
  }
  function decide(claim: Claim, decision: 'confirmed' | 'rejected') {
    if (busy || claim.state !== 'pending') return;
    reviewError = ''; review = { organizationID, claim, decision };
  }
  async function confirmDecision() {
    if (!review || busy) return;
    const selected = review; busy = selected.claim.id; reviewError = '';
    const url = `/api/v1/organizations/${encodeURIComponent(selected.organizationID)}/payment-claims/${encodeURIComponent(selected.claim.id)}/decide`;
    try {
      let intent = intents.get(url);
      if (!intent) { intent = new MutationIntent(account.userID, url); intents.set(url, intent); }
      await intent.run({ decision: selected.decision, reason: selected.decision === 'confirmed' ? 'Supplier checked the receiving account and confirmed the transfer.' : 'Supplier checked the receiving account and could not locate the transfer.' }, value => claimRow(record(value).payment_claim));
      review = null;
      if (organizationID === selected.organizationID) await load();
    } catch (cause) { reviewError = cause instanceof Error ? cause.message : 'The decision was not confirmed. Check the record before submitting another.'; }
    finally { busy = ''; }
  }
  async function start() {
    const request = businesses.begin(); loading = true; error = '';
    try {
      const result = await checkedJSON('/api/v1/organizations', rows('organizations', organization), { signal: request.signal });
      if (!request.current()) return;
      organizations = result;
      const requested = new URLSearchParams(location.search).get('organization');
      organizationID = result.find(item => item.id === (requested || organizationID))?.id ?? result[0]?.id ?? '';
      await load();
    } catch (cause) { if (request.current()) { error = publicError(cause, 'your businesses'); loading = false; } }
  }
  onMount(() => { void start(); return () => { reads.cancel(); businesses.cancel(); }; });
</script>

<svelte:head><title>Payments — Kredit</title></svelte:head>
<main class="shell workspace payments-page">
	<header class="page-heading"><div><p class="eyebrow">Payments</p><h1>Your money, clearly.</h1><p class="lede">See what has actually entered your account, and check the payments customers say they made.</p></div>{#if organizations.length > 1}<label>Business<select bind:value={organizationID} disabled={!!busy} onchange={load}>{#each organizations as organization}<option value={organization.id}>{organization.trading_name || organization.legal_name}</option>{/each}</select></label>{/if}</header>
	{#if error}<div class="error-box" role="alert"><div><strong>We could not open your payments.</strong><p>{error}</p></div><button type="button" onclick={() => organizationID ? load() : start()}>Try again</button></div>{/if}
	{#if loading}<div class="loading" role="status"><span class="sr-only">Opening your payments</span><Skeleton rows={5} tall /></div>{:else if !error && !organizationID}<section class="empty-state"><h2>Add your business first</h2><a href="/app/overview">Add business details</a></section>{:else if !error}
		<section class="money-summary" aria-label="Payment summary"><article class="total"><span>Money received</span><strong><Money amountKobo={receivedTotal} /></strong><small>Every payment you confirmed</small></article><article class:needs-action={pendingClaims.length > 0}><span>Waiting for your answer</span><strong><Money amountKobo={pendingTotal} /></strong><small>{pendingClaims.length} {pendingClaims.length === 1 ? 'payment' : 'payments'} to check</small></article><article><span>Payments saved</span><strong>{payments.length}</strong><small>Payments in this checked record</small></article></section>
		<section class="review-section" aria-labelledby="review-title"><header><div><p class="eyebrow">Needs your answer</p><h2 id="review-title">Check your bank for these.</h2></div><span>{pendingClaims.length}</span></header>
		{#if pendingClaims.length}<div class="claim-list">{#each pendingClaims as claim}<article><div class="claim-amount"><span>Transfer reported</span><strong><Money amountKobo={claim.amount_kobo} /></strong></div><dl><div><dt>Transfer number</dt><dd>{claim.transfer_reference}</dd></div><div><dt>Payment day</dt><dd>{date(claim.paid_at)}</dd></div><div><dt>Check before</dt><dd>{date(claim.hold_expires_at)}</dd></div></dl><p>Open your bank app and look for the money before you answer.</p><div class="claim-actions"><button disabled={!!busy} onclick={() => decide(claim, 'confirmed')}>{busy === claim.id ? 'Saving…' : 'Yes, I got the money'}</button><button class="secondary" disabled={!!busy} onclick={() => decide(claim, 'rejected')}>I cannot find this money</button></div></article>{/each}</div>{:else}<div class="all-clear"><span aria-hidden="true">✓</span><div><h3>Nothing to check right now.</h3><p>You have answered every payment your customers reported.</p></div></div>{/if}</section>
		<section class="history" aria-labelledby="history-title"><header><div><p class="eyebrow">Your records</p><h2 id="history-title">Money received.</h2></div><div class="filters"><label><span>Find a payment</span><input type="search" bind:value={query} placeholder="Customer or transfer number" /></label><label><span>Show</span><select bind:value={status}><option value="all">All payments</option><option value="recognized">Received</option><option value="reversed">Reversed</option></select></label></div></header>
		{#if visiblePayments.length}<div class="payment-table" role="table" aria-label="Payments received"><div class="table-head" role="row"><span role="columnheader">Customer</span><span role="columnheader">Amount</span><span role="columnheader">How</span><span role="columnheader">Date</span><span role="columnheader">Status</span><span aria-hidden="true"></span></div>{#each visiblePayments as payment}<div class="payment-row" role="row"><div role="cell"><strong>{payment.buyer_legal_name || 'Customer'}</strong><small>{payment.description || payment.reference || 'Sale payment'}</small></div><div role="cell"><strong><Money amountKobo={payment.amount_kobo} /></strong></div><span role="cell">{source(payment.source_type)}</span><span role="cell">{date(payment.paid_at)}</span><span role="cell" class:reversed={payment.state === 'reversed'} class="payment-state">{stateLabel(payment.state)}</span><a role="cell" href={`/app/credit/${payment.id}?organization=${organizationID}`}>Open sale →</a></div>{/each}</div>
		{:else if payments.length}<div class="empty-history"><h3>No payment matches that.</h3><p>Try a different customer name, transfer number or status.</p></div>{:else}<div class="empty-history"><span aria-hidden="true">₦</span><h3>No confirmed payments yet.</h3><p>Verified payments appear here. A reported transfer stays separate until it is confirmed.</p><a class="primary" href="/app/credit/new">Add a sale</a></div>{/if}</section>
	{/if}
</main>
{#if review}<PaymentReview amount={review.claim.amount_kobo} reference={review.claim.transfer_reference} decision={review.decision} busy={!!busy} error={reviewError} onconfirm={confirmDecision} oncancel={() => { review = null; reviewError = ''; }} />{/if}

<style>
	.payments-page{max-width:76rem;padding-bottom:5rem}.page-heading{display:flex;justify-content:space-between;align-items:end;gap:2rem;padding-bottom:2.2rem;border-bottom:3px solid #17181b}.page-heading>div{max-width:48rem}.page-heading h1{max-width:11ch;margin:.45rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(3.2rem,7vw,6rem);font-weight:500;line-height:.9;letter-spacing:-.06em}.page-heading label,.filters label{display:grid;gap:.35rem;font-size:.82rem;font-weight:750}.page-heading select,.filters input,.filters select{box-sizing:border-box;min-height:3rem;padding:.65rem .75rem;border:1px solid #99958c;border-radius:0;background:#fff;color:#17181b;font:inherit}.error-box{display:flex;justify-content:space-between;gap:1rem;align-items:center;margin:1.5rem 0;padding:1rem;color:#fff;background:#b42318}.error-box p{margin:.25rem 0}.error-box button{padding:.65rem 1rem;border:0;border-radius:0;background:#fff;color:#17181b;font-weight:750}.loading{padding:2rem 0}.money-summary{display:grid;grid-template-columns:1.4fr 1fr 1fr;margin:2rem 0 4rem;border-top:1px solid var(--color-border);border-left:1px solid var(--color-border)}.money-summary article{display:grid;align-content:start;min-height:9rem;padding:1.35rem;border-right:1px solid var(--color-border);border-bottom:1px solid var(--color-border);background:#fffdf8}.money-summary article.total{color:#fff;background:#2738d6}.money-summary article.needs-action{box-shadow:inset 0 .35rem #ff5b3a}.money-summary span,.money-summary small{font-size:.82rem}.money-summary strong{margin:.6rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(2rem,4vw,3.3rem);font-weight:500;letter-spacing:-.04em}.money-summary .total>span,.money-summary .total strong,.money-summary .total strong :global(span){color:#fff}.money-summary .total small{color:#d9dcff}.review-section{margin-bottom:5rem}.review-section>header,.history>header{display:flex;justify-content:space-between;align-items:end;gap:2rem;padding-bottom:1rem;border-bottom:1px solid var(--color-border)}.review-section h2,.history h2{margin:.2rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(2rem,4vw,3.4rem);font-weight:500;letter-spacing:-.04em}.review-section>header>span{display:grid;place-items:center;width:2.6rem;height:2.6rem;background:#ff5b3a;color:#17181b;font-weight:850}.claim-list{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1rem;margin-top:1rem}.claim-list article{padding:1.4rem;border:1px solid var(--color-border);background:#fffdf8;box-shadow:6px 6px 0 #ded8cc}.claim-amount{display:flex;justify-content:space-between;gap:1rem;padding-bottom:1rem;border-bottom:2px solid #17181b}.claim-amount span{max-width:10rem}.claim-amount strong{font-family:Georgia,'Times New Roman',serif;font-size:1.7rem;font-weight:500}.claim-list dl{display:grid;gap:.6rem}.claim-list dl div{display:flex;justify-content:space-between;gap:1rem}.claim-list dt{color:var(--color-muted)}.claim-list dd{margin:0;text-align:right;font-weight:700}.claim-list>article>p{padding:.7rem;background:#f0ece3}.claim-actions{display:grid;grid-template-columns:1fr 1fr;gap:.6rem}.claim-actions button{min-height:3rem;padding:.65rem;border:1px solid #2738d6;border-radius:0;background:#2738d6;color:#fff;font:inherit;font-weight:750}.claim-actions .secondary{border-color:#17181b;background:#fff;color:#17181b}.all-clear{display:flex;gap:1rem;align-items:center;padding:2rem 0}.all-clear>span{display:grid;place-items:center;width:3.2rem;height:3.2rem;background:#e6f7ed;color:#126542;font-size:1.4rem;font-weight:900}.all-clear h3,.all-clear p{margin:.2rem 0}.filters{display:flex;gap:.7rem}.filters label:first-child{min-width:min(19rem,50vw)}.payment-table{margin-top:1rem;border-top:1px solid var(--color-border)}.table-head,.payment-row{display:grid;grid-template-columns:minmax(11rem,1.5fr) minmax(8rem,.8fr) minmax(9rem,1fr) minmax(7rem,.8fr) minmax(6rem,.7fr) auto;gap:1rem;align-items:center;padding:.85rem;border-right:1px solid var(--color-border);border-bottom:1px solid var(--color-border);border-left:1px solid var(--color-border)}.table-head{background:#17181b;color:#fff;font-size:.75rem;font-weight:750;text-transform:uppercase;letter-spacing:.06em}.payment-row{background:#fffdf8}.payment-row>div{display:grid;gap:.2rem}.payment-row small{color:var(--color-muted)}.payment-row a{color:#2738d6;font-weight:750;text-decoration:none;white-space:nowrap}.payment-state{width:max-content;padding:.3rem .5rem;background:#e6f7ed;color:#126542;font-size:.78rem;font-weight:800}.payment-state.reversed{background:#fde8e4;color:#9b2c20}.empty-history{margin-top:1rem;padding:clamp(2rem,6vw,4rem);border:1px solid var(--color-border);background:#ebe7de}.empty-history>span{font-family:Georgia,'Times New Roman',serif;font-size:3rem;color:#2738d6}.empty-history h3{margin:.5rem 0;font-family:Georgia,'Times New Roman',serif;font-size:2rem;font-weight:500}.empty-history p{max-width:34rem;color:var(--color-muted)}@media(max-width:800px){.page-heading,.review-section>header,.history>header{display:block}.page-heading>label{margin-top:1rem}.money-summary{grid-template-columns:1fr}.claim-list{grid-template-columns:1fr}.filters{display:grid;margin-top:1rem}.filters label:first-child{min-width:0}.payment-table{border:0}.table-head{display:none}.payment-row{grid-template-columns:1fr auto;gap:.65rem}.payment-row>[role='cell']{grid-column:1}.payment-row>[role='cell']:nth-child(2){grid-column:2;grid-row:1}.payment-row>[role='cell']:nth-child(3),.payment-row>[role='cell']:nth-child(4){display:inline}.payment-row>[role='cell']:nth-child(5){grid-column:1}.payment-row>a[role='cell']{grid-column:2;grid-row:3}.claim-actions{grid-template-columns:1fr}.error-box{align-items:stretch;flex-direction:column}.error-box button{width:100%}}
	/* The most important number uses the strongest contrast in the product. */
	.money-summary article.total{color:#fff;background:#17181b;box-shadow:inset 0 .4rem #ff5b3a}
	.money-summary .total>span,.money-summary .total strong,.money-summary .total strong :global(span){color:#fff!important}
	.money-summary .total small{color:#d8d7d2}
</style>
