<script lang="ts">
	import Money from './Money.svelte';
	let { action, amountKobo, disabled = false, onconfirm = undefined } = $props<{ action: string; amountKobo: number; disabled?: boolean; onconfirm?: () => void }>();
</script>

<section class="card">
	<h2>Confirm action</h2>
	<p>{action} <strong><Money amountKobo={amountKobo} /></strong>?</p>
	<p class="muted">Check the amount carefully. If your network drops, look at the payment history before you try again. It may have already gone through.</p>
	<button class="primary" type="button" {disabled} onclick={() => onconfirm?.()}>{action}</button>
</section>
