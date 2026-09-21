<script lang="ts">
	import { chooseWorkspace, requestedWorkspace } from '$lib/workspace-context';
	import { onMount } from 'svelte';
	import { exactKobo, formatKobo, sumKobo, type KoboValue } from '$lib/money';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';
	import { checkedJSON, LatestRequest, record, rows, publicError } from '$lib/api/reliable';
	import { organization, kobo, paymentRow } from '$lib/records';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import ShareActions from '$lib/components/ShareActions.svelte';
	type Organization = { id: string; legal_name: string; trading_name?: string };
	type Summary = {
		obligation_count: number;
		outstanding_kobo: KoboValue;
		overdue_kobo: KoboValue;
		voluntary_paid_kobo: KoboValue;
		collected_paid_kobo: KoboValue;
	};
	type PortfolioHealth = {
		on_time_collection_bps: number;
		disputed_ratio_bps: number;
		top_concentration_bps: number;
		health_rating: string;
	};
	type BranchExposure = {
		branch_id: string;
		branch_name: string;
		territory: string;
		total_outstanding_kobo: KoboValue;
		ageing_buckets: Record<string, KoboValue>;
		customer_count: number;
		overdue_count: number;
	};
	type EnterpriseReport = {
		organization_id: string;
		generated_at: string;
		portfolio_health: PortfolioHealth;
		branch_exposures: BranchExposure[];
		integrity_hash: string;
	};
	let organizations = $state<Organization[]>([]);
	let organizationID = $state('');
	let summary = $state<Summary | null>(null);
	let buckets = $state<Record<string, KoboValue>>({});
	let fees = $state<{ total_fees_kobo: KoboValue } | null>(null);
	let payments = $state<any[]>([]);
	let enterprise = $state<EnterpriseReport | null>(null);
	let sharePeriod = $state('today');
	let loading = $state(true);
	let error = $state('');
	let exportKey = '';
	let exporting = $state(false);
	let exportingEnterprise = $state(false);
	const money = (value: KoboValue) => formatKobo(value);
	// Money is summed in exact kobo. A figure the API did not return is null, not
	// zero: reporting an unread total as ₦0 would read as a real, settled balance.
	const paidTotal = $derived(sumKobo([summary?.voluntary_paid_kobo, summary?.collected_paid_kobo]));
	const trackedValue = $derived(sumKobo([paidTotal, summary?.outstanding_kobo]));
	// Percentages are presentation, so they are the one place a ratio is taken —
	// from BigInt inputs, and only when both sides are known.
	const percent = (part: KoboValue, whole: KoboValue) => {
		const a = exactKobo(part),
			b = exactKobo(whole);
		if (a === null || b === null || b <= 0n) return null;
		return Number((a * 100n) / b);
	};
	const receivedRate = $derived(percent(paidTotal, trackedValue));
	const overdueShare = $derived(percent(summary?.overdue_kobo, summary?.outstanding_kobo));
	const recognizedPayments = $derived(
		payments.filter((payment) =>
			['recognized', 'confirmed', 'paid'].includes(String(payment.state ?? payment.status ?? '').toLowerCase())
		)
	);
	const averagePayment = $derived.by(() => {
		if (!recognizedPayments.length) return null;
		const total = sumKobo(recognizedPayments.map((payment) => payment.amount_kobo));
		const exact = exactKobo(total);
		return exact === null ? null : exact / BigInt(recognizedPayments.length);
	});
	const reportsRequest = new LatestRequest(),
		businessRequest = new LatestRequest();
	async function load() {
		const request = reportsRequest.begin(),
			scope = organizationID;
		loading = true;
		error = '';
		summary = null;
		buckets = {};
		fees = null;
		payments = [];
		enterprise = null;
		if (!scope) {
			loading = false;
			return;
		}
		const root = `/api/v1/organizations/${encodeURIComponent(scope)}`;
		try {
			const [nextSummary, nextBuckets, nextFees, nextPayments, nextEnterprise] = await Promise.all([
				checkedJSON(
					`${root}/reports/receivables`,
					(value) => {
						const data = record(record(value).summary);
						if (!Number.isSafeInteger(data.obligation_count) || Number(data.obligation_count) < 0)
							throw new Error('Incomplete count');
						return {
							obligation_count: Number(data.obligation_count),
							outstanding_kobo: kobo(data.outstanding_kobo),
							overdue_kobo: kobo(data.overdue_kobo),
							voluntary_paid_kobo: kobo(data.voluntary_paid_kobo),
							collected_paid_kobo: kobo(data.collected_paid_kobo)
						};
					},
					{ signal: request.signal }
				),
				checkedJSON(
					`${root}/reports/ageing`,
					(value) =>
						Object.fromEntries(Object.entries(record(record(value).buckets)).map(([key, value]) => [key, kobo(value)])),
					{ signal: request.signal }
				),
				checkedJSON(`${root}/reports/fees`, (value) => ({ total_fees_kobo: kobo(record(value).total_fees_kobo) }), {
					signal: request.signal
				}),
				checkedJSON(`${root}/payments`, rows('payments', paymentRow), { signal: request.signal }),
				checkedJSON(
					`${root}/reports/enterprise`,
					(value) => {
						const data = record(value);
						const h = record(data.portfolio_health);
						const rawB = Array.isArray(data.branch_exposures) ? data.branch_exposures : [];
						const branches: BranchExposure[] = rawB.map((b: any) => ({
							branch_id: String(b.branch_id ?? ''),
							branch_name: String(b.branch_name ?? ''),
							territory: String(b.territory ?? ''),
							total_outstanding_kobo: kobo(b.total_outstanding_kobo),
							ageing_buckets: Object.fromEntries(
								Object.entries(record(b.ageing_buckets ?? {})).map(([k, v]) => [k, kobo(v)])
							),
							customer_count: Number(b.customer_count ?? 0),
							overdue_count: Number(b.overdue_count ?? 0)
						}));
						return {
							organization_id: String(data.organization_id ?? ''),
							generated_at: String(data.generated_at ?? ''),
							portfolio_health: {
								on_time_collection_bps: Number(h.on_time_collection_bps ?? 0),
								disputed_ratio_bps: Number(h.disputed_ratio_bps ?? 0),
								top_concentration_bps: Number(h.top_concentration_bps ?? 0),
								health_rating: String(h.health_rating ?? 'HEALTHY')
							},
							branch_exposures: branches,
							integrity_hash: String(data.integrity_hash ?? '')
						};
					},
					{ signal: request.signal }
				).catch(() => null)
			]);
			if (!request.current() || organizationID !== scope) return;
			summary = nextSummary;
			buckets = nextBuckets;
			fees = nextFees;
			payments = nextPayments;
			enterprise = nextEnterprise;
		} catch (cause) {
			if (request.current()) error = publicError(cause, 'your reports');
		} finally {
			if (request.current()) loading = false;
		}
	}
	async function initialize() {
		const request = businessRequest.begin();
		loading = true;
		error = '';
		try {
			const next = await checkedJSON('/api/v1/organizations', rows('organizations', organization), {
				signal: request.signal
			});
			if (!request.current()) return;
			organizations = next;
			organizationID = requestedWorkspace(next);
			await load();
		} catch (cause) {
			if (request.current()) {
				error = publicError(cause, 'your businesses');
				loading = false;
			}
		}
	}
	// The business day starts at midnight in Lagos wherever the phone happens to be.
	function lagosDayStart(daysBack: number): number {
		const parts = new Intl.DateTimeFormat('en-CA', {
			timeZone: 'Africa/Lagos',
			year: 'numeric',
			month: '2-digit',
			day: '2-digit'
		}).format(new Date());
		return new Date(`${parts}T00:00:00+01:00`).getTime() - daysBack * 86400000;
	}
	const shared = $derived.by(() => {
		const start = lagosDayStart(sharePeriod === 'today' ? 0 : 6);
		const received = recognizedPayments.filter((payment) => {
			const at = Date.parse(payment.paid_at ?? payment.created_at ?? '');
			return Number.isFinite(at) && at >= start && at <= Date.now();
		});
		return { count: received.length, total: sumKobo(received.map((payment) => payment.amount_kobo)) };
	});
	const summaryText = $derived(
		`Kredit ${sharePeriod === 'today' ? 'today' : 'last 7 days'}: ${shared.count} payment${shared.count === 1 ? '' : 's'} received, ${money(shared.total)} in total. ${money(summary?.outstanding_kobo)} still owed; ${money(summary?.overdue_kobo)} overdue.`
	);
	function bucketName(bucket: string) {
		return (
			(
				{
					current: 'Not overdue',
					not_due: 'Not overdue',
					'1_7': '1–7 days overdue',
					'8_30': '8–30 days overdue',
					'31_60': '31–60 days overdue',
					'61_plus': '61+ days overdue',
					paid: 'Paid'
				} as Record<string, string>
			)[bucket.toLowerCase()] ?? bucket.replaceAll('_', ' ')
		);
	}
	async function exportCSV() {
		if (exporting || loading || !summary || !organizationID) return;
		exporting = true;
		exportKey ||= idempotencyKey();
		try {
			const response = await fetch(`/api/v1/organizations/${organizationID}/reports/exports?format=csv`, {
				method: 'POST',
				signal: AbortSignal.timeout(20000),
				credentials: 'include',
				redirect: 'error',
				headers: { ...csrfHeaders(), 'Idempotency-Key': exportKey }
			});
			if (!response.ok) {
				error = 'We could not create that file. Please try again.';
				return;
			}
			if (!response.headers.get('Content-Type')?.toLowerCase().startsWith('text/csv'))
				throw new Error('Unexpected export');
			const objectURL = URL.createObjectURL(await response.blob());
			exportKey = '';
			const link = document.createElement('a');
			link.href = objectURL;
			link.download = 'kredit-receivables.csv';
			document.body.append(link);
			link.click();
			link.remove();
			setTimeout(() => URL.revokeObjectURL(objectURL), 30_000);
		} catch {
			error = 'We could not download the report. Please try again.';
		} finally {
			exporting = false;
		}
	}
	async function exportEnterpriseCSV() {
		if (exportingEnterprise || loading || !organizationID) return;
		exportingEnterprise = true;
		try {
			const response = await fetch(`/api/v1/organizations/${organizationID}/reports/enterprise/exports`, {
				method: 'POST',
				signal: AbortSignal.timeout(20000),
				credentials: 'include',
				headers: { ...csrfHeaders(), 'Idempotency-Key': idempotencyKey() }
			});
			if (!response.ok) {
				error = 'Could not generate enterprise report. Please try again.';
				return;
			}
			const blob = await response.blob();
			const objectURL = URL.createObjectURL(blob);
			const link = document.createElement('a');
			link.href = objectURL;
			link.download = `kredit-enterprise-exposure-${organizationID}.csv`;
			document.body.append(link);
			link.click();
			link.remove();
			setTimeout(() => URL.revokeObjectURL(objectURL), 30_000);
		} catch {
			error = 'We could not download the enterprise report. Please try again.';
		} finally {
			exportingEnterprise = false;
		}
	}
	onMount(() => {
		void initialize();
		return () => {
			businessRequest.cancel();
			reportsRequest.cancel();
		};
	});
