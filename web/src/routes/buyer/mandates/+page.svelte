<script lang="ts">
	import { onMount } from 'svelte';
	import { checkedJSON, LatestRequest, publicError, record, rows, text } from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';
	import { kobo } from '$lib/records';
	import Money from '$lib/components/Money.svelte';
	import { productLabel } from '$lib/product-language';
	let mandates: any[] = $state([]), error = $state(''), busy = $state(''), loading = $state(true), notice = $state('');
	const reads = new LatestRequest(), intents = new Map<string, MutationIntent>();
	function mandateRecord(value: unknown) {
		const row = record(value);
		for (const key of ['id', 'provider', 'status']) if (!text(row[key])) throw new Error('Incomplete bank permission');
		if (!['NOT_STARTED','PENDING','ACTIVE','PAUSED','CANCELLED','EXPIRED','FAILED'].includes(String(row.status))) throw new Error('Unknown bank permission status');
		kobo(row.amount_ceiling_kobo); return row;
	}
	async function load() {
		const read = reads.begin(); loading = true; error = ''; mandates = [];
		try { const result = await checkedJSON('/api/v1/buyer/mandates', rows('mandates', mandateRecord), {signal: read.signal}); if (read.current()) mandates = result; }
		catch (cause) { if (read.current()) error = publicError(cause, 'your bank debit permissions'); }
		finally { if (read.current()) loading = false; }
	}
	async function command(mandate: any, action: 'cancel' | 'restore') {
		if (busy || loading) return;
		busy = mandate.id; error = ''; notice = '';
		const path = `/api/v1/buyer/mandates/${encodeURIComponent(mandate.id)}/${action}`;
		try {
			if (!intents.has(path)) intents.set(path, new MutationIntent('buyer-bank-permission', path));
			await intents.get(path)!.run(action === 'cancel' ? {reason: 'Cancelled by the buyer'} : undefined, value => {
				const saved = mandateRecord(record(value).mandate);
				if (action === 'cancel' && (saved.id !== mandate.id || saved.status !== 'CANCELLED')) throw new Error('Cancellation not confirmed');
				return saved;
			});
			notice = action === 'cancel' ? 'Cancellation confirmed. New debit requests under this permission are stopped.' : 'Your bank permission update has been saved.';
			await load();
		} catch (cause) { error = cause instanceof Error ? cause.message : 'We could not confirm your bank permission change.'; }
		finally { busy = ''; }
	}
	onMount(() => { void load(); return () => reads.cancel(); });
</script>
<svelte:head><title>Bank debit — Kredit</title></svelte:head>
<main class="shell workspace"><p class="eyebrow">Bank debit</p><h1>You decide what leaves your bank.</h1><p class="lede">You can cancel this permission to stop new debit requests. A request already sent to your bank may still complete; cancelling does not remove money you already owe.</p>
{#if error}<p class="error" role="alert">{error}</p>{/if}
{#if notice}<p role="status">{notice}</p>{/if}
{#if loading}<p role="status">Opening bank debit permissions…</p>{:else if mandates.length}<section class="cards">{#each mandates as mandate}<article><div><strong>{mandate.provider}</strong><span>{productLabel(mandate.status)}</span></div><p>Permission limit: <Money amountKobo={mandate.amount_ceiling_kobo} /></p><details class="reference"><summary>Permission number</summary>{mandate.id}</details>{#if ['ACTIVE','PENDING','PAUSED'].includes(mandate.status)}<button class="danger" disabled={busy!==''} onclick={()=>command(mandate,'cancel')}>Stop bank debit</button>{:else if ['CANCELLED','EXPIRED','FAILED'].includes(mandate.status)}{#if mandate.provider==='mono-sweep'}<a href="/buyer/requests">Open a sale to give fresh bank permission →</a>{:else}<button disabled={busy!==''} onclick={()=>command(mandate,'restore')}>Allow bank debit again</button>{/if}{/if}</article>{/each}</section>{:else if !error}<section class="empty"><h2>You have not given bank debit permission yet</h2><p>It will show here after you accept your first sale.</p></section>{/if}{#if error}<button disabled={busy!=='' || loading} onclick={load}>Reload permissions</button>{/if}</main>
<style>.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(17rem,1fr));gap:1rem}.cards article,.empty{padding:1.25rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}.cards article>div{display:flex;justify-content:space-between;gap:1rem}.cards span{color:var(--color-muted)}.reference{font-size:.8rem;overflow-wrap:anywhere}.danger{background:var(--color-destructive)}.error{color:var(--color-destructive)}</style>
