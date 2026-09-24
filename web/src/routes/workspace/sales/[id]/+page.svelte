<script lang="ts">
	import SaleProgress from '$lib/components/SaleProgress.svelte';
	import SaleCosts from '$lib/components/SaleCosts.svelte';
	import { actualPaymentTime } from '$lib/financial-input';
	import { localInput, lagosISO } from '$lib/admin-client';
	import { feeDisclosure } from '$lib/fee-terms';
	import { exactKobo, formatKobo, nairaInput, parseNaira, type KoboValue } from '$lib/money';
	import { page } from '$app/state';
	import { getContext, untrack } from 'svelte';
	import { ACCOUNT_CONTEXT, type AccountContext } from '$lib/account-context';
	import {
		checkedJSON,
		LatestRequest,
		optionalText,
		readResource,
		record,
		rows,
		text,
		publicError
	} from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';
	import { saleView, dateLabel, kobo, organization, timeLabel, type SaleView } from '$lib/records';
	import PaymentReview from '$lib/components/PaymentReview.svelte';

	import Money from '$lib/components/Money.svelte';
	import ShareActions from '$lib/components/ShareActions.svelte';
	import { productLabel } from '$lib/product-language';
	const id = $derived(page.params.id ?? '');
	const account = getContext<AccountContext>(ACCOUNT_CONTEXT);
	const reads = new LatestRequest(),
		// eslint-disable-next-line svelte/prefer-svelte-reactivity -- idempotency keys are never rendered
		intents = new Map<string, MutationIntent>();
	let loading = $state(true),
		paymentError = $state(''),
		scheduleError = $state(''),
		claimError = $state(''),
		collectionError = $state(''),
		eligibilityError = $state('');
	type SupplierSale = SaleView & { request: SaleView['request'] & { version: number } };
	type SalePayment = {
		id: string;
		amount_kobo: KoboValue;
		state: string;
		source_type: string;
		provider_reference: string;
	};
	type ScheduleItem = {
		id: string;
		state: string;
		due_at: string;
		principal_due_kobo: KoboValue;
		allocated_kobo: KoboValue;
	};
	type PaymentClaim = {
		id: string;
		amount_kobo: KoboValue;
		state: string;
		transfer_reference: string;
		paid_at: string;
		review_reason: string;
	};
	type CollectionAttempt = { id: string; requested_amount_kobo: KoboValue; state: string; requested_at: string };
	type Eligibility = { eligible: boolean; amount_kobo: KoboValue; reasons: string[] };
	/** A payment day with principal still unpaid, compared exactly in kobo. */
	function hasUnpaid(item: ScheduleItem) {
		const due = exactKobo(item.principal_due_kobo),
			paid = exactKobo(item.allocated_kobo);
		return item.state !== 'CANCELLED' && due !== null && paid !== null && due > paid;
	}
	let review = $state<{ claim: PaymentClaim; decision: 'confirmed' | 'rejected'; reason: string } | null>(null);
	function intentFor(url: string) {
		let intent = intents.get(url);
		if (!intent) {
			intent = new MutationIntent(account.userID, url);
			intents.set(url, intent);
		}
		return intent;
	}
	function entity(value: unknown, field: string) {
		const result = record(value);
		if (!text(record(result[field]).id)) throw new Error('Missing record reference');
		return result;
	}
	function checkedSale(value: unknown): SupplierSale {
		const result = entity(value, 'request'),
			sale = record(result.request);
		text(sale.state);
		text(sale.buyer_legal_name);
		text(sale.goods_description);
		kobo(sale.principal_kobo);
		text(sale.due_date);
		if (!Number.isSafeInteger(sale.version) || Number(sale.version) < 0) throw new Error('Missing sale version');
		if (result.agreement) {
			text(record(result.agreement).id);
			text(record(result.agreement).document_hash);
		}
		if (result.obligation) {
			text(record(result.obligation).id);
			kobo(record(result.obligation).outstanding_kobo);
		}
		const verified = saleView(result);
		return { ...verified, request: { ...verified.request, version: Number(sale.version) } };
	}
	function salePayment(value: unknown): SalePayment {
		const row = record(value);
		return {
			id: text(row.id),
			amount_kobo: kobo(row.amount_kobo),
			state: text(row.state),
			source_type: optionalText(row.source_type),
			provider_reference: optionalText(row.provider_reference)
		};
	}
	function scheduleItem(value: unknown): ScheduleItem {
		const row = record(value);
		const due = text(row.due_at);
		if (!Number.isFinite(Date.parse(due))) throw new Error('Invalid payment date');
		return {
			id: text(row.id),
			state: text(row.state),
			due_at: due,
			principal_due_kobo: kobo(row.principal_due_kobo),
			allocated_kobo: kobo(row.allocated_kobo)
		};
	}
	function paymentClaim(value: unknown): PaymentClaim {
		const row = record(value);
		return {
			id: text(row.id),
			amount_kobo: kobo(row.amount_kobo),
			state: text(row.state),
			transfer_reference: optionalText(row.transfer_reference),
			paid_at: optionalText(row.paid_at),
			review_reason: optionalText(row.review_reason)
		};
	}
	function collectionAttempt(value: unknown): CollectionAttempt {
		const row = record(value);
		return {
			id: text(row.id),
			requested_amount_kobo: kobo(row.requested_amount_kobo),
			state: text(row.state),
			requested_at: optionalText(row.requested_at)
		};
	}

	let organizationID = $state(''),
		view = $state<SupplierSale | null>(null),
		payments: SalePayment[] = $state([]),
		error = $state(''),
		busy = $state(false);
	let scheduleItems: ScheduleItem[] = $state([]),
		paymentClaims: PaymentClaim[] = $state([]),
		collectionAttempts: CollectionAttempt[] = $state([]),
		eligibility: Eligibility | null = $state(null),
		notice = $state('');
	const nextDueItem = $derived(scheduleItems.find(hasUnpaid));
	let deliveryMethod = $state('supplier_delivery'),
		releaseNotes = $state(''),
		paymentAmount = $state(''),
		paymentReference = $state(''),
		paymentPaidAt = $state(''),
		paymentSource = $state('supplier_recorded_transfer');
	let paymentNotice = $state('');
	let reversePaymentID = $state(''),
		reverseReason = $state(''),
		claimReviewReason = $state(''),
		operationType = $state('write-off'),
		operationAmount = $state(''),
		operationReason = $state('');
	let draftPrincipal = $state(''),
		draftGoods = $state(''),
		draftDueDate = $state(''),
		draftCollectionAt = $state(''),
		draftGraceHours = $state(0);
	let disputeAmount = $state(''),
		disputeReason = $state(''),
		disputeExplanation = $state(''),
		disputeEffect = $state('CONTESTED_ONLY');
	async function load(requestID = id, selectedOrg = page.url.searchParams.get('organization') ?? organizationID) {
		const request = reads.begin();
		loading = true;
		error = '';
		view = null;
		payments = [];
		paymentClaims = [];
		scheduleItems = [];
		collectionAttempts = [];
		eligibility = null;
		paymentError = '';
		scheduleError = '';
		claimError = '';
		collectionError = '';
		eligibilityError = '';
		try {
			let org = selectedOrg;
			if (!org)
				org =
					(
						await checkedJSON('/api/v1/organizations', rows('organizations', organization), { signal: request.signal })
					)[0]?.id ?? '';
			if (!request.current()) return;
			organizationID = org;
			if (!org || !requestID) {
				error = 'Add your business first, then open this sale.';
				return;
			}
			const base = `/api/v1/organizations/${encodeURIComponent(org)}/credit-requests/${encodeURIComponent(requestID)}`;
			const result = await checkedJSON(base, checkedSale, { signal: request.signal });
			if (!request.current()) return;
			view = result;
			if (result.request.state === 'DRAFT') {
				draftPrincipal = nairaInput(result.request.principal_kobo);
				draftGoods = result.request.goods_description;
				draftDueDate = result.request.due_date;
				draftCollectionAt = localInput(result.request.collection_at);
				draftGraceHours = result.request.grace_hours;
			}
			if (result.obligation) {
				const [p, s, c, e, a] = await Promise.all([
					readResource(base, `${base}/payments`, rows('payments', salePayment), request.signal, 'payment history'),
					readResource(
						base,
						`${base}/schedule`,
						(value) => ({ items: rows('items', scheduleItem)(value) }),
						request.signal,
						'payment dates'
					),
					readResource(
						base,
						`${base}/payment-claims`,
						rows('payment_claims', paymentClaim),
						request.signal,
						'reported transfers'
					),
					readResource(
						base,
						`${base}/collection/eligibility`,
						(value): Eligibility => {
							const data = record(value);
							if (typeof data.eligible !== 'boolean') throw new Error('Missing eligibility');
							if (
								data.reasons !== undefined &&
								(!Array.isArray(data.reasons) || !data.reasons.every((value) => typeof value === 'string'))
							)
								throw new Error('Invalid eligibility reasons');
							return {
								eligible: data.eligible,
								amount_kobo: data.eligible ? kobo(data.amount_kobo) : null,
								reasons: (data.reasons as string[] | undefined) ?? []
							};
						},
						request.signal,
						'bank debit eligibility'
					),
					readResource(
						base,
						`${base}/collections`,
						rows('attempts', collectionAttempt),
						request.signal,
						'bank debit history'
					)
				]);
				if (!request.current()) return;
				if (p.state === 'ready') payments = p.data;
				else if (p.state === 'error') paymentError = p.message;
				if (s.state === 'ready') {
					scheduleItems = s.data.items;
				} else if (s.state === 'error') scheduleError = s.message;
				if (c.state === 'ready') paymentClaims = c.data;
				else if (c.state === 'error') claimError = c.message;
				if (e.state === 'ready') eligibility = e.data;
				else if (e.state === 'error') eligibilityError = e.message;
				if (a.state === 'ready') collectionAttempts = a.data;
				else if (a.state === 'error') collectionError = a.message;
			}
		} catch (cause) {
			if (request.current()) {
				view = null;
				error = publicError(cause, 'this sale');
			}
		} finally {
			if (request.current()) loading = false;
		}
	}
	async function mutate(url: string, body: unknown, field: string, method = 'POST') {
		if (busy || loading) return false;
		busy = true;
		error = '';
		notice = '';
		paymentNotice = '';
		const requestID = id,
			org = organizationID;
		try {
			await intentFor(url).run(body, (value) => entity(value, field), method);
			if (id !== requestID || organizationID !== org) return false;
			await load();
			return true;
		} catch (cause) {
			if (id === requestID && organizationID === org)
				error =
					cause instanceof Error
						? cause.message
						: 'We have not confirmed the result. Check the record before submitting another request.';
			return false;
		} finally {
			busy = false;
		}
	}
	function draftChanged() {
		if (!view || view.request.state !== 'DRAFT') return false;
		return (
			draftPrincipal !== nairaInput(view.request.principal_kobo) ||
			draftGoods !== view.request.goods_description ||
			draftDueDate !== view.request.due_date ||
			draftCollectionAt !== localInput(view.request.collection_at) ||
			Number(draftGraceHours) !== view.request.grace_hours
		);
	}
	async function updateDraft() {
		error = '';
		if (!view) return;
		const expectedVersion = view.request.version;
		try {
			const principalKobo = parseNaira(draftPrincipal);
			if (principalKobo <= 0) throw new Error('Enter a sale amount above zero.');
			await mutate(
				`/api/v1/organizations/${encodeURIComponent(organizationID)}/credit-requests/${encodeURIComponent(id)}`,
				{
					expected_version: expectedVersion,
					principal_kobo: principalKobo,
					goods_description: draftGoods,
					due_date: draftDueDate,
					grace_hours: Number(draftGraceHours),
					collection_at: lagosISO(draftCollectionAt)
				},
				'request',
				'PATCH'
			);
		} catch {
			error = 'Check the sale amount and payment dates before saving.';
		}
	}
	async function command(path: string, body: unknown = undefined) {
		if (path === 'send' && draftChanged()) {
			error = 'Save your changes before sending this sale.';
			return false;
		}
		return mutate(
			`/api/v1/organizations/${encodeURIComponent(organizationID)}/credit-requests/${encodeURIComponent(id)}/${path}`,
			body,
			path === 'payments' ? 'payment' : path === 'disputes' ? 'dispute' : 'request'
		);
	}
	async function recordPayment() {
		paymentNotice = '';
		try {
			const amount = parseNaira(paymentAmount);
			if (amount <= 0) {
				error = 'Enter an amount above zero.';
				return;
			}
			const paidAt = actualPaymentTime(paymentPaidAt);
			const confirmed = await command('payments', {
				amount_kobo: amount,
				currency: 'NGN',
				source_type: paymentSource,
				provider_reference: paymentReference,
				paid_at: paidAt
			});
			if (confirmed) {
				paymentNotice = `${formatKobo(amount)} payment saved.`;
				paymentAmount = '';
				paymentReference = '';
				paymentPaidAt = '';
			}
		} catch {
			error = 'Check the amount and actual payment date/time before saving this payment.';
		}
	}
	async function openDispute(event: SubmitEvent) {
		event.preventDefault();
		const amount = parseNaira(disputeAmount);
		if (amount <= 0 || !disputeReason.trim()) {
			error = 'Please enter how much is affected and what is wrong.';
			return;
		}
		await command('disputes', {
			disputed_amount_kobo: amount,
			reason: disputeReason,
			explanation: disputeExplanation,
			collection_effect: disputeEffect
		});
		if (!error) {
			disputeAmount = '';
			disputeReason = '';
			disputeExplanation = '';
		}
	}
	async function outsideCommand(path: string, body: unknown = {}) {
		const field = path.endsWith('/reverse')
			? 'payment'
			: path.includes('/payment-claims/')
				? 'payment_claim'
				: path.endsWith('/write-off') || path.endsWith('/fee-waiver')
					? 'action'
					: 'attempt';
		return mutate(path, body, field);
	}
	async function startCollection() {
		// The API accepts the stable Idempotency-Key header; do not generate a second identity in the body.
		if (
			await outsideCommand(
				`/api/v1/organizations/${encodeURIComponent(organizationID)}/credit-requests/${encodeURIComponent(id)}/collection`
			)
		)
			notice = 'The collection request was accepted. Check its status below; this is not a confirmed payment.';
	}
	async function collectionAction(attempt: CollectionAttempt, action: string) {
		if (
			await outsideCommand(
				`/api/v1/organizations/${encodeURIComponent(organizationID)}/collections/${encodeURIComponent(attempt.id)}/${action}`
			)
		)
			notice =
				action === 'retry' ? 'We have asked the bank again.' : 'We checked with the bank. The latest result is below.';
	}
	async function reversePayment() {
		if (!reversePaymentID || !reverseReason.trim()) {
			error = 'Choose the payment, and say why it should be removed.';
			return;
		}
		if (
			await outsideCommand(
				`/api/v1/organizations/${encodeURIComponent(organizationID)}/credit-requests/${encodeURIComponent(id)}/payments/${encodeURIComponent(reversePaymentID)}/reverse`,
				{ reason: reverseReason }
			)
		) {
			notice = 'Payment removed. The money left to pay has been updated.';
			reversePaymentID = '';
			reverseReason = '';
		}
	}
	function decidePaymentClaim(claim: PaymentClaim, decision: 'confirmed' | 'rejected') {
		if (busy) return;
		if (!claimReviewReason.trim()) {
			error = 'Please say why you are accepting or rejecting this.';
			return;
		}
		error = '';
		review = { claim, decision, reason: claimReviewReason.trim() };
	}
	async function confirmClaim() {
		if (!review || busy) return;
		const selected = review;
		if (
			await outsideCommand(
				`/api/v1/organizations/${encodeURIComponent(organizationID)}/payment-claims/${encodeURIComponent(selected.claim.id)}/decide`,
				{
					decision: selected.decision,
					reason: selected.reason
				}
			)
		) {
			notice =
				selected.decision === 'confirmed'
					? 'The transfer is confirmed in the payment record.'
					: 'The transfer was marked not received.';
			review = null;
			claimReviewReason = '';
		}
	}
	async function changeMoney() {
		const amount_kobo = parseNaira(operationAmount);
		if (amount_kobo <= 0 || !operationReason.trim()) {
			error = 'Please enter the amount and the reason.';
			return;
		}
		const path = operationType === 'write-off' ? 'write-off' : 'fee-waiver';
		if (
			await outsideCommand(
				`/api/v1/organizations/${encodeURIComponent(organizationID)}/credit-requests/${encodeURIComponent(id)}/${path}`,
				{
					amount_kobo,
					reason: operationReason,
					approved_by: ''
				}
			)
		) {
			notice =
				operationType === 'write-off'
					? 'Done. The money owed has been reduced.'
					: 'Done. The Kredit fee has been reduced.';
			operationAmount = '';
			operationReason = '';
		}
	}
	async function openInvoice() {
		const documentID = view?.request.invoice_document_id;
		if (busy || loading || !documentID) return;
		error = '';
		try {
			const url = await checkedJSON(
				`/api/v1/organizations/${encodeURIComponent(organizationID)}/documents/${encodeURIComponent(documentID)}/download`,
				(value) => {
					const url = new URL(text(record(value).url), location.origin);
					if (
						!['https:', 'http:'].includes(url.protocol) ||
						url.username ||
						url.password ||
						(location.protocol === 'https:' && url.protocol !== 'https:')
					)
						throw new Error('Invalid invoice link');
					return url.href;
				}
			);
			window.location.assign(url);
		} catch {
			error = 'We could not open that invoice. Please try again.';
		}
	}

	$effect(() => {
		const requestID = id,
			selectedOrg = page.url.searchParams.get('organization') ?? '';
		untrack(() => {
			review = null;
			notice = '';
			paymentNotice = '';
			paymentAmount = '';
			paymentReference = '';
			paymentPaidAt = '';
			claimReviewReason = '';
			void load(requestID, selectedOrg);
		});
		return () => reads.cancel();
	});
