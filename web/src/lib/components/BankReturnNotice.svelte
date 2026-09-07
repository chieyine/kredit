<script lang="ts">
  import { getContext, onMount } from 'svelte';
  import { page } from '$app/state';
  import { ACCOUNT_CONTEXT, type AccountContext } from '$lib/account-context';
  const account = getContext<AccountContext>(ACCOUNT_CONTEXT);
  let requestID = $state('');
  onMount(() => {
    try {
      const key = `kredit.bank-return.${account.userID}`, raw = sessionStorage.getItem(key);
      if (!raw) return;
      const value = JSON.parse(raw);
      if (typeof value.requestID !== 'string' || !/^[A-Za-z0-9_-]{1,128}$/.test(value.requestID) || !Number.isFinite(value.expiresAt) || value.expiresAt <= Date.now() || value.expiresAt > Date.now() + 3600000) { sessionStorage.removeItem(key); return; }
      requestID = value.requestID;
    } catch { /* The customer can still find the sale in their account. */ }
  });
</script>
{#if requestID && page.url.pathname !== `/buyer/credit-requests/${requestID}`}
  <aside class="bank-return" aria-label="Continue bank permission"><div><strong>Continue your sale</strong><p>Returning from the bank-permission page does not confirm approval. Open the sale to check its current status.</p></div><a href={`/buyer/credit-requests/${encodeURIComponent(requestID)}`}>Check sale and permission</a></aside>
{/if}
<style>.bank-return{display:flex;align-items:center;justify-content:space-between;gap:1rem;max-width:70rem;margin:1rem auto;padding:1rem;background:#eef0ff;border:1px solid #b5bdf1;color:#23308d;border-radius:.4rem}.bank-return p{margin:.35rem 0;line-height:1.6;font-size:.95rem}.bank-return a{display:inline-flex;align-items:center;min-height:3rem;flex-shrink:0;padding:.5rem .8rem;color:inherit;border:1px solid currentColor;border-radius:.3rem;font-weight:650}@media(max-width:600px){.bank-return{flex-direction:column;align-items:stretch;margin:1rem}}</style>
