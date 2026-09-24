<script lang="ts">
	import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
	import { chooseWorkspace, requestedWorkspace } from '$lib/workspace-context';
	import { onMount } from 'svelte';
	import { checkedJSON, LatestRequest, record, rows, text } from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';
	import { distributorTemplate, parseDistributorCSV, type DistributorRow } from '$lib/distributor-import';
	let businesses = $state<
			{
				id: string;
				name: string;
			}[]
		>([]),
		business = $state(''),
		loading = $state(true),
		error = $state('');
	let roster = $state<DistributorRow[]>([]),
		running = $state(false),
		stopped = $state(false),
		approved = $state(false);
	import { decodeImportBatch as decodeBatch, type ImportBatch as Batch } from '$lib/distributor-import-batches';
	let batches = $state<Batch[]>([]),
		batch = $state<Batch | null>(null),
		sourceHash = $state(''),
		saving = $state(false);
	const endpoint = () => `/api/v1/organizations/${encodeURIComponent(business)}/distributor-imports`;
	const batchReads = new LatestRequest();
	let batchLoading = $state(false),
		batchError = $state(''),
		organizationError = $state('');
	async function loadBatches() {
		const request = batchReads.begin();
		batches = [];
		batchError = '';
		if (!business) return;
		batchLoading = true;
		try {
			const items = await checkedJSON(endpoint(), rows('batches', decodeBatch), { signal: request.signal });
			if (request.current()) batches = items;
		} catch (cause) {
			if (request.current()) batchError = cause instanceof Error ? cause.message : 'Saved imports could not be loaded.';
		} finally {
			if (request.current()) batchLoading = false;
		}
	}
	function applyBatch(value: Batch) {
		if (!value.contacts) throw new Error('Import contacts are unavailable');
		batch = value;
		roster = value.contacts;
		sourceHash = value.source_hash;
		results = {};
		approved = value.state === 'approved' || value.state === 'completed';
		for (const n of value.completed_rows) {
			const row = roster[n - 1];
			if (row) results[row.source_reference] = { state: 'existing_invitation', link: '' };
		}
	}
	async function openBatch(id: string) {
		if (running || saving) return;
		saving = true;
		error = '';
		try {
			const value = await checkedJSON(`${endpoint()}/${encodeURIComponent(id)}`, decodeBatch);
			if (!Array.isArray(value.contacts) || value.contacts.length !== value.row_count)
				throw new Error('Import contacts are unavailable');
			applyBatch(value);
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'The import could not be opened.';
		} finally {
			saving = false;
		}
	}
	async function saveBatch() {
		if (saving || running || !roster.length || !business) return;
		saving = true;
		error = '';
		try {
			applyBatch(
				await new MutationIntent('save-import:' + business + ':' + sourceHash, endpoint()).run(
					{ source_hash: sourceHash, contacts: roster },
					decodeBatch
				)
			);
			await loadBatches();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'The roster could not be saved.';
		} finally {
			saving = false;
		}
	}
	async function cancelBatch() {
		if (!batch || running || saving) return;
		saving = true;
		error = '';
		try {
			batch = await new MutationIntent('cancel-import:' + batch.id, `${endpoint()}/${batch.id}`).run(
				{ action: 'cancel' },
				decodeBatch
			);
			await loadBatches();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Cancellation could not be confirmed.';
		} finally {
			saving = false;
		}
	}
	let results = $state<
		Record<
			string,
			{
				state: string;
				link: string;
			}
		>
	>({});
	const reads = new LatestRequest();
	// eslint-disable-next-line svelte/prefer-svelte-reactivity -- idempotency keys are never rendered
	const intents = new Map<string, MutationIntent>();
	async function load() {
		loading = true;
		organizationError = '';
		const request = reads.begin();
		try {
			const items = await checkedJSON(
				'/api/v1/organizations',
				rows('organizations', (value) => {
					const v = record(value);
					return {
						id: text(v.id),
						name: (typeof v.trading_name === 'string' ? v.trading_name : '') || text(v.legal_name)
					};
				}),
				{ signal: request.signal }
			);
			if (request.current()) {
				businesses = items;
				business = requestedWorkspace(items);
				await loadBatches();
			}
		} catch (cause) {
			if (request.current())
				organizationError = cause instanceof Error ? cause.message : 'Businesses could not be loaded.';
		} finally {
			if (request.current()) loading = false;
		}
	}
	onMount(() => {
		void load();
		return () => {
			reads.cancel();
			batchReads.cancel();
		};
	});
	function template() {
		const url = URL.createObjectURL(new Blob([distributorTemplate], { type: 'text/csv;charset=utf-8' }));
		const a = document.createElement('a');
		a.href = url;
		a.download = 'kredit-business-contacts.csv';
		a.click();
		setTimeout(() => URL.revokeObjectURL(url), 1000);
	}
	async function readFile(event: Event) {
		error = '';
		roster = [];
		results = {};
		approved = false;
		batch = null;
		sourceHash = '';
		const file = (event.target as HTMLInputElement).files?.[0];
		if (!file) return;
		saving = true;
		try {
			if (file.size > 1000000) throw new Error('Use a CSV file smaller than 1 MB.');
			const data = await file.arrayBuffer();
			sourceHash = Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256', data)))
				.map((v) => v.toString(16).padStart(2, '0'))
				.join('');
			roster = parseDistributorCSV(new TextDecoder().decode(data));
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'The file could not be read.';
		} finally {
			saving = false;
		}
	}
	async function send() {
		if (running || !approved || !business || !batch || !roster.length || batch.state === 'cancelled') return;
		running = true;
		stopped = false;
		error = '';
		const owner = business;
		const current = batch.id;
		try {
			batch = await new MutationIntent('approve-import:' + current, `${endpoint()}/${current}`).run(
				{ action: 'approve' },
				decodeBatch
			);
			for (let n = 0; n < roster.length; n++) {
				if (stopped) break;
				const row = roster[n];
				const key = row.source_reference;
				if (
					results[key]?.state === 'accepted' ||
					results[key]?.state === 'sent' ||
					results[key]?.state === 'existing_invitation' ||
					results[key]?.state === 'manual_handoff_required'
				)
					continue;
				results[key] = { state: 'processing', link: '' };
				const path = `/api/v1/organizations/${encodeURIComponent(owner)}/distributor-imports/${encodeURIComponent(current)}/rows/${n + 1}`;
				const scope = owner + ':' + current + ':' + key;
				if (!intents.has(scope)) intents.set(scope, new MutationIntent(scope, path));
				try {
					results[key] = await intents.get(scope)!.run({}, (value) => {
						const v = record(value);
						if (v.delivery_state === 'accepted') return { state: 'accepted', link: '' };
						const link = new URL(text(v.invitation_url), location.origin);
						if (
							!['https:', 'http:'].includes(link.protocol) ||
							link.username ||
							link.password ||
							!link.pathname.startsWith('/buyer-invitations/')
						)
							throw new Error('Invalid invitation link');
						const state = text(v.delivery_state);
						if (!['sent', 'existing_invitation', 'manual_handoff_required'].includes(state))
							throw new Error('Unconfirmed invitation delivery');
						return { state, link: link.href };
					});
				} catch (cause) {
					results[key] = { state: 'needs_review', link: '' };
					throw cause;
				}
			}
			batch = await checkedJSON(`${endpoint()}/${current}`, decodeBatch);
		} catch (cause) {
			error =
				cause instanceof Error
					? cause.message
					: 'The invitation result is uncertain. Reopen the saved batch before continuing.';
		} finally {
			running = false;
			await loadBatches();
		}
	}
	async function recoverLink(n: number) {
		if (!batch || running || saving) return;
		saving = true;
		error = '';
		try {
			const value = await new MutationIntent(
				'recover-import:' + batch.id + ':' + n,
				`${endpoint()}/${batch.id}/rows/${n}`
			).run({}, (value) => {
				const v = record(value);
				if (v.delivery_state === 'accepted') return { state: 'accepted', link: '' };
				const link = new URL(text(v.invitation_url), location.origin);
				if (
					!['https:', 'http:'].includes(link.protocol) ||
					link.username ||
					link.password ||
					!link.pathname.startsWith('/buyer-invitations/')
				)
					throw new Error('Invalid invitation link');
				return { state: 'existing_invitation', link: link.href };
			});
			results[roster[n - 1].source_reference] = value;
		} catch {
			error =
				'This link could not be recovered. The invitation may already be accepted or expired. Check Track invitations.';
		} finally {
			saving = false;
		}
	}
	const completed = $derived(
		Object.values(results).filter((v) =>
			['accepted', 'sent', 'existing_invitation', 'manual_handoff_required'].includes(v.state)
		).length
	);
	function label(state: string) {
		return (
			(
				{
					accepted: 'Already accepted',
					sent: 'Sent',
					existing_invitation: 'Existing invitation',
					manual_handoff_required: 'Share link manually',
					processing: 'Creating invitation',
					needs_review: 'Review before continuing'
				} as Record<string, string>
			)[state] ?? 'Ready'
		);
	}
