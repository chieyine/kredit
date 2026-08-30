<script lang="ts">
	import { page } from '$app/state';

	const notFound = $derived(page.status === 404);
	const title = $derived(notFound ? 'Page not found' : 'Something went wrong');
	const detail = $derived(notFound
		? 'The link may be expired, mistyped, or no longer available.'
		: 'Your action was not submitted again. Return to a safe page and retry only after checking its current status.');
</script>

<svelte:head>
	<title>{title} — Kredit</title>
</svelte:head>

<main class="error-page">
	<div class="mark" aria-hidden="true">{notFound ? '404' : '!'}</div>
	<p class="eyebrow">{notFound ? 'Page not found' : `Problem ${page.status}`}</p>
	<h1>{title}</h1>
	<p class="lede">{detail}</p>
	<div class="actions">
		<a class="primary" href="/">Go to Kredit home</a>
		<button type="button" onclick={() => history.back()}>Go back</button>
	</div>
	<p class="support">Still stuck? <a href="/legal/complaints">Ask for help</a>. Share the page address, but never send your password, one-time code or full bank details.</p>
</main>

<style>
	.error-page{max-width:44rem;min-height:70vh;margin:0 auto;padding:clamp(4rem,12vw,9rem) 1.25rem}.mark{display:grid;place-items:center;width:4rem;height:4rem;border-radius:1.25rem;background:var(--color-surface-muted);color:var(--color-primary);font-size:1.2rem;font-weight:900;box-shadow:var(--shadow-sm)}h1{margin:.6rem 0 1rem;font-size:clamp(3rem,10vw,6rem);line-height:.95;letter-spacing:-.06em}.actions{display:flex;flex-wrap:wrap;gap:.75rem;margin:2rem 0}.actions button{padding:.65rem 1rem;border:1px solid var(--color-border);border-radius:999px;background:var(--color-surface);color:var(--color-foreground);font:inherit;font-weight:700;cursor:pointer}.support{padding-top:1.5rem;border-top:1px solid var(--color-border);color:var(--color-muted);line-height:1.6}.support a{color:var(--color-primary);font-weight:700}
</style>
