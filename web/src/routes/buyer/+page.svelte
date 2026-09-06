<script lang="ts">
	import { onMount } from 'svelte';
	import { formatKobo, sumKobo } from '$lib/money';
	import { productLabel } from '$lib/product-language';
	import FeedbackPrompt from '$lib/components/FeedbackPrompt.svelte';

	let portal = $state<any>(null);
	let requests = $state<any[]>([]);
	let error = $state('');
	let loading = $state(true);
	const outstanding = $derived(sumKobo(requests.map((item) => item.obligation?.outstanding_kobo ?? 0)));
	const pending = $derived(requests.filter((item)=>['SENT','BUYER_REVIEWING','PENDING_BUYER_CONFIRMATION'].includes(String(item.request?.state??item.state??'').toUpperCase())));
	const openBalances = $derived(requests.filter((item)=>Number(item.obligation?.outstanding_kobo??0)>0));
	const nextPayment = $derived.by(()=>{
		const dated=openBalances.map((item)=>({item,date:new Date(`${item.request?.due_date??item.obligation?.due_date??''}T12:00:00`)})).filter((entry)=>!Number.isNaN(entry.date.getTime())).sort((a,b)=>a.date.getTime()-b.date.getTime());
		return dated[0]??null;
	});
	const overdueCount = $derived(openBalances.filter((item)=>{
		const state=String(item.obligation?.state??item.request?.state??'').toUpperCase();
		if(state==='OVERDUE')return true;
		const due=item.request?.due_date??item.obligation?.due_date;
		return due?new Date(`${due}T23:59:59`).getTime()<Date.now():false;
	}).length);

	onMount(async () => {
		loading=true;
		const [meResponse,requestsResponse] = await Promise.all([fetch('/api/v1/buyer/me'),fetch('/api/v1/buyer/credit-requests')]);
		if (!meResponse.ok) {
			error = 'Open the private link your seller sent you. That link is how you get into this account.';
			loading=false;
			return;
		}
		portal = (await meResponse.json()).portal;
		if(requestsResponse.ok) requests=(await requestsResponse.json()).requests??[];
		loading=false;
	});
</script>

<svelte:head><title>Customer account — Kredit</title></svelte:head>

