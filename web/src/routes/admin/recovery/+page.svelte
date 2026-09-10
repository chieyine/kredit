<script lang="ts">
	import { checkedJSON, publicError, record, rows } from '$lib/api/reliable';
	let error = $state('');
	import{onMount}from'svelte';import{idempotencyKey}from'$lib/api/client';import{adminPost}from'$lib/admin-client';let actionKey='';import ProtectedActionDialog from'$lib/components/ProtectedActionDialog.svelte';
	let items=$state<any[]>([]),selected=$state<any>(null),decision=$state(''),message=$state(''),dialogOpen=$state(false),loading=$state(true);
	async function load() {
        loading = true; error = '';
        try { items = await checkedJSON('/api/v1/ops/account-recovery?state=PENDING_REVIEW', rows('requests', record)); }
        catch (cause) { error = publicError(cause, 'recovery requests'); }
        finally { loading = false; }
    }
	function begin(item:any,nextDecision:string){selected=item;decision=nextDecision;actionKey='';dialogOpen=true}
 async function confirm(reason:string){actionKey ||= idempotencyKey();await adminPost(`/api/v1/ops/account-recovery/${selected.id}/review`,{decision,reason,expected_version:selected.version},'POST',actionKey);message='Your review has been recorded.';actionKey='';await load();return true}
	onMount(load);
</script>
<svelte:head><title>Account recovery review — Kredit</title></svelte:head>
<main class="shell workspace"><p class="eyebrow">Operations / Recovery</p><h1>Recovery review queue.</h1><p class="lede">To approve one you need independent proof, recent MFA, a different reviewer from whoever raised it, a written reason and a 24-hour cooling-off wait.</p>{#if message}<p class="notice" role="status">{message}</p>{/if}{#if error}<section role="alert"><p class="error">{error}</p><button type="button" onclick={load}>Try again</button></section>{:else if loading}<p role="status">Loading recovery requests…</p>{:else}{#each items as item}<article><header><strong>Recovery {item.id}</strong><span class="status">pending review</span></header><p>{item.independent_factor_count} independent factors · version {item.version}</p><div class="actions"><button onclick={()=>begin(item,'approve')}>Approve with cooling-off</button><button class="danger" onclick={()=>begin(item,'reject')}>Reject</button></div></article>{:else}<section class="empty-state"><h2>Nothing waiting for review</h2><p>A request shows up here once independent proof has been collected.</p></section>{/each}{/if}</main>
{#if selected}<ProtectedActionDialog bind:open={dialogOpen} title={`${decision==='approve'?'Approve':'Reject'} account recovery`} description={decision==='approve'?'A 24-hour cooling-off period will begin and sensitive financial changes remain blocked.':'Access will not be restored. Record the evidence-based reason for rejection.'} confirmLabel={decision==='approve'?'Approve with cooling-off':'Reject recovery'} onconfirm={confirm}/>{/if}
<style>article{margin:1rem 0;padding:1rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}header{display:flex;justify-content:space-between;gap:1rem}.actions{display:flex;flex-wrap:wrap;gap:.5rem}.actions button{padding:.65rem .85rem;border:1px solid var(--color-border);border-radius:.6rem;background:var(--color-surface);color:var(--color-foreground);font:inherit;font-weight:700;cursor:pointer}.actions .danger{color:var(--color-destructive)}</style>
