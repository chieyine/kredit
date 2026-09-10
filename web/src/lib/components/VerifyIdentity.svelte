<script lang="ts">
 import {adminGet,adminPost} from '$lib/admin-client';
 let code=$state(''),busy=$state(false),message=$state('');
 async function verify(){if(busy||!/^\d{6}$/.test(code))return;busy=true;message='';try{const me=await adminGet('/api/v1/me');if(!me.mfa_enrolled){message='Set up an authenticator in account security first.';return}const result=await adminPost('/api/v1/mfa/totp/verify',{code});if(result.authentication_level!=='AAL2')throw new Error('We could not confirm your identity. Try again.');code='';message='Identity confirmed. You can now submit your decision.'}catch(e){message=String(e)}finally{busy=false}}
</script>
<details><summary>Prove it is really you before this change</summary><p>Open your authenticator app and type the six digits it shows. <a href="/app/settings/security">Account safety settings</a>.</p><form onsubmit={(e)=>{e.preventDefault();verify()}}><label>Authenticator code<input disabled={busy} bind:value={code} inputmode="numeric" autocomplete="one-time-code" pattern={'[0-9]{6}'} maxlength="6" required/></label><button disabled={busy||!/^\d{6}$/.test(code)}>Confirm it is me</button></form>{#if message}<p role="status">{message}</p>{/if}</details>
<style>details{padding:1rem;border:1px solid #bbb6ac;margin:1rem 0}input,button{padding:.7rem;margin:.5rem}input{width:8rem}summary{cursor:pointer}</style>
