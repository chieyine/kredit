<script lang="ts">
	import { chooseWorkspace, requestedWorkspace } from '$lib/workspace-context';
	import { getContext, onMount } from 'svelte';
	import { ACCOUNT_CONTEXT, type AccountContext } from '$lib/account-context';
	import { LatestRequest, readResource, record, rows, type Resource } from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';
	import {
		organization,
		saleView,
		paymentRow,
		workRow,
		receivables,
		type Organization,
		type SaleView,
		type PaymentRow,
		type WorkRow,
		type Receivables
	} from '$lib/records';
	import { attentionItems } from '$lib/attention';
	import type { PageData } from './$types';
	import { exactKobo } from '$lib/money';
	import Money from '$lib/components/Money.svelte';
	import ResourceNotice from '$lib/components/ResourceNotice.svelte';
	import BusinessNextSteps from '$lib/components/BusinessNextSteps.svelte';
	import FeedbackPrompt from '$lib/components/FeedbackPrompt.svelte';
	import { read as readConsumer, purchase, type Purchase } from '$lib/consumer';
	import { paymentDateAfter } from '$lib/sale-defaults';
	const account = getContext<AccountContext>(ACCOUNT_CONTEXT);
	let { data }: { data: PageData } = $props();
	// Capturing the initial value is correct here, not an oversight: the
	// workspace layout renders this page inside
	// {#key page.url.pathname + page.url.search}, so switching business via
	// chooseWorkspace() changes the query string and destroys and recreates
	// this component. Every mount therefore sees its own load() result.
	// svelte-ignore state_referenced_locally
	const seeded = data.prefetched;
	const fromServer = <T,>(value: T): Resource<T> =>
		seeded
			? { state: 'ready', scope: seeded.organizationID, data: value, checkedAt: seeded.checkedAt }
			: { state: 'loading', scope: '' };
	const pending = <T,>(scope = ''): Resource<T> => ({ state: 'loading', scope });
	let businesses = $state<Resource<Organization[]>>(seeded ? fromServer(seeded.businesses) : pending());
	let organizationID = $state(seeded?.organizationID ?? '');
	let sales = $state<Resource<SaleView[]>>(seeded ? fromServer(seeded.sales) : pending());
	let payments = $state<Resource<PaymentRow[]>>(seeded ? fromServer(seeded.payments) : pending());
	let overdue = $state<Resource<WorkRow[]>>(seeded ? fromServer(seeded.overdue) : pending());
	let claims = $state<Resource<WorkRow[]>>(seeded ? fromServer(seeded.claims) : pending());
	let disputes = $state<Resource<WorkRow[]>>(seeded ? fromServer(seeded.disputes) : pending());
	let due = $state<Resource<WorkRow[]>>(seeded ? fromServer(seeded.due) : pending());
	let summary = $state<Resource<Receivables>>(seeded ? fromServer(seeded.summary) : pending());
	let people = $state<Resource<Purchase[]>>(pending());
	let visibleCount = $state(5),
		legalName = $state(''),
		tradingName = $state(''),
		registrationInfo = $state(''),
		businessType = $state('unregistered_business'),
		address = $state(''),
		industry = $state(''),
		createBusy = $state(false),
		createError = $state('');
	let creation: MutationIntent | null = null;
	const businessRequest = new LatestRequest(),
		dashboardRequest = new LatestRequest();
	const organizations = $derived(businesses.state === 'ready' ? businesses.data : []);
	const currentBusiness = $derived(organizations.find((item) => item.id === organizationID));
	const businessTitle = $derived(
		currentBusiness
			? currentBusiness.trading_name || currentBusiness.legal_name || 'Business overview'
			: 'Business overview'
	);
	const allChecked = $derived(
		[sales, payments, overdue, claims, disputes, summary, due].every(
			(item) => item.state === 'ready' && item.scope === organizationID
		)
	);
	// Only what the seller must do something about; late and waiting customers show in the list below.
	// Only what the seller must do something about. Late and waiting customers
	// already show in the customer list below, so they are not repeated here.
	const attention = $derived(
		attentionItems(
			organizationID,
			sales.state === 'ready' ? sales.data : [],
			claims.state === 'ready' ? claims.data : [],
			overdue.state === 'ready' ? overdue.data : [],
			disputes.state === 'ready' ? disputes.data : [],
			due.state === 'ready' ? due.data : []
		).filter((item) => /^(claim|dispute|release|receipt|draft)-/.test(item.id))
	);
	type Owing = {
		key: string;
		name: string;
		kind: 'Business' | 'Person';
		owed: bigint;
		waiting: number;
		waitingKobo: bigint;
		late: boolean;
		href: string;
	};
	const today = paymentDateAfter(0);
	const weekAhead = paymentDateAfter(7);
	const closedStates = ['CANCELLED', 'DECLINED', 'EXPIRED', 'CLOSED', 'PAID', 'WRITTEN_OFF', 'REJECTED'];
	const owing = $derived.by(() => {
		const query = `?organization=${encodeURIComponent(organizationID)}`;
		// eslint-disable-next-line svelte/prefer-svelte-reactivity -- a local tally, never rendered directly
		const list = new Map<string, Owing>();
		const late = new Set(
			(overdue.state === 'ready' ? overdue.data : []).flatMap((row) => [row.credit_request_id, row.obligation_id])
		);
		for (const view of sales.state === 'ready' ? sales.data : []) {
			const r = view.request;
			if (closedStates.includes(r.state.toUpperCase())) continue;
			const id = r.buyer_business_id || r.buyer_user_id;
			const row = list.get(`b:${id}`) ?? {
				key: `b:${id}`,
				name: r.buyer_legal_name || 'Customer',
				kind: 'Business' as const,
				owed: 0n,
				waiting: 0,
				waitingKobo: 0n,
				late: false,
				href: r.buyer_business_id
					? `/workspace/partners/customers/${encodeURIComponent(r.buyer_business_id)}${query}`
					: `/workspace/sales/${encodeURIComponent(r.id)}${query}`
			};
			if (view.obligation) row.owed += exactKobo(view.obligation.outstanding_kobo) ?? 0n;
			else {
				row.waiting += 1;
				row.waitingKobo += exactKobo(r.principal_kobo) ?? 0n;
			}
			if (late.has(r.id) || (view.obligation && late.has(view.obligation.id))) row.late = true;
			list.set(row.key, row);
		}
		for (const sale of people.state === 'ready' ? people.data : []) {
			if (!['offered', 'active', 'received'].includes(sale.state)) continue;
			const row = list.get(`p:${sale.target}`) ?? {
				key: `p:${sale.target}`,
				name: sale.customer_name || sale.target,
				kind: 'Person' as const,
				owed: 0n,
				waiting: 0,
				waitingKobo: 0n,
				late: false,
				href: `/workspace/sales/consumers/${encodeURIComponent(sale.id)}${query}`
			};
			if (sale.state === 'offered') {
				row.waiting += 1;
				row.waitingKobo += BigInt(Math.trunc(sale.terms?.total_kobo ?? 0));
			} else row.owed += BigInt(Math.trunc(sale.outstanding_kobo ?? 0));
			if (sale.schedule_progress?.some((item) => item.date < today && item.paid_kobo < item.amount_kobo))
				row.late = true;
			list.set(row.key, row);
		}
		return [...list.values()]
			.filter((row) => row.owed > 0n || row.waiting > 0)
			.sort((a, b) => Number(b.late) - Number(a.late) || (b.owed > a.owed ? 1 : b.owed < a.owed ? -1 : 0));
	});
	const personOwed = $derived(
		(people.state === 'ready' ? people.data : [])
			.filter((sale) => sale.state === 'active' || sale.state === 'received')
			.reduce((sum, sale) => sum + BigInt(Math.trunc(sale.outstanding_kobo ?? 0)), 0n)
	);
	const totalOwed = $derived(
		summary.state === 'ready' ? (exactKobo(summary.data.outstanding_kobo) ?? 0n) + personOwed : null
	);
	const dueThisWeek = $derived(
		(due.state === 'ready' ? due.data.length : 0) +
			(people.state === 'ready'
				? people.data.filter((sale) =>
						sale.schedule_progress?.some(
							(item) => item.date >= today && item.date <= weekAhead && item.paid_kobo < item.amount_kobo
						)
					).length
				: 0)
	);
	const scopeQuery = $derived(`?organization=${encodeURIComponent(organizationID)}`);
	async function loadRequests() {
		const scope = organizationID;
		const request = dashboardRequest.begin();
		visibleCount = 5;
		sales = pending(scope);
		people = pending(scope);
		payments = pending(scope);
		overdue = pending(scope);
		claims = pending(scope);
		disputes = pending(scope);
		summary = pending(scope);
		due = pending(scope);
		if (!scope) return;
		const root = `/api/v1/organizations/${encodeURIComponent(scope)}`;
		const results = await Promise.all([
			readResource(scope, `${root}/credit-requests`, rows('requests', saleView), request.signal, 'your sales'),
			readResource(scope, `${root}/payments`, rows('payments', paymentRow), request.signal, 'your payments'),
			readResource(scope, `${root}/overdue`, rows('overdue', workRow), request.signal, 'overdue sales'),
			readResource(
				scope,
				`${root}/payment-claims`,
				rows('payment_claims', workRow),
				request.signal,
				'reported transfers'
			),
			readResource(scope, `${root}/disputes`, rows('disputes', workRow), request.signal, 'reported problems'),
			readResource(scope, `${root}/reports/receivables`, receivables, request.signal, 'your balance'),
			readResource(scope, `${root}/due`, rows('due', workRow), request.signal, 'upcoming payments')
		]);
		if (!request.current() || organizationID !== scope) return;
		[sales, payments, overdue, claims, disputes, summary, due] = results;
		await loadPeople(scope);
	}
	async function loadPeople(scope: string) {
		try {
			const data = record(await readConsumer(`/api/v1/organizations/${encodeURIComponent(scope)}/consumer-sales`));
			if (!Array.isArray(data.items)) throw new Error('The list of persons could not be checked.');
			if (organizationID === scope)
				people = { state: 'ready', scope, data: data.items.map(purchase), checkedAt: new Date().toISOString() };
		} catch (cause) {
			if (organizationID === scope)
				people = {
					state: 'error',
					scope,
					status: 0,
					message: cause instanceof Error ? cause.message : 'Credit given to persons could not be loaded.'
				};
		}
	}
	async function load() {
		const request = businessRequest.begin();
		businesses = pending(account.userID);
		const result = await readResource(
			account.userID,
			'/api/v1/organizations',
			rows('organizations', organization),
			request.signal,
			'your businesses'
		);
		if (!request.current()) return;
		businesses = result;
		if (result.state === 'ready') {
			try {
				organizationID = requestedWorkspace(result.data);
			} catch {
				businesses = {
					state: 'error',
					scope: '',
					message: 'This business is not available in your account.',
					status: 404
				};
				return;
			}
			if (organizationID) await loadRequests();
		}
	}
	async function createOrganization() {
		if (createBusy) return;
		createBusy = true;
		createError = '';
		try {
			creation ??= new MutationIntent(account.userID, '/api/v1/organizations');
			await creation.run(
				{
					legal_name: legalName.trim(),
					trading_name: tradingName.trim(),
					business_type: businessType,
					registration_info: registrationInfo.trim(),
					business_address: address.trim(),
					industry: industry.trim(),
					timezone: 'Africa/Lagos',
					currency: 'NGN'
				},
				(value) => organization(record(value).organization)
			);
			await load();
		} catch (cause) {
			createError = cause instanceof Error ? cause.message : 'We could not confirm your business details.';
		} finally {
			createBusy = false;
		}
	}
	onMount(() => {
		// The server load already fetched this request's data. Re-fetching on
		// mount would discard it and reintroduce the two round trips it exists
		// to remove; a business change still goes through load() below.
		if (!seeded) void load();
		else if (organizationID) void loadPeople(organizationID);
		return () => {
			businessRequest.cancel();
			dashboardRequest.cancel();
		};
	});
