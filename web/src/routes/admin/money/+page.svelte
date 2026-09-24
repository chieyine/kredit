<script lang="ts">
	import { readableDateTime } from '$lib/datetime';
	import { checkedJSON, optionalNumber, optionalText, publicError, record, rows, text } from '$lib/api/reliable';
	import type { KoboValue } from '$lib/money';
	import { kobo } from '$lib/records';
	import { onMount } from 'svelte';
	import Money from '$lib/components/Money.svelte';
	type MoneySummary = {
		received_kobo: KoboValue;
		payment_count: number;
		outstanding_kobo: KoboValue;
		collection_requested_kobo: KoboValue;
		collection_succeeded_kobo: KoboValue;
		[key: string]: unknown;
	};
	type MoneyActivity = {
		id: string;
		kind: string;
		amount_kobo: KoboValue;
		state: string;
		reference: string;
		occurred_at: string;
	};
	let summary = $state<MoneySummary | null>(null),
		activity = $state<MoneyActivity[]>([]),
		loading = $state(true),
		error = $state('');
	async function load() {
		loading = true;
		error = '';
		try {
			const data = await checkedJSON('/api/v1/ops/money?limit=100', (value) => {
				const body = record(value),
					totals = record(body.summary);
				return {
					summary: {
						...totals,
						received_kobo: kobo(totals.received_kobo),
						payment_count: optionalNumber(totals.payment_count),
						outstanding_kobo: kobo(totals.outstanding_kobo),
						collection_requested_kobo: kobo(totals.collection_requested_kobo),
						collection_succeeded_kobo: kobo(totals.collection_succeeded_kobo)
					},
					activity: rows('activity', (value): MoneyActivity => {
						const item = record(value);
						return {
							id: text(item.id),
							kind: text(item.kind),
							amount_kobo: kobo(item.amount_kobo),
							state: text(item.state),
							reference: optionalText(item.reference),
							occurred_at: optionalText(item.occurred_at)
						};
					})(body)
				};
			});
			summary = data.summary;
			activity = data.activity;
		} catch (cause) {
			error = publicError(cause, 'money activity');
		} finally {
			loading = false;
		}
	}
	onMount(load);
</script>

<svelte:head><title>Money — Kredit admin</title></svelte:head>
<main class="shell workspace money">
	<header>
		<p class="eyebrow">Admin / Money</p>
		<h1>Money</h1>
		<p>Confirmed payments, bank-debit attempts and customer balances. Nothing here is an estimate from a bank alert.</p>
	</header>
	{#if error}<section role="alert">
			<p class="error">{error}</p>
			<button type="button" onclick={load}>Try again</button>
		</section>{:else if loading}<p>Checking the latest money records…</p>{:else if summary}<section class="totals">
			<article>
				<span>Money received</span><strong><Money amountKobo={summary.received_kobo} /></strong><small
					>{summary.payment_count} payment records</small
				>
			</article>
			<article>
				<span>Money still owed</span><strong><Money amountKobo={summary.outstanding_kobo} /></strong><small
					>Across all open sales</small
				>
			</article>
			<article>
				<span>Debits requested</span><strong><Money amountKobo={summary.collection_requested_kobo} /></strong><small
					>Every bank-debit attempt</small
				>
			</article>
			<article>
				<span>Debits that worked</span><strong><Money amountKobo={summary.collection_succeeded_kobo} /></strong><small
					>Confirmed by the payment company</small
				>
			</article>
		</section>
		<section class="activity">
			<div>
				<p class="eyebrow">Latest activity</p>
				<h2>Payments and bank debits</h2>
			</div>
			<div class="table-wrap">
				<table>
					<thead><tr><th>Type</th><th>Amount</th><th>Status</th><th>Reference</th><th>Time</th></tr></thead><tbody
						>{#each activity as item (`${item.kind}:${item.id}`)}<tr
								><td>{item.kind === 'payment' ? 'Payment' : 'Bank debit'}</td><td
									><strong><Money amountKobo={item.amount_kobo} /></strong></td
								><td>{item.state.replaceAll('_', ' ')}</td><td><code>{item.reference}</code></td><td
									>{readableDateTime(item.occurred_at)}</td
								></tr
							>{/each}</tbody
					>
				</table>
			</div>
		</section>{/if}
</main>

<style>
	.money > header {
		padding: 2rem 0;
		border-bottom: 3px solid var(--color-primary);
	}
	.money h1 {
		max-width: 15ch;
		margin: 0.4rem 0;
		font-family: var(--font-serif);
		font-size: clamp(1.9rem, 3.4vw, 2.6rem);
		line-height: 1.1;
	}
	.money header p {
		max-width: 42rem;
	}
	.totals {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		margin: 2rem 0 4rem;
		border-top: 1px solid var(--color-border);
		border-left: 1px solid var(--color-border);
	}
	.totals article {
		display: grid;
		gap: 0.6rem;
		min-height: 9rem;
		padding: 1rem;
		border-right: 1px solid var(--color-border);
		border-bottom: 1px solid var(--color-border);
		background: var(--color-surface);
	}
	.totals article > strong {
		font-family: var(--font-serif);
		font-size: 1.65rem;
	}
	.totals small {
		align-self: end;
		color: var(--color-foreground);
	}
	.activity > div:first-child {
		border-bottom: 3px solid var(--color-primary);
	}
	.activity h2 {
		font-family: var(--font-serif);
		font-size: 2.3rem;
		font-weight: 500;
	}
	.table-wrap {
		overflow-x: auto;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		background: var(--color-surface);
	}
	th,
	td {
		padding: 1rem;
		text-align: left;
		border-bottom: 1px solid var(--color-border);
	}
	code {
		font-size: 0.72rem;
	}
	@media (max-width: 850px) {
		.totals {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}
	@media (max-width: 440px) {
		.totals {
			grid-template-columns: 1fr;
		}
	}
</style>
