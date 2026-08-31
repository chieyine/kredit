<script lang="ts">
	import { shareOrCopy, whatsappURL } from '$lib/product-tools';
	let { title = 'Kredit', text, url = '', compact = false } = $props<{ title?: string; text: string; url?: string; compact?: boolean }>();
	let message = $state('');
	async function share() {
		try {
			const result = await shareOrCopy(title, text, url);
			message = result === 'shared' ? 'Shared.' : result === 'copied' ? 'Copied. You can paste it into any message.' : 'Sharing is not available on this phone.';
		} catch { message = 'Sharing was cancelled.'; }
	}
</script>

<div class:compact class="share-actions">
	<a href={whatsappURL([text, url].filter(Boolean).join('\n'))} target="_blank" rel="noreferrer">Send on WhatsApp</a>
	<button type="button" onclick={share}>Share another way</button>
	{#if message}<small role="status">{message}</small>{/if}
</div>

<style>
	.share-actions{display:flex;flex-wrap:wrap;align-items:center;gap:.65rem;margin:.8rem 0}.share-actions a,.share-actions button{display:inline-flex;align-items:center;justify-content:center;min-height:2.75rem;padding:.55rem .8rem;border:1px solid var(--color-foreground);background:var(--color-surface);color:var(--color-foreground);font:inherit;font-weight:750;text-decoration:none}.share-actions a{border-color:#16794e;background:#16794e;color:#fff}.share-actions small{flex-basis:100%;color:var(--color-muted)}.compact{margin:.35rem 0}.compact a,.compact button{min-height:2.5rem;font-size:.82rem}
</style>
