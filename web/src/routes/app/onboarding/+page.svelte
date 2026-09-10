<script lang="ts">
	import { onMount } from 'svelte';
	import { checkedJSON, LatestRequest, record, rows, text, publicError } from '$lib/api/reliable';
 import { MutationIntent, MutationError } from '$lib/api/mutation';
 import { settingsProfile } from '$lib/api/onboarding-settings';
	import { productLabel } from '$lib/product-language';
	let orgID='', profile:any={}, readiness:any={requirements:[],missing:[]}, permissions:any={}, terms='',privacy='', loading=true,busy=false,message='';
	let repName='',repTitle='',contact='',channel='phone',challengeID='',code='';
 let loadError='';
 const reads=new LatestRequest(); const intents=new Map<string,MutationIntent>();
 function summary(value:unknown) {
  const data=record(value);
  if(typeof data.ready!=='boolean'||typeof data.state!=='string'||!Array.isArray(data.missing)||!Array.isArray(data.requirements))throw new Error('Incomplete setup');
  const requirement=(value:unknown)=>{const row=record(value);if(!text(row.code)||!text(row.label))throw new Error('Incomplete step');const path=text(row.manage_path);if(typeof row.complete!=='boolean'||!path.startsWith('/app/')||/[\\\x00-\x1f]/.test(path))throw new Error('Incomplete step');return row;};
  const all=data.requirements.map(requirement),missing=data.missing.map(requirement);
  if(new Set(all.map(row=>row.code)).size!==all.length||missing.length!==all.filter(row=>!row.complete).length||missing.some(row=>row.complete||!all.some(item=>item.code===row.code&&!item.complete)))throw new Error('Inconsistent setup progress');
  if(data.ready!==(missing.length===0))throw new Error('Inconsistent setup readiness');
  return {...data,requirements:all,missing};
 }

 async function load(){
  const request=reads.begin();loading=true;loadError='';
  try{
   const orgs=await checkedJSON('/api/v1/organizations',rows('organizations',value=>{const id=text(record(value).id);if(!id)throw new Error('Missing business');return id;}),{signal:request.signal});
   if(!request.current())return;const selected=new URLSearchParams(location.search).get('organization');orgID=selected?(orgs.includes(selected)?selected:''):orgs[0]??'';if(!orgID){if(selected)throw new Error('Business access could not be verified');return;}
   const d=await checkedJSON(`/api/v1/organizations/${encodeURIComponent(orgID)}/onboarding`,value=>{const d=record(value);const permissions=record(d.permissions);for(const value of Object.values(permissions))if(typeof value!=='boolean')throw new Error('Incomplete permissions');return {profile:settingsProfile(d.profile),readiness:summary(d.readiness),permissions,terms:text(d.current_terms_version),privacy:text(d.current_privacy_version)};},{signal:request.signal});
   if(!request.current())return;
   profile=d.profile;readiness=d.readiness;permissions=d.permissions;terms=d.terms;privacy=d.privacy;repName=String(profile.authorized_representative_name??'');repTitle=String(profile.authorized_representative_title??'');
  }catch(cause){if(request.current())loadError=publicError(cause,'your setup');}
  finally{if(request.current())loading=false;}
 }
 async function mutate(path:string,method:string,body:unknown){
  if(busy||loading||loadError||!orgID)return null;busy=true;message='';
  try{
   const url=`/api/v1/organizations/${encodeURIComponent(orgID)}/onboarding/${path}`;
   let intent=intents.get(url);if(!intent){intent=new MutationIntent('onboarding',url);intents.set(url,intent);}
   const d=await intent.run(body,value=>{
    const d=record(value);
    if(path==='contacts/challenges'){if(!text(d.challenge_id)||!Number.isFinite(Date.parse(text(d.expires_at))))throw new Error('Incomplete code');if(d.development_code!==undefined)text(d.development_code);}
    else {d.profile=settingsProfile(d.profile);d.readiness=summary(d.readiness);}
    return d;
   },method);
   if(d.profile){profile=d.profile;readiness=d.readiness;}message='Saved. Your setup has been updated.';return d;
  }catch(cause){message=cause instanceof MutationError?cause.message:publicError(cause,'this setup step');return null;}
  finally{busy=false;}
 }
	async function saveRepresentative(){await mutate('representative','PATCH',{expected_version:profile.version,name:repName,title:repTitle})}
	async function requestContact(){const d=await mutate('contacts/challenges','POST',{identifier:contact,channel});if(d){challengeID=String(d.challenge_id);code=String(d.development_code??'');message='Code sent. Enter it below.'}}
	async function verifyContact(){const d=await mutate('contacts/verify','POST',{identifier:contact,channel,challenge_id:challengeID,code});if(d){challengeID='';code='';contact=''}}
	async function submitKYB(){await mutate('kyb','POST',{expected_version:profile.version})}
	async function refreshKYB(){await mutate('kyb/reconcile','POST',{})}
	async function acceptConsents(){await mutate('consents','POST',{expected_version:profile.version,terms_version:terms,privacy_version:privacy})}
	function simpleStep(code:string,label:string){return ({business_identity:'Who runs this business',email_verified:'Confirm your email',phone_verified:'Confirm your phone number',kyb_approved:'Check your business or your ID',settlement_verified:'Where we send your money',billing_configured:'How you pay Kredit fees',credit_policy:'Your normal sale settings',current_consents:'Agree to the terms',owner_mfa:'Extra sign-in safety for you',finance_mfa:'Extra safety for staff who touch money'} as Record<string,string>)[code]??label}
	onMount(()=>{void load();return()=>reads.cancel();});
