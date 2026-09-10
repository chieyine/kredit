<script lang="ts">
 import { onMount } from 'svelte';
 import { checkedJSON, LatestRequest, record, rows, text, publicError } from '$lib/api/reliable';
 import { MutationIntent, MutationError } from '$lib/api/mutation';
 import { productLabel } from '$lib/product-language';
 type Member = { user_id: string; role: string; status: string };
 let orgID=$state(''), members:Member[]=$state([]), target=$state(''), channel=$state('email'), role=$state('sales'), message=$state(''), error=$state(''), busy=$state(''), loading=$state(true);
 let roleDraft:Record<string,string>=$state({});
 const reads = new LatestRequest();
 const intents = new Map<string, MutationIntent>();
 function decodeMember(value: unknown): Member {
  const item=record(value); const user_id=text(item.user_id), role=text(item.role), status=text(item.status);
  if (!user_id || !['owner','administrator','sales','finance','collections','viewer'].includes(role) || !['invited','active','suspended','removed'].includes(status)) throw new Error('Incomplete membership');
  return {user_id,role,status};
 }
 function intent(url: string) { let value=intents.get(url); if (!value) { value=new MutationIntent('team',url); intents.set(url,value); } return value; }
 async function load() {
  const request=reads.begin(); loading=true; error='';
  try {
   const organizations=await checkedJSON('/api/v1/organizations',rows('organizations',value=>{const item=record(value);const id=text(item.id);if(!id)throw new Error('Missing business');return id;}),{signal:request.signal});
   if (!request.current()) return;
   const selected=new URLSearchParams(location.search).get('organization');
   orgID=selected?(organizations.includes(selected)?selected:''):organizations[0]??'';
   if(selected&&!orgID)throw new Error('Business access could not be verified');
   const next=orgID?await checkedJSON(`/api/v1/organizations/${encodeURIComponent(orgID)}/members`,rows('members',decodeMember),{signal:request.signal}):[];
   if (!request.current()) return;
   members=next; roleDraft=Object.fromEntries(next.map(member=>[member.user_id,member.role]));
  } catch(cause) { if(request.current()){members=[]; error=publicError(cause,'your staff list');} }
  finally { if(request.current()) loading=false; }
 }
 async function invite(event:SubmitEvent) {
  event.preventDefault(); if(busy||loading||!orgID||!target.trim())return;
  busy='invite';error='';message='';
  try {
   await intent(`/api/v1/organizations/${encodeURIComponent(orgID)}/members`).run({target:target.trim(),channel,role},value=>decodeMember(record(value).membership));
   target='';message='Invitation created. Ask your worker to sign in with the email or phone number you entered.';await load();
  } catch(cause){error=cause instanceof MutationError?cause.message:publicError(cause,'this invitation');}
  finally {busy='';}
 }
 async function change(member:Member,body:{role?:string;status?:string},label:string) {
  if(busy||loading||!orgID)return;busy=member.user_id;error='';message='';
  try {
   await intent(`/api/v1/organizations/${encodeURIComponent(orgID)}/members/${encodeURIComponent(member.user_id)}`).run(body,value=>decodeMember(record(value).membership),'PATCH');
   message=label;await load();
  } catch(cause){error=cause instanceof MutationError?cause.message:publicError(cause,'this access change');}
  finally {busy='';}
 }
 onMount(()=>{void load();return()=>reads.cancel();});