</script>

<svelte:head><title>Reports — Kredit</title></svelte:head>
<main class="shell workspace reports">
	<header>
		<p class="eyebrow">Business performance</p>
		<h1>Business reports</h1>
		<p class="lede">
			What you are owed, what is late, what has come in, how old the debts are and what Kredit has charged you.
		</p>
	</header>
	<div class="toolbar">
		<label
			>Business<select
				disabled={exporting || exportingEnterprise}
				bind:value={organizationID}
				onchange={() => chooseWorkspace(organizationID)}
				>{#each organizations as organization}<option value={organization.id}
						>{organization.trading_name || organization.legal_name}</option
					>{/each}</select
			></label
		>
		<div class="actions">
			<button onclick={exportCSV} disabled={!summary || exporting || loading || Boolean(error)}
				>{exporting ? 'Preparing…' : 'Download Receivables CSV'}</button
			>
			<button
				class="secondary-btn"
				onclick={exportEnterpriseCSV}
				disabled={!enterprise || exportingEnterprise || loading || Boolean(error)}
				>{exportingEnterprise ? 'Signing…' : 'Enterprise Report CSV'}</button
			>
		</div>
	</div>
	{#if error}<p class="error" role="alert">{error}</p>
		<button onclick={initialize}>Try again</button>{:else if loading}<div class="loading" aria-live="polite">
			<Skeleton rows={4} tall />
		</div>{:else if !organizations.length}<p>
			Add your business to see its reports. <a href="/workspace/today">Set up your business →</a>
		</p>{:else if summary}
		<section class="stats">
			<article class="focus">
				<span>Owed to you now</span><strong>{money(summary.outstanding_kobo)}</strong><small
					>{summary.obligation_count} open sale{summary.obligation_count === 1 ? '' : 's'}</small
				>
			</article>
			<article>
				<span>Late right now</span><strong>{money(summary.overdue_kobo)}</strong><small
					>{overdueShare === null ? 'Share unavailable' : `${overdueShare}% of what you are owed`}</small
				>
			</article>
			<article>
				<span>Received</span><strong>{money(paidTotal)}</strong><small
					>{receivedRate === null ? 'Share unavailable' : `${receivedRate}% of everything Kredit is following`}</small
				>
			</article>
			<article>
				<span>Average payment</span><strong>{money(averagePayment)}</strong><small
					>{recognizedPayments.length} payment{recognizedPayments.length === 1 ? '' : 's'} recorded</small
				>
			</article>
			<article>
				<span>Kredit fees</span><strong>{money(fees?.total_fees_kobo)}</strong><small
					>What Kredit has charged you, after any reductions</small
				>
			</article>
		</section>
		<section class="performance">
			<div>
				<p class="eyebrow">How collection is going</p>
				<h2>
					{receivedRate === null
						? 'This share could not be worked out.'
						: `${receivedRate}% of the money Kredit is following has reached you.`}
				</h2>
				<p>
					This compares money already received with money received plus money still owed. Use it to get a feel for how
					things are going. It is not an accounting statement.
				</p>
			</div>
			<div class="meter" aria-label={receivedRate === null ? 'Share unavailable' : `${receivedRate}% received`}>
				<div><span style={`width:${receivedRate ?? 0}%`}></span></div>
				<p><b>{money(paidTotal)}</b> received <span>·</span> <b>{money(summary.outstanding_kobo)}</b> still owed</p>
			</div>
		</section>

		{#if enterprise}
			<section class="card enterprise-health">
				<div class="health-header">
					<div>
						<p class="eyebrow">Enterprise Risk Analytics</p>
						<h2>Portfolio Health & Concentration</h2>
					</div>
					<div class="badge-wrap">
						<span class={`health-badge badge-${enterprise.portfolio_health.health_rating.toLowerCase()}`}>
							{enterprise.portfolio_health.health_rating}
						</span>
					</div>
				</div>
				<div class="health-grid">
					<article>
						<span>On-Time Collection</span>
						<strong>{((enterprise.portfolio_health.on_time_collection_bps ?? 10000) / 100).toFixed(1)}%</strong>
						<small>Settled without overdue notice</small>
					</article>
					<article>
						<span>Disputed Ratio</span>
						<strong>{((enterprise.portfolio_health.disputed_ratio_bps ?? 0) / 100).toFixed(1)}%</strong>
						<small>Open dispute exposure</small>
					</article>
					<article>
						<span>Concentration Risk</span>
						<strong>{((enterprise.portfolio_health.top_concentration_bps ?? 0) / 100).toFixed(1)}%</strong>
						<small>Single highest counterparty</small>
					</article>
					<article>
						<span>Ledger Integrity</span>
						<strong class="hash-code"
							>{enterprise.integrity_hash ? `${enterprise.integrity_hash.slice(0, 8)}…` : 'Verified'}</strong
						>
						<small>SHA-256 state seal</small>
					</article>
				</div>
			</section>

			{#if enterprise.branch_exposures && enterprise.branch_exposures.length > 0}
				<section class="card branch-section">
					<h2>Branch & Territory Exposure</h2>
					<p>Breakdown of outstanding credit balances and ageing across operating branches.</p>
					<div class="table-wrap">
						<table class="branch-table">
							<thead>
								<tr>
									<th>Branch / Facility</th>
									<th>Territory</th>
									<th>Accounts</th>
									<th>0–30 Days</th>
									<th>31–60 Days</th>
									<th>61–90 Days</th>
									<th>90+ Days</th>
									<th>Total Outstanding</th>
								</tr>
							</thead>
							<tbody>
								{#each enterprise.branch_exposures as branch}
									<tr>
										<td><strong>{branch.branch_name || branch.branch_id}</strong></td>
										<td>{branch.territory || 'National'}</td>
										<td>{branch.customer_count} ({branch.overdue_count} late)</td>
										<td>{money(branch.ageing_buckets['0-30'] ?? 0)}</td>
										<td>{money(branch.ageing_buckets['31-60'] ?? 0)}</td>
										<td>{money(branch.ageing_buckets['61-90'] ?? 0)}</td>
										<td>{money(branch.ageing_buckets['90+'] ?? 0)}</td>
										<td class="total-col">{money(branch.total_outstanding_kobo)}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</section>
			{/if}
		{/if}

		<section class="card ageing">
			<h2>How old is the money you are owed?</h2>
			<p>The longer a debt sits, the harder it usually gets to collect. Chase the old ones first.</p>
			<dl>
				{#each Object.entries(buckets) as [bucket, amount]}<div>
						<dt>{bucketName(bucket)}</dt>
						<dd>{money(amount)}</dd>
					</div>{/each}
			</dl>
			<a href={`/workspace/overdue?organization=${encodeURIComponent(organizationID)}`}>See who is late →</a>
		</section>
		<section class="card share-card">
			<div>
				<h2>Send someone a money update</h2>
				<p>Send a short summary to a partner or a staff member, without giving them your whole book.</p>
				<label
					>Period<select bind:value={sharePeriod}
						><option value="today">Today</option><option value="week">Last 7 days</option></select
					></label
				>
			</div>
			<div>
				<strong>{money(shared.total)}</strong><small
					>{shared.count} payment{shared.count === 1 ? '' : 's'} received</small
				><ShareActions compact title="Kredit money update" text={summaryText} />
			</div>
		</section>
	{/if}
</main>

<style>
	.reports > header {
		max-width: 62rem;
	}
	.reports h1 {
		max-width: 15ch;
		font-family: var(--font-serif);
		font-size: clamp(3rem, 6vw, 5.4rem);
		font-weight: 500;
		line-height: 0.92;
		letter-spacing: -0.055em;
	}
	.toolbar {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
		align-items: end;
		margin: 2rem 0;
		padding: 1rem;
		flex-wrap: wrap;
		color: var(--color-on-primary);
		background: var(--color-primary);
		--color-muted: rgb(255 255 255 / 0.72);
	}
	.toolbar label {
		display: grid;
		gap: 0.4rem;
		font-weight: 700;
	}
	.toolbar select,
	.toolbar button {
		min-height: 3rem;
		padding: 0.75rem;
		border: 1px solid var(--color-border-strong);
		border-radius: 0;
		background: var(--color-surface);
		color: var(--color-primary);
	}
	.toolbar .actions {
		display: flex;
		gap: 0.75rem;
		flex-wrap: wrap;
	}
	.toolbar button {
		background: var(--color-primary);
		color: var(--color-on-primary);
		border-color: var(--color-primary);
		font-weight: 750;
		--color-muted: rgb(255 255 255 / 0.72);
	}
	.toolbar .secondary-btn {
		background: var(--color-surface);
		color: var(--color-primary);
		border-color: var(--color-border-strong);
	}
	.stats {
		display: grid;
		grid-template-columns: 1.2fr repeat(4, 1fr);
		border-top: 1px solid var(--color-border);
		border-left: 1px solid var(--color-border);
	}
	.stats article,
	.card {
		padding: 1.25rem;
		border: 0;
		border-right: 1px solid var(--color-border);
		border-bottom: 1px solid var(--color-border);
		border-radius: 0;
		background: var(--color-surface);
	}
	.stats article.focus {
		color: var(--color-on-primary);
		background: var(--color-primary);
		--color-muted: rgb(255 255 255 / 0.72);
	}
	.stats span,
	.stats small {
		display: block;
		color: var(--color-muted);
	}
	.stats .focus span,
	.stats .focus small {
		color: var(--color-surface-muted);
	}
	.stats strong {
		display: block;
		margin: 0.45rem 0;
		font-family: var(--font-serif);
		font-size: clamp(1.5rem, 2.5vw, 2.2rem);
		font-weight: 500;
	}
	.performance {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: clamp(2rem, 7vw, 7rem);
		align-items: center;
		margin: 2rem 0;
		padding: 2rem;
		border: 1px solid var(--color-border);
		background: var(--color-primary);
		color: var(--color-on-primary);
		--color-muted: rgb(255 255 255 / 0.72);
	}
	.performance h2 {
		max-width: 13ch;
		margin: 0.5rem 0;
		font-family: var(--font-serif);
		font-size: clamp(2rem, 4vw, 3.5rem);
		font-weight: 500;
		line-height: 0.98;
	}
	.performance > div > p:last-child {
		color: var(--color-muted);
		line-height: 1.65;
	}
	.meter > div {
		height: 1rem;
		background: var(--color-foreground);
		--color-muted: rgb(255 255 255 / 0.72);
	}
	.meter > div span {
		display: block;
		height: 100%;
		background: var(--color-accent);
	}
	.meter p {
		display: flex;
		flex-wrap: wrap;
		gap: 0.45rem;
		color: var(--color-muted);
	}
	.meter b {
		color: var(--color-on-primary);
	}
	.card {
		margin-top: 1rem;
		border: 1px solid var(--color-border);
	}
	.ageing > p {
		color: var(--color-muted);
	}
	.card dl {
		display: grid;
		gap: 0.7rem;
	}
	.card dl div {
		display: flex;
		justify-content: space-between;
		gap: 2rem;
		border-bottom: 1px solid var(--color-border);
		padding-bottom: 0.7rem;
	}
	.card dd {
		font-weight: 700;
	}
	.ageing > a {
		display: inline-block;
		margin-top: 1rem;
		font-weight: 750;
	}
	.enterprise-health {
		margin: 2rem 0;
		border: 1px solid var(--color-border);
	}
	.health-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
		margin-bottom: 1.5rem;
	}
	.health-header h2 {
		font-family: var(--font-serif);
		font-size: clamp(1.75rem, 3vw, 2.2rem);
		font-weight: 500;
		margin: 0.25rem 0;
	}
	.health-badge {
		display: inline-block;
		padding: 0.35rem 0.85rem;
		font-size: 0.85rem;
		font-weight: 750;
		letter-spacing: 0.05em;
		text-transform: uppercase;
		border-radius: 2px;
	}
	.badge-excellent {
		background: #0d5e36;
		color: #ffffff;
	}
	.badge-healthy {
		background: #004b87;
		color: #ffffff;
	}
	.badge-watch {
		background: #b25e00;
		color: #ffffff;
	}
	.badge-stressed {
		background: #9b1c1c;
		color: #ffffff;
	}
	.health-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 1rem;
		border-top: 1px solid var(--color-border);
		padding-top: 1.25rem;
	}
	.health-grid article span {
		display: block;
		font-size: 0.85rem;
		color: var(--color-muted);
	}
	.health-grid article strong {
		display: block;
		font-family: var(--font-serif);
		font-size: clamp(1.4rem, 2vw, 1.8rem);
		font-weight: 500;
		margin: 0.3rem 0;
	}
	.health-grid article small {
		display: block;
		font-size: 0.8rem;
		color: var(--color-muted);
	}
	.hash-code {
		font-family: monospace;
		font-size: 1.1rem !important;
		letter-spacing: -0.05em;
	}
	.branch-section {
		margin: 2rem 0;
		border: 1px solid var(--color-border);
	}
	.branch-section h2 {
		font-family: var(--font-serif);
		font-size: clamp(1.5rem, 2.5vw, 1.8rem);
		font-weight: 500;
		margin: 0 0 0.25rem;
	}
	.branch-section p {
		color: var(--color-muted);
		margin-bottom: 1.25rem;
	}
	.table-wrap {
		overflow-x: auto;
	}
	.branch-table {
		width: 100%;
		border-collapse: collapse;
		text-align: left;
		font-size: 0.9rem;
	}
	.branch-table th {
		background: var(--color-primary);
		color: var(--color-on-primary);
		padding: 0.6rem 0.8rem;
		font-weight: 700;
	}
	.branch-table td {
		padding: 0.7rem 0.8rem;
		border-bottom: 1px solid var(--color-border);
	}
	.branch-table .total-col {
		font-weight: 750;
		text-align: right;
	}
	.share-card {
		display: grid;
		grid-template-columns: 1fr auto;
		gap: 2rem;
		align-items: center;
		margin-top: 2rem;
		background: var(--color-background);
	}
	.share-card h2,
	.share-card p {
		margin: 0.25rem 0;
	}
	.share-card label {
		display: grid;
		gap: 0.3rem;
		width: max-content;
		margin-top: 0.8rem;
		font-weight: 750;
	}
	.share-card select {
		min-height: 2.7rem;
		padding: 0.55rem;
		border: 1px solid var(--color-border);
		background: var(--color-surface);
		font: inherit;
	}
	.share-card > div:last-child > strong,
	.share-card > div:last-child > small {
		display: block;
	}
	.share-card > div:last-child > strong {
		font-family: var(--font-serif);
		font-size: 2rem;
		font-weight: 500;
	}
	.error {
		color: var(--color-overdue);
	}
	@media (max-width: 1050px) {
		.stats {
			grid-template-columns: 1fr 1fr;
		}
		.stats .focus {
			grid-column: 1/-1;
		}
		.health-grid {
			grid-template-columns: 1fr 1fr;
		}
	}
	@media (max-width: 700px) {
		.stats,
		.performance {
			grid-template-columns: 1fr;
		}
		.stats .focus {
			grid-column: auto;
		}
		.toolbar {
			align-items: stretch;
			flex-direction: column;
		}
		.toolbar .actions {
			flex-direction: column;
		}
		.toolbar button {
			width: 100%;
		}
		.share-card {
			grid-template-columns: 1fr;
		}
		.card dl div {
			align-items: flex-start;
			flex-direction: column;
			gap: 0.25rem;
		}
		.health-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
