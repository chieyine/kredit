<script lang="ts">
	import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
	import { checkedJSON, optionalText, publicError, record, rows, text } from '$lib/api/reliable';
	let error = $state('');
	import { productLabel } from '$lib/product-language';
	import { onMount } from 'svelte';
	import { idempotencyKey } from '$lib/api/client';
	import { adminPost } from '$lib/admin-client';
	let actionKey = '';
	import ProtectedActionDialog from '$lib/components/ProtectedActionDialog.svelte';
	type PrivacyRequest = {
		id: string;
		request_type: string;
		state: string;
		due_at: string;
		details: string;
		decided_by: string;
		version: number;
	};
	let items = $state<PrivacyRequest[]>([]),
		selected = $state<PrivacyRequest | null>(null),
		decision = $state(''),
		message = $state(''),
		dialogOpen = $state(false),
		loading = $state(true);
	function privacyRecord(value: unknown): PrivacyRequest {
		const item = record(value);
		const due = text(item.due_at);
		if (!Number.isSafeInteger(item.version) || Number(item.version) < 1 || !Number.isFinite(Date.parse(due)))
			throw new Error('Unverified privacy request');
		return {
			id: text(item.id),
			request_type: text(item.request_type),
			state: text(item.state),
			due_at: due,
			details: optionalText(item.details),
			decided_by: optionalText(item.decided_by),
			version: Number(item.version)
		};
	}
	async function load() {
		loading = true;
		error = '';
		try {
			items = await checkedJSON('/api/v1/ops/privacy-requests', rows('requests', privacyRecord));
		} catch (cause) {
			error = publicError(cause, 'privacy requests');
		} finally {
			loading = false;
		}
	}
	function begin(item: PrivacyRequest, nextDecision: string) {
		selected = item;
		decision = nextDecision;
		actionKey = '';
		dialogOpen = true;
	}
	async function confirm(reason: string) {
		if (!selected) throw new Error('Choose a request first.');
		const target = selected;
		actionKey ||= idempotencyKey();
		const completing = decision === 'COMPLETE';
		const result = await adminPost(
			`/api/v1/ops/privacy-requests/${encodeURIComponent(target.id)}/${completing ? 'complete' : 'decide'}`,
			completing
				? { decision_reviewer_id: target.decided_by, expected_version: target.version, reason }
				: { decision, reason, expected_version: target.version },
			'POST',
			actionKey
		);
		const saved = privacyRecord(record(result).request);
		if (
			saved.id !== target.id ||
			!(completing
				? saved.state === 'COMPLETED'
				: saved.state === decision || (decision === 'APPROVED' && saved.state === 'PARTIALLY_APPROVED')) ||
			saved.version <= target.version
		)
			throw new Error('The result could not be confirmed. Check this request before trying again.');
		message = completing
			? 'Completed work recorded. The customer can see your completion explanation.'
			: saved.state === 'PARTIALLY_APPROVED'
				? 'Partially approved. Retained records and processing restrictions are recorded.'
				: 'Decision recorded. Complete the approved work under your current governance settings.';
		actionKey = '';
		await load();
		return true;
	}
	onMount(load);
</script>

<svelte:head><title>Privacy review — Kredit</title></svelte:head>
<main class="shell workspace">
	<p class="eyebrow">Operations / Privacy</p>
	<h1>Privacy request queue.</h1>
	<p class="lede">
		Keep the financial records the law requires. In solo-owner mode, the owner can finish an approved request with a
		recorded reason. Delegated teams use a different reviewer.
	</p>
	<VerifyIdentity />{#if message}<p class="notice" role="status">{message}</p>{/if}{#if error}<section role="alert">
			<p class="error">{error}</p>
			<button type="button" onclick={load}>Try again</button>
		</section>{:else if loading}<p role="status">
			Loading privacy requests…
		</p>{:else}{#each items as item (item.id)}<article>
				<header>
					<strong>{productLabel(item.request_type)}</strong><span class="status">{productLabel(item.state)}</span>
				</header>
				<p>Due {new Date(item.due_at).toLocaleDateString('en-NG')} · {item.details}</p>
				<div class="actions">
					{#if item.state === 'APPROVED' || item.state === 'PARTIALLY_APPROVED'}<button
							onclick={() => begin(item, 'COMPLETE')}>Record completed work</button
						>{:else if item.state !== 'COMPLETED'}<button onclick={() => begin(item, 'APPROVED')}>Approve</button
						><button onclick={() => begin(item, 'CLARIFICATION_REQUIRED')}>Ask for more information</button><button
							class="danger"
							onclick={() => begin(item, 'REJECTED')}>Reject</button
						>{/if}
				</div>
			</article>{:else}<section class="empty-state">
				<h2>No privacy requests</h2>
				<p>New information requests will appear here.</p>
			</section>{/each}{/if}
</main>
{#if selected}<ProtectedActionDialog
		bind:open={dialogOpen}
		title={decision === 'COMPLETE'
			? 'Finish privacy request'
			: `${decision.replaceAll('_', ' ').toLowerCase()} privacy request`}
		description={decision === 'COMPLETE'
			? 'Describe the work already completed and any information retained. For correction or deletion, complete the approved work before recording it here; this review action does not itself edit or delete account records. Your explanation is shown to the customer.'
			: 'Write the reason for this decision. Completion follows your current owner or delegated-team governance settings.'}
		confirmLabel={decision === 'COMPLETE' ? 'Finish request' : 'Record decision'}
		onconfirm={confirm}
	/>{/if}

<style>
	article {
		margin: 1rem 0;
		padding: 1rem;
		border: 1px solid var(--color-border);
		border-radius: 1rem;
		background: var(--color-surface);
	}
	header {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
	}
	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}
	.actions button {
		padding: 0.65rem 0.85rem;
		border: 1px solid var(--color-border);
		border-radius: 0.6rem;
		background: var(--color-surface);
		color: var(--color-foreground);
		font: inherit;
		font-weight: 700;
		cursor: pointer;
	}
	.actions .danger {
		color: var(--color-destructive);
	}
</style>