</script>
<svelte:head><title>Set up your account — Kredit</title></svelte:head>
<main class="shell workspace onboarding"><header><p class="eyebrow">Set up your account</p><h1>You do not have to finish everything today.</h1><p class="lede">You can write down your first sale before any of this is finished. Do the important parts now, and come back for the money settings later.</p><p class="unregistered"><strong>You do not need a CAC number to start.</strong> Choose “Not registered yet” when you add your business. We will tell you clearly when we need your ID, your business papers or your bank details. That only happens before money moves.</p></header>
{#if loading}<p>Opening your setup…</p>{:else if loadError}<p role="alert">{loadError}</p><button onclick={load}>Try again</button>{:else if !orgID}<section class="card"><h2>Add your business first</h2><p>Add your business details, then come back here.</p><a href="/app/overview">Go to my dashboard →</a></section>{:else}
<section class="activation"><div><p class="eyebrow">The fastest way to understand Kredit</p><h2>Write down one real sale.</h2><p>Pick a customer, put in the goods and the money, set the payment day, then let them read it. One sale and you will understand the whole thing.</p></div><div><a class="activation-cta" href="/app/credit/quick">Add my first sale →</a><small>Bank details, fees, limits and extra safety can all wait until money is about to move.</small></div></section>
<section class="readiness" class:ready={readiness.ready}><div><p class="eyebrow">Ready for money?</p><h2>{readiness.ready?'Account setup complete':productLabel(readiness.state,'Finish the steps below')}</h2><p>{readiness.ready?'Your account setup checks are complete. Each sale and payment still needs its own eligibility and provider checks.':`${readiness.missing?.length??0} thing${(readiness.missing?.length??0)===1?'':'s'} still to do before money can move in or out.`}</p></div><strong>{readiness.requirements?.filter((r:any)=>r.complete).length}/{readiness.requirements?.length}</strong></section>
{#if message}<p class="notice" role="status">{message} {#if message.includes('MFA')}<a href="/app/settings/security">Open security</a>{/if}</p>{/if}
<section class="steps" aria-label="Setup progress">{#each readiness.requirements??[] as requirement}<article class:done={requirement.complete}><span>{requirement.complete?'✓':'○'}</span><div><strong>{simpleStep(requirement.code,requirement.label)}</strong>{#if !requirement.complete}<a href={`${requirement.manage_path}${requirement.manage_path.includes('?')?'&':'?'}organization=${encodeURIComponent(orgID)}`}>Do this now →</a>{/if}</div></article>{/each}</section>
<div class="forms">
{#if permissions.business}<section class="card"><p class="step">1 · Who is in charge</p><h2>Who is responsible for this business?</h2><label>Full name<input disabled={busy} bind:value={repName} autocomplete="name" /></label><label>What do they do here?<input disabled={busy} bind:value={repTitle} placeholder="For example: owner or manager" /></label><button disabled={busy||!repName||!repTitle} onclick={saveRepresentative}>Save</button></section>{/if}
{#if permissions.consents}<section class="card"><p class="step">2 · Your phone or email</p><h2>Confirm we can reach you</h2><div class="row"><label>Send the code by<select disabled={busy||Boolean(challengeID)} bind:value={channel}><option value="phone">Phone</option><option value="email">Email</option></select></label><label>Your email or phone number<input disabled={busy||Boolean(challengeID)} bind:value={contact} type={channel==='phone'?'tel':'email'} autocomplete={channel==='phone'?'tel':'email'} placeholder={channel==='phone'?'+234…':'you@business.com'} /></label></div>{#if challengeID}<label>The six-digit code<input disabled={busy} bind:value={code} inputmode="numeric" autocomplete="one-time-code" pattern={'[0-9]{6}'} maxlength="6" /></label><button disabled={busy||!/^\d{6}$/.test(code)} onclick={verifyContact}>Confirm my {channel}</button><button disabled={busy} onclick={()=>{challengeID='';code='';message='';}}>Use another contact or request a new code</button>{:else}<button disabled={busy||!contact} onclick={requestContact}>Send me the code</button>{/if}</section>{/if}
{#if permissions.business}<section class="card"><p class="step">3 · Check your business</p><h2>Show us who you are</h2><p>Registered or not, you can start here. Depending on your account we may ask for the owner's details, business papers or some other proof. We will never ask for your password or your bank PIN. Anybody who does is not us.</p>{#if profile.kyb_provider_reference && !['approved','rejected'].includes(profile.kyb_state)}<button disabled={busy} onclick={refreshKYB}>Check my status again</button>{:else}<button disabled={busy} onclick={submitKYB}>Start the check</button>{/if}<small>Where this stands: {productLabel(profile.kyb_state)}</small></section>{/if}
{#if permissions.consents}<section class="card"><p class="step">4 · The rules</p><h2>Read what you are agreeing to</h2><p>Please read the <a href={`/legal/terms?version=${encodeURIComponent(terms)}`}>terms</a> and the <a href={`/legal/privacy?version=${encodeURIComponent(privacy)}`}>privacy notice</a> before you agree to them.</p><button disabled={busy||(profile.terms_version===terms&&profile.privacy_version===privacy)} onclick={acceptConsents}>{profile.terms_version===terms&&profile.privacy_version===privacy?'Agreed':'I have read them and I agree'}</button></section>{/if}
<section class="card links"><p class="step">5 · Money and safety</p><h2>Finish these before money moves</h2><a href="/app/settings/settlement">Where we send your money →</a><a href="/app/settings/billing">How you pay Kredit fees →</a><a href="/app/settings/credit-policy">Your normal sale settings →</a><a href="/app/settings/security">Extra sign-in safety →</a><a href="/app/team">Protect staff who handle money →</a></section>
</div>{/if}</main>
<style>.onboarding header{max-width:58rem}.onboarding h1{max-width:14ch;font-family:var(--font-serif);font-size:clamp(3rem,6vw,5.2rem);font-weight:500;line-height:.94;letter-spacing:-.055em}.unregistered{max-width:50rem;padding:1rem;border-left:4px solid #2738d6;background:#eef0ff}.activation{display:grid;grid-template-columns:1fr auto;gap:3rem;align-items:center;margin:2rem 0;padding:1.5rem 1.7rem;color:#fff;background:#2738d6;box-shadow:8px 8px 0 #17181b}.activation h2{margin:.4rem 0;font-family:var(--font-serif);font-size:clamp(2rem,4vw,3rem);font-weight:500}.activation p:last-child{max-width:45rem;color:#d8dbff;line-height:1.6}.activation>div:last-child{display:grid;gap:.7rem;max-width:18rem}.activation-cta{display:inline-flex;padding:.85rem 1rem;background:#fff;color:#17181b;font-weight:850;text-decoration:none}.activation small{color:#d8dbff;line-height:1.45}.readiness{display:flex;justify-content:space-between;align-items:center;margin:2rem 0;padding:1.4rem;border:1px solid #e0b54f;background:#fff8e6}.readiness.ready{border-color:var(--color-positive);background:#effbf3}.readiness h2,.readiness p{margin:.2rem 0}.readiness>strong{font-size:2rem}.steps{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.65rem;margin:1.5rem 0}.steps article{display:flex;gap:.7rem;padding:.9rem;border:1px solid var(--color-border);background:#fffdf8}.steps article.done span{color:var(--color-positive)}.steps article div{display:grid;gap:.25rem}.forms{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1rem}.card{display:grid;align-content:start;gap:.75rem;padding:1.3rem;border:1px solid var(--color-border);border-radius:0;background:var(--color-surface)}.card h2,.card p{margin:0}.card label{display:grid;gap:.35rem;font-weight:700}.card input,.card select,.card button{box-sizing:border-box;width:100%;padding:.75rem;border:1px solid var(--color-border);border-radius:0;font:inherit}.card button{background:var(--color-primary);color:white;font-weight:800;cursor:pointer}.card button:disabled{opacity:.55}.row{display:grid;grid-template-columns:8rem 1fr;gap:.5rem}.links a{font-weight:750}.step{color:var(--color-primary);font-weight:800}.notice{padding:1rem;background:#eef5ff}@media(max-width:720px){.activation,.steps,.forms{grid-template-columns:1fr}.activation{gap:1rem;box-shadow:6px 6px 0 #17181b}.activation>div:last-child{max-width:none}.row{grid-template-columns:1fr}.readiness>strong{font-size:1.4rem}}</style>
