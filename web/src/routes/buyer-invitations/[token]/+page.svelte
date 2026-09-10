<script lang="ts">
	import { page } from '$app/state';
	import { untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { checkedJSON, LatestRequest, record, text, publicError, RequestError } from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';

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

	let preview = $state<Preview | null>(null);
	let challengeId = $state('');
	let developmentCode = $state('');
	let code = $state('');
	let fullName = $state('');
	let loading = $state(true);
	let error = $state('');
	let busy = $state(false);
	const reads = new LatestRequest();
	let acceptance = $state<MutationIntent | null>(null);
	let activeToken = '';
	const invitationPath = () => `/api/v1/buyer-invitations/${encodeURIComponent(activeToken)}`;

	async function loadPreview() {
		const request = reads.begin(); loading = true; error = ''; preview = null;
		try {
			const result = await checkedJSON(invitationPath(), value => {
				const row = record(value), invitation = record(row.invitation), supplier = record(row.supplier);
				for (const key of ['proposed_legal_name','proposed_business_type','proposed_address','proposed_industry','expires_at']) text(invitation[key]);
				if (!Number.isFinite(Date.parse(String(invitation.expires_at)))) throw new Error('Invalid invitation');
				text(supplier.legal_name); text(supplier.trading_name);
				return row as unknown as Preview;
			}, { signal: request.signal });
			if (request.current()) preview = result;
		} catch (cause) {
			if (request.current()) error = cause instanceof RequestError && [404,410].includes(cause.status)
				? 'This invitation has expired, or it is no longer valid. Ask the seller to send you a new link.'
				: publicError(cause, 'this invitation');
		} finally { if (request.current()) loading = false; }
	}
	async function requestCode() {
		if (busy || acceptance?.unresolved) return; const token = activeToken; busy = true; error = '';
		try {
			const result = await checkedJSON(invitationPath()+'/otp', value => {
				const row=record(value); const id=text(row.challenge_id); if(!id)throw new Error('Missing challenge');
				return { id, developmentCode: typeof row.development_code==='string'?row.development_code:'' };
			}, {method:'POST'});
			if (token !== activeToken) return;
			challengeId=result.id; developmentCode=result.developmentCode; code='';
		} catch(cause) { if (token === activeToken) error=publicError(cause,'code delivery'); }
		finally { if (token === activeToken) busy=false; }
	}
	async function accept() {
		if(busy||!challengeId||!/^\d{6}$/.test(code)||!fullName.trim())return;
		const token = activeToken; busy=true;error='';
		try {
			acceptance ??= new MutationIntent('accept-buyer-invitation',invitationPath()+'/accept');
			await acceptance.run({challenge_id:challengeId,code,full_name:fullName.trim()},value=>{
				const row=record(value);text(record(row.user).id);text(record(row.session).id);record(row.portal);return row;
			});
			if (token === activeToken) await goto('/buyer');
		}catch(cause){if (token === activeToken) error=cause instanceof Error?cause.message:'We could not confirm your details.';}
		finally{if (token === activeToken) busy=false;}
	}
	$effect(() => {
		const token = page.params.token ?? '';
		untrack(() => {
			activeToken = token; acceptance = null; challengeId = ''; developmentCode = ''; code = ''; fullName = ''; busy = false;
			void loadPreview();
		});
		return () => reads.cancel();
	});

</script>

<svelte:head>
	<title>Join {preview?.supplier.legal_name ?? 'Kredit'}</title>
</svelte:head>

<main class="shell">
	{#if loading}
		<p>Opening your invitation…</p>
	{:else if preview}
		<section class="panel" aria-labelledby="invite-title">
			<p class="eyebrow">Your private link</p>
			<h1 id="invite-title">{preview.supplier.trading_name || preview.supplier.legal_name} wants to add you as a customer.</h1>
			<p>Check that the details below are correct. Then we will send a six-digit code, to be sure this phone or email really belongs to you.</p>
			<dl>
				<div><dt>Business name</dt><dd>{preview.invitation.proposed_legal_name}</dd></div>
				<div><dt>Business type</dt><dd>{preview.invitation.proposed_business_type}</dd></div>
				<div><dt>Address</dt><dd>{preview.invitation.proposed_address}</dd></div>
				<div><dt>What you sell</dt><dd>{preview.invitation.proposed_industry}</dd></div>
			</dl>
			{#if !challengeId}
				<button class="primary" disabled={busy} onclick={requestCode}>{busy ? 'Sending…' : 'Send me my code'}</button>
			{:else}
				<label>Full name<input disabled={busy} bind:value={fullName} autocomplete="name" /></label>
				<label>The six-digit code we sent you<input disabled={busy} bind:value={code} inputmode="numeric" autocomplete="one-time-code" maxlength="6" /></label>
				<button disabled={busy || !!acceptance?.unresolved} onclick={requestCode}>Send a new code</button>
				{#if developmentCode}<p class="hint">Development code: {developmentCode}</p>{/if}
				<button class="primary" disabled={busy || !fullName.trim() || !/^\d{6}$/.test(code)} onclick={accept}>{busy ? 'Confirming…' : 'Yes, this is my business'}</button>
			{/if}
			{#if error}<p class="error" role="alert">{error}</p>{/if}
		</section>
	{:else}
		<p class="error" role="alert">{error}</p><button onclick={loadPreview}>Try again</button>
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
