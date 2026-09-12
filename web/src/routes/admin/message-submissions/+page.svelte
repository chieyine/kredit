<script lang="ts">
 import { onMount } from 'svelte';
 import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
 import { checkedJSON, record, rows, text, publicError } from '$lib/api/reliable';
 import { MutationIntent } from '$lib/api/mutation';
 import { localTime } from '$lib/admin-client';
 type Submission = { id:string; channel:string; notification_id:string; created_at:string };
 let items=$state<Submission[]>([]), loading=$state(true), busy=$state(false), error=$state(''), message=$state('');
 let selected=$state<Submission|null>(null), action=$state('record_reference'), reference=$state(''), reason=$state(''), reviewed=$state(false);
 let intent:MutationIntent|null=null;
 async function load(){loading=true;error='';try{items=await checkedJSON('/api/v1/ops/message-submissions',rows('submissions',value=>{const row=record(value);return{id:text(row.id),channel:text(row.channel),notification_id:text(row.notification_id),created_at:text(row.created_at)}}));if(selected&&!items.some(item=>item.id===selected?.id))selected=null;}catch(cause){error=publicError(cause,'Messages could not be loaded.')}finally{loading=false}}
 function choose(item:Submission){selected=item;action='record_reference';reference='';reason='';reviewed=false;message='';intent=new MutationIntent('message-recovery',`/api/v1/ops/message-submissions/${encodeURIComponent(item.id)}/resolve`)}
 async function resolve(){if(!selected||!intent||busy||!reviewed)return;busy=true;error='';try{await intent.run({action,provider_reference:action==='record_reference'?reference.trim():'',reason:reason.trim()},value=>{const row=record(value);if(row.state!==(action==='record_reference'?'ACCEPTED':'REJECTED')||row.delivery!=='not_confirmed')throw new Error('Resolution was not confirmed');return true});selected=null;message='Resolution saved. No message was resent.';await load()}catch(cause){error=publicError(cause,'Resolution is unconfirmed. Refresh the list before trying again.')}finally{busy=false}}
 onMount(load);
</script>
<svelte:head><title>Message recovery — Kredit</title></svelte:head>
<main class="shell workspace">
 <p class="eyebrow">Provider operations</p><h1>Resolve an uncertain message</h1>
 <p>These sends did not return a confirmed result. Check the original provider account before resolving one. Kredit keeps them on hold to prevent duplicates.</p>
 <a href="/admin/provider-work">Back to provider work</a>
 <VerifyIdentity />
 {#if message}<p role="status">{message}</p>{/if}
 {#if error}<p role="alert">{error}</p>{/if}
 <button disabled={busy||loading} onclick={load}>Refresh messages</button>
 {#if loading}<p role="status">Loading messages…</p>{:else if !error&&!items.length}<p>No messages need recovery.</p>{:else if !error}
 <ul>{#each items as item(item.id)}<li><p>{item.channel} · Started {localTime(item.created_at)}</p><p>Submission: {item.id}</p>{#if item.notification_id}<p>Notification: {item.notification_id}</p>{:else}<p>Account verification or chat reply</p>{/if}<button disabled={busy} onclick={()=>choose(item)}>Review message</button></li>{/each}</ul>{/if}
 {#if selected}<form onsubmit={(event)=>{event.preventDefault();void resolve()}}><fieldset disabled={busy}>
 <legend>Resolve selected message</legend>
 <label>Provider result<select bind:value={action} onchange={()=>reviewed=false}><option value="record_reference">Provider accepted it — record the message ID</option><option value="close_without_resend">Close this attempt without resending</option></select></label>
 {#if action==='record_reference'}<label>Original provider message ID<input bind:value={reference} oninput={()=>reviewed=false} required maxlength="512" /></label><p>This records acceptance only. Delivery still needs an authenticated provider confirmation. It does not start a payment deadline.</p>{:else}<p>This stops automatic recovery of this attempt. Its delivery remains unknown, and no replacement is sent.</p>{/if}
 <label>Evidence checked and reason<textarea bind:value={reason} oninput={()=>reviewed=false} required minlength="8" maxlength="2000"></textarea></label>
 <p>Do not enter passwords, API keys, identity numbers or message contents.</p>
 <label class="confirm"><input type="checkbox" bind:checked={reviewed} required />I checked the original provider account and reviewed this action.</label>
 <button type="submit" disabled={!reviewed}>{busy?'Saving…':'Save resolution'}</button><button type="button" onclick={()=>selected=null}>Cancel</button>
 </fieldset></form>{/if}
</main>
<style>main{max-width:850px;padding-block:2rem}ul{list-style:none;padding:0}li,form{padding:1.25rem;border:1px solid var(--border,#d8e1df);border-radius:1rem;margin-top:1rem}fieldset{border:0;padding:0;display:grid;gap:1rem;min-width:0}label{display:grid;gap:.5rem}input,select,textarea{width:100%;box-sizing:border-box;padding:.75rem;font:inherit}li p{overflow-wrap:anywhere}button{padding:.75rem 1rem;cursor:pointer}textarea{min-height:7rem}.confirm{display:flex;align-items:start}.confirm input{width:auto}</style>
