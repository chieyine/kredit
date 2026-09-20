<script lang="ts">
	import { onMount } from 'svelte';
	import { adminPost } from '$lib/admin-client';
	import { checkedJSON, record, text, publicError } from '$lib/api/reliable';
	let enrolled = false, loading = true, loadError = '';
	let secret = '', code = '', message = '';
	let busy = false;
	let copied = false;
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
			const nextSecret=text(body.secret);
			if(!nextSecret)throw new Error('We could not confirm the setup key. Try again.');
			secret=nextSecret;
			message='Add Kredit to your authenticator app, then type the six-digit code it shows.';
		}catch(cause){message=cause instanceof Error?cause.message:'Setup could not be confirmed.';}
		finally{busy=false;}
	}
	async function copySecret() {
		if (!secret) return;
		try {
			await navigator.clipboard.writeText(secret);
			copied = true;
			setTimeout(() => { copied = false; }, 3000);
		} catch {
			// Clipboard API fallback
		}
	}
	async function verify() {
		if(busy||!/^\d{6}$/.test(code))return;busy=true;message='';
		try {
			const body=await adminPost('/api/v1/mfa/totp/verify',{code});
			if(body.authentication_level!=='AAL2')throw new Error('We could not confirm verification. Refresh your safety settings.');
			if(Array.isArray(body.recovery_codes)&&body.recovery_codes.length)recoveryCodes=body.recovery_codes.map(text);
			enrolled=true;secret='';code='';
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
	<h1>Account security</h1>
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
			<p>Use an app like Google Authenticator, Microsoft Authenticator, or 1Password. Never send these codes to anybody, not even to somebody who says they are from Kredit.</p>
			<button class="primary" type="button" disabled={busy} onclick={beginEnrollment}>{busy ? 'Starting…' : 'Start extra safety'}</button>
		{:else}
			<div class="setup-guide">
				<p>In your authenticator app (Google Authenticator, Microsoft Authenticator, etc.):</p>
				<ol class="guide-steps">
					<li>Tap <strong>+</strong> and select <strong>Enter a setup key</strong>.</li>
					<li>For <strong>Account name</strong>, type <code>Kredit</code> (or any name you won't forget).</li>
					<li>For <strong>Your key</strong>, copy and enter the key below:</li>
				</ol>
				<div class="key-container">
					<code class="key-text">{secret}</code>
					<button type="button" class="copy-btn" onclick={copySecret}>
						{copied ? '✓ Copied' : 'Copy key'}
					</button>
				</div>
			</div>
			<label class="field">
				Type the six-digit code from your app
				<input disabled={busy} bind:value={code} inputmode="numeric" autocomplete="one-time-code" maxlength="6" placeholder="123456" />
			</label>
			<button class="primary" type="button" disabled={busy || !/^\d{6}$/.test(code)} onclick={verify}>{busy ? 'Checking…' : 'Check code and turn it on'}</button>
		{/if}
	</section>
	{/if}
	{#if recoveryCodes.length}<section class="card"><h2>Write these backup codes down now</h2><p>Keep them somewhere safe and private. Each code works once, and we cannot show them to you again.</p><ul>{#each recoveryCodes as recoveryCode}<li><code>{recoveryCode}</code></li>{/each}</ul></section>{/if}
</main>

<style>
	.form-page { max-width: 52rem; }
	.card { display: grid; gap: 1.2rem; padding: 1.5rem; border: 1px solid var(--color-border); border-radius: 1rem; background: var(--color-surface); }
	.card label { display: grid; gap: .4rem; font-weight: 600; }
	.card input, .card button { padding: .75rem 1rem; border: 1px solid var(--color-border); border-radius: .65rem; font: inherit; }
	.card button.primary { background: var(--color-primary); color:var(--color-on-primary); font-weight: 700; cursor: pointer; }
	.card button.primary:disabled { opacity: 0.5; cursor: not-allowed; }
	.notice { padding: 1rem; background:var(--color-background); border-radius: .7rem; border-left:4px solid var(--color-primary, var(--color-primary)); }
	.setup-guide { display: grid; gap: .8rem; padding: 1.2rem; background:var(--color-surface-muted, var(--color-surface)); border-radius: .8rem; border: 1px solid var(--color-border); }
	.guide-steps { margin: 0; padding-left: 1.4rem; display: grid; gap: .5rem; line-height: 1.5; }
	.guide-steps code { font-weight: 700; background:var(--color-surface); padding: .15rem .4rem; border-radius: .3rem; border: 1px solid var(--color-border); }
	.key-container { display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding: .8rem 1rem; background:var(--color-surface); border: 1px solid var(--color-border); border-radius: .65rem; flex-wrap: wrap; }
	.key-text { font-family: monospace; font-size: 1.05rem; font-weight: 700; letter-spacing: .08em; word-break: break-all; color:var(--color-text, var(--color-primary)); }
	.copy-btn { padding: .5rem .9rem; border: 1px solid var(--color-border); border-radius: .5rem; background:var(--color-surface, var(--color-surface)); font-weight: 700; font-size: .85rem; cursor: pointer; transition: all .15s ease; white-space: nowrap; }
	.copy-btn:hover { border-color:var(--color-primary, var(--color-primary)); color:var(--color-primary, var(--color-primary)); }
</style>
