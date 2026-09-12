<script lang="ts">
	import { page } from '$app/state';
	import { adminPost } from '$lib/admin-client';
	let identifier=$state(''),channel=$state('email'),requestID=$state(''),recoveryCode=$state(''),secondIdentifier=$state(''),secondChannel=$state('phone'),challengeID=$state(''),contactCode=$state(''),completionToken=$state(''),message=$state(''),error=$state(''),busy=$state('');
	$effect(()=>{
		const request=new URLSearchParams(page.url.hash.slice(1)).get('request')??page.url.searchParams.get('request');
		const token=new URLSearchParams(page.url.hash.slice(1)).get('token')??page.url.searchParams.get('token');
		if(request!==null)requestID=request;
		if(token!==null)completionToken=token;
	});
	async function post(path:string,body:unknown){
		if(busy)return null;
		busy=path;error='';message='';
		try { const data=await adminPost(path,body);message=typeof data.message==='string'?data.message:'Saved.';return data; }
		catch(cause){error=cause instanceof Error?cause.message:'We could not confirm this step. Check your connection and try again.';return null;}
		finally{busy='';}
	}
	async function start(){const data=await post('/api/v1/account-recovery/requests',{identifier,channel});if(data?.development_request_id)requestID=data.development_request_id}
	async function addRecoveryCode(){const data=await post(`/api/v1/account-recovery/requests/${encodeURIComponent(requestID)}/evidence`,{factor_type:'recovery_code',proof:recoveryCode});if(data)recoveryCode=''}
	async function requestContactCode(){const data=await post('/api/v1/auth/otp/challenges',{identifier:secondIdentifier,channel:secondChannel,purpose:'recovery'});if(data){challengeID=data.challenge_id;contactCode=data.development_code??''}}
	function changeContact(){if(busy)return;challengeID='';contactCode='';error='';message='';}
	async function addContact(){const data=await post(`/api/v1/account-recovery/requests/${encodeURIComponent(requestID)}/evidence`,{factor_type:secondChannel==='email'?'verified_email':'verified_phone',proof:'',challenge_id:challengeID,code:contactCode,channel:secondChannel,identifier:secondIdentifier});if(data){challengeID='';contactCode='';secondIdentifier=''}}
	async function complete(){const data=await post(`/api/v1/account-recovery/requests/${encodeURIComponent(requestID)}/complete`,{token:completionToken});if(data){completionToken='';message='Your account recovery is complete. Sign in again, then set up a new authenticator in Security settings before making protected changes.'}}
	async function cancel(){const data=await post(`/api/v1/me/account-recovery/${encodeURIComponent(requestID)}/cancel`,{});if(data)message='This recovery request was cancelled.'}
</script>
<svelte:head><title>Get back into your account — Kredit</title></svelte:head>
<main class="shell recovery"><a href="/">Kredit</a><p class="eyebrow">Account help</p><h1>Let us get you back in.</h1><p>We will ask for more than your phone number. You need two different proofs. This is what stops somebody else from stealing your account.</p>{#if message}<p class="notice" role="status">{message}</p>{/if}{#if error}<p class="error" role="alert">{error}</p>{/if}
	<section><h2>1. Tell us who you are</h2><label>Your email or phone number<input disabled={Boolean(busy)} bind:value={identifier} autocomplete="username"/></label><label>Which one did you type?<select disabled={Boolean(busy)} bind:value={channel}><option value="email">Email</option><option value="phone">Phone</option></select></label><button onclick={start} disabled={Boolean(busy)||!identifier}>Start</button></section>
	{#if requestID}<section><h2>2. Give us two proofs</h2><p>First, a backup code if you kept one.</p><label>Backup code<input disabled={Boolean(busy)} bind:value={recoveryCode} autocomplete="one-time-code"/></label><button onclick={addRecoveryCode} disabled={Boolean(busy)||!recoveryCode}>Check this backup code</button><hr/><p>Second, another phone number or email that belongs to this same account.</p><label>Use<select disabled={Boolean(busy)||Boolean(challengeID)} bind:value={secondChannel}><option value="phone">Phone</option><option value="email">Email</option></select></label><label>{secondChannel==='phone'?'Phone number':'Email'}<input disabled={Boolean(busy)||Boolean(challengeID)} bind:value={secondIdentifier} type={secondChannel==='phone'?'tel':'email'} autocomplete={secondChannel==='phone'?'tel':'email'}/></label>{#if challengeID}<label>The six-digit code we sent<input disabled={Boolean(busy)} bind:value={contactCode} inputmode="numeric" maxlength="6" autocomplete="one-time-code"/></label><button onclick={addContact} disabled={Boolean(busy)||contactCode.length!==6}>Check this code</button><button class="secondary" onclick={changeContact} disabled={Boolean(busy)}>Use a different contact</button>{:else}<button onclick={requestContactCode} disabled={Boolean(busy)||!secondIdentifier}>Send me a code</button>{/if}</section>
	<section><h2>3. Finish, once we approve it</h2><p>Kredit will check your request, then wait a short safety period before finishing. We will send you a private code to finish with.</p><label>The private code we sent you<input disabled={Boolean(busy)} bind:value={completionToken} autocomplete="off"/></label><button onclick={complete} disabled={Boolean(busy)||!completionToken}>Finish and get back in</button><details><summary>Did you not ask for this?</summary><p>If you are already signed in and you did not ask to recover this account, stop it now.</p><button class="secondary" onclick={cancel} disabled={Boolean(busy)}>Stop this request</button></details></section>{/if}
</main>
<style>.recovery{max-width:44rem;padding-top:4rem}.recovery h1{font-size:clamp(2.5rem,7vw,5rem);line-height:1}.recovery section{display:grid;gap:.8rem;margin:1rem 0;padding:1.3rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}label{display:grid;gap:.35rem;font-weight:700}input,select,button{padding:.8rem;border:1px solid var(--color-border);border-radius:.65rem;font:inherit}button{background:var(--color-primary);color:white;font-weight:800}.secondary{background:var(--color-surface);color:var(--color-destructive);border-color:var(--color-destructive)}.notice{padding:1rem;background:#eef5ff;border-radius:.7rem}.error{color:var(--color-destructive)}hr{width:100%;border:0;border-top:1px solid var(--color-border)}</style>
