<script lang="ts">
	import { readableDate } from '$lib/datetime';
	import { buyerEndpoint } from '$lib/buyer-navigation';
	import RepaymentCustomer from '$lib/components/RepaymentCustomer.svelte';
	import IdentityChecks from '$lib/components/IdentityChecks.svelte';

	import { page } from '$app/state';
	import { workspaceHref } from '$lib/workspace-navigation';
	const businessQuery = $derived(
		page.url.searchParams.get('business_id')
			? `?business_id=${encodeURIComponent(page.url.searchParams.get('business_id')!)}`
			: ''
	);
	import { adminPost } from '$lib/admin-client';
	let checkingVerification = $state(false);
	async function checkVerification() {
		if (checkingVerification) return;
		checkingVerification = true;
		error = '';
		try {
			await adminPost(`/api/v1/buyer/me/verification/refresh${businessQuery}`, {});
			await load();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'We could not check verification.';
		} finally {
			checkingVerification = false;
		}
	}
	import { exactKobo, formatKobo, sumKobo, type KoboValue } from '$lib/money';
	import { kobo, saleView, type SaleView } from '$lib/records';
	import { checkedJSON, LatestRequest, record, rows, text, publicError } from '$lib/api/reliable';
	import FeedbackPrompt from '$lib/components/FeedbackPrompt.svelte';

	type Portal = {
		person: { full_name: string };
		business: { id: string; legal_name: string };
		verification_current: boolean;
	};
	type PaymentDay = {
		obligation_id: string;
		outstanding_kobo: KoboValue;
		next_due_kobo: KoboValue;
		next_due_at: string | null;
		overdue: boolean;
	};
	let portal = $state<Portal | null>(null);
	let requests = $state<SaleView[]>([]);
	let error = $state('');
	let balancesUnavailable = $state(false);
	let loading = $state(true);
	let paymentDays = $state<PaymentDay[]>([]),
		datesUnavailable = $state(true);
	// A sale whose obligation did not load contributes nothing we can vouch for,
	// so the total is reported as unconfirmed rather than quietly understated.
	const outstanding = $derived(
		balancesUnavailable
			? null
			: sumKobo(
					requests.map((item) =>
						item.obligation
							? item.obligation.outstanding_kobo
							: [
										'DRAFT',
										'SENT',
										'BUYER_REVIEWING',
										'BUYER_ACCEPTED',
										'VERIFICATION_PENDING',
										'READY_TO_RELEASE',
										'GOODS_RELEASED',
										'RECEIPT_CONFIRMATION_PENDING',
										'CANCELLED',
										'DECLINED'
								  ].includes(item.request.state)
								? 0
								: null
					)
				)
	);
	const pending = $derived(
		requests.filter((item) =>
			['SENT', 'BUYER_REVIEWING', 'PENDING_BUYER_CONFIRMATION'].includes(item.request.state.toUpperCase())
		)
	);
	const openBalances = $derived(requests.filter((item) => (exactKobo(item.obligation?.outstanding_kobo) ?? 0n) > 0n));

	const nextPayment = $derived(
		[...paymentDays]
			.filter((item) => item.next_due_at && (exactKobo(item.outstanding_kobo) ?? 0n) > 0n)
			.sort((a, b) => Date.parse(a.next_due_at ?? '') - Date.parse(b.next_due_at ?? ''))[0] ?? null
	);
	const overdueCount = $derived(
		paymentDays.filter((item) => item.overdue && (exactKobo(item.outstanding_kobo) ?? 0n) > 0n).length
	);
	function decodeDays(value: unknown): PaymentDay[] {
		return rows('obligations', (value): PaymentDay => {
			const item = record(value);
			const nextDueAt = item.next_due_at == null ? null : text(item.next_due_at);
			if (typeof item.overdue !== 'boolean' || (nextDueAt !== null && !Number.isFinite(Date.parse(nextDueAt))))
				throw new Error('Invalid payment dates');
			return {
				obligation_id: text(item.obligation_id),
				outstanding_kobo: kobo(item.outstanding_kobo),
				next_due_kobo: kobo(item.next_due_kobo),
				next_due_at: nextDueAt,
				overdue: item.overdue
			};
		})(value);
	}

	const reads = new LatestRequest();
	async function load(query = businessQuery) {
		const request = reads.begin();
		loading = true;
		error = '';
		balancesUnavailable = true;
		datesUnavailable = true;
		paymentDays = [];
		try {
			const [account, sales, days] = await Promise.allSettled([
				checkedJSON(
					`/api/v1/buyer/me${query}`,
					(value): Portal => {
						const response = record(value),
							portal = record(response.portal),
							business = record(portal.business);
						return {
							person: { full_name: text(record(portal.person).full_name) },
							business: { id: text(business.id), legal_name: text(business.legal_name) },
							verification_current: response.verification_current === true
						};
					},
					{ signal: request.signal }
				),
				checkedJSON(buyerEndpoint('/api/v1/buyer/credit-requests'), rows('requests', saleView), {
					signal: request.signal
				}),
				checkedJSON(buyerEndpoint('/api/v1/buyer/history'), decodeDays, { signal: request.signal })
			]);
			if (!request.current()) return;
			if (account.status === 'rejected') {
				portal = null;
				error = publicError(account.reason, 'your account');
				return;
			}
			const verified = account.value;
			portal = verified;
			if (days.status !== 'fulfilled') error = publicError(days.reason, 'your payment dates');
			if (sales.status === 'fulfilled') {
				requests = sales.value.filter((item) => item.request.buyer_business_id === verified.business.id);
				balancesUnavailable = false;
				if (days.status === 'fulfilled') {
					const ids = new Set(requests.map((item) => item.obligation?.id).filter(Boolean));
					paymentDays = days.value.filter((item) => ids.has(item.obligation_id));
					datesUnavailable = false;
				}
			} else {
				requests = [];
				error = publicError(sales.reason, 'your balances');
			}
		} catch (cause) {
			if (request.current()) error = publicError(cause, 'your account');
		} finally {
			if (request.current()) loading = false;
		}
	}
	$effect(() => {
		const query = businessQuery;
		void load(query);
		return () => reads.cancel();
	});
