<script lang="ts">
	import { onMount } from 'svelte';
	import { checkedJSON, LatestRequest, publicError, rows } from '$lib/api/reliable';
	import { requestedWorkspace, chooseWorkspace } from '$lib/workspace-context';
	import { organization, customer, type Customer, type Organization } from '$lib/records';
	import { tradeLine, type TradeLine } from '$lib/trade-line-records';
	import { productLabel } from '$lib/product-language';
	import Money from '$lib/components/Money.svelte';
	let businesses = $state<Organization[]>([]),
		customers = $state<Customer[]>([]),
		lines = $state<TradeLine[]>([]);
	let organizationID = $state(''),
		loading = $state(true),
		error = $state('');
	const reads = new LatestRequest();
	async function load() {
		const request = reads.begin();
		loading = true;
		error = '';
		lines = [];
		try {
			businesses = await checkedJSON('/api/v1/organizations', rows('organizations', organization), {
				signal: request.signal
			});
			if (!request.current()) return;
			organizationID = requestedWorkspace(businesses);
			if (!organizationID) return;
			const base = `/api/v1/organizations/${encodeURIComponent(organizationID)}`;
			const [records, names] = await Promise.all([
				checkedJSON(`${base}/trade-lines`, rows('trade_lines', tradeLine), { signal: request.signal }),
				checkedJSON(`${base}/customers`, rows('customers', customer), { signal: request.signal })
			]);
			if (request.current()) {
				lines = records;
				customers = names;
			}
		} catch (cause) {
			if (request.current()) error = publicError(cause, 'existing customer limits');
		} finally {
			if (request.current()) loading = false;
		}
	}
	const name = (line: TradeLine) => {
		const item = customers.find((c) => c.buyer_business_id === line.buyer_business_id);
		return item?.trading_name || item?.legal_name || 'Business customer';
	};
	onMount(() => {
		void load();
		return () => reads.cancel();
	});
</script>

<svelte:head><title>Existing customer limits — Kredit</title></svelte:head>
<main class="shell workspace">
	<h1>Existing customer limits</h1>
	<p>Open an earlier agreement to view its sales or finish a delivery.</p>
	{#if businesses.length > 1}<label
			>Business<select bind:value={organizationID} onchange={() => chooseWorkspace(organizationID)}
				>{#each businesses as business (business.id)}<option value={business.id}
						>{business.trading_name || business.legal_name}</option
					>{/each}</select
			></label
		>{/if}
	{#if loading}<p role="status">Opening agreements…</p>{:else if error}<p role="alert">{error}</p>
		<button onclick={load}>Try again</button>
	{:else if !lines.length}<p>No existing customer limits.</p>
	{:else}<div class="record-list">
			{#each lines as line (line.id)}
				<a
					class="record-row"
					href={`/workspace/sales/limits/${encodeURIComponent(line.id)}?organization=${encodeURIComponent(organizationID)}`}
				>
					<span><strong>{name(line)}</strong><small>{productLabel(line.state)}</small></span>
					<span>Outstanding <Money amountKobo={line.current_exposure_kobo} /></span>
				</a>
			{/each}
		</div>{/if}
	<a href={`/workspace/give?organization=${encodeURIComponent(organizationID)}`}>Give goods on credit →</a>
</main>
