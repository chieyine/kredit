<script lang="ts">
	import { onMount } from 'svelte';
	import { checkedJSON, LatestRequest, record, rows, text } from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';
	import { requestedWorkspace, chooseWorkspace } from '$lib/workspace-context';
	import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
	type Scope = {
		user_id: string;
		name: string;
		mode: 'all' | 'branches';
		branch_ids: string[];
		version: number;
		active: boolean;
	};
	let organization = $state(''),
		businesses = $state<{ id: string; name: string }[]>([]),
		branches = $state<{ id: string; name: string; active: boolean }[]>([]),
		scopes = $state<Scope[]>([]);
	let owner = $state(false),
		loading = $state(true),
		busy = $state(false),
		error = $state(''),
		message = $state('');
	const reads = new LatestRequest();
	const base = () => `/api/v1/organizations/${encodeURIComponent(organization)}`;
	function scope(value: unknown): Scope {
		const s = record(value);
		if (
			(s.mode !== 'all' && s.mode !== 'branches') ||
			!Array.isArray(s.branch_ids) ||
			!s.branch_ids.every((b) => typeof b === 'string') ||
			!Number.isSafeInteger(s.version) ||
			typeof s.active !== 'boolean'
		)
			throw new Error('Team access could not be verified.');
		return {
			user_id: text(s.user_id),
			name: text(s.name),
			mode: s.mode,
			branch_ids: s.branch_ids,
			version: Number(s.version),
			active: s.active
		};
	}
	async function load() {
		const request = reads.begin();
		loading = true;
		error = '';
		scopes = [];
		branches = [];
		owner = false;
		try {
			const available = await checkedJSON(
				'/api/v1/organizations',
				rows('organizations', (value) => {
					const b = record(value);
					return {
						id: text(b.id),
						name: typeof b.trading_name === 'string' && b.trading_name ? b.trading_name : text(b.legal_name)
					};
				}),
				{ signal: request.signal }
			);
			if (!request.current()) return;
			businesses = available;
			organization = requestedWorkspace(available);
			if (!organization) return;
			const [directory, network] = await Promise.all([
				checkedJSON(
					`${base()}/branch-access`,
					(value) => {
						const d = record(value);
						if (typeof d.can_manage !== 'boolean') throw new Error('Owner permissions unavailable.');
						return { owner: d.can_manage, scopes: rows('scopes', scope)(d) };
					},
					{ signal: request.signal }
				),
				checkedJSON(
					`${base()}/network-operations`,
					rows('branches', (value) => {
						const b = record(value);
						return { id: text(b.id), name: text(b.name), active: b.active === true };
					}),
					{ signal: request.signal }
				)
			]);
			if (!request.current()) return;
			owner = directory.owner;
			scopes = directory.scopes;
			branches = network;
		} catch (cause) {
			if (request.current()) error = cause instanceof Error ? cause.message : 'Team access could not be loaded.';
		} finally {
			if (request.current()) loading = false;
		}
	}
	function toggle(s: Scope, id: string, checked: boolean) {
		s.branch_ids = checked ? [...new Set([...s.branch_ids, id])] : s.branch_ids.filter((b) => b !== id);
	}
	async function save(s: Scope) {
		if (busy || !owner) return;
		busy = true;
		error = '';
		message = '';
		try {
			await new MutationIntent(
				`branch-access:${organization}:${s.user_id}`,
				`${base()}/branch-access/${encodeURIComponent(s.user_id)}`
			).run({ mode: s.mode, branch_ids: s.mode === 'all' ? [] : s.branch_ids, version: s.version }, record, 'PUT');
			message = 'Team access saved.';
			await load();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'The result could not be confirmed. Refresh before retrying.';
		} finally {
			busy = false;
		}
	}
	onMount(() => {
		void load();
		return () => reads.cancel();
	});
</script>

<svelte:head><title>Branch access — Kredit</title></svelte:head>
<main class="shell workspace access">
	<p class="eyebrow">Partners</p>
	<h1>Choose where your team can work</h1>
	<p>Limit staff to customers assigned to selected branches. Their job role still decides what they can do.</p>
	<VerifyIdentity />
	<label
		>Business<select bind:value={organization} disabled={busy || loading} onchange={() => chooseWorkspace(organization)}
			>{#each businesses as b (b.id)}<option value={b.id}>{b.name}</option>{/each}</select
		></label
	>
	{#if error}<p role="alert">{error}</p>
		<button disabled={busy} onclick={load}>Refresh access</button>{/if}{#if message}<p role="status">{message}</p>{/if}
	{#if loading}<p role="status">Loading branch access…</p>{:else if !organization && !error}<p>
			<a href="/workspace/onboarding">Set up a business first.</a>
		</p>{:else if !error}
		<p>
			Owners and administrators have company-wide access. Only owners can change branch access. Customers without a
			branch stay with the company-wide team.
		</p>
		<p>
			Branch access covers assigned business customers and single-sale records, including their payment history.
			Company-wide settings, personal consumer sales, shared document storage and other unallocated records require
			company-wide access. Purchasing permissions remain separate.
		</p>
		<p>
			<a href={`/workspace/partners/operations?organization=${encodeURIComponent(organization)}`}
				>Manage branches and customer assignments</a
			>
		</p>
		{#if !scopes.length}<p>
				No eligible staff members. <a href={`/workspace/team?organization=${encodeURIComponent(organization)}`}
					>Manage your team</a
				>.
			</p>{/if}
		{#each scopes as s (s.user_id)}<section class="card">
				<h2>{s.name}</h2>
				{#if !s.active}<p role="status">
						Membership has changed. An owner must save a new access decision before this member can work with customer
						records.
					</p>{/if}
				<fieldset disabled={busy || !owner}>
					<label
						>Access for {s.name}<select bind:value={s.mode}
							><option value="all">All business customers</option><option value="branches"
								>Selected branches only</option
							></select
						></label
					>
					{#if s.mode === 'branches'}{#each branches as b (b.id)}<label class="check"
								><input
									type="checkbox"
									checked={s.branch_ids.includes(b.id)}
									disabled={!b.active && !s.branch_ids.includes(b.id)}
									onchange={(e) => toggle(s, b.id, e.currentTarget.checked)}
								/>{b.name}{b.active ? '' : ' (closed)'}</label
							>{/each}{#if !s.branch_ids.length}<p>
								No branches selected: this member cannot access business customer sales.
							</p>{/if}{/if}
					{#if owner}<button onclick={() => save(s)}>Save access for {s.name}</button>{/if}
				</fieldset>
			</section>{/each}{/if}
</main>

<style>
	.access {
		max-width: 58rem;
	}
	.card {
		padding: 1.4rem;
		margin: 1rem 0;
		border: 1px solid var(--color-border);
		border-radius: 1rem;
		background: var(--color-surface);
	}
	fieldset {
		border: 0;
		padding: 0;
		min-width: 0;
	}
	label {
		display: grid;
		gap: 0.5rem;
		margin: 1rem 0;
	}
	.check {
		display: flex;
		align-items: center;
		gap: 0.6rem;
	}
	input,
	select,
	button {
		font: inherit;
		padding: 0.7rem;
		max-width: 100%;
		box-sizing: border-box;
		border: 1px solid var(--color-border);
		border-radius: 0.5rem;
		background: var(--color-surface);
		color: inherit;
	}
	button:disabled {
		opacity: 0.6;
	}
	p {
		line-height: 1.7;
	}
</style>
