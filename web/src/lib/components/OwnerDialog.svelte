<script lang="ts">
	import { tick, type Snippet } from 'svelte';

	// One modal for the owner console. It uses the platform's own <dialog>, which
	// brings the focus trap, the inert background and the Escape key with it —
	// the hand-rolled overlay this replaces had none of those, and it guarded
	// secrets and ownership.
	let {
		open = $bindable(false),
		title,
		description = '',
        busy = false,
		children,
		onclose
	}: {
		open?: boolean;
		title: string;
		description?: string;
        busy?: boolean;
		children: Snippet;
		onclose?: () => void;
	} = $props();

	let dialog = $state<HTMLDialogElement | null>(null);
	let returnFocus: HTMLElement | null = null;
	let wasOpen = false;
	const titleID = $props.id();

	function close() {
        if (busy) return;
		open = false;
		onclose?.();
	}

	$effect(() => {
		if (open && !wasOpen) {
			wasOpen = true;
			returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
			void tick().then(() => {
				if (!open || !dialog?.isConnected) return;
                if (!dialog.open) dialog.showModal();
				// Start on the first control rather than on the dialog itself, so the
				// first Tab does not jump past the form.
				dialog?.querySelector<HTMLElement>('input, select, textarea, button:not(.close-btn)')?.focus({ preventScroll: true });
			});
		} else if (!open && wasOpen) {
			wasOpen = false;
			if (dialog?.open) dialog.close();
			returnFocus?.focus();
			returnFocus = null;
		}
	});
</script>

{#if open}
	<dialog
		bind:this={dialog}
		aria-labelledby={titleID}
		oncancel={(event) => { event.preventDefault(); close(); }}
		onclick={(event) => { if (event.target === dialog) close(); }}
	>
		<header>
			<div>
				<h2 id={titleID}>{title}</h2>
				{#if description}<p>{description}</p>{/if}
			</div>
			<button class="close-btn" type="button" aria-label="Close" disabled={busy} onclick={close}>✕</button>
		</header>
		<div class="body">{@render children()}</div>
	</dialog>
{/if}

<style>
	dialog {
        display: flex;
        flex-direction: column;
        overflow: hidden;
		box-sizing: border-box;
		width: min(38rem, calc(100vw - 2rem));
		max-height: calc(100dvh - 3rem);
		padding: 0;
		border: 1px solid var(--color-border);
		background: var(--color-surface);
		color: var(--color-foreground);
	}
	dialog:not([open]) { display: none; }
	dialog::backdrop { background: rgb(23 24 27 / 0.55); }
	header {
        flex-shrink: 0;
		display: flex;
		align-items: start;
		justify-content: space-between;
		gap: 1rem;
		padding: 1.25rem 1.4rem 1rem;
		border-bottom: 1px solid var(--color-border);
	}
	h2 { margin: 0; font-size: 1.25rem; line-height: 1.3; }
	header p { margin: 0.4rem 0 0; color: var(--color-muted); line-height: 1.6; }
	.close-btn {
		display: grid;
		place-items: center;
		width: 2.75rem;
		height: 2.75rem;
		flex-shrink: 0;
		border: 1px solid var(--color-border);
		background: transparent;
		color: inherit;
		font-size: 1.1rem;
		cursor: pointer;
	}
	.body { min-height: 0; overflow-y: auto; padding: 1.25rem 1.4rem 1.4rem; }
</style>
