<script lang="ts">
	import { checkedJSON, optionalText, publicError, record, rows, text, LatestRequest } from '$lib/api/reliable';
	import { onMount } from 'svelte';
	type Metric = {
		key: string;
		label: string;
		unit: string;
		definition: string;
		source: string;
		value: number;
		target_status: string;
	};
	type Feedback = Record<'total' | 'yes' | 'partly' | 'no' | 'seller' | 'buyer' | 'clear_percent', number>;
	type ReconciliationRow = { event: string; status: string; source_count: number; event_count: number };
	type Scorecard = {
		generated_at: string;
		from: string;
		to: string;
		refresh_mode: string;
		reconciliation_ok: boolean;
		kpis: Metric[];
		drivers: Metric[];
		guardrails: Metric[];
		feedback: Feedback;
		reconciliation: ReconciliationRow[];
	};
	let scorecard: Scorecard | null = null;
	let error = '';
	let loading = true;
	let to = new Date().toISOString().slice(0, 10);
	let from = new Date(Date.now() - 7 * 86400000).toISOString().slice(0, 10);
	let organization = '';
	const reads = new LatestRequest();
	const finite = (value: unknown, message: string) => {
		if (typeof value !== 'number' || !Number.isFinite(value)) throw new Error(message);
		return value;
	};
	function metric(value: unknown): Metric {
		const item = record(value);
		return {
			key: text(item.key),
			label: text(item.label),
			unit: text(item.unit),
			definition: text(item.definition),
			source: text(item.source),
			value: finite(item.value, 'Invalid metric'),
			target_status: optionalText(item.target_status)
		};
	}
	function decodeScorecard(value: unknown): Scorecard {
		const card = record(record(value).scorecard);
		const date = (key: string) => {
			const raw = text(card[key]);
			if (!Number.isFinite(Date.parse(raw))) throw new Error('Invalid scorecard date');
			return raw;
		};
		if (typeof card.reconciliation_ok !== 'boolean') throw new Error('Missing reconciliation result');
		const answers = record(card.feedback);
		const count = (key: keyof Feedback) => {
			const n = finite(answers[key], 'Invalid feedback');
			if (n < 0) throw new Error('Invalid feedback');
			return n;
		};
		return {
			generated_at: date('generated_at'),
			from: date('from'),
			to: date('to'),
			refresh_mode: text(card.refresh_mode),
			reconciliation_ok: card.reconciliation_ok,
			kpis: rows('kpis', metric)(card),
			drivers: rows('drivers', metric)(card),
			guardrails: rows('guardrails', metric)(card),
			feedback: {
				total: count('total'),
				yes: count('yes'),
				partly: count('partly'),
				no: count('no'),
				seller: count('seller'),
				buyer: count('buyer'),
				clear_percent: count('clear_percent')
			},
			reconciliation: rows('reconciliation', (value): ReconciliationRow => {
				const row = record(value);
				return {
					event: text(row.event),
					status: text(row.status),
					source_count: finite(row.source_count, 'Invalid reconciliation count'),
					event_count: finite(row.event_count, 'Invalid reconciliation count')
				};
			})(card)
		};
	}
	const format = (metric: Metric) =>
		metric.target_status === 'no_data'
			? 'No data yet'
			: metric.unit === 'percent'
				? `${metric.value.toFixed(1)}%`
				: metric.unit === 'kobo'
					? new Intl.NumberFormat('en-NG', { style: 'currency', currency: 'NGN' }).format(metric.value / 100)
					: metric.unit === 'cases_per_100_active_suppliers'
						? `${metric.value.toFixed(1)} per 100`
						: `${metric.value.toFixed(metric.unit === 'hours' || metric.unit === 'days' ? 1 : 0)} ${metric.unit}`;
	const findMetric = (key: string) =>
		[...(scorecard?.kpis ?? []), ...(scorecard?.drivers ?? []), ...(scorecard?.guardrails ?? [])].find(
			(item) => item.key === key
		);
	const showMetric = (key: string) => {
		const item = findMetric(key);
		return item ? format(item) : 'Not measured yet';
	};
	const feedbackValue = () =>
		scorecard?.feedback?.total ? `${scorecard.feedback.clear_percent.toFixed(1)}%` : 'No answers yet';
	async function load() {
		const request = reads.begin();
		loading = true;
		error = '';
		// eslint-disable-next-line svelte/prefer-svelte-reactivity -- built and consumed within one request
		const params = new URLSearchParams({ from, to });
		if (organization.trim()) params.set('organization_id', organization.trim());
		try {
			const result = await checkedJSON(`/api/v1/ops/analytics/scorecard?${params}`, decodeScorecard, {
				signal: request.signal
			});
			if (request.current()) scorecard = result;
		} catch (cause) {
			if (request.current()) error = publicError(cause, 'the scorecard');
		} finally {
			if (request.current()) loading = false;
		}
	}
	onMount(() => {
		void load();
		return () => reads.cancel();
	});
