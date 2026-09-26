<script lang="ts">
	import { readableDate } from '$lib/datetime';
	import { checkedJSON, LatestRequest, record, rows, text, publicError } from '$lib/api/reliable';
	import { kobo } from '$lib/records';
	import { page } from '$app/state';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';
	import { exactKobo, formatKobo, type KoboValue } from '$lib/money';
	import { productLabel } from '$lib/product-language';

	type ScheduleItem = {
		id: string;
		state: string;
		due_at: string;
		collection_at: string;
		principal_due_kobo: KoboValue;
		allocated_kobo: KoboValue;
	};
	type CollectionNotice = { schedule_item_id: string; notification_id: string; acknowledged: boolean };
	type StateRow = { id: string; state: string };
	type ObligationDetail = {
		view: {
			request: { id: string; goods_description: string };
			obligation: { outstanding_kobo: KoboValue; payment_status: string };
		};
		schedule_items: ScheduleItem[];
		collection_notices: CollectionNotice[];
		payments: StateRow[];
		payment_claims: StateRow[];
	};
	let data = $state<ObligationDetail | null>(null);
	let error = $state('');
	let notice = $state('');
	let busy = $state('');

	const reads = new LatestRequest();
	// eslint-disable-next-line svelte/prefer-svelte-reactivity -- retry cache, never rendered
	const keys = new Map<string, string>();
	const stateRow = (value: unknown): StateRow => {
		const item = record(value);
		return { id: text(item.id), state: text(item.state) };
	};
	function decode(value: unknown): ObligationDetail {
		const body = record(value),
			view = record(body.view),
			request = record(view.request),
			obligation = record(view.obligation);
		return {
			view: {
				request: { id: text(request.id), goods_description: text(request.goods_description) },
				obligation: {
					outstanding_kobo: kobo(obligation.outstanding_kobo),
					payment_status: text(obligation.payment_status)
				}
			},
			schedule_items: rows('schedule_items', (value): ScheduleItem => {
				const item = record(value);
				const due = text(item.due_at),
					collection = text(item.collection_at);
				if (!Number.isFinite(Date.parse(due)) || !Number.isFinite(Date.parse(collection)))
					throw new Error('Invalid payment date');
				return {
					id: text(item.id),
					state: text(item.state),
					due_at: due,
					collection_at: collection,
					principal_due_kobo: kobo(item.principal_due_kobo),
					allocated_kobo: kobo(item.allocated_kobo)
				};
			})(body),
			collection_notices: rows('collection_notices', (value): CollectionNotice => {
				const item = record(value);
				if (typeof item.acknowledged !== 'boolean') throw new Error('Invalid notice state');
				return {
					schedule_item_id: text(item.schedule_item_id),
					notification_id: text(item.notification_id),
					acknowledged: item.acknowledged
				};
			})(body),
			payments: rows('payments', stateRow)(body),
			payment_claims: rows('payment_claims', stateRow)(body)
		};
	}
	/** Unpaid principal on a payment day, computed exactly; null when an amount cannot be verified. */
	function unpaid(item: ScheduleItem): bigint | null {
		const due = exactKobo(item.principal_due_kobo),
			paid = exactKobo(item.allocated_kobo);
		return due === null || paid === null ? null : due - paid;
	}
	const nextDue = $derived(data?.schedule_items.find((i) => i.state !== 'CANCELLED' && (unpaid(i) ?? 0n) > 0n) ?? null);
	const confirmedPayments = $derived(data?.payments.filter((p) => p.state === 'recognized').length ?? 0);
	const waitingClaims = $derived(
		data?.payment_claims.filter((p) => ['pending', 'expired'].includes(p.state)).length ?? 0
	);
	async function loadSale(id = page.params.id) {
		const request = reads.begin();
		error = '';
		data = null;
		notice = '';
		try {
			const result = await checkedJSON(`/api/v1/buyer/obligations/${encodeURIComponent(id ?? '')}`, decode, {
				signal: request.signal
			});
			if (request.current()) data = result;
		} catch (cause) {
			if (request.current()) error = publicError(cause, 'this sale');
		}
	}
	$effect(() => {
		const id = page.params.id;
		void loadSale(id);
		return () => reads.cancel();
	});

	async function acknowledgeNotice(itemID: string, notificationID: string) {
		if (busy) return;
		busy = itemID;
		const saleID = page.params.id;
		error = '';
		notice = '';
		try {
			const response = await fetch(
				`/api/v1/buyer/schedule-items/${encodeURIComponent(itemID)}/collection-notice/acknowledge`,
				{
					method: 'POST',
					credentials: 'include',
					signal: AbortSignal.timeout(20000),
					redirect: 'error',
					body: JSON.stringify({ notification_id: notificationID }),
					headers: {
						'Content-Type': 'application/json',
						'Idempotency-Key':
							keys.get(notificationID) ??
							(() => {
								const key = idempotencyKey();
								keys.set(notificationID, key);
								return key;
							})(),
						...csrfHeaders()
					}
				}
			);
			if (!response.ok) {
				const result = await response.json().catch(() => ({}));
				throw new Error(result.detail ?? 'We could not save your answer. Please try again.');
			}
			if (page.params.id !== saleID) return;
			keys.delete(notificationID);
			if (data)
				data.collection_notices = data.collection_notices.map((item) =>
					item.notification_id === notificationID ? { ...item, acknowledged: true } : item
				);
			notice =
				'Saved. This only says you saw the notice. It does not mean you have paid, and you can still report a problem.';
		} catch {
			if (page.params.id === saleID)
				error = 'We could not confirm your answer. Make sure you received the notice, then retry the same action.';
		} finally {
			busy = '';
		}
	}

	const money = formatKobo;
