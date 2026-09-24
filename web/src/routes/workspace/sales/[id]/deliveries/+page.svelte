<script lang="ts">
	import { readableDate } from '$lib/datetime';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { formatKobo, parseNaira, type KoboValue } from '$lib/money';
	import { checkedJSON, LatestRequest, record, rows, text } from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';
	import { kobo } from '$lib/records';

	const orderID = $derived(page.params.id ?? '');
	const organizationID = $derived(page.url.searchParams.get('organization') ?? '');

	type LineItem = {
		id: string;
		sku: string;
		description: string;
		unit_price_kobo: KoboValue;
		quantity: number;
		fulfilled_quantity: number;
		total_kobo: KoboValue;
	};

	type Shipment = {
		id: string;
		tracking_reference: string;
		carrier: string;
		dispatched_at: string;
		status: string;
	};

	type CreditNote = {
		id: string;
		amount_kobo: KoboValue;
		reason: string;
		issued_by: string;
		approved_by: string;
		status: string;
		created_at: string;
	};

	let loading = $state(true),
		busy = $state(false),
		error = $state(''),
		message = $state('');
	let lineItems = $state<LineItem[]>([]);
	let shipments = $state<Shipment[]>([]);
	let creditNotes = $state<CreditNote[]>([]);

	// Dispatch form state
	let newCarrier = $state(''),
		newTracking = $state(''),
		dispatchQuantities = $state<Record<string, number>>({});

	// Credit note form state
	let creditAmount = $state(''),
		creditReason = $state('');

	const reads = new LatestRequest();

	function decodeLineItem(value: unknown): LineItem {
		const r = record(value);
		return {
			id: text(r.id),
			sku: typeof r.sku === 'string' ? r.sku : '',
			description: text(r.description),
			unit_price_kobo: kobo(r.unit_price_kobo),
			quantity: Number(r.quantity),
			fulfilled_quantity: Number(r.fulfilled_quantity ?? 0),
			total_kobo: kobo(r.total_kobo)
		};
	}

	function decodeShipment(value: unknown): Shipment {
		const r = record(value);
		return {
			id: text(r.id),
			tracking_reference: typeof r.tracking_reference === 'string' ? r.tracking_reference : '',
			carrier: typeof r.carrier === 'string' ? r.carrier : '',
			dispatched_at: text(r.dispatched_at),
			status: text(r.status)
		};
	}

	function decodeCreditNote(value: unknown): CreditNote {
		const r = record(value);
		return {
			id: text(r.id),
			amount_kobo: kobo(r.amount_kobo),
			reason: text(r.reason),
			issued_by: text(r.issued_by),
			approved_by: typeof r.approved_by === 'string' ? r.approved_by : '',
			status: text(r.status),
			created_at: text(r.created_at)
		};
	}

	async function load() {
		const req = reads.begin();
		loading = true;
		error = '';
		try {
			if (!organizationID || !orderID) return;
			const base = `/api/v1/organizations/${encodeURIComponent(organizationID)}/credit-requests/${encodeURIComponent(orderID)}`;

			const res = await checkedJSON(
				`${base}/deliveries`,
				(val) => {
					const r = record(val);
					return {
						items: rows('line_items', decodeLineItem)(r),
						shipments: rows('shipments', decodeShipment)(r),
						credit_notes: rows('credit_notes', decodeCreditNote)(r)
					};
				},
				{ signal: req.signal }
			);

			if (!req.current()) return;
			lineItems = res.items;
			shipments = res.shipments;
			creditNotes = res.credit_notes;
		} catch (cause) {
			if (req.current()) error = cause instanceof Error ? cause.message : 'Deliveries could not be loaded.';
		} finally {
			if (req.current()) loading = false;
		}
	}

	async function dispatchShipment() {
		if (busy || !newCarrier.trim()) return;
		busy = true;
		error = '';
		message = '';
		try {
			const items = Object.entries(dispatchQuantities)
				.filter(([_, q]) => q > 0)
				.map(([id, q]) => ({ line_item_id: id, quantity: q }));

			const base = `/api/v1/organizations/${encodeURIComponent(organizationID)}/credit-requests/${encodeURIComponent(orderID)}`;
			await new MutationIntent(`shipment:${orderID}`, `${base}/shipments`).run(
				{
					carrier: newCarrier.trim(),
					tracking_reference: newTracking.trim(),
					items
				},
				record,
				'POST'
			);

			message = 'Shipment dispatched successfully.';
			newCarrier = '';
			newTracking = '';
			dispatchQuantities = {};
			await load();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Shipment could not be dispatched.';
		} finally {
			busy = false;
		}
	}

	async function createCreditNote() {
		const amount = parseNaira(creditAmount);
		if (amount <= 0 || !creditReason.trim()) {
			error = 'Enter a valid amount and reason for the credit note.';
			return;
		}
		busy = true;
		error = '';
		message = '';
		try {
			const base = `/api/v1/organizations/${encodeURIComponent(organizationID)}/credit-requests/${encodeURIComponent(orderID)}`;
			await new MutationIntent(`credit-note:${orderID}`, `${base}/credit-notes`).run(
				{
					amount_kobo: amount,
					reason: creditReason.trim()
				},
				record,
				'POST'
			);

			message = 'Credit note draft created. A second person must approve it before it takes effect.';
			creditAmount = '';
			creditReason = '';
			await load();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Credit note could not be created.';
		} finally {
			busy = false;
		}
	}

	onMount(() => {
		void load();
		return () => reads.cancel();
	});
