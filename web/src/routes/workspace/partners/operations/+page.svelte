<script lang="ts">
	import { onMount } from 'svelte';
	import { checkedJSON, LatestRequest, record, rows, text } from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';
	import { requestedWorkspace, chooseWorkspace } from '$lib/workspace-context';
	import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
	type Branch = { id: string; name: string; territory: string; active: boolean; version: number };
	type Partner = {
		business_id: string;
		name: string;
		branch_id: string;
		manager_id: string;
		manager_active: boolean;
		version: number;
	};
	let organization = $state(''),
		businesses = $state<{ id: string; name: string }[]>([]),
		branches = $state<Branch[]>([]),
		partners = $state<Partner[]>([]),
		managers = $state<{ id: string; name: string }[]>([]);
	let loading = $state(true),
		busy = $state(false),
		error = $state(''),
		message = $state(''),
		canManage = $state(false),
		filter = $state('');
	let newBranch = $state({ id: '', name: '', territory: '', active: true, version: 0 });
	const reads = new LatestRequest();
	const base = () => `/api/v1/organizations/${encodeURIComponent(organization)}`;
	const version = (v: unknown) => {
		if (!Number.isSafeInteger(v) || Number(v) < 0) throw new Error('Record version is unavailable.');
		return Number(v);
	};
	const boolean = (v: unknown) => {
		if (typeof v !== 'boolean') throw new Error('Record status is unavailable.');
		return v;
	};
	const optional = (v: unknown) => (typeof v === 'string' ? v : '');
	function branch(value: unknown): Branch {
		const b = record(value);
		return {
			id: text(b.id),
			name: text(b.name),
			territory: optional(b.territory),
			active: boolean(b.active),
			version: version(b.version)
		};
	}
	async function load() {
		const request = reads.begin();
		loading = true;
		error = '';
		branches = [];
		partners = [];
		managers = [];
		canManage = false;
		try {
			const items = await checkedJSON(
				'/api/v1/organizations',
				rows('organizations', (value) => {
					const b = record(value);
					return { id: text(b.id), name: optional(b.trading_name) || text(b.legal_name) };
				}),
				{ signal: request.signal }
			);
			if (!request.current()) return;
			businesses = items;
			organization = requestedWorkspace(items);
			if (!organization) return;
			const data = await checkedJSON(
				`${base()}/network-operations`,
				(value) => {
					const d = record(value);
					return {
						branches: rows('branches', branch)(d),
						partners: rows('partners', (value) => {
							const p = record(value);
							return {
								business_id: text(p.business_id),
								name: text(p.name),
								branch_id: optional(p.branch_id),
								manager_id: optional(p.manager_id),
								manager_active: boolean(p.manager_active),
								version: version(p.version)
							};
						})(d),
						managers: rows('managers', (value) => {
							const m = record(value);
							return { id: text(m.id), name: text(m.name) };
						})(d),
						can_manage: boolean(d.can_manage)
					};
				},
				{ signal: request.signal }
			);
			if (!request.current()) return;
			branches = data.branches;
			partners = data.partners;
			managers = data.managers;
			canManage = data.can_manage;
		} catch (cause) {
			if (request.current()) error = cause instanceof Error ? cause.message : 'Business records could not be loaded.';
		} finally {
			if (request.current()) loading = false;
		}
	}
	async function save(path: string, payload: unknown, label: string) {
		if (busy || loading || !canManage) return;
		busy = true;
		error = '';
		message = '';
		try {
			await new MutationIntent(`network:${organization}:${path}`, `${base()}${path}`).run(payload, record, 'PUT');
			message = label;
			await load();
			return true;
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'The change could not be confirmed. Refresh before retrying.';
			return false;
		} finally {
			busy = false;
		}
	}
	async function saveBranch(b: Branch, isNew = false) {
		if (!b.name.trim()) {
			error = 'Enter a branch name.';
			return;
		}
		if (!b.id) b.id = crypto.randomUUID();
		const ok = await save(
			`/branches/${encodeURIComponent(b.id)}`,
			{ name: b.name, territory: b.territory, active: b.active, version: b.version },
			'Branch saved.'
		);
		if (ok && isNew) newBranch = { id: '', name: '', territory: '', active: true, version: 0 };
	}
	function assign(p: Partner) {
		void save(
			`/customers/${encodeURIComponent(p.business_id)}/assignment`,
			{ branch_id: p.branch_id, manager_id: p.manager_id, version: p.version },
			'Partner assignment saved.'
		);
	}
	const visible = $derived(partners.filter((p) => !filter || p.branch_id === filter));
	onMount(() => {
		void load();
		return () => reads.cancel();
	});
</script>