</script>

<svelte:head><title>Invite your business network — Kredit</title></svelte:head>
<main class="shell workspace">
	<p class="eyebrow">Trading partners</p>
	<h1>Invite your business network</h1>
	<p class="lede">
		Import up to 200 distributors or retailers, review their details, then invite them to verify their businesses.
		Credit terms and bank permissions are agreed separately.
	</p>
	<a href={`/workspace/partners/customers?organization=${encodeURIComponent(business)}`}>View business customers</a> ·
	<a href={`/workspace/partners/invitations?organization=${encodeURIComponent(business)}`}>Track invitations</a>
	{#if error}<p role="alert" class="error">{error}</p>{/if}
	<VerifyIdentity />
	<section class="card">
		<h2>Prepare your roster</h2>
		<label
			>Your business<select
				bind:value={business}
				disabled={loading || running || saving}
				onchange={() => chooseWorkspace(business)}
				>{#each businesses as item (item.id)}<option value={item.id}>{item.name}</option>{/each}</select
			></label
		>
		{#if organizationError}<p role="alert">{organizationError}</p>
			<button onclick={load}>Retry businesses</button>{/if}
		{#if !loading && !organizationError && !businesses.length}<p>
				<a href="/workspace/today">Add your business first.</a>
			</p>{/if}
		<p>
			Use a stable customer reference from your records. Reimporting the same reference and details returns the existing
			invitation. Changed details require review.
		</p>
		<button type="button" onclick={template}>Download CSV template</button>
		<p>
			Business types: unregistered_business, registered_business, sole_proprietor, limited_company, partnership. Export
			Excel files as CSV before uploading.
		</p>
		<label
			>Business contacts CSV<input
				type="file"
				accept=".csv,text/csv"
				onchange={readFile}
				disabled={running || saving}
			/></label
		>
	</section>
	<section class="card">
		<h2>Saved imports</h2>
		<p>
			Return here to continue a saved roster. Invitations already created remain in your network when a batch is
			cancelled.
		</p>
		{#if batchLoading}<p role="status">Loading saved imports…</p>{:else if batchError}<p role="alert">{batchError}</p>
			<button onclick={loadBatches}>Retry saved imports</button
			>{:else if batches.length}{#each batches as item (item.id)}<p>
					<button disabled={running || saving} onclick={() => openBatch(item.id)}
						>Open {item.row_count}-contact import · {item.source_hash.slice(0, 8)}</button
					>
					{item.completed_rows.length} created · {item.state} · {new Date(item.created_at).toLocaleString()}
				</p>{/each}{:else}<p>No saved imports for this business.</p>{/if}
	</section>
	{#if roster.length}<section class="card">
			<h2>Review {roster.length} contact{roster.length === 1 ? '' : 's'}</h2>
			<p aria-live="polite">
				{completed} invitation{completed === 1 ? '' : 's'} created or recovered. Sending stops at the first uncertain result.
			</p>
			<div class="table-scroll">
				<table>
					<thead><tr><th>Reference</th><th>Business</th><th>Contact</th><th>Progress</th></tr></thead><tbody
						>{#each roster as row, n (n)}<tr
								><td>{row.source_reference.replace(/^roster:/, '')}</td><td
									>{row.legal_name}<small>{row.business_address}</small></td
								><td>{row.target}</td><td
									><span>{label(results[row.source_reference]?.state)}</span>{#if results[row.source_reference]?.link}<a
											href={results[row.source_reference].link}
											target="_blank"
											rel="noreferrer">Open private invitation</a
										>{:else if results[row.source_reference]?.state === 'existing_invitation' && batch?.state !== 'cancelled'}<button
											disabled={running || saving}
											onclick={() => recoverLink(n + 1)}>Recover private link</button
										>{/if}</td
								></tr
							>{/each}</tbody
					>
				</table>
			</div>
			{#if !batch}<button class="primary" onclick={saveBatch} disabled={saving || !business}
					>{saving ? 'Saving roster…' : 'Save this roster'}</button
				>
				<p>Save before inviting. Progress is kept with your business, so you can return later.</p>
			{:else}<p>
					Batch status: <strong>{batch.state}</strong>. Recovered invitations are not confirmation of message delivery.
					Use Track invitations to review the records.
				</p>{/if}
			<label class="approval"
				><input type="checkbox" bind:checked={approved} disabled={running || saving || batch?.state === 'cancelled'} />I
				reviewed these contacts and am authorized to invite them for this business.</label
			>
			<button
				class="primary"
				onclick={send}
				disabled={!approved ||
					!business ||
					!batch ||
					running ||
					saving ||
					batch.state === 'cancelled' ||
					completed === roster.length}>{running ? 'Creating invitations…' : 'Create and send invitations'}</button
			>
			{#if batch && batch.state !== 'cancelled' && batch.state !== 'completed'}<button
					onclick={cancelBatch}
					disabled={running || saving}>Cancel remaining invitations</button
				>{/if}
			{#if running}<button onclick={() => (stopped = true)} disabled={stopped}
					>{stopped ? 'Stopping after this invitation…' : 'Stop after this invitation'}</button
				>{/if}
		</section>{/if}
</main>

<style>
	.card {
		padding: 1.5rem;
		margin: 1.5rem 0;
	}
	.card label {
		display: grid;
		gap: 0.5rem;
		margin: 1rem 0;
	}
	.card select,
	.card input[type='file'] {
		padding: 0.75rem;
		max-width: 32rem;
	}
	.table-scroll {
		overflow: auto;
	}
	table {
		width: 100%;
		border-collapse: collapse;
	}
	th,
	td {
		text-align: left;
		padding: 0.85rem;
		border-bottom: 1px solid var(--color-border);
	}
	td small,
	td a {
		display: block;
		margin-top: 0.4rem;
	}
	.approval {
		display: flex !important;
		align-items: center;
	}
	button {
		margin: 0.4rem 0.6rem 0.4rem 0;
	}
	h1 {
		font-size: clamp(1.9rem, 3.4vw, 2.6rem);
		max-width: 20ch;
	}
</style>
