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
			message = body.detail ?? 'MFA enrollment could not be started.';
			return;
		}
		secret = body.secret ?? '';
		otpURI = body.otpauth_uri ?? '';
		message = 'Add this account to your authenticator, then enter the six-digit code.';
	}

	async function verify() {
		busy = true;
		const response = await fetch('/api/v1/mfa/totp/verify', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json', ...csrfHeaders() }, body: JSON.stringify({ code }) });
		const body = await response.json().catch(() => ({}));
		busy = false;
		if (!response.ok) {
			message = body.detail ?? 'That authenticator code was not accepted.';
			return;
		}
		enrolled = true;
		recoveryCodes = body.recovery_codes ?? [];
		secret = '';
		otpURI = '';
		code = '';
		message = 'MFA is enabled for this session. High-impact actions now require step-up authentication.';
	}

	async function regenerateCodes() {
		busy = true;
		const response = await fetch('/api/v1/me/recovery-codes/regenerate', { method: 'POST', credentials: 'include', headers: { 'Idempotency-Key': idempotencyKey(), ...csrfHeaders() } });
		const body = await response.json().catch(() => ({}));
		busy = false;
		if (!response.ok) { message = body.detail ?? 'Recovery codes could not be regenerated.'; return; }
		recoveryCodes = body.recovery_codes ?? [];
		message = 'New recovery codes created. Every older code is now invalid.';
	}

	onMount(load);
</script>

<svelte:head><title>Security settings — Kredit</title></svelte:head>
<main class="shell workspace form-page">
	<p class="eyebrow">Settings / Access</p>
	<h1>Protect high-impact actions.</h1>
	<p class="lede">MFA protects financial changes, member administration, and other material operations.</p>
	{#if message}<p class="notice" role="status">{message}</p>{/if}
	<section class="card">
		<h2>{enrolled ? 'Authenticator enabled' : 'Enable authenticator MFA'}</h2>
		{#if enrolled}
			<p>Your account is enrolled. Re-authenticate with your authenticator when a material action asks for step-up confirmation.</p>
			<label class="field">Authenticator code<input bind:value={code} inputmode="numeric" autocomplete="one-time-code" maxlength="6" /></label>
			<button class="primary" type="button" disabled={busy || code.length !== 6} onclick={verify}>{busy ? 'Verifying…' : 'Refresh secure session'}</button>
			<button type="button" disabled={busy} onclick={regenerateCodes}>Regenerate recovery codes</button>
		{:else if !secret}
			<p>Use an authenticator app rather than receiving financial step-up codes by email or chat.</p>
			<button class="primary" type="button" disabled={busy} onclick={beginEnrollment}>{busy ? 'Starting…' : 'Start MFA enrollment'}</button>
		{:else}
			<p>Scan this URI in your authenticator app, or enter the secret manually.</p>
			<p><code>{secret}</code></p>
			<p class="muted"><code>{otpURI}</code></p>
			<label class="field">Authenticator code<input bind:value={code} inputmode="numeric" autocomplete="one-time-code" maxlength="6" /></label>
			<button class="primary" type="button" disabled={busy || code.length !== 6} onclick={verify}>{busy ? 'Verifying…' : 'Verify and enable MFA'}</button>
		{/if}
	</section>
	{#if recoveryCodes.length}<section class="card"><h2>Save these recovery codes now</h2><p>Each code works once. They will not be shown again.</p><ul>{#each recoveryCodes as recoveryCode}<li><code>{recoveryCode}</code></li>{/each}</ul></section>{/if}
</main>
