<script lang="ts">
	import { getContext, untrack } from 'svelte';
	import { browser } from '$app/environment';
	import { ACCOUNT_CONTEXT, type AccountContext } from '$lib/account-context';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';
	import { boundedFetch } from '$lib/api/reliable';

	let { area, organizationID = '' } = $props<{ area: 'seller' | 'buyer'; organizationID?: string }>();
	const account = getContext<AccountContext | undefined>(ACCOUNT_CONTEXT);
	let visible = $state(false),
		busy = $state(false),
		completed = $state(false),
		message = $state('');
	let requestKey = '';
	let pendingAnswer: 'yes' | 'partly' | 'no' | null = $state(null);
	const storageKey = $derived(`kredit-feedback-v2:${account?.userID ?? ''}:${area}:${organizationID || 'personal'}`);
	$effect(() => {
		const key = storageKey;
		if (!browser) return;
		untrack(() => {
			busy = false;
			completed = false;
			message = '';
			requestKey = '';
			pendingAnswer = null;
			visible = true;
			try {
				visible = localStorage.getItem(key) !== 'done';
			} catch {
				/* Feedback works without browser storage. */
			}
		});
	});
	async function answer(value: 'yes' | 'partly' | 'no') {
		if (busy || completed) return;
		if (pendingAnswer && pendingAnswer !== value) {
			message = 'We could not confirm your earlier answer. Retry the same answer to check its result.';
			return;
		}
		const key = storageKey;
		busy = true;
		message = '';
		try {
			requestKey ||= idempotencyKey();
			pendingAnswer = value;
			const response = await boundedFetch('/api/v1/me/product-feedback', {
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json', 'Idempotency-Key': requestKey, ...csrfHeaders() },
				body: JSON.stringify({
					area,
					screen: 'overview',
					answer: value,
					...(organizationID ? { organization_id: organizationID } : {})
				})
			});
			if (storageKey !== key) return;
			if (!response.ok) {
				if (response.status >= 400 && response.status < 500 && response.status !== 409) {
					requestKey = '';
					pendingAnswer = null;
				}
				throw new Error('We could not confirm your answer was saved. Please try again.');
			}
			try {
				localStorage.setItem(key, 'done');
			} catch {
				/* The server already accepted the answer. */
			}
			completed = true;
			message = 'Thank you. Your answer helps us improve Kredit.';
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
			<p class="eyebrow">Quick question</p>
			<h2 id={`feedback-title-${area}`}>Was this page easy to understand?</h2>
			{#if message}<p class:error={message.startsWith('We could not')} role="status">{message}</p>{/if}
		</div>
		<div class="answers" aria-label="Choose one answer">
			<button disabled={busy || completed} onclick={() => answer('yes')}>Yes</button>
			<button disabled={busy || completed} onclick={() => answer('partly')}>Partly</button>
			<button disabled={busy || completed} onclick={() => answer('no')}>No</button>
			<button
				class="later"
				disabled={busy}
				onclick={() => {
					visible = false;
				}}>{completed ? 'Close' : 'Not now'}</button
			>
		</div>
	</section>
{/if}

<style>
	/* A question in passing, not a banner: it sits under the page's real work
	   and reads quieter than any action on it. */
	.feedback {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 2rem;
		margin: 3rem 0 0;
		padding: 1.1rem 0 0;
		border-top: 1px solid var(--color-border);
	}
	.feedback h2 {
		margin: 0.15rem 0;
		font-family: var(--font-sans);
		font-size: 1rem;
		font-weight: 600;
	}
	.feedback p {
		margin: 0.2rem 0;
	}
	.answers {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
	}
	.answers button {
		min-width: 4rem;
		min-height: 2.5rem;
		padding: 0.45rem 0.85rem;
		border: 1px solid var(--color-border-strong);
		background: var(--color-surface);
		color: var(--color-foreground);
		font: inherit;
		font-size: 0.92rem;
		font-weight: 550;
		cursor: pointer;
		transition:
			border-color 180ms ease,
			background-color 180ms ease;
	}
	.answers button:hover:not(:disabled),
	.answers button:focus-visible {
		border-color: var(--color-primary);
		background: var(--color-surface-muted);
	}
	.answers button:disabled {
		cursor: wait;
		opacity: 0.6;
	}
	.answers .later {
		border-color: transparent;
		background: transparent;
		color: var(--color-muted);
	}
	.error {
		color: var(--color-overdue);
	}
	@media (max-width: 720px) {
		.feedback {
			align-items: stretch;
			flex-direction: column;
			gap: 0.9rem;
		}
		.answers button {
			flex: 1;
		}
	}
</style>
