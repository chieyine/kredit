<script lang="ts">
	import { onMount } from 'svelte';
	import { productLabel } from '$lib/product-language';
	import FeedbackPrompt from '$lib/components/FeedbackPrompt.svelte';

	let portal: any = null;
	let error = '';

	onMount(async () => {
		const response = await fetch('/api/v1/buyer/me');
		if (!response.ok) {
			error = 'Open the private Kredit link your seller sent you to access this account.';
			return;
		}
		portal = (await response.json()).portal;
	});
</script>

<svelte:head><title>Customer account — Kredit</title></svelte:head>

<main class="shell">
	{#if portal}
		<p class="eyebrow">Your Kredit account</p>
		<h1>{portal.business.legal_name}</h1>
		<p>Signed in as {portal.person.full_name}. Review your sales, payments and permissions here.</p>
		<section class="grid">
			<article><span>Business verification</span><strong>{productLabel(portal.business.status)}</strong></article>
			<article><span>Verification checks</span><strong>{portal.verification_cases.length}</strong></article>
			<article><span>Permissions you have given</span><strong>{portal.consents.length}</strong></article>
		</section>
		<FeedbackPrompt area="buyer" />
	{:else if error}
		<p class="eyebrow">Your Kredit account</p>
		<h1>Use your private link.</h1>
		<p class="error" role="alert">{error}</p>
		<p class="help">Look for the link in the message from your seller. If it has expired, ask them to send a new one. Do not share your sign-in code with anyone.</p>
	{:else}
		<p class="eyebrow">Your Kredit account</p>
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
