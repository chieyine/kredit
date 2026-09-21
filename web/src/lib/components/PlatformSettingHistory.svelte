<script lang="ts">
	import OwnerDialog from '$lib/components/OwnerDialog.svelte';
	import { localTime } from '$lib/admin-client';
	import type { SettingHistory } from '$lib/admin/platform-settings';

	let {
		settingKey,
		entries,
		loading,
		error,
		readable,
		onretry,
		onclose
	}: {
		settingKey: string;
		entries: SettingHistory[];
		loading: boolean;
		error: string;
		readable: (value: unknown) => string;
		onretry: (key: string) => void;
		onclose: () => void;
	} = $props();
</script>

<OwnerDialog
	open={true}
	title="History"
	description="Every change to this setting, oldest last. This record cannot be edited."
	{onclose}
>
	{#if loading}
		<div class="loading-state">Opening history…</div>
	{:else if error}<p role="alert" class="error">{error}</p>
		<button type="button" onclick={() => onretry(settingKey)}>Try again</button>
	{:else if entries.length === 0}
		<div class="empty-state">This setting has not been changed yet.</div>
	{:else}
		<div class="table-wrap">
			<table class="history-table">
				<thead>
					<tr>
						<th>Version</th>
						<th>Action</th>
						<th>Changed to</th>
						<th>When</th>
						<th>Reason and who</th>
					</tr>
				</thead>
				<tbody>
					{#each entries as h}
						<tr>
							<td>v{h.version}</td>
							<td><span class="action-tag">{h.action}</span></td>
							<td>{readable(h.new_value)}</td>
							<td>{localTime(h.recorded_at)}</td>
							<td>
								<strong>{h.reason}</strong>
								{#if h.actor_id}<p class="actor-sub">Changed by {h.actor_id}</p>{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</OwnerDialog>

<style>
	.table-wrap {
		overflow-x: auto;
	}

	.history-table {
		width: 100%;
		border-collapse: collapse;
		text-align: left;
		font-size: 0.9rem;
	}

	.history-table th {
		background: var(--color-surface);
		padding: 0.8rem 1rem;
		border-bottom: 1px solid var(--color-border);
		font-weight: 600;
		color: var(--color-primary);
	}

	.history-table td {
		padding: 0.8rem 1rem;
		border-bottom: 1px solid var(--color-border);
		vertical-align: top;
	}

	.actor-sub {
		margin: 0.2rem 0 0;
		font-size: 0.8rem;
		color: var(--color-muted, #666);
	}

	.action-tag {
		display: inline-block;
		padding: 0.1rem 0.5rem;
		border-radius: 999px;
		background: var(--color-surface);
		font-size: 0.8rem;
	}

	.loading-state,
	.empty-state {
		padding: 1rem;
		color: var(--color-muted, #666);
	}

	.error {
		color: var(--color-danger, #b00020);
	}
</style>
