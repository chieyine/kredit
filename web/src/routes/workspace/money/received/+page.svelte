<script lang="ts">
	import { chooseWorkspace, requestedWorkspace } from '$lib/workspace-context';
	import { getContext, onMount } from 'svelte';
	import { formatKobo, sumKobo } from '$lib/money';
	import { ACCOUNT_CONTEXT, type AccountContext } from '$lib/account-context';
	import { checkedJSON, LatestRequest, publicError, record, rows, text } from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';
	import { kobo, organization, timeLabel, type Organization } from '$lib/records';
	import type { KoboValue } from '$lib/money';
	import Money from '$lib/components/Money.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import PaymentReview from '$lib/components/PaymentReview.svelte';
	type Payment = {
		id: string;
		payment_id: string;
		buyer_legal_name: string;
		description: string;
		reference: string;
		amount_kobo: KoboValue;
		state: string;
		source_type: string;
		paid_at: string;
	};
	type Claim = {
		id: string;
		state: string;
		amount_kobo: KoboValue;
		transfer_reference: string;
		paid_at: string;
		hold_expires_at: string;
		credit_request_id: string;
		buyer_legal_name: string;
		goods_description: string;
	};
	const account = getContext<AccountContext>(ACCOUNT_CONTEXT);
	const optional = (value: unknown) => (typeof value === 'string' ? value : '');
	const payment = (value: unknown): Payment => {
		const row = record(value);
		return {
			id: text(row.id),
			payment_id: optional(row.payment_id),
			buyer_legal_name: text(row.buyer_legal_name),
			description: optional(row.description),
			reference: optional(row.reference),
			amount_kobo: kobo(row.amount_kobo),
			state: text(row.state),
			source_type: text(row.source_type),
			paid_at: text(row.paid_at)
		};
	};
	const claimRow = (value: unknown): Claim => {
		const row = record(value);
		return {
			id: text(row.id),
			state: text(row.state),
			amount_kobo: kobo(row.amount_kobo),
			transfer_reference: text(row.transfer_reference),
			paid_at: text(row.paid_at),
			hold_expires_at: optional(row.hold_expires_at),
			credit_request_id: optional(row.credit_request_id),
			buyer_legal_name: optional(row.buyer_legal_name),
			goods_description: optional(row.goods_description)
		};
	};
	let organizations = $state<Organization[]>([]),
		payments = $state<Payment[]>([]),
		claims = $state<Claim[]>([]),
		done = $state('');
	let organizationID = $state(''),
		error = $state(''),
		busy = $state(''),
		query = $state(''),
		status = $state('all'),
		reviewError = $state('');
	let loading = $state(true),
		review = $state<{ organizationID: string; claim: Claim; decision: 'confirmed' | 'rejected' } | null>(null);
	const reads = new LatestRequest(),
		businesses = new LatestRequest();
	// eslint-disable-next-line svelte/prefer-svelte-reactivity -- idempotency keys are never rendered
	const intents = new Map<string, MutationIntent>();
	const pendingClaims = $derived(claims.filter((claim) => claim.state === 'pending'));
	const pendingTotal = $derived(sumKobo(pendingClaims.map((claim) => claim.amount_kobo)));
	const receivedTotal = $derived(
		sumKobo(payments.filter((payment) => payment.state === 'recognized').map((payment) => payment.amount_kobo))
	);
	const visiblePayments = $derived(
		payments.filter(
			(payment) =>
				`${payment.buyer_legal_name} ${payment.description} ${payment.reference}`
					.toLowerCase()
					.includes(query.trim().toLowerCase()) &&
				(status === 'all' || payment.state === status)
		)
	);
	const date = (value: string) => (value ? timeLabel(value) : 'Not available');
	const source = (value: string) =>
		(
			({
				integrated_voluntary: 'Paid online',
				supplier_recorded_transfer: 'Bank transfer',
				buyer_payment_claim: 'Customer reported transfer',
				cash_recorded: 'Cash',
				kredit_collection: 'Collected by Kredit',
				adjustment: 'Account correction'
			}) as Record<string, string>
		)[value] ?? 'Payment';
	const stateLabel = (value: string) =>
		(
			({
				recognized: 'Received',
				reversed: 'Reversed',
				pending: 'Awaiting confirmation',
				confirmed: 'Received',
				rejected: 'Not received',
				expired: 'Expired'
			}) as Record<string, string>
		)[value] ?? 'Status unavailable';
	async function load() {
		const scope = organizationID,
			request = reads.begin();
		loading = true;
		error = '';
		payments = [];
		claims = [];
		if (!scope) {
			loading = false;
			return;
		}
		try {
			const root = `/api/v1/organizations/${encodeURIComponent(scope)}`;
			const [newPayments, newClaims] = await Promise.all([
				checkedJSON(`${root}/payments`, rows('payments', payment), { signal: request.signal }),
				checkedJSON(`${root}/payment-claims`, rows('payment_claims', claimRow), { signal: request.signal })
			]);
			if (!request.current() || organizationID !== scope) return;
			payments = newPayments;
			claims = newClaims;
		} catch (cause) {
			if (request.current() && organizationID === scope) error = publicError(cause, 'your payments');
		} finally {
			if (request.current() && organizationID === scope) loading = false;
		}
	}
	function decide(claim: Claim, decision: 'confirmed' | 'rejected') {
		if (busy || claim.state !== 'pending') return;
		done = '';
		reviewError = '';
		review = { organizationID, claim, decision };
	}
	async function confirmDecision() {
		if (!review || busy || review.organizationID !== organizationID) return;
		const selected = review;
		busy = selected.claim.id;
		reviewError = '';
		const url = `/api/v1/organizations/${encodeURIComponent(selected.organizationID)}/payment-claims/${encodeURIComponent(selected.claim.id)}/decide`;
		try {
			let intent = intents.get(url);
			if (!intent) {
				intent = new MutationIntent(account.userID, url);
				intents.set(url, intent);
			}
			await intent.run(
				{
					decision: selected.decision,
					reason:
						selected.decision === 'confirmed'
							? 'Supplier checked the receiving account and confirmed the transfer.'
							: 'Supplier checked the receiving account and could not locate the transfer.'
				},
				(value) => {
					const updated = claimRow(record(value).payment_claim);
					if (updated.id !== selected.claim.id || updated.state !== selected.decision)
						throw new Error('Payment decision not confirmed');
					return updated;
				}
			);
			review = null;
			done =
				selected.decision === 'confirmed'
					? `${formatKobo(selected.claim.amount_kobo)} recorded as received. The customer's balance has gone down.`
					: 'Marked as not received. The customer is told and their balance stays the same.';
			if (organizationID === selected.organizationID) await load();
		} catch (cause) {
			reviewError =
				cause instanceof Error
					? cause.message
					: 'The decision was not confirmed. Check the record before submitting another.';
		} finally {
			busy = '';
		}
	}
	async function start() {
		const request = businesses.begin();
		loading = true;
		error = '';
		try {
			const result = await checkedJSON('/api/v1/organizations', rows('organizations', organization), {
				signal: request.signal
			});
			if (!request.current()) return;
			organizations = result;
			const params = new URLSearchParams(location.search);
			query = params.get('q') ?? '';
			organizationID = requestedWorkspace(result);
			await load();
		} catch (cause) {
			if (request.current()) {
				error = publicError(cause, 'your businesses');
				loading = false;
			}
		}
	}
	onMount(() => {
		void start();
		return () => {
			reads.cancel();
			businesses.cancel();
		};
	});
