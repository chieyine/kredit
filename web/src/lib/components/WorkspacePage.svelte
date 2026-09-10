<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import Money from '$lib/components/Money.svelte';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import { checkedJSON, publicError, RequestError, record as objectRecord, text } from '$lib/api/reliable';
	import type { KoboValue } from '$lib/money';

	/**
	 * A list of records, where the caller says what each record means.
	 *
	 * The previous version guessed: it tried six field names for a title and
	 * printed the record's UUID when none matched, which is what the audit trail
	 * and half the workspace lists ended up showing. Guessing is what made every
	 * list look the same and read like nothing. Each page now names its own
	 * title, status, amount and link, and a record it cannot describe is not
	 * rendered as a row of identifiers.
	 */
	type Row = Record<string, any>;
	let {
		eyebrow,
		title,
		description,
		primaryLabel = '',
		primaryHref = '',
		endpoint = '',
		organizationPath = '',
		collectionKey = '',
		emptyTitle = 'Nothing here yet',
		emptyCopy = 'New items appear here.',
		rowTitle,
		rowStatus = () => '',
		rowDetail = () => '',
		rowAmount = () => null,
		rowAmountLabel = '',
		rowHref = () => '',
		searchPlaceholder = 'Search',
		keep = () => true
	}: {
		eyebrow: string;
		title: string;
		description: string;
		primaryLabel?: string;
		primaryHref?: string;
		endpoint?: string;
		organizationPath?: string;
		collectionKey?: string;
		emptyTitle?: string;
		emptyCopy?: string;
		rowTitle: (record: Row) => string;
		rowStatus?: (record: Row) => string;
		rowDetail?: (record: Row) => string;
		rowAmount?: (record: Row) => KoboValue;
		rowAmountLabel?: string;
		rowHref?: (record: Row, organizationID: string) => string;
		searchPlaceholder?: string;
		/**
		 * Which records belong on this page. Two pages read the same endpoint —
		 * sales a customer has not answered yet, and sales they are now paying
		 * off — and before this they showed the identical list under two names.
		 */
		keep?: (record: Row) => boolean;
	} = $props();

	let loading = $state(true), error = $state(''), records = $state<Row[]>([]);
	let organizations = $state<Row[]>([]), organizationID = $state(''), query = $state(''), pageNumber = $state(1);
	const pageSize = 20;

	// Search reads the words this page actually shows. Matching a stringified
	// record meant a search for "paid" hit any row whose internal id contained
	// those letters, and the result could not be explained to the person typing.
	const mine = $derived(records.filter((record) => keep(record)));
	const filtered = $derived.by(() => {
		const needle = query.trim().toLowerCase();
		if (!needle) return mine;
		return mine.filter((record) =>
			`${rowTitle(record)} ${rowDetail(record)} ${rowStatus(record)}`.toLowerCase().includes(needle)
		);
	});
	const visible = $derived(filtered.slice((pageNumber - 1) * pageSize, pageNumber * pageSize));
	const pages = $derived(Math.max(1, Math.ceil(filtered.length / pageSize)));

	let requestVersion = 0;
	let controller: AbortController | undefined;

	async function refresh(keepRecords = false) {
		const version = ++requestVersion;
		controller?.abort();
		controller = new AbortController();
		const signal = controller.signal;
		loading = true;
		error = '';
		if (!keepRecords) records = [];
		pageNumber = 1;
		async function read(url: string) {
			try { return await checkedJSON<any>(url, value => value, { signal }); }
			catch(cause) {
				if(cause instanceof RequestError && cause.status === 401) location.assign(`/app?next=${encodeURIComponent(page.url.pathname + page.url.search)}`);
				throw cause;
			}
		}
		try {
			if (organizationPath && !organizations.length) {
				const data = await read('/api/v1/organizations');
				if (version !== requestVersion) return;
				if (!Array.isArray(data.organizations)) throw new Error('unavailable');
				organizations = data.organizations.map((value: unknown) => {
                  const item = objectRecord(value);
                  text(item.id); text(item.legal_name);
                  return item;
                });
				const requested = new URLSearchParams(location.search).get('organization');
				organizationID = organizations.find((item) => item.id === requested)?.id ?? organizations[0]?.id ?? '';
			}
			if (organizationPath && !organizationID) return;
			const data = await read(organizationPath ? `/api/v1/organizations/${encodeURIComponent(organizationID)}${organizationPath}` : endpoint);
			if (version !== requestVersion) return;
			const value = collectionKey ? data[collectionKey] : data;
			if (!Array.isArray(value)) throw new Error('unavailable');
			records = value.map(objectRecord);
		} catch (cause) {
			if (version === requestVersion && !signal.aborted) {
				error = publicError(cause, title.toLowerCase());
			}
		} finally {
			if (version === requestVersion) loading = false;
		}
	}

	onMount(() => {
		if (endpoint || organizationPath) void refresh();
		else loading = false;
		return () => { requestVersion++; controller?.abort(); };
	});
</script>

<svelte:head><title>{title} — Kredit</title></svelte:head>

