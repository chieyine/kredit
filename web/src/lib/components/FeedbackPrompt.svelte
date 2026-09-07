<script lang="ts">
	import { getContext, untrack } from 'svelte';
	import { browser } from '$app/environment';
	import { ACCOUNT_CONTEXT, type AccountContext } from '$lib/account-context';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';

	let { area, organizationID = '' } = $props<{ area: 'seller' | 'buyer'; organizationID?: string }>();
	const account = getContext<AccountContext | undefined>(ACCOUNT_CONTEXT);
	let visible = $state(false), busy = $state(false), completed = $state(false), message = $state('');
	let requestKey = '';
	const storageKey = $derived(`kredit-feedback-v2:${account?.userID ?? ''}:${area}:${organizationID || 'personal'}`);
	$effect(() => {
		const key = storageKey;
		if (!browser) return;
		untrack(() => {
			busy = false; completed = false; message = ''; requestKey = ''; visible = true;
			try { visible = localStorage.getItem(key) !== 'done'; } catch { /* Feedback works without browser storage. */ }
		});
	});
	async function answer(value: 'yes' | 'partly' | 'no') {
		if (busy || completed) return;
		const key = storageKey;
		busy = true; message = '';
		try {
			requestKey ||= idempotencyKey();
			const response = await fetch('/api/v1/me/product-feedback', {
				method: 'POST', credentials: 'include',
				headers: { 'Content-Type': 'application/json', 'Idempotency-Key': requestKey, ...csrfHeaders() },
				body: JSON.stringify({ area, screen: 'overview', answer: value, ...(organizationID ? { organization_id: organizationID } : {}) })
			});
			if (storageKey !== key) return;
			if (!response.ok) {
				if (response.status >= 400 && response.status < 500 && response.status !== 409) requestKey = '';
				throw new Error('We could not confirm your answer was saved. Please try again.');
			}
			try { localStorage.setItem(key, 'done'); } catch { /* The server already accepted the answer. */ }
			completed = true; message = 'Thank you. Your answer helps us improve Kredit.';
		} catch {
			if (storageKey === key) message = 'We could not confirm your answer was saved. Please try again.';
		} finally {
			if (storageKey === key) busy = false;
		}
	}
</script>

{#if visible}
	<section class="feedback" aria-labelledby={`feedback-title-${area}`}>
		<div>
			<p class="eyebrow">Help us make Kredit better</p>
			<h2 id={`feedback-title-${area}`}>Was this page easy to understand?</h2>
			{#if message}<p class:error={message.startsWith('We could not')} role="status">{message}</p>{/if}
		</div>
		<div class="answers" aria-label="Choose one answer">
			<button disabled={busy || completed} onclick={() => answer('yes')}>Yes</button>
			<button disabled={busy || completed} onclick={() => answer('partly')}>Partly</button>
			<button disabled={busy || completed} onclick={() => answer('no')}>No</button>
			<button class="later" disabled={busy} onclick={() => { visible = false; }}>{completed ? 'Close' : 'Not now'}</button>
		</div>
	</section>
{/if}

<style>
	.feedback{display:flex;align-items:center;justify-content:space-between;gap:2rem;margin:2.5rem 0 0;padding:1.35rem 1.5rem;border:1px solid #aba79f;border-left:6px solid #e85f3d;background:#fffdf8}.feedback h2{margin:.25rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(1.35rem,3vw,1.8rem);font-weight:500}.feedback p{margin:.3rem 0}.answers{display:flex;flex-wrap:wrap;gap:.55rem}.answers button{min-width:4.5rem;min-height:2.75rem;padding:.6rem .85rem;border:1px solid #17181b;border-radius:0;background:#17181b;color:#fff;font:inherit;font-weight:750;cursor:pointer}.answers button:hover,.answers button:focus-visible{background:#2738d6;border-color:#2738d6}.answers button:disabled{cursor:wait;opacity:.6}.answers .later{color:#17181b;background:transparent;border-color:#aaa69e}.error{color:#b42318}@media(max-width:720px){.feedback{align-items:stretch;flex-direction:column}.answers button{flex:1}}
</style>
