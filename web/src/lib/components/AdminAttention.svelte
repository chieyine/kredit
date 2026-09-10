<script lang="ts">
 import {onMount} from 'svelte';import {adminGet} from '$lib/admin-client';
 let items:any[]=$state([]),error=$state('');
 async function load(){error='';items=[];try{const b=await adminGet('/api/v1/ops/attention');if(!Array.isArray(b.items))throw new Error('Attention list could not be verified.');items=b.items}catch(e){error=e instanceof Error?e.message:'Attention list unavailable'}};onMount(load);
</script>
<section aria-label="Priority work"><h2>Work needing attention</h2>{#if error}<p role="alert">{error} <button onclick={load}>Try again</button></p>{/if}<div>{#each items as item}<a href={item.href}><strong>{item.count}</strong><b>{item.label}</b><span>{item.action} →</span></a>{/each}</div></section>
<style>section{margin:2rem 0}section>div{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:1rem}a{display:grid;gap:.6rem;border:1px solid #bcb7ac;background:#fffdf6;padding:1rem;color:#17181b;text-decoration:none}strong{font:2rem var(--font-serif)}span{font-size:.85rem}</style>
