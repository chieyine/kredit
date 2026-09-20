<script lang="ts">
  import { checkedJSON, LatestRequest, publicError, record, rows, text } from '$lib/api/reliable';
  import { saleView, type SaleView } from '$lib/records';
  import { sumKobo } from '$lib/money';
  import Money from './Money.svelte';

  let { organizationID }: { organizationID: string } = $props();
  let loading = $state(true), error = $state(''), businessID = $state('');
  let purchases = $state<SaleView[]>([]);
  const reads = new LatestRequest();
  const beforeDebt = new Set(['DRAFT', 'SENT', 'BUYER_REVIEWING', 'BUYER_ACCEPTED', 'VERIFICATION_PENDING', 'READY_TO_RELEASE', 'GOODS_RELEASED', 'RECEIPT_CONFIRMATION_PENDING', 'CANCELLED', 'DECLINED']);
  const outstanding = $derived(sumKobo(purchases.map(item => item.obligation?.outstanding_kobo ?? (beforeDebt.has(item.request.state) ? 0 : null))));
  const waiting = $derived(purchases.filter(item => ['SENT', 'BUYER_REVIEWING'].includes(item.request.state)).length);

  async function load(scope: string) {
    const request = reads.begin();
    loading = true; error = ''; businessID = ''; purchases = [];
    try {
      const profiles = await checkedJSON('/api/v1/buyer/businesses', rows('businesses', value => {
        const item = record(value);
        return { id: text(item.id), workspaceID: typeof item.workspace_id === 'string' ? item.workspace_id : '' };
      }), { signal: request.signal });
      if (!request.current()) return;
      const profile = profiles.find(item => item.workspaceID === scope);
      if (!profile) return;
      const sales = await checkedJSON(`/api/v1/buyer/credit-requests?business_id=${encodeURIComponent(profile.id)}&organization=${encodeURIComponent(scope)}`, rows('requests', saleView), { signal: request.signal });
      if (!request.current()) return;
      businessID = profile.id;
      purchases = sales.filter(item => item.request.buyer_business_id === profile.id);
    } catch (cause) {
      if (request.current()) error = publicError(cause, 'this business’s purchases');
    } finally {
      if (request.current()) loading = false;
    }
  }
  $effect(() => {
    if (organizationID) void load(organizationID);
    return () => reads.cancel();
  });
</script>

<section class="purchasing-summary" aria-label="Supplier balances" aria-busy={loading}>
  <h2>Your business owes suppliers</h2>
  {#if loading}
    <p role="status">Checking this business’s purchases…</p>
  {:else if error}
    <p role="alert">{error}</p><button onclick={() => load(organizationID)}>Try purchases again</button>
  {:else if !businessID}
    <p>No purchasing profile is available to your account for this business. Its owner can connect it when accepting a supplier invitation.</p>
  {:else}
    <strong class="amount">{#if outstanding === null}Not confirmed{:else}<Money amountKobo={outstanding} />{/if}</strong>
    <p>{outstanding === null ? 'One or more obligations could not be confirmed. Review the individual purchases.' : 'Supplier obligations for this business. Customer receivables do not reduce this amount.'}</p>
    {#if waiting}<p>{waiting} purchase offer{waiting === 1 ? '' : 's'} awaiting review.</p>{/if}
    <a href={`/workspace/purchases?business_id=${encodeURIComponent(businessID)}`}>Open this business’s purchases →</a>
  {/if}
</section>

<style>
  .purchasing-summary{margin-block:1rem;padding:1.5rem;border:1px solid var(--color-border);border-radius:.65rem;background:var(--color-surface)}
  h2{font-size:1rem;margin:0 0 .5rem}.amount{display:block;font-size:clamp(1.8rem,5vw,2.5rem);font-variant-numeric:tabular-nums}
  p{max-width:65ch;line-height:1.6;color:var(--color-muted)}a{display:inline-flex;min-height:44px;align-items:center;color:var(--color-primary);font-weight:650}
  button{padding:.7rem 1rem;font:inherit;border:1px solid var(--color-border);background:var(--color-surface);color:inherit}
</style>
