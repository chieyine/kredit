<script lang="ts">
	import { onMount } from 'svelte';
	import AdminAttention from '$lib/components/AdminAttention.svelte';
	import { adminGet, localTime } from '$lib/admin-client';
	import { optionalText, record, rows, text } from '$lib/api/reliable';
	type ExpiringMandate = { id: string; expires_at: string; buyer_id: string; buyer: string };
	type UncertainDebit = { id: string; state: string; requested_at: string };
	type FailedNotice = { id: string; recipient: string; template: string; channel: string; reason: string };
	type Details = { mandates: ExpiringMandate[]; debits: UncertainDebit[]; notices: FailedNotice[] };
	function details(body: Record<string, unknown>): Details {
		return {
			mandates: rows('mandates', (value): ExpiringMandate => {
				const item = record(value);
				return {
					id: text(item.id),
					expires_at: text(item.expires_at),
					buyer_id: optionalText(item.buyer_id),
					buyer: optionalText(item.buyer)
				};
			})(body),
			debits: rows('debits', (value): UncertainDebit => {
				const item = record(value);
				return { id: text(item.id), state: text(item.state), requested_at: text(item.requested_at) };
			})(body),
			notices: rows('notices', (value): FailedNotice => {
				const item = record(value);
				return {
					id: text(item.id),
					recipient: optionalText(item.recipient),
					template: optionalText(item.template),
					channel: optionalText(item.channel),
					reason: optionalText(item.reason)
				};
			})(body)
		};
	}
	let data: Details | null = $state(null),
		error = $state('');
	async function load() {
		data = null;
		error = '';
		try {
			const body = await adminGet('/api/v1/ops/attention/details');
			data = details(body);
		} catch (e) {
			error = e instanceof Error ? e.message : 'We could not open this. Try again.';
		}
	}
	onMount(load);
</script>

<svelte:head><title>Work needing attention — Kredit</title></svelte:head>
<main class="shell workspace">
	<h1>Work needing attention</h1>
	<AdminAttention />{#if error}<p role="alert">{error} <button onclick={load}>Try again</button></p>{/if}{#if data}<p>
			Shows up to 100 records in each group. Settle existing cases through their own recorded process.
		</p>
		<h2>Bank debit permissions about to expire</h2>
		{#each data.mandates as item (item.id)}<article>
				<h3>{item.buyer}</h3>
				<p>Expires {localTime(item.expires_at)} · Reference {item.id}</p>
				<p>
					Ask the customer to review their bank debit permission in their account. The payment company needs a fresh
					permission.
				</p>
				<a href={`/admin/users?q=${encodeURIComponent(item.buyer_id || '')}`}>Open customer account →</a>
			</article>{:else}<p>No bank debit permission is close to expiring.</p>{/each}
		<h2>Debits with an unclear result</h2>
		{#each data.debits as item (item.id)}<article>
				<p>{item.state} since {localTime(item.requested_at)} · Reference {item.id}</p>
				<a href={`/admin/controls?target_type=collection&target_id=${encodeURIComponent(item.id)}`}
					>Look up this debit →</a
				>
			</article>{:else}<p>Every debit has a clear result.</p>{/each}
		<h2>Messages that failed to send</h2>
		{#each data.notices as item (item.id)}<article>
				<h3>{item.recipient} · {item.template}</h3>
				<p>{item.channel} · {item.reason || 'Delivery failed'} · Reference {item.id}</p>
				<a href="/admin/diagnostics">Check message delivery →</a>
			</article>{:else}<p>No message failed to send.</p>{/each}{/if}
</main>

<style>
	article {
		padding: 1rem;
		margin: 1rem 0;
		border: 1px solid var(--color-border);
		background: var(--color-surface);
	}
	p {
		overflow-wrap: anywhere;
	}
</style>