</script>

<svelte:head>
	<title>Deliveries and credit notes — Kredit</title>
</svelte:head>

<main class="shell workspace deliveries-page">
	<div class="header-nav">
		<a href={`/workspace/sales/${orderID}?organization=${encodeURIComponent(organizationID)}`} class="back-link">
			← Back to the sale
		</a>
	</div>

	<p class="eyebrow">Sales · Deliveries</p>
	<h1>Deliveries and credit notes</h1>
	<p class="lede">
		Record each shipment, what the customer confirmed receiving, and any credit note that reduces what they owe.
	</p>

	{#if message}<p role="status" class="alert success">{message}</p>{/if}
	{#if error}<p role="alert" class="alert danger">{error}</p>{/if}

	{#if loading}
		<p role="status">Loading delivery details…</p>
	{:else}
		<!-- Section 1: Line Items -->
		<section class="card">
			<h2>Items on this order</h2>
			{#if !lineItems.length}
				<p class="empty-note">This sale is recorded as one total, without a list of separate items.</p>
			{:else}
				<div class="table-container">
					<table>
						<thead>
							<tr>
								<th>Item</th>
								<th>SKU</th>
								<th>Unit Price</th>
								<th>Ordered</th>
								<th>Fulfilled</th>
								<th>Total</th>
							</tr>
						</thead>
						<tbody>
							{#each lineItems as item, i (i)}
								<tr>
									<td><strong>{item.description}</strong></td>
									<td><code>{item.sku || '—'}</code></td>
									<td>{formatKobo(item.unit_price_kobo)}</td>
									<td>{item.quantity}</td>
									<td>
										<span class={item.fulfilled_quantity >= item.quantity ? 'badge success' : 'badge warning'}>
											{item.fulfilled_quantity} / {item.quantity}
										</span>
									</td>
									<td>{formatKobo(item.total_kobo)}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>

		<!-- Section 2: Partial Shipments -->
		<section class="card">
			<h2>Shipments</h2>
			{#if shipments.length}
				<div class="shipment-list">
					{#each shipments as s, i (i)}
						<div class="shipment-item">
							<div>
								<strong>{s.carrier}</strong> · <code>{s.tracking_reference || 'No tracking ref'}</code>
								<p class="muted">Dispatched {readableDate(s.dispatched_at)}</p>
							</div>
							<span class={`badge ${s.status === 'delivered' ? 'success' : 'info'}`}>{s.status}</span>
						</div>
					{/each}
				</div>
			{:else}
				<p class="empty-note">No shipments recorded yet.</p>
			{/if}

			<form
				onsubmit={(e) => {
					e.preventDefault();
					void dispatchShipment();
				}}
				class="dispatch-form"
			>
				<h3>Record a shipment</h3>
				<div class="form-row">
					<label
						>Who is carrying the goods<input
							bind:value={newCarrier}
							placeholder="For example: our own van"
							required
							disabled={busy}
						/></label
					>
					<label
						>Waybill or tracking number<input bind:value={newTracking} placeholder="Optional" disabled={busy} /></label
					>
				</div>
				{#if lineItems.length}
					<div class="qty-inputs">
						{#each lineItems as item (item.id)}
							<label>
								Ship qty for {item.description} (max {item.quantity - item.fulfilled_quantity})
								<input
									type="number"
									min="0"
									max={item.quantity - item.fulfilled_quantity}
									bind:value={dispatchQuantities[item.id]}
									disabled={busy}
								/>
							</label>
						{/each}
					</div>
				{/if}
				<button type="submit" disabled={busy || !newCarrier.trim()}>Save shipment</button>
			</form>
		</section>

		<!-- Section 3: Credit Notes -->
		<section class="card">
			<h2>Approved credit notes</h2>
			<p class="muted">
				A credit note reduces what the customer owes, for example after a return or a discount. A second person must
				approve it: whoever drafts a credit note cannot approve it.
			</p>

			{#if creditNotes.length}
				<div class="credit-notes-list">
					{#each creditNotes as cn, i (i)}
						<div class="credit-note-item">
							<div>
								<strong>{formatKobo(cn.amount_kobo)}</strong>: {cn.reason}
								<p class="muted">Created {readableDate(cn.created_at)}</p>
							</div>
							<span class={`badge ${cn.status === 'approved' ? 'success' : 'warning'}`}>{cn.status}</span>
						</div>
					{/each}
				</div>
			{:else}
				<p class="empty-note">No credit notes issued for this order.</p>
			{/if}

			<form
				onsubmit={(e) => {
					e.preventDefault();
					void createCreditNote();
				}}
				class="credit-form"
			>
				<h3>Draft a credit note</h3>
				<div class="form-row">
					<label
						>Amount (₦)<input
							bind:value={creditAmount}
							placeholder="e.g. 50,000.00"
							inputmode="decimal"
							required
							disabled={busy}
						/></label
					>
					<label
						>Reason<input
							bind:value={creditReason}
							placeholder="For example: 5 cartons returned damaged"
							required
							disabled={busy}
						/></label
					>
				</div>
				<button type="submit" disabled={busy || !creditAmount || !creditReason.trim()}>Save draft</button>
			</form>
		</section>
	{/if}
</main>

<style>
	.deliveries-page {
		max-width: 64rem;
	}
	.header-nav {
		margin-bottom: 1.5rem;
	}
	.back-link {
		color: var(--color-primary);
		font-weight: 500;
		text-decoration: none;
	}
	.back-link:hover {
		text-decoration: underline;
	}
	.card {
		padding: 1.5rem;
		margin: 1.5rem 0;
		border: 1px solid var(--color-border);
		border-radius: 1rem;
		background: var(--color-surface);
	}
	.table-container {
		overflow-x: auto;
		margin-top: 1rem;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.95rem;
	}
	th,
	td {
		padding: 0.75rem 1rem;
		text-align: left;
		border-bottom: 1px solid var(--color-border);
	}
	th {
		font-weight: 600;
		color: var(--color-text-muted);
	}
	.badge {
		display: inline-block;
		padding: 0.25rem 0.6rem;
		border-radius: 0.4rem;
		font-size: 0.8rem;
		font-weight: 600;
		background: var(--color-surface-hover);
	}
	.badge.success {
		background: #e6f4ea;
		color: #137333;
	}
	.badge.warning {
		background: #fef7e0;
		color: #b06000;
	}
	.badge.info {
		background: #e8f0fe;
		color: #1a73e8;
	}
	.muted,
	.empty-note {
		color: var(--color-text-muted);
		font-size: 0.9rem;
		margin-top: 0.5rem;
	}
	.shipment-list,
	.credit-notes-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		margin: 1rem 0;
	}
	.shipment-item,
	.credit-note-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.8rem 1rem;
		border: 1px solid var(--color-border);
		border-radius: 0.6rem;
		background: var(--color-surface-hover);
	}
	.dispatch-form,
	.credit-form {
		margin-top: 1.5rem;
		padding-top: 1.5rem;
		border-top: 1px solid var(--color-border);
	}
	.form-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
		margin: 1rem 0;
	}
	@media (max-width: 640px) {
		.form-row {
			grid-template-columns: 1fr;
		}
	}
	label {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		font-size: 0.9rem;
	}
	input,
	button {
		font: inherit;
		padding: 0.75rem 1rem;
		border-radius: 0.6rem;
		border: 1px solid var(--color-border);
	}
	input {
		background: var(--color-surface);
		color: inherit;
	}
	button {
		background: var(--color-primary);
		color: var(--color-on-primary);
		font-weight: 600;
		cursor: pointer;
		border: none;
		align-self: flex-start;
	}
	button:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}
	.alert {
		padding: 0.75rem 1rem;
		border-radius: 0.6rem;
		margin: 1rem 0;
	}
	.alert.success {
		background: #e6f4ea;
		color: #137333;
	}
	.alert.danger {
		background: #fce8e6;
		color: #c5221f;
	}
</style>
