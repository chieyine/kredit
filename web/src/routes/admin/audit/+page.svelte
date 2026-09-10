<script lang="ts">
	import { onMount } from 'svelte';
	import { localTime } from '$lib/admin-client';
	import { checkedJSON, publicError, record, rows, text, LatestRequest } from '$lib/api/reliable';
	const requests = new LatestRequest();

	type AuditEvent = {
		id: string;
		occurred_at: string;
		actor_user_id?: string;
		organization_id?: string;
		action: string;
		resource_type: string;
		resource_id?: string;
		outcome: string;
		severity: string;
		request_id?: string;
		metadata?: Record<string, string>;
	};

	let events = $state<AuditEvent[]>([]);
	let organizationID = $state('');
	let query = $state('');
	let loading = $state(true);
	let error = $state('');

	// The action is a dotted path: "credit.request.sent", "auth.mfa.verified".
	// Read the last word as the thing that happened and the rest as where.
	function happened(action: string): string {
		const parts = action.split('.');
		const verb = parts.pop() ?? action;
		return verb.replaceAll('_', ' ').replace(/^./, (c) => c.toUpperCase());
	}
	function where(action: string): string {
		const parts = action.split('.');
		parts.pop();
		return parts.join(' · ').replaceAll('_', ' ');
	}

	const visible = $derived.by(() => {
		const needle = query.trim().toLowerCase();
		if (!needle) return events;
		return events.filter((event) =>
			[event.action, event.resource_type, event.resource_id, event.outcome, event.actor_user_id, event.request_id]
				.some((value) => String(value ?? '').toLowerCase().includes(needle))
		);
	});

	async function load() {
		const request = requests.begin();
		loading = true;
		error = '';
		try {
			const path = '/api/v1/ops/audit';
			const scope = organizationID.trim();
			const result = await checkedJSON(scope ? `${path}?organization_id=${encodeURIComponent(scope)}` : path, rows('events', value => {
				const item = record(value);
				for (const key of ['id', 'occurred_at', 'action', 'resource_type', 'outcome', 'severity']) text(item[key]);
				return item as AuditEvent;
			}), { signal: request.signal });
			if (request.current()) events = result;
		} catch (cause) {
			// An audit trail that renders an outage as an empty list is worse than
			// one that will not load: it reads as "nothing happened".
			if (request.current()) {
				events = [];
				error = publicError(cause, 'the audit trail');
			}
		} finally {
			if (request.current()) loading = false;
		}
	}

	onMount(() => { void load(); return () => requests.cancel(); });
</script>

<svelte:head><title>Audit trail — Kredit admin</title></svelte:head>

<main class="shell workspace audit">
	<header>
		<div>
			<p class="eyebrow">Admin / Audit</p>
			<h1>Audit trail</h1>
			<p class="lede">Who did what, to which record, and whether it worked. Entries can be added but never changed or removed.</p>
		</div>
	</header>

	<form class="filters" onsubmit={(event) => { event.preventDefault(); load(); }}>
		<label>Business <small>optional</small><input bind:value={organizationID} placeholder="Business reference" /></label>
		<label class="find">Find<input type="search" bind:value={query} placeholder="Action, record or reference" /></label>
		<button disabled={loading}>{loading ? 'Opening…' : 'Apply'}</button>
	</form>

	{#if error}
		<p class="error" role="alert">{error} <button type="button" onclick={load}>Try again</button></p>
	{:else if loading}
		<p role="status">Opening the audit trail…</p>
	{:else if !visible.length}
		<section class="empty">
			<h2>{events.length ? 'Nothing matches that' : 'No entries yet'}</h2>
			<p>{events.length ? 'Try a different action, record or reference.' : 'Account and money actions appear here as they happen.'}</p>
		</section>
	{:else}
		<p class="count" aria-live="polite">{visible.length} entr{visible.length === 1 ? 'y' : 'ies'}</p>
		<div class="table-wrap">
			<table>
				<caption class="sr-only">Audit entries, newest first</caption>
				<thead>
					<tr><th scope="col">When</th><th scope="col">What happened</th><th scope="col">Record</th><th scope="col">Result</th></tr>
				</thead>
				<tbody>
					{#each visible as event (event.id)}
						<tr class:failed={event.outcome !== 'success'} class:warned={event.severity === 'warning'}>
							<td><time datetime={event.occurred_at}>{localTime(event.occurred_at)}</time></td>
							<td>
								<strong>{happened(event.action)}</strong>
								{#if where(event.action)}<small>{where(event.action)}</small>{/if}
							</td>
							<td>
								<span>{event.resource_type.replaceAll('_', ' ')}</span>
								{#if event.resource_id}<details><summary>Reference</summary><code>{event.resource_id}</code></details>{/if}
							</td>
							<td>
								<span class="outcome">{event.outcome === 'success' ? 'Done' : event.outcome === 'denied' ? 'Refused' : event.outcome.replaceAll('_', ' ')}</span>
								{#if event.actor_user_id || event.request_id}
									<details><summary>Who and when</summary>
										{#if event.actor_user_id}<p>Person <code>{event.actor_user_id}</code></p>{/if}
										{#if event.request_id}<p>Request <code>{event.request_id}</code></p>{/if}
									</details>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</main>

<style>
	.audit > header { padding: 2rem 0 1.5rem; border-bottom: 1px solid var(--color-border); }
	.audit h1 { margin: .35rem 0; font-size: 1.9rem; line-height: 1.2; }
	.lede { max-width: 60ch; color: var(--color-muted); line-height: 1.6; }
	.filters { display: flex; align-items: end; flex-wrap: wrap; gap: 1rem; margin: 1.5rem 0; }
	.filters label { display: grid; gap: .35rem; font-weight: 650; }
	.filters .find { flex: 1; min-width: min(100%, 16rem); }
	.filters input { box-sizing: border-box; width: 100%; min-height: 3rem; padding: .7rem; border: 1px solid var(--color-border); background: var(--color-surface); font: inherit; }
	.filters button { min-height: 3rem; padding: .7rem 1.1rem; border: 1px solid var(--color-primary); background: var(--color-primary); color: #fff; font: inherit; font-weight: 700; }
	.count { color: var(--color-muted); }
	.table-wrap { overflow-x: auto; }
	table { width: 100%; border-collapse: collapse; }
	th, td { padding: .8rem .75rem; border-bottom: 1px solid var(--color-border); text-align: left; vertical-align: top; }
	th { background: var(--color-surface-muted); font-size: .8rem; text-transform: uppercase; letter-spacing: .05em; }
	td time { white-space: nowrap; font-variant-numeric: tabular-nums; }
	td small { display: block; color: var(--color-muted); }
	.outcome { font-weight: 700; }
	tr.failed .outcome { color: var(--color-destructive); }
	tr.warned .outcome { color: var(--color-warning); }
	details { margin-top: .35rem; }
	summary { min-height: 2.25rem; display: flex; align-items: center; color: var(--color-primary); cursor: pointer; font-size: .88rem; }
	code { overflow-wrap: anywhere; font-size: .82rem; }
	.empty { padding: 2rem; border: 1px dashed var(--color-border); }
	.empty h2 { margin: 0 0 .4rem; font-size: 1.15rem; }
	.error { padding: 1rem; border-left: 3px solid var(--color-destructive); background: #ffebe9; line-height: 1.6; }
	.error button { margin-left: .5rem; padding: .45rem .75rem; border: 1px solid currentColor; background: transparent; color: inherit; font: inherit; }
	@media (max-width: 640px) { .filters label { width: 100%; } .filters button { width: 100%; } }
</style>
