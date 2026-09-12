<script lang="ts">
 import SellerSettlements from '$lib/components/SellerSettlements.svelte';
 import {onMount} from 'svelte';
 import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
 import {loadOnboardingSettings,settingsProfile} from '$lib/api/onboarding-settings';
 import {checkedJSON,record,rows,text,publicError} from '$lib/api/reliable';
 import {MutationIntent} from '$lib/api/mutation';
 import {productLabel} from '$lib/product-language';
 let organizationName='',orgID='',profile:Record<string,unknown>={},permissions:Record<string,unknown>={},message='',loadError='',bankError='',busy=false,loading=true;
 let banks:{code:string;name:string}[]=[],bankCode='',accountNumber='',reviewed=false;
 let intent:MutationIntent|null=null;
 async function load(){loading=true;loadError='';bankError='';try{const result=await loadOnboardingSettings(orgID||new URLSearchParams(location.search).get('organization'));orgID=result.orgID;organizationName=result.organizationName;profile=result.profile;permissions=result.permissions;intent=new MutationIntent('settlement-account',`/api/v1/organizations/${encodeURIComponent(orgID)}/onboarding/settlement`);if(permissions.settlement){try{banks=await checkedJSON(`/api/v1/organizations/${encodeURIComponent(orgID)}/onboarding/settlement/banks`,rows('banks',value=>{const bank=record(value);return{code:text(bank.code),name:text(bank.name)}}));}catch(cause){bankError=publicError(cause,'Banks could not be loaded.')}}}catch(cause){loadError=publicError(cause,'Settings could not be loaded.')}finally{loading=false}}
 async function save(){if(busy||loading||loadError||!orgID||!intent||!reviewed)return;busy=true;message='';try{profile=await intent.run({expected_version:profile.version,bank_code:bankCode,account_number:accountNumber},value=>settingsProfile(record(value).profile),'PUT');accountNumber='';reviewed=false;message='Your account was registered. It is awaiting business ownership review.';}catch(cause){message=publicError(cause,'The registration is unconfirmed. Check your saved account or contact support before trying again.')}finally{busy=false}}
 onMount(load);
</script>
<svelte:head><title>Bank account — Kredit</title></svelte:head>
<main class="shell workspace form-page"><p class="eyebrow">Settings / Bank account</p><h1>Where should your payments go?</h1><p>Add the business bank account that should receive your customers’ payments. Kredit keeps the provider reference and last four digits, not the full account number.</p>
{#if message}<p class="notice" role="status">{message}</p>{/if}
{#if loading}<p role="status">Loading your settings…</p>{:else if loadError}<p role="alert">{loadError}</p><button onclick={load}>Try again</button>{:else}
<p>Business: {organizationName}</p><section class="card"><h2>Your bank account</h2><p>{productLabel(profile.settlement_state,'Not started')}</p>
{#if profile.settlement_account_last4}<p>{profile.settlement_bank_name} · {profile.settlement_account_name} · •••• {profile.settlement_account_last4}</p>{/if}
{#if permissions.settlement}<VerifyIdentity />
{#if bankError}<p role="alert">{bankError}</p><button disabled={busy} onclick={load}>Refresh bank list</button>{:else}
<form onsubmit={(event)=>{event.preventDefault();void save()}}><fieldset disabled={busy}>
<label>Bank<select bind:value={bankCode} onchange={()=>reviewed=false} required><option value="">Choose your bank</option>{#each banks as bank(bank.code)}<option value={bank.code}>{bank.name}</option>{/each}</select></label>
<label>Account number<input bind:value={accountNumber} oninput={()=>reviewed=false} inputmode="numeric" autocomplete="off" pattern={'[0-9]{10}'} maxlength="10" required /></label>
<p>The bank will return the account name. Registration is followed by an ownership review before this account becomes active. Changing your destination does not redirect an earlier payment.</p>
<label class="confirm"><input type="checkbox" bind:checked={reviewed} required />I checked the bank and account number. This account belongs to this business.</label>
<button disabled={!reviewed||!bankCode||!/^\d{10}$/.test(accountNumber)}>{busy?'Registering…':'Register business account'}</button>
</fieldset></form>{/if}
{:else}<p>Only the owner or a person with financial access can change this account.</p>{/if}</section>{/if}
<a href="/app/settings">All settings</a></main>
<style>.form-page{max-width:52rem}.card,fieldset{display:grid;gap:1rem}.card{padding:1.4rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}fieldset{border:0;padding:0;min-width:0}label{display:grid;gap:.4rem}input,select,button{padding:.75rem;border:1px solid var(--color-border);border-radius:.65rem;font:inherit}button{background:var(--color-primary);color:white}.confirm{display:flex;align-items:start}.notice{padding:1rem;background:#eef5ff;border-radius:.7rem}</style>

{#if orgID&&!loading&&!loadError}<SellerSettlements organizationID={orgID}/>{/if}
