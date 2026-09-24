<script lang="ts">
	import { readableDate } from '$lib/datetime';
	import { onMount } from 'svelte';
	import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
	import { checkedJSON, LatestRequest, record, rows, text } from '$lib/api/reliable';
	import { kobo } from '$lib/records';
	import { MutationIntent } from '$lib/api/mutation';
	import { requestedWorkspace, chooseWorkspace } from '$lib/workspace-context';
	import { formatKobo, nairaInput, parseNaira, exactKobo, type KoboValue } from '$lib/money';
	type Proposal = { principal_kobo: KoboValue; goods_description: string; due_date: string; created_by: string };
	type Approval = {
		id: string;
		request_id: string;
		drawdown_id?: string;
		kind: string;
		state: string;
		requested_by: string;
		customer_name: string;
		reason: string;
		stale: boolean;
		proposal: Proposal;
	};
	type Reviewer = { user_id: string; name: string; role: string; ceiling_kobo: KoboValue; version: number };
	let reviewers = $state<Reviewer[]>([]),
		ceilings = $state<Record<string, string>>({});
	type Controls = { enabled: boolean; threshold_kobo: KoboValue; version: number };
	let organization = $state(''),
		businesses = $state<{ id: string; name: string }[]>([]);
	let loading = $state(true),
		busy = $state(false),
		error = $state(''),
		message = $state('');
	let controls = $state<Controls | null>(null),
		approvals = $state<Approval[]>([]),
		role = $state(''),
		user = $state('');
	let enabled = $state(false),
		threshold = $state('0.00'),
		reasons = $state<Record<string, string>>({});
	let draft = $state<{
		id: string;
		buyer_legal_name: string;
		principal_kobo: KoboValue;
		goods_description: string;
		state: string;
	} | null>(null);
	const reads = new LatestRequest();
	const base = () => `/api/v1/organizations/${encodeURIComponent(organization)}`;
	function decodeReviewer(value: unknown): Reviewer {
		const r = record(value);
		if (!Number.isSafeInteger(r.version) || Number(r.version) < 0)
			throw new Error('Reviewer limits could not be verified.');
		return {
			user_id: text(r.user_id),
			name: text(r.name),
			role: text(r.role),
			ceiling_kobo: r.ceiling_kobo === null ? null : kobo(r.ceiling_kobo),
			version: Number(r.version)
		};
	}
	function withinLimit(amount: KoboValue) {
		const limit = reviewers.find((r) => r.user_id === user)?.ceiling_kobo;
		return limit === null || limit === undefined || exactKobo(amount)! <= exactKobo(limit)!;
	}
	function saveLimit(reviewer: Reviewer) {
		const amount = parseNaira(ceilings[reviewer.user_id]);
		if (amount < 0) {
			error = 'Enter a valid reviewer limit.';
			return;
		}
		void change(
			`/credit-approvals/reviewers/${encodeURIComponent(reviewer.user_id)}`,
			{ ceiling_kobo: amount, version: reviewer.version },
			'Reviewer limit saved.',
			'PUT'
		);
	}
	function decodeControls(value: unknown): Controls {
		const c = record(value);
		if (typeof c.enabled !== 'boolean' || !Number.isSafeInteger(c.version) || Number(c.version) < 0)
			throw new Error('Approval settings could not be verified.');
		return { enabled: c.enabled, threshold_kobo: kobo(c.threshold_kobo), version: Number(c.version) };
	}
	function decodeApproval(value: unknown): Approval {
		const a = record(value),
			p = record(a.proposal);
		const state = text(a.state);
		if (!['pending', 'approved', 'rejected'].includes(state) || typeof a.stale !== 'boolean')
			throw new Error('Approval state could not be verified.');
		const kind = typeof a.kind === 'string' ? a.kind : 'sale';
		const reqID = typeof a.request_id === 'string' ? a.request_id : '';
		const drawID = typeof a.drawdown_id === 'string' ? a.drawdown_id : '';
		return {
			id: text(a.id),
			request_id: reqID,
			drawdown_id: drawID,
			kind,
			state,
			requested_by: text(a.requested_by),
			customer_name: text(a.customer_name),
			reason: typeof a.reason === 'string' ? a.reason : '',
			stale: a.stale,
			proposal: {
				principal_kobo: kobo(p.principal_kobo),
				goods_description: text(p.goods_description),
				due_date: typeof p.due_date === 'string' ? p.due_date : '',
				created_by: text(p.created_by)
			}
		};
	}
	async function load() {
		const request = reads.begin();
		loading = true;
		error = '';
		approvals = [];
		controls = null;
		draft = null;
		try {
			const items = await checkedJSON(
				'/api/v1/organizations',
				rows('organizations', (value) => {
					const b = record(value);
					return {
						id: text(b.id),
						name: (typeof b.trading_name === 'string' ? b.trading_name : '') || text(b.legal_name)
					};
				}),
				{ signal: request.signal }
			);
			if (!request.current()) return;
			businesses = items;
			organization = requestedWorkspace(items);
			if (!organization) return;
			const result = await checkedJSON(
				`${base()}/credit-approvals`,
				(value) => {
					const r = record(value);
					return {
						controls: decodeControls(r.controls),
						role: text(r.role),
						user: text(r.user_id),
						approvals: rows('approvals', decodeApproval)(r),
						reviewers: rows('reviewers', decodeReviewer)(r)
					};
				},
				{ signal: request.signal }
			);
			if (!request.current()) return;
			controls = result.controls;
			enabled = controls.enabled;
			threshold = nairaInput(controls.threshold_kobo);
			role = result.role;
			user = result.user;
			approvals = result.approvals;
			reviewers = result.reviewers;
			ceilings = Object.fromEntries(reviewers.map((r) => [r.user_id, nairaInput(r.ceiling_kobo)]));
			const id = new URLSearchParams(location.search).get('request');
			if (id) {
				const selected = await checkedJSON(
					`${base()}/credit-requests/${encodeURIComponent(id)}`,
					(value) => {
						const r = record(record(value).request);
						return {
							id: text(r.id),
							buyer_legal_name: text(r.buyer_legal_name),
							principal_kobo: kobo(r.principal_kobo),
							goods_description: text(r.goods_description),
							state: text(r.state)
						};
					},
					{ signal: request.signal }
				);
				if (request.current()) draft = selected;
			}
		} catch (cause) {
			if (request.current()) error = cause instanceof Error ? cause.message : 'Credit approvals could not be loaded.';
		} finally {
			if (request.current()) loading = false;
		}
	}
	async function change(path: string, body: unknown, success: string, method = 'POST') {
		if (busy || loading || !controls) return;
		busy = true;
		error = '';
		message = '';
		try {
			await new MutationIntent(`credit-approval:${organization}:${path}`, `${base()}${path}`).run(body, record, method);
			message = success;
			await load();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'The change could not be confirmed. Refresh before retrying.';
		} finally {
			busy = false;
		}
	}
	function save() {
		const amount = parseNaira(threshold);
		if (amount < 0) {
			error = 'Enter a valid amount, including zero if every offer needs approval.';
			return;
		}
		void change(
			'/credit-approvals/policy',
			{ enabled, threshold_kobo: amount, version: controls!.version },
			'Approval rule saved.',
			'PUT'
		);
	}
	onMount(() => {
		void load();
		return () => reads.cancel();
	});
