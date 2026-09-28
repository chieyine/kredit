<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { chooseWorkspace, requestedWorkspace } from '$lib/workspace-context';
	import { read, readAllPurchases, purchase, Mutation, money, label, type Purchase } from '$lib/consumer';
	import { record } from '$lib/api/reliable';
	import { organization, type Organization } from '$lib/records';
	let { mode = 'buyer' }: { mode?: 'buyer' | 'seller' | 'admin' } = $props();
	let items = $state<Purchase[]>([]),
		businesses = $state<Organization[]>([]),
		org = $state(''),
		error = $state(''),
		message = $state(''),
		busy = $state(false),
		loaded = $state(false),
		search = $state('');
	let enabled = $state(true),
		number = $state(''),
		evidence = $state('');
	let cursor = $state('');
	const settingsMutation = new Mutation();
	const customerContact = $derived(mode === 'seller' ? (page.url.searchParams.get('customer') ?? '') : '');
	const matching = $derived(
		items.filter(
			(i) =>
				(!customerContact || i.target === customerContact) &&
				`${i.terms.item} ${i.customer_name} ${i.target} ${i.id}`.toLowerCase().includes(search.toLowerCase())
		)
	);
	const endpoint = $derived(
		mode === 'buyer'
			? '/api/v1/buyer/purchases'
			: mode === 'admin'
				? '/api/v1/ops/consumer-sales'
				: `/api/v1/organizations/${encodeURIComponent(org)}/consumer-sales`
	);
	function href(s: Purchase) {
		return mode === 'buyer'
			? `/personal/purchases/${s.id}`
			: mode === 'admin'
				? `/admin/consumer-sales/${s.id}`
				: `/workspace/sales/consumers/${s.id}?organization=${s.organization_id}`;
	}
	async function load() {
		busy = true;
		error = '';
		loaded = false;
		items = [];
		try {
			if (mode === 'seller' && !org) return;
			if (customerContact) {
				items = await readAllPurchases(endpoint);
				cursor = '';
				loaded = true;
				return;
			}
			const data = record(await read(endpoint));
			if (!Array.isArray(data.items)) throw new Error('The purchase list could not be verified.');
			items = data.items.map(purchase);
			cursor = typeof data.next_cursor === 'string' ? data.next_cursor : '';
			loaded = true;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Could not load purchases.';
		} finally {
			busy = false;
		}
	}
	async function more() {
		busy = true;
		error = '';
		try {
			const data = record(await read(endpoint + '?before=' + encodeURIComponent(cursor)));
			if (!Array.isArray(data.items)) throw new Error('Could not verify the next page.');
			items = [...items, ...data.items.map(purchase)];
			cursor = typeof data.next_cursor === 'string' ? data.next_cursor : '';
		} catch (e) {
			error = e instanceof Error ? e.message : 'Could not load more purchases.';
		} finally {
			busy = false;
		}
	}
	async function retry() {
		busy = true;
		error = '';
		try {
			await settingsMutation.retry();
			message = 'Settings confirmed.';
			await load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Could not confirm previous action.';
		} finally {
			busy = false;
		}
	}
	async function init() {
		try {
			if (mode === 'seller') {
				const data = record(await read('/api/v1/organizations'));
				if (!Array.isArray(data.organizations)) throw new Error('Could not load businesses.');
				businesses = data.organizations.map(organization);
				org = requestedWorkspace(businesses);
			}
			await load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Could not load businesses.';
		}
	}
	async function connectBank() {
		busy = true;
		error = '';
		try {
			await settingsMutation.send(
				`/api/v1/organizations/${encodeURIComponent(org)}/consumer-bank`,
				{ account_number: number },
				(value) => {
					const result = record(value);
					if (result.saved !== true) throw new Error('Receiving account confirmation was incomplete.');
					return result;
				}
			);
			message = 'Receiving account matched. Consumer sales are available when your business checks are complete.';
			number = '';
		} catch (e) {
			error = e instanceof Error ? e.message : 'Could not confirm the receiving account.';
		} finally {
			busy = false;
		}
	}
	async function saveSettings() {
		busy = true;
		error = '';
		try {
			await settingsMutation.send(
				`/api/v1/ops/organizations/${encodeURIComponent(org)}/consumer-settings`,
				{ enabled, review_evidence: evidence },
				(value) => {
					const result = record(value);
					if (typeof result.enabled !== 'boolean' || typeof result.review_evidence !== 'string')
						throw new Error('Settings confirmation was incomplete.');
					return result;
				}
			);
			message = 'Restriction updated. Eligible retailers can use consumer sales automatically.';
		} catch (e) {
			error = e instanceof Error ? e.message : 'Settings were not saved.';
		} finally {
			busy = false;
		}
	}
	async function loadSettings() {
		busy = true;
		error = '';
		try {
			const s = record(await read(`/api/v1/ops/organizations/${encodeURIComponent(org)}/consumer-settings`));
			enabled = s.enabled === true;
			evidence = String(s.review_evidence ?? '');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Settings unavailable.';
		} finally {
			busy = false;
		}
	}
	onMount(init);
