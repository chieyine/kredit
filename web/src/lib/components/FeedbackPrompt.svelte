<script lang="ts">
	import { onMount } from 'svelte';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';

	let { area, organizationID = '' } = $props<{ area: 'seller' | 'buyer'; organizationID?: string }>();
	let visible = $state(false);
	let busy = $state(false);
	let message = $state('');
	const storageKey = $derived(`kredit-feedback-overview-${area}-${organizationID || 'personal'}`);

	onMount(() => {
		visible = localStorage.getItem(storageKey) !== 'done';
	});

	async function answer(value: 'yes' | 'partly' | 'no') {
		busy = true;
		message = '';
		const response = await fetch('/api/v1/me/product-feedback', {
			method: 'POST',
			credentials: 'include',
			headers: { 'Content-Type': 'application/json', 'Idempotency-Key': idempotencyKey(), ...csrfHeaders() },
			body: JSON.stringify({ area, screen: 'overview', answer: value, ...(organizationID ? { organization_id: organizationID } : {}) })
		});
		const result = await response.json().catch(() => ({}));
		busy = false;
		if (!response.ok) {
			message = result.detail ?? 'We could not save your answer. Please try again.';
			return;
		}
		localStorage.setItem(storageKey, 'done');
		message = 'Thank you. Your answer will help us make Kredit easier.';
		setTimeout(() => { visible = false; }, 2200);
	}
</script>

{#if visible}
	<section class="feedback" aria-labelledby={`feedback-title-${area}`}>
		<div>
			<p class="eyebrow">Help us improve Kredit</p>
			<h2 id={`feedback-title-${area}`}>Was this page easy to understand?</h2>
			{#if message}<p class:error={message.startsWith('We could not')} role="status">{message}</p>{/if}
		</div>
		<div class="answers" aria-label="Choose one answer">
			<button disabled={busy} onclick={() => answer('yes')}>Yes</button>
			<button disabled={busy} onclick={() => answer('partly')}>Partly</button>
			<button disabled={busy} onclick={() => answer('no')}>No</button>
			<button class="later" disabled={busy} onclick={() => { visible = false; }}>Not now</button>
		</div>
	</section>
{/if}

<style>
	.feedback{display:flex;align-items:center;justify-content:space-between;gap:2rem;margin:2.5rem 0 0;padding:1.35rem 1.5rem;border:1px solid #aba79f;border-left:6px solid #e85f3d;background:#fffdf8}.feedback h2{margin:.25rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(1.35rem,3vw,1.8rem);font-weight:500}.feedback p{margin:.3rem 0}.answers{display:flex;flex-wrap:wrap;gap:.55rem}.answers button{min-width:4.5rem;min-height:2.75rem;padding:.6rem .85rem;border:1px solid #17181b;border-radius:0;background:#17181b;color:#fff;font:inherit;font-weight:750;cursor:pointer}.answers button:hover,.answers button:focus-visible{background:#2738d6;border-color:#2738d6}.answers button:disabled{cursor:wait;opacity:.6}.answers .later{color:#17181b;background:transparent;border-color:#aaa69e}.error{color:#b42318}@media(max-width:720px){.feedback{align-items:stretch;flex-direction:column}.answers button{flex:1}}
</style>
