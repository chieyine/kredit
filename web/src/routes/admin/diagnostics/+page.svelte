<script lang="ts">
    import { onMount } from 'svelte';
    import { checkedJSON, publicError, record, rows } from '$lib/api/reliable';
    let data: any = null, error = '';
    async function load() {
        data = null; error = '';
        try { data = await checkedJSON('/api/v1/ops/diagnostics?window_minutes=60', value => {
            const diagnostics = record(record(value).diagnostics);
            record(diagnostics.integrity); rows('queues', record)(diagnostics); rows('provider', record)(diagnostics);
            return diagnostics;
        }); } catch (cause) { error = publicError(cause, 'system diagnostics'); }
    }
    onMount(load);
</script>
<svelte:head><title>Operations diagnostics — Kredit</title></svelte:head><main class="shell workspace"><p class="eyebrow">Operations / Diagnostics</p><h1>One-hour operational health.</h1><p>Provider latency signals, webhook lag, queue age, reconciliation, drift, dead letters, notifications, scanning, mandates, and settlements—without raw payloads or unredacted correlation identifiers.</p>{#if error}<section role="alert"><p class="error">{error}</p><button type="button" onclick={load}>Try again</button></section>{:else if !data}<p>Loading diagnostics…</p>{:else}<p>Correlation: <code>{data.correlation_id}</code></p><h2>Integrity signals</h2><section>{#each Object.entries(data.integrity) as [name,value]}<article><strong>{value}</strong><span>{name.replaceAll('_',' ')}</span></article>{/each}</section><h2>Queues</h2>{#each data.queues as queue}<p><strong>{queue.queue}</strong> · {queue.count} waiting · oldest {queue.oldest_seconds}s</p>{:else}<p>No queue backlog.</p>{/each}<h2>Providers</h2>{#each data.provider as item}<p><strong>{item.provider}</strong> · {item.total} events · {item.errors} errors · oldest unprocessed {item.oldest_unprocessed_seconds}s</p>{:else}<p>No provider activity in this window.</p>{/each}{/if}</main><style>section{display:grid;grid-template-columns:repeat(auto-fit,minmax(10rem,1fr));gap:1rem}article{padding:1rem;border:1px solid var(--color-border);border-radius:1rem}article strong,article span{display:block}</style>
