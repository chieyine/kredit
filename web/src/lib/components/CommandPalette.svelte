<script lang="ts">
	import { goto } from '$app/navigation';
	import { tick } from 'svelte';

	let { links, open = $bindable(false) }: { links: [string, string][]; open?: boolean } = $props();

	let query = $state('');
	let selected = $state(0);
	let dialog: HTMLDialogElement = $state()!;
	let searchInput: HTMLInputElement = $state()!;
	let returnFocus: HTMLElement | null = null;

	function show() {
		returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
		open = true;
		query = '';
		selected = 0;
	}

	async function close() {
		if (dialog?.open) dialog.close();
		open = false;
		await tick();
		returnFocus?.focus();
	}

	$effect(() => {
		if (!open) return;
		if (!returnFocus) returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
		void tick().then(() => { dialog?.showModal(); searchInput?.focus(); });
	});

	const results = $derived.by(() => {
		const q = query.trim().toLowerCase();
		if (!q) return links;
		return links.filter(([label, href]) => `${label} ${href}`.toLowerCase().includes(q));
	});

	function onKeydown(event: KeyboardEvent) {
		if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
			event.preventDefault();
			if (open) close(); else show();
			return;
		}
		if (!open) return;
		if (event.key === 'Escape') { event.preventDefault(); close(); }
		else if (event.key === 'ArrowDown') { event.preventDefault(); selected = Math.min(selected + 1, results.length - 1); }
		else if (event.key === 'ArrowUp') { event.preventDefault(); selected = Math.max(selected - 1, 0); }
		else if (event.key === 'Enter' && results[selected]) {
			close();
			goto(results[selected][1]);
		}
	}

	function pick(href: string) {
		close();
		goto(href);
	}

	function trapFocus(event: KeyboardEvent) {
		if (event.key !== 'Tab' || !dialog) return;
		const focusable = Array.from(dialog.querySelectorAll<HTMLElement>('input, button, a[href], select, textarea, [tabindex]:not([tabindex="-1"])')).filter((element) => !element.hasAttribute('disabled'));
		if (!focusable.length) return;
		const first = focusable[0], last = focusable[focusable.length - 1];
		if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
		else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
	}
</script>

<svelte:window onkeydown={onKeydown} />

{#if open}
	<dialog bind:this={dialog} class="palette" aria-label="Go to page" onkeydown={trapFocus} oncancel={(event)=>{event.preventDefault();close()}} onclick={(event)=>{if(event.target===dialog)close()}}>
		<input
			bind:this={searchInput}
			type="search"
			placeholder="Search pages…"
			bind:value={query}
			oninput={() => (selected = 0)}
			aria-label="Search pages"
		/>
		<ul role="listbox">
			{#each results as [label, href], index (href)}
				<li>
					<button
						type="button"
						class:selected={index === selected}
						role="option"
						aria-selected={index === selected}
						onpointerenter={() => (selected = index)}
						onclick={() => pick(href)}
					>{label}<span>{href}</span></button>
				</li>
			{:else}
				<li class="none">No matches.</li>
			{/each}
		</ul>
		<p class="hint"><kbd>↑↓</kbd> navigate · <kbd>↵</kbd> open · <kbd>esc</kbd> close</p>
	</dialog>
{/if}

<style>
	.palette { position: fixed; top: 14vh; width: min(34rem, calc(100vw - 2rem)); max-height:80vh; margin:0 auto; padding:0; overflow: hidden; border: 1px solid var(--color-border); border-radius: var(--radius-lg); background: var(--color-surface); color:var(--color-foreground); box-shadow: var(--shadow-md); }
	.palette[open]{display:grid}.palette::backdrop{background:rgb(16 45 42 / .55)}
	.palette input { border: 0; border-bottom: 1px solid var(--color-border); border-radius: 0; padding: 0.95rem 1.1rem; font: inherit; font-size: 1.05rem; background: transparent; color: inherit; outline: none; }
	.palette ul { max-height: 18rem; margin: 0; padding: 0.35rem; list-style: none; overflow-y: auto; }
	.palette button { display: flex; width: 100%; justify-content: space-between; align-items: center; gap: 1rem; border: 0; border-radius: var(--radius-sm); padding: 0.65rem 0.75rem; background: transparent; color: inherit; font-weight: 650; text-align: left; cursor: pointer; }
	.palette button span { color: var(--color-muted); font-size: 0.8rem; font-weight: 500; }
	.palette button.selected { background: var(--color-surface-muted); color: var(--color-primary); }
	.palette .none { padding: 0.9rem; color: var(--color-muted); }
	.hint { display: flex; gap: 0.75rem; align-items: center; margin: 0; padding: 0.6rem 0.9rem; border-top: 1px solid var(--color-border); background: var(--color-surface-muted); color: var(--color-muted); font-size: 0.78rem; }
	kbd { padding: 0.05rem 0.4rem; border: 1px solid var(--color-border); border-radius: 0.3rem; background: var(--color-surface); font-size: 0.72rem; }
</style>
