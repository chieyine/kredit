<script lang="ts">import Money from './Money.svelte'; let { items = [] } = $props<{ items?: Array<{ label: string; amountKobo: number }> }>();</script>
<dl class="breakdown">{#each items as item}<div><dt>{item.label}</dt><dd><Money amountKobo={item.amountKobo} /></dd></div>{/each}</dl>
