<script lang="ts">
	import { onMount } from 'svelte';
	import { localTime } from '$lib/admin-client';
	import { formatKobo, exactKobo, type KoboValue } from '$lib/money';
	import { checkedJSON, optionalText, record, rows, text, publicError, LatestRequest } from '$lib/api/reliable';
	const requests = new LatestRequest();
	type ScheduleEntry = { id: string; due_at: string; principal_due_kobo: KoboValue; allocated_kobo: KoboValue };
	type DateEntry = { item_id: string; due_at: string };
	type HistoryEvent = { occurred_at: string; actor_name: string; action: string; reason: string };
	type HistoryItem = {
		id: string;
		kind: string;
		state: string;
		created_at: string;
		proposer: string;
		approver: string;
		reason: string;
		before_values: Record<string, unknown>;
		after_values: Record<string, unknown>;
		/** Present only for schedule amendments: the original days and the proposed ones. */
		amendment: { items: ScheduleEntry[]; dates: DateEntry[] } | null;
		events: HistoryEvent[];
	};
	const amount = (value: unknown): KoboValue =>
		typeof value === 'number' || typeof value === 'string' || typeof value === 'bigint' ? value : null;
	function historyItem(value: unknown): HistoryItem {
		const item = record(value);
		const kind = text(item.kind);
		const before = record(item.before_values),
			after = record(item.after_values);
		return {
			id: text(item.id),
			kind,
			state: text(item.state),
			created_at: text(item.created_at),
			proposer: text(item.proposer),
			approver: optionalText(item.approver),
			reason: text(item.reason),
			before_values: before,
			after_values: after,
			amendment:
				kind === 'schedule_amendment'
					? {
							items: rows('items', (value): ScheduleEntry => {
								const entry = record(value);
								return {
									id: text(entry.id),
									due_at: text(entry.due_at),
									principal_due_kobo: amount(entry.principal_due_kobo),
									allocated_kobo: amount(entry.allocated_kobo)
								};
							})(before),
							dates: rows('dates', (value): DateEntry => {
								const entry = record(value);
								return { item_id: text(entry.item_id), due_at: text(entry.due_at) };
							})(after)
						}
					: null,
			events: rows('events', (value): HistoryEvent => {
				const event = record(value);
				return {
					occurred_at: text(event.occurred_at),
					actor_name: text(event.actor_name),
					action: text(event.action),
					reason: text(event.reason)
				};
			})(item)
		};
	}
	function difference(a: unknown, b: unknown) {
		const x = exactKobo(amount(a)),
			y = exactKobo(amount(b));
		return formatKobo(x === null || y === null ? null : x - y);
	}
	let items: HistoryItem[] = $state([]),
		q = $state(''),
		kind = $state(''),
		offset = $state(0),
		more = $state(false),
		busy = $state(false),
		error = $state('');
	let params = $derived(`q=${encodeURIComponent(q)}&kind=${encodeURIComponent(kind)}&offset=${offset}`);
	async function load() {
		const request = requests.begin();
		busy = true;
		error = '';
		items = [];
		more = false;
		try {
			const result = await checkedJSON(`/api/v1/ops/change-history?${params}`, rows('items', historyItem), {
				signal: request.signal
			});
			if (request.current()) {
				items = result.slice(0, 100);
				more = result.length > 100;
			}
		} catch (cause) {
			if (request.current()) error = publicError(cause, 'change history');
		} finally {
			if (request.current()) busy = false;
		}
	}
	function value(key: string, v: unknown): string {
		if (v === undefined || v === null) return '—';
		if (key.endsWith('_kobo')) return formatKobo(amount(v));
		if (key.endsWith('_bps')) return `${Number(v) / 100}%`;
		if (typeof v === 'object') return JSON.stringify(v, null, 2);
		return String(v);
	}
	onMount(() => {
		const p = new URLSearchParams(location.search);
		q = p.get('q') || '';
		kind = p.get('kind') || '';
		void load();
		return () => requests.cancel();
	});
</script>