</script>

<svelte:head><title>Money you owe — Kredit</title></svelte:head>

<main class="shell workspace">
	<p class="eyebrow">Money I owe</p>
	{#if data}
		<h1>{data.view.request.goods_description}</h1>
		{#if notice}<p class="notice" role="status">{notice}</p>{/if}
		{#if error}<p role="alert">{error}</p>{/if}
		<section class="summary">
			<article><span>Money left to pay</span><strong>{money(data.view.obligation.outstanding_kobo)}</strong></article>
			<article>
				<span>Where this stands</span><strong>{productLabel(data.view.obligation.payment_status)}</strong>
			</article>
			<article>
				<span>Pay before</span><strong>{nextDue ? readableDate(nextDue.due_at) : 'Nothing due right now'}</strong>
			</article>
		</section>
		<h2>Your payment days</h2>
		{#if data.schedule_items.length}
			<div class="table">
				<table>
					<thead
						><tr
							><th>Pay before</th><th>Money to pay</th><th>Money paid</th><th>Where it stands</th><th>Debit notice</th
							></tr
						></thead
					><tbody>
						{#each data.schedule_items as item (item.id)}
							{@const debitNotice = data.collection_notices.find((n) => n.schedule_item_id === item.id)}
							<tr
								><td>{readableDate(item.due_at)}</td><td>{money(item.principal_due_kobo)}</td><td
									>{money(item.allocated_kobo)}</td
								><td>{productLabel(item.state)}</td><td
									>{#if debitNotice}<p>
											Debit date: {readableDate(item.collection_at)}. Up to {money(unpaid(item))} still due.
										</p>
										{#if debitNotice.acknowledged}<span>Notice acknowledged</span>{:else}<button
												class="secondary"
												disabled={Boolean(busy)}
												onclick={() => acknowledgeNotice(item.id, debitNotice.notification_id)}
												>{busy === item.id ? 'Saving…' : 'I have seen this notice'}</button
											>{/if}{:else if item.state === 'PAID' || item.state === 'CANCELLED'}Not needed{:else}No delivered
										debit notice yet{/if}</td
								></tr
							>
						{/each}
					</tbody>
				</table>
			</div>
		{:else}<p>You pay this sale once, all at one time.</p>{/if}
		<h2>What you have paid</h2>
		<p>
			{confirmedPayments}
			{confirmedPayments === 1 ? 'payment' : 'payments'} confirmed. {waitingClaims} still waiting for the seller to check
			their bank.
		</p>
		<a href={`/workspace/purchases/orders/${encodeURIComponent(data.view.request.id)}`}
			>Pay this sale, or report a problem →</a
		>
	{:else if error}<h1>We could not open this sale.</h1>
		<p role="alert">{error}</p>
		<button type="button" onclick={() => loadSale()}>Try again</button>
	{:else}<p>Opening this sale…</p>{/if}
</main>

<style>
	.summary {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 1rem;
	}
	.summary article {
		display: grid;
		gap: 0.5rem;
		padding: 1rem;
		border: 1px solid var(--color-border);
		border-radius: 1rem;
		background: var(--color-surface);
	}
	.summary span,
	th {
		color: var(--color-muted);
	}
	.table {
		overflow-x: auto;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		background: var(--color-surface);
	}
	th,
	td {
		padding: 0.8rem;
		text-align: left;
		border-bottom: 1px solid var(--color-border);
	}
	.notice {
		padding: 0.8rem;
		border-radius: 0.75rem;
		background: var(--color-background);
		color: var(--color-positive);
	}
	@media (max-width: 700px) {
		.summary {
			grid-template-columns: 1fr;
		}
	}
</style>
