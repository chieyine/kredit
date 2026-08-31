<script lang="ts">
	import { onMount } from 'svelte';
	import { productLabel } from '$lib/product-language';
	import FeedbackPrompt from '$lib/components/FeedbackPrompt.svelte';

	let portal: any = null;
	let error = '';

	onMount(async () => {
		const response = await fetch('/api/v1/buyer/me');
		if (!response.ok) {
			error = 'Open the private link the seller sent you.';
			return;
		}
		portal = (await response.json()).portal;
	});
</script>

<svelte:head><title>Buyer portal — Kredit</title></svelte:head>

<main class="shell">
	{#if portal}
		<p class="eyebrow">Your customer account</p>
		<h1>{portal.business.legal_name}</h1>
		<p>Signed in as {portal.person.full_name}</p>
		<section class="grid">
			<article><span>Your business check</span><strong>{productLabel(portal.business.status)}</strong></article>
			<article><span>Checks completed</span><strong>{portal.verification_cases.length}</strong></article>
			<article><span>Rules you agreed to</span><strong>{portal.consents.length}</strong></article>
		</section>
		<FeedbackPrompt area="buyer" />
	{:else if error}
		<p class="eyebrow">Your customer account</p>
		<h1>Open your private link.</h1>
		<p class="error" role="alert">{error}</p>
		<p class="help">The link is in the message from the seller. If it has expired, ask the seller to send another one.</p>
	{:else}
		<p class="eyebrow">Your customer account</p>
		<h1>Opening your account…</h1>
	{/if}
</main>

<style>
	.eyebrow { color: #2738d6; font-weight: 700; text-transform: uppercase; letter-spacing: 0.08em; font-size: 0.78rem; }
	h1 { font-size: clamp(2.5rem, 7vw, 5rem); line-height: 1; letter-spacing: -0.055em; }
	.grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 1rem; margin-top: 3rem; }
	article { display: grid; gap: 0.75rem; padding: 1.25rem; border: 1px solid var(--color-border); border-radius: 1rem; background: var(--color-surface); }
	article span { color: var(--color-muted); } article strong { font-size: 1.25rem; }
	.error { color: #b42318; }
	.help { max-width: 34rem; color: var(--color-muted); line-height: 1.65; }
	@media (max-width: 720px) { .grid { grid-template-columns: 1fr; } }
</style>
