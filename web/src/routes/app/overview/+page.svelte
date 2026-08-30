<script lang="ts">
	import { onMount } from 'svelte';
	import { csrfHeaders } from '$lib/api/client';
	import Money from '$lib/components/Money.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import { productLabel } from '$lib/product-language';
	let organizations: any[] = $state([]), requests: any[] = $state([]), payments: any[] = $state([]), organizationID = $state(''), loading = $state(true), error = $state('');
	let legalName = $state(''), tradingName = $state(''), businessType = $state('unregistered_business'), address = $state(''), industry = $state('');
	async function loadRequests() {
		if (!organizationID) { requests = []; payments = []; return; }
		const response = await fetch(`/api/v1/organizations/${organizationID}/credit-requests`, { credentials: 'include' });
		if (!response.ok) { error = 'We could not load your credit requests.'; return; }
		requests = (await response.json()).requests ?? [];
		const paymentResponse = await fetch(`/api/v1/organizations/${organizationID}/payments`, { credentials: 'include' });
		payments = paymentResponse.ok ? ((await paymentResponse.json()).payments ?? []) : [];
	}
	async function load() {
		loading = true; error = '';
		const response = await fetch('/api/v1/organizations', { credentials: 'include' });
		if (response.status === 401) { location.assign('/app'); return; }
		if (!response.ok) { error = 'We could not load your workspace.'; loading = false; return; }
		organizations = (await response.json()).organizations ?? [];
		organizationID = organizations[0]?.id ?? '';
		await loadRequests(); loading = false;
	}
	async function createOrganization() {
		error = '';
		const response = await fetch('/api/v1/organizations', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json', ...csrfHeaders() }, body: JSON.stringify({ legal_name: legalName, trading_name: tradingName, business_type: businessType, business_address: address, industry, timezone: 'Africa/Lagos', currency: 'NGN' }) });
		const body = await response.json().catch(() => ({}));
		if (!response.ok) { error = body.detail ?? 'The organisation could not be created.'; return; }
		await load();
	}

	const checklist = $derived.by(() => {
		const progressed = requests.some((view) => view.request && view.request.state !== 'DRAFT');
		return [
			{ done: organizations.length > 0, label: 'Add your business', href: undefined as string | undefined },
			{ done: requests.length > 0, label: 'Add your first sale', href: '/app/credit/new' },
			{ done: progressed, label: 'Send the sale to your customer', href: requests.length ? `/app/credit/${requests[0]?.request?.id}?organization=${organizationID}` : '/app/credit/new' },
			{ done: payments.length > 0, label: 'Get your first payment', href: '/app/payments' }
		];
	});
	const remainingSteps = $derived(checklist.filter((item) => !item.done).length);
	onMount(load);
</script>
<svelte:head><title>Workspace overview — Kredit</title></svelte:head>
<main class="shell workspace">
	<header class="heading"><div><p class="eyebrow">Your business</p><h1>Good day.<br />Here is your money.</h1><p class="lede">See who owes you, when they should pay and what has come in.</p></div>{#if organizations.length}<a class="primary" href="/app/credit/new">Add a sale</a>{/if}</header>
	{#if error}<p class="error" role="alert">{error}</p>{/if}
	{#if loading}<Skeleton rows={4} tall />
	{:else if !organizations.length}
		<section class="card onboarding"><p class="eyebrow">First step</p><h2>Tell us about your business</h2><p>Your business does not need to be registered before you add it. We may ask for other proof before money can move.</p><form onsubmit={(event) => { event.preventDefault(); createOrganization(); }}><label>Your name or registered business name<input bind:value={legalName} required /></label><label>Shop or business name <small>if different</small><input bind:value={tradingName} /></label><label>Business type<select bind:value={businessType}><option value="unregistered_business">Not registered yet</option><option value="registered_business">Business name registered with CAC</option><option value="sole_proprietor">One-person business</option><option value="limited_company">Limited company</option><option value="partnership">Partnership</option></select></label><label>What do you sell?<input bind:value={industry} placeholder="For example: food, medicine or building materials" required /></label><label class="wide">Where is the business?<textarea bind:value={address} placeholder="Shop number, street, area, town and state" required></textarea></label><button class="primary wide">Add my business</button></form></section>
	{:else}
		{#if remainingSteps > 0}
			<section class="card setup" aria-label="Getting started">
				<h2>Getting started <span class="count">{checklist.length - remainingSteps}/{checklist.length}</span></h2>
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
		<div class="toolbar"><label>Business<select bind:value={organizationID} onchange={loadRequests}>{#each organizations as org}<option value={org.id}>{org.trading_name || org.legal_name}</option>{/each}</select></label><button onclick={loadRequests}>Check again</button></div>
		{#if requests.length}<section class="records">{#each requests as view}<article><div><strong>{view.request?.buyer_legal_name ?? 'Customer'}</strong><span>{productLabel(view.request?.state)}</span></div><p>{view.request?.goods_description}</p><p><strong><Money amountKobo={view.request?.principal_kobo ?? 0} /></strong> · pay before {view.request?.due_date}</p><a href={`/app/credit/${view.request?.id}?organization=${organizationID}`}>Open sale →</a></article>{/each}</section>{:else}<section class="empty-state"><h2>No sales here yet</h2><p>When a customer takes goods and will pay later, add the sale here.</p><a class="primary" href="/app/credit/new">Add your first sale</a></section>{/if}
	{/if}
</main>
<style>.heading{display:flex;justify-content:space-between;align-items:end;gap:3rem;padding-bottom:2.2rem;border-bottom:3px solid #17181b}.heading h1{max-width:13ch;font-family:Georgia,'Times New Roman',serif;font-size:clamp(3.2rem,7vw,6rem);font-weight:500;line-height:.9;letter-spacing:-.06em;margin:.6rem 0}.onboarding{max-width:52rem;margin:2rem auto;padding:clamp(1.4rem,4vw,2.5rem);box-shadow:10px 10px 0 #2738d6}.onboarding form,.records{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1rem}.onboarding label{display:grid;gap:.45rem;font-weight:700}.onboarding input,.onboarding select,.onboarding textarea,.toolbar select{box-sizing:border-box;width:100%;padding:.8rem;border:1px solid #aaa69e;border-radius:0;background:#fff;font:inherit}.wide{grid-column:1/-1}.setup{margin-top:2rem;padding:1.5rem;color:#fff;background:#17181b}.setup h2{display:flex;align-items:center;gap:.75rem;margin-top:0;font-family:Georgia,'Times New Roman',serif;font-size:1.5rem;font-weight:500}.setup .count{padding:.2rem .6rem;background:#2738d6;color:#fff;font-family:Inter,sans-serif;font-size:.8rem}.setup ol{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:0;margin:0;padding:0;border-top:1px solid #46474c;border-left:1px solid #46474c;list-style:none}.setup li{display:flex;gap:.6rem;align-items:start;padding:1rem;border-right:1px solid #46474c;border-bottom:1px solid #46474c;color:#fff;font-weight:650}.setup li.done{color:#92938f}.setup .tick{flex:0 0 auto;display:grid;place-items:center;width:1.4rem;height:1.4rem;margin-top:.1rem;border:2px solid #67686d;border-radius:50%;font-size:.75rem;font-weight:800}.setup li.done .tick{border-color:#e85f3d;background:#e85f3d;color:#fff}.setup a{color:#aeb4ff;text-decoration:none}.toolbar{display:flex;gap:1rem;align-items:end;margin:2rem 0;padding:1rem;background:#e8e3d9}.toolbar label{display:grid;gap:.35rem}.records article{min-height:10rem;padding:1.35rem;border:1px solid var(--color-border);background:#fffdf8;box-shadow:6px 6px 0 #ded8cc}.records article>div{display:flex;justify-content:space-between}.records span{font-size:.8rem;color:var(--color-muted)}.empty-state{margin-top:2rem;padding:clamp(2rem,5vw,4rem);color:#fff;background:#2738d6}.empty-state p{color:#d4d7ff}.empty-state .primary{color:#17181b;background:#fff}@media(max-width:720px){.heading{display:block}.heading .primary{margin-top:1rem}.onboarding form,.records,.setup ol{grid-template-columns:1fr}.wide{grid-column:auto}.onboarding{box-shadow:6px 6px 0 #2738d6}}</style>
