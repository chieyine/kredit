<script lang="ts">
	import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
	import { onMount } from 'svelte';
	import { adminGet, localTime, localInput, lagosISO } from '$lib/admin-client';
	import { formatKobo, parseNaira, exactKobo, type KoboValue } from '$lib/money';
	import { LatestRequest, record, rows, text } from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';
	const reads = new LatestRequest();
	// eslint-disable-next-line svelte/prefer-svelte-reactivity -- idempotency keys are never rendered
	const intents = new Map<string, MutationIntent>();
	function write(url: string, payload: unknown, decode: (value: unknown) => unknown) {
		let intent = intents.get(url);
		if (!intent) {
			intent = new MutationIntent('admin-financial-changes', url);
			intents.set(url, intent);
		}
		return intent.run(payload, decode);
	}
	type ScheduleItem = {
		id: string;
		due_at: string;
		principal_due_kobo: KoboValue;
		allocated_kobo: KoboValue;
		cancelled: boolean;
	};
	type Snapshot = { outstanding_kobo: KoboValue; items: ScheduleItem[] };
	type ProposedDate = { item_id: string; due_at: string };
	type Change = {
		id: string;
		kind: string;
		state: string;
		reason: string;
		proposed_by: string;
		proposer: string;
		approver: string;
		expires_at: string;
		obligation_id: string;
		before_values: Snapshot;
		proposed_values: { amount_kobo: KoboValue; dates: ProposedDate[] };
	};
	type ChangeContext = { obligation_id: string; snapshot: Snapshot };
	function exactAmount(value: unknown, message: string): KoboValue {
		if (typeof value !== 'number' && typeof value !== 'string' && typeof value !== 'bigint') throw new Error(message);
		if (exactKobo(value) === null) throw new Error(message);
		return value;
	}
	function snapshot(value: unknown, requireItems = false): Snapshot {
		const result = record(value);
		const outstanding = exactAmount(result.outstanding_kobo, 'Outstanding amount was incomplete');
		const items =
			requireItems || result.items !== undefined
				? rows('items', (value): ScheduleItem => {
						const item = record(value);
						return {
							id: text(item.id),
							due_at: text(item.due_at),
							principal_due_kobo: exactAmount(item.principal_due_kobo, 'Schedule amounts were incomplete'),
							allocated_kobo: exactAmount(item.allocated_kobo, 'Schedule amounts were incomplete'),
							cancelled: item.cancelled === true
						};
					})(result)
				: [];
		return { outstanding_kobo: outstanding, items };
	}
	function change(value: unknown): Change {
		const item = record(value);
		const proposed = record(item.proposed_values);
		const kind = text(item.kind);
		const dates =
			kind === 'schedule_amendment'
				? rows('dates', (value): ProposedDate => {
						const date = record(value);
						return { item_id: text(date.item_id), due_at: text(date.due_at) };
					})(proposed)
				: [];
		return {
			id: text(item.id),
			kind,
			state: text(item.state),
			reason: text(item.reason),
			proposed_by: text(item.proposed_by),
			proposer: text(item.proposer),
			approver: typeof item.approver === 'string' ? item.approver : '',
			expires_at: text(item.expires_at),
			obligation_id: text(item.obligation_id),
			before_values: snapshot(item.before_values, kind === 'schedule_amendment'),
			proposed_values: {
				amount_kobo:
					kind === 'schedule_amendment' ? 0 : exactAmount(proposed.amount_kobo, 'Proposed amount was incomplete'),
				dates
			}
		};
	}
	function difference(a: KoboValue, b: KoboValue) {
		const x = exactKobo(a),
			y = exactKobo(b);
		return formatKobo(x === null || y === null ? null : x - y);
	}
	function unpaid(item: ScheduleItem) {
		const due = exactKobo(item.principal_due_kobo),
			allocated = exactKobo(item.allocated_kobo);
		return due !== null && allocated !== null && due > allocated;
	}
	let changes: Change[] = $state([]),
		context: ChangeContext | null = $state(null),
		reference = $state(''),
		kind = $state('write_off'),
		amount = $state(''),
		reason = $state(''),
		expires = $state(''),
		dates: Record<string, string> = $state({}),
		notes: Record<string, string> = $state({});
	let proposalID = $state(crypto.randomUUID()),
		busy = $state(false),
		error = $state(''),
		message = $state(''),
		actor = $state(''),
		roles: string[] = $state([]),
		q = $state(''),
		offset = $state(0),
		more = $state(false);
	let isPlatformOwner = $state(false),
		governanceMode = $state('unavailable');
	let canPropose = $derived(
		roles.includes('platform_admin') || roles.includes('finance_operator') || roles.includes('platform_owner')
	);
	let canApprove = $derived(
		roles.includes('platform_admin') || roles.includes('approver') || roles.includes('platform_owner')
	);
	async function load() {
		const read = reads.begin();
		busy = true;
		error = '';
		changes = [];
		more = false;
		roles = [];
		governanceMode = 'unavailable';
		isPlatformOwner = false;
		try {
			const [b, c, gov] = await Promise.all([
				adminGet(`/api/v1/ops/admin-changes?q=${encodeURIComponent(q)}&offset=${offset}`, read.signal),
				adminGet('/api/v1/ops/capabilities', read.signal),
				adminGet('/api/v1/ops/governance', read.signal)
			]);
			if (!read.current()) return;
			const mode = gov.governance && typeof gov.governance === 'object' ? record(gov.governance).mode : undefined;
			if (
				!Array.isArray(b.changes) ||
				!Array.isArray(c.roles) ||
				typeof mode !== 'string' ||
				!['solo_owner', 'delegated_team'].includes(mode)
			)
				throw new Error('Proposals and approval rules could not be verified.');
			const verified = rows('changes', change)(b);
			const verifiedRoles = c.roles.map(text);
			const actorID = text(c.actor_id);
			changes = verified.slice(0, 100);
			more = verified.length > 100;
			actor = actorID;
			roles = verifiedRoles;
			isPlatformOwner = verifiedRoles.includes('platform_owner');
			governanceMode = mode;
		} catch (e) {
			if (read.current()) error = e instanceof Error ? e.message : 'That did not go through. Try again.';
		} finally {
			if (read.current()) busy = false;
		}
	}
	async function lookup() {
		if (busy) return;
		busy = true;
		error = '';
		context = null;
		try {
			const loaded = await adminGet(`/api/v1/ops/change-context?q=${encodeURIComponent(reference)}`);
			const verified: ChangeContext = {
				obligation_id: text(loaded.obligation_id),
				snapshot: snapshot(loaded.snapshot, true)
			};
			context = verified;
			dates = Object.fromEntries(
				verified.snapshot.items.filter((i) => !i.cancelled && unpaid(i)).map((i) => [i.id, localInput(i.due_at)])
			);
			proposalID = crypto.randomUUID();
		} catch (e) {
			error = e instanceof Error ? e.message : 'That did not go through. Try again.';
		} finally {
			busy = false;
		}
	}
	async function propose() {
		if (busy || !context) return;
		const obligationID = context.obligation_id;
		busy = true;
		error = '';
		message = '';
		try {
			const values =
				kind === 'schedule_amendment'
					? {
							amount_kobo: 0,
							dates: Object.entries(dates).map(([item_id, date]) => ({ item_id, due_at: lagosISO(date) }))
						}
					: { amount_kobo: parseNaira(amount), dates: [] };
			const expectedID = proposalID;
			await write(
				'/api/v1/ops/admin-changes',
				{ id: expectedID, obligation_id: obligationID, kind, reason, values, expires_at: lagosISO(expires) },
				(value) => {
					if (record(value).id !== expectedID) throw new Error('Proposal was not confirmed');
					return true;
				}
			);
			context = null;
			amount = '';
			reason = '';
			message = 'Proposal recorded for review.';
			await load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'That did not go through. Try again.';
		} finally {
			busy = false;
		}
	}
	async function decide(c: Change, action: string) {
		if (busy) return;
		busy = true;
		error = '';
		try {
			await write(
				`/api/v1/ops/admin-changes/${encodeURIComponent(c.id)}/decision`,
				{ action, reason: notes[c.id] || '' },
				(value) => {
					if (record(value).recorded !== true) throw new Error('Decision was not confirmed');
					return true;
				}
			);
			message = 'Decision recorded.';
			await load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'That did not go through. Try again.';
		} finally {
			busy = false;
		}
	}
	async function soloApprove(c: Change) {
		if (busy) return;
		busy = true;
		error = '';
		message = '';
		try {
			await write(
				'/api/v1/ops/solo-owner/approve',
				{ target_type: 'financial_change', target_id: c.id, reason: notes[c.id] || '', confirm: true },
				(value) => {
					if (record(value).approved !== true) throw new Error('Approval was not confirmed');
					return true;
				}
			);
			message = 'Approved. The change is recorded.';
			await load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'That did not go through. Try again.';
		} finally {
			busy = false;
		}
	}
	onMount(() => {
		q = new URLSearchParams(location.search).get('change') || '';
		void load();
		return () => reads.cancel();
	});
