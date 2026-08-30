<script lang="ts">
	import { onMount } from 'svelte';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';
	import Money from '$lib/components/Money.svelte';
	import { productLabel } from '$lib/product-language';
	let mandates: any[] = $state([]); let error = $state(''); let busy = $state('');
	async function load(){ const response=await fetch('/api/v1/buyer/mandates',{credentials:'include'}); if(response.ok) mandates=(await response.json()).mandates??[]; else error='We could not open your bank debit settings.'; }
	async function command(mandate:any,action:'cancel'|'restore'){
		busy=mandate.id; error='';
		const response=await fetch(`/api/v1/buyer/mandates/${mandate.id}/${action}`,{method:'POST',credentials:'include',headers:{...(action==='cancel'?{'Content-Type':'application/json'}:{}),'Idempotency-Key':idempotencyKey(),...csrfHeaders()},body:action==='cancel'?JSON.stringify({reason:'Cancelled by the buyer'}):undefined});
		const result=await response.json().catch(()=>({})); busy=''; if(!response.ok){error=result.detail??'We could not change your bank debit permission.';return} await load();
	}
	onMount(load);
</script>
<svelte:head><title>Bank debit — Kredit</title></svelte:head>
<main class="shell workspace"><p class="eyebrow">Bank debit</p><h1>You control money from your bank.</h1><p class="lede">You can stop Kredit from taking money. A new sale will need your permission again.</p>
{#if error}<p class="error" role="alert">{error}</p>{/if}
{#if mandates.length}<section class="cards">{#each mandates as mandate}<article><div><strong>{mandate.provider}</strong><span>{productLabel(mandate.status)}</span></div><p>Most Kredit can debit: <Money amountKobo={mandate.amount_ceiling_kobo} /></p><details class="reference"><summary>Permission number</summary>{mandate.id}</details>{#if mandate.status==='ACTIVE'}<button class="danger" disabled={busy===mandate.id} onclick={()=>command(mandate,'cancel')}>Stop bank debit</button>{:else if ['CANCELLED','EXPIRED','FAILED'].includes(mandate.status)}<button disabled={busy===mandate.id} onclick={()=>command(mandate,'restore')}>Allow bank debit again</button>{/if}</article>{/each}</section>{:else}<section class="empty"><h2>No bank debit yet</h2><p>Your bank debit permission will show here after you accept a sale.</p></section>{/if}</main>
<style>.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(17rem,1fr));gap:1rem}.cards article,.empty{padding:1.25rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}.cards article>div{display:flex;justify-content:space-between;gap:1rem}.cards span{color:var(--color-muted)}.reference{font-size:.8rem;overflow-wrap:anywhere}.danger{background:var(--color-destructive)}.error{color:var(--color-destructive)}</style>