</script>

<svelte:head><title>Purchases — Kredit</title></svelte:head>

<main class="shell buyer-home">
	{#if portal}
		<header class="buyer-head">
			<div>
				<p class="eyebrow">Your business · Purchases</p>
				<h1>Supplier purchases</h1>
				<p>Signed in as <strong>{portal.person.full_name}</strong> for {portal.business.legal_name}.</p>
			</div>
			{#if pending.length}<a class="primary" href={workspaceHref('/workspace/purchases/orders', page.url)}
					>Read {pending.length} sale{pending.length === 1 ? '' : 's'} waiting for you</a
				>{/if}
		</header>
		{#if !portal.verification_current}<section>
				<h2>Your account checks need attention</h2>
				<p>
					Your details are saved. Identity, business and authority checks must be current before you can accept a sale.
					If a check has expired, contact support to renew it.
				</p>
				<button disabled={checkingVerification || loading} onclick={checkVerification}
					>{checkingVerification ? 'Checking…' : 'Check verification status'}</button
				>
			</section>{/if}
		{#if error}<p role="alert">{error}</p>
			<button onclick={() => void load()} disabled={loading}>Try again</button>{/if}{#if loading}<p role="status">
				Opening your balances…
			</p>{:else}
			<section class="owe-hero" aria-label="What you owe">
				<div class="owed">
					<span>You owe</span><strong>{outstanding === null ? 'Not confirmed' : formatKobo(outstanding)}</strong><small
						>{outstanding === null
							? 'We could not check every sale. Open them one by one.'
							: `Across ${openBalances.length} sale${openBalances.length === 1 ? '' : 's'}`}</small
					>
				</div>
				{#if outstanding === null || datesUnavailable}<p class="due-strip calm">
						Payment dates could not be confirmed.
					</p>{:else if overdueCount}<a
						class="due-strip late"
						href={workspaceHref('/workspace/purchases/obligations', page.url)}
						><span>{overdueCount} payment day{overdueCount === 1 ? ' has' : 's have'} passed</span><em
							>See what is late →</em
						></a
					>{:else if nextPayment}<a class="due-strip" href={workspaceHref('/workspace/purchases/obligations', page.url)}
						><span>Next: {formatKobo(nextPayment.next_due_kobo)}</span><em
							>by {readableDate(nextPayment.next_due_at ?? '')} →</em
						></a
					>{:else if outstanding > 0n}<a
						class="due-strip"
						href={workspaceHref('/workspace/purchases/obligations', page.url)}
						>Open your sales to check payment days →</a
					>{:else}<p class="due-strip calm">Nothing is due right now.</p>{/if}
			</section>
			<section class="next-actions">
				<div>
					<p class="eyebrow">What needs you</p>
					<h2>
						{pending.length
							? 'Review your new purchase offers.'
							: overdueCount
								? 'A payment day has passed.'
								: outstanding === null
									? 'We could not confirm your total.'
									: outstanding > 0n
										? 'See your payment schedule.'
										: 'No outstanding supplier balance.'}
					</h2>
				</div>
				<div class="action-copy">
					{#if pending.length}<p>
							Your supplier has proposed a purchase. Review the goods, total and payment dates before accepting.
						</p>
						<a href={workspaceHref('/workspace/purchases/orders', page.url)}>Read the sale →</a
						>{:else if overdueCount}<p>
							Open your balances and see what is late, what you have already paid and what is still left.
						</p>
						<a href={workspaceHref('/workspace/purchases/obligations', page.url)}>See what is late →</a
						>{:else if outstanding === null}<p>
							We could not confirm one or more of your balances. Open each sale instead of trusting the total above.
						</p>
						<a href={workspaceHref('/workspace/purchases/obligations', page.url)}>Check my balances →</a
						>{:else if outstanding > 0n}<p>
							You can open your balances any time and see what you agreed to, what you have paid and what is left.
						</p>
						<a href={workspaceHref('/workspace/purchases/obligations', page.url)}>See my balances →</a>{:else}<p>
							If a seller sends you a new sale, it will show up right here.
						</p>
						<a href={workspaceHref('/workspace/purchases/history', page.url)}>View payment history →</a>{/if}
				</div>
			</section>
		{/if}
		<FeedbackPrompt area="buyer" />
	{:else if error}
		<p class="eyebrow">Your business · Purchases</p>
		<h1>We could not open your account.</h1>
		<p class="error" role="alert">{error}</p>
		<button onclick={() => void load()} disabled={loading}>Try again</button>
		<p><a href="/signin?next=%2Fworkspace%2Fpurchases">Sign in to your account →</a></p>
		<p class="help">
			Look for the link in the message your seller sent you. If it will not open, just ask them for a fresh one. And
			never give your sign-in code to anybody, not even to somebody who says they are from Kredit.
		</p>
	{:else}
		<p class="eyebrow">Your business · Purchases</p>
		<h1>Opening your account…</h1>
	{/if}
	<IdentityChecks />
	{#if portal?.business?.id}<RepaymentCustomer
			businessID={portal.business.id}
			businessName={portal.business.legal_name}
		/>{/if}
</main>

<style>
	.owe-hero {
		margin: 1.5rem 0;
	}
	.owe-hero .owed {
		background: var(--color-primary);
		color: var(--color-on-primary);
		padding: 1.6rem 1.4rem 1.5rem;
		--color-muted: rgb(255 255 255 / 0.72);
	}
	.owe-hero .owed span {
		display: block;
		font-size: 0.7rem;
		font-weight: 800;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--color-muted);
	}
	.owe-hero .owed strong {
		display: block;
		font-family: var(--font-serif);
		font-size: clamp(2.6rem, 7vw, 4rem);
		font-weight: 500;
		line-height: 1;
		margin: 0.5rem 0 0.4rem;
		letter-spacing: -0.03em;
		font-variant-numeric: tabular-nums;
	}
	.owe-hero .owed small {
		color: var(--color-muted);
		font-size: 0.82rem;
	}
	.due-strip {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		margin: 0;
		padding: 0.85rem 1.4rem;
		background: var(--color-background);
		color: var(--color-primary);
		font-weight: 750;
		font-size: 0.9rem;
		text-decoration: none;
	}
	.due-strip em {
		font-style: normal;
		font-variant-numeric: tabular-nums;
	}
	.due-strip.late {
		background: var(--color-overdue);
		color: var(--color-on-primary);
	}
	.due-strip.calm {
		color: var(--color-foreground);
		background: var(--color-surface-muted);
	}
	.buyer-home {
		padding-bottom: 6rem;
	}
	.buyer-head {
		display: flex;
		justify-content: space-between;
		align-items: end;
		gap: 2rem;
		padding: 3rem 0 2rem;
		border-bottom: 3px solid var(--color-primary);
	}
	.buyer-head h1,
	.buyer-home > h1 {
		max-width: 13ch;
		margin: 0.5rem 0;
		font-family: var(--font-serif);
		font-size: clamp(1.9rem, 3.4vw, 2.6rem);
		line-height: 1.1;
		letter-spacing: -0.03em;
	}
	.buyer-head p {
		color: var(--color-foreground);
	}
	.next-actions {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: clamp(2rem, 8vw, 8rem);
		margin: 3rem 0;
		padding: 2rem 0;
		border-top: 1px solid var(--color-border);
		border-bottom: 1px solid var(--color-border);
	}
	.next-actions h2 {
		max-width: 12ch;
		margin: 0.5rem 0;
		font-family: var(--font-serif);
		font-size: clamp(2.2rem, 4vw, 3.8rem);
		font-weight: 500;
		line-height: 0.96;
		letter-spacing: -0.04em;
	}
	.action-copy {
		padding-top: 0.5rem;
	}
	.action-copy p {
		max-width: 36rem;
		color: var(--color-foreground);
		line-height: 1.7;
	}
	.action-copy a {
		color: var(--color-primary);
		font-weight: 800;
	}
	.error {
		color: var(--color-overdue);
	}
	.help {
		max-width: 34rem;
		color: var(--color-muted);
		line-height: 1.65;
	}
	@media (max-width: 900px) {
	}
	@media (max-width: 720px) {
		.buyer-head {
			display: block;
		}
		.buyer-head .primary {
			margin-top: 1rem;
		}
		.next-actions {
			grid-template-columns: 1fr;
		}
		.next-actions {
			gap: 1rem;
		}
	}
</style>
