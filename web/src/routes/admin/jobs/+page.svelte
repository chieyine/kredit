<script lang="ts">
	import { checkedJSON, publicError, record, rows } from '$lib/api/reliable';
	let error = $state('');
	import { onMount } from 'svelte';
	import { idempotencyKey } from '$lib/api/client';
 import { adminPost } from '$lib/admin-client';
 let actionKey = '';
	import ProtectedActionDialog from '$lib/components/ProtectedActionDialog.svelte';
	let jobs=$state<any[]>([]),message=$state(''),loading=$state(true),selected=$state<any>(null),dialogOpen=$state(false);
	async function load() {
        loading = true; error = '';
        try { jobs = await checkedJSON('/api/v1/ops/jobs', rows('jobs', record)); }
        catch (cause) { error = publicError(cause, 'background jobs'); }
        finally { loading = false; }
    }
	function input(reason:string){return{command_type:'retry_job',target_type:'job',target_id:String(selected.id),reason,expected_version:selected.attempt+1}}
 async function preview(reason:string){const body=await adminPost('/api/v1/ops/commands/preview',input(reason));if(typeof body.command?.impact_preview?.effect!=='string')throw new Error('The impact preview could not be verified.');actionKey=idempotencyKey();return body.command.impact_preview.effect}
 async function confirm(reason:string){actionKey ||= idempotencyKey();await adminPost('/api/v1/ops/commands',input(reason),'POST',actionKey);message='Job safely requeued.';actionKey='';await load();return true}

	onMount(load);
</script>
<svelte:head><title>Operations jobs — Kredit</title></svelte:head>
<main class="shell workspace"><p class="eyebrow">Operations / Jobs</p><h1>Durable work queues.</h1><p class="lede">Payloads remain hidden. Failed work can only be retried after an impact preview, reason, current-version check, recent MFA, and confirmation.</p>{#if message}<p class="notice" role="status">{message}</p>{/if}{#if error}<section role="alert"><p class="error">{error}</p><button type="button" onclick={load}>Try again</button></section>{:else if loading}<p role="status">Loading jobs…</p>{:else}{#each jobs as job}<article><header><strong>{job.kind}</strong><span class="status">{job.state}</span></header><p>{job.queue} · attempt {job.attempt} of {job.max_attempts}</p><code>Job {job.id}</code>{#if ['retryable','discarded','cancelled'].includes(job.state)}<button onclick={()=>{selected=job;dialogOpen=true}}>Preview safe retry</button>{/if}</article>{:else}<section class="empty-state"><h2>No jobs found</h2><p>The queue has no visible work in the selected operational window.</p></section>{/each}{/if}</main>
{#if selected}<ProtectedActionDialog bind:open={dialogOpen} title="Retry this operation safely" description="Kredit will reuse the original idempotency reference. Review provider and ledger status before confirming." confirmLabel="Confirm safe retry" {preview} onconfirm={confirm}/>{/if}
<style>article{padding:1rem;margin:1rem 0;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}header{display:flex;justify-content:space-between;gap:1rem}article button{display:block;margin-top:.75rem;padding:.7rem;border:1px solid var(--color-border);border-radius:.6rem;background:var(--color-surface);color:var(--color-foreground);font:inherit;font-weight:700;cursor:pointer}</style>
