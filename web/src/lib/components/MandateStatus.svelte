<script lang="ts">import StatusPill from './StatusPill.svelte'; let { status, provider } = $props<{ status: string; provider?: string }>();</script>
<div><StatusPill {status} />{#if provider}<small>Payment company: {provider}</small>{/if}</div>
