<script lang="ts">
	import { onMount } from 'svelte';
	import { readLocal, setLowData } from '$lib/product-tools';
	let enabled = $state(false), message = $state('');
	onMount(()=>{enabled=readLocal('kredit:low-data',false)});
	function save(){setLowData(enabled);message=enabled?'Kredit will now use fewer effects on this phone.':'Normal display is back on this phone.'}
</script>
<svelte:head><title>Data use — Kredit</title></svelte:head>
<main class="shell workspace data-page"><a href="/app/settings">← Settings</a><p class="eyebrow">Data use</p><h1>Use less data.</h1><p class="lede">Turn this on if your network is slow or you want a simpler display. Your sales and records will not change.</p><section class="card"><label><input type="checkbox" bind:checked={enabled}/><span><strong>Use less data on this phone</strong><small>Removes movement, blur and heavy page effects.</small></span></label><button class="primary" onclick={save}>Save this setting</button>{#if message}<p role="status">{message}</p>{/if}</section><section class="card"><h2>When the network stops</h2><p>Keep this page open. Kredit will tell you when you are offline. An unfinished new sale is also kept on this phone so you can continue later.</p></section></main>
<style>.data-page{max-width:48rem}.card{display:grid;gap:1rem;margin-top:1rem;padding:1.3rem;border:1px solid var(--color-border);background:var(--color-surface)}.card label{display:flex;gap:.8rem;align-items:start}.card input{width:1.3rem;height:1.3rem}.card span,.card small{display:block}.card small{margin-top:.25rem;color:var(--color-muted)}.card button{width:max-content}</style>
