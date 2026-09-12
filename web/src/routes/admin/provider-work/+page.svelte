<script lang="ts">
 import {onMount} from 'svelte';
 import {checkedJSON,record,rows,text,publicError} from '$lib/api/reliable';
 import {timeLabel} from '$lib/records';
 type Item={kind:string;id:string;provider:string;state:string;organization_id:string;target_id:string;started_at:string;next_action:string};
 let items=$state<Item[]>([]),connections=$state<{provider:string;state:string;action:string}[]>([]),counts=$state<Record<string,unknown>>({}),error=$state(''),loading=$state(true),asOf=$state('');
 let filter=$state('all');
 const visible=$derived(items.filter(item=>filter==='all'||item.kind===filter));
 function action(item:Item){
  if(item.kind==='verification')return '/admin/verification-requests';
  if(item.kind==='authorization')return '/admin/mandate-authorizations';
  if(item.kind==='message_submission')return '/admin/message-submissions';
  if(item.kind==='notification')return '/admin/platform-settings';
  return `/admin/controls?target_type=${item.kind==='document'?'document':'collection'}&target_id=${encodeURIComponent(item.id)}&organization_id=${encodeURIComponent(item.organization_id)}`;
 }
 async function load(){loading=true;error='';try{
  const result=await checkedJSON('/api/v1/ops/provider-work',value=>{const data=record(value),work=record(data.work);return{connections:rows('connections',value=>{const r=record(value);return{provider:text(r.provider),state:text(r.state),action:text(r.action)}})(data),items:rows('items',value=>{const r=record(value);for(const key of ['kind','id','provider','state','target_id','started_at','next_action'])text(r[key]);return r as Item})(work),counts:record(work.counts),asOf:text(work.as_of)}});
  items=result.items;connections=result.connections;counts=result.counts;asOf=result.asOf;
 }catch(cause){error=publicError(cause,'provider work')}finally{loading=false}}
 onMount(load);
 const status=(state:string)=>({not_configured:'Not configured',configured_unverified:'Configured — live operation not confirmed',configuration_unavailable:'Configuration could not be read',disabled:'Disabled'}[state]??state);
</script>
<svelte:head><title>Provider work — Kredit</title></svelte:head>
<main class="shell workspace"><h1>Provider work needing attention</h1><p>Configuration, waiting requests and failed outcomes are shown separately. No recent activity does not prove that a provider is healthy.</p><button disabled={loading} onclick={load}>Refresh records</button><a href="/admin/diagnostics">System diagnostics</a>
{#if error}<p role="alert">{error}</p>{/if}{#if loading}<p role="status">Loading current records…</p>{:else if !error}
<p>Checked {timeLabel(asOf)}. Showing the oldest 100 records; counts include the full queue.</p>
<section class="connections">{#each connections as item}<article><h2>{item.provider}</h2><strong>{status(item.state)}</strong><p>{item.action}</p><a href={item.provider==='Mono'?'/admin/mono':'/admin/platform-settings'}>Open settings</a></article>{/each}</section>
<label>Work type<select bind:value={filter}><option value="all">All</option>{#each Object.entries(counts) as [kind,count]}<option value={kind}>{kind} ({String(count)})</option>{/each}</select></label>
<ul>{#each visible as item(item.kind+item.id)}<li><h2>{item.provider} · {item.kind}</h2><p>{item.state} · {timeLabel(item.started_at)}</p><p>{item.next_action}</p><p>Record <code>{item.id}</code></p><p>Affected record <code>{item.target_id}</code></p><a href={action(item)}>Review</a></li>{:else}<li>No matching work in these records.</li>{/each}</ul>{/if}</main>
<style>main{padding-block:2rem}p{line-height:1.6}h2{font-size:1.1rem}.connections{display:grid;grid-template-columns:repeat(auto-fit,minmax(15rem,1fr));gap:1rem;margin:1.5rem 0}article,li{border:1px solid var(--color-border);border-radius:.75rem;padding:1rem}ul{list-style:none;padding:0;display:grid;gap:1rem}code{overflow-wrap:anywhere}label{display:grid;gap:.5rem}select,button{padding:.75rem;font:inherit}main>a{margin-left:1rem}</style>
