<script lang="ts">
	import { page } from '$app/state';
	import {
		checkedJSON,
		optionalNumber,
		optionalText,
		publicError,
		record,
		rows,
		text,
		LatestRequest
	} from '$lib/api/reliable';
	const requests = new LatestRequest();
	import { onMount } from 'svelte';
	type AdminUser = {
		id: string;
		status: string;
		display_name: string;
		identifier: string;
		organization_count: number;
		last_authenticated_at: string;
		version: number;
	};
	let users = $state<AdminUser[]>([]),
		query = $state(''),
		loading = $state(true),
		error = $state('');
	async function load() {
		const request = requests.begin();
		loading = true;
		error = '';
		try {
			const result = await checkedJSON(
				`/api/v1/ops/users?q=${encodeURIComponent(query.trim())}`,
				rows('users', (value): AdminUser => {
					const item = record(value);
					if (!Number.isSafeInteger(item.version) || Number(item.version) < 1)
						throw new Error('Missing current version');
					return {
						id: text(item.id),
						status: text(item.status),
						display_name: text(item.display_name),
						identifier: text(item.identifier),
						organization_count: optionalNumber(item.organization_count),
						last_authenticated_at: optionalText(item.last_authenticated_at),
						version: Number(item.version)
					};
				}),
				{ signal: request.signal }
			);
			if (request.current()) users = result;
		} catch (cause) {
			if (request.current()) error = publicError(cause, 'users');
		} finally {
			if (request.current()) loading = false;
		}
	}
	onMount(() => {
		query = page.url.searchParams.get('q') || '';
		void load();
		return () => requests.cancel();
	});
</script>

<svelte:head><title>Users — Kredit admin</title></svelte:head>
<main class="shell workspace admin-page">
	<header>
		<div>
			<p class="eyebrow">Admin / Users</p>
			<h1>Every Kredit user.</h1>
			<p>Find an account, understand its status and use protected controls when necessary.</p>
		</div>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				load();
			}}
		>
			<label>Find a user<input type="search" bind:value={query} placeholder="Name, email, phone or user ID" /></label
			><button>Search</button>
		</form>
	</header>
	{#if error}<section role="alert">
			<p class="error">{error}</p>
			<button type="button" onclick={load}>Try again</button>
		</section>{:else if loading}<p role="status">Loading users…</p>{:else}<div class="table-wrap">
			<table>
				<caption>{users.length} user{users.length === 1 ? '' : 's'} shown</caption><thead
					><tr><th>User</th><th>Status</th><th>Businesses</th><th>Last sign-in</th><th></th></tr></thead
				><tbody
					>{#each users as user (user.id)}<tr
							><td><strong>{user.display_name}</strong><small>{user.identifier}</small><code>{user.id}</code></td><td
								><span class:bad={user.status !== 'active'}>{user.status}</span></td
							><td>{user.organization_count}</td><td
								>{user.last_authenticated_at
									? new Date(user.last_authenticated_at).toLocaleString('en-NG')
									: 'Never'}</td
							><td
								><a
									href={`/admin/controls?target_type=user&target_id=${encodeURIComponent(user.id)}&status=${encodeURIComponent(user.status)}&version=${user.version}`}
									>Manage →</a
								></td
							></tr
						>{/each}</tbody
				>
			</table>
		</div>
		{#if !users.length}<section class="empty-state">
				<h2>No user found</h2>
				<p>Check the spelling or use the exact email, phone number or user ID.</p>
			</section>{/if}{/if}
</main>

<style>
	.admin-page > header {
		display: flex;
		align-items: end;
		justify-content: space-between;
		gap: 2rem;
		padding: 2rem 0;
		border-bottom: 3px solid var(--color-primary);
	}
	.admin-page h1 {
		margin: 0.4rem 0;
		font-family: var(--font-serif);
		font-size: clamp(2.5rem, 6vw, 4.5rem);
		font-weight: 500;
		line-height: 0.95;
	}
	.admin-page header p {
		max-width: 38rem;
	}
	.admin-page form {
		display: flex;
		align-items: end;
		gap: 0.5rem;
		min-width: min(100%, 28rem);
	}
	label {
		display: grid;
		flex: 1;
		gap: 0.35rem;
		font-size: 0.78rem;
		font-weight: 750;
	}
	input,
	button {
		min-height: 2.8rem;
		padding: 0.6rem 0.75rem;
		border: 1px solid var(--color-primary);
		background: var(--color-surface);
		font: inherit;
	}
	button {
		background: var(--color-primary);
		color: var(--color-on-primary);
		font-weight: 800;
	}
	.table-wrap {
		overflow-x: auto;
		margin-top: 2rem;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		background: var(--color-surface);
	}
	caption {
		text-align: left;
		padding: 0.75rem 0;
		font-weight: 800;
	}
	th,
	td {
		padding: 1rem;
		text-align: left;
		border-bottom: 1px solid var(--color-border);
		vertical-align: top;
	}
	td strong,
	td small,
	td code {
		display: block;
	}
	td small,
	td code {
		margin-top: 0.3rem;
		color: var(--color-foreground);
		font-size: 0.76rem;
	}
	td span {
		padding: 0.25rem 0.45rem;
		background: var(--color-background);
		color: var(--color-positive);
		font-size: 0.7rem;
		font-weight: 800;
		text-transform: capitalize;
	}
	.bad {
		background: var(--color-background) !important;
		color: var(--color-overdue) !important;
	}
	td a {
		color: var(--color-primary);
		font-weight: 800;
		white-space: nowrap;
	}
	@media (max-width: 760px) {
		.admin-page > header {
			align-items: stretch;
			flex-direction: column;
		}
		.admin-page form {
			min-width: 0;
		}
	}
</style>
