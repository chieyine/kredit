<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';

	let error = '';

	onMount(async () => {
		const response = await fetch(`/api/v1/secure-link?${page.url.searchParams.toString()}`, {
			credentials: 'include',
			cache: 'no-store'
		});
		const body = await response.json().catch(() => ({}));
		if (!response.ok || typeof body.redirect_to !== 'string') {
			error = body.detail ?? 'This private link is invalid or has expired.';
			return;
		}
		window.location.replace(body.redirect_to);
	});
</script>

<svelte:head><title>Opening secure link — Kredit</title></svelte:head>

<main class="shell prose-page">
	<p class="eyebrow">Private link</p>
	<h1>{error ? 'This link cannot be opened.' : 'Opening your secure Kredit page…'}</h1>
	<p class:error role={error ? 'alert' : undefined}>{error || 'Please wait while the signed link is verified.'}</p>
	{#if error}<a href="/">Return to Kredit</a>{/if}
</main>

