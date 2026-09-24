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
	import type { KoboValue } from '$lib/money';
	import { kobo } from '$lib/records';
	const requests = new LatestRequest();
	import { onMount } from 'svelte';
	import Money from '$lib/components/Money.svelte';
	type AdminOrganization = {
		id: string;
		status: string;
		business_type: string;
		legal_name: string;
		trading_name: string;
		industry: string;
		outstanding_kobo: KoboValue;
		open_sales: number;
		member_count: number;
		version: number;
	};
	let organizations = $state<AdminOrganization[]>([]),
		query = $state(''),
		loading = $state(true),
		error = $state('');
	async function load() {
		const request = requests.begin();
		loading = true;
		error = '';
		try {
			const result = await checkedJSON(
				`/api/v1/ops/organizations?q=${encodeURIComponent(query.trim())}`,
				rows('organizations', (value): AdminOrganization => {
					const item = record(value);
					if (!Number.isSafeInteger(item.version) || Number(item.version) < 1)
						throw new Error('Missing current version');
					return {
						id: text(item.id),
						status: text(item.status),
						business_type: text(item.business_type),
						legal_name: text(item.legal_name),
						trading_name: optionalText(item.trading_name),
						industry: optionalText(item.industry),
						outstanding_kobo: kobo(item.outstanding_kobo),
						open_sales: optionalNumber(item.open_sales),
						member_count: optionalNumber(item.member_count),
						version: Number(item.version)
					};
				}),
				{ signal: request.signal }
			);
			if (request.current()) organizations = result;
		} catch (cause) {
			if (request.current()) error = publicError(cause, 'businesses');
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

<svelte:head><title>Businesses — Kredit admin</title></svelte:head>
<main class="shell workspace directory">
	<header>
		<div>
			<p class="eyebrow">Admin / Businesses</p>
			<h1>Every business on Kredit.</h1>
			<p>Registered and unregistered businesses appear together, with their real account status and money position.</p>
		</div>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				load();
			}}
		>
			<label>Find a business<input type="search" bind:value={query} placeholder="Business name or ID" /></label><button
				>Search</button
			>
		</form>
	</header>
	{#if error}<section role="alert">
			<p class="error">{error}</p>
			<button type="button" onclick={load}>Try again</button>
		</section>{:else if loading}<p>Loading businesses…</p>{:else}<section>
			{#each organizations as org (org.id)}<article>
					<div class="top">
						<div>
							<small>{org.business_type.replaceAll('_', ' ')}</small>
							<h2>{org.trading_name || org.legal_name}</h2>
							{#if org.trading_name}<p>{org.legal_name}</p>{/if}
						</div>
						<span class:attention={!['verified', 'active'].includes(org.status)}>{org.status.replaceAll('_', ' ')}</span
						>
					</div>
					<dl>
						<div>
							<dt>Money owed</dt>
							<dd><Money amountKobo={org.outstanding_kobo} /></dd>
						</div>
						<div>
							<dt>Open sales</dt>
							<dd>{org.open_sales}</dd>
						</div>
						<div>
							<dt>Team members</dt>
							<dd>{org.member_count}</dd>
						</div>
						<div>
							<dt>Industry</dt>
							<dd>{org.industry}</dd>
						</div>
					</dl>
					<footer>
						<code>{org.id}</code><a
							href={`/admin/controls?target_type=organization&target_id=${encodeURIComponent(org.id)}&organization_id=${encodeURIComponent(org.id)}&version=${org.version}&status=${encodeURIComponent(org.status)}`}
							>Manage business →</a
						>
					</footer>
				</article>{/each}
		</section>
		{#if !organizations.length}<div class="empty-state">
				<h2>No business found</h2>
				<p>Try the business name or exact organization ID.</p>
			</div>{/if}{/if}
</main>

<style>
	.directory > header {
		display: flex;
		align-items: end;
		justify-content: space-between;
		gap: 2rem;
		padding: 2rem 0;
		border-bottom: 3px solid var(--color-primary);
	}
	.directory h1 {
		margin: 0.4rem 0;
		font-family: var(--font-serif);
		font-size: clamp(2.5rem, 6vw, 4.5rem);
		font-weight: 500;
		line-height: 0.95;
	}
	.directory header p {
		max-width: 40rem;
	}
	.directory form {
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
	.directory > section {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 1rem;
		margin-top: 2rem;
	}
	article {
		display: grid;
		gap: 1.3rem;
		padding: 1.25rem;
		border: 1px solid var(--color-border);
		background: var(--color-surface);
	}
	.top {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
	}
	.top small {
		text-transform: capitalize;
		color: var(--color-foreground);
	}
	.top h2 {
		margin: 0.25rem 0;
		font-family: var(--font-serif);
		font-size: 1.5rem;
		font-weight: 500;
	}
	.top p {
		margin: 0;
		color: var(--color-foreground);
	}
	.top > span {
		align-self: start;
		padding: 0.3rem 0.5rem;
		background: var(--color-background);
		color: var(--color-positive);
		font-size: 0.7rem;
		font-weight: 850;
		text-transform: capitalize;
	}
	.top .attention {
		background: var(--color-background);
		color: var(--color-overdue);
	}
	dl {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		margin: 0;
		border-top: 1px solid var(--color-border);
		border-left: 1px solid var(--color-border);
	}
	dl div {
		padding: 0.7rem;
		border-right: 1px solid var(--color-border);
		border-bottom: 1px solid var(--color-border);
	}
	dt {
		color: var(--color-foreground);
		font-size: 0.7rem;
	}
	dd {
		margin: 0.25rem 0 0;
		font-weight: 800;
	}
	footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
	}
	code {
		overflow: hidden;
		color: var(--color-foreground);
		font-size: 0.68rem;
		text-overflow: ellipsis;
	}
	a {
		color: var(--color-primary);
		font-weight: 800;
		white-space: nowrap;
	}
	@media (max-width: 760px) {
		.directory > header {
			align-items: stretch;
			flex-direction: column;
		}
		.directory form {
			min-width: 0;
		}
		.directory > section {
			grid-template-columns: 1fr;
		}
	}
</style>