</script>

<svelte:head><title>Who owes me — Kredit</title></svelte:head>
<main class="shell workspace account-home">
	<header class="task-heading">
		<div>
			<p class="eyebrow">{businessTitle}</p>
			<h1>Who owes me</h1>
		</div>
		{#if organizationID}<a class="primary give" href="/workspace/give{scopeQuery}">Give goods on credit</a>{/if}
	</header>
	<ResourceNotice resource={businesses} label="Businesses" retry={load} />
	{#if businesses.state === 'ready' && !organizations.length}
		<section class="card onboarding">
			<h2>Tell us about your business</h2>
			<p>This takes a minute. Next, you add the bank account your money will go to.</p>
			<p class="small">
				Did a supplier send you a link? Open that link instead. Buying for yourself? <a href="/personal/purchases"
					>See what you owe</a
				>.
			</p>
			<form
				class="form-grid"
				onsubmit={(event) => {
					event.preventDefault();
					void createOrganization();
				}}
			>
				<label
					>Business name<input
						bind:value={legalName}
						autocomplete="organization"
						required
						disabled={createBusy}
					/></label
				>
				<label
					>What do you sell?<input
						bind:value={industry}
						placeholder="Provisions, drinks, medicines…"
						required
						disabled={createBusy}
					/></label
				>
				<label class="wide"
					>Shop address<input
						bind:value={address}
						placeholder="Shop number, street, area, town"
						required
						disabled={createBusy}
					/></label
				>
				<label
					>Is the business registered?<select bind:value={businessType} disabled={createBusy}
						><option value="unregistered_business">Not registered</option><option value="registered_business"
							>Business name (CAC)</option
						><option value="sole_proprietor">Sole proprietor</option><option value="limited_company"
							>Limited company</option
						><option value="partnership">Partnership</option></select
					></label
				>
				{#if businessType !== 'unregistered_business'}<label
						>CAC number<input
							bind:value={registrationInfo}
							maxlength="80"
							disabled={createBusy}
							placeholder="RC or BN"
						/></label
					>{/if}
				{#if createError}<p class="error wide" role="alert">{createError}</p>{/if}<button
					class="primary wide"
					disabled={createBusy}>{createBusy ? 'Saving…' : 'Save and continue'}</button
				>
			</form>
		</section>
	{:else if currentBusiness}
		{#if organizations.length > 1}<label class="switch"
				>Business<select bind:value={organizationID} onchange={() => chooseWorkspace(organizationID)}
					>{#each organizations as org (org.id)}<option value={org.id}>{org.trading_name || org.legal_name}</option
						>{/each}</select
				></label
			>{/if}
		<BusinessNextSteps {organizationID} />
		<section class="balance-card" aria-label="What you are owed" aria-busy={summary.state === 'loading'}>
			<p>Customers owe you</p>
			{#if totalOwed !== null}
				<strong class="balance"><Money amountKobo={totalOwed} /></strong>
				<div class="figures">
					<span
						><small>Past due date</small><strong
							>{#if summary.state === 'ready'}<Money amountKobo={summary.data.overdue_kobo} />{/if}</strong
						></span
					>
					<span><small>Due in the next 7 days</small><strong>{dueThisWeek}</strong></span>
				</div>
			{:else}<strong class="balance" aria-hidden="true">—</strong>{/if}
		</section>
		{#if totalOwed === null}<ResourceNotice resource={summary} label="Balance" retry={loadRequests} />{/if}

		{#if allChecked && attention.length}<section class="attention" aria-labelledby="attention-heading">
				<h2 id="attention-heading">Check these</h2>
				<div class="action-list">
					{#each attention.slice(0, visibleCount) as item (item.id)}<a href={item.href}
							><span><strong>{item.title}</strong><small>{item.detail}</small></span><span aria-hidden="true">→</span
							></a
						>{/each}
				</div>
				{#if attention.length > visibleCount}<button
						class="secondary"
						type="button"
						onclick={() => (visibleCount += 10)}>Show {attention.length - visibleCount} more</button
					>{/if}
			</section>{/if}

		<section class="owing" aria-labelledby="owing-heading">
			<h2 id="owing-heading">Customers</h2>
			{#each [{ resource: sales, label: 'Credit to businesses' }, { resource: people, label: 'Credit to persons' }, { resource: overdue, label: 'Past due dates' }, { resource: payments, label: 'Payments' }, { resource: claims, label: 'Reported transfers' }, { resource: disputes, label: 'Reported problems' }] as item, i (i)}<ResourceNotice
					resource={item.resource}
					label={item.label}
					retry={loadRequests}
				/>{/each}
			{#if owing.length}<div class="record-list">
					{#each owing as row (row.key)}<a class="record-row" class:late={row.late} href={row.href}
							><span
								><strong>{row.name}</strong><small
									>{row.kind}{row.late ? ' · Past due date' : ''}{row.waiting
										? ` · ${row.waiting} not yet accepted`
										: ''}</small
								></span
							><strong
								>{#if row.owed > 0n}<Money amountKobo={row.owed} />{:else}<small class="pending"
										>Waiting: <Money amountKobo={row.waitingKobo} /></small
									>{/if}</strong
							></a
						>{/each}
				</div>
			{:else if sales.state === 'ready' && people.state === 'ready'}<div class="empty-state">
					<h3>Nobody owes you yet</h3>
					<p>When you give goods on credit, the customer shows here with what he owes and when.</p>
					<a class="primary" href="/workspace/give{scopeQuery}">Give goods on credit</a>
				</div>{/if}
		</section>
		<FeedbackPrompt area="seller" {organizationID} />
	{/if}
</main>

<style>
	.account-home {
		max-width: 48rem;
		padding-bottom: 3rem;
	}
	.task-heading {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding-block: 1rem 1.25rem;
	}
	.task-heading h1 {
		margin: 0.25rem 0 0;
		font-size: clamp(1.8rem, 5vw, 2.4rem);
		line-height: 1.15;
	}
	.give {
		white-space: nowrap;
	}
	.switch {
		display: grid;
		gap: 0.4rem;
		max-width: 22rem;
		margin-bottom: 1rem;
	}
	.switch select {
		font: inherit;
		padding: 0.7rem;
		border: 1px solid var(--color-border);
		border-radius: 0.35rem;
		background: var(--color-surface);
	}
	.balance-card {
		padding: 1.5rem;
		background: var(--color-primary);
		color: var(--color-on-primary);
		--color-muted: rgb(255 255 255 / 0.72);
	}
	.balance-card > p {
		margin: 0;
		color: var(--color-muted);
	}
	.balance {
		display: block;
		margin: 0.4rem 0 1rem;
		font-size: clamp(2.1rem, 7vw, 3.2rem);
		line-height: 1.15;
		letter-spacing: -0.03em;
		font-variant-numeric: tabular-nums;
		overflow-wrap: anywhere;
	}
	.figures {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
		padding-top: 1rem;
		border-top: 1px solid rgb(255 255 255 / 0.2);
	}
	.figures span {
		display: grid;
		gap: 0.2rem;
	}
	.figures small {
		color: var(--color-muted);
	}
	.figures strong {
		font-size: 1.2rem;
		font-variant-numeric: tabular-nums;
	}
	section.attention,
	section.owing {
		margin-top: 2rem;
	}
	h2 {
		margin: 0 0 0.75rem;
		font-size: 1.25rem;
	}
	.action-list {
		display: grid;
		gap: 0.5rem;
		margin-bottom: 0.75rem;
	}
	.action-list a {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding: 0.9rem 1rem;
		border: 1px solid var(--color-border);
		border-left: 4px solid var(--color-accent);
		border-radius: 0.4rem;
		background: var(--color-surface);
		color: inherit;
		text-decoration: none;
	}
	.action-list span:first-child,
	.record-row span {
		display: grid;
		gap: 0.2rem;
	}
	.action-list small,
	.record-row small {
		color: var(--color-muted);
	}
	.record-list {
		display: grid;
		border-top: 1px solid var(--color-border);
	}
	.record-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		min-height: 3.5rem;
		padding: 0.8rem 0.25rem;
		border-bottom: 1px solid var(--color-border);
		color: inherit;
		text-decoration: none;
	}
	.record-row > strong {
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
	}
	.pending {
		font-weight: 500;
	}
	.record-row.late small {
		color: var(--color-overdue, var(--color-accent));
		font-weight: 600;
	}
	.onboarding {
		padding: 1.5rem;
	}
	.onboarding .small {
		color: var(--color-muted);
		font-size: 0.9rem;
	}
	.form-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
	}
	.form-grid label {
		display: grid;
		gap: 0.45rem;
	}
	.form-grid input,
	.form-grid select {
		box-sizing: border-box;
		width: 100%;
		font: inherit;
		padding: 0.8rem;
		border: 1px solid var(--color-border);
		border-radius: 0.35rem;
		background: var(--color-surface);
	}
	.wide {
		grid-column: 1/-1;
	}
	.empty-state {
		padding: 1.5rem;
		border: 1px dashed var(--color-border);
		border-radius: 0.5rem;
	}
	.empty-state h3 {
		margin: 0;
		font-size: 1rem;
	}
	.empty-state p {
		line-height: 1.6;
		color: var(--color-muted);
	}
	@media (max-width: 560px) {
		.task-heading {
			align-items: start;
			flex-direction: column;
		}
		.form-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