</script>
<svelte:head><title>Team — Kredit</title></svelte:head>
<main class="shell workspace team"><p class="eyebrow">Your staff</p><h1>Decide what each person can do.</h1><p class="lede">Invite a worker, change what they can do, or stop their access. Nobody can remove the owner.</p>
	{#if message}<p class="notice" role="status">{message}</p>{/if}{#if error}<p class="error" role="alert">{error} <button onclick={load}>Refresh team</button></p>{/if}
	{#if !loading&&!error&&!orgID}<p>Create your business account before inviting staff. <a href="/app/onboarding">Set up your business →</a></p>{/if}<section class="card"><h2>Invite a worker</h2><form onsubmit={invite}><fieldset disabled={Boolean(busy)||loading||!orgID}><label>Their email or phone number<input bind:value={target} type={channel==='email'?'email':'tel'} required /></label><label>Send the invite by<select bind:value={channel}><option value="email">Email</option><option value="phone">Phone</option></select></label><label>What will they do here?<select bind:value={role}><option value="sales">Add sales</option><option value="finance">Handle money</option><option value="collections">Follow late payments</option><option value="administrator">Manage the account</option><option value="viewer">Look only</option></select></label><button class="primary" disabled={busy==='invite'||!target}>{busy==='invite'?'Sending…':'Send invite'}</button></fieldset></form></section>
	<section class="card"><h2>People who can enter your account</h2>{#if loading}<p role="status">Opening your staff list…</p>{:else if members.length}<div class="member-list">{#each members as member}<article><div><strong>{productLabel(member.role)}</strong><span class="status">{productLabel(member.status)}</span><small>Account {member.user_id}</small></div>{#if member.role!=='owner'&&member.status!=='removed'}<div class="controls"><label>What can they do?<span class="sr-only"> for {member.user_id}</span><select bind:value={roleDraft[member.user_id]} disabled={member.status!=='active'||Boolean(busy)||loading}><option value="sales">Add sales</option><option value="finance">Handle money</option><option value="collections">Follow late payments</option><option value="administrator">Manage the account</option><option value="viewer">Look only</option></select></label><button disabled={member.status!=='active'||roleDraft[member.user_id]===member.role||Boolean(busy)||loading} onclick={()=>change(member,{role:roleDraft[member.user_id]},'Saved. What this person can do has changed.')}>Save</button>{#if member.status==='active'}<button class="danger" disabled={Boolean(busy)||loading} onclick={()=>change(member,{status:'suspended'},'Done. This person can no longer enter your account.')}>Stop their access</button>{:else if member.status==='suspended'}<button disabled={Boolean(busy)||loading} onclick={()=>change(member,{status:'active'},'Done. This person can enter your account again.')}>Let them back in</button>{/if}<button class="danger" disabled={Boolean(busy)||loading} onclick={()=>change(member,{status:'removed'},'Done. This person has been removed from your business.')}>Remove</button></div>{/if}</article>{/each}</div>{:else}<p>You have not added any worker yet.</p>{/if}</section>
</main>
<style>.team h1{font-size:clamp(2.4rem,6vw,4.5rem);line-height:1}.card{margin:1.5rem 0;padding:1.3rem}.card fieldset{border:0;margin:0;padding:0;min-width:0;display:grid;grid-template-columns:2fr 1fr 1fr auto;align-items:end;gap:.7rem}label{display:grid;gap:.3rem;font-weight:700}input,select,button{box-sizing:border-box;padding:.7rem;border:1px solid var(--color-border);border-radius:.65rem;background:var(--color-surface);color:inherit;font:inherit}.member-list{display:grid;gap:.75rem}.member-list article{display:grid;grid-template-columns:minmax(16rem,1fr) 2fr;gap:1rem;padding:1rem;border-top:1px solid var(--color-border)}.member-list article>div:first-child{display:grid;grid-template-columns:auto auto 1fr;align-items:center;gap:.55rem}.member-list strong{text-transform:capitalize}.member-list small{grid-column:1/-1;overflow-wrap:anywhere}.controls{display:flex;flex-wrap:wrap;align-items:end;gap:.5rem}.controls select{min-width:10rem}.danger{color:var(--color-destructive);border-color:var(--color-destructive)}.notice{padding:1rem;border-left:4px solid var(--color-positive);background:var(--color-surface-muted)}@media(max-width:820px){.card fieldset,.member-list article{grid-template-columns:1fr}.controls{align-items:stretch;flex-direction:column}.controls button,.controls label{width:100%}}</style>
