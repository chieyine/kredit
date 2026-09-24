<script lang="ts">
	import { workspaceHref } from '$lib/workspace-navigation';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { checkedJSON, LatestRequest, publicError, record, RequestError, rows, text } from '$lib/api/reliable';
	let { children } = $props();
	type Business = { id: string; workspaceID: string; name: string };
	let businesses = $state<Business[]>([]),
		selected = $state(''),
		loading = $state(true),
		error = $state('');
	const reads = new LatestRequest();
	const current = $derived(businesses.find((item) => item.id === selected));
	async function load() {
		const request = reads.begin();
		loading = true;
		error = '';
		try {
			const result = await checkedJSON(
				'/api/v1/buyer/businesses',
				rows('businesses', (value) => {
					const item = record(value);
					return {
						id: text(item.id),
						workspaceID: typeof item.workspace_id === 'string' ? item.workspace_id : '',
						name: (typeof item.trading_name === 'string' ? item.trading_name : '') || text(item.legal_name)
					};
				}),
				{ signal: request.signal }
			);
			if (!request.current()) return;
			businesses = result;
			let businessID = page.url.searchParams.get('business_id');
			const workspaceID = page.url.searchParams.get('organization');
			// Notification links may enter directly without a workspace query. Resolve the
			// record's actual business before rendering a selector or any financial action.
			const detail = page.url.pathname.match(
				/^\/workspace\/purchases\/(orders|obligations|disputes|bank-authorization)\/([^/]+)$/
			);
			if (detail) {
				const id = encodeURIComponent(decodeURIComponent(detail[2]));
				let identity = '';
				if (detail[1] === 'bank-authorization') {
					identity = await checkedJSON(
						`/api/v1/buyer/bank-authorization/${encodeURIComponent(id)}`,
						(value) => text(record(record(value).mandate).business_id),
						{ signal: request.signal }
					);
				} else {
					let endpoint = detail[1] === 'orders' ? `credit-requests/${id}` : `obligations/${id}`;
					if (detail[1] === 'disputes') {
						const obligation = await checkedJSON(
							`/api/v1/buyer/disputes/${encodeURIComponent(id)}`,
							(value) => text(record(record(value).dispute).obligation_id),
							{ signal: request.signal }
						);
						endpoint = `obligations/${encodeURIComponent(obligation)}`;
					}
					identity = await checkedJSON(
						`/api/v1/buyer/${endpoint}`,
						(value) => {
							const data = record(value),
								view = detail[1] === 'orders' ? data : record(data.view);
							return text(record(view.request).buyer_business_id);
						},
						{ signal: request.signal }
					);
				}
				if (!request.current()) return;
				if (!identity) throw new Error('The business for this record could not be confirmed.');
				if (businessID && businessID !== identity)
					throw new Error('This purchase belongs to a different business. Open it from that business’s purchases.');
				businessID = identity;
			}
			const found = businessID
				? result.find((item) => item.id === businessID && (!workspaceID || item.workspaceID === workspaceID))
				: workspaceID
					? result.find((item) => item.workspaceID === workspaceID)
					: result[0];
			selected = found?.id ?? '';
			if ((businessID || workspaceID) && !found && result.length)
				error = 'This business is not available for purchases in your account. Choose one of your businesses below.';
			if (found) {
				const url = new URL(page.url);
				url.searchParams.set('business_id', found.id);
				if (found.workspaceID) url.searchParams.set('organization', found.workspaceID);
				if (url.search !== page.url.search)
					await goto(url.pathname + url.search + url.hash, { replaceState: true, noScroll: true });
			}
		} catch (cause) {
			if (!request.current()) return;
			// A record link that is wrong, expired or someone else's is not a
			// connection problem; say what actually happened.
			error =
				cause instanceof RequestError && [403, 404].includes(cause.status)
					? 'We could not find this in your purchases. Open it again from your list.'
					: publicError(cause, 'your purchasing businesses');
		} finally {
			if (request.current()) loading = false;
		}
	}
	function change() {
		const business = businesses.find((item) => item.id === selected);
		const destination = ['/workspace/purchases/permissions', '/workspace/purchases/access'].includes(page.url.pathname)
			? page.url.pathname
			: '/workspace/purchases';
		if (business)
			void goto(
				`${destination}?business_id=${encodeURIComponent(business.id)}${business.workspaceID ? '&organization=' + encodeURIComponent(business.workspaceID) : ''}`
			);
	}
	onMount(() => {
		void load();
		return () => reads.cancel();
	});
</script>

{#if loading}<div class="shell purchase-context" role="status">Opening your purchasing workspace…</div>
{:else}
	<div class="shell purchase-context">
		{#if businesses.length}<label
				>Purchasing business<select bind:value={selected} onchange={change}
					><option value="" disabled>Choose your business</option>{#each businesses as business (business.id)}<option
							value={business.id}>{business.name}</option
						>{/each}</select
				></label
			>{/if}
		{#if error}<p role="alert">{error}</p>
			<button onclick={load}>Try again</button>{/if}
		{#if !error && !current}<section>
				<h1>Connect with your first supplier.</h1>
				<p>
					Open the private invitation your manufacturer or distributor sent you. You can connect an existing business
					instead of creating it again.
				</p>
				<a href={workspaceHref('/workspace/today', page.url)}>Back to your workspace →</a>
				<p>Buying for yourself? <a href="/personal/purchases">Open personal purchases</a>.</p>
			</section>{/if}
	</div>
	{#if !error && current}{@render children()}{/if}
{/if}

<style>
	.purchase-context {
		max-width: 66rem;
		padding-block: 1rem;
	}
	.purchase-context label {
		display: grid;
		gap: 0.4rem;
		max-width: 28rem;
		font-weight: 650;
	}
	.purchase-context select {
		padding: 0.65rem;
		font: inherit;
		border: 1px solid var(--color-border);
		background: var(--color-surface);
		border-radius: 0.4rem;
	}
	.purchase-context section {
		padding: 2rem 0;
		max-width: 60ch;
	}
	.purchase-context p {
		line-height: 1.65;
	}
	.purchase-context a {
		color: var(--color-primary);
	}
	.purchase-context button {
		padding: 0.6rem 1rem;
		font: inherit;
	}
</style>
