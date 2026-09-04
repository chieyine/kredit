<script lang="ts">
	import { onMount } from 'svelte';
 import { sumKobo } from '$lib/money';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';
	import Money from '$lib/components/Money.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';

	let organizations: any[] = $state([]), payments: any[] = $state([]), claims: any[] = $state([]);
	let organizationID = $state(''), error = $state(''), busy = $state(''), query = $state(''), status = $state('all');
	let loading = $state(true);
	const pendingClaims = $derived(claims.filter((claim) => claim.state === 'pending'));
	const pendingTotal = $derived(sumKobo(pendingClaims.map((claim)=>claim.amount_kobo)));
	const receivedTotal = $derived(sumKobo(payments.filter((payment)=>payment.state==='recognized').map((payment)=>payment.amount_kobo)));
	const visiblePayments = $derived(payments.filter((payment) => {
		const words = `${payment.buyer_legal_name ?? ''} ${payment.description ?? ''} ${payment.reference ?? ''}`.toLowerCase();
		return (status === 'all' || payment.state === status) && words.includes(query.trim().toLowerCase());
	}));
	const date = (value: string) => value ? new Date(value).toLocaleDateString('en-NG', { day: 'numeric', month: 'short', year: 'numeric' }) : 'Date not available';
	const source = (value: string) => ({ integrated_voluntary: 'Paid online', supplier_recorded_transfer: 'Bank transfer', buyer_payment_claim: 'Customer reported payment', cash_recorded: 'Cash', kredit_collection: 'Collected by Kredit', adjustment: 'Account correction' } as Record<string, string>)[value] ?? 'Payment';
	const stateLabel = (value: string) => ({ recognized: 'Received', reversed: 'Reversed', pending: 'Check now', confirmed: 'Received', rejected: 'Not received', expired: 'Time ended' } as Record<string, string>)[value] ?? value?.replaceAll('_', ' ') ?? 'Recorded';

	function signIn() { window.location.replace(`/app?next=${encodeURIComponent('/app/payments')}`); }
	async function load() {
		if (!organizationID) return;
		loading = true; error = '';
		try {
			const [p, c] = await Promise.all([fetch(`/api/v1/organizations/${organizationID}/payments`, { credentials: 'include' }), fetch(`/api/v1/organizations/${organizationID}/payment-claims`, { credentials: 'include' })]);
			if (p.status === 401 || c.status === 401) { signIn(); return; }
			if (!p.ok || !c.ok) throw new Error('We could not open your payment records.');
			payments = (await p.json()).payments ?? []; claims = (await c.json()).payment_claims ?? [];
		} catch (cause) { error = cause instanceof Error ? cause.message : 'We could not open your payment records.'; }
		finally { loading = false; }
	}
 async function decide(claim: any, decision: 'confirmed' | 'rejected') {
  busy=claim.id;error='';
  try {
   const reason=decision==='confirmed'?'Seller confirmed the money arrived':'Seller could not find this payment';
   const response=await fetch(`/api/v1/organizations/${organizationID}/payment-claims/${claim.id}/decide`,{method:'POST',credentials:'include',headers:{'Content-Type':'application/json','Idempotency-Key':idempotencyKey(),...csrfHeaders()},body:JSON.stringify({decision,reason})});
   const result=await response.json().catch(()=>({}));
   if(!response.ok)throw new Error(result.detail??'We could not save your answer.');
   await load();
  }catch(cause){error=cause instanceof Error?cause.message:'We could not save your answer.'}finally{busy=''}
 }

	onMount(async () => {
		try {
			const response = await fetch('/api/v1/organizations', { credentials: 'include' });
			if (response.status === 401) { signIn(); return; }
			if (!response.ok) throw new Error('We could not open your business.');
			organizations = (await response.json()).organizations ?? []; organizationID = organizations[0]?.id ?? '';
			if (organizationID) await load(); else loading = false;
		} catch (cause) { error = cause instanceof Error ? cause.message : 'We could not open your business.'; loading = false; }
	});
</script>

