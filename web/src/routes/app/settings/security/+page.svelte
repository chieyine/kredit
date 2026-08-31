<script lang="ts">
	import { onMount } from 'svelte';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';

	let enrolled = false;
	let secret = '';
	let otpURI = '';
	let code = '';
	let message = '';
	let busy = false;
	let recoveryCodes: string[] = [];

	async function load() {
			const response = await fetch('/api/v1/me', { credentials: 'include' });
		if (response.ok) enrolled = Boolean((await response.json()).mfa_enrolled);
	}

	async function beginEnrollment() {
		busy = true;
		message = '';
		const response = await fetch('/api/v1/mfa/totp/enroll', { method: 'POST', credentials: 'include', headers: csrfHeaders() });
		const body = await response.json().catch(() => ({}));
		busy = false;
		if (!response.ok) {
			message = body.detail ?? 'We could not start extra sign-in safety.';
			return;
		}
		secret = body.secret ?? '';
		otpURI = body.otpauth_uri ?? '';
		message = 'Add Kredit to your authenticator app, then enter the six-digit code.';
	}

	async function verify() {
		busy = true;
		const response = await fetch('/api/v1/mfa/totp/verify', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json', ...csrfHeaders() }, body: JSON.stringify({ code }) });
		const body = await response.json().catch(() => ({}));
		busy = false;
		if (!response.ok) {
			message = body.detail ?? 'That six-digit code did not work.';
			return;
		}
		enrolled = true;
		recoveryCodes = body.recovery_codes ?? [];
		secret = '';
		otpURI = '';
		code = '';
		message = 'Extra sign-in safety is now on. Kredit will ask for a new code before an important change.';
	}

	async function regenerateCodes() {
		busy = true;
		const response = await fetch('/api/v1/me/recovery-codes/regenerate', { method: 'POST', credentials: 'include', headers: { 'Idempotency-Key': idempotencyKey(), ...csrfHeaders() } });
		const body = await response.json().catch(() => ({}));
		busy = false;
		if (!response.ok) { message = body.detail ?? 'We could not make new backup codes.'; return; }
		recoveryCodes = body.recovery_codes ?? [];
		message = 'New backup codes are ready. Your old codes will no longer work.';
	}

	onMount(load);
</script>

<svelte:head><title>Security settings — Kredit</title></svelte:head>
<main class="shell workspace form-page">
	<p class="eyebrow">Settings / Access</p>
	<h1>Keep your money and account safe.</h1>
	<p class="lede">Use a six-digit code from an authenticator app before changing money, bank or staff details.</p>
	{#if message}<p class="notice" role="status">{message}</p>{/if}
	<section class="card">
		<h2>{enrolled ? 'Extra sign-in safety is on' : 'Add extra sign-in safety'}</h2>
		{#if enrolled}
			<p>When Kredit asks, open your authenticator app and enter the new six-digit code.</p>
			<label class="field">Authenticator code<input bind:value={code} inputmode="numeric" autocomplete="one-time-code" maxlength="6" /></label>
			<button class="primary" type="button" disabled={busy || code.length !== 6} onclick={verify}>{busy ? 'Checking…' : 'Check this code'}</button>
			<button type="button" disabled={busy} onclick={regenerateCodes}>Make new backup codes</button>
		{:else if !secret}
			<p>Use an authenticator app such as Google Authenticator or Microsoft Authenticator. Do not send these codes to anyone.</p>
			<button class="primary" type="button" disabled={busy} onclick={beginEnrollment}>{busy ? 'Starting…' : 'Start extra safety'}</button>
		{:else}
			<p>In your authenticator app, add a new account and enter the setup key below.</p>
			<p><code>{secret}</code></p>
			<p class="muted"><code>{otpURI}</code></p>
			<label class="field">Authenticator code<input bind:value={code} inputmode="numeric" autocomplete="one-time-code" maxlength="6" /></label>
			<button class="primary" type="button" disabled={busy || code.length !== 6} onclick={verify}>{busy ? 'Checking…' : 'Check code and turn it on'}</button>
		{/if}
	</section>
	{#if recoveryCodes.length}<section class="card"><h2>Save these backup codes now</h2><p>Keep them somewhere safe. Each code works once and will not be shown again.</p><ul>{#each recoveryCodes as recoveryCode}<li><code>{recoveryCode}</code></li>{/each}</ul></section>{/if}
</main>
