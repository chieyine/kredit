<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	let identifier = '', challengeID = '', code = '', developmentCode = '', error = '';
	let channel: 'email' | 'sms' = 'email';
	let busy = false;
	function destination() {
		const next = page.url.searchParams.get('next') ?? '';
		return next.startsWith('/') && !next.startsWith('//') ? next : '/app/overview';
	}
	onMount(async () => { if ((await fetch('/api/v1/me', { credentials: 'include' })).ok) await goto(destination()); });
	async function requestCode() {
		busy = true; error = '';
		const response = await fetch('/api/v1/auth/otp/challenges', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ identifier, channel, purpose: 'login' }) });
		const body = await response.json().catch(() => ({})); busy = false;
		if (!response.ok) { error = body.detail ?? 'We could not send a verification code.'; return; }
		challengeID = body.challenge_id; developmentCode = body.development_code ?? '';
	}
	async function verifyCode() {
		busy = true; error = '';
		const response = await fetch('/api/v1/auth/otp/verify', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ challenge_id: challengeID, code, device_label: navigator.userAgent.slice(0, 120) }) });
		const body = await response.json().catch(() => ({})); busy = false;
		if (!response.ok) { error = body.detail ?? 'That code was not accepted.'; return; }
		await goto(destination());
	}
</script>
<svelte:head><title>Sign in — Kredit</title></svelte:head>
<main class="shell auth-page">
	<section class="auth-copy"><p class="eyebrow">Seller account</p><h1>Welcome back.</h1><p class="lede">Sign in to see your customers, sales and payments.</p></section>
	<form class="card auth-card" onsubmit={(event) => { event.preventDefault(); challengeID ? verifyCode() : requestCode(); }}>
		<h2>{challengeID ? 'Enter the code' : 'Sign in'}</h2>
		{#if !challengeID}
			<label>Email or phone<input bind:value={identifier} type={channel === 'email' ? 'email' : 'tel'} autocomplete={channel === 'email' ? 'email' : 'tel'} required /></label>
			<label>Send code by<select bind:value={channel}><option value="email">Email</option><option value="sms">SMS</option></select></label>
		{:else}
			<p>We sent a six-digit code to <strong>{identifier}</strong>.</p>
			<label>Six-digit code<input bind:value={code} inputmode="numeric" autocomplete="one-time-code" minlength="6" maxlength="6" required /></label>
			{#if developmentCode}<p class="notice">Development code: <strong>{developmentCode}</strong></p>{/if}
		{/if}
		{#if error}<p class="error" role="alert">{error}</p>{/if}
		<button class="primary" disabled={busy || (challengeID ? code.length !== 6 : !identifier)}>{busy ? 'Please wait…' : challengeID ? 'Enter my account' : 'Send me a code'}</button>
		{#if challengeID}<button class="secondary" type="button" onclick={() => { challengeID = ''; code = ''; developmentCode = ''; }}>Change phone or email</button>{/if}
	</form>
</main>
<style>.auth-page{display:grid;grid-template-columns:1.1fr minmax(18rem,28rem);gap:clamp(2rem,8vw,7rem);align-items:center;min-height:calc(100vh - 6rem);padding-block:4rem}.auth-copy h1{font-size:clamp(3rem,8vw,6rem);letter-spacing:-.06em;line-height:.95;margin:.5rem 0}.auth-card{display:grid;gap:1rem;padding:clamp(1.25rem,4vw,2rem)}label{display:grid;gap:.4rem;font-weight:750}input,select{box-sizing:border-box;width:100%;padding:.8rem;border:1px solid var(--color-border);border-radius:.65rem;font:inherit;background:var(--color-surface)}.auth-card button{width:100%}.secondary{background:transparent;border:1px solid var(--color-border);padding:.75rem;border-radius:999px}.error{color:var(--color-danger,#b42318)}@media(max-width:760px){.auth-page{grid-template-columns:1fr;align-items:start}.auth-copy h1{font-size:3.5rem}}</style>
