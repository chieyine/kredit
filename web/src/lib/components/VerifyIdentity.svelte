<script lang="ts">
 import {adminGet,adminPost} from '$lib/admin-client';
 let code=$state(''),busy=$state(false),message=$state('');
 async function verify(){busy=true;message='';try{const me=await adminGet('/api/v1/me');if(!me.mfa_enrolled){message='Set up an authenticator in account security first.';return}await adminPost('/api/v1/mfa/totp/verify',{code});code='';message='Identity confirmed. You can now submit your decision.'}catch(e){message=String(e)}finally{busy=false}}
</script>
<details><summary>Confirm your identity before a protected change</summary><p>Use your authenticator code if your recent verification has expired. <a href="/app/settings/security">Manage account security</a>.</p><form onsubmit={(e)=>{e.preventDefault();verify()}}><label>Authenticator code<input bind:value={code} inputmode="numeric" autocomplete="one-time-code" pattern="[0-9]{6}" maxlength="6" required/></label><button disabled={busy}>Confirm identity</button></form>{#if message}<p role="status">{message}</p>{/if}</details>
<style>details{padding:1rem;border:1px solid #bbb6ac;margin:1rem 0}input,button{padding:.7rem;margin:.5rem}input{width:8rem}summary{cursor:pointer}</style>
