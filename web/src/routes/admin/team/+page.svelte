<script lang="ts">
	import { onMount } from 'svelte';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';

	type UserResult = { id: string; display_name: string; identifier: string; status: string };
	type TeamMember = { assignment_id: string; user_id: string; display_name: string; identifier: string; role: string; expires_at?: string };

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

	async function load() {
		loading = true;
		error = '';
		const response = await fetch('/api/v1/ops/team', { credentials: 'include' });
		const data = await response.json().catch(() => ({}));
		loading = false;
		if (!response.ok) { error = data.detail ?? 'Admin team could not be loaded.'; return; }
		members = data.members ?? [];
	}

	async function findUser(event: SubmitEvent) {
		event.preventDefault();
		if (userQuery.trim().length < 2) return;
		searching = true;
		selectedUser = null;
		const response = await fetch(`/api/v1/ops/users?q=${encodeURIComponent(userQuery.trim())}&limit=10`, { credentials: 'include' });
		const data = await response.json().catch(() => ({}));
		searching = false;
		if (!response.ok) { message = data.detail ?? 'Users could not be searched.'; return; }
		userResults = data.users ?? [];
	}

	function chooseUser(user: UserResult) {
		selectedUser = user;
		userResults = [];
		userQuery = user.display_name;
	}

	async function grant(event: SubmitEvent) {
		event.preventDefault();
		if (!selectedUser) return;
		busy = true;
		message = '';
		const response = await fetch(`/api/v1/ops/team/${selectedUser.id}/roles`, {
			method: 'POST', credentials: 'include',
			headers: { 'Content-Type': 'application/json', 'Idempotency-Key': idempotencyKey(), ...csrfHeaders() },
			body: JSON.stringify({ role, reason, expires_at: expires ? new Date(expires).toISOString() : '' })
		});
		const data = await response.json().catch(() => ({}));
		busy = false;
		message = response.ok ? 'Admin access was granted and recorded.' : (data.detail ?? 'Access could not be granted.');
		if (response.ok) { selectedUser = null; userQuery = ''; reason = ''; expires = ''; await load(); }
	}

	async function revoke(event: SubmitEvent) {
		event.preventDefault();
		if (!revokeTarget) return;
		busy = true;
		const response = await fetch(`/api/v1/ops/team/roles/${revokeTarget.assignment_id}`, {
			method: 'DELETE', credentials: 'include',
			headers: { 'Content-Type': 'application/json', 'Idempotency-Key': idempotencyKey(), ...csrfHeaders() },
			body: JSON.stringify({ reason: revokeReason })
		});
		const data = await response.json().catch(() => ({}));
		busy = false;
		message = response.ok ? 'Admin access was removed.' : (data.detail ?? 'Access could not be removed.');
		if (response.ok) { revokeTarget = null; revokeReason = ''; await load(); }
	}

	onMount(load);
</script>

<svelte:head><title>Admin team — Kredit</title></svelte:head>

