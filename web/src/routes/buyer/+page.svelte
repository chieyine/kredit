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
			<section class="owe-hero" aria-label="What you owe">
				<div class="owed"><span>You owe</span><strong>{formatKobo(outstanding)}</strong><small>Across {openBalances.length} sale{openBalances.length===1?'':'s'}</small></div>
				{#if overdueCount}<a class="due-strip late" href="/buyer/obligations"><span>{overdueCount} payment day{overdueCount===1?' has':'s have'} passed</span><em>See what is late →</em></a>{:else if nextPayment}<a class="due-strip" href="/buyer/obligations"><span>Next: {formatKobo(nextPayment.item.obligation?.outstanding_kobo??0)}</span><em>by {new Date(nextPayment.date).toLocaleDateString('en-NG',{day:'numeric',month:'short'})} →</em></a>{:else}<p class="due-strip calm">Nothing is due right now.</p>{/if}
			</section>
			<section class="next-actions">
				<div><p class="eyebrow">What needs you</p><h2>{pending.length ? 'Read the sale before you say yes.' : overdueCount ? 'A payment day has passed.' : outstanding === null ? 'We could not confirm your total.' : outstanding > 0n ? 'Everything is up to date.' : 'You owe nothing right now.'}</h2></div>
				<div class="action-copy">{#if pending.length}<p>A seller has sent you a sale. Read what they wrote: the goods, the money, the day. Nobody can force you into it. You decide.</p><a href="/buyer/requests">Read the sale →</a>{:else if overdueCount}<p>Open your balances and see what is late, what you have already paid and what is still left.</p><a href="/buyer/obligations">See what is late →</a>{:else if outstanding === null}<p>We could not confirm one or more of your balances. Open each sale instead of trusting the total above.</p><a href="/buyer/obligations">Check my balances →</a>{:else if outstanding > 0n}<p>You can open your balances any time and see what you agreed to, what you have paid and what is left.</p><a href="/buyer/obligations">See my balances →</a>{:else}<p>If a seller sends you a new sale, it will show up right here.</p><a href="/buyer/history">See how I have paid before →</a>{/if}</div>
			</section>
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

<style>.owe-hero{margin:1.5rem 0}.owe-hero .owed{background:#17181b;color:#fff;padding:1.6rem 1.4rem 1.5rem}.owe-hero .owed span{display:block;font-size:.7rem;font-weight:800;letter-spacing:.1em;text-transform:uppercase;color:#b9b8b3}.owe-hero .owed strong{display:block;font-family:Georgia,'Times New Roman',serif;font-size:clamp(2.6rem,7vw,4rem);font-weight:500;line-height:1;margin:.5rem 0 .4rem;letter-spacing:-.03em;font-variant-numeric:tabular-nums}.owe-hero .owed small{color:#b9b8b3;font-size:.82rem}.due-strip{display:flex;align-items:center;justify-content:space-between;gap:1rem;margin:0;padding:.85rem 1.4rem;background:#eef0ff;color:#2738d6;font-weight:750;font-size:.9rem;text-decoration:none}.due-strip em{font-style:normal;font-variant-numeric:tabular-nums}.due-strip.late{background:#ec6a47;color:#fff}.due-strip.calm{color:#5f645f;background:#e8e3d9}.buyer-home{padding-bottom:6rem}.eyebrow{color:#2738d6;font-weight:800;text-transform:uppercase;letter-spacing:.1em;font-size:.72rem}.buyer-head{display:flex;justify-content:space-between;align-items:end;gap:2rem;padding:3rem 0 2rem;border-bottom:3px solid #17181b}.buyer-head h1,.buyer-home>h1{max-width:13ch;margin:.5rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(3rem,7vw,5.8rem);font-weight:500;line-height:.92;letter-spacing:-.06em}.buyer-head p{color:#656862}.next-actions{display:grid;grid-template-columns:1fr 1fr;gap:clamp(2rem,8vw,8rem);margin:3rem 0;padding:2rem 0;border-top:1px solid #cfc9be;border-bottom:1px solid #cfc9be}.next-actions h2{max-width:12ch;margin:.5rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(2.2rem,4vw,3.8rem);font-weight:500;line-height:.96;letter-spacing:-.04em}.action-copy{padding-top:.5rem}.action-copy p{max-width:36rem;color:#626762;line-height:1.7}.action-copy a{color:#2738d6;font-weight:800}.error{color:#b42318}.help{max-width:34rem;color:var(--color-muted);line-height:1.65}@media(max-width:900px){}@media(max-width:720px){.buyer-head{display:block}.buyer-head .primary{margin-top:1rem}.next-actions{grid-template-columns:1fr}.next-actions{gap:1rem}}
</style>