</script>

<svelte:head><title>Sale {id} — Kredit</title></svelte:head>
<section class="page-shell">
	<a href="/workspace/today">← Your sales</a>
	{#if error}<p class="error" role="alert">{error}</p>{/if}
	{#if notice}<p class="success notice" role="status">{notice}</p>{/if}
	{#if !view}{#if loading}<p role="status">Opening sale…</p>{:else}<button onclick={() => load()}>Try again</button
			>{/if}{:else}
		<p class="eyebrow">Sale · {productLabel(view.request.state)}</p>
		<h1>{view.request.buyer_legal_name}</h1>
		<p class="muted">The sale, the goods and every payment stay together here.</p>
		<SaleProgress {view} audience="seller" /><SaleCosts {view} />
		<div class="quick-actions">
			<a
				class="repeat"
				href={`/workspace/sales/new?organization=${encodeURIComponent(organizationID)}&customer=${encodeURIComponent(view.request.buyer_user_id ?? '')}&goods=${encodeURIComponent(view.request.goods_description)}&amount=${encodeURIComponent(nairaInput(view.request.principal_kobo))}`}
				>Sell these same goods again</a
			><a class="repeat" href={`/workspace/sales/${id}/deliveries?organization=${encodeURIComponent(organizationID)}`}
				>Deliveries & Credit Notes →</a
			><ShareActions
				compact
				title="Kredit payment reminder"
				text={`Hello ${view.request.buyer_legal_name}, this is a reminder that ${formatKobo(view.obligation?.outstanding_kobo ?? view.request.principal_kobo)} is left for ${view.request.goods_description}. Payment day: ${nextDueItem?.due_at?.slice(0, 10) ?? (view.obligation ? 'Check your current payment schedule' : view.request.due_date)}.`}
			/>
		</div>
		<section class="detail-grid">
			<article class="card">
				<h2>The sale</h2>
				<p>{feeDisclosure(view.request.fee_terms)}</p>
				<dl>
					<div>
						<dt>Money to pay</dt>
						<dd><Money amountKobo={view.request.principal_kobo} /></dd>
					</div>
					<div>
						<dt>Goods</dt>
						<dd>{view.request.goods_description}</dd>
					</div>
					<div>
						<dt>{view.obligation ? 'Original payment day' : 'Pay before'}</dt>
						<dd>{view.request.due_date}</dd>
					</div>
					<div>
						<dt>{view.obligation ? 'Original bank debit date' : 'Bank debit after'}</dt>
						<dd>{timeLabel(view.request.collection_at)}</dd>
					</div>
					<div>
						<dt>Extra time</dt>
						<dd>{view.request.grace_hours} hours</dd>
					</div>
				</dl>
				{#if view.request.invoice_document_id}<p>
						<button type="button" onclick={openInvoice}>Open the invoice →</button>
					</p>{/if}
			</article>
			<article class="card">
				<h2>What has happened so far</h2>
				<p><strong>{productLabel(view.request.state)}</strong></p>
				{#if view.agreement?.document_hash}<details>
						<summary>Technical record (for reference)</summary>
						<p>Sale record code<br /><code>{view.agreement.document_hash}</code></p>
					</details>{/if}{#if view.obligation}<p>
						Money left<br /><strong><Money amountKobo={view.obligation.outstanding_kobo} /></strong>
					</p>
					<p>
						<a
							href={`/api/v1/organizations/${encodeURIComponent(organizationID)}/credit-requests/${encodeURIComponent(id)}/agreement-document`}
							target="_blank"
							rel="noreferrer">Print or save a copy of this sale →</a
						>
					</p>{/if}
			</article>
		</section>
		{#if view.request.state === 'DRAFT'}<section class="card action">
				<h2>Check it before you send</h2>
				<a
					href={`/workspace/sales/approvals?organization=${encodeURIComponent(organizationID)}&request=${encodeURIComponent(view.request.id)}`}
					>Ask for internal approval</a
				>
				<p>You can still change anything now. Once you send it, your customer must see exactly this sale.</p>
				<label
					>How much must they pay? (₦)<input
						disabled={busy || loading}
						bind:value={draftPrincipal}
						inputmode="decimal"
					/></label
				><label>Goods<textarea disabled={busy || loading} bind:value={draftGoods}></textarea></label><label
					>Day they must pay<input disabled={busy || loading} type="date" bind:value={draftDueDate} /></label
				><label
					>Bank debit may be considered from (Nigerian time)<input
						disabled={busy || loading}
						type="datetime-local"
						bind:value={draftCollectionAt}
					/></label
				><label
					>Extra hours you are giving them<input
						disabled={busy || loading}
						type="number"
						min="0"
						max="720"
						bind:value={draftGraceHours}
					/></label
				>{#if draftChanged()}<p>Save your changes before sending.</p>{/if}
				<div class="button-row">
					<button disabled={busy || loading} onclick={updateDraft}>Save for later</button><button
						class="primary"
						disabled={busy || loading || draftChanged()}
						onclick={() => command('send')}>Send it to my customer</button
					><button class="danger" disabled={busy || loading} onclick={() => command('cancel')}>Cancel this draft</button
					>
				</div>
			</section>{/if}
		{#if view.request.state === 'SENT' || view.request.state === 'BUYER_REVIEWING'}<section class="card action">
				<h2>Waiting for your customer</h2>
				<p>Changed your mind? You can cancel it. The record of what happened stays.</p>
				<button class="danger" disabled={busy || loading} onclick={() => command('cancel')}>Cancel this sale</button>
			</section>{/if}
		{#if view.request.state === 'READY_TO_RELEASE'}<section class="card action">
				<h2>Have the goods been dispatched?</h2>
				<label
					>How did they get the goods?<select disabled={busy || loading} bind:value={deliveryMethod}
						><option value="supplier_delivery">We delivered them</option><option value="buyer_collection"
							>The customer came and collected</option
						><option value="third_party_delivery">Somebody else delivered them</option></select
					></label
				><label>Delivery note number<textarea disabled={busy || loading} bind:value={releaseNotes}></textarea></label
				><button
					class="primary"
					disabled={busy || loading}
					onclick={() => command('release', { delivery_method: deliveryMethod, notes: releaseNotes })}
					>Yes, the goods have left</button
				>
			</section>{/if}
		{#if view.obligation}
			<section class="card action">
				<h2>Enter money you have received</h2>
				<p>Only enter money that has actually reached you. Open your bank app and see it first.</p>
				{#if paymentNotice}<p class="success" role="status">{paymentNotice}</p>{/if}<label
					>How did they pay you?<select disabled={busy || loading} bind:value={paymentSource}
						><option value="supplier_recorded_transfer">Bank transfer or POS</option><option value="cash_recorded"
							>Cash</option
						></select
					></label
				><label
					>When did the money reach you? (Nigerian time)<input
						type="datetime-local"
						disabled={busy || loading}
						bind:value={paymentPaidAt}
						required
					/></label
				><label
					>How much did you receive? (₦)<input
						disabled={busy || loading}
						bind:value={paymentAmount}
						inputmode="decimal"
					/></label
				><label
					>{paymentSource === 'cash_recorded' ? 'Receipt or note' : 'Transfer or POS number'}
					<small>optional</small><input disabled={busy || loading} bind:value={paymentReference} /></label
				>{#if paymentSource === 'cash_recorded'}<p class="warning">
						If they paid cash, give them a receipt before you save this.
					</p>{/if}<button class="primary" disabled={busy || loading} onclick={recordPayment}>Save this payment</button>
				<h3>Payments so far</h3>
				{#if loading}<p role="status">Checking payment history…</p>{:else if paymentError}<p role="alert">
						{paymentError}
					</p>
					<button onclick={() => load()}>Check payment history again</button>{:else if payments.length}<ul
						class="plain-list"
					>
						{#each payments as payment, i (i)}<li>
								<span><Money amountKobo={payment.amount_kobo} /> · {productLabel(payment.source_type)}</span><strong
									>{productLabel(payment.state)}</strong
								>
							</li>{/each}
					</ul>{:else}<p>No payment yet.</p>{/if}
			</section>

			<details class="card action">
				<summary><strong>Payment days</strong> <span class="hint">the agreed dates</span></summary>{#if loading}<p>
						Checking the payment schedule…
					</p>{:else if scheduleError}<p role="alert">{scheduleError}</p>{:else if scheduleItems.length}<div
						class="table-wrap"
					>
						<table>
							<thead><tr><th>Pay before</th><th>Expected</th><th>Paid</th><th>Now</th></tr></thead><tbody
								>{#each scheduleItems as item, i (i)}<tr
										><td>{dateLabel(item.due_at.slice(0, 10))}</td><td
											><Money amountKobo={item.principal_due_kobo} /></td
										><td><Money amountKobo={item.allocated_kobo} /></td><td>{productLabel(item.state)}</td></tr
									>{/each}</tbody
							>
						</table>
					</div>{:else}<p>No scheduled payments are present in this checked record.</p>{/if}
				<p class="muted">
					These are the payment days as they stand. To change a date, somebody at Kredit must review it and your
					customer must agree to it. Contact support to ask.
				</p>
			</details>

			{#if claimError}<p role="alert">{claimError}</p>{/if}
			{#if paymentClaims.length}<section class="card action">
					<h2>Transfers your customer says they sent</h2>
					<p>
						Open your bank and verify the amount, reference and payment date first. Reject an incorrect report so the
						customer can correct it. The moment you accept it, the balance drops.
					</p>
					{#if paymentClaims.length}<div class="claim-list">
							{#each paymentClaims as claim, i (i)}<article>
									<div>
										<strong><Money amountKobo={claim.amount_kobo} /></strong><span
											>{productLabel(claim.state)} · {claim.transfer_reference}</span
										><small>Customer’s payment date: {timeLabel(claim.paid_at)}</small>
									</div>
									{#if ['pending', 'expired'].includes(claim.state?.toLowerCase())}<label
											>Why are you accepting or rejecting this?<input
												disabled={busy || loading}
												bind:value={claimReviewReason}
												placeholder="For example: Seen in our bank account"
											/></label
										>
										<div class="button-row">
											<button
												class="primary"
												disabled={busy || loading}
												onclick={() => decidePaymentClaim(claim, 'confirmed')}>Yes, this money reached me</button
											><button disabled={busy || loading} onclick={() => decidePaymentClaim(claim, 'rejected')}
												>No, I never received it</button
											>
										</div>{:else if claim.review_reason}<p>{claim.review_reason}</p>{/if}
								</article>{/each}
						</div>{:else}<p>Your customer has not reported any transfer.</p>{/if}
				</section>{/if}

			<details class="card action" open={eligibility?.eligible}>
				<summary
					><strong>Ask their bank for the money</strong>
					<span class="hint"
						>{eligibility?.eligible
							? 'available now'
							: eligibilityError || loading
								? 'status unavailable'
								: 'not available yet'}</span
					></summary
				>{#if loading}<p>Checking bank debit eligibility…</p>{:else if eligibilityError}<p role="alert">
						{eligibilityError}
					</p>{:else if eligibility?.eligible}<p>
						The currently eligible amount is <strong><Money amountKobo={eligibility.amount_kobo} /></strong> under the customer’s
						bank permission. Collection is not guaranteed.
					</p>
					<button class="primary" disabled={busy || loading} onclick={startCollection}>Ask the bank now</button
					>{:else}<p>Bank debit cannot start yet.</p>
					{#if eligibility?.reasons?.length}<ul>
							{#each eligibility.reasons as reason, i (i)}<li>{productLabel(reason)}</li>{/each}
						</ul>{/if}{/if}{#if collectionError}<p role="alert">
						{collectionError}
					</p>{/if}{#if collectionAttempts.length}<h3>What happened with bank debits</h3>
					<ul class="plain-list">
						{#each collectionAttempts as attempt, i (i)}<li>
								<span
									><Money amountKobo={attempt.requested_amount_kobo} /> · {productLabel(attempt.state)}<small
										>{timeLabel(attempt.requested_at)}</small
									></span
								><span class="button-row"
									>{#if attempt.state === 'FAILED'}<button
											disabled={busy || loading}
											onclick={() => collectionAction(attempt, 'retry')}>Try again</button
										>{/if}{#if attempt.state === 'SUBMITTED' || attempt.state === 'UNKNOWN'}<button
											disabled={busy || loading}
											onclick={() => collectionAction(attempt, 'reconcile')}>Check what happened</button
										>{/if}</span
								>
							</li>{/each}
					</ul>{/if}
			</details>

			<details class="card action">
				<summary><strong>Fix a mistake</strong></summary>
				<p>Only use this when something you saved is genuinely wrong. Every change you make here is recorded.</p>
				<h3>Remove a payment entered by mistake</h3>
				<label
					>Payment<select disabled={busy || loading} bind:value={reversePaymentID}
						><option value="">Choose a payment</option
						>{#each payments.filter((payment) => payment.state?.toUpperCase() === 'RECOGNIZED') as payment (payment.id)}<option
								value={payment.id}
								>{formatKobo(payment.amount_kobo)} · {payment.provider_reference ||
									productLabel(payment.source_type)}</option
							>{/each}</select
					></label
				><label
					>Why is it wrong?<textarea disabled={busy || loading} bind:value={reverseReason} rows="3"></textarea></label
				><button disabled={busy || loading} onclick={reversePayment}>Remove this payment</button>
				<hr />
				<h3>Reduce what is owed on this sale</h3>
				<label
					>What do you want to reduce?<select disabled={busy || loading} bind:value={operationType}
						><option value="write-off">The money the customer owes</option><option value="fee-waiver"
							>A Kredit fee</option
						></select
					></label
				><label>Amount (₦)<input disabled={busy || loading} bind:value={operationAmount} inputmode="decimal" /></label
				><label>Reason<textarea disabled={busy || loading} bind:value={operationReason} rows="3"></textarea></label>
				<p>
					Larger corrections are proposed and recorded in Admin, under Financial changes. This form handles the smaller
					ones.
				</p>
				<button disabled={busy || loading} onclick={changeMoney}>Save this change</button>
			</details>

			<details class="card action">
				<summary><strong>Report a problem</strong> <span class="hint">goods or amount wrong</span></summary>
				<p>
					Say how much money is affected, and what went wrong. Only the disputed amount is held. A debit already
					submitted to the bank may still complete.
				</p>
				<form onsubmit={openDispute}>
					<label
						>How much money is affected? (₦)<input
							disabled={busy || loading}
							bind:value={disputeAmount}
							inputmode="decimal"
							required
						/></label
					><label
						>In one line, what is wrong?<input disabled={busy || loading} bind:value={disputeReason} required /></label
					><label
						>Tell us the full story<textarea disabled={busy || loading} bind:value={disputeExplanation} rows="4"
						></textarea></label
					><label
						>While this is sorted out, what should happen to bank debit?<select
							disabled={busy || loading}
							bind:value={disputeEffect}><option value="CONTESTED_ONLY">Hold only the money in question</option></select
						></label
					><button class="primary" disabled={busy || loading}>Report this problem</button>
				</form>
			</details>
		{/if}
	{/if}
</section>
{#if review}<PaymentReview
		amount={review.claim.amount_kobo}
		reference={review.claim.transfer_reference}
		decision={review.decision}
		{busy}
		{error}
		onconfirm={confirmClaim}
		oncancel={() => {
			if (!busy) {
				review = null;
				error = '';
			}
		}}
	/>{/if}

<style>
	details.action > summary {
		cursor: pointer;
		list-style: none;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		min-height: 2.75rem;
		font-size: 1rem;
	}
	details.action > summary::-webkit-details-marker {
		display: none;
	}
	details.action > summary::after {
		content: '+';
		font-weight: 700;
		color: var(--color-accent, var(--color-primary));
	}
	details.action[open] > summary::after {
		content: '\2013';
	}
	details.action > summary .hint {
		font-size: 0.78rem;
		font-weight: 600;
		color: var(--color-muted, var(--color-foreground));
	}
	details.action[open] > summary {
		margin-bottom: 1rem;
		padding-bottom: 0.75rem;
		border-bottom: 1px solid var(--color-border, var(--color-border));
	}
	.detail-grid {
		display: grid;
		grid-template-columns: 2fr 1fr;
		gap: 1rem;
		margin: 1.5rem 0;
	}
	.card {
		padding: 1.5rem;
	}
	.notice {
		padding: 0.8rem 1rem;
		background: var(--color-background);
	}
	.quick-actions {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.7rem;
	}
	.repeat {
		display: inline-flex;
		align-items: center;
		min-height: 2.5rem;
		padding: 0.35rem 0.75rem;
		border: 1px solid var(--color-border);
		font-weight: 750;
		text-decoration: none;
	}
	dl div {
		display: flex;
		justify-content: space-between;
		gap: 2rem;
		border-bottom: 1px solid var(--color-border);
		padding: 0.7rem 0;
	}
	dd {
		margin: 0;
		text-align: right;
		font-weight: 700;
	}
	.action,
	.action form {
		display: grid;
		gap: 0.8rem;
		margin: 1rem 0;
	}
	.action form {
		margin: 0;
	}
	.action label {
		display: grid;
		gap: 0.35rem;
		font-weight: 700;
	}
	.action input,
	.action select,
	.action textarea {
		padding: 0.75rem;
		border: 1px solid var(--color-border);
		border-radius: 0.7rem;
		font: inherit;
	}
	.button-row {
		display: flex;
		flex-wrap: wrap;
		gap: 0.7rem;
	}
	.danger {
		color: var(--color-destructive);
		border-color: var(--color-destructive);
		background: var(--color-surface);
	}
	.warning {
		padding: 0.75rem;
		border-left: 4px solid var(--color-warning);
		background: var(--color-surface);
	}
	.success {
		color: var(--color-positive);
		font-weight: 750;
	}
	.table-wrap {
		overflow-x: auto;
	}
	table {
		width: 100%;
		border-collapse: collapse;
	}
	th,
	td {
		text-align: left;
		padding: 0.75rem;
		border-bottom: 1px solid var(--color-border);
		white-space: nowrap;
	}
	.plain-list,
	.claim-list {
		display: grid;
		gap: 0.65rem;
		padding: 0;
		list-style: none;
	}
	.plain-list li,
	.claim-list article {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding: 0.8rem;
		border: 1px solid var(--color-border);
		border-radius: 0.75rem;
	}
	.plain-list span,
	.claim-list article > div {
		display: grid;
		gap: 0.2rem;
	}
	.claim-list article {
		align-items: stretch;
		display: grid;
	}
	.claim-list small,
	.plain-list small {
		display: block;
		color: var(--color-muted);
	}
	details summary {
		cursor: pointer;
	}
	hr {
		width: 100%;
		border: 0;
		border-top: 1px solid var(--color-border);
	}
	code {
		overflow-wrap: anywhere;
	}
	@media (max-width: 720px) {
		.detail-grid {
			grid-template-columns: 1fr;
		}
		.plain-list li {
			align-items: flex-start;
			flex-direction: column;
		}
	}
</style>
