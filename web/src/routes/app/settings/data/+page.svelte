<script lang="ts">
	import { onMount } from 'svelte';
	import { readLocal, setLowData } from '$lib/product-tools';
	let enabled = $state(false), message = $state('');
	onMount(()=>{enabled=readLocal('kredit:low-data',false)});
	function save(){setLowData(enabled);message=enabled?'Done. Kredit will now use less data on this phone.':'Done. The normal display is back on this phone.'}
</script>
<svelte:head><title>Data use — Kredit</title></svelte:head>
<main class="shell workspace data-page"><a href="/app/settings">← Settings</a><p class="eyebrow">Data use</p><h1>Use less data.</h1><p class="lede">Turn this on if your network is slow, or data is expensive. Your sales and records stay exactly the same. Only the display gets lighter.</p><section class="card"><label><input type="checkbox" bind:checked={enabled}/><span><strong>Use less data on this phone</strong><small>Removes movement, blur and anything heavy on the page.</small></span></label><button class="primary" onclick={save}>Save this setting</button>{#if message}<p role="status">{message}</p>{/if}</section><section class="card"><h2>When your network cuts</h2><p>Keep the page open. Kredit will tell you the moment you go offline. Anything you were typing into a new sale is kept on this phone, so you can carry on later.</p></section></main>
<style>.data-page{max-width:48rem}.card{display:grid;gap:1rem;margin-top:1rem;padding:1.3rem;border:1px solid var(--color-border);background:var(--color-surface)}.card label{display:flex;gap:.8rem;align-items:start}.card input{width:1.3rem;height:1.3rem}.card span,.card small{display:block}.card small{margin-top:.25rem;color:var(--color-muted)}.card button{width:max-content}</style>
