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
			message = body.detail ?? 'We could not turn on extra sign-in safety. Please try again.';
			return;
		}
		secret = body.secret ?? '';
		otpURI = body.otpauth_uri ?? '';
		message = 'Add Kredit to your app, then type the six-digit code it shows.';
	}

	async function verify() {
		busy = true;
		const response = await fetch('/api/v1/mfa/totp/verify', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json', ...csrfHeaders() }, body: JSON.stringify({ code }) });
		const body = await response.json().catch(() => ({}));
		busy = false;
		if (!response.ok) {
			message = body.detail ?? 'That code did not work. Check the six digits and try again.';
			return;
		}
		enrolled = true;
		recoveryCodes = body.recovery_codes ?? [];
		secret = '';
		otpURI = '';
		code = '';
		message = 'Extra sign-in safety is now on. We will ask for a code before any important change.';
	}

	async function regenerateCodes() {
		busy = true;
		const response = await fetch('/api/v1/me/recovery-codes/regenerate', { method: 'POST', credentials: 'include', headers: { 'Idempotency-Key': idempotencyKey(), ...csrfHeaders() } });
		const body = await response.json().catch(() => ({}));
		busy = false;
		if (!response.ok) { message = body.detail ?? 'We could not make new backup codes. Please try again.'; return; }
		recoveryCodes = body.recovery_codes ?? [];
		message = 'Your new backup codes are ready. The old ones no longer work.';
	}

	onMount(load);
</script>

<svelte:head><title>Security settings — Kredit</title></svelte:head>
<main class="shell workspace form-page">
	<p class="eyebrow">Settings / Access</p>
	<h1>Keep your money and your account safe.</h1>
	<p class="lede">Ask for a six-digit code from an app on your phone before anybody can change money, bank or staff details. Even if somebody steals your phone number, they still cannot get in.</p>
	{#if message}<p class="notice" role="status">{message}</p>{/if}
	<section class="card">
		<h2>{enrolled ? 'Extra sign-in safety is on' : 'Add extra sign-in safety'}</h2>
		{#if enrolled}
			<p>When Kredit asks, open that app on your phone and type the six digits it shows.</p>
			<label class="field">Authenticator code<input bind:value={code} inputmode="numeric" autocomplete="one-time-code" maxlength="6" /></label>
			<button class="primary" type="button" disabled={busy || code.length !== 6} onclick={verify}>{busy ? 'Checking…' : 'Check this code'}</button>
			<button type="button" disabled={busy} onclick={regenerateCodes}>Make new backup codes</button>
		{:else if !secret}
			<p>Use an app like Google Authenticator or Microsoft Authenticator. Never send these codes to anybody, not even to somebody who says they are from Kredit.</p>
			<button class="primary" type="button" disabled={busy} onclick={beginEnrollment}>{busy ? 'Starting…' : 'Start extra safety'}</button>
		{:else}
			<p>In that app, add a new account and type the setup key below.</p>
			<p><code>{secret}</code></p>
			<p class="muted"><code>{otpURI}</code></p>
			<label class="field">Authenticator code<input bind:value={code} inputmode="numeric" autocomplete="one-time-code" maxlength="6" /></label>
			<button class="primary" type="button" disabled={busy || code.length !== 6} onclick={verify}>{busy ? 'Checking…' : 'Check code and turn it on'}</button>
		{/if}
	</section>
	{#if recoveryCodes.length}<section class="card"><h2>Write these backup codes down now</h2><p>Keep them somewhere safe and private. Each code works once, and we cannot show them to you again.</p><ul>{#each recoveryCodes as recoveryCode}<li><code>{recoveryCode}</code></li>{/each}</ul></section>{/if}
</main>
