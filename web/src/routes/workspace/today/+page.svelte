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
	const pastDue = $derived(summary.state === 'ready' ? (exactKobo(summary.data.overdue_kobo) ?? 0n) : 0n);
	const todayLabel = new Intl.DateTimeFormat('en-NG', {
		day: 'numeric',
		month: 'short',
		year: 'numeric',
		timeZone: 'Africa/Lagos'
	}).format(new Date());
	const initials = (name: string) =>
		name
			.replace(/^\+?\d[\d\s]*$/, '#')
			.split(/\s+/)
			.filter((part) => /[A-Za-z#]/.test(part[0] ?? ''))
			.slice(0, 2)
			.map((part) => part[0]!.toUpperCase())
			.join('') || '•';
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
<main class="shell k-page">
	<header class="k-head">
		<div>
			<p class="k-eyebrow">{businessTitle}</p>
			<h1>Who owes me</h1>
		</div>
		{#if organizationID}<a class="primary" href="/workspace/give{scopeQuery}"
				>Give goods on credit <span aria-hidden="true">→</span></a
			>{/if}
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
		<section class="k-ink hero" aria-label="What you are owed" aria-busy={summary.state === 'loading'}>
			<div class="k-ink-bar"><span>Your balance</span><span class="k-mono">{todayLabel}</span></div>
			<p class="k-figure">
				<small>Customers owe you</small>
				{#if totalOwed !== null}<strong class="balance"><Money amountKobo={totalOwed} /></strong>{:else}<strong
						class="balance"
						aria-hidden="true">—</strong
					>{/if}
			</p>
			<div class="k-stats">
				<span class:alert={pastDue > 0n}
					><small>Past due date</small><strong
						>{#if summary.state === 'ready'}<Money amountKobo={summary.data.overdue_kobo} />{:else}—{/if}</strong
					></span
				>
				<span><small>Due in 7 days</small><strong>{dueThisWeek}</strong></span>
				<span><small>Customers owing</small><strong>{owing.filter((row) => row.owed > 0n).length}</strong></span>
			</div>
		</section>
		{#if totalOwed === null}<ResourceNotice resource={summary} label="Balance" retry={loadRequests} />{/if}

		{#if allChecked && attention.length}<section class="k-section" aria-labelledby="attention-heading">
				<div class="k-section-head">
					<h2 id="attention-heading">Check these</h2>
					<span>{attention.length} to look at</span>
				</div>
				<div class="k-ledger">
					{#each attention.slice(0, visibleCount) as item (item.id)}<a class="k-todo" href={item.href}
							><span><strong>{item.title}</strong><small>{item.detail}</small></span><span
								>{item.action} <span aria-hidden="true">→</span></span
							></a
						>{/each}
				</div>
				{#if attention.length > visibleCount}<button
						class="secondary more"
						type="button"
						onclick={() => (visibleCount += 10)}>Show {attention.length - visibleCount} more</button
					>{/if}
			</section>{/if}

		<section class="k-section" aria-labelledby="owing-heading">
			<div class="k-section-head">
				<h2 id="owing-heading">Customers</h2>
				<a href="/workspace/partners/customers{scopeQuery}">All customers <span aria-hidden="true">→</span></a>
			</div>
			{#each [{ resource: sales, label: 'Credit to businesses' }, { resource: people, label: 'Credit to persons' }, { resource: overdue, label: 'Past due dates' }, { resource: payments, label: 'Payments' }, { resource: claims, label: 'Reported transfers' }, { resource: disputes, label: 'Reported problems' }] as item, i (i)}<ResourceNotice
					resource={item.resource}
					label={item.label}
					retry={loadRequests}
				/>{/each}
			{#if owing.length}<div class="k-ledger record-list">
					<div class="k-ledger-head"><span>Customer</span><span>Owes you</span></div>
					{#each owing as row (row.key)}<a class="k-row record-row" class:late={row.late} href={row.href}
							><span class="k-mark" class:person={row.kind === 'Person'} aria-hidden="true">{initials(row.name)}</span
							><span class="k-who"
								><strong>{row.name}</strong><small
									>{#if row.late}<span class="k-tag">Past due</span>{' · '}{/if}{row.kind}{row.waiting
										? ` · ${row.waiting} not yet accepted`
										: ''}</small
								></span
							><span class="k-amount"
								>{#if row.owed > 0n}<strong><Money amountKobo={row.owed} /></strong>{:else}<strong class="pending"
										><Money amountKobo={row.waitingKobo} /></strong
									><small>Waiting to accept</small>{/if}</span
							></a
						>{/each}
				</div>
			{:else if sales.state === 'ready' && people.state === 'ready'}<div class="k-ledger k-empty">
					<h3>Nobody owes you yet</h3>
					<p>Give goods on credit and the customer shows here, with what he owes and when.</p>
					<a class="primary" href="/workspace/give{scopeQuery}">Give goods on credit</a>
				</div>{/if}
		</section>
		<FeedbackPrompt area="seller" {organizationID} />
	{/if}
</main>

<style>
	.hero .balance {
		color: var(--kredit-cream);
	}
	.switch {
		display: grid;
		gap: 0.4rem;
		max-width: 22rem;
		margin-bottom: 1.25rem;
		font-weight: 600;
	}
	.switch select {
		font: inherit;
		font-weight: 400;
		padding: 0.7rem;
		border: 1px solid var(--color-border);
		background: var(--color-surface);
	}
	.k-section-head a {
		color: var(--color-primary);
		font-weight: 600;
		font-size: 0.9rem;
		text-decoration: none;
	}
	.pending {
		color: var(--color-muted);
		font-weight: 550;
	}
	.more {
		margin-top: 0.75rem;
	}
	.onboarding {
		padding: 2rem;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
	}
	.onboarding h2 {
		margin-top: 0;
		font-family: var(--font-display);
		font-weight: 450;
		font-size: 1.6rem;
	}
	.onboarding .small {
		color: var(--color-muted);
		font-size: 0.9rem;
	}
	.form-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
		margin-top: 1.5rem;
	}
	.form-grid label {
		display: grid;
		gap: 0.45rem;
		font-weight: 600;
	}
	.form-grid input,
	.form-grid select {
		box-sizing: border-box;
		width: 100%;
		font: inherit;
		font-weight: 400;
		padding: 0.8rem;
		border: 1px solid var(--color-border-strong);
		background: var(--color-surface);
	}
	.wide {
		grid-column: 1/-1;
	}
	@media (max-width: 560px) {
		.form-grid {
			grid-template-columns: 1fr;
		}
		.onboarding {
			padding: 1.25rem;
		}
	}
</style>
