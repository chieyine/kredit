<script lang="ts">import StatusPill from './StatusPill.svelte'; let { name, status = 'pending' } = $props<{ name: string; status?: string }>();</script>
<section class="card"><h2>{name}</h2><StatusPill {status} /><p>Only people allowed to use this account can see these details.</p></section>
