<script lang="ts">
	import { MutationIntent } from '$lib/api/mutation';
 import { record, text } from '$lib/api/reliable';
 import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
 const intent=new MutationIntent('admin-controls','/api/v1/ops/commands');
	import { adminPost, lagosISO } from '$lib/admin-client';
 import { onMount } from 'svelte';
	import { page } from '$app/state';
	let form: HTMLFormElement;
	let targetType = $state('user'), targetId = $state(''), organizationId = $state('');
	let scope = $state('all_sensitive'), expiresAt = $state(''), version = $state(1), reason = $state('');
	let command=$state('suspend_user');
 let reviewed: ReturnType<typeof payload> | null = $state(null);
 let preview: any = $state(null), message = $state(''), ready = $state(false), busy = $state(false);
	onMount(() => {
		targetType=page.url.searchParams.get('target_type')||targetType;
		const suspended=page.url.searchParams.get('status')==='suspended';
		if(targetType==='document') command='retry_document_scan';
		else if(targetType==='collection') command='resolve_unknown_submission';
		else if(targetType==='organization') command=suspended?'restore_organization':'suspend_organization';
		else if(targetType==='user') command=suspended?'restore_user':'suspend_user';
		targetId=page.url.searchParams.get('target_id')||'';
		organizationId=page.url.searchParams.get('organization_id')||'';
		const requestedVersion=Number(page.url.searchParams.get('version')||1);
		version=Number.isSafeInteger(requestedVersion)&&requestedVersion>0?requestedVersion:1;
		ready = true;
	});
	function payload() {
		const command = new FormData(form).get('command')?.toString() ?? 'suspend_user';
		return { command_type: command, target_type: targetType, target_id: targetId, organization_id: organizationId || undefined, scope, expires_at: expiresAt ? lagosISO(expiresAt) : undefined, expected_version: Number(version), reason };
	}
	async function inspect() {
		if (busy) return;
		busy = true;
		message = '';
		try {
            const candidate = payload();
            const data = await adminPost('/api/v1/ops/commands/preview', candidate);
            if (!data.command?.impact_preview || typeof data.command.impact_preview.effect !== 'string') throw new Error('Preview unavailable');
            if(!Number.isSafeInteger(data.command.current_version)||data.command.current_version<1)throw new Error('Preview version unavailable');
            version=data.command.current_version; reviewed = {...candidate,expected_version:version}; preview = data.command;
            message = 'Read the impact below before you apply this.';
		} catch {
			preview = null;
			message = 'The preview could not be prepared. Nothing was changed.';
		} finally {
			busy = false;
		}
	}

	// One click is one command. These actions retry a bank debit and suspend
	// accounts, so a second click must never become a second effect: the request
	// identity is created once for the reviewed preview and reused on retry.
	async function apply() {
		if (!preview || !reviewed || busy) return;
        const candidate=reviewed;
		busy = true;
		message = '';
		try {
            const data = await intent.run(candidate, value => {const response=record(value),command=record(response.command);text(command.id);if(command.state!=='APPLIED'||command.command_type!==candidate.command_type||command.target_id!==candidate.target_id)throw new Error('Result unavailable');return {id:text(command.id)};});
            message = `Done. Reference ${data.id}`;
            preview = null; reviewed = null;
		} catch (cause) {
			message = cause instanceof Error ? cause.message : 'We have not confirmed the result. Check the record before applying this again.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Protected controls — Kredit</title></svelte:head>
<main class="shell workspace">
	<p class="eyebrow">Operations / Controls</p><h1>Protected account and financial controls.</h1>
	<p>Before any action here goes through, all of this must be true: you signed in with MFA recently, you hold the exact permission, the record is on its current version, you gave a clear reason, you looked at the impact preview and the impact preview explains whether a user notification is expected. Every action is recorded permanently.</p>
	<VerifyIdentity/>
	<form oninput={() => { preview = null; reviewed = null; }} bind:this={form} data-ready={ready} onsubmit={(event) => { event.preventDefault(); inspect(); }}>
        <fieldset disabled={busy}>
		<label>Action<select name="command" bind:value={command} onchange={() => { preview = null; if(command==='retry_document_scan')targetType='document'; }}><option value="retry_document_scan">Retry a quarantined document scan</option><option value="suspend_user">Suspend a user</option><option value="restore_user">Restore a user</option><option value="suspend_organization">Suspend a business</option><option value="restore_organization">Restore a business</option><option value="place_risk_hold">Place a temporary hold</option><option value="lift_risk_hold">Remove a temporary hold</option><option value="request_reconciliation">Ask for a money check</option><option value="resolve_unknown_submission">Resolve an unknown bank result</option><option value="retry_collection">Retry a bank debit</option><option value="cancel_collection">Cancel a bank debit</option></select></label>
		<label>Target type<input bind:value={targetType} required /></label><label>Target ID<input bind:value={targetId} required /></label>
		<label>Organization ID (when known)<input bind:value={organizationId}/></label><label>Current version<input type="number" min="1" bind:value={version}/></label>
		<label>Scope (risk holds only)<select bind:value={scope}><option>all_sensitive</option><option>credit</option><option>release</option><option>collection</option><option>settlement</option></select></label>
		<label>Expires (Lagos time; risk holds only)<input type="datetime-local" bind:value={expiresAt}/></label><label>Structured reason<textarea bind:value={reason} minlength="8" required></textarea></label>
		<button type="submit" class="primary" disabled={busy}>{busy ? 'Checking…' : 'Preview impact'}</button>
        </fieldset>
	</form>
	{#if message}<p aria-live="polite">{message}</p>{/if}
	{#if preview}<section><h2>Impact preview</h2><p>{preview.impact_preview.effect}</p><dl><dt>Current version</dt><dd>{preview.current_version}</dd><dt>User notification</dt><dd>{preview.impact_preview.will_notify ? 'Required' : 'No'}</dd><dt>Audit</dt><dd>{preview.impact_preview.audit}</dd></dl><button type="button" class="primary" disabled={busy} onclick={apply}>{busy ? 'Applying…' : 'Apply this change'}</button></section>{/if}
</main>
<style>fieldset{display:grid;gap:1rem;border:0;padding:0;margin:0;min-width:0}form{display:grid;gap:1rem;max-width:42rem}label{display:grid;gap:.35rem}input,select,textarea,button{padding:.75rem;border:1px solid var(--color-border);border-radius:.6rem}section{margin-top:1.5rem;padding:1rem;border:1px solid var(--color-border);border-radius:1rem}</style>