<main class="shell workspace">
	<header class="page-head">
		<div>
			<p class="eyebrow">{eyebrow}</p>
			<h1>{title}</h1>
			<p class="lede">{description}</p>
		</div>
		{#if primaryHref}<a class="primary" href={primaryHref}>{primaryLabel}</a>{/if}
	</header>

	{#if endpoint || organizationPath}
		<div class="toolbar">
			{#if organizations.length > 1}
				<label>Business
					<select bind:value={organizationID} onchange={() => refresh()}>
						{#each organizations as org}<option value={org.id}>{org.trading_name || org.legal_name}</option>{/each}
					</select>
				</label>
			{/if}
			<label class="search"><span>Find</span><input bind:value={query} oninput={() => (pageNumber = 1)} type="search" placeholder={searchPlaceholder} /></label>
			<button type="button" onclick={() => refresh(true)} disabled={loading}>{loading ? 'Checking…' : 'Check again'}</button>
		</div>
	{/if}

	{#if loading}
		<div role="status"><span class="sr-only">Opening {title}</span><Skeleton rows={5} /></div>
	{:else if error}
		<div class="error" role="alert"><p>{error}</p><button type="button" onclick={() => refresh()}>Try again</button></div>
	{:else if filtered.length}
		<p class="count">{filtered.length === mine.length ? `${mine.length} ${mine.length === 1 ? 'item' : 'items'}` : `${filtered.length} of ${mine.length}`}</p>
		<ul class="records">
			<!-- Several attempts or businesses can legitimately link to the same detail page. -->
			{#each visible as record}
				{@const href = rowHref(record, organizationID)}
				{@const amount = rowAmount(record)}
				<li>
					<svelte:element this={href ? 'a' : 'div'} href={href || undefined} class="record">
						<span class="who">
							<strong>{rowTitle(record)}</strong>
							{#if rowDetail(record)}<small>{rowDetail(record)}</small>{/if}
						</span>
						{#if amount !== null && amount !== undefined}
							<span class="amount"><Money amountKobo={amount} />{#if rowAmountLabel}<small>{rowAmountLabel}</small>{/if}</span>
						{/if}
						{#if rowStatus(record)}<StatusPill status={rowStatus(record)} />{/if}
					</svelte:element>
				</li>
			{/each}
		</ul>
		{#if pages > 1}
			<nav class="pagination" aria-label="Pages">
				<button type="button" disabled={pageNumber === 1} onclick={() => pageNumber--}>Back</button>
				<span>Page {pageNumber} of {pages}</span>
				<button type="button" disabled={pageNumber >= pages} onclick={() => pageNumber++}>Next</button>
			</nav>
		{/if}
	{:else}
		<section class="empty">
			<h2>{query ? 'Nothing matches that' : emptyTitle}</h2>
			<p>{query ? 'Try a different name or word.' : emptyCopy}</p>
			{#if primaryHref && !query}<a class="primary" href={primaryHref}>{primaryLabel}</a>{/if}
		</section>
	{/if}
</main>

<style>
	.page-head { display: flex; align-items: end; justify-content: space-between; gap: 2rem; padding: 1.5rem 0; border-bottom: 1px solid var(--color-border); }
	.page-head h1 { margin: .3rem 0; font-size: 1.9rem; line-height: 1.2; }
	.lede { max-width: 60ch; margin: .4rem 0 0; color: var(--color-muted); line-height: 1.6; }
	.toolbar { display: flex; align-items: end; flex-wrap: wrap; gap: .75rem; margin: 1.5rem 0; }
	.toolbar label { display: grid; gap: .35rem; font-weight: 650; }
	.toolbar .search { flex: 1; min-width: min(100%, 15rem); }
	.toolbar input, .toolbar select { box-sizing: border-box; width: 100%; min-height: 3rem; padding: .7rem; border: 1px solid var(--color-border); background: var(--color-surface); color: inherit; font: inherit; }
	.toolbar button { min-height: 3rem; padding: .7rem 1rem; border: 1px solid var(--color-border); background: var(--color-surface); color: inherit; font: inherit; }
	.count { color: var(--color-muted); font-size: .9rem; }
	.records { display: grid; margin: .5rem 0 0; padding: 0; list-style: none; border-top: 1px solid var(--color-border); }
	.record { display: flex; align-items: center; justify-content: space-between; gap: 1.25rem; min-height: 4rem; padding: .9rem .25rem; border-bottom: 1px solid var(--color-border); color: inherit; text-decoration: none; }
	a.record:hover { background: var(--color-surface-muted); }
	.who { display: grid; gap: .25rem; min-width: 0; }
	.who strong { overflow-wrap: anywhere; }
	.who small, .amount small { color: var(--color-muted); }
	.amount { display: grid; gap: .2rem; text-align: right; white-space: nowrap; font-weight: 700; font-variant-numeric: tabular-nums; }
	.pagination { display: flex; align-items: center; justify-content: center; gap: 1rem; margin: 1.5rem 0; }
	.pagination button { min-height: 2.75rem; padding: .6rem 1rem; border: 1px solid var(--color-border); background: var(--color-surface); color: inherit; font: inherit; }
	.empty { margin-top: 1.5rem; padding: 2rem; border: 1px dashed var(--color-border); }
	.empty h2 { margin: 0 0 .4rem; font-size: 1.15rem; }
	.empty p { margin: 0 0 1rem; color: var(--color-muted); line-height: 1.6; }
	.error { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 1rem; margin: 1.5rem 0; padding: 1rem; border-left: 3px solid var(--color-destructive); background: #ffebe9; }
	.error p { margin: 0; line-height: 1.6; }
	.error button { min-height: 2.75rem; padding: .55rem .9rem; border: 1px solid currentColor; background: transparent; color: inherit; font: inherit; }
	@media (max-width: 640px) {
		.page-head { display: block; }
		.page-head .primary { margin-top: 1rem; }
		.toolbar label, .toolbar button { width: 100%; }
		.record { align-items: start; flex-direction: column; gap: .5rem; }
		.amount { text-align: left; }
	}
</style>