<main class="shell workspace team">
	<header><p class="eyebrow">Admin / Team</p><h1>Who can run Kredit.</h1><p>Admin access is separate from business accounts. Every change needs recent verification and is permanently recorded.</p></header>
	{#if message}<p class="notice" role="status">{message}</p>{/if}
	{#if error}
		<p class="error" role="alert">{error}</p>
	{:else}
		<div class="columns">
			<section>
				<h2>Current admin access</h2>
				{#if loading}<p>Loading the admin team…</p>{:else}
					{#each members as item}
						<article><div><strong>{item.display_name}</strong><span>{item.identifier}</span></div><div><b>{item.role.replaceAll('_', ' ')}</b><small>{item.expires_at ? `Ends ${new Date(item.expires_at).toLocaleString('en-NG')}` : 'No automatic end date'}</small></div><button disabled={busy} onclick={() => { revokeTarget = item; revokeReason = ''; }}>Remove access</button></article>
					{:else}<p>No active admin roles were found.</p>{/each}
				{/if}
				{#if revokeTarget}
					<form class="revoke" onsubmit={revoke}><h3>Remove {revokeTarget.role.replaceAll('_', ' ')} access?</h3><p>{revokeTarget.display_name} will immediately lose this admin role.</p><label>Reason<textarea bind:value={revokeReason} minlength="8" maxlength="1000" rows="3" required></textarea></label><div><button type="button" onclick={() => revokeTarget = null}>Keep access</button><button class="danger" disabled={busy}>Remove access</button></div></form>
				{/if}
			</section>

			<section class="grant">
				<h2>Give admin access</h2>
				<form class="user-search" onsubmit={findUser}><label>Find the person<input type="search" bind:value={userQuery} minlength="2" placeholder="Name, email or phone number" required /></label><button disabled={searching}>{searching ? 'Searching…' : 'Find user'}</button></form>
				{#if userResults.length}
					<div class="results" aria-label="User search results">{#each userResults as user}<button type="button" onclick={() => chooseUser(user)}><strong>{user.display_name}</strong><span>{user.identifier} · {user.status}</span></button>{/each}</div>
				{:else if userQuery && !selectedUser && !searching}<p class="hint">Search and choose one Kredit user before giving access.</p>{/if}
				{#if selectedUser}
					<form class="grant-form" onsubmit={grant}>
						<div class="selected"><small>Selected user</small><strong>{selectedUser.display_name}</strong><span>{selectedUser.identifier}</span></div>
						<label>Role<select bind:value={role}><option value="support_agent">Support agent</option><option value="compliance_reviewer">Compliance reviewer</option><option value="dispute_reviewer">Dispute reviewer</option><option value="finance_operator">Financial operator — propose corrections and date changes</option><option value="policy_manager">Policy manager — propose business policies</option><option value="approver">Approver — independently approve changes</option><option value="access_administrator">Access administrator — manage admin team</option><option value="platform_admin">Platform administrator</option></select></label>
						<label>Access ends <small>optional</small><input type="datetime-local" bind:value={expires} /></label>
						<label>Why are you giving access?<textarea bind:value={reason} minlength="8" maxlength="1000" rows="4" required></textarea></label>
						<button class="primary" disabled={busy}>{busy ? 'Saving…' : 'Give this access'}</button>
					</form>
				{/if}
			</section>
		</div>
	{/if}
</main>

<style>
	.team>header{padding:2rem 0;border-bottom:3px solid #17181b}.team h1{margin:.4rem 0;font-family:Georgia,serif;font-size:clamp(2.5rem,6vw,4.5rem);font-weight:500;line-height:.95}.team header p{max-width:44rem}.columns{display:grid;grid-template-columns:1.4fr .8fr;gap:2rem;margin-top:2rem}.columns>section{min-width:0}.columns h2{font-family:Georgia,serif;font-size:1.8rem;font-weight:500}article{display:grid;grid-template-columns:1fr auto auto;align-items:center;gap:1rem;padding:1rem 0;border-bottom:1px solid #bbb6ac}article strong,article span,article b,article small{display:block}article span,article small{margin-top:.25rem;color:#66635e;font-size:.72rem}article b{text-transform:capitalize}article button{padding:.55rem;border:1px solid #9b351f;background:transparent;color:#9b351f;font:inherit;font-size:.74rem;font-weight:800}.grant{padding:1.2rem;background:#17181b;color:#fff}.grant form,.grant label,.revoke,.revoke label{display:grid;gap:.7rem}.grant label,.revoke label{gap:.3rem;font-size:.76rem;font-weight:750}.grant input,.grant select,.grant textarea,.grant button,.revoke textarea,.revoke button{box-sizing:border-box;width:100%;padding:.7rem;border:1px solid #55565b;background:#242529;color:white;font:inherit}.user-search{grid-template-columns:1fr auto;align-items:end}.user-search button{width:auto;white-space:nowrap}.results{display:grid;margin:.6rem 0;border:1px solid #55565b}.results button{display:grid;gap:.2rem;text-align:left;border-width:0 0 1px}.results button:last-child{border-bottom:0}.results span,.selected span,.selected small,.hint{color:#b8b9bc;font-size:.74rem}.selected{display:grid;gap:.2rem;padding:.75rem;border-left:4px solid #ff6848;background:#242529}.selected strong{font-size:1.05rem}.grant-form{margin-top:1rem}.grant .primary{background:#ff6848;color:#17181b;font-weight:850}.revoke{margin-top:1rem;padding:1rem;border:2px solid #9b351f;background:#fff4ee}.revoke h3,.revoke p{margin:0}.revoke textarea{background:white;color:#17181b;border-color:#aaa}.revoke>div{display:flex;gap:.5rem}.revoke button{background:white;color:#17181b;border-color:#17181b}.revoke .danger{background:#9b351f;color:white;border-color:#9b351f}@media(max-width:800px){.columns{grid-template-columns:1fr}}@media(max-width:540px){article{grid-template-columns:1fr}article button{justify-self:start}.user-search{grid-template-columns:1fr}.user-search button{width:100%}}
</style>