<main class="shell buyer-home">
	{#if portal}
		<header class="buyer-head"><div><p class="eyebrow">Your Kredit account</p><h1>Here is what you owe.</h1><p>Signed in as <strong>{portal.person.full_name}</strong> for {portal.business.legal_name}.</p></div>{#if pending.length}<a class="primary" href="/buyer/requests">Read {pending.length} sale{pending.length===1?'':'s'} waiting for you</a>{/if}</header>
		{#if loading}<p role="status">Opening your balances…</p>{:else}
			<section class="balance-board" aria-label="Your credit summary">
				<article class="main-balance"><span>Money left to pay</span><strong>{formatKobo(outstanding)}</strong><small>Across {openBalances.length} sale{openBalances.length===1?'':'s'}</small><a href="/buyer/obligations">See what I owe →</a></article>
				<article><span>Next payment</span>{#if nextPayment}<strong>{new Date(nextPayment.date).toLocaleDateString('en-NG',{day:'numeric',month:'short'})}</strong><small>{formatKobo(nextPayment.item.obligation?.outstanding_kobo??0)} left on that sale</small>{:else}<strong>—</strong><small>Nothing due yet</small>{/if}<a href="/buyer/obligations">See payment dates →</a></article>
				<article class:danger={overdueCount>0}><span>Overdue</span><strong>{overdueCount}</strong><small>{overdueCount?`${overdueCount} sale${overdueCount===1?' has':'s have'} passed the payment day`:'Nothing is late'}</small><a href="/buyer/obligations">Review balances →</a></article>
				<article><span>Sales waiting for you</span><strong>{pending.length}</strong><small>{pending.length?'Read the details before you agree':'No sale is waiting for you'}</small><a href="/buyer/requests">Read them →</a></article>
			</section>
			<section class="next-actions">
				<div><p class="eyebrow">What needs you</p><h2>{pending.length ? 'Read the sale before you say yes.' : overdueCount ? 'A payment day has passed.' : outstanding === null ? 'We could not confirm your total.' : outstanding > 0n ? 'Everything is up to date.' : 'You owe nothing right now.'}</h2></div>
				<div class="action-copy">{#if pending.length}<p>A seller has sent you a sale. Read what they wrote: the goods, the money, the day. Nobody can force you into it. You decide.</p><a href="/buyer/requests">Read the sale →</a>{:else if overdueCount}<p>Open your balances and see what is late, what you have already paid and what is still left.</p><a href="/buyer/obligations">See what is late →</a>{:else if outstanding === null}<p>We could not confirm one or more of your balances. Open each sale instead of trusting the total above.</p><a href="/buyer/obligations">Check my balances →</a>{:else if outstanding > 0n}<p>You can open your balances any time and see what you agreed to, what you have paid and what is left.</p><a href="/buyer/obligations">See my balances →</a>{:else}<p>If a seller sends you a new sale, it will show up right here.</p><a href="/buyer/history">See how I have paid before →</a>{/if}</div>
			</section>
			<section class="trust" aria-label="Your controls"><div><span>01</span><strong>You read it before you agree.</strong><p>The seller, the goods, the money and the payment day are all in front of you first.</p></div><div><span>02</span><strong>Bank debit is a separate thing.</strong><p>You can see the most we could ever take, and you can stop that permission whenever you like.</p></div><div><span>03</span><strong>You can always report a problem.</strong><p>If the goods or the amount are wrong, say so from the sale itself, and it stays on record.</p></div></section>
			<section class="account-facts"><article><span>Your business check</span><strong>{productLabel(portal.business.status)}</strong></article><article><span>Checks done</span><strong>{portal.verification_cases.length}</strong></article><article><span>Permissions you gave</span><strong>{portal.consents.length}</strong></article></section>
		{/if}
		<FeedbackPrompt area="buyer" />
	{:else if error}
		<p class="eyebrow">Your Kredit account</p>
		<h1>Open your private link.</h1>
		<p class="error" role="alert">{error}</p>
		<p class="help">Look for the link in the message your seller sent you. If it will not open, just ask them for a fresh one. And never give your sign-in code to anybody, not even to somebody who says they are from Kredit.</p>
	{:else}
		<p class="eyebrow">Your Kredit account</p>
		<h1>Opening your account…</h1>
	{/if}
</main>

<style>
	.buyer-home{padding-bottom:6rem}.eyebrow{color:#2738d6;font-weight:800;text-transform:uppercase;letter-spacing:.1em;font-size:.72rem}.buyer-head{display:flex;justify-content:space-between;align-items:end;gap:2rem;padding:3rem 0 2rem;border-bottom:3px solid #17181b}.buyer-head h1,.buyer-home>h1{max-width:13ch;margin:.5rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(3rem,7vw,5.8rem);font-weight:500;line-height:.92;letter-spacing:-.06em}.buyer-head p{color:#656862}.balance-board{display:grid;grid-template-columns:1.25fr repeat(3,1fr);margin:2rem 0;border-top:1px solid #cfc9be;border-left:1px solid #cfc9be}.balance-board article{display:grid;align-content:start;min-height:12rem;padding:1.2rem;border-right:1px solid #cfc9be;border-bottom:1px solid #cfc9be;background:#fffdf8}.balance-board .main-balance{color:#fff;background:#2738d6}.balance-board span{font-size:.7rem;font-weight:800;letter-spacing:.08em;text-transform:uppercase}.balance-board strong{margin:.55rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(2rem,4vw,3rem);font-weight:500}.balance-board small{color:#686b66;line-height:1.5}.balance-board .main-balance small{color:#d8dbff}.balance-board a{align-self:end;margin-top:1.2rem;color:#2738d6;font-size:.78rem;font-weight:800;text-decoration:none}.balance-board .main-balance a{color:#fff}.balance-board article.danger{border-top:4px solid #b42318}.next-actions{display:grid;grid-template-columns:1fr 1fr;gap:clamp(2rem,8vw,8rem);margin:3rem 0;padding:2rem 0;border-top:1px solid #cfc9be;border-bottom:1px solid #cfc9be}.next-actions h2{max-width:12ch;margin:.5rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(2.2rem,4vw,3.8rem);font-weight:500;line-height:.96;letter-spacing:-.04em}.action-copy{padding-top:.5rem}.action-copy p{max-width:36rem;color:#626762;line-height:1.7}.action-copy a{color:#2738d6;font-weight:800}.trust{display:grid;grid-template-columns:repeat(3,1fr);margin:2rem 0;border-top:1px solid #cfc9be;border-left:1px solid #cfc9be}.trust>div{padding:1.3rem;border-right:1px solid #cfc9be;border-bottom:1px solid #cfc9be;background:#f7f3ea}.trust span{color:#e85f3d;font-size:.68rem;font-weight:850}.trust strong{display:block;margin:1.2rem 0 .5rem;font-family:Georgia,'Times New Roman',serif;font-size:1.15rem}.trust p{margin:0;color:#686b66;font-size:.86rem;line-height:1.6}.account-facts{display:grid;grid-template-columns:repeat(3,1fr);gap:1rem;margin-top:2rem}.account-facts article{display:grid;gap:.55rem;padding:1rem;border:1px solid var(--color-border);background:var(--color-surface)}.account-facts span{color:var(--color-muted);font-size:.78rem}.error{color:#b42318}.help{max-width:34rem;color:var(--color-muted);line-height:1.65}@media(max-width:900px){.balance-board{grid-template-columns:1fr 1fr}.balance-board .main-balance{grid-column:1/-1}}@media(max-width:720px){.buyer-head{display:block}.buyer-head .primary{margin-top:1rem}.balance-board,.next-actions,.trust,.account-facts{grid-template-columns:1fr}.balance-board .main-balance{grid-column:auto}.balance-board article{min-height:auto}.next-actions{gap:1rem}.trust>div{min-height:auto}}
</style>
