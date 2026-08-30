<script lang="ts">import Money from './Money.svelte'; let { amountKobo, label = 'Outstanding' } = $props<{ amountKobo: number; label?: string }>();</script>
<div><span>{label}</span><strong><Money {amountKobo} /></strong></div>