</script>

<main class="shell workspace feature-page consumer-sales">
	<header class="feature-heading">
		<div>
			<p class="eyebrow">
				{mode === 'seller'
					? 'Retailer → Consumer'
					: mode === 'admin'
						? 'Super admin · Retailer → Consumer'
						: 'Personal purchases'}
			</p>
			<h1>
				{mode === 'seller'
					? customerContact
						? 'Customer purchases'
						: 'Consumer sales'
					: mode === 'admin'
						? 'Consumer purchases'
						: 'Your purchases'}
			</h1>
			<p class="lede">
				{mode === 'seller'
					? 'Sell goods to individuals for personal use, on installments or layaway. Customer payments go to your receiving account.'
					: mode === 'admin'
						? 'Review purchases, resolve exceptions and support your retailers.'
						: 'See what you have paid, what is due and when your goods will arrive.'}
			</p>
		</div>
		<a
			class="secondary"
			href={mode === 'seller' && !customerContact
				? `/workspace/give?organization=${encodeURIComponent(org)}&kind=person`
				: '#purchase-list'}>{mode === 'seller' && !customerContact ? 'Create an offer →' : 'View purchases →'}</a
		>
	</header>
	{#if error}<p class="error" role="alert">{error}</p>
		{#if settingsMutation.pending}<button onclick={retry} disabled={busy}>Retry the same unconfirmed action</button
			>{/if}{/if}{#if message}<p role="status" class="message">{message}</p>{/if}
	{#if mode === 'seller' && !customerContact}<label
			>Retailer business<select bind:value={org} onchange={() => chooseWorkspace(org)} disabled={busy}
				>{#each businesses as b (b.id)}<option value={b.id}>{b.trading_name || b.legal_name}</option>{/each}</select
			></label
		>
		<details>
			<summary>Connect your receiving account<span>One setup step for direct customer payments</span></summary>
			<p>
				Enter the receiving account already connected in your <a
					href={`/workspace/settings/settlement?organization=${encodeURIComponent(org)}`}>settlement settings</a
				>. Kredit matches it automatically. You do not need a separate consumer-sale approval.
			</p>
			<form
				onsubmit={(e) => {
					e.preventDefault();
					void connectBank();
				}}
			>
				<label
					>Receiving account number<input
						bind:value={number}
						pattern="[0-9]{10}"
						maxlength="10"
						required
						disabled={busy}
					/></label
				><button disabled={busy || !org}>Use this account</button>
			</form>
		</details>
	{/if}
	{#if mode === 'admin'}<details>
			<summary>Restrict or restore a retailer when needed</summary>
			<p>
				Eligible retailers activate automatically. Use this control only for an exception, such as suspected fraud or an
				unresolved operating issue. Removing a restriction does not bypass business or bank checks.
			</p>
			<a href="/admin/organizations">Browse businesses and their IDs</a><label
				>Retailer business ID<input bind:value={org} disabled={busy} /></label
			><button onclick={loadSettings} disabled={busy || !org}>Load restriction</button>
			<form
				onsubmit={(e) => {
					e.preventDefault();
					void saveSettings();
				}}
			>
				<fieldset disabled={busy || !org}>
					<label>Reason<textarea bind:value={evidence} minlength="20" maxlength="2000" required></textarea></label
					><label><input type="checkbox" bind:checked={enabled} />No consumer-sale restriction</label><button
						>Save exception decision</button
					>
				</fieldset>
			</form>
		</details>{/if}
	<section id="purchase-list">
		<p class="eyebrow">Payment & delivery progress</p>
		<h2>Purchases</h2>
		<label>Find a purchase<input bind:value={search} type="search" /></label><button onclick={load} disabled={busy}
			>Refresh list</button
		>{#if busy}<p>Loading…</p>{:else if !loaded}<p>
				Choose a business or retry loading the list.
			</p>{:else}{#each matching as sale, i (i)}<a class="sale record-row" href={href(sale)}
					><strong>{sale.terms.item}</strong><span
						>{sale.customer_name || sale.target} · {label(sale.state)}{sale.case_state
							? ` · ${label(sale.case_state)}`
							: ''}</span
					><span>{money(sale.outstanding_kobo)} left to pay · {money(sale.refund_due_kobo)} refund due</span></a
				>{:else}<div class="empty-state">
					<h3>No matching purchases yet.</h3>
					<p>
						{mode === 'seller'
							? 'Create an offer to start tracking a customer’s payments and delivery.'
							: search
								? 'Try another name or item.'
								: 'Received a purchase link from your retailer? Open that private link and sign in with the contact they invited. Your purchases will then appear here.'}
					</p>
				</div>{/each}{#if cursor}<button onclick={more} disabled={busy}>Load older purchases</button>{/if}{/if}
	</section>
</main>
