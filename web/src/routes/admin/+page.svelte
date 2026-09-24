<script lang="ts">
	import { checkedJSON, publicError, record, text } from '$lib/api/reliable';
	import { onMount } from 'svelte';
	import AdminAttention from '$lib/components/AdminAttention.svelte';
	let overview = $state<Record<string, unknown> | null>(null),
		role = $state(''),
		error = $state(''),
		loading = $state(true);
	async function load() {
		loading = true;
		error = '';
		try {
			const data = await checkedJSON('/api/v1/ops/overview', (value) => {
				const body = record(value);
				return { overview: record(body.overview), role: text(body.role) };
			});
			overview = data.overview;
			role = data.role;
		} catch (cause) {
			error = publicError(cause, 'the platform overview');
		} finally {
			loading = false;
		}
	}
	onMount(load);
	const labels: { [key: string]: string } = {
		queued_jobs: 'Work in progress',
		failed_jobs: 'Jobs to retry',
		dead_letter_jobs: 'Stopped jobs',
		pending_outbox: 'Messages waiting',
		failed_outbox: 'Messages that failed',
		provider_failures: 'Provider problems',
		open_cases: 'Open support cases',
		open_disputes: 'Open disputes'
	};
	const destinations = [
		['Users', 'Find an account, check where it stands, or open a protected control.', '/admin/users', '01'],
		['Businesses', 'Review every business on Kredit, registered or not.', '/admin/organizations', '02'],
		['Money', 'Payments, bank debits and money still owed.', '/admin/money', '03'],
		['Support cases', 'Work through customer and business requests.', '/admin/cases', '04'],
		['Disputes', 'Review the proof and record a fair decision.', '/admin/disputes', '05'],
		['System health', 'Queues, providers, and anything that has stalled.', '/admin/diagnostics', '06']
	];
</script>

<svelte:head><title>Operations overview — Kredit</title></svelte:head>
<main class="shell workspace admin-home">
	<header>
		<div>
			<p class="eyebrow">Kredit admin</p>
			<h1>Run the whole platform from one place.</h1>
			<p class="lede">Help users, check businesses and money, settle problems and keep Kredit healthy.</p>
		</div>
		{#if role}<span class="role">{role.replaceAll('_', ' ')}</span>{/if}
	</header>
	{#if loading}<p aria-live="polite">Loading the latest position…</p>{:else if error}<section role="alert">
			<p class="error">{error}</p>
			<button type="button" onclick={load}>Try again</button>
		</section>{:else}<AdminAttention />
		<section class="health" aria-label="Work needing attention">
			{#each Object.entries(overview ?? {}) as [key, value] (key)}<a
					href={key.includes('job')
						? '/admin/jobs'
						: key.includes('provider') || key.includes('outbox')
							? '/admin/diagnostics'
							: key === 'open_cases'
								? '/admin/cases'
								: key === 'open_disputes'
									? '/admin/disputes'
									: '/admin'}
					><span>{labels[key] ?? key}</span><strong>{value}</strong><small
						>{Number(value) === 0 ? 'Nothing waiting' : 'Open and review →'}</small
					></a
				>{/each}
		</section>
		<div class="section-title">
			<p class="eyebrow">Admin work</p>
			<h2>Choose what you need to work on.</h2>
		</div>
		<section class="destinations">
			{#each destinations as item, i (i)}<a href={item[2]}
					><b>{item[3]}</b>
					<div>
						<h3>{item[0]}</h3>
						<p>{item[1]}</p>
					</div>
					<span>→</span></a
				>{/each}
		</section>{/if}
</main>

<style>
	.admin-home > header {
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: 2rem;
		padding: 2.5rem 0 2rem;
		border-bottom: 3px solid var(--color-primary);
	}
	.admin-home h1 {
		max-width: 14ch;
		margin: 0.5rem 0;
		font-family: var(--font-serif);
		font-size: clamp(2.6rem, 6vw, 5rem);
		font-weight: 500;
		line-height: 0.94;
		letter-spacing: -0.05em;
	}
	.role {
		flex: none;
		padding: 0.55rem 0.75rem;
		background: var(--color-primary);
		color: var(--color-on-primary);
		font-size: 0.72rem;
		font-weight: 800;
		text-transform: capitalize;
	}
	.health {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		border-left: 1px solid var(--color-border);
		border-top: 1px solid var(--color-border);
		margin: 2rem 0 4rem;
	}
	.health a {
		display: grid;
		gap: 0.65rem;
		min-height: 9rem;
		padding: 1rem;
		border-right: 1px solid var(--color-border);
		border-bottom: 1px solid var(--color-border);
		color: var(--color-primary);
		text-decoration: none;
		background: var(--color-surface);
	}
	.health a > span {
		font-size: 0.78rem;
		font-weight: 750;
	}
	.health strong {
		font-family: var(--font-serif);
		font-size: 2.6rem;
		font-weight: 500;
	}
	.health small {
		align-self: end;
		color: var(--color-foreground);
	}
	.section-title {
		display: flex;
		align-items: end;
		justify-content: space-between;
		border-bottom: 3px solid var(--color-primary);
	}
	.section-title h2 {
		font-family: var(--font-serif);
		font-size: clamp(1.8rem, 4vw, 3rem);
		font-weight: 500;
	}
	.destinations {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}
	.destinations a {
		display: grid;
		grid-template-columns: auto 1fr auto;
		gap: 1rem;
		min-height: 8rem;
		padding: 1.3rem 0;
		border-bottom: 1px solid var(--color-border);
		color: var(--color-primary);
		text-decoration: none;
	}
	.destinations a:nth-child(odd) {
		padding-right: 1.5rem;
		border-right: 1px solid var(--color-border);
	}
	.destinations a:nth-child(even) {
		padding-left: 1.5rem;
	}
	.destinations b {
		color: var(--color-primary);
	}
	.destinations h3 {
		margin: 0;
		font-family: var(--font-serif);
		font-size: 1.5rem;
		font-weight: 500;
	}
	.destinations p {
		margin: 0.4rem 0;
		color: var(--color-foreground);
	}
	.destinations > a > span {
		color: var(--color-accent);
		font-size: 1.3rem;
	}
	@media (max-width: 760px) {
		.admin-home > header {
			align-items: flex-start;
			flex-direction: column;
		}
		.health {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
		.destinations {
			grid-template-columns: 1fr;
		}
		.destinations a:nth-child(n) {
			padding-left: 0;
			padding-right: 0;
			border-right: 0;
		}
	}
	@media (max-width: 420px) {
		.health {
			grid-template-columns: 1fr;
		}
		.health a {
			min-height: 7rem;
		}
	}
</style>
