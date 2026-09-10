<script lang="ts">
	import { onMount } from 'svelte';
	import { checkedJSON, record, rows, text, publicError, LatestRequest } from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';
	import ShareActions from '$lib/components/ShareActions.svelte';
	let organizations:any[]=$state([]),organizationID=$state(''),targetType=$state('email'),target=$state(''),legalName=$state(''),tradingName=$state(''),businessType=$state('unregistered_business'),address=$state(''),industry=$state(''),busy=$state(false),error=$state(''),invitationURL=$state(''),copied=$state(false);
	let loading=$state(true);
	const reads=new LatestRequest(), intents=new Map<string,MutationIntent>();
	async function load(){
		const request=reads.begin();loading=true;error='';
		try{const items=await checkedJSON('/api/v1/organizations',rows('organizations',value=>{const row=record(value);if(!text(row.id))throw new Error('Missing business');text(row.legal_name);return row;}),{signal:request.signal});if(request.current()){organizations=items;organizationID=String(items[0]?.id??'');}}
		catch(cause){if(request.current())error=publicError(cause,'your businesses');}
		finally{if(request.current())loading=false;}
	}
	onMount(()=>{void load();return()=>reads.cancel();});
	async function submit(event:SubmitEvent){
		event.preventDefault();if(busy||loading||!organizationID)return;busy=true;error='';
		try{
			const path=`/api/v1/organizations/${encodeURIComponent(organizationID)}/buyer-invitations`;
			if(!intents.has(path))intents.set(path,new MutationIntent('create-customer-invitation',path));
			invitationURL=await intents.get(path)!.run({target,target_type:targetType,legal_name:legalName,trading_name:tradingName,business_type:businessType,business_address:address,industry},value=>{
				const link=text(record(value).invitation_url),url=new URL(link,location.origin);
				if(!['https:','http:'].includes(url.protocol)||!url.pathname.startsWith('/buyer-invitations/')||url.username||url.password)throw new Error('Invalid customer link');
				return url.href;
			});
		}catch(cause){error=cause instanceof Error?cause.message:'We could not confirm the customer link.';}
		finally{busy=false;}
	}
	async function copy(){try{await navigator.clipboard.writeText(invitationURL);copied=true;setTimeout(()=>copied=false,1800);}catch{error='Copying is unavailable. Select the customer link and copy it.';}}

</script>
<svelte:head><title>Add a customer — Kredit</title></svelte:head>
<main class="shell workspace invite"><a href="/app/customers">← Customers</a><p class="eyebrow">Add a customer</p><h1>Who will take the goods?</h1><p class="lede">Add their details, then send them a private link. They confirm the details themselves before you can sell to them.</p>
{#if error}<p class="error" role="alert">{error}</p>{/if}
{#if invitationURL}<section class="card result" aria-live="polite"><h2>Their link is ready</h2><p>Send this private link to your customer. It works once, and it stops working after a while.</p><input value={invitationURL} readonly aria-label="Customer link"/><ShareActions title="Your Kredit customer link" text={`Hello ${legalName}, please open this private link to check your details for our credit sales.`} url={invitationURL}/><button onclick={copy}>{copied?'Copied':'Copy only the link'}</button><a href="/app/customers">Back to customers</a></section>{:else}
<form class="card" onsubmit={submit}>{#if loading}<p role="status">Opening your businesses…</p>{:else if !organizations.length}<p>No business is available. <a href="/app/onboarding">Set up your business</a> or <button type="button" onclick={load}>try again</button>.</p>{/if}<fieldset disabled={busy||loading}><div class="grid"><label>Your business<select bind:value={organizationID} required>{#each organizations as organization}<option value={organization.id}>{organization.trading_name||organization.legal_name}</option>{/each}</select></label><label>Send the link by<select bind:value={targetType}><option value="email">Email</option><option value="phone">Phone</option></select></label><label>Customer's {targetType==='email'?'email':'phone number'}<input bind:value={target} type={targetType==='email'?'email':'tel'} required autocomplete={targetType==='email'?'email':'tel'}/></label><label>Their name, or their business name<input bind:value={legalName} required autocomplete="name"/></label><label>The name people know them by <small>if different</small><input bind:value={tradingName} autocomplete="organization"/></label><label>Business type<select bind:value={businessType}><option value="unregistered_business">Not registered yet</option><option value="registered_business">Business name registered with CAC</option><option value="sole_proprietor">One-person business</option><option value="limited_company">Limited company</option><option value="partnership">Partnership</option></select></label><label>What do they sell?<input bind:value={industry} required/></label><label class="wide">Where is their business?<textarea bind:value={address} rows="3" required autocomplete="street-address"></textarea></label></div></fieldset><button class="primary" disabled={busy||!organizationID}>{busy?'Making the link…':'Make customer link'}</button></form>{/if}</main>
<style>fieldset{border:0;padding:0;margin:0;min-width:0}.invite{max-width:52rem}.invite h1{font-size:clamp(2.7rem,6vw,4.5rem);line-height:1;margin:.5rem 0}.card{margin-top:2rem;padding:2rem}.grid{display:grid;grid-template-columns:1fr 1fr;gap:1rem;margin-bottom:1.3rem}.grid label{display:grid;gap:.4rem;font-weight:700}.grid input,.grid select,.grid textarea,.result input{box-sizing:border-box;width:100%;padding:.75rem;border:1px solid var(--color-border);border-radius:.7rem;font:inherit}.wide{grid-column:span 2}.result{display:grid;gap:1rem}@media(max-width:680px){.grid{grid-template-columns:1fr}.wide{grid-column:auto}.card{padding:1.2rem}}</style>
