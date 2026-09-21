<script lang="ts">
	import { workspaceHref } from '$lib/workspace-navigation';
	import { page } from '$app/state';
	import DisputeDetail from '$lib/components/DisputeDetail.svelte';
</script>

<svelte:head><title>Reported problem — Kredit</title></svelte:head>
<main class="shell workspace">
	<p class="eyebrow">The problem you reported</p>
	<h1>Purchase dispute</h1>
	<DisputeDetail
		endpoint={`/api/v1/buyer/disputes/${encodeURIComponent(page.params.id ?? '')}`}
		backHref={workspaceHref('/workspace/purchases/disputes', page.url)}
	/>
</main>
