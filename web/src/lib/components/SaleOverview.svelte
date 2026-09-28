<script lang="ts">
	import { onDestroy } from 'svelte';
	import { checkedJSON, LatestRequest, record, text } from '$lib/api/reliable';
	import { readableDate } from '$lib/datetime';
	import { timeLabel, type SaleView } from '$lib/records';
	import { productLabel } from '$lib/product-language';
	import Money from '$lib/components/Money.svelte';
	let {
		view,
		organizationID,
		disabled = false
	} = $props<{ view: SaleView; organizationID: string; disabled?: boolean }>();
	const reads = new LatestRequest();
	onDestroy(() => reads.cancel());
	let error = $state(''),
		opening = $state(false);
	async function openInvoice() {
		const documentID = view.request.invoice_document_id;
		if (disabled || opening || !documentID) return;
		const request = reads.begin();
		error = '';
		opening = true;
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
				},
				{ signal: request.signal }
			);
			if (request.current()) window.location.assign(url);
		} catch {
			if (request.current()) error = 'We could not open that invoice. Please try again.';
		} finally {
			opening = false;
		}
	}
</script>

<section class="detail-grid">
	<article class="card">
		<h2>The sale</h2>
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
				<dd>{readableDate(view.request.due_date)}</dd>
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
				<button type="button" disabled={disabled || opening} onclick={openInvoice}>Open the invoice →</button>
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
					href={`/api/v1/organizations/${encodeURIComponent(organizationID)}/credit-requests/${encodeURIComponent(view.request.id)}/agreement-document`}
					target="_blank"
					rel="noreferrer">Print or save a copy of this sale →</a
				>
			</p>{/if}
	</article>
</section>
{#if error}<p role="alert">{error}</p>{/if}

<style>
	.detail-grid {
		display: grid;
		grid-template-columns: 2fr 1fr;
		gap: 1rem;
		margin: 1.5rem 0;
	}
	.card {
		padding: 1.5rem;
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
	code {
		overflow-wrap: anywhere;
	}
	@media (max-width: 720px) {
		.detail-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
