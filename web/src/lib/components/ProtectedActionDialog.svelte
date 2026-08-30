<script lang="ts">
	import { tick } from 'svelte';
	let { open = $bindable(false), title, description, confirmLabel, previewLabel = 'Preview impact', preview, onconfirm }: { open?: boolean; title: string; description: string; confirmLabel: string; previewLabel?: string; preview?: (reason: string) => Promise<string | undefined>; onconfirm: (reason: string) => Promise<boolean> } = $props();
	let dialog: HTMLDialogElement = $state()!;
	let reason = $state(''), impact = $state(''), error = $state(''), busy = $state(false), wasOpen = false;
	$effect(() => {
		if (open && !wasOpen) { reason='';impact='';error='';busy=false;wasOpen=true;void tick().then(()=>dialog?.showModal()); }
		else if (!open && wasOpen) { wasOpen=false;if(dialog?.open)dialog.close(); }
	});
	function close(){if(!busy)open=false}
	async function proceed(){
		if(reason.trim().length<8){error='Give a specific reason using at least 8 characters.';return}
		busy=true;error='';
		try{if(preview&&!impact){impact=(await preview(reason.trim()))??'';if(!impact)error='The impact preview could not be loaded. Nothing was changed.'}else if(await onconfirm(reason.trim()))open=false}
		catch(cause){error=cause instanceof Error?cause.message:'The action could not be completed.'}finally{busy=false}
	}
</script>
{#if open}
	<dialog bind:this={dialog} aria-labelledby="protected-action-title" oncancel={(event)=>{event.preventDefault();close()}}>
		<form method="dialog" onsubmit={(event)=>{event.preventDefault();proceed()}}>
			<div class="dialog-mark" aria-hidden="true">!</div><h2 id="protected-action-title">{title}</h2><p class="muted">{description}</p>
			<label>Reason for this action<textarea bind:value={reason} rows="4" minlength="8" maxlength="1000" required disabled={busy||Boolean(impact)}></textarea></label>
			{#if impact}<aside aria-live="polite"><strong>Verified impact preview</strong><p>{impact}</p><p>No action has been applied yet.</p></aside>{/if}
			{#if error}<p class="error" role="alert">{error}</p>{/if}
			<div class="actions"><button type="button" onclick={close} disabled={busy}>Cancel</button><button class="primary" disabled={busy||reason.trim().length<8}>{busy?'Working…':preview&&!impact?previewLabel:confirmLabel}</button></div>
		</form>
	</dialog>
{/if}
<style>dialog{width:min(34rem,calc(100vw - 2rem));padding:0;border:1px solid var(--color-border);border-radius:0;background:var(--color-surface);color:var(--color-foreground);box-shadow:var(--shadow-md)}dialog::backdrop{background:rgb(23 24 27 / .68);backdrop-filter:blur(3px)}form{display:grid;gap:1rem;padding:clamp(1.25rem,4vw,2rem)}h2{margin:0;font-family:Georgia,'Times New Roman',serif;font-size:1.65rem;font-weight:500;letter-spacing:-.03em}.dialog-mark{display:grid;place-items:center;width:2.5rem;height:2.5rem;border-radius:0;background:#fff4e5;color:var(--color-warning);font-weight:900}label{display:grid;gap:.4rem;font-weight:750}textarea{box-sizing:border-box;width:100%;padding:.75rem;border:1px solid var(--color-border);border-radius:0;background:var(--color-surface);font:inherit;resize:vertical}aside{padding:1rem;border:1px solid var(--color-border);border-radius:0;background:var(--color-surface-muted)}aside p{margin:.4rem 0 0;color:var(--color-muted);line-height:1.5}.actions{display:flex;justify-content:flex-end;gap:.7rem}.actions button{padding:.65rem 1rem;border:1px solid var(--color-border);border-radius:0;background:var(--color-surface);color:var(--color-foreground);font:inherit;font-weight:750;cursor:pointer}.actions .primary{border:0;background:var(--color-primary);color:#fff}</style>
