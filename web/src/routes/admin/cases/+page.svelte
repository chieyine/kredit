<script lang="ts">
	import { checkedJSON, optionalText, publicError, record, text, rows, LatestRequest } from '$lib/api/reliable';
	const requests = new LatestRequest();
	import { onMount } from 'svelte';
	type SupportCase = {
		id: string;
		state: string;
		subject_type: string;
		subject_id: string;
		break_glass: boolean;
		updated_at: string;
	};
	let items = $state<SupportCase[]>([]),
		stateFilter = $state(''),
		loading = $state(true),
		error = $state('');
	async function load() {
		const request = requests.begin();
		loading = true;
		error = '';
		try {
			const result = await checkedJSON(
				`/api/v1/ops/cases?state=${encodeURIComponent(stateFilter)}`,
				rows('cases', (value): SupportCase => {
					const item = record(value);
					return {
						id: text(item.id),
						state: text(item.state),
						subject_type: text(item.subject_type),
						subject_id: optionalText(item.subject_id),
						break_glass: item.break_glass === true,
						updated_at: optionalText(item.updated_at)
					};
				}),
				{ signal: request.signal }
			);
			if (request.current()) items = result;
		} catch (cause) {
			if (request.current()) error = publicError(cause, 'support cases');
		} finally {
			if (request.current()) loading = false;
		}
	}
	onMount(() => {
		void load();
		return () => requests.cancel();
	});
</script>

<svelte:head><title>Support cases — Kredit admin</title></svelte:head>
<main class="shell workspace queue">
	<header>
		<div>
			<p class="eyebrow">Admin / Support</p>
			<h1>Help people reach an answer.</h1>
			<p>Every case carries its own history. Who owns it, and why it was opened.</p>
		</div>
		<label
			>Show<select bind:value={stateFilter} onchange={load}
				><option value="">All cases</option><option value="OPEN">Open</option><option value="IN_PROGRESS"
					>Being handled</option
				><option value="RESOLVED">Resolved</option><option value="CLOSED">Closed</option></select
			></label
		>
	</header>
	{#if error}<section role="alert">
			<p class="error">{error}</p>
			<button type="button" onclick={load}>Try again</button>
		</section>{:else if loading}<p>Loading support cases…</p>{:else}<section>
			{#each items as item (item.id)}<a href={`/admin/cases/${encodeURIComponent(item.id)}`}
					><span class:urgent={item.break_glass}>{item.break_glass ? 'Urgent access' : 'Support case'}</span>
					<div>
						<h2>{item.subject_type.replaceAll('_', ' ')}</h2>
						<p>{item.subject_id}</p>
						<small>Updated {new Date(item.updated_at).toLocaleString('en-NG')}</small>
					</div>
					<strong>{item.state.replaceAll('_', ' ')}</strong><b>→</b></a
				>{:else}<div class="empty-state">
					<h2>No cases in this view</h2>
					<p>Choose another status or return later.</p>
				</div>{/each}
		</section>{/if}
</main>

<style>
	.queue > header {
		display: flex;
		align-items: end;
		justify-content: space-between;
		gap: 2rem;
		padding: 2rem 0;
		border-bottom: 3px solid var(--color-primary);
	}
	.queue h1 {
		margin: 0.4rem 0;
		font-family: var(--font-serif);
		font-size: clamp(1.9rem, 3.4vw, 2.6rem);
		line-height: 1.1;
	}
	.queue label {
		display: grid;
		gap: 0.35rem;
		font-size: 0.75rem;
		font-weight: 800;
	}
	.queue select {
		min-height: 2.8rem;
		padding: 0.5rem;
		border: 1px solid var(--color-primary);
		background: var(--color-surface);
	}
	.queue section {
		display: grid;
	}
	.queue section > a {
		display: grid;
		grid-template-columns: 8rem 1fr auto auto;
		align-items: center;
		gap: 1rem;
		padding: 1.2rem 0;
		border-bottom: 1px solid var(--color-border);
		color: var(--color-primary);
		text-decoration: none;
	}
	.queue a > span {
		font-size: 0.7rem;
		font-weight: 850;
		color: var(--color-primary);
	}
	.queue .urgent {
		color: var(--color-overdue);
	}
	.queue h2 {
		margin: 0;
		font-family: var(--font-serif);
		font-size: 1.35rem;
		font-weight: 500;
		text-transform: capitalize;
	}
	.queue p,
	.queue small {
		margin: 0.25rem 0;
		color: var(--color-foreground);
	}
	.queue a > strong {
		font-size: 0.72rem;
		text-transform: capitalize;
	}
	.queue a > b {
		color: var(--color-accent);
	}
	@media (max-width: 650px) {
		.queue > header {
			align-items: stretch;
			flex-direction: column;
		}
		.queue section > a {
			grid-template-columns: 1fr auto;
		}
		.queue a > span,
		.queue a > div {
			grid-column: 1;
		}
		.queue a > strong,
		.queue a > b {
			grid-column: 2;
		}
		.queue a > strong {
			grid-row: 1;
		}
		.queue a > b {
			grid-row: 2;
		}
	}
</style>
