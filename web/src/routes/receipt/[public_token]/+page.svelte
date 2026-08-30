<script lang="ts">
	import { onMount } from 'svelte'; import { page } from '$app/state';
	import Money from '$lib/components/Money.svelte';
	let receipt:any=$state(null); let error=$state('');
	onMount(async()=>{const response=await fetch(`/api/v1/public/receipts/${page.params.public_token}`);const result=await response.json().catch(()=>({}));if(response.ok)receipt=result.receipt;else error=result.detail??'Receipt unavailable.'});
</script>
<svelte:head><title>Receipt — Kredit</title></svelte:head>
<main class="shell prose-page"><p class="eyebrow">Payment receipt</p>{#if receipt}<h1>This money was received.</h1><dl><div><dt>Money paid</dt><dd><Money amountKobo={receipt.amount_kobo} /></dd></div><div><dt>Payment day</dt><dd>{new Date(receipt.paid_at).toLocaleString('en-NG')}</dd></div><div><dt>Receipt number</dt><dd>{receipt.reference}</dd></div></dl><p>Names and bank details are hidden on this shared receipt.</p>{:else if error}<h1>This receipt cannot be opened.</h1><p role="alert">{error}</p>{:else}<p>Opening your receipt…</p>{/if}</main>
<style>dl{display:grid;gap:.75rem;padding:1.25rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}dl div{display:flex;justify-content:space-between;gap:2rem}dt{color:var(--color-muted)}dd{font-weight:750;text-align:right;overflow-wrap:anywhere}</style>
