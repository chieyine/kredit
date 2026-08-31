<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { idempotencyKey } from '$lib/api/client';

	type Preview = {
		invitation: {
			proposed_legal_name: string;
			proposed_business_type: string;
			proposed_address: string;
			proposed_industry: string;
			expires_at: string;
		};
		supplier: { legal_name: string; trading_name: string };
	};

	let preview: Preview | null = null;
	let challengeId = '';
	let developmentCode = '';
	let code = '';
	let fullName = '';
	let loading = true;
	let error = '';
	let acceptanceKey = '';

	async function loadPreview() {
		const response = await fetch(`/api/v1/buyer-invitations/${page.params.token}`);
		if (!response.ok) {
			error = 'This invitation is unavailable or has expired.';
			return;
		}
		preview = await response.json();
	}

	async function requestCode() {
		error = '';
		const response = await fetch(`/api/v1/buyer-invitations/${page.params.token}/otp`, { method: 'POST' });
		const body = await response.json();
		if (!response.ok) {
			error = body.detail ?? 'We could not send the six-digit code.';
			return;
		}
		challengeId = body.challenge_id;
		developmentCode = body.development_code ?? '';
	}

	async function accept() {
		error = '';
		if (!acceptanceKey) acceptanceKey = idempotencyKey();
		const response = await fetch(`/api/v1/buyer-invitations/${page.params.token}/accept`, {
			method: 'POST',
			headers: { 'content-type': 'application/json', 'Idempotency-Key': acceptanceKey },
			body: JSON.stringify({ challenge_id: challengeId, code, full_name: fullName })
		});
		const body = await response.json();
		if (!response.ok) {
			error = body.detail ?? 'We could not check your details.';
			return;
		}
		await goto('/buyer');
	}

	onMount(async () => {
		await loadPreview();
		loading = false;
	});
</script>

<svelte:head>
	<title>Join {preview?.supplier.legal_name ?? 'Kredit'}</title>
</svelte:head>

<main class="shell">
	{#if loading}
		<p>Loading your invitation…</p>
	{:else if preview}
		<section class="panel" aria-labelledby="invite-title">
			<p class="eyebrow">Your private link</p>
			<h1 id="invite-title">{preview.supplier.trading_name || preview.supplier.legal_name} wants to add you as a customer.</h1>
			<p>Check the details below. We will send a six-digit code to make sure this phone or email belongs to you.</p>
			<dl>
				<div><dt>Business</dt><dd>{preview.invitation.proposed_legal_name}</dd></div>
				<div><dt>Type</dt><dd>{preview.invitation.proposed_business_type}</dd></div>
				<div><dt>Address</dt><dd>{preview.invitation.proposed_address}</dd></div>
				<div><dt>Industry</dt><dd>{preview.invitation.proposed_industry}</dd></div>
			</dl>
			{#if !challengeId}
				<button class="primary" on:click={requestCode}>Send me a code</button>
			{:else}
				<label>Full name<input bind:value={fullName} autocomplete="name" /></label>
				<label>Six-digit code<input bind:value={code} inputmode="numeric" autocomplete="one-time-code" maxlength="6" /></label>
				{#if developmentCode}<p class="hint">Development code: {developmentCode}</p>{/if}
				<button class="primary" on:click={accept}>Yes, these details are mine</button>
			{/if}
			{#if error}<p class="error" role="alert">{error}</p>{/if}
		</section>
	{:else}
		<p class="error" role="alert">{error}</p>
	{/if}
</main>

<style>
	.panel { max-width: 42rem; margin: 5rem auto; padding: 2rem; border: 1px solid var(--color-border); border-radius: 1.25rem; background: var(--color-surface); }
	.eyebrow { color: #2738d6; font-weight: 700; text-transform: uppercase; letter-spacing: 0.08em; font-size: 0.78rem; }
	h1 { font-size: clamp(2rem, 6vw, 4rem); line-height: 1; letter-spacing: -0.045em; }
	dl { display: grid; gap: 0.75rem; margin: 2rem 0; }
	dl div { display: flex; justify-content: space-between; gap: 1rem; border-bottom: 1px solid var(--color-border); padding-bottom: 0.5rem; }
	dt { color: var(--color-muted); } dd { margin: 0; font-weight: 700; text-align: right; }
	label { display: grid; gap: 0.35rem; margin: 1rem 0; font-weight: 700; }
	input { border: 1px solid #aaa69e; border-radius: 0.6rem; padding: 0.75rem; }
	button { border: 0; border-radius: 999px; padding: 0.8rem 1.2rem; font-weight: 700; cursor: pointer; }
	.primary { color: white; background: #2738d6; }
	.hint { color: #2738d6; } .error { color: #b42318; }
</style>
