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
		if (!responses[0].ok) { error = 'We could not load your sales. Please try again.'; return; }
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
		if (!response.ok) { error = 'We could not load your business account. Please try again.'; loading = false; return; }
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
			{ done: requests.length > 0, label: 'Create your first credit sale', href: '/app/credit/new' },
			{ done: progressed, label: 'Send the sale for customer approval', href: requests.length ? `/app/credit/${requests[0]?.request?.id}?organization=${organizationID}` : '/app/credit/new' },
			{ done: payments.length > 0, label: 'Record your first payment', href: '/app/payments' }
		];
	});
	const remainingSteps = $derived(checklist.filter((item) => !item.done).length);
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
		for (const claim of claims.filter((item) => item.state === 'pending')) items.push({ key:`claim-${claim.id}`, tone:'money', title:'A payment needs confirmation', detail:`A customer reported paying ${new Intl.NumberFormat('en-NG',{style:'currency',currency:'NGN'}).format(Number(claim.amount_kobo||0)/100)}.`, href:'/app/payments', action:'Review payment' });
		for (const item of overdue) items.push({ key:`late-${item.id}`, tone:'late', title:`Payment from ${item.buyer_legal_name} is overdue`, detail:`${item.description} · ${new Intl.NumberFormat('en-NG',{style:'currency',currency:'NGN'}).format(Number(item.amount_kobo||item.outstanding_kobo||0)/100)} remains unpaid.`, href:`/app/credit/${item.id}?organization=${organizationID}`, action:'View sale' });
		for (const view of requests) {
			const state=view.request?.state;
			if(state==='DRAFT')items.push({key:`draft-${view.request.id}`,tone:'normal',title:'Finish this credit sale',detail:`${view.request.buyer_legal_name} has not received it yet.`,href:`/app/credit/${view.request.id}?organization=${organizationID}`,action:'Finish and send'});
			if(state==='SENT'||state==='BUYER_REVIEWING')items.push({key:`wait-${view.request.id}`,tone:'normal',title:'Waiting for customer approval',detail:`${view.request.buyer_legal_name} still needs to review and accept this sale.`,href:`/app/credit/${view.request.id}?organization=${organizationID}`,action:'View sale'});
			if(state==='READY_TO_RELEASE')items.push({key:`goods-${view.request.id}`,tone:'goods',title:'Customer accepted — goods can be released',detail:`${view.request.buyer_legal_name} has accepted the sale terms.`,href:`/app/credit/${view.request.id}?organization=${organizationID}`,action:'Record delivery'});
		}
		for(const item of disputes.filter((entry)=>entry.state==='OPEN'||entry.state==='UNDER_REVIEW'))items.push({key:`problem-${item.id}`,tone:'problem',title:'A dispute needs your attention',detail:item.reason||'Open the dispute to see what happened and what you need to do next.',href:`/app/disputes/${item.id}?organization=${organizationID}`,action:'Review dispute'});
		return items.slice(0,8);
	});
	onMount(load);
