<script lang="ts">
	import { onMount } from 'svelte';
	import { adminPost } from '$lib/admin-client';
	import { checkedJSON, record, text, publicError } from '$lib/api/reliable';
	let enrolled = false, loading = true, loadError = '';
	let secret = '', otpURI = '', code = '', message = '';
	let busy = false;
	let recoveryCodes: string[] = [];
	async function load() {
		loading=true;loadError='';
		try { enrolled=await checkedJSON('/api/v1/me',value=>{const body=record(value);if(typeof body.mfa_enrolled!=='boolean')throw new Error('Missing safety status');return body.mfa_enrolled;}); }
		catch(cause){loadError=publicError(cause,'your sign-in safety');}
		finally{loading=false;}
	}
	async function beginEnrollment() {
		if(busy||loading||loadError)return;busy=true;message='';
		try {
			const body=await adminPost('/api/v1/mfa/totp/enroll',{});
			const nextSecret=text(body.secret),nextURI=text(body.otpauth_uri);
			if(!nextSecret||!nextURI)throw new Error('We could not confirm the setup key. Try again.');
			secret=nextSecret;otpURI=nextURI;
			message='Add Kredit to your app, then type the six-digit code it shows.';
		}catch(cause){message=cause instanceof Error?cause.message:'Setup could not be confirmed.';}
		finally{busy=false;}
	}
	async function verify() {
		if(busy||!/^\d{6}$/.test(code))return;busy=true;message='';
		try {
			const body=await adminPost('/api/v1/mfa/totp/verify',{code});
			if(body.authentication_level!=='AAL2')throw new Error('We could not confirm verification. Refresh your safety settings.');
			if(Array.isArray(body.recovery_codes)&&body.recovery_codes.length)recoveryCodes=body.recovery_codes.map(text);
			enrolled=true;secret='';otpURI='';code='';
			message='Extra sign-in safety is now on. We will ask for a code before any important change.';
		}catch(cause){message=cause instanceof Error?cause.message:'Verification could not be confirmed.';}
		finally{busy=false;}
	}
	async function regenerateCodes() {
		if(busy)return;busy=true;message='';recoveryCodes=[];
		try {
			const body=await adminPost('/api/v1/me/recovery-codes/regenerate',{});
			if(!Array.isArray(body.recovery_codes)||!body.recovery_codes.length)throw new Error('We could not retrieve your new backup codes. Make a new set before relying on them.');
			recoveryCodes=body.recovery_codes.map(text);
			message='Your new backup codes are ready. The old ones no longer work.';
		}catch(cause){message=cause instanceof Error?cause.message:'New backup codes could not be confirmed.';}
		finally{busy=false;}
	}
	onMount(load);
</script>

<svelte:head><title>Security settings — Kredit</title></svelte:head>
<main class="shell workspace form-page">
	<p class="eyebrow">Settings / Access</p>
	<h1>Keep your money and your account safe.</h1>
	<p class="lede">Ask for a six-digit code from an app on your phone before anybody can change money, bank or staff details. This adds protection if somebody gains access to your phone number.</p>
	{#if message}<p class="notice" role="status">{message}</p>{/if}
	{#if loading}<p role="status">Checking sign-in safety…</p>{:else if loadError}<p class="error" role="alert">{loadError}</p><button onclick={load}>Try again</button>{:else}
	<section class="card">
		<h2>{enrolled ? 'Extra sign-in safety is on' : 'Add extra sign-in safety'}</h2>
		{#if enrolled}
			<p>When Kredit asks, open that app on your phone and type the six digits it shows.</p>
			<label class="field">Authenticator code<input disabled={busy} bind:value={code} inputmode="numeric" autocomplete="one-time-code" maxlength="6" /></label>
			<button class="primary" type="button" disabled={busy || !/^\d{6}$/.test(code)} onclick={verify}>{busy ? 'Checking…' : 'Check this code'}</button>
			<button type="button" disabled={busy} onclick={regenerateCodes}>Make new backup codes</button>
		{:else if !secret}
			<p>Use an app like Google Authenticator or Microsoft Authenticator. Never send these codes to anybody, not even to somebody who says they are from Kredit.</p>
			<button class="primary" type="button" disabled={busy} onclick={beginEnrollment}>{busy ? 'Starting…' : 'Start extra safety'}</button>
		{:else}
			<p>In that app, add a new account and type the setup key below.</p>
			<p><code>{secret}</code></p>
			<p class="muted"><code>{otpURI}</code></p>
			<label class="field">Authenticator code<input disabled={busy} bind:value={code} inputmode="numeric" autocomplete="one-time-code" maxlength="6" /></label>
			<button class="primary" type="button" disabled={busy || !/^\d{6}$/.test(code)} onclick={verify}>{busy ? 'Checking…' : 'Check code and turn it on'}</button>
		{/if}
	</section>
	{/if}
	{#if recoveryCodes.length}<section class="card"><h2>Write these backup codes down now</h2><p>Keep them somewhere safe and private. Each code works once, and we cannot show them to you again.</p><ul>{#each recoveryCodes as recoveryCode}<li><code>{recoveryCode}</code></li>{/each}</ul></section>{/if}
</main>
