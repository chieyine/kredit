<script lang="ts">
	import { page } from '$app/state';
	import { formatKobo, type KoboValue } from '$lib/money';
	import { organization, kobo, customer } from '$lib/records';
	import {
		checkedJSON,
		LatestRequest,
		optionalText,
		record,
		rows,
		text,
		publicError,
		RequestError
	} from '$lib/api/reliable';
	import { productLabel } from '$lib/product-language';
	import ShareActions from '$lib/components/ShareActions.svelte';
	import { readLocal, writeLocal } from '$lib/product-tools';
	type PaymentHistory = {
		current_active_principal_kobo: KoboValue;
		active_obligations: number;
		completed_obligations: number;
		on_time_percentage: number;
	};
	type StatementLine = {
		credit_request_id: string;
		buyer_name: string;
		payment_status: string;
		outstanding_kobo: KoboValue;
	};
	type Statement = { buyer_id: string; obligations: StatementLine[] };
	let history = $state<PaymentHistory | null>(null),
		statement = $state<Statement | null>(null);
	let businessName = $state(''),
		buyerUserID = $state('');
	let error = $state(''),
		organizationID = $state(''),
		note = $state(''),
		noteMessage = $state('');
	const requests = new LatestRequest(),
		money = formatKobo;
	function decodeHistory(value: unknown): PaymentHistory {
		const row = record(value);
		for (const key of ['active_obligations', 'completed_obligations'])
			if (!Number.isSafeInteger(row[key]) || Number(row[key]) < 0) throw new Error('Invalid sale count');
		if (
			typeof row.on_time_percentage !== 'number' ||
			!Number.isFinite(row.on_time_percentage) ||
			row.on_time_percentage < 0 ||
			row.on_time_percentage > 100
		)
			throw new Error('Invalid payment history');
		return {
			current_active_principal_kobo: kobo(row.current_active_principal_kobo),
			active_obligations: Number(row.active_obligations),
			completed_obligations: Number(row.completed_obligations),
			on_time_percentage: row.on_time_percentage
		};
	}
	function decodeStatement(value: unknown): Statement {
		const row = record(value);
		return {
			buyer_id: text(row.buyer_id),
			obligations: rows('obligations', (value): StatementLine => {
				const sale = record(value);
				return {
					credit_request_id: text(sale.credit_request_id),
					buyer_name: optionalText(sale.buyer_name),
					payment_status: text(sale.payment_status),
					outstanding_kobo: kobo(sale.outstanding_kobo)
				};
			})(row)
		};
	}
	async function loadCustomer(customerID = page.params.id, selected = page.url.searchParams.get('organization')) {
		const request = requests.begin();
		error = '';
		history = null;
		statement = null;
		organizationID = '';
		businessName = '';
		buyerUserID = '';
		note = '';
		noteMessage = '';
		try {
			const organizations = await checkedJSON('/api/v1/organizations', rows('organizations', organization), {
				signal: request.signal
			});
			const org = selected ? organizations.find((item) => item.id === selected) : organizations[0];
			if (!org) throw new RequestError('This business workspace is not available.', 404);
			const customers = await checkedJSON(
				`/api/v1/organizations/${encodeURIComponent(org.id)}/customers`,
				rows('customers', customer),
				{ signal: request.signal }
			);
			const match = customers.find((item) => item.buyer_business_id === customerID);
			if (!match) throw new RequestError('This customer business is not connected to your workspace.', 404);
			const base = `/api/v1/organizations/${encodeURIComponent(org.id)}/customers/${encodeURIComponent(match.buyer_user_id)}`;
			const query = `?buyer_business_id=${encodeURIComponent(customerID ?? '')}`;
			const [h, statementResult] = await Promise.all([
				checkedJSON(`${base}/history${query}`, decodeHistory, { signal: request.signal }),
				checkedJSON(`${base}/statement${query}`, decodeStatement, { signal: request.signal })
			]);
			if (!request.current()) return;
			if (statementResult.buyer_id !== match.buyer_user_id) throw new Error('Wrong customer statement');
			organizationID = org.id;
			buyerUserID = match.buyer_user_id;
			businessName = match.trading_name || match.legal_name;
			history = h;
			statement = statementResult;
			const saved = readLocal<unknown>(`kredit:customer-note:${org.id}:${customerID}`, '');
			note = typeof saved === 'string' ? saved : '';
		} catch (cause) {
			if (request.current()) error = publicError(cause, 'this customer');
		}
	}
	$effect(() => {
		void loadCustomer(page.params.id, page.url.searchParams.get('organization'));
		return () => requests.cancel();
	});
	function customerName() {
		return businessName || 'Customer relationship';
	}
	function saveNote() {
		if (!organizationID || !history) return;
		noteMessage = writeLocal(`kredit:customer-note:${organizationID}:${page.params.id}`, note.trim())
			? 'Saved on this phone. It is cleared when you sign out.'
			: 'This phone could not save the note. Copy it somewhere safe before leaving.';
	}
</script>

