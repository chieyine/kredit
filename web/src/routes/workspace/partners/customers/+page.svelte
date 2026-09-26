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
		kindFilter = $state('all'),
		loading = $state(true),
		error = $state(''),
		peopleError = $state('');
	const reads = new LatestRequest();
	const query = $derived(`?organization=${encodeURIComponent(organizationID)}`);
	const shown = $derived(
		list.filter(
			(row) =>
				(kindFilter === 'all' || row.kind === kindFilter) &&
				row.name.toLowerCase().includes(search.trim().toLowerCase())
		)
	);
	const totalOwed = $derived(list.reduce((sum, row) => sum + row.owed, 0n));
	const businessName = $derived(
		businesses.find((org) => org.id === organizationID)?.trading_name ||
			businesses.find((org) => org.id === organizationID)?.legal_name ||
			''
	);
	const initials = (name: string) =>
		name
			.split(/\s+/)
			.filter((part) => /[A-Za-z]/.test(part[0] ?? ''))
			.slice(0, 2)
			.map((part) => part[0]!.toUpperCase())
			.join('') || '#';

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
<main class="shell k-page">
	<header class="k-head">
		<div>
			<p class="k-eyebrow">{businessName || 'Your business'}</p>
			<h1>Customers</h1>
			{#if list.length}<p>
					{list.length} customer{list.length === 1 ? '' : 's'} · <Money amountKobo={totalOwed} /> owed to you
				</p>{/if}
		</div>
		<a class="primary" href="/workspace/give{query}">Give goods on credit <span aria-hidden="true">→</span></a>
	</header>
	{#if businesses.length > 1}<label class="field"
			>Business<select bind:value={organizationID} onchange={() => chooseWorkspace(organizationID)}
				>{#each businesses as org (org.id)}<option value={org.id}>{org.trading_name || org.legal_name}</option
					>{/each}</select
			></label
		>{/if}
	{#if error}<p class="error" role="alert">{error} <button onclick={load}>Try again</button></p>{/if}
	{#if peopleError}<p class="error" role="alert">{peopleError}</p>{/if}
	{#if loading}<p role="status" class="loading">Opening your customers…</p>
	{:else if !list.length && !error}<div class="k-ledger k-empty">
			<h3>No customers yet</h3>
			<p>Give goods on credit to a business or a person, and he will show here.</p>
			<a class="primary" href="/workspace/give{query}">Give goods on credit</a>
		</div>
	{:else if list.length}
		<div class="tools">
			<label class="field search"
				>Find a customer<input type="search" bind:value={search} placeholder="Name or phone number" /></label
			>
			<div class="filters" role="group" aria-label="Show">
				{#each [['all', 'All'], ['Business', 'Businesses'], ['Person', 'Persons']] as [value, label] (value)}<button
						type="button"
						class:on={kindFilter === value}
						aria-pressed={kindFilter === value}
						onclick={() => (kindFilter = value)}>{label}</button
					>{/each}
			</div>
		</div>
		<div class="k-ledger record-list">
			<div class="k-ledger-head"><span>Customer</span><span>Owes you</span></div>
			{#each shown as row (row.key)}<a class="k-row" href={row.href}
					><span class="k-mark" class:person={row.kind === 'Person'} aria-hidden="true">{initials(row.name)}</span><span
						class="k-who"><strong>{row.name}</strong><small>{row.kind}{row.note ? ` · ${row.note}` : ''}</small></span
					><span class="k-amount"><strong><Money amountKobo={row.owed} /></strong></span></a
				>{:else}<p class="none">No customer matches that.</p>{/each}
		</div>
	{/if}
</main>

<style>
	.field {
		display: grid;
		gap: 0.4rem;
		max-width: 22rem;
		margin-bottom: 1rem;
		font-weight: 600;
		font-size: 0.9rem;
	}
	select,
	input {
		font: inherit;
		font-weight: 400;
		padding: 0.75rem 0.9rem;
		border: 1px solid var(--color-border-strong);
		background: var(--color-surface);
	}
	.tools {
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: 1rem;
		flex-wrap: wrap;
		margin-bottom: 1rem;
	}
	.tools .search {
		flex: 1 1 18rem;
		margin: 0;
	}
	.filters {
		display: flex;
		border: 1px solid var(--color-border-strong);
	}
	.filters button {
		min-height: 3rem;
		padding: 0 1rem;
		border: 0;
		border-left: 1px solid var(--color-border-strong);
		background: var(--color-surface);
		color: var(--color-muted);
		font-weight: 600;
	}
	.filters button:first-child {
		border-left: 0;
	}
	.filters button.on {
		background: var(--kredit-ink);
		color: var(--kredit-cream);
	}
	.loading,
	.none {
		padding: 1.5rem 1.25rem;
		color: var(--color-muted);
	}
</style>