<svelte:head><title>Change history — Kredit admin</title></svelte:head>
<main class="shell workspace">
	<h1>Change history</h1>
	<p>
		Search all retained policy changes, financial proposals and review assignments. Each record includes the original
		values, proposed changes, reasons and decision-makers.
	</p>
	<form
		onsubmit={(e) => {
			e.preventDefault();
			offset = 0;
			load();
		}}
	>
		<label>Search reason, person or reference<input type="search" bind:value={q} /></label><label
			>Change type<select bind:value={kind}
				><option value="">All permitted changes</option><option value="policy">Business policies</option><option
					value="write_off">Write-offs</option
				><option value="fee_waiver">Fee waivers</option><option value="schedule_amendment">Repayment dates</option
				><option value="assignment">Review assignments</option></select
			></label
		><button disabled={busy}>Search</button>
	</form>
	<a href={`/api/v1/ops/change-history?${params}&format=csv`}>Export this page as CSV</a>{#if error}<p role="alert">
			{error}
		</p>{/if}
	{#each items as item (item.id)}<article>
			<h2>{item.kind.replaceAll('_', ' ')} · {item.state}</h2>
			<p>
				{localTime(item.created_at)} · {item.proposer}{#if item.approver}
					· Decision by {item.approver}{/if}
			</p>
			<p>{item.reason}</p>
			<small>Reference: {item.id}</small>{#if item.kind === 'write_off'}<p>
					Proposed outstanding principal: {difference(
						item.before_values.outstanding_kobo,
						item.after_values.amount_kobo
					)}
				</p>{/if}
			<details>
				<summary>Previous and proposed values</summary>{#if item.amendment}<table>
						<thead><tr><th>Unpaid amount</th><th>Previous date</th><th>Proposed date</th></tr></thead><tbody
							>{#each item.amendment.dates as date, dateIndex (dateIndex)}{@const old = item.amendment.items.find(
									(i) => i.id === date.item_id
								)}<tr
									><td>{difference(old?.principal_due_kobo, old?.allocated_kobo)}</td><td
										>{localTime(old?.due_at || '')}</td
									><td>{localTime(date.due_at)}</td></tr
								>{/each}</tbody
						>
					</table>{:else}<table>
						<thead><tr><th>Field</th><th>Previous</th><th>Proposed</th></tr></thead><tbody
							>{#each [...new Set( [...Object.keys(item.before_values), ...Object.keys(item.after_values)] )] as key (key)}<tr
									><th>{key.replaceAll('_', ' ')}</th><td><pre>{value(key, item.before_values[key])}</pre></td><td
										><pre>{value(key, item.after_values[key])}</pre></td
									></tr
								>{/each}</tbody
						>
					</table>{/if}
			</details>
			{#each item.events as event, i (i)}<p>
					{localTime(event.occurred_at)} · {event.actor_name} · {event.action}: {event.reason}
				</p>{/each}
		</article>{:else}<p>
			{busy ? 'Loading history…' : error ? 'History unavailable.' : 'No matching changes.'}
		</p>{/each}<button
		disabled={busy || !offset}
		onclick={() => {
			offset = Math.max(0, offset - 100);
			load();
		}}>Previous</button
	><button
		disabled={busy || !more}
		onclick={() => {
			offset += 100;
			load();
		}}>Next</button
	>
</main>

<style>
	label {
		display: block;
		margin: 0.7rem 0;
	}
	input,
	select {
		display: block;
		padding: 0.6rem;
		width: 30rem;
		max-width: 100%;
	}
	button {
		padding: 0.7rem;
		margin: 0.5rem 0.5rem 0.5rem 0;
	}
	article {
		border: 1px solid var(--color-border);
		background: var(--color-surface);
		margin: 1.2rem 0;
		padding: 1.2rem;
		overflow: auto;
	}
	table {
		width: 100%;
	}
	th,
	td {
		text-align: left;
		vertical-align: top;
		padding: 0.6rem;
		border-bottom: 1px solid var(--color-border);
	}
	pre {
		white-space: pre-wrap;
		overflow-wrap: anywhere;
		max-width: 30rem;
	}
	small,
	p {
		overflow-wrap: anywhere;
	}
</style>