<svelte:head><title>Customer — Kredit</title></svelte:head>
<main class="shell workspace">
	<a href={`/workspace/partners/customers?organization=${encodeURIComponent(organizationID)}`}>← Customers</a>
	<p class="eyebrow">Customer</p>
	<h1>{customerName()}</h1>
	<p class="lede">
		You only see sales this customer made with you. No other seller can see your records, and you cannot see theirs.
	</p>
	{#if error}<p class="error" role="alert">
			{error} <button type="button" onclick={() => loadCustomer()}>Try again</button>
		</p>{:else if !history}<p>Opening this customer…</p>{:else}<div class="customer-actions">
			<a
				class="primary-link"
				href={`/workspace/sales/new?customer=${encodeURIComponent(buyerUserID)}&customer_business=${encodeURIComponent(page.params.id ?? '')}&organization=${encodeURIComponent(organizationID)}`}
				>Sell to them again</a
			><button type="button" onclick={() => window.print()}>Print this page</button>
		</div>
		<ShareActions
			title="Kredit customer statement"
			text={`Kredit statement: ${money(history.current_active_principal_kobo)} is still owed across ${history.active_obligations ?? 0} open sale(s). ${history.completed_obligations ?? 0} sale(s) fully paid.`}
		/>
		<section class="stats">
			<article><span>Sales still open</span><strong>{history.active_obligations ?? 0}</strong></article>
			<article>
				<span>Money they still owe you</span><strong>{money(history.current_active_principal_kobo)}</strong>
			</article>
			<article><span>Sales fully paid</span><strong>{history.completed_obligations ?? 0}</strong></article>
			<article>
				<span>Paid on time</span><strong
					>{history.completed_obligations
						? `${Number(history.on_time_percentage ?? 0).toFixed(0)}%`
						: 'No record yet'}</strong
				>
			</article>
		</section>
		<section class="card">
			<h2>Your private note</h2>
			<p>
				Directions to their shop, their usual order, anything you need to remember. The note stays on this phone, Kredit
				never receives it, and signing out removes it.
			</p>
			<label
				>Note about this customer<textarea
					bind:value={note}
					rows="3"
					placeholder="For example: delivers to the second shop on Mondays"
				></textarea></label
			><button type="button" onclick={saveNote}>Save note</button>{#if noteMessage}<small role="status"
					>{noteMessage}</small
				>{/if}
		</section>
		<section class="card">
			<h2>Sales with you</h2>
			{#if statement?.obligations?.length}<div class="table">
					{#each statement.obligations as obligation, i (i)}<a
							href={`/workspace/sales/${encodeURIComponent(obligation.credit_request_id)}?organization=${encodeURIComponent(organizationID)}`}
							><span>{obligation.buyer_name || 'Credit sale'}</span><strong>{money(obligation.outstanding_kobo)}</strong
							><small>{productLabel(obligation.payment_status)}</small></a
						>{/each}
				</div>{:else}<p>This customer has no open sale with you right now.</p>{/if}
		</section>{/if}
</main>

<style>
	.customer-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.7rem;
		margin-top: 1.25rem;
	}
	.customer-actions a,
	.customer-actions button,
	.card button {
		padding: 0.7rem 0.9rem;
		border: 1px solid var(--color-foreground);
		background: var(--color-surface);
		color: inherit;
		font: inherit;
		font-weight: 750;
		text-decoration: none;
	}
	.customer-actions .primary-link {
		border-color: var(--color-primary);
		background: var(--color-primary);
		color: var(--color-on-primary);
	}
	.stats {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(11rem, 1fr));
		gap: 1rem;
		margin: 2rem 0;
	}
	.stats article,
	.card {
		padding: 1.2rem;
		border: 1px solid var(--color-border);
		border-radius: 1rem;
		background: var(--color-surface);
	}
	.card {
		margin-top: 1rem;
	}
	.card label {
		display: grid;
		gap: 0.4rem;
		font-weight: 750;
	}
	.card textarea {
		box-sizing: border-box;
		width: 100%;
		margin: 0.4rem 0;
		padding: 0.75rem;
		border: 1px solid var(--color-border);
		font: inherit;
	}
	.stats span {
		display: block;
		color: var(--color-muted);
	}
	.stats strong {
		display: block;
		font-size: 1.45rem;
		margin-top: 0.4rem;
	}
	.table {
		display: grid;
	}
	.table a {
		display: grid;
		grid-template-columns: 1fr auto auto;
		gap: 1rem;
		padding: 0.9rem 0;
		border-bottom: 1px solid var(--color-border);
		color: inherit;
		text-decoration: none;
	}
	.table small {
		color: var(--color-muted);
	}
	.error {
		color: var(--color-overdue);
	}
	@media print {
		.customer-actions,
		:global(.share-actions) {
			display: none !important;
		}
	}
	@media (max-width: 600px) {
		.table a {
			grid-template-columns: 1fr auto;
		}
		.table small {
			grid-column: 1/-1;
		}
	}
</style>
