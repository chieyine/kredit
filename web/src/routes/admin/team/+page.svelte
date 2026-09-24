<script lang="ts">
	import { onMount } from 'svelte';
	import { lagosISO, localTime } from '$lib/admin-client';
	import { checkedJSON, rows, record, publicError, text, LatestRequest } from '$lib/api/reliable';

	import { MutationIntent } from '$lib/api/mutation';
	const requests = new LatestRequest();
	const searches = new LatestRequest();
	// eslint-disable-next-line svelte/prefer-svelte-reactivity -- idempotency keys are never rendered
	const intents = new Map<string, MutationIntent>();
	function intent(url: string) {
		let value = intents.get(url);
		if (!value) {
			value = new MutationIntent('admin-team', url);
			intents.set(url, value);
		}
		return value;
	}
	function member(value: unknown): TeamMember {
		const item = record(value);
		for (const key of ['assignment_id', 'user_id', 'display_name', 'identifier', 'role']) text(item[key]);
		if (item.expires_at != null) text(item.expires_at);
		return item as TeamMember;
	}
	function userResult(value: unknown): UserResult {
		const item = record(value);
		for (const key of ['id', 'display_name', 'identifier', 'status']) text(item[key]);
		return item as UserResult;
	}

	type UserResult = { id: string; display_name: string; identifier: string; status: string };
	type TeamMember = {
		assignment_id: string;
		user_id: string;
		display_name: string;
		identifier: string;
		role: string;
		expires_at?: string;
	};

	let members = $state<TeamMember[]>([]);
	let userResults = $state<UserResult[]>([]);
	let selectedUser = $state<UserResult | null>(null);
	let userQuery = $state('');
	let role = $state('support_agent');
	let reason = $state('');
	let expires = $state('');
	let loading = $state(true);
	let searching = $state(false);
	let busy = $state(false);
	let message = $state('');
	let error = $state('');
	let revokeTarget = $state<TeamMember | null>(null);
	let revokeReason = $state('');
	let grantReview = $state<{ user: UserResult; role: string; reason: string; expires_at: string } | null>(null);
	const roleEffect = (value: string) =>
		({
			support_agent: 'Handle support requests within assigned access.',
			compliance_reviewer: 'Reviews compliance and audit evidence, and what providers returned.',
			dispute_reviewer: 'Review and decide reported sale problems.',
			finance_operator: 'Propose financial corrections; approval follows the current governance rules.',
			policy_manager: 'Propose changes to business policies.',
			approver: 'Approve eligible changes under the current governance rules.',
			access_administrator: 'Manage administrator access and role assignments.',
			platform_admin: 'Manage platform operations, financial workflows and permitted administrator access.'
		})[value] ?? 'Review the role permissions before granting access.';
	function previewGrant(event: SubmitEvent) {
		event.preventDefault();
		if (!selectedUser || busy) return;
		try {
			grantReview = {
				user: { ...selectedUser },
				role,
				reason: reason.trim(),
				expires_at: expires ? lagosISO(expires) : ''
			};
			message = '';
		} catch {
			message = 'Check the access expiry date.';
		}
	}

	async function load() {
		const request = requests.begin();
		loading = true;
		error = '';
		try {
			const result = await checkedJSON('/api/v1/ops/team', rows('members', member), { signal: request.signal });
			if (request.current()) members = result;
		} catch (cause) {
			if (request.current()) error = publicError(cause, 'the admin team');
		} finally {
			if (request.current()) loading = false;
		}
	}

	async function findUser(event: SubmitEvent) {
		event.preventDefault();
		if (searching || busy || userQuery.trim().length < 2) return;
		const request = searches.begin();
		searching = true;
		selectedUser = null;
		userResults = [];
		message = '';
		try {
			const result = await checkedJSON(
				`/api/v1/ops/users?q=${encodeURIComponent(userQuery.trim())}&limit=10`,
				rows('users', userResult),
				{ signal: request.signal }
			);
			if (request.current()) userResults = result;
		} catch (cause) {
			if (request.current()) message = publicError(cause, 'matching users');
		} finally {
			if (request.current()) searching = false;
		}
	}

	function chooseUser(user: UserResult) {
		selectedUser = user;
		userResults = [];
		userQuery = user.display_name;
	}

	async function grant() {
		if (!grantReview || busy) return;
		const reviewed = grantReview;
		busy = true;
		message = '';
		try {
			const target = reviewed.user.id;
			const requestedRole = reviewed.role;
			await intent(`/api/v1/ops/team/${encodeURIComponent(target)}/roles`).run(
				{ role: reviewed.role, reason: reviewed.reason, expires_at: reviewed.expires_at },
				(value) => {
					const saved = member(record(value).member);
					if (saved.user_id !== target || saved.role !== requestedRole) throw new Error('Access was not confirmed');
					return saved;
				}
			);
			message = 'Admin access was granted and recorded.';
			grantReview = null;
			selectedUser = null;
			userQuery = '';
			reason = '';
			expires = '';
			await load();
		} catch (cause) {
			message = cause instanceof Error ? cause.message : 'Access could not be confirmed.';
		} finally {
			busy = false;
		}
	}

	async function revoke(event: SubmitEvent) {
		event.preventDefault();
		if (!revokeTarget || busy) return;
		busy = true;
		message = '';
		try {
			await intent(`/api/v1/ops/team/roles/${encodeURIComponent(revokeTarget.assignment_id)}`).run(
				{ reason: revokeReason },
				(value) => {
					if (record(value).revoked !== true) throw new Error('Removal was not confirmed');
					return true;
				},
				'DELETE'
			);
			message = 'Admin access was removed.';
			revokeTarget = null;
			revokeReason = '';
			await load();
		} catch (cause) {
			message = cause instanceof Error ? cause.message : 'Access removal could not be confirmed.';
		} finally {
			busy = false;
		}
	}

	onMount(() => {
		void load();
		return () => {
			requests.cancel();
			searches.cancel();
		};
	});
