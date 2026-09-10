<script lang="ts">
	import { checkedJSON, publicError, record, text, rows } from '$lib/api/reliable';
	let query=$state(''), loading=$state(false), error=$state(''), searched=$state(false), results:any[]=$state([]);
	async function search(event: SubmitEvent) {
        event.preventDefault(); if (loading) return;
        error = ''; searched = true; results = [];
        if (query.trim().length < 4) { error = 'Enter at least four characters from the exact reference.'; return; }
        loading = true;
        try { results = await checkedJSON(`/api/v1/ops/search?q=${encodeURIComponent(query.trim())}`, rows('results', value => { const item=record(value); for(const key of ['id', 'type', 'reference', 'state']) text(item[key]); return item; })); }
        catch (cause) { error = publicError(cause, 'this reference'); }
        finally { loading = false; }
    }
</script>
<svelte:head><title>Operations search — Kredit</title></svelte:head>
<main class="shell workspace"><p class="eyebrow">Operations / Search</p><h1>Find an exact reference.</h1><p class="lede">Search agreements, payments, collections, cases, disputes, and uploaded documents. Every lookup is audited.</p><form onsubmit={search}><label for="reference">Reference</label><div><input id="reference" bind:value={query} autocomplete="off" minlength="4" maxlength="128" required /><button class="primary" disabled={loading}>{loading?'Searching…':'Search'}</button></div></form>{#if error}<p class="error" role="alert">{error}</p>{:else if loading}<p role="status">Checking the reference…</p>{:else if searched&&!results.length}<p>No matching reference was found.</p>{:else if results.length}<section>{#each results as result}<article><div><strong>{result.type.replaceAll('_',' ')}</strong><span>{result.state}</span></div><code>{result.reference}</code>{#if result.type==='document'}<a href={`/admin/controls?target_type=document&target_id=${encodeURIComponent(result.id)}`}>Review scan recovery →</a>{:else if result.type==='support_case'}<a href={`/admin/cases/${encodeURIComponent(result.id)}`}>Review case →</a>{:else if result.type==='dispute'}<a href={`/admin/disputes/${encodeURIComponent(result.id)}`}>Review dispute →</a>{/if}</article>{/each}</section>{/if}</main>
<style>form{max-width:42rem;margin:2rem 0}label{display:block;font-weight:750;margin-bottom:.4rem}form div{display:flex;gap:.6rem}input{flex:1;padding:.8rem;border:1px solid var(--color-border);border-radius:.7rem;font:inherit}section{display:grid;gap:.8rem}article{padding:1rem;border:1px solid var(--color-border);border-radius:.8rem;background:var(--color-surface)}article div{display:flex;justify-content:space-between;text-transform:capitalize}code{display:block;margin:.6rem 0;color:var(--color-muted);overflow-wrap:anywhere}</style>
