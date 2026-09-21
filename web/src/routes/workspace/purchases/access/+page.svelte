<script lang="ts">
	import { onMount } from 'svelte';
	import { checkedJSON, record, rows, text } from '$lib/api/reliable';
	import { adminPost } from '$lib/admin-client';
	import IdentityChecks from '$lib/components/IdentityChecks.svelte';
	let profiles = $state<{ id: string; name: string; workspace: string }[]>([]),
		business = $state(''),
		name = $state(''),
		consent = $state(false),
		loading = $state(true),
		busy = $state(false),
		error = $state(''),
		message = $state(''),
		terms = $state(''),
		privacy = $state(''),
		identity = $state(''),
		notice = $state('');
	async function load() {
		loading = true;
		error = '';
		consent = false;
		try {
			const d = await checkedJSON('/api/v1/buyer/me/purchasing-access', record);
			profiles = rows('businesses', (value) => {
				const b = record(value);
				return {
					id: text(b.id),
					name: text(b.legal_name),
					workspace: typeof b.workspace_id === 'string' ? b.workspace_id : ''
				};
			})(d);
			const params = new URLSearchParams(location.search);
			business =
				profiles.find((p) => p.id === params.get('business_id') || p.workspace === params.get('organization'))?.id ??
				profiles[0]?.id ??
				'';
			terms = text(record(d.legal_versions).terms_version);
			privacy = text(record(d.legal_versions).privacy_version);
			identity = text(d.identity_notice_version);
			notice = text(d.identity_notice);
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Your access could not be loaded.';
		} finally {
			loading = false;
		}
	}
	async function enroll() {
		if (busy || !business || !consent) return;
		busy = true;
		error = '';
		message = '';
		try {
			await adminPost('/api/v1/buyer/me/purchasing-access', {
				business_id: business,
				full_name: name,
				consents_accepted: consent,
				terms_version: terms,
				privacy_version: privacy,
				identity_notice_version: identity
			});
			message = 'Your profile is ready. Start or refresh verification below.';
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Setup could not be confirmed.';
		} finally {
			busy = false;
		}
	}
	async function verify() {
		if (busy || !business) return;
		busy = true;
		error = '';
		try {
			await adminPost(`/api/v1/buyer/me/verification/refresh?business_id=${encodeURIComponent(business)}`, {});
			message = 'Verification status saved. Refresh saved checks below to continue.';
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Verification could not be refreshed.';
		} finally {
			busy = false;
		}
	}
	onMount(load);
</script>

<svelte:head><title>Staff purchasing setup — Kredit</title></svelte:head>
<main class="shell workspace setup">
	<p class="eyebrow">Purchases</p>
	<h1>Buy on behalf of your business</h1>
	<p>
		Your owner chooses your purchasing permissions. You complete your own identity and authority checks before accepting
		purchase terms.
	</p>
	{#if error}<p role="alert">{error}</p>
		<button disabled={busy} onclick={load}>Refresh access</button>{/if}{#if message}<p role="status">{message}</p>{/if}
	{#if loading}<p role="status">Checking your access…</p>{:else if !profiles.length && !error}<p>
			No business purchasing access is available. Ask your owner to add you to the team and grant purchasing
			permissions.
		</p>{:else if !error}
		<form
			onsubmit={(e) => {
				e.preventDefault();
				void enroll();
			}}
		>
			<fieldset disabled={busy}>
				<p>Acting for <strong>{profiles.find((p) => p.id === business)?.name}</strong></p>
				<label>Your full name<input bind:value={name} required maxlength="200" autocomplete="name" /></label>
				<p>{notice}</p>
				<label class="consent"
					><input type="checkbox" bind:checked={consent} required />I accept the <a href="/legal/terms">terms</a> and
					<a href="/legal/privacy">privacy notice</a>, and consent to identity and business-authority checks.</label
				><button>Set up my purchasing profile</button>
			</fieldset>
		</form>
		<button disabled={busy || !business} onclick={verify}>Start or refresh verification</button>
		<p>
			Creating a profile does not verify your identity or approve a purchase. Existing verified personal details are
			retained.
		</p>
		<a href={`/workspace/purchases?business_id=${encodeURIComponent(business)}`}>Open business purchases →</a>
	{/if}<IdentityChecks />
</main>

<style>
	.setup {
		max-width: 48rem;
	}
	fieldset {
		border: 0;
		padding: 0;
		min-width: 0;
	}
	label {
		display: grid;
		gap: 0.5rem;
		margin: 1rem 0;
	}
	.consent {
		display: block;
		line-height: 1.7;
	}
	.consent input {
		margin-right: 0.5rem;
	}
	input,
	button {
		font: inherit;
		padding: 0.7rem;
		border: 1px solid var(--color-border);
		border-radius: 0.5rem;
		background: var(--color-surface);
		color: inherit;
		max-width: 100%;
		box-sizing: border-box;
	}
	button {
		margin: 0.5rem 0;
	}
	p {
		line-height: 1.7;
	}
</style>