<svelte:head><title>Payments — Kredit</title></svelte:head>
<main class="shell workspace payments-page">
	<header class="page-heading"><div><p class="eyebrow">Payments</p><h1>Your money, clearly.</h1><p class="lede">See what has entered your account and check payments customers say they made.</p></div>{#if organizations.length > 1}<label>Business<select bind:value={organizationID} onchange={load}>{#each organizations as organization}<option value={organization.id}>{organization.trading_name || organization.legal_name}</option>{/each}</select></label>{/if}</header>
	{#if error}<div class="error-box" role="alert"><div><strong>Payments could not open.</strong><p>{error}</p></div><button type="button" onclick={load}>Try again</button></div>{/if}
	{#if loading}<div class="loading" role="status"><span class="sr-only">Opening your payments</span><Skeleton rows={5} tall /></div>{:else if !error}
		<section class="money-summary" aria-label="Payment summary"><article class="total"><span>Money received</span><strong><Money amountKobo={receivedTotal} /></strong><small>All confirmed payments</small></article><article class:needs-action={pendingClaims.length > 0}><span>Waiting for your answer</span><strong><Money amountKobo={pendingTotal} /></strong><small>{pendingClaims.length} {pendingClaims.length === 1 ? 'payment' : 'payments'} to check</small></article><article><span>Payments recorded</span><strong>{payments.length}</strong><small>Complete payment history</small></article></section>
		<section class="review-section" aria-labelledby="review-title"><header><div><p class="eyebrow">Needs your answer</p><h2 id="review-title">Check these payments.</h2></div><span>{pendingClaims.length}</span></header>
		{#if pendingClaims.length}<div class="claim-list">{#each pendingClaims as claim}<article><div class="claim-amount"><span>Customer says they paid</span><strong><Money amountKobo={claim.amount_kobo} /></strong></div><dl><div><dt>Transfer number</dt><dd>{claim.transfer_reference}</dd></div><div><dt>Payment day</dt><dd>{date(claim.paid_at)}</dd></div><div><dt>Check before</dt><dd>{date(claim.hold_expires_at)}</dd></div></dl><p>Look at your bank account before answering.</p><div class="claim-actions"><button disabled={busy === claim.id} onclick={() => decide(claim, 'confirmed')}>{busy === claim.id ? 'Saving…' : 'Yes, I got the money'}</button><button class="secondary" disabled={busy === claim.id} onclick={() => decide(claim, 'rejected')}>I cannot find it</button></div></article>{/each}</div>{:else}<div class="all-clear"><span aria-hidden="true">✓</span><div><h3>Nothing to check.</h3><p>You have answered every payment report.</p></div></div>{/if}</section>
		<section class="history" aria-labelledby="history-title"><header><div><p class="eyebrow">Your records</p><h2 id="history-title">Payments received.</h2></div><div class="filters"><label><span>Find a payment</span><input type="search" bind:value={query} placeholder="Customer or transfer number" /></label><label><span>Show</span><select bind:value={status}><option value="all">All payments</option><option value="recognized">Received</option><option value="reversed">Reversed</option></select></label></div></header>
		{#if visiblePayments.length}<div class="payment-table" role="table" aria-label="Payments received"><div class="table-head" role="row"><span role="columnheader">Customer</span><span role="columnheader">Amount</span><span role="columnheader">How</span><span role="columnheader">Date</span><span role="columnheader">Status</span><span aria-hidden="true"></span></div>{#each visiblePayments as payment}<div class="payment-row" role="row"><div role="cell"><strong>{payment.buyer_legal_name || 'Customer'}</strong><small>{payment.description || payment.reference || 'Sale payment'}</small></div><div role="cell"><strong><Money amountKobo={payment.amount_kobo} /></strong></div><span role="cell">{source(payment.source_type)}</span><span role="cell">{date(payment.paid_at)}</span><span role="cell" class:reversed={payment.state === 'reversed'} class="payment-state">{stateLabel(payment.state)}</span><a role="cell" href={`/app/credit/${payment.id}?organization=${organizationID}`}>Open sale →</a></div>{/each}</div>
		{:else if payments.length}<div class="empty-history"><h3>No payment matches.</h3><p>Try another customer, transfer number or status.</p></div>{:else}<div class="empty-history"><span aria-hidden="true">₦</span><h3>No payment has arrived yet.</h3><p>When a customer pays, the money and payment details will appear here.</p><a class="primary" href="/app/credit/new">Add a sale</a></div>{/if}</section>
	{/if}
</main>

<style>
	.payments-page{max-width:76rem;padding-bottom:5rem}.page-heading{display:flex;justify-content:space-between;align-items:end;gap:2rem;padding-bottom:2.2rem;border-bottom:3px solid #17181b}.page-heading>div{max-width:48rem}.page-heading h1{max-width:11ch;margin:.45rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(3.2rem,7vw,6rem);font-weight:500;line-height:.9;letter-spacing:-.06em}.page-heading label,.filters label{display:grid;gap:.35rem;font-size:.82rem;font-weight:750}.page-heading select,.filters input,.filters select{box-sizing:border-box;min-height:3rem;padding:.65rem .75rem;border:1px solid #99958c;border-radius:0;background:#fff;color:#17181b;font:inherit}.error-box{display:flex;justify-content:space-between;gap:1rem;align-items:center;margin:1.5rem 0;padding:1rem;color:#fff;background:#b42318}.error-box p{margin:.25rem 0}.error-box button{padding:.65rem 1rem;border:0;border-radius:0;background:#fff;color:#17181b;font-weight:750}.loading{padding:2rem 0}.money-summary{display:grid;grid-template-columns:1.4fr 1fr 1fr;margin:2rem 0 4rem;border-top:1px solid var(--color-border);border-left:1px solid var(--color-border)}.money-summary article{display:grid;align-content:start;min-height:9rem;padding:1.35rem;border-right:1px solid var(--color-border);border-bottom:1px solid var(--color-border);background:#fffdf8}.money-summary article.total{color:#fff;background:#2738d6}.money-summary article.needs-action{box-shadow:inset 0 .35rem #ff5b3a}.money-summary span,.money-summary small{font-size:.82rem}.money-summary strong{margin:.6rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(2rem,4vw,3.3rem);font-weight:500;letter-spacing:-.04em}.money-summary .total>span,.money-summary .total strong,.money-summary .total strong :global(span){color:#fff}.money-summary .total small{color:#d9dcff}.review-section{margin-bottom:5rem}.review-section>header,.history>header{display:flex;justify-content:space-between;align-items:end;gap:2rem;padding-bottom:1rem;border-bottom:1px solid var(--color-border)}.review-section h2,.history h2{margin:.2rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(2rem,4vw,3.4rem);font-weight:500;letter-spacing:-.04em}.review-section>header>span{display:grid;place-items:center;width:2.6rem;height:2.6rem;background:#ff5b3a;color:#17181b;font-weight:850}.claim-list{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1rem;margin-top:1rem}.claim-list article{padding:1.4rem;border:1px solid var(--color-border);background:#fffdf8;box-shadow:6px 6px 0 #ded8cc}.claim-amount{display:flex;justify-content:space-between;gap:1rem;padding-bottom:1rem;border-bottom:2px solid #17181b}.claim-amount span{max-width:10rem}.claim-amount strong{font-family:Georgia,'Times New Roman',serif;font-size:1.7rem;font-weight:500}.claim-list dl{display:grid;gap:.6rem}.claim-list dl div{display:flex;justify-content:space-between;gap:1rem}.claim-list dt{color:var(--color-muted)}.claim-list dd{margin:0;text-align:right;font-weight:700}.claim-list>article>p{padding:.7rem;background:#f0ece3}.claim-actions{display:grid;grid-template-columns:1fr 1fr;gap:.6rem}.claim-actions button{min-height:3rem;padding:.65rem;border:1px solid #2738d6;border-radius:0;background:#2738d6;color:#fff;font:inherit;font-weight:750}.claim-actions .secondary{border-color:#17181b;background:#fff;color:#17181b}.all-clear{display:flex;gap:1rem;align-items:center;padding:2rem 0}.all-clear>span{display:grid;place-items:center;width:3.2rem;height:3.2rem;background:#e6f7ed;color:#126542;font-size:1.4rem;font-weight:900}.all-clear h3,.all-clear p{margin:.2rem 0}.filters{display:flex;gap:.7rem}.filters label:first-child{min-width:min(19rem,50vw)}.payment-table{margin-top:1rem;border-top:1px solid var(--color-border)}.table-head,.payment-row{display:grid;grid-template-columns:minmax(11rem,1.5fr) minmax(8rem,.8fr) minmax(9rem,1fr) minmax(7rem,.8fr) minmax(6rem,.7fr) auto;gap:1rem;align-items:center;padding:.85rem;border-right:1px solid var(--color-border);border-bottom:1px solid var(--color-border);border-left:1px solid var(--color-border)}.table-head{background:#17181b;color:#fff;font-size:.75rem;font-weight:750;text-transform:uppercase;letter-spacing:.06em}.payment-row{background:#fffdf8}.payment-row>div{display:grid;gap:.2rem}.payment-row small{color:var(--color-muted)}.payment-row a{color:#2738d6;font-weight:750;text-decoration:none;white-space:nowrap}.payment-state{width:max-content;padding:.3rem .5rem;background:#e6f7ed;color:#126542;font-size:.78rem;font-weight:800}.payment-state.reversed{background:#fde8e4;color:#9b2c20}.empty-history{margin-top:1rem;padding:clamp(2rem,6vw,4rem);border:1px solid var(--color-border);background:#ebe7de}.empty-history>span{font-family:Georgia,'Times New Roman',serif;font-size:3rem;color:#2738d6}.empty-history h3{margin:.5rem 0;font-family:Georgia,'Times New Roman',serif;font-size:2rem;font-weight:500}.empty-history p{max-width:34rem;color:var(--color-muted)}@media(max-width:800px){.page-heading,.review-section>header,.history>header{display:block}.page-heading>label{margin-top:1rem}.money-summary{grid-template-columns:1fr}.claim-list{grid-template-columns:1fr}.filters{display:grid;margin-top:1rem}.filters label:first-child{min-width:0}.payment-table{border:0}.table-head{display:none}.payment-row{grid-template-columns:1fr auto;gap:.65rem}.payment-row>[role='cell']{grid-column:1}.payment-row>[role='cell']:nth-child(2){grid-column:2;grid-row:1}.payment-row>[role='cell']:nth-child(3),.payment-row>[role='cell']:nth-child(4){display:inline}.payment-row>[role='cell']:nth-child(5){grid-column:1}.payment-row>a[role='cell']{grid-column:2;grid-row:3}.claim-actions{grid-template-columns:1fr}.error-box{align-items:stretch;flex-direction:column}.error-box button{width:100%}}
	/* The most important number uses the strongest contrast in the product. */
	.money-summary article.total{color:#fff;background:#17181b;box-shadow:inset 0 .4rem #ff5b3a}
	.money-summary .total>span,.money-summary .total strong,.money-summary .total strong :global(span){color:#fff!important}
	.money-summary .total small{color:#d8d7d2}
</style>
