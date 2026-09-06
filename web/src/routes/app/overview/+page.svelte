<script lang="ts">
	import { onMount } from 'svelte';
	import { csrfHeaders } from '$lib/api/client';
	import Money from '$lib/components/Money.svelte';
	import FeedbackPrompt from '$lib/components/FeedbackPrompt.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import { productLabel } from '$lib/product-language';
	let organizations: any[] = $state([]), requests: any[] = $state([]), payments: any[] = $state([]), overdue: any[] = $state([]), claims: any[] = $state([]), disputes: any[] = $state([]), organizationID = $state(''), loading = $state(true), error = $state('');
	let receivables: { obligation_count:number; outstanding_kobo:number; overdue_kobo:number; voluntary_paid_kobo:number; collected_paid_kobo:number } | null = $state(null);
	let legalName = $state(''), tradingName = $state(''), businessType = $state('unregistered_business'), address = $state(''), industry = $state('');
	const money = (value = 0) => new Intl.NumberFormat('en-NG', { style: 'currency', currency: 'NGN', maximumFractionDigits: 0 }).format(value / 100);
	async function loadRequests() {
		if (!organizationID) { requests = []; payments = []; overdue = []; claims = []; disputes = []; receivables = null; return; }
		const endpoints = ['credit-requests', 'payments', 'overdue', 'payment-claims', 'disputes'];
		const responses = await Promise.all([
			...endpoints.map((name) => fetch(`/api/v1/organizations/${organizationID}/${name}`, { credentials: 'include' })),
			fetch(`/api/v1/organizations/${organizationID}/reports/receivables`, { credentials: 'include' })
		]);
		if (!responses[0].ok) { error = 'We could not open your sales. Please try again.'; return; }
		requests = (await responses[0].json()).requests ?? [];
		payments = responses[1].ok ? ((await responses[1].json()).payments ?? []) : [];
		overdue = responses[2].ok ? ((await responses[2].json()).overdue ?? []) : [];
		claims = responses[3].ok ? ((await responses[3].json()).payment_claims ?? []) : [];
		disputes = responses[4].ok ? ((await responses[4].json()).disputes ?? []) : [];
		receivables = responses[5].ok ? ((await responses[5].json()).summary ?? null) : null;
	}
	async function load() {
		loading = true; error = '';
		const response = await fetch('/api/v1/organizations', { credentials: 'include' });
		if (response.status === 401) { location.assign('/app'); return; }
		if (!response.ok) { error = 'We could not open your business account. Please try again.'; loading = false; return; }
		organizations = (await response.json()).organizations ?? [];
		organizationID = organizations[0]?.id ?? '';
		await loadRequests(); loading = false;
	}
	async function createOrganization() {
		error = '';
		const response = await fetch('/api/v1/organizations', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json', ...csrfHeaders() }, body: JSON.stringify({ legal_name: legalName, trading_name: tradingName, business_type: businessType, business_address: address, industry, timezone: 'Africa/Lagos', currency: 'NGN' }) });
		const body = await response.json().catch(() => ({}));
		if (!response.ok) { error = body.detail ?? 'We could not add your business. Check the details and try again.'; return; }
		await load();
	}

	const checklist = $derived.by(() => {
		const progressed = requests.some((view) => view.request && view.request.state !== 'DRAFT');
		return [
			{ done: organizations.length > 0, label: 'Add your business', href: undefined as string | undefined },
			{ done: requests.length > 0, label: 'Write down your first sale', href: '/app/credit/new' },
			{ done: progressed, label: 'Send it to your customer', href: requests.length ? `/app/credit/${requests[0]?.request?.id}?organization=${organizationID}` : '/app/credit/new' },
			{ done: payments.length > 0, label: 'Enter your first payment', href: '/app/payments' }
		];
	});
	const remainingSteps = $derived(checklist.filter((item) => !item.done).length);
	const nextStep = $derived(checklist.find((item) => !item.done));
	const dueSoon = $derived.by(() => {
		const now = new Date(); now.setHours(0,0,0,0);
		const end = new Date(now); end.setDate(end.getDate() + 7);
		return requests.filter((view) => {
			const state = String(view.request?.state ?? '').toUpperCase();
			if (['COMPLETED','PAID','CANCELLED','REJECTED','EXPIRED','DRAFT'].includes(state)) return false;
			if (!view.request?.due_date) return false;
			const due = new Date(`${view.request.due_date}T12:00:00`);
			return due >= now && due <= end;
		}).length;
	});
	const paymentsThisMonth = $derived.by(() => {
		const now = new Date();
		return payments.filter((payment) => {
			const state = String(payment.state ?? payment.status ?? '').toLowerCase();
			if (!['recognized','confirmed','paid'].includes(state)) return false;
			const when = new Date(payment.paid_at ?? payment.created_at ?? 0);
			return when.getFullYear() === now.getFullYear() && when.getMonth() === now.getMonth();
		}).reduce((sum, payment) => sum + Number(payment.amount_kobo ?? 0), 0);
	});
	const attention = $derived.by(() => {
		const items: { key:string; tone:string; title:string; detail:string; href:string; action:string }[] = [];
		for (const claim of claims.filter((item) => item.state === 'pending')) items.push({ key:`claim-${claim.id}`, tone:'money', title:'Somebody says they paid you', detail:`A customer says they have paid you ${new Intl.NumberFormat('en-NG',{style:'currency',currency:'NGN'}).format(Number(claim.amount_kobo||0)/100)}.`, href:'/app/payments', action:'Check it' });
		for (const item of overdue) items.push({ key:`late-${item.id}`, tone:'late', title:`${item.buyer_legal_name} has passed their payment day`, detail:`${item.description} · ${new Intl.NumberFormat('en-NG',{style:'currency',currency:'NGN'}).format(Number(item.amount_kobo||item.outstanding_kobo||0)/100)} is still unpaid.`, href:`/app/credit/${item.id}?organization=${organizationID}`, action:'Open the sale' });
		for (const view of requests) {
			const state=view.request?.state;
			if(state==='DRAFT')items.push({key:`draft-${view.request.id}`,tone:'normal',title:'You never finished this sale',detail:`${view.request.buyer_legal_name} has not even seen it yet.`,href:`/app/credit/${view.request.id}?organization=${organizationID}`,action:'Finish and send'});
			if(state==='SENT'||state==='BUYER_REVIEWING')items.push({key:`wait-${view.request.id}`,tone:'normal',title:'Waiting on your customer',detail:`${view.request.buyer_legal_name} still has to read this and accept it.`,href:`/app/credit/${view.request.id}?organization=${organizationID}`,action:'Open the sale'});
			if(state==='READY_TO_RELEASE')items.push({key:`goods-${view.request.id}`,tone:'goods',title:'Accepted — you can send the goods',detail:`${view.request.buyer_legal_name} has agreed to this sale.`,href:`/app/credit/${view.request.id}?organization=${organizationID}`,action:'Mark goods sent'});
		}
		for(const item of disputes.filter((entry)=>entry.state==='OPEN'||entry.state==='UNDER_REVIEW'))items.push({key:`problem-${item.id}`,tone:'problem',title:'A customer reported a problem',detail:item.reason||'Open it to see what happened and what to do next.',href:`/app/disputes/${item.id}?organization=${organizationID}`,action:'Open the problem'});
		return items.slice(0,8);
	});
	onMount(load);
</script>
<svelte:head><title>Dashboard — Kredit</title></svelte:head>
<main class="shell workspace">
	<header class="heading"><div><p class="eyebrow">Your business</p><h1>Here is where your money is.</h1><p class="lede">What you are owed, which payments are coming, what has landed and anything that needs you today.</p></div>{#if organizations.length}<a class="primary" href="/app/credit/quick">Add a sale</a>{/if}</header>
	{#if error}<p class="error" role="alert">{error}</p>{/if}
	{#if loading}<Skeleton rows={4} tall />
	{:else if !organizations.length}
		<section class="card onboarding"><p class="eyebrow">Start here</p><h2>Add your business</h2><p>You do not need a CAC registration to start. Later on, before money can move, we may come back and ask you for a few more details.</p><form onsubmit={(event) => { event.preventDefault(); createOrganization(); }}><label>Your name, or your registered business name<input bind:value={legalName} required /></label><label>The name people know you by <small>if different</small><input bind:value={tradingName} /></label><label>Business type<select bind:value={businessType}><option value="unregistered_business">Not registered yet</option><option value="registered_business">Business name registered with CAC</option><option value="sole_proprietor">Sole proprietor</option><option value="limited_company">Limited company</option><option value="partnership">Partnership</option></select></label><label>What do you sell?<input bind:value={industry} placeholder="For example: food, medicine or building materials" required /></label><label class="wide">Where is your business?<textarea bind:value={address} placeholder="Shop number, street, area, town and state" required></textarea></label><button class="primary wide">Add my business</button></form></section>
	{:else}
		<div class="toolbar"><label>Business<select bind:value={organizationID} onchange={loadRequests}>{#each organizations as org}<option value={org.id}>{org.trading_name || org.legal_name}</option>{/each}</select></label><button onclick={loadRequests}>Refresh</button></div>
		<section class="money-hero" aria-label="What you are owed">
			<div class="owed"><span>You are owed</span><strong>{receivables ? money(receivables.outstanding_kobo) : '—'}</strong><small>{receivables ? `Across ${receivables.obligation_count} sale${receivables.obligation_count===1?'':'s'}` : 'Open reports for the full picture'}</small></div>
			{#if receivables && receivables.overdue_kobo > 0}<a class="late-strip" href="/app/overdue"><span>{overdue.length || 'Some'} customer{overdue.length===1?'':'s'} late</span><em>{money(receivables.overdue_kobo)} →</em></a>{:else}<p class="ok-strip">Nobody is late right now.</p>{/if}
		</section>
		{#if remainingSteps > 0}
			<a class="setup-line" href={nextStep?.href ?? '/app/onboarding'}><strong>Finish setting up</strong><span>{checklist.length - remainingSteps} of {checklist.length} done · next: {nextStep?.label}</span><i aria-hidden="true">→</i></a>
		{/if}
		<section class="today" aria-labelledby="today-title"><header><div><p class="eyebrow">Needs your attention</p><h2 id="today-title">{attention.length ? `${attention.length} ${attention.length===1?'thing':'things'} need you` : 'Nothing needs you right now'}</h2></div><a href="/app/credit/quick">Add a sale →</a></header>{#if attention.length}<div class="attention-list">{#each attention.slice(0,3) as item}<article class={item.tone}><div><strong>{item.title}</strong><p>{item.detail}</p></div><a href={item.href}>{item.action} →</a></article>{/each}</div>{#if attention.length > 3}<details class="more-attention"><summary>{attention.length - 3} more thing{attention.length-3===1?'':'s'} need you</summary><div class="attention-list">{#each attention.slice(3) as item}<article class={item.tone}><div><strong>{item.title}</strong><p>{item.detail}</p></div><a href={item.href}>{item.action} →</a></article>{/each}</div></details>{/if}{:else}<p>Nothing needs you right now. Late payments, customer answers and payments to check will land here the moment they come up.</p>{/if}</section>
		{#if !requests.length}<section class="empty-state"><h2>No sales yet</h2><p>Next time somebody takes goods and promises to pay you later, write it down here before the goods leave your shop.</p><a class="primary" href="/app/credit/quick">Add my first sale</a></section>{/if}
		<FeedbackPrompt area="seller" {organizationID} />
	{/if}
</main>
<style>.money-hero{margin:1.5rem 0}.money-hero .owed{background:#17181b;color:#fff;padding:1.6rem 1.4rem 1.5rem}.money-hero .owed span{display:block;font-size:.7rem;font-weight:750;letter-spacing:.1em;text-transform:uppercase;color:#b9b8b3}.money-hero .owed strong{display:block;font-family:Georgia,serif;font-size:clamp(2.6rem,7vw,4rem);font-weight:500;line-height:1;margin:.5rem 0 .4rem;letter-spacing:-.03em;font-variant-numeric:tabular-nums}.money-hero .owed small{color:#b9b8b3;font-size:.82rem}.late-strip{display:flex;align-items:center;justify-content:space-between;gap:1rem;padding:.85rem 1.4rem;background:#ec6a47;color:#fff;font-weight:750;font-size:.9rem;text-decoration:none}.late-strip em{font-style:normal;font-variant-numeric:tabular-nums}.ok-strip{margin:0;padding:.85rem 1.4rem;background:#eef0ff;color:#2738d6;font-weight:750;font-size:.9rem}.setup-line{display:flex;align-items:center;gap:.9rem;padding:.9rem 1.1rem;margin-bottom:1.5rem;border:1px solid var(--color-border);background:var(--color-surface,#fffdfa);text-decoration:none;font-size:.9rem}.setup-line span{color:var(--color-muted);font-size:.82rem}.setup-line i{margin-left:auto;font-style:normal;color:#2738d6;font-weight:800}.more-attention{margin-top:.75rem}.more-attention>summary{cursor:pointer;min-height:2.75rem;display:flex;align-items:center;font-weight:750;font-size:.86rem;color:#2738d6}.heading{display:flex;justify-content:space-between;align-items:end;gap:3rem;padding-bottom:2.2rem;border-bottom:3px solid #17181b}.heading h1{max-width:13ch;font-family:Georgia,'Times New Roman',serif;font-size:clamp(3.2rem,7vw,6rem);font-weight:500;line-height:.9;letter-spacing:-.06em;margin:.6rem 0}.onboarding{max-width:52rem;margin:2rem auto;padding:clamp(1.4rem,4vw,2.5rem);box-shadow:10px 10px 0 #2738d6}.onboarding form{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1rem}.onboarding label{display:grid;gap:.45rem;font-weight:700}.onboarding input,.onboarding select,.onboarding textarea,.toolbar select{box-sizing:border-box;width:100%;padding:.8rem;border:1px solid #aaa69e;border-radius:0;background:#fff;font:inherit}.wide{grid-column:1/-1}.toolbar{display:flex;gap:1rem;align-items:end;margin:2rem 0 1rem;padding:1rem;background:#e8e3d9}.toolbar label{display:grid;gap:.35rem}.setup-copy{max-width:46rem;color:#b6b6b1;line-height:1.6}.today{margin:1.5rem 0;border:1px solid var(--color-border);background:var(--color-surface)}.today>header{display:flex;justify-content:space-between;align-items:end;gap:1rem;padding:1.25rem;border-bottom:3px solid #17181b}.today h2{margin:.2rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(1.8rem,4vw,2.6rem);font-weight:500}.attention-list{display:grid}.attention-list article{display:grid;grid-template-columns:1fr auto;align-items:center;gap:1rem;padding:1rem 1.25rem;border-bottom:1px solid var(--color-border);border-left:5px solid #2738d6}.attention-list article.late,.attention-list article.problem{border-left-color:#b42318}.attention-list article.money{border-left-color:#16794e}.attention-list article.goods{border-left-color:#a15c00}.attention-list p{margin:.25rem 0;color:var(--color-muted)}.today>p{padding:1.25rem}.empty-state{margin-top:2rem;padding:clamp(2rem,5vw,4rem);color:#fff;background:#2738d6}.empty-state p{color:#d4d7ff}.empty-state .primary{color:#17181b;background:#fff}@media(max-width:900px){}@media(max-width:720px){.heading{display:block}.heading .primary{margin-top:1rem}.onboarding form{grid-template-columns:1fr}.wide{grid-column:auto}.onboarding{box-shadow:6px 6px 0 #2738d6}.toolbar{align-items:stretch;flex-direction:column}.today>header{align-items:start;flex-direction:column}.attention-list article{grid-template-columns:1fr}.attention-list article a{width:max-content}}</style>
