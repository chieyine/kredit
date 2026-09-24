<script lang="ts">
	import { checkedJSON, publicError, record, rows, text } from '$lib/api/reliable';
	let error = $state('');
	import { onMount } from 'svelte';
	import { idempotencyKey } from '$lib/api/client';
	import { adminPost, commandPreview } from '$lib/admin-client';
	let actionKey = '';
	import ProtectedActionDialog from '$lib/components/ProtectedActionDialog.svelte';
	type ProviderEvent = { provider: string; event_id: string; event_type: string; state: string; attempts: number };
	function providerEvent(value: unknown): ProviderEvent {
		const item = record(value);
		if (!Number.isSafeInteger(item.attempts)) throw new Error('Incomplete provider event');
		return {
			provider: text(item.provider),
			event_id: text(item.event_id),
			event_type: text(item.event_type),
			state: text(item.state),
			attempts: Number(item.attempts)
		};
	}
	let events = $state<ProviderEvent[]>([]),
		selected = $state<ProviderEvent | null>(null),
		message = $state(''),
		dialogOpen = $state(false),
		loading = $state(true);
	async function load() {
		loading = true;
		error = '';
		try {
			events = await checkedJSON('/api/v1/ops/provider-events', rows('events', providerEvent));
		} catch (cause) {
			error = publicError(cause, 'provider events');
		} finally {
			loading = false;
		}
	}
	function input(reason: string) {
		if (!selected) throw new Error('Choose an event first.');
		return {
			command_type: 'retry_webhook',
			target_type: selected.provider,
			target_id: selected.event_id,
			reason,
			expected_version: selected.attempts + 1
		};
	}
	async function preview(reason: string) {
		const { impact_preview: impact } = commandPreview(await adminPost('/api/v1/ops/commands/preview', input(reason)));
		actionKey = idempotencyKey();
		return impact.effect;
	}
	async function confirm(reason: string) {
		actionKey ||= idempotencyKey();
		await adminPost('/api/v1/ops/commands', input(reason), 'POST', actionKey);
		message = 'Webhook safely requeued.';
		actionKey = '';
		await load();
		return true;
	}

	onMount(load);
</script>

<svelte:head><title>Provider events — Kredit</title></svelte:head>
<main class="shell workspace">
	<p class="eyebrow">Operations / Provider events</p>
	<h1>Webhook processing history.</h1>
	<p class="lede">Raw payloads remain private. Duplicate identity is preserved through every controlled replay.</p>
	{#if message}<p class="notice" role="status">{message}</p>{/if}{#if error}<section role="alert">
			<p class="error">{error}</p>
			<button type="button" onclick={load}>Try again</button>
		</section>{:else if loading}<p role="status">
			Loading provider events…
		</p>{:else}{#each events as item (`${item.provider}:${item.event_id}`)}<article>
				<header><strong>{item.provider} · {item.event_type}</strong><span class="status">{item.state}</span></header>
				<code>{item.event_id}</code>
				<p>{item.attempts} processing attempts</p>
				{#if item.state === 'failed'}<button
						onclick={() => {
							selected = item;
							dialogOpen = true;
						}}>Preview safe replay</button
					>{/if}
			</article>{:else}<section class="empty-state">
				<h2>No provider events found</h2>
				<p>No events are visible in the selected operational window.</p>
			</section>{/each}{/if}
</main>
{#if selected}<ProtectedActionDialog
		bind:open={dialogOpen}
		title="Replay this provider event safely"
		description="The original provider event identity and idempotency controls are preserved. Confirm only after checking the provider's final state."
		confirmLabel="Confirm safe replay"
		{preview}
		onconfirm={confirm}
	/>{/if}

<style>
	article {
		padding: 1rem;
		margin: 1rem 0;
		border: 1px solid var(--color-border);
		border-radius: 1rem;
		background: var(--color-surface);
	}
	header {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
	}
	article button {
		padding: 0.7rem;
		border: 1px solid var(--color-border);
		border-radius: 0.6rem;
		background: var(--color-surface);
		color: var(--color-foreground);
		font: inherit;
		font-weight: 700;
		cursor: pointer;
	}
</style>