</script>
<svelte:head><title>Seller dashboard — Kredit</title></svelte:head>
<main class="shell workspace">
	<header class="heading"><div><p class="eyebrow">Your business</p><h1>Know what is owed.<br />Know what is next.</h1><p class="lede">Your daily view of outstanding balances, upcoming payment dates, money received and anything that needs your attention.</p></div>{#if organizations.length}<a class="primary" href="/app/credit/new">Add credit sale</a>{/if}</header>
	{#if error}<p class="error" role="alert">{error}</p>{/if}
	{#if loading}<Skeleton rows={4} tall />
	{:else if !organizations.length}
		<section class="card onboarding"><p class="eyebrow">Start here</p><h2>Add your business</h2><p>You can add your business even if it is not yet registered. Before money can move, Kredit may ask for additional verification.</p><form onsubmit={(event) => { event.preventDefault(); createOrganization(); }}><label>Your name or registered business name<input bind:value={legalName} required /></label><label>Trading or shop name <small>if different</small><input bind:value={tradingName} /></label><label>Business type<select bind:value={businessType}><option value="unregistered_business">Not registered yet</option><option value="registered_business">Business name registered with CAC</option><option value="sole_proprietor">Sole proprietor</option><option value="limited_company">Limited company</option><option value="partnership">Partnership</option></select></label><label>What does your business sell?<input bind:value={industry} placeholder="For example: food, medicine or building materials" required /></label><label class="wide">Business address<textarea bind:value={address} placeholder="Shop number, street, area, town and state" required></textarea></label><button class="primary wide">Add business</button></form></section>
	{:else}
		<div class="toolbar"><label>Business<select bind:value={organizationID} onchange={loadRequests}>{#each organizations as org}<option value={org.id}>{org.trading_name || org.legal_name}</option>{/each}</select></label><button onclick={loadRequests}>Refresh</button></div>
		<section class="money-board" aria-label="Business money summary">
			<article class="hero-stat"><span>Outstanding</span><strong>{receivables ? money(receivables.outstanding_kobo) : '—'}</strong><small>{receivables ? `${receivables.obligation_count} active balance${receivables.obligation_count===1?'':'s'}` : 'Open reports for full totals'}</small><a href="/app/collections">See money owed →</a></article>
			<article><span>Overdue</span><strong>{receivables ? money(receivables.overdue_kobo) : '—'}</strong><small>{overdue.length} payment{overdue.length===1?'':'s'} need follow-up</small><a href="/app/overdue">Review overdue →</a></article>
			<article><span>Due in 7 days</span><strong>{dueSoon}</strong><small>Upcoming customer payment dates</small><a href="/app/collections">Plan follow-up →</a></article>
			<article><span>Received this month</span><strong>{money(paymentsThisMonth)}</strong><small>Payments recorded in Kredit</small><a href="/app/payments">See payments →</a></article>
		</section>
		{#if remainingSteps > 0}
			<section class="card setup" aria-label="Getting started">
				<h2>Get to your first completed sale <span class="count">{checklist.length - remainingSteps}/{checklist.length}</span></h2>
				<p class="setup-copy">You only need the basics to start. Finish the next incomplete step and Kredit will guide you from there.</p>
				<ol>
					{#each checklist as item}
						<li class:done={item.done}>
							<span class="tick" aria-hidden="true">{item.done ? '✓' : ''}</span>
							{#if item.done}<s>{item.label}</s>{:else if item.href}<a href={item.href}>{item.label} →</a>{:else}<span>{item.label}</span>{/if}
						</li>
					{/each}
				</ol>
			</section>
		{/if}
		<section class="today" aria-labelledby="today-title"><header><div><p class="eyebrow">Needs your attention</p><h2 id="today-title">{attention.length ? `${attention.length} ${attention.length===1?'item':'items'} to review` : 'You are all caught up'}</h2></div><a href="/app/credit/new">Add credit sale →</a></header>{#if attention.length}<div class="attention-list">{#each attention as item}<article class={item.tone}><div><strong>{item.title}</strong><p>{item.detail}</p></div><a href={item.href}>{item.action} →</a></article>{/each}</div>{:else}<p>No action is needed right now. Late payments, customer responses and payment checks will appear here when they need you.</p>{/if}</section>
		{#if requests.length}<section class="recent-head"><div><p class="eyebrow">Recent credit sales</p><h2>Pick up where you left off.</h2></div><a href="/app/collections">See all balances →</a></section><section class="records">{#each requests.slice(0,6) as view}<article><div><strong>{view.request?.buyer_legal_name ?? 'Customer'}</strong><span>{productLabel(view.request?.state)}</span></div><p>{view.request?.goods_description}</p><p><strong><Money amountKobo={view.request?.principal_kobo ?? 0} /></strong> · due {view.request?.due_date}</p><a href={`/app/credit/${view.request?.id}?organization=${organizationID}`}>View sale →</a></article>{/each}</section>{:else}<section class="empty-state"><h2>No credit sales yet</h2><p>When a customer takes goods now and will pay later, record the sale before the goods leave.</p><a class="primary" href="/app/credit/new">Create first credit sale</a></section>{/if}
		<FeedbackPrompt area="seller" {organizationID} />
	{/if}
</main>
<style>.heading{display:flex;justify-content:space-between;align-items:end;gap:3rem;padding-bottom:2.2rem;border-bottom:3px solid #17181b}.heading h1{max-width:13ch;font-family:Georgia,'Times New Roman',serif;font-size:clamp(3.2rem,7vw,6rem);font-weight:500;line-height:.9;letter-spacing:-.06em;margin:.6rem 0}.onboarding{max-width:52rem;margin:2rem auto;padding:clamp(1.4rem,4vw,2.5rem);box-shadow:10px 10px 0 #2738d6}.onboarding form,.records{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1rem}.onboarding label{display:grid;gap:.45rem;font-weight:700}.onboarding input,.onboarding select,.onboarding textarea,.toolbar select{box-sizing:border-box;width:100%;padding:.8rem;border:1px solid #aaa69e;border-radius:0;background:#fff;font:inherit}.wide{grid-column:1/-1}.toolbar{display:flex;gap:1rem;align-items:end;margin:2rem 0 1rem;padding:1rem;background:#e8e3d9}.toolbar label{display:grid;gap:.35rem}.money-board{display:grid;grid-template-columns:1.25fr repeat(3,1fr);margin:1rem 0 2rem;border-top:1px solid #cfc9be;border-left:1px solid #cfc9be}.money-board article{display:grid;align-content:start;min-height:12rem;padding:1.2rem;border-right:1px solid #cfc9be;border-bottom:1px solid #cfc9be;background:#fffdf8}.money-board .hero-stat{color:#fff;background:#2738d6}.money-board span{font-size:.72rem;font-weight:800;letter-spacing:.08em;text-transform:uppercase}.money-board strong{display:block;margin:.55rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(1.8rem,3vw,2.8rem);font-weight:500;letter-spacing:-.04em}.money-board small{color:#6b6d68;line-height:1.5}.money-board .hero-stat small{color:#d8dbff}.money-board a{align-self:end;margin-top:1.25rem;color:#2738d6;font-size:.78rem;font-weight:780;text-decoration:none}.money-board .hero-stat a{color:#fff}.setup{margin-top:2rem;padding:1.5rem;color:#fff;background:#17181b}.setup h2{display:flex;align-items:center;gap:.75rem;margin:0;font-family:Georgia,'Times New Roman',serif;font-size:1.5rem;font-weight:500}.setup-copy{max-width:46rem;color:#b6b6b1;line-height:1.6}.setup .count{padding:.2rem .6rem;background:#2738d6;color:#fff;font-family:Inter,sans-serif;font-size:.8rem}.setup ol{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:0;margin:1rem 0 0;padding:0;border-top:1px solid #46474c;border-left:1px solid #46474c;list-style:none}.setup li{display:flex;gap:.6rem;align-items:start;padding:1rem;border-right:1px solid #46474c;border-bottom:1px solid #46474c;color:#fff;font-weight:650}.setup li.done{color:#92938f}.setup .tick{flex:0 0 auto;display:grid;place-items:center;width:1.4rem;height:1.4rem;margin-top:.1rem;border:2px solid #67686d;border-radius:50%;font-size:.75rem;font-weight:800}.setup li.done .tick{border-color:#e85f3d;background:#e85f3d;color:#fff}.setup a{color:#aeb4ff;text-decoration:none}.today{margin:1.5rem 0;border:1px solid var(--color-border);background:var(--color-surface)}.today>header{display:flex;justify-content:space-between;align-items:end;gap:1rem;padding:1.25rem;border-bottom:3px solid #17181b}.today h2,.recent-head h2{margin:.2rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(1.8rem,4vw,2.6rem);font-weight:500}.attention-list{display:grid}.attention-list article{display:grid;grid-template-columns:1fr auto;align-items:center;gap:1rem;padding:1rem 1.25rem;border-bottom:1px solid var(--color-border);border-left:5px solid #2738d6}.attention-list article.late,.attention-list article.problem{border-left-color:#b42318}.attention-list article.money{border-left-color:#16794e}.attention-list article.goods{border-left-color:#a15c00}.attention-list p{margin:.25rem 0;color:var(--color-muted)}.today>p{padding:1.25rem}.recent-head{display:flex;justify-content:space-between;align-items:end;gap:1rem;margin-top:2.5rem;padding-bottom:.9rem;border-bottom:1px solid #cfc9be}.recent-head a{font-weight:750}.records article{min-height:10rem;padding:1.35rem;border:1px solid var(--color-border);background:#fffdf8;box-shadow:6px 6px 0 #ded8cc}.records article>div{display:flex;justify-content:space-between;gap:1rem}.records span{font-size:.8rem;color:var(--color-muted)}.empty-state{margin-top:2rem;padding:clamp(2rem,5vw,4rem);color:#fff;background:#2738d6}.empty-state p{color:#d4d7ff}.empty-state .primary{color:#17181b;background:#fff}@media(max-width:900px){.money-board{grid-template-columns:1fr 1fr}.money-board .hero-stat{grid-column:1/-1}.setup ol{grid-template-columns:1fr 1fr}}@media(max-width:720px){.heading{display:block}.heading .primary{margin-top:1rem}.onboarding form,.records,.setup ol,.money-board{grid-template-columns:1fr}.money-board .hero-stat{grid-column:auto}.money-board article{min-height:auto}.wide{grid-column:auto}.onboarding{box-shadow:6px 6px 0 #2738d6}.toolbar{align-items:stretch;flex-direction:column}.today>header,.recent-head{align-items:start;flex-direction:column}.attention-list article{grid-template-columns:1fr}.attention-list article a{width:max-content}}</style>