</script>

<svelte:head><title>Admin team — Kredit</title></svelte:head>

<main class="shell workspace team">
	<header>
		<p class="eyebrow">Admin / Team</p>
		<h1>Who can run Kredit.</h1>
		<p>
			Admin access is separate from business accounts. Every change needs recent verification, and every change is
			recorded permanently.
		</p>
	</header>
	{#if message}<p class="notice" role="status">{message}</p>{/if}
	{#if error}
		<p class="error" role="alert">{error}</p>
		<button onclick={load}>Try again</button>
	{:else}
		<div class="columns">
			<section>
				<h2>Current admin access</h2>
				{#if loading}<p>Loading the admin team…</p>{:else}
					{#each members as item, i (i)}
						<article>
							<div><strong>{item.display_name}</strong><span>{item.identifier}</span></div>
							<div>
								<b>{item.role.replaceAll('_', ' ')}</b><small
									>{item.expires_at ? `Ends ${localTime(item.expires_at)}` : 'No automatic end date'}</small
								>
							</div>
							<button
								disabled={busy}
								onclick={() => {
									revokeTarget = item;
									revokeReason = '';
								}}>Remove access</button
							>
						</article>
					{:else}<p>No active admin roles were found.</p>{/each}
				{/if}
				{#if revokeTarget}
					<form class="revoke" onsubmit={revoke}>
						<h3>Remove {revokeTarget.role.replaceAll('_', ' ')} access?</h3>
						<p>{revokeTarget.display_name} will immediately lose this admin role.</p>
						<label
							>Reason<textarea
								disabled={busy}
								bind:value={revokeReason}
								minlength="8"
								maxlength="1000"
								rows="3"
								required
							></textarea></label
						>
						<div>
							<button type="button" disabled={busy} onclick={() => (revokeTarget = null)}>Keep access</button><button
								class="danger"
								disabled={busy}>Remove access</button
							>
						</div>
					</form>
				{/if}
			</section>

			<section class="grant">
				<h2>Give admin access</h2>
				<form class="user-search" onsubmit={findUser}>
					<label
						>Find the person<input
							type="search"
							disabled={busy || searching}
							bind:value={userQuery}
							minlength="2"
							placeholder="Name, email or phone number"
							required
						/></label
					><button disabled={searching || busy}>{searching ? 'Searching…' : 'Find user'}</button>
				</form>
				{#if userResults.length}
					<div class="results" aria-label="User search results">
						{#each userResults as user, i (i)}<button type="button" onclick={() => chooseUser(user)}
								><strong>{user.display_name}</strong><span>{user.identifier} · {user.status}</span></button
							>{/each}
					</div>
				{:else if userQuery && !selectedUser && !searching}<p class="hint">
						Search and choose one Kredit user before giving access.
					</p>{/if}
				{#if selectedUser}
					<form class="grant-form" onsubmit={previewGrant}>
						<div class="selected">
							<small>Selected user</small><strong>{selectedUser.display_name}</strong><span
								>{selectedUser.identifier}</span
							>
						</div>
						<label
							>Role<select disabled={busy} bind:value={role}
								><option value="support_agent">Support agent</option><option value="compliance_reviewer"
									>Compliance reviewer</option
								><option value="dispute_reviewer">Dispute reviewer</option><option value="finance_operator"
									>Financial operator: propose corrections and date changes</option
								><option value="policy_manager">Policy manager: propose business policies</option><option
									value="approver">Approver: independently approve changes</option
								><option value="access_administrator">Access administrator: manage admin team</option><option
									value="platform_admin">Platform administrator</option
								></select
							></label
						>
						<label
							>Access ends (Lagos time) <small>optional</small><input
								type="datetime-local"
								disabled={busy}
								bind:value={expires}
							/></label
						>
						<label
							>Why are you giving access?<textarea
								disabled={busy}
								bind:value={reason}
								minlength="8"
								maxlength="1000"
								rows="4"
								required
							></textarea></label
						>
						<button class="primary" disabled={busy}>{busy ? 'Saving…' : 'Review this access'}</button>
					</form>
					{#if grantReview}<section aria-label="Review administrator access">
							<h3>Confirm this access change</h3>
							<p>
								<strong>{grantReview.user.display_name}</strong> ({grantReview.user.identifier}) will receive
								<strong>{grantReview.role.replaceAll('_', ' ')}</strong> access.
							</p>
							<p>{roleEffect(grantReview.role)}</p>
							<p>
								Ends: {grantReview.expires_at ? localTime(grantReview.expires_at) : 'No expiry. Active until revoked'}
							</p>
							<p>Recorded reason: {grantReview.reason}</p>
							<p>The server rechecks your authority before saving. This change is kept in the audit trail.</p>
							<button type="button" disabled={busy} onclick={grant}>Grant the reviewed access</button><button
								type="button"
								disabled={busy}
								onclick={() => (grantReview = null)}>Back to editing</button
							>
						</section>{/if}
				{/if}
			</section>
		</div>
	{/if}
</main>

<style>
	.team > header {
		padding: 2rem 0;
		border-bottom: 3px solid var(--color-primary);
	}
	.team h1 {
		margin: 0.4rem 0;
		font-family: var(--font-serif);
		font-size: clamp(1.9rem, 3.4vw, 2.6rem);
		line-height: 1.1;
	}
	.team header p {
		max-width: 44rem;
	}
	.columns {
		display: grid;
		grid-template-columns: 1.4fr 0.8fr;
		gap: 2rem;
		margin-top: 2rem;
	}
	.columns > section {
		min-width: 0;
	}
	.columns h2 {
		font-family: var(--font-serif);
		font-size: 1.8rem;
		font-weight: 500;
	}
	article {
		display: grid;
		grid-template-columns: 1fr auto auto;
		align-items: center;
		gap: 1rem;
		padding: 1rem 0;
		border-bottom: 1px solid var(--color-border);
	}
	article strong,
	article span,
	article b,
	article small {
		display: block;
	}
	article span,
	article small {
		margin-top: 0.25rem;
		color: var(--color-foreground);
		font-size: 0.72rem;
	}
	article b {
		text-transform: capitalize;
	}
	article button {
		padding: 0.55rem;
		border: 1px solid var(--color-overdue);
		background: transparent;
		color: var(--color-overdue);
		font: inherit;
		font-size: 0.74rem;
		font-weight: 800;
	}
	.grant {
		padding: 1.2rem;
		background: var(--color-primary);
		color: var(--color-on-primary);
	}
	.grant form,
	.grant label,
	.revoke,
	.revoke label {
		display: grid;
		gap: 0.7rem;
	}
	.grant label,
	.revoke label {
		gap: 0.3rem;
		font-size: 0.76rem;
		font-weight: 750;
	}
	.grant input,
	.grant select,
	.grant textarea,
	.grant button,
	.revoke textarea,
	.revoke button {
		box-sizing: border-box;
		width: 100%;
		padding: 0.7rem;
		border: 1px solid var(--color-border-strong);
		background: var(--color-foreground);
		color: var(--color-on-primary);
		font: inherit;
	}
	.user-search {
		grid-template-columns: 1fr auto;
		align-items: end;
	}
	.user-search button {
		width: auto;
		white-space: nowrap;
	}
	.results {
		display: grid;
		margin: 0.6rem 0;
		border: 1px solid var(--color-border-strong);
	}
	.results button {
		display: grid;
		gap: 0.2rem;
		text-align: left;
		border-width: 0 0 1px;
	}
	.results button:last-child {
		border-bottom: 0;
	}
	.results span,
	.selected span,
	.selected small,
	.hint {
		color: var(--color-muted);
		font-size: 0.74rem;
	}
	.selected {
		display: grid;
		gap: 0.2rem;
		padding: 0.75rem;
		border-left: 3px solid var(--color-accent);
		background: var(--color-surface-muted);
	}
	.selected strong {
		font-size: 1.05rem;
	}
	.grant-form {
		margin-top: 1rem;
	}
	.grant .primary {
		background: var(--color-accent);
		color: var(--color-on-primary);
		font-weight: 850;
	}
	.revoke {
		margin-top: 1rem;
		padding: 1rem;
		border: 2px solid var(--color-overdue);
		background: var(--color-background);
	}
	.revoke h3,
	.revoke p {
		margin: 0;
	}
	.revoke textarea {
		background: var(--color-surface);
		color: var(--color-primary);
		border-color: var(--color-border);
	}
	.revoke > div {
		display: flex;
		gap: 0.5rem;
	}
	.revoke button {
		background: var(--color-surface);
		color: var(--color-primary);
		border-color: var(--color-primary);
	}
	.revoke .danger {
		background: var(--color-overdue);
		color: var(--color-on-primary);
		border-color: var(--color-overdue);
	}
	@media (max-width: 800px) {
		.columns {
			grid-template-columns: 1fr;
		}
	}
	@media (max-width: 540px) {
		article {
			grid-template-columns: 1fr;
		}
		article button {
			justify-self: start;
		}
		.user-search {
			grid-template-columns: 1fr;
		}
		.user-search button {
			width: 100%;
		}
	}
</style>
