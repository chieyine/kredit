<script lang="ts">
	import { onMount } from 'svelte';
	import { checkedJSON, optionalNumber, optionalText, publicError, record, rows, text } from '$lib/api/reliable';
	type Queue = { queue: string; count: number; oldest_seconds: number };
	type ProviderHealth = { provider: string; total: number; errors: number; oldest_unprocessed_seconds: number };
	type Diagnostics = {
		correlation_id: string;
		integrity: Record<string, unknown>;
		queues: Queue[];
		provider: ProviderHealth[];
	};
	let data: Diagnostics | null = $state(null),
		error = $state('');
	async function load() {
		data = null;
		error = '';
		try {
			data = await checkedJSON('/api/v1/ops/diagnostics?window_minutes=60', (value) => {
				const diagnostics = record(record(value).diagnostics);
				return {
					correlation_id: optionalText(diagnostics.correlation_id),
					integrity: record(diagnostics.integrity),
					queues: rows('queues', (value): Queue => {
						const item = record(value);
						return {
							queue: text(item.queue),
							count: optionalNumber(item.count),
							oldest_seconds: optionalNumber(item.oldest_seconds)
						};
					})(diagnostics),
					provider: rows('provider', (value): ProviderHealth => {
						const item = record(value);
						return {
							provider: text(item.provider),
							total: optionalNumber(item.total),
							errors: optionalNumber(item.errors),
							oldest_unprocessed_seconds: optionalNumber(item.oldest_unprocessed_seconds)
						};
					})(diagnostics)
				};
			});
		} catch (cause) {
			error = publicError(cause, 'system diagnostics');
		}
	}
	onMount(load);
</script>

<svelte:head><title>Operations diagnostics — Kredit</title></svelte:head>
<main class="shell workspace">
	<p class="eyebrow">Operations / Diagnostics</p>
	<h1>The last hour</h1>
	<p>
		Provider latency signals, webhook lag, queue age, reconciliation, drift, dead letters, notifications, scanning,
		mandates and settlements, without raw payloads or unredacted correlation identifiers.
	</p>
	{#if error}<section role="alert">
			<p class="error">{error}</p>
			<button type="button" onclick={load}>Try again</button>
		</section>{:else if !data}<p>Loading diagnostics…</p>{:else}<p>Correlation: <code>{data.correlation_id}</code></p>
		<h2>Integrity signals</h2>
		<section>
			{#each Object.entries(data.integrity) as [name, value] (name)}<article>
					<strong>{value}</strong><span>{name.replaceAll('_', ' ')}</span>
				</article>{/each}
		</section>
		<h2>Queues</h2>
		{#each data.queues as queue (queue.queue)}<p>
				<strong>{queue.queue}</strong> · {queue.count} waiting · oldest {queue.oldest_seconds}s
			</p>{:else}<p>No queue backlog.</p>{/each}
		<h2>Providers</h2>
		{#each data.provider as item (item.provider)}<p>
				<strong>{item.provider}</strong> · {item.total} events · {item.errors} errors · oldest unprocessed {item.oldest_unprocessed_seconds}s
			</p>{:else}<p>No provider activity in this window.</p>{/each}{/if}
</main>

<style>
	section {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
		gap: 1rem;
	}
	article {
		padding: 1rem;
		border: 1px solid var(--color-border);
		border-radius: 1rem;
	}
	article strong,
	article span {
		display: block;
	}
</style>
