<script lang="ts">import StatusPill from './StatusPill.svelte'; let { legalName, state = 'pending' } = $props<{ legalName: string; state?: string }>();</script>
<section class="card"><h2>{legalName}</h2><p>Business check</p><StatusPill status={state} /></section>