</script>

<svelte:head><title>Application evidence — Kredit</title></svelte:head>
<main class="shell workspace analytics">
	<p class="eyebrow">Operations</p>
	<h1>Application evidence</h1>
	<p class="lede">
		Review figures calculated from Kredit records for the selected period. Check each measure’s definition and
		reconciliation status before sharing it.
	</p>
	<form
		onsubmit={(event) => {
			event.preventDefault();
			load();
		}}
		aria-label="Scorecard filters"
	>
		<label>From<input type="date" bind:value={from} required /></label>
		<label>To<input type="date" bind:value={to} required /></label>
		<label>Supplier organisation UUID (optional)<input bind:value={organization} autocomplete="off" /></label>
		<button disabled={loading}>{loading ? 'Refreshing…' : 'Apply filters'}</button>
	</form>
	{#if error}<section role="alert">
			<p class="error">{error}</p>
			<button type="button" onclick={load}>Try again</button>
		</section>{:else if loading}<p role="status">Calculating the live scorecard…</p>{:else if scorecard}
		<div class="status" class:healthy={scorecard.reconciliation_ok}>
			<strong>{scorecard.reconciliation_ok ? 'Reconciled' : 'Review required'}</strong><span
				>{scorecard.refresh_mode} · generated {new Date(scorecard.generated_at).toLocaleString()}</span
			>
		</div>
		<section class="application-snapshot" aria-labelledby="application-title">
			<header>
				<div>
					<p class="eyebrow">Application snapshot</p>
					<h2 id="application-title">
						Evidence for {new Date(scorecard.from).toLocaleDateString()} to {new Date(
							scorecard.to
						).toLocaleDateString()}
					</h2>
				</div>
				<button type="button" onclick={() => window.print()}>Print or save as PDF</button>
			</header>
			<div class="proof-grid">
				<article>
					<span>Trade credit recorded</span><strong>{showMetric('gross_trade_credit_volume')}</strong><small
						>Actual sales activated in Kredit</small
					>
				</article>
				<article>
					<span>Businesses using Kredit</span><strong>{showMetric('active_suppliers')}</strong><small
						>Suppliers with an active sale</small
					>
				</article>
				<article>
					<span>Sales accepted</span><strong>{showMetric('sent_to_acceptance')}</strong><small
						>Sent sales that customers accepted</small
					>
				</article>
				<article>
					<span>Customers who bought again</span><strong>{showMetric('repeat_sale_rate')}</strong><small
						>Supplier-customer pairs with another sale</small
					>
				</article>
				<article>
					<span>Suppliers who returned</span><strong>{showMetric('supplier_retention')}</strong><small
						>Active in this period and the last one</small
					>
				</article>
				<article>
					<span>Page was easy to understand</span><strong>{feedbackValue()}</strong><small
						>{scorecard.feedback.total} real {scorecard.feedback.total === 1 ? 'answer' : 'answers'} · {scorecard
							.feedback.seller} seller · {scorecard.feedback.buyer} customer</small
					>
				</article>
			</div>
			<p class="evidence-note">
				<strong>Evidence check:</strong>
				{scorecard.reconciliation_ok
					? 'The product events match the main trade records for this period.'
					: 'Some event counts do not match the main records. Fix the differences before using these numbers in an application.'}
			</p>
		</section>
		<section class="feedback-breakdown" aria-labelledby="feedback-title">
			<div>
				<p class="eyebrow">From page feedback</p>
				<h2 id="feedback-title">Can people understand Kredit?</h2>
				<p>
					These answers come from the question shown inside a seller or customer account. We do not collect a comment, a
					phone number or an email with the answer.
				</p>
			</div>
			<dl>
				<div>
					<dt>Yes</dt>
					<dd>{scorecard.feedback.yes}</dd>
				</div>
				<div>
					<dt>Partly</dt>
					<dd>{scorecard.feedback.partly}</dd>
				</div>
				<div>
					<dt>No</dt>
					<dd>{scorecard.feedback.no}</dd>
				</div>
			</dl>
		</section>
		<details class="full-scorecard">
			<summary>Open the full product scorecard</summary>
			<h2>Main product numbers</h2>
			<section class="metrics" aria-label="Primary pilot KPIs">
				{#each scorecard.kpis as metric (metric.key)}<article>
						<strong>{format(metric)}</strong>
						<h3>{metric.label}</h3>
						<p>{metric.definition}</p>
						<small>Source: {metric.source} · target: baseline required</small>
					</article>{/each}
			</section>
			<h2>What moves the numbers</h2>
			<section class="metrics" aria-label="KPI drivers">
				{#each scorecard.drivers as metric (metric.key)}<article>
						<strong>{format(metric)}</strong>
						<h3>{metric.label}</h3>
						<p>{metric.definition}</p>
						<small>Source: {metric.source}</small>
					</article>{/each}
			</section>
			<h2>Checks that protect the pilot</h2>
			<section class="metrics" aria-label="Pilot guardrails">
				{#each scorecard.guardrails as metric (metric.key)}<article>
						<strong>{format(metric)}</strong>
						<h3>{metric.label}</h3>
						<p>{metric.definition}</p>
						<small>Source: {metric.source}</small>
					</article>{/each}
			</section>
			<h2>Record check</h2>
			<div class="table-wrap">
				<table>
					<caption>Product events compared with the main records</caption><thead
						><tr><th>Event</th><th>Main record</th><th>Events</th><th>Status</th></tr></thead
					><tbody
						>{#each scorecard.reconciliation as row, i (i)}<tr
								><th>{row.event}</th><td>{row.source_count}</td><td>{row.event_count}</td><td
									><span class:ok={row.status === 'reconciled'}>{row.status}</span></td
								></tr
							>{/each}</tbody
					>
				</table>
			</div>
		</details>
	{/if}
</main>

<style>
	.analytics {
		padding-bottom: 4rem;
	}
	.analytics > h1 {
		max-width: 12ch;
		font-family: var(--font-serif);
		font-size: clamp(1.9rem, 3.4vw, 2.6rem);
		line-height: 1.1;
		letter-spacing: -0.03em;
	}
	.analytics form {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(11rem, 1fr));
		gap: 1rem;
		align-items: end;
		margin: 2rem 0;
		padding: 1rem;
		border: 1px solid var(--color-border);
		border-radius: 0;
	}
	.analytics label {
		display: grid;
		gap: 0.4rem;
		font-weight: 700;
	}
	.analytics input,
	.analytics button {
		min-height: 2.75rem;
		border: 1px solid var(--color-border);
		border-radius: 0;
		padding: 0.6rem;
	}
	.analytics button {
		background: var(--color-primary);
		color: var(--color-on-primary);
		font-weight: 800;
		--color-muted: rgb(255 255 255 / 0.72);
	}
	.status {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
		flex-wrap: wrap;
		padding: 1rem;
		border-left: 0.35rem solid var(--color-warning);
		background: var(--color-background);
	}
	.status.healthy {
		border-color: var(--color-positive);
		background: var(--color-background);
	}
	.status span {
		color: var(--color-muted);
	}
	.application-snapshot {
		margin: 1.5rem 0;
		border: 3px solid var(--color-primary);
		background: var(--color-surface);
		box-shadow: 10px 10px 0 var(--color-border);
	}
	.application-snapshot > header {
		display: flex;
		align-items: end;
		justify-content: space-between;
		gap: 1rem;
		padding: 1.5rem;
		border-bottom: 3px solid var(--color-primary);
	}
	.application-snapshot h2 {
		max-width: 22ch;
		margin: 0.25rem 0;
		font-family: var(--font-serif);
		font-size: clamp(1.8rem, 4vw, 3rem);
		font-weight: 500;
	}
	.proof-grid {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
	}
	.proof-grid article {
		display: grid;
		gap: 0.5rem;
		padding: 1.25rem;
		border-right: 1px solid var(--color-border);
		border-bottom: 1px solid var(--color-border);
	}
	.proof-grid span {
		font-weight: 750;
	}
	.proof-grid strong {
		font-family: var(--font-serif);
		font-size: clamp(1.6rem, 3vw, 2.4rem);
		font-weight: 500;
		color: var(--color-primary);
	}
	.proof-grid small {
		color: var(--color-muted);
	}
	.evidence-note {
		margin: 0;
		padding: 1.25rem;
		background: var(--color-background);
	}
	.feedback-breakdown {
		display: grid;
		grid-template-columns: 1.5fr 1fr;
		gap: 2rem;
		margin: 3rem 0;
		padding: clamp(1.5rem, 4vw, 2.5rem);
		color: var(--color-on-primary);
		background: var(--color-primary);
		--color-muted: rgb(255 255 255 / 0.72);
	}
	.feedback-breakdown h2 {
		font-family: var(--font-serif);
		font-size: 2.25rem;
		font-weight: 500;
	}
	.feedback-breakdown p {
		color: var(--color-muted);
	}
	.feedback-breakdown dl {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		margin: 0;
	}
	.feedback-breakdown dl div {
		display: grid;
		align-content: center;
		padding: 1rem;
		border: 1px solid var(--color-border-strong);
		text-align: center;
	}
	.feedback-breakdown dt {
		color: var(--color-muted);
	}
	.feedback-breakdown dd {
		margin: 0.35rem 0;
		font-family: var(--font-serif);
		font-size: 2.4rem;
	}
	.full-scorecard {
		margin-top: 2rem;
		border-top: 3px solid var(--color-primary);
	}
	.full-scorecard summary {
		padding: 1.2rem 0;
		font-size: 1.1rem;
		font-weight: 800;
		cursor: pointer;
	}
	.metrics {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(15rem, 1fr));
		gap: 1rem;
		margin-bottom: 2rem;
	}
	.metrics article {
		border: 1px solid var(--color-border);
		border-radius: 0;
		padding: 1.15rem;
		background: var(--color-surface);
	}
	.metrics article > strong {
		font-size: 1.55rem;
		color: var(--color-primary);
	}
	.metrics h3 {
		margin: 0.45rem 0;
	}
	.metrics p {
		min-height: 3.8rem;
	}
	.metrics small {
		color: var(--color-muted);
	}
	.table-wrap {
		overflow-x: auto;
	}
	table {
		border-collapse: collapse;
		width: 100%;
		background: var(--color-surface);
	}
	caption {
		text-align: left;
		padding: 0.75rem 0;
	}
	th,
	td {
		text-align: left;
		padding: 0.75rem;
		border-bottom: 1px solid var(--color-border);
	}
	td span {
		color: var(--color-overdue);
		font-weight: 800;
	}
	.ok {
		color: var(--color-positive);
	}
	@media (max-width: 48rem) {
		.proof-grid {
			grid-template-columns: 1fr 1fr;
		}
		.feedback-breakdown {
			grid-template-columns: 1fr;
		}
		.application-snapshot > header {
			align-items: start;
			flex-direction: column;
		}
	}
	@media (max-width: 40rem) {
		.proof-grid {
			grid-template-columns: 1fr;
		}
		.metrics p {
			min-height: 0;
		}
	}
	@media print {
		.analytics > form,
		.analytics > .status,
		.application-snapshot button,
		.feedback-breakdown,
		.full-scorecard {
			display: none;
		}
		.application-snapshot {
			box-shadow: none;
		}
		.analytics {
			padding: 0;
		}
		.application-snapshot {
			margin: 0;
		}
	}
</style>
