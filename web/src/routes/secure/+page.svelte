<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
    import { checkedJSON, record, text } from '$lib/api/reliable';

	let error = '';

    async function load() {
        error = '';
        try {
            const target = await checkedJSON(`/api/v1/secure-link?${page.url.searchParams.toString()}`, value => text(record(value).redirect_to));
            const destination = new URL(target, location.origin);
            if (destination.origin !== location.origin || !['http:', 'https:'].includes(destination.protocol)) throw new Error('Invalid destination');
            window.location.replace(destination.href);
        } catch {
            error = 'This private link could not be verified. Try again, or ask the person who sent it for a new one.';
        }
    }
    onMount(load);
</script>

<svelte:head><title>Opening secure link — Kredit</title></svelte:head>

<main class="shell prose-page">
	<p class="eyebrow">Private link</p>
	<h1>{error ? 'This link cannot be opened.' : 'Opening your secure Kredit page…'}</h1>
	<p class:error role={error ? 'alert' : undefined}>{error || 'Please wait while the signed link is verified.'}</p>
	{#if error}<button type="button" onclick={load}>Try again</button> <a href="/">Go to the home page</a>{/if}
</main>