<svelte:head><title>Branches and account managers — Kredit</title></svelte:head>
<main class="shell workspace operations">
	<p class="eyebrow">Partners</p>
	<h1>Organize your network</h1>
	<p class="lede">Group customers by branch and territory, and give each relationship an account manager.</p>
	<VerifyIdentity />
	{#if message}<p role="status">{message}</p>{/if}
	{#if error}<p role="alert">{error}</p>
		<button disabled={busy} onclick={load}>Refresh records</button>{/if}
	<label
		>Business<select bind:value={organization} disabled={busy || loading} onchange={() => chooseWorkspace(organization)}
			>{#each businesses as b (b.id)}<option value={b.id}>{b.name}</option>{/each}</select
		></label
	>
	{#if loading}<p role="status">Loading your network…</p>{:else if !organization && !error}<p>
			<a href="/workspace/onboarding">Set up your business first.</a>
		</p>{:else if !error}
		<p>
			Assignments organize customer service. Team permissions continue to control access; a branch assignment does not
			grant access to a customer's own network.
		</p>
		<p>
			<a href={`/workspace/partners/access?organization=${encodeURIComponent(organization)}`}>Set team branch access</a>
		</p>
		<section>
			<h2>Branches and territories</h2>
			<p>
				Closing a branch also suspends branch-limited access to its customers. Company-wide staff retain access to those
				records.
			</p>
			{#if canManage}<form
					class="card"
					onsubmit={(e) => {
						e.preventDefault();
						void saveBranch(newBranch, true);
					}}
				>
					<h3>Add a branch</h3>
					<fieldset disabled={busy}>
						<label>Branch name<input bind:value={newBranch.name} maxlength="100" required /></label><label
							>Territory<input
								bind:value={newBranch.territory}
								maxlength="160"
								placeholder="For example, Lagos Mainland"
							/></label
						><button>Add branch</button>
					</fieldset>
				</form>{/if}
			{#if !branches.length}<p>No branches yet. You can still manage customers without a branch.</p>{/if}
			{#each branches as b (b.id)}<article class="card">
					<h3>{b.name}</h3>
					{#if canManage}<fieldset disabled={busy}>
							<label>Branch name for {b.name}<input bind:value={b.name} maxlength="100" /></label><label
								>Territory for {b.name}<input bind:value={b.territory} maxlength="160" /></label
							><label class="toggle"><input type="checkbox" bind:checked={b.active} />Open for new assignments</label
							><button onclick={() => saveBranch(b)}>Save branch</button>
						</fieldset>{:else}<p>
							{b.territory || 'No territory specified'} · {b.active ? 'Open' : 'Closed to new assignments'}
						</p>{/if}
				</article>{/each}
		</section>
		<section>
			<h2>Customer assignments</h2>
			<label
				>Filter by branch<select bind:value={filter}
					><option value="">All branches</option>{#each branches as b (b.id)}<option value={b.id}>{b.name}</option
						>{/each}</select
				></label
			>
			{#if !visible.length}<p>
					No customers match this view. <a
						href={`/workspace/partners/invitations?organization=${encodeURIComponent(organization)}`}
						>View invitations</a
					>.
				</p>{/if}
			{#each visible as p (p.business_id)}<article class="card">
					<h3>
						<a href={`/workspace/partners/customers/${p.business_id}?organization=${encodeURIComponent(organization)}`}
							>{p.name}</a
						>
					</h3>
					{#if p.manager_id && !p.manager_active}<p role="status">
							The assigned manager is no longer available. Choose a current team member.
						</p>{/if}
					{#if p.branch_id && branches.some((b) => b.id === p.branch_id && !b.active)}<p>
							This branch is closed to new assignments. Choose an open branch before saving changes.
						</p>{/if}
					<fieldset disabled={busy || !canManage}>
						<label
							>Branch for {p.name}<select bind:value={p.branch_id}
								><option value="">No branch assigned</option
								>{#each branches.filter((b) => b.active || b.id === p.branch_id) as b (b.id)}<option
										value={b.id}
										disabled={!b.active}>{b.name}{b.active ? '' : ' (closed)'}</option
									>{/each}</select
							></label
						><label
							>Account manager for {p.name}<select bind:value={p.manager_id}
								><option value="">No manager assigned</option
								>{#if p.manager_id && !managers.some((m) => m.id === p.manager_id)}<option value={p.manager_id} disabled
										>Previous manager unavailable</option
									>{/if}{#each managers as m (m.id)}<option value={m.id}>{m.name}</option>{/each}</select
							></label
						>{#if canManage}<button onclick={() => assign(p)}>Save assignment for {p.name}</button>{/if}
					</fieldset>
				</article>{/each}
		</section>{/if}
</main>

<style>
	.operations {
		max-width: 64rem;
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
		margin: 0;
		display: grid;
		gap: 0.8rem;
		min-width: 0;
	}
	label {
		display: grid;
		gap: 0.4rem;
		margin: 0.7rem 0;
	}
	input,
	select,
	button {
		font: inherit;
		padding: 0.7rem;
		border: 1px solid var(--color-border);
		border-radius: 0.6rem;
		background: var(--color-surface);
		color: inherit;
		max-width: 100%;
		box-sizing: border-box;
	}
	.toggle {
		display: flex;
		align-items: center;
	}
	button {
		cursor: pointer;
		justify-self: start;
	}
	button:disabled {
		opacity: 0.6;
		cursor: default;
	}
	h1 {
		font-size: clamp(2rem, 5vw, 3.5rem);
		line-height: 1.1;
	}
	.card h3 {
		overflow-wrap: anywhere;
	}
</style>
