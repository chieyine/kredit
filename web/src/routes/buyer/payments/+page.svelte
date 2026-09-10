<script lang="ts">
    import { checkedJSON, LatestRequest, publicError, record, rows, text } from '$lib/api/reliable';
	import { onMount } from 'svelte';
	import { kobo } from '$lib/records';
	import { readableDate, readableDateTime } from '$lib/datetime';
	import Money from '$lib/components/Money.svelte';
	import { productLabel } from '$lib/product-language';
	let claims:any[]=$state([]),loading=$state(true),error=$state('');
    const reads = new LatestRequest();
    function claimRecord(value: unknown) {
        const claim = record(value);
        for (const key of ['id','state','transfer_reference','created_at','paid_at','hold_expires_at']) text(claim[key]);
        if (!claim.id || !['pending','confirmed','rejected','expired'].includes(String(claim.state))) throw new Error('Incomplete transfer report');
        for (const key of ['created_at','paid_at','hold_expires_at']) if (!Number.isFinite(Date.parse(String(claim[key])))) throw new Error('Incomplete transfer date');
        kobo(claim.amount_kobo); return claim;
    }
    async function load() {
        const read = reads.begin(); loading = true; error = '';
        try {
            const result = await checkedJSON('/api/v1/buyer/payment-claims', rows('payment_claims', claimRecord), {signal: read.signal});
            if (read.current()) claims = result.sort((a,b) => Date.parse(String(b.created_at))-Date.parse(String(a.created_at)));
        } catch (cause) { if (read.current()) error = publicError(cause, 'your reported transfers'); }
        finally { if (read.current()) loading = false; }
    }
    onMount(() => { void load(); return () => reads.cancel(); });
</script>
<svelte:head><title>Transfers I reported — Kredit</title></svelte:head>
<main class="shell workspace reported"><p class="eyebrow">Transfers I reported</p><h1>What happened after you told a seller you paid.</h1><p class="lede">A seller has to see the money in their bank before your balance can drop. That is what protects both of you.</p>{#if loading}<p role="status">Opening your transfers…</p>{:else if error}<section role="alert"><p class="error">{error}</p><button type="button" onclick={load}>Try again</button></section>{:else if claims.length}<section>{#each claims as claim}<article><div><strong><Money amountKobo={claim.amount_kobo}/></strong><span>{productLabel(claim.state)}</span></div><dl><div><dt>Transfer number</dt><dd>{claim.transfer_reference}</dd></div><div><dt>Payment day</dt><dd>{readableDate(claim.paid_at)}</dd></div>{#if claim.review_reason}<div><dt>What the seller said</dt><dd>{claim.review_reason}</dd></div>{/if}</dl>{#if claim.state==='pending'}<p>Awaiting seller review. Temporary collection-pause deadline: {readableDateTime(claim.hold_expires_at)}. After this deadline, collection may resume.</p>{/if}</article>{/each}</section>{:else}<section class="empty-state"><h2>You have not reported any transfer</h2><p>When you tell a seller you have sent money, you can follow their answer here.</p><a href="/buyer/obligations">See what I owe →</a></section>{/if}</main>
<style>.reported{max-width:58rem}.reported>section{display:grid;gap:.8rem;margin-top:2rem}.reported article{padding:1.2rem;border:1px solid var(--color-border);background:var(--color-surface)}article>div{display:flex;justify-content:space-between;gap:1rem}article>div>strong{font-size:1.5rem}dl{display:grid;gap:.5rem}dl div{display:flex;justify-content:space-between;gap:1rem}dd{margin:0;text-align:right;font-weight:700}.error{color:var(--color-destructive)}</style>
