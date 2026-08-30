<script lang="ts">import Money from './Money.svelte'; import StatusPill from './StatusPill.svelte'; let { status, reason, blockedKobo = 0 } = $props<{ status: string; reason: string; blockedKobo?: number }>();</script>
<section class="card"><h2>Reported problem</h2><StatusPill {status} /><p>{reason}</p>{#if blockedKobo}<p>Payment paused: <Money amountKobo={blockedKobo} /></p>{/if}</section>
