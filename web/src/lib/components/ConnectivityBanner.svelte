<script lang="ts">
	import { onMount } from 'svelte';

	let online = $state(true);
	let restored = $state(false);
	let restoreTimer: ReturnType<typeof setTimeout> | undefined;

	function update(next: boolean) {
		const wasOffline = !online;
		online = next;
		if (next && wasOffline) {
			restored = true;
			if (restoreTimer) clearTimeout(restoreTimer);
			restoreTimer = setTimeout(() => (restored = false), 4000);
		} else if (!next) {
			restored = false;
		}
	}

	onMount(() => {
		online = navigator.onLine;
		const handleOnline = () => update(true);
		const handleOffline = () => update(false);
		window.addEventListener('online', handleOnline);
		window.addEventListener('offline', handleOffline);
		return () => {
			window.removeEventListener('online', handleOnline);
			window.removeEventListener('offline', handleOffline);
			if (restoreTimer) clearTimeout(restoreTimer);
		};
	});
</script>

{#if !online}
	<div class="connectivity offline" role="status" aria-live="polite">
		<strong>You are offline.</strong>
		<span>New actions cannot be sent. An earlier request may still be processing. Check its status after reconnecting.</span>
	</div>
{:else if restored}
	<div class="connectivity restored" role="status" aria-live="polite">
		<strong>Connection restored.</strong>
		<span>Before you try anything again, check whether it already went through.</span>
	</div>
{/if}

<style>
	.connectivity{position:sticky;top:0;z-index:40;display:flex;justify-content:center;gap:.6rem;padding:.55rem 1rem;border-bottom:1px solid currentColor;font-size:.9rem;line-height:1.45}.connectivity strong{white-space:nowrap}.offline{background:#fff1d6;color:#714400}.restored{background:#e8f6ed;color:#155f3d}@media(max-width:640px){.connectivity{display:grid;gap:.1rem}.connectivity strong{white-space:normal}}
</style>