</script>

<svelte:head><title>Financial changes — Kredit admin</title></svelte:head>
<main class="shell workspace">
	<h1>Financial changes</h1>
	<VerifyIdentity />
	<p>
		{governanceMode === 'solo_owner'
			? 'You approve your own corrections, with a fresh authenticator code and a written reason.'
			: governanceMode === 'delegated_team'
				? 'Every correction needs a second administrator to approve it.'
				: 'Who approves a correction could not be read just now. Record the proposal; the approval rule is applied when the decision is made.'}
		Changing a payment date also needs the customer to accept it. The amounts and the original agreement are never overwritten
		when dates change.
	</p>
	{#if error}<p role="alert">{error}</p>{/if}{#if message}<p role="status">{message}</p>{/if}
	{#if canPropose}<section>
			<h2>Propose a change</h2>
			<form
				onsubmit={(e) => {
					e.preventDefault();
					lookup();
				}}
			>
				<label>Sale or obligation reference<input disabled={busy} bind:value={reference} required /></label><button
					disabled={busy}>Load the current details</button
				>
			</form>
			{#if context}<p>Outstanding: <strong>{formatKobo(context.snapshot.outstanding_kobo)}</strong></p>
				<form
					onsubmit={(e) => {
						e.preventDefault();
						propose();
					}}
				>
					<label
						>Change<select disabled={busy} bind:value={kind}
							><option value="write_off">Write off money owed</option><option value="fee_waiver"
								>Cancel seller fees</option
							><option value="schedule_amendment">Change payment dates</option></select
						></label
					>
					{#if kind === 'schedule_amendment'}<p>
							List every unpaid part-payment in its current order. Dates must leave enough notice. Any debit with an
							unclear result must be settled first.
						</p>
						{#each context.snapshot.items.filter((i) => i.id in dates) as item (item.id)}<label
								>{difference(item?.principal_due_kobo, item?.allocated_kobo)} · currently {localTime(
									item?.due_at || ''
								)}<input type="datetime-local" disabled={busy} bind:value={dates[item.id]} required /></label
							>{/each}{:else}<label
							>Amount (₦)<input disabled={busy} inputmode="decimal" bind:value={amount} required /></label
						>{/if}
					<label
						>Proposal expires (Lagos, within 30 days)<input
							type="datetime-local"
							disabled={busy}
							bind:value={expires}
							required
						/></label
					><label
						>Reason<textarea disabled={busy} bind:value={reason} minlength="8" maxlength="2000" required
						></textarea></label
					><button disabled={busy}
						>{governanceMode === 'solo_owner' ? 'Record this proposal' : 'Send for approval'}</button
					>
				</form>{/if}
		</section>{/if}
	<h2>Review proposals</h2>
	<form
		onsubmit={(e) => {
			e.preventDefault();
			offset = 0;
			load();
		}}
	>
		<label>Proposal or obligation reference<input type="search" bind:value={q} /></label><button disabled={busy}
			>Find a proposal</button
		>
	</form>
	{#each changes as c (c.id)}<article id={c.id}>
			<h3>{c.kind.replaceAll('_', ' ')} · {c.state.replaceAll('_', ' ')}</h3>
			<p>{c.reason}</p>
			<p>
				Proposed by {c.proposer}{#if c.approver}
					· Approved by {c.approver}{/if}
			</p>
			<p>Expires {localTime(c.expires_at)} · Reference {c.obligation_id}</p>
			{#if c.kind === 'schedule_amendment'}<table>
					<thead><tr><th>Unpaid amount</th><th>Current date at proposal</th><th>Proposed date</th></tr></thead><tbody
						>{#each c.proposed_values.dates as date, idx (idx)}{@const item = c.before_values.items.find(
								(i) => i.id === date.item_id
							)}<tr
								><td>{difference(item?.principal_due_kobo, item?.allocated_kobo)}</td><td
									>{localTime(item?.due_at || '')}</td
								><td>{localTime(date.due_at)}</td></tr
							>{/each}</tbody
					>
				</table>
				<p>Approval sends these dates to the buyer. Dates take effect only after the buyer accepts.</p>{:else}<p>
					Proposed {c.kind === 'write_off' ? 'principal reduction' : 'fee waiver'}:
					<strong>{formatKobo(c.proposed_values.amount_kobo)}</strong>
				</p>
				<p>Outstanding at proposal: {formatKobo(c.before_values.outstanding_kobo)}</p>{/if}
			{#if ['pending', 'awaiting_buyer'].includes(c.state)}<label
					>Decision notes<textarea disabled={busy} bind:value={notes[c.id]} minlength="8" maxlength="2000"
					></textarea></label
				>{#if canApprove && c.state === 'pending' && c.proposed_by !== actor}<button
						disabled={busy || (notes[c.id] || '').trim().length < 8}
						onclick={() => decide(c, 'approve')}>Approve exactly this</button
					><button disabled={busy || (notes[c.id] || '').trim().length < 8} onclick={() => decide(c, 'reject')}
						>Reject</button
					>{/if}{#if c.state === 'pending' && c.proposed_by === actor && isPlatformOwner && governanceMode === 'solo_owner'}<button
						class="primary"
						disabled={busy || (notes[c.id] || '').trim().length < 8}
						onclick={() => soloApprove(c)}>Approve my own change</button
					>{/if}{#if c.proposed_by === actor || canApprove}<button
						disabled={busy || (notes[c.id] || '').trim().length < 8}
						onclick={() => decide(c, 'cancel')}>Cancel proposal</button
					>{/if}{/if}
			<p><a href={`/admin/history?q=${c.id}`}>See every decision</a></p>
		</article>{:else}{#if !busy && !error}<p>No matching proposals.</p>{/if}{/each}
	<button
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
	section,
	article {
		border: 1px solid var(--color-border);
		padding: 1.2rem;
		background: var(--color-surface);
		margin: 1.5rem 0;
		overflow: auto;
	}
	label {
		display: block;
		margin: 0.8rem 0;
	}
	input,
	select,
	textarea {
		display: block;
		width: 30rem;
		max-width: 100%;
		padding: 0.7rem;
	}
	button {
		padding: 0.7rem;
		margin: 0.5rem 0.5rem 0.5rem 0;
	}
	th,
	td {
		text-align: left;
		padding: 0.7rem;
		border-bottom: 1px solid var(--color-border);
	}
	p {
		overflow-wrap: anywhere;
	}
</style>
