<script lang="ts">
  import { onMount } from 'svelte';
  import { LatestRequest, readResource, rows, type Resource } from '$lib/api/reliable';
  import { organization, saleView, dateLabel, type Organization, type SaleView } from '$lib/records';
  import { productLabel } from '$lib/product-language';
  import Money from '$lib/components/Money.svelte';
  import ResourceNotice from '$lib/components/ResourceNotice.svelte';
  let businesses = $state<Resource<Organization[]>>({ state: 'loading', scope: '' });
  let sales = $state<Resource<SaleView[]>>({ state: 'loading', scope: '' });
  let organizationID = $state(''), query = $state(''), filter = $state('all'), visible = $state(20);
  const requests = new LatestRequest(), businessRequests = new LatestRequest();
  const matching = $derived(sales.state === 'ready' ? sales.data.filter(({ request }) => `${request.buyer_legal_name} ${request.goods_description} ${request.id}`.toLocaleLowerCase('en-NG').includes(query.trim().toLocaleLowerCase('en-NG')) && (filter === 'all' || (filter === 'draft' ? request.state === 'DRAFT' : filter === 'closed' ? ['PAID', 'COMPLETED', 'CANCELLED', 'DECLINED', 'REJECTED', 'EXPIRED'].includes(request.state) : !['DRAFT', 'PAID', 'COMPLETED', 'CANCELLED', 'DECLINED', 'REJECTED', 'EXPIRED'].includes(request.state)))) : []);
  async function loadSales() {
    const scope = organizationID, request = requests.begin(); visible = 20;
    sales = { state: 'loading', scope };
    if (!scope) return;
    const result = await readResource(scope, `/api/v1/organizations/${encodeURIComponent(scope)}/credit-requests`, rows('requests', saleView), request.signal, 'your sales');
    if (request.current() && organizationID === scope) sales = result;
  }
  async function load() {
    const request = businessRequests.begin();
    businesses = { state: 'loading', scope: '' };
    const result = await readResource('', '/api/v1/organizations', rows('organizations', organization), request.signal, 'your businesses');
    if (!request.current()) return;
    businesses = result;
    if (businesses.state !== 'ready') return;
    const requested = new URLSearchParams(location.search).get('organization');
    organizationID = businesses.data.find(item => item.id === requested)?.id ?? businesses.data[0]?.id ?? '';
    if (organizationID) await loadSales();
  }
  onMount(() => { void load(); return () => { requests.cancel(); businessRequests.cancel(); }; });
</script>
<svelte:head><title>Your credit sales — Kredit</title></svelte:head>
<main class="shell sales-page"><header class="task-heading"><div><p class="eyebrow">Your records</p><h1>Sales</h1><p>Find a customer, check a payment date or continue a draft.</p></div><a class="primary" href={`/app/credit/quick?organization=${encodeURIComponent(organizationID)}`}>Add a sale</a></header>
  <ResourceNotice resource={businesses} label="Businesses" retry={load} />
  {#if businesses.state === 'ready' && businesses.data.length}<div class="filters"><label>Business<select bind:value={organizationID} onchange={loadSales}>{#each businesses.data as item}<option value={item.id}>{item.trading_name || item.legal_name}</option>{/each}</select></label><label class="search-field">Find a sale<input type="search" bind:value={query} placeholder="Customer, goods or sale number" /></label><label>Show<select bind:value={filter}><option value="all">All sales</option><option value="open">Open sales</option><option value="draft">Drafts</option><option value="closed">Closed sales</option></select></label></div>
    <ResourceNotice resource={sales} label="Sales" retry={loadSales} />
    {#if sales.state === 'ready'}<p class="result-count" role="status">{matching.length} matching sale{matching.length === 1 ? '' : 's'}</p>{#if matching.length}<div class="sale-list">{#each matching.slice(0, visible) as { request, obligation } (request.id)}<a href={`/app/credit/${encodeURIComponent(request.id)}?organization=${encodeURIComponent(organizationID)}`}><div><strong>{request.buyer_legal_name}</strong><p>{request.goods_description}</p><small>{productLabel(request.state)} · Pay by {dateLabel(request.due_date)}</small></div><div class="sale-money"><strong><Money amountKobo={obligation?.outstanding_kobo ?? request.principal_kobo} /></strong><small>{obligation ? 'Left to pay' : 'Sale amount'}</small></div></a>{/each}</div>{#if matching.length > visible}<button class="secondary" type="button" onclick={() => visible += 20}>Show more sales</button>{/if}{:else}<div class="empty-state"><h2>{sales.data.length ? 'No sales match this search' : 'No sales yet'}</h2><p>{sales.data.length ? 'Try another customer name, sale number or filter.' : 'Add a sale to keep the goods, terms and payments together.'}</p></div>{/if}{/if}
  {:else if businesses.state === 'ready'}<a href="/app/overview">Add your business first</a>{/if}
</main>
<style>
  .sales-page{max-width:66rem}.task-heading{display:flex;justify-content:space-between;align-items:center;gap:1rem;margin:1rem 0 2rem}.task-heading h1{font-family:inherit;font-size:2rem;margin:.3rem 0;line-height:1.2}.task-heading p{color:var(--color-muted)}.filters{display:flex;flex-wrap:wrap;gap:1rem}.filters label{display:grid;gap:.45rem}.filters .search-field{flex:1;min-width:min(100%,14rem)}.filters input,.filters select{box-sizing:border-box;width:100%;font:inherit;padding:.75rem;border:1px solid var(--color-border);border-radius:.35rem;background:var(--color-surface)}.result-count{font-size:.9rem;color:var(--color-muted);margin-block:1.25rem}.sale-list{border:1px solid var(--color-border);border-radius:.5rem;overflow:hidden;background:var(--color-surface)}.sale-list>a{display:flex;justify-content:space-between;gap:1.5rem;padding:1.2rem;text-decoration:none;border-bottom:1px solid var(--color-border)}.sale-list>a:last-child{border:0}.sale-list p{margin:.4rem 0;line-height:1.5}.sale-list small{display:block;color:var(--color-muted);line-height:1.5}.sale-money{text-align:right;white-space:nowrap;font-variant-numeric:tabular-nums}.sale-money small{margin-top:.5rem}.secondary{margin-top:1rem;padding:.75rem 1rem;border:1px solid var(--color-border);background:transparent;border-radius:.35rem}.empty-state{padding:2rem;border:1px dashed var(--color-border);border-radius:.5rem}.empty-state h2{font-size:1.25rem}@media(max-width:560px){.task-heading{flex-wrap:wrap}.filters label{width:100%}.sale-list>a{flex-direction:column;gap:1rem}.sale-money{text-align:left}.sale-money small{display:inline;margin-left:.6rem}}
</style>
