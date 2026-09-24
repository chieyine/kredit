<script lang="ts">
	import { exactKobo, formatKobo, nairaInput, parseNaira, type KoboValue } from '$lib/money';
	import { timeLabel } from '$lib/records';
	import { page } from '$app/state';
	import { checkedJSON, record, text, LatestRequest, publicError } from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';
	import { disputeDetail, type Dispute, type DisputeDecision, type DisputeEvidence } from '$lib/disputes';
	let dispute = $state<Dispute | null>(null),
		evidence: DisputeEvidence[] = $state([]),
		decisions: DisputeDecision[] = $state([]),
		error = $state(''),
		actionError = $state(''),
		notice = $state(''),
		loading = $state(true),
		busy = $state(false),
		outcome = $state('VALID_AMOUNT_CONFIRMED'),
		validNaira = $state(''),
		adjustmentNaira = $state('0.00'),
		remainingNaira = $state(''),
		reason = $state('');
	const money = (value: KoboValue) => formatKobo(value);
	const reads = new LatestRequest();
	let intent: MutationIntent | null = null;
	async function load() {
		const request = reads.begin(),
			id = page.params.id!;
		loading = true;
		error = '';
		dispute = null;
		try {
			const data = await checkedJSON(
				`/api/v1/ops/disputes/${encodeURIComponent(id)}`,
				(value) => {
					const detail = disputeDetail(value);
					if (detail.dispute.id !== id) throw new Error('Dispute identity mismatch');
					if (!detail.dispute.collection_effect || !detail.dispute.reason) throw new Error('Incomplete dispute record');
					return detail;
				},
				{ signal: request.signal }
			);
			if (!request.current()) return;
			dispute = data.dispute;
			evidence = data.evidence;
			decisions = data.decisions;
			remainingNaira = nairaInput(data.dispute.remaining_disputed_kobo);
			validNaira = nairaInput(validPrincipal(data.dispute));
		} catch (cause) {
			if (request.current()) error = publicError(cause, 'this dispute');
		} finally {
			if (request.current()) loading = false;
		}
	}
	function validPrincipal(item: Dispute): KoboValue {
		const total = exactKobo(item.total_disputed_kobo),
			rest = exactKobo(item.remaining_disputed_kobo);
		if (total === null || rest === null) return null;
		const value = total - rest;
		return value > 0n ? value : 0n;
	}
	async function openDocument(id: string) {
		try {
			const url = await checkedJSON(
				`/api/v1/ops/disputes/${encodeURIComponent(page.params.id!)}/documents/${encodeURIComponent(id)}/download`,
				(value) => {
					const url = new URL(text(record(value).url), location.origin);
					if (
						!['https:', 'http:'].includes(url.protocol) ||
						url.username ||
						url.password ||
						(location.protocol === 'https:' && url.protocol !== 'https:')
					)
						throw new Error('Invalid document link');
					return url.href;
				}
			);
			location.assign(url);
		} catch (cause) {
			actionError = publicError(cause, 'this document');
		}
	}
	async function decide(e: SubmitEvent) {
		e.preventDefault();
		if (busy || !intent) return;
		const id = page.params.id,
			write = intent;
		const amounts = {
			valid_principal_kobo: parseNaira(validNaira),
			adjustment_kobo: parseNaira(adjustmentNaira),
			remaining_disputed_kobo: parseNaira(remainingNaira)
		};
		if (Object.values(amounts).some((v) => !Number.isSafeInteger(v) || v < 0)) {
			actionError = 'Check the three amounts. Use figures only, with up to two decimal places.';
			return;
		}
		busy = true;
		actionError = '';
		notice = '';
		try {
			await write.run({ outcome, ...amounts, reason }, (value) => {
				const body = record(value);
				if (text(record(body.dispute).id) !== id) throw new Error('Unconfirmed dispute');
				text(record(body.decision).id);
				return body;
			});
			if (id === page.params.id) {
				notice = 'Decision recorded. The sale balance and dispute history were updated.';
				reason = '';
				await load();
			}
		} catch (cause) {
			if (id === page.params.id)
				actionError = cause instanceof Error ? cause.message : 'The decision could not be confirmed.';
		} finally {
			busy = false;
		}
	}
	$effect(() => {
		const id = page.params.id;
		if (id) {
			notice = '';
			reason = '';
			actionError = '';
			adjustmentNaira = '0.00';
			intent = new MutationIntent('admin-dispute', `/api/v1/ops/disputes/${encodeURIComponent(id)}/decide`);
			void load();
		}
		return () => reads.cancel();
	});
</script>

