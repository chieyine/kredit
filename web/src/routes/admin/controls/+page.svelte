<script lang="ts">
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	let form: HTMLFormElement;
	let targetType = $state('user'), targetId = $state(''), organizationId = $state('');
	let scope = $state('all_sensitive'), expiresAt = $state(''), version = $state(1), reason = $state('');
	let command=$state('suspend_user');
 let preview: any = $state(null), message = $state(''), ready = $state(false);
	onMount(() => {targetType=page.url.searchParams.get('target_type')||targetType;if(targetType==='collection')command='resolve_unknown_submission';targetId=page.url.searchParams.get('target_id')||'';organizationId=page.url.searchParams.get('organization_id')||'';version=Number(page.url.searchParams.get('version')||1);ready = true;});
	function payload() {
		const command = new FormData(form).get('command')?.toString() ?? 'suspend_user';
		return { command_type: command, target_type: targetType, target_id: targetId, organization_id: organizationId || undefined, scope, expires_at: expiresAt ? new Date(expiresAt).toISOString() : undefined, expected_version: Number(version), reason };
	}
	async function inspect() {
		message = '';
		const response = await fetch('/api/v1/ops/commands/preview', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json', ...csrfHeaders() }, body: JSON.stringify(payload()) });
		const data = await response.json(); preview = response.ok ? data.command : null; message = response.ok ? 'Review the impact below before applying.' : data.detail ?? 'Preview failed.';
	}
	async function apply() {
		if (!preview) return;
		const response = await fetch('/api/v1/ops/commands', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json', 'Idempotency-Key': idempotencyKey(), ...csrfHeaders() }, body: JSON.stringify(payload()) });
		const data = await response.json(); message = response.ok ? `Applied. Command ${data.command.id}` : data.detail ?? 'Command failed.'; if (response.ok) preview = null;
	}
</script>

<svelte:head><title>Protected controls — Kredit</title></svelte:head>
<main class="shell workspace">
	<p class="eyebrow">Operations / Controls</p><h1>Protected account and financial controls.</h1>
	<p>Every action needs recent MFA, an explicit permission, current version, structured reason, impact preview, immutable audit record, and affected-user notification.</p>
	<form bind:this={form} data-ready={ready} onsubmit={(event) => { event.preventDefault(); inspect(); }}>
		<label>Action<select name="command" bind:value={command} onchange={() => preview = null}><option value="suspend_user">Suspend a user</option><option value="restore_user">Restore a user</option><option value="suspend_organization">Suspend a business</option><option value="restore_organization">Restore a business</option><option value="place_risk_hold">Place a temporary hold</option><option value="lift_risk_hold">Remove a temporary hold</option><option value="request_reconciliation">Ask for a money check</option><option value="resolve_unknown_submission">Resolve an unknown bank result</option><option value="retry_collection">Retry a bank debit</option><option value="cancel_collection">Cancel a bank debit</option></select></label>
		<label>Target type<input bind:value={targetType} required /></label><label>Target ID<input bind:value={targetId} required /></label>
		<label>Organization ID (when known)<input bind:value={organizationId}/></label><label>Current version<input type="number" min="1" bind:value={version}/></label>
		<label>Scope (risk holds only)<select bind:value={scope}><option>all_sensitive</option><option>credit</option><option>release</option><option>collection</option><option>settlement</option></select></label>
		<label>Expires (risk holds only)<input type="datetime-local" bind:value={expiresAt}/></label><label>Structured reason<textarea bind:value={reason} minlength="8" required></textarea></label>
		<button type="submit" class="primary">Preview impact</button>
	</form>
	{#if message}<p aria-live="polite">{message}</p>{/if}
	{#if preview}<section><h2>Impact preview</h2><p>{preview.impact_preview.effect}</p><dl><dt>Current version</dt><dd>{preview.current_version}</dd><dt>User notification</dt><dd>{preview.impact_preview.will_notify ? 'Required' : 'No'}</dd><dt>Audit</dt><dd>{preview.impact_preview.audit}</dd></dl><button type="button" class="primary" onclick={apply}>Apply protected command</button></section>{/if}
</main>
<style>form{display:grid;gap:1rem;max-width:42rem}label{display:grid;gap:.35rem}input,select,textarea,button{padding:.75rem;border:1px solid var(--color-border);border-radius:.6rem}section{margin-top:1.5rem;padding:1rem;border:1px solid var(--color-border);border-radius:1rem}</style>
