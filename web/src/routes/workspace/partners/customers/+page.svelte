<script lang="ts">
	import { onMount } from 'svelte';
	import { chooseWorkspace, requestedWorkspace } from '$lib/workspace-context';
	import { checkedJSON, LatestRequest, publicError, record, rows, optionalText } from '$lib/api/reliable';
	import { organization, type Organization } from '$lib/records';
	import { read as readConsumer, purchase } from '$lib/consumer';
	import { exactKobo, type KoboValue } from '$lib/money';
	import Money from '$lib/components/Money.svelte';

	type Row = { key: string; name: string; kind: 'Business' | 'Person'; owed: bigint; href: string; note: string };
	let businesses = $state<Organization[]>([]),
		organizationID = $state(''),
		list = $state<Row[]>([]),
		search = $state(''),
		loading = $state(true),
		error = $state(''),
		peopleError = $state('');
	const reads = new LatestRequest();
	const query = $derived(`?organization=${encodeURIComponent(organizationID)}`);
	const shown = $derived(list.filter((row) => row.name.toLowerCase().includes(search.trim().toLowerCase())));

	async function load() {
		const request = reads.begin();
		loading = true;
		error = '';
		peopleError = '';
		try {
			businesses = await checkedJSON('/api/v1/organizations', rows('organizations', organization), {
				signal: request.signal
			});
			organizationID = requestedWorkspace(businesses);
			if (!organizationID) {
				list = [];
				return;
			}
			const scope = organizationID,
				q = `?organization=${encodeURIComponent(scope)}`;
			const firms = await checkedJSON(
				`/api/v1/organizations/${encodeURIComponent(scope)}/customers`,
				rows('customers', (value) => {
					const row = record(value);
					return {
						id: optionalText(row.buyer_business_id) || optionalText(row.id),
						name: optionalText(row.trading_name) || optionalText(row.legal_name) || 'Customer',
						owed: exactKobo(row.outstanding_kobo as KoboValue) ?? 0n,
						state: optionalText(row.state ?? row.status)
					};
				}),
				{ signal: request.signal }
			);
			const next: Row[] = firms.map((firm) => ({
				key: `b:${firm.id}`,
				name: firm.name,
				kind: 'Business',
				owed: firm.owed,
				href: `/workspace/partners/customers/${encodeURIComponent(firm.id)}${q}`,
				note: ['invited', 'pending'].includes(firm.state.toLowerCase()) ? 'Has not finished joining' : ''
			}));
			try {
				const data = record(await readConsumer(`/api/v1/organizations/${encodeURIComponent(scope)}/consumer-sales`));
				// eslint-disable-next-line svelte/prefer-svelte-reactivity -- a local tally, never rendered directly
				const people = new Map<string, Row>();
				for (const sale of (Array.isArray(data.items) ? data.items : []).map(purchase)) {
					if (['cancelled', 'declined'].includes(sale.state)) continue;
					const row = people.get(sale.target) ?? {
						key: `p:${sale.target}`,
						name: sale.customer_name || sale.target,
						kind: 'Person' as const,
						owed: 0n,
						href: `/workspace/sales/consumers/${encodeURIComponent(sale.id)}${q}`,
						note: ''
					};
					if (sale.state === 'offered') row.note = 'Waiting for him to accept';
					else row.owed += BigInt(Math.trunc(sale.outstanding_kobo ?? 0));
					people.set(sale.target, row);
				}
				next.push(...people.values());
			} catch (cause) {
				peopleError = publicError(cause, 'credit given to persons');
			}
			if (request.current())
				list = next.sort((a, b) => (b.owed > a.owed ? 1 : b.owed < a.owed ? -1 : a.name.localeCompare(b.name)));
		} catch (cause) {
			if (request.current()) error = publicError(cause, 'your customers');
		} finally {
			if (request.current()) loading = false;
		}
	}
	onMount(() => {
		void load();
		return () => reads.cancel();
	});
</script>

<svelte:head><title>Customers — Kredit</title></svelte:head>
<main class="shell customers">
	<header>
		<h1>Customers</h1>
		<a class="primary" href="/workspace/give{query}">Give goods on credit</a>
	</header>
	{#if businesses.length > 1}<label class="switch"
			>Business<select bind:value={organizationID} onchange={() => chooseWorkspace(organizationID)}
				>{#each businesses as org (org.id)}<option value={org.id}>{org.trading_name || org.legal_name}</option
					>{/each}</select
			></label
		>{/if}
	{#if error}<p class="error" role="alert">{error} <button onclick={load}>Try again</button></p>{/if}
	{#if peopleError}<p class="error" role="alert">{peopleError}</p>{/if}
	{#if loading}<p role="status">Opening your customers…</p>
	{:else if !list.length && !error}<div class="empty-state">
			<h2>No customers yet</h2>
			<p>Give goods on credit to a business or a person, and he will show here.</p>
		</div>
	{:else if list.length}
		{#if list.length > 8}<label class="search">Find a customer<input type="search" bind:value={search} /></label>{/if}
		<div class="record-list">
			{#each shown as row (row.key)}<a class="record-row" href={row.href}
					><span><strong>{row.name}</strong><small>{row.kind}{row.note ? ` · ${row.note}` : ''}</small></span><span
						class="owed"><small>Owes you</small><strong><Money amountKobo={row.owed} /></strong></span
					></a
				>{:else}<p>No customer matches “{search}”.</p>{/each}
		</div>
	{/if}
</main>

<style>
	.customers {
		max-width: 48rem;
		padding-bottom: 3rem;
	}
	header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding-block: 1rem 1.25rem;
	}
	h1 {
		margin: 0;
		font-size: clamp(1.8rem, 5vw, 2.4rem);
	}
	.switch,
	.search {
		display: grid;
		gap: 0.4rem;
		max-width: 22rem;
		margin-bottom: 1rem;
	}
	select,
	input {
		font: inherit;
		padding: 0.7rem;
		border: 1px solid var(--color-border);
		border-radius: 0.35rem;
		background: var(--color-surface);
	}
	.record-list {
		display: grid;
		border-top: 1px solid var(--color-border);
	}
	.record-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		min-height: 3.5rem;
		padding: 0.8rem 0.25rem;
		border-bottom: 1px solid var(--color-border);
		color: inherit;
		text-decoration: none;
	}
	.record-row span {
		display: grid;
		gap: 0.2rem;
	}
	.record-row small {
		color: var(--color-muted);
	}
	.owed {
		text-align: right;
		font-variant-numeric: tabular-nums;
	}
	.empty-state {
		padding: 1.5rem;
		border: 1px dashed var(--color-border);
		border-radius: 0.5rem;
	}
	.empty-state h2 {
		margin: 0;
		font-size: 1.1rem;
	}
	.empty-state p {
		color: var(--color-muted);
	}
	@media (max-width: 560px) {
		header {
			align-items: start;
			flex-direction: column;
		}
	}
</style>