<svelte:head><title>Dispute review — Kredit</title></svelte:head>
<main class="shell workspace">
	<a href="/admin/disputes">← All disputes</a>
	<p class="eyebrow">Admin / Dispute</p>
	<h1>Dispute review</h1>
	<p class="lede">Evidence and decisions are shown together before any money is changed.</p>
	{#if notice}<p class="notice" role="status">{notice}</p>{/if}{#if actionError}<p class="error" role="alert">
			{actionError}
		</p>{/if}{#if loading}<p>Loading dispute…</p>{:else if error}<section role="alert">
			<p class="error">{error}</p>
			<button onclick={load} disabled={busy}>Reload dispute</button>
		</section>{:else if dispute}<section class="summary">
			<div><span>Status</span><strong>{dispute.state.replaceAll('_', ' ')}</strong></div>
			<div><span>Disputed</span><strong>{money(dispute.total_disputed_kobo)}</strong></div>
			<div><span>Still disputed</span><strong>{money(dispute.remaining_disputed_kobo)}</strong></div>
			<div><span>Bank-debit effect</span><strong>{dispute.collection_effect.replaceAll('_', ' ')}</strong></div>
		</section>
		<article>
			<h2>{dispute.reason}</h2>
			<p>{dispute.explanation}</p>
		</article>
		<div class="columns">
			<section>
				<h2>Evidence</h2>
				{#if evidence.length}<ol>
						{#each evidence as item, i (i)}<li>
								<p>{item.statement || 'Document evidence'}</p>
								{#if item.document_id}<button type="button" onclick={() => openDocument(item.document_id)}
										>Open document</button
									>{/if}<span>{timeLabel(item.submitted_at)}</span>
							</li>{/each}
					</ol>{:else}<p>No evidence has been recorded.</p>{/if}
				<h2>Earlier decisions</h2>
				{#each decisions as item, i (i)}<div class="decision">
						<strong>{item.outcome.replaceAll('_', ' ')}</strong>
						<p>{item.reason}</p>
						<small>{timeLabel(item.decided_at)}</small>
					</div>{:else}<p>No decision has been recorded.</p>{/each}
			</section>
			{#if !['RESOLVED', 'WITHDRAWN'].includes(dispute.state)}<section class="action">
					<h2>Record a decision</h2>
					<form onsubmit={decide}>
						<fieldset disabled={busy}>
							<label
								>Decision<select bind:value={outcome}
									><option value="VALID_AMOUNT_CONFIRMED">Confirm the valid amount</option><option
										value="PARTIAL_ADJUSTMENT">Make a partial adjustment</option
									><option value="FULL_ADJUSTMENT">Remove the disputed amount</option></select
								></label
							><label>Correct sale amount (₦)<input inputmode="decimal" maxlength="40" bind:value={validNaira} /></label
							><label
								>Amount to remove from the balance (₦)<input
									inputmode="decimal"
									maxlength="40"
									bind:value={adjustmentNaira}
								/></label
							><label
								>Amount still in question (₦)<input
									inputmode="decimal"
									maxlength="40"
									bind:value={remainingNaira}
								/></label
							><label>Reason<textarea bind:value={reason} minlength="8" rows="4" required></textarea></label><button
								disabled={busy}>{busy ? 'Saving…' : 'Record this decision'}</button
							>
						</fieldset>
					</form>
				</section>{/if}
		</div>{/if}
</main>

<style>
	.action fieldset {
		display: grid;
		gap: 0.7rem;
		border: 0;
		padding: 0;
		margin: 0;
		min-width: 0;
	}
	h1 {
		font-family: var(--font-serif);
		font-size: clamp(2.5rem, 6vw, 4.5rem);
		font-weight: 500;
	}
	.summary,
	.columns {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 1rem;
		margin: 1.5rem 0;
	}
	.summary {
		grid-template-columns: repeat(4, minmax(0, 1fr));
	}
	.summary div,
	article,
	li,
	.decision {
		padding: 1rem;
		border: 1px solid var(--color-border);
		background: var(--color-surface);
	}
	.summary div {
		display: grid;
		gap: 0.3rem;
	}
	.summary span,
	li span {
		color: var(--color-muted);
	}
	ol {
		display: grid;
		gap: 0.7rem;
		padding: 0;
		list-style: none;
	}
	.decision {
		margin: 0.7rem 0;
	}
	.action {
		align-self: start;
		padding: 1.2rem;
		background: var(--color-primary);
		color: var(--color-on-primary);
	}
	.action form,
	.action label {
		display: grid;
		gap: 0.7rem;
	}
	.action label {
		gap: 0.3rem;
	}
	.action input,
	.action select,
	.action textarea,
	.action button {
		padding: 0.7rem;
		border: 1px solid var(--color-border-strong);
		background: var(--color-foreground);
		color: var(--color-on-primary);
		font: inherit;
	}
	.action button {
		background: var(--color-accent);
		color: var(--color-on-primary);
		font-weight: 850;
	}
	@media (max-width: 760px) {
		.summary,
		.columns {
			grid-template-columns: 1fr;
		}
	}
</style>