</script>

<svelte:head><title>Money in — Kredit</title></svelte:head>
<main class="shell k-page payments-page">
	<header class="page-heading">
		<div>
			<p class="k-eyebrow">Your business</p>
			<h1>Money in</h1>
			<p class="lede">
				What your customers have paid, and transfers they say they sent. Check your bank before you confirm a transfer.
			</p>
			<p class="money-links">
				<a href={`/workspace/settings/settlement?organization=${encodeURIComponent(organizationID)}`}
					>Where your money goes</a
				>
				<a href={`/workspace/settings/billing?organization=${encodeURIComponent(organizationID)}`}>Kredit fees</a>
			</p>
		</div>
		{#if organizations.length > 1}<label
				>Business<select
					bind:value={organizationID}
					disabled={!!busy || !!review}
					onchange={() => chooseWorkspace(organizationID)}
					>{#each organizations as organization (organization.id)}<option value={organization.id}
							>{organization.trading_name || organization.legal_name}</option
						>{/each}</select
				></label
			>{/if}
	</header>
	{#if error}<div class="error-box" role="alert">
			<div>
				<strong>We could not open your payments.</strong>
				<p>{error}</p>
			</div>
			<button type="button" onclick={() => (organizationID ? load() : start())}>Try again</button>
		</div>{/if}
	{#if loading}<div class="loading" role="status">
			<span class="sr-only">Opening your payments</span><Skeleton rows={5} tall />
		</div>{:else if !error && !organizationID}<section class="empty-state">
			<h2>Add your business first</h2>
			<a href="/workspace/today">Add business details</a>
		</section>{:else if !error}
		{#if done}<p class="notice" role="status">{done}</p>{/if}
		<section class="k-ink" aria-label="Payment summary">
			<div class="k-ink-bar"><span>Money in</span><span class="k-mono">Confirmed payments</span></div>
			<p class="k-figure">
				<small>Money received</small><strong><Money amountKobo={receivedTotal} /></strong>
			</p>
			<div class="k-stats">
				<span class:alert={pendingClaims.length > 0}
					><small>Waiting for your answer</small><strong><Money amountKobo={pendingTotal} /></strong></span
				>
				<span><small>Transfers to check</small><strong>{pendingClaims.length}</strong></span>
				<span><small>Payments saved</small><strong>{payments.length}</strong></span>
			</div>
		</section>

		<section class="k-section" aria-labelledby="review-title">
			<div class="k-section-head">
				<h2 id="review-title">Check your bank for these</h2>
				<span>{pendingClaims.length ? `${pendingClaims.length} to answer` : 'All answered'}</span>
			</div>
			{#if pendingClaims.length}<div class="claims">
					{#each pendingClaims as claim (claim.id)}<article class="k-ledger claim">
							<div class="claim-top">
								<span class="k-eyebrow">{claim.buyer_legal_name || 'A customer'} says he sent</span>
								<strong class="claim-figure"><Money amountKobo={claim.amount_kobo} /></strong>
								{#if claim.goods_description}<p class="claim-sale">
										For {claim.goods_description}{#if claim.credit_request_id}<span class="sep" aria-hidden="true"
												>·</span
											><a
												href={`/workspace/sales/${encodeURIComponent(claim.credit_request_id)}?organization=${encodeURIComponent(organizationID)}`}
												>Open sale</a
											>{/if}
									</p>{/if}
							</div>
							<dl>
								<div>
									<dt>Transfer number</dt>
									<dd class="k-code">{claim.transfer_reference}</dd>
								</div>
								<div>
									<dt>Payment day</dt>
									<dd>{date(claim.paid_at)}</dd>
								</div>
								<div>
									<dt>Check before</dt>
									<dd>{date(claim.hold_expires_at)}</dd>
								</div>
							</dl>
							<p class="hint">Open your bank app and look for the money before you answer.</p>
							<div class="claim-actions">
								<button class="primary" disabled={!!busy} onclick={() => decide(claim, 'confirmed')}
									>{busy === claim.id ? 'Saving…' : 'Yes, I got the money'}</button
								><button class="secondary" disabled={!!busy} onclick={() => decide(claim, 'rejected')}
									>I cannot find this money</button
								>
							</div>
						</article>{/each}
				</div>{:else}<div class="k-ledger all-clear">
					<span aria-hidden="true">✓</span>
					<div>
						<h3>Nothing to check right now.</h3>
						<p>You have answered every payment your customers reported.</p>
					</div>
				</div>{/if}
		</section>

		<section class="k-section" aria-labelledby="history-title">
			<div class="k-section-head">
				<h2 id="history-title">Money received</h2>
			</div>
			<div class="filters">
				<label
					><span>Find a payment</span><input
						type="search"
						bind:value={query}
						placeholder="Customer or transfer number"
					/></label
				><label
					><span>Show</span><select bind:value={status}
						><option value="all">All payments</option><option value="recognized">Received</option><option
							value="reversed">Reversed</option
						></select
					></label
				>
			</div>
			{#if visiblePayments.length}<div class="k-ledger payment-table" role="table" aria-label="Payments received">
					<div class="k-ledger-head table-head" role="row">
						<span role="columnheader">Customer</span><span role="columnheader">Amount</span><span role="columnheader"
							>How</span
						><span role="columnheader">Date</span><span role="columnheader">Status</span><span aria-hidden="true"
						></span>
					</div>
					{#each visiblePayments as payment (payment.id)}<div class="payment-row" role="row">
							<div role="cell" class="who">
								<strong>{payment.buyer_legal_name || 'Customer'}</strong><small
									>{payment.description || payment.reference || 'Sale payment'}</small
								>
							</div>
							<div role="cell" class="amount"><strong><Money amountKobo={payment.amount_kobo} /></strong></div>
							<span role="cell" class="muted">{source(payment.source_type)}</span><span role="cell" class="muted"
								>{date(payment.paid_at)}</span
							><span role="cell" class:reversed={payment.state === 'reversed'} class="k-tag quiet payment-state"
								>{stateLabel(payment.state)}</span
							><a
								role="cell"
								class="open"
								href={`/workspace/sales/${encodeURIComponent(payment.id)}?organization=${encodeURIComponent(organizationID)}`}
								>Open sale <span aria-hidden="true">→</span></a
							>
						</div>{/each}
				</div>
			{:else if payments.length}<div class="k-ledger k-empty">
					<h3>No payment matches that.</h3>
					<p>Try a different customer name, transfer number or status.</p>
				</div>{:else}<div class="k-ledger k-empty">
					<h3>No confirmed payments yet.</h3>
					<p>Verified payments appear here. A reported transfer stays separate until it is confirmed.</p>
					<a class="primary" href={`/workspace/give?organization=${encodeURIComponent(organizationID)}`}
						>Give goods on credit</a
					>
				</div>{/if}
		</section>
	{/if}
</main>
{#if review}<PaymentReview
		amount={review.claim.amount_kobo}
		reference={review.claim.transfer_reference}
		decision={review.decision}
		busy={!!busy}
		error={reviewError}
		onconfirm={confirmDecision}
		oncancel={() => {
			review = null;
			reviewError = '';
		}}
	/>{/if}

<style>
	.payments-page {
		max-width: 60rem;
	}
	.page-heading {
		display: flex;
		justify-content: space-between;
		align-items: end;
		gap: 1rem 2rem;
		flex-wrap: wrap;
		padding-bottom: 1.75rem;
	}
	.page-heading h1 {
		margin: 0.35rem 0 0.5rem;
		font-size: clamp(2.1rem, 5vw, 3rem);
		line-height: 1.02;
		letter-spacing: -0.04em;
	}
	.page-heading .lede {
		margin: 0;
		color: var(--color-muted);
		max-width: 56ch;
		line-height: 1.55;
	}
	.page-heading label,
	.filters label {
		display: grid;
		gap: 0.35rem;
		font-size: 0.85rem;
		font-weight: 650;
	}
	.page-heading select,
	.filters input,
	.filters select {
		box-sizing: border-box;
		min-height: 3rem;
		padding: 0.65rem 0.8rem;
		border: 1px solid var(--color-border-strong);
		background: var(--color-surface);
		color: var(--color-foreground);
		font: inherit;
		font-weight: 400;
	}
	.money-links {
		display: flex;
		flex-wrap: wrap;
		gap: 0.6rem;
		margin: 1.1rem 0 0;
	}
	.money-links a {
		display: inline-flex;
		align-items: center;
		min-height: 2.75rem;
		padding: 0 1rem;
		border: 1px solid var(--color-border-strong);
		background: var(--color-surface);
		color: var(--color-foreground);
		font-size: 0.9rem;
		font-weight: 600;
		text-decoration: none;
	}
	.money-links a:hover {
		border-color: var(--kredit-ink);
	}
	.error-box {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		flex-wrap: wrap;
		padding: 1rem 1.25rem;
		border-left: 3px solid var(--color-destructive);
		background: var(--color-surface);
		color: var(--color-destructive);
	}
	.error-box p {
		margin: 0.25rem 0 0;
	}
	.error-box button {
		border: 1px solid currentColor;
		background: transparent;
		color: inherit;
	}
	.notice {
		padding: 0.9rem 1.1rem;
		border-left: 3px solid var(--color-positive);
		background: var(--color-surface);
		color: var(--color-positive);
		font-weight: 600;
	}
	.claims {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 22rem), 1fr));
		gap: 1rem;
	}
	.claim {
		display: grid;
		gap: 1rem;
		padding: 1.25rem;
		border-top: 3px solid var(--kredit-orange);
	}
	.claim-top {
		display: grid;
		gap: 0.35rem;
	}
	.claim-figure {
		font-size: 1.9rem;
		font-weight: 650;
		letter-spacing: -0.035em;
		font-variant-numeric: tabular-nums;
	}
	.claim-sale {
		margin: 0;
		color: var(--color-muted);
		font-size: 0.9rem;
	}
	.claim-sale .sep {
		margin-inline: 0.35rem;
	}
	.claim-sale a {
		color: var(--color-primary);
		font-weight: 600;
	}
	.claim dl {
		display: grid;
		gap: 0.55rem;
		margin: 0;
		padding-top: 1rem;
		border-top: 1px solid var(--color-border);
	}
	.claim dl div {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
	}
	.claim dt {
		color: var(--color-muted);
	}
	.claim dd {
		margin: 0;
		font-weight: 600;
		text-align: right;
	}
	.k-code {
		font-family: ui-monospace, Menlo, Consolas, monospace;
		font-size: 0.9rem;
	}
	.hint {
		margin: 0;
		padding: 0.75rem 0.9rem;
		background: var(--color-background);
		color: var(--color-muted);
		font-size: 0.9rem;
		line-height: 1.5;
	}
	.claim-actions {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.6rem;
	}
	.claim-actions button {
		width: 100%;
	}
	.all-clear {
		display: flex;
		align-items: center;
		gap: 1rem;
		padding: 1.25rem;
	}
	.all-clear > span {
		display: grid;
		place-items: center;
		width: 2.4rem;
		height: 2.4rem;
		background: var(--kredit-ink);
		color: var(--kredit-orange);
		font-weight: 700;
	}
	.all-clear h3 {
		margin: 0;
	}
	.all-clear p {
		margin: 0.2rem 0 0;
		color: var(--color-muted);
	}
	.filters {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 12rem;
		gap: 0.75rem;
		margin-bottom: 1rem;
	}
	.table-head,
	.payment-row {
		display: grid;
		grid-template-columns: minmax(0, 1.6fr) minmax(0, 1fr) minmax(0, 0.8fr) minmax(0, 1.1fr) minmax(0, 0.9fr) auto;
		align-items: center;
		gap: 1rem;
	}
	.payment-row {
		min-height: 4.25rem;
		padding: 0.85rem 1.25rem;
		border-top: 1px solid var(--color-border);
	}
	.payment-row .who {
		display: grid;
		gap: 0.2rem;
		min-width: 0;
	}
	.payment-row small,
	.muted {
		color: var(--color-muted);
		font-size: 0.88rem;
	}
	.amount {
		font-variant-numeric: tabular-nums;
	}
	.payment-state.reversed {
		color: var(--color-overdue);
	}
	.open {
		color: var(--color-primary);
		font-weight: 600;
		font-size: 0.9rem;
		white-space: nowrap;
		text-decoration: none;
	}
	@media (max-width: 760px) {
		.table-head {
			display: none;
		}
		.payment-row {
			grid-template-columns: minmax(0, 1fr) auto;
			gap: 0.35rem 1rem;
		}
		.payment-row .amount {
			text-align: right;
		}
		.payment-row .muted,
		.payment-state,
		.open {
			grid-column: 1 / -1;
		}
		.filters {
			grid-template-columns: 1fr;
		}
		.claim-actions {
			grid-template-columns: 1fr;
		}
	}
</style>