</script>

<svelte:head><title>Credit approvals — Kredit</title></svelte:head>
<main class="shell workspace approvals">
	<p class="eyebrow">Sales</p>
	<h1>Credit approvals</h1>
	<p class="lede">Have a second person review larger business credit offers before they reach your customers.</p>
	<VerifyIdentity />
	{#if message}<p role="status">{message}</p>{/if}
	{#if error}<p role="alert">{error}</p>
		<button disabled={busy} onclick={load}>Refresh approvals</button>{/if}
	<label
		>Business<select bind:value={organization} disabled={busy || loading} onchange={() => chooseWorkspace(organization)}
			>{#each businesses as business (business.id)}<option value={business.id}>{business.name}</option>{/each}</select
		></label
	>
	{#if loading}<p role="status">Loading approval records…</p>{:else if !organization && !error}<p>
			<a href="/workspace/onboarding">Set up your business to manage credit approvals.</a>
		</p>{/if}
	{#if controls}
		<section class="card">
			<h2>When a second person must approve</h2>
			<p>
				This rule applies to single-sale business credit offers. A reviewer must be a different owner, administrator or
				finance team member. The customer still needs to accept the offer.
			</p>
			{#if role === 'owner'}<label class="toggle"
					><input type="checkbox" bind:checked={enabled} disabled={busy} />Require independent approval</label
				><label>Approval required above (₦)<input bind:value={threshold} inputmode="decimal" disabled={busy} /></label>
				<p>Set zero to require approval for every offer. Keep at least two authorized people on your team.</p>
				<button onclick={save} disabled={busy}>Save approval rule</button>
			{:else}<p>
					{controls.enabled
						? `Offers above ${formatKobo(controls.threshold_kobo)} require approval.`
						: 'Independent approval is not currently required.'} A business owner can change this rule.
				</p>{/if}
		</section>
		<section class="card">
			<h2>Reviewer limits</h2>
			<p>
				Limits apply to the full principal of each single-sale offer. A blank limit means no additional ceiling. Set
				zero to prevent approvals. The independent-review rule still applies.
			</p>
			{#each reviewers as reviewer, i (i)}<div>
					<strong>{reviewer.name}</strong>
					<p>
						{reviewer.role} · {reviewer.ceiling_kobo === null
							? 'No additional ceiling'
							: formatKobo(reviewer.ceiling_kobo)}
					</p>
					{#if role === 'owner'}<label
							>Approval limit for {reviewer.name} (₦)<input
								bind:value={ceilings[reviewer.user_id]}
								inputmode="decimal"
								disabled={busy}
							/></label
						><button disabled={busy || !ceilings[reviewer.user_id]} onclick={() => saveLimit(reviewer)}
							>Save limit for {reviewer.name}</button
						>{/if}
				</div>{/each}
		</section>
		{#if draft}<section class="card">
				<h2>Request a review</h2>
				<p><strong>{draft.buyer_legal_name} · {formatKobo(draft.principal_kobo)}</strong></p>
				<p>{draft.goods_description}</p>
				<a href={`/workspace/sales/${draft.id}?organization=${encodeURIComponent(organization)}`}
					>Review the complete offer</a
				>
				{#if draft.state === 'DRAFT' && ['owner', 'administrator', 'sales'].includes(role)}<button
						disabled={busy}
						onclick={() =>
							change(
								`/credit-requests/${draft!.id}/approval`,
								{},
								'Review requested. A different reviewer can now decide.'
							)}>Request internal approval</button
					>{:else}<p>Only editable drafts can be submitted for approval.</p>{/if}
			</section>{/if}
		<section>
			<h2>Approval records</h2>
			<p>Up to 100 records, with pending reviews first. Open a saved draft to request a review.</p>
			{#if !approvals.length}<p>No approval requests yet.</p>{/if}
			{#each approvals as item (item.id)}<article class="card">
					<h3>{item.customer_name} · {formatKobo(item.proposal.principal_kobo)}</h3>
					<p>
						<span class="badge">{item.kind === 'drawdown' ? 'Trade line drawdown' : 'Single sale'}</span>
						{item.proposal.goods_description}
					</p>
					<p>
						{item.proposal.due_date ? `Due ${readableDate(item.proposal.due_date)} · ` : ''}<strong>{item.state}</strong
						>
					</p>
					<a
						href={item.kind === 'drawdown'
							? `/workspace/purchases/trade-lines?organization=${encodeURIComponent(organization)}`
							: `/workspace/sales/${item.request_id}?organization=${encodeURIComponent(organization)}`}
						>Review complete terms</a
					>
					{#if item.reason}<p>Decision reason: {item.reason}</p>{/if}
					{#if item.stale}<p>
							This record no longer matches an editable draft. Review the current sale before requesting another
							approval.
						</p>
					{:else if item.state === 'pending' && ['owner', 'administrator', 'finance'].includes(role) && item.requested_by !== user && item.proposal.created_by !== user}
						<label
							>Reason for your decision<textarea bind:value={reasons[item.id]} disabled={busy} maxlength="1000"
							></textarea></label
						>
						{#if !withinLimit(item.proposal.principal_kobo)}<p>
								This offer exceeds your approval limit. Ask another authorized reviewer.
							</p>{/if}
						<div class="actions">
							<button
								disabled={busy ||
									!withinLimit(item.proposal.principal_kobo) ||
									(reasons[item.id]?.trim().length ?? 0) < 3}
								onclick={() =>
									change(
										`/credit-approvals/${item.id}`,
										{ decision: 'approved', reason: reasons[item.id].trim() },
										'Approval recorded. The seller can now send these exact terms.'
									)}>Approve exact terms</button
							><button
								disabled={busy || (reasons[item.id]?.trim().length ?? 0) < 3}
								onclick={() =>
									change(
										`/credit-approvals/${item.id}`,
										{ decision: 'rejected', reason: reasons[item.id].trim() },
										'Offer rejected. Its draft must be revised before another review.'
									)}>Reject offer</button
							>
						</div>
					{:else if item.state === 'pending'}<p>A different authorized reviewer must decide this request.</p>{/if}
				</article>{/each}
		</section>
	{/if}
</main>

<style>
	.approvals {
		max-width: 64rem;
	}
	.card {
		padding: 1.4rem;
		margin: 1.4rem 0;
		border: 1px solid var(--color-border);
		border-radius: 1rem;
		background: var(--color-surface);
	}
	label {
		display: grid;
		gap: 0.5rem;
		margin: 1rem 0;
	}
	input,
	select,
	textarea,
	button {
		font: inherit;
		padding: 0.75rem;
		border: 1px solid var(--color-border);
		border-radius: 0.6rem;
		background: var(--color-surface);
		color: inherit;
	}
	textarea {
		min-height: 6rem;
	}
	.toggle {
		display: flex;
		align-items: center;
	}
	.actions {
		display: flex;
		gap: 0.75rem;
		flex-wrap: wrap;
	}
	.card > a {
		display: block;
		margin: 0.8rem 0;
	}
	button {
		cursor: pointer;
	}
	button:disabled {
		cursor: default;
		opacity: 0.6;
	}
	.badge {
		display: inline-block;
		padding: 0.2rem 0.5rem;
		border-radius: 0.4rem;
		background: var(--color-surface-hover);
		font-size: 0.8rem;
		margin-right: 0.4rem;
	}
</style>
