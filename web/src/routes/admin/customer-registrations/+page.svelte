<script lang="ts">
 import { onMount } from 'svelte';
 import { checkedJSON, record, rows, text, publicError } from '$lib/api/reliable';
 import { MutationIntent } from '$lib/api/mutation';
 import { localTime } from '$lib/admin-client';
 type Registration={id:string;business_id:string;created_at:string};
 let items=$state<Registration[]>([]);let loading=$state(true);let busy=$state(false);let error=$state('');let message=$state('');
 let selected=$state<Registration|null>(null);let action=$state('link');let reference=$state('');let reason=$state('');
 let intent:MutationIntent|null=null;
 async function load(){loading=true;error='';try{items=await checkedJSON('/api/v1/ops/customer-registrations',rows('registrations',value=>{const row=record(value);return{id:text(row.id),business_id:text(row.business_id),created_at:text(row.created_at)}}));}catch(cause){error=publicError(cause,'Registrations could not be loaded.')}finally{loading=false}}
 function choose(item:Registration){selected=item;reference='';reason='';action='link';message='';intent=new MutationIntent('customer-registration',`/api/v1/ops/customer-registrations/${encodeURIComponent(item.id)}`)}
 async function resolve(){if(!selected||!intent||busy)return;busy=true;error='';try{await intent.run({action,provider_reference:reference.trim(),reason:reason.trim()},body=>{if(record(body).resolved!==true)throw new Error('Resolution was not confirmed');return true});selected=null;message='Resolution saved.';await load()}catch(cause){error=publicError(cause,'Resolution is unconfirmed. Check the current record before retrying.')}finally{busy=false}}
 onMount(load);
</script>
<svelte:head><title>Customer registration recovery — Kredit</title></svelte:head>
<main class="shell workspace"><p class="eyebrow">Bank connection / Recovery</p><h1>Resolve an interrupted registration</h1><p>A saved attempt prevents duplicate provider registrations. Check the provider dashboard before choosing an action. Fresh identity confirmation is required.</p>
<a href="/admin/mono">Back to bank connection</a>
{#if message}<p role="status">{message}</p>{/if}{#if error}<p role="alert">{error}</p><button disabled={busy} onclick={load}>Check current records</button>{/if}
{#if loading}<p role="status">Loading registrations…</p>{:else if !items.length && !error}<p>No registrations need recovery.</p>{:else}<ul>{#each items as item(item.id)}<li><p>Business reference: {item.business_id}</p><p>Started {localTime(item.created_at)}</p><button disabled={busy} onclick={()=>choose(item)}>Review attempt</button></li>{/each}</ul>{/if}
{#if selected}<form onsubmit={(event)=>{event.preventDefault();void resolve()}}><fieldset disabled={busy}><legend>Resolve selected attempt</legend><label>Provider result<select bind:value={action}><option value="link">Customer exists — verify and attach its reference</option><option value="not_created">Provider confirms no customer was created — permit a new attempt</option></select></label>
{#if action==='link'}<label>Provider customer reference<input bind:value={reference} required maxlength="128" /></label><p>Kredit will fetch the provider record and compare it with the original identity. A mismatched or incomplete record cannot be attached.</p>{:else}<p>Only choose this after checking that the provider did not create a customer. It allows the customer to submit a new registration.</p>{/if}
<label>Evidence checked and reason<textarea bind:value={reason} required minlength="20" maxlength="2000"></textarea></label><p>Do not enter identity numbers, passwords or API keys in this note.</p><button type="submit">{busy?'Saving…':'Save resolution'}</button><button type="button" onclick={()=>selected=null}>Cancel</button></fieldset></form>{/if}</main>
<style>main{max-width:850px;padding-block:2rem}ul{list-style:none;padding:0}li,form{padding:1.25rem;border:1px solid var(--border,#d8e1df);border-radius:1rem;margin-top:1rem}fieldset{border:0;padding:0;display:grid;gap:1rem;min-width:0}label{display:grid;gap:.5rem}input,select,textarea{width:100%;box-sizing:border-box;padding:.75rem;font:inherit}li p{overflow-wrap:anywhere}button{padding:.75rem 1rem;cursor:pointer}textarea{min-height:7rem}</style>
