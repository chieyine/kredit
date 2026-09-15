<script lang="ts">import AuthGate from '$lib/components/AuthGate.svelte';import DSA from '$lib/components/DSA.svelte';</script>
<svelte:head><title>Field agent rewards — Kredit</title></svelte:head>
<AuthGate area="field agent account"><DSA/></AuthGate>
