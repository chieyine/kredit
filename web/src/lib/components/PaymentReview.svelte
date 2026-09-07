<script lang="ts">
  import Money from './Money.svelte';
  import type { KoboValue } from '$lib/money';
  let { amount, reference, decision, onconfirm, oncancel, busy, error }: { amount: KoboValue; reference: string; decision: 'confirmed' | 'rejected'; onconfirm: () => void; oncancel: () => void; busy: boolean; error: string } = $props();
  let dialog = $state<HTMLDialogElement | null>(null), checked = $state(false);
  $effect(() => { if (dialog && !dialog.open) dialog.showModal(); });
</script>
<dialog bind:this={dialog} aria-labelledby="payment-review-title" oncancel={event => { event.preventDefault(); if (!busy) oncancel(); }}>
  <h2 id="payment-review-title">{decision === 'confirmed' ? 'Confirm money received' : 'Mark transfer not found'}</h2>
  <p class="amount"><Money amountKobo={amount} /></p><p>Transfer reference: <strong>{reference}</strong></p>
  <p>{decision === 'confirmed' ? 'This records a payment and reduces the customer’s balance. Check your bank account, not just a screenshot or debit alert.' : 'This does not record a payment. The customer will be told you could not find the transfer.'}</p>
  <label><input type="checkbox" bind:checked disabled={busy} />I have checked the bank account and transfer details.</label>
  {#if error}<p role="alert">{error}</p>{/if}
  <div class="actions"><button class="secondary" disabled={busy} onclick={oncancel}>Back</button><button class="primary" disabled={busy || !checked} onclick={onconfirm}>{busy ? 'Checking result…' : decision === 'confirmed' ? 'Confirm received' : 'Mark not found'}</button></div>
</dialog>
<style>dialog{box-sizing:border-box;width:min(32rem,calc(100vw - 2rem));max-height:calc(100dvh - 2rem);overflow:auto;padding:1.5rem;border:1px solid #9d9b95;border-radius:.6rem;background:var(--color-surface);color:var(--color-foreground)}dialog:not([open]){display:none}dialog::backdrop{background:rgb(23 24 27 / .55)}h2{font-size:1.4rem;line-height:1.3;margin-top:0}p{line-height:1.6;overflow-wrap:anywhere}.amount{font-size:2rem;font-weight:750;font-variant-numeric:tabular-nums;margin:.5rem 0}label{display:flex;align-items:start;gap:.75rem;line-height:1.5}input{margin-top:.2rem;width:1.25rem;height:1.25rem;min-height:0;flex-shrink:0}.actions{display:flex;justify-content:flex-end;gap:.75rem;margin-top:1.5rem}button{font:inherit;min-height:3rem;padding:.6rem .9rem;border-radius:.35rem}.secondary{background:transparent;border:1px solid var(--color-border)}[role=alert]{padding:.75rem;border-left:3px solid #945010;background:#fff3e0;color:#603f16}</style>
