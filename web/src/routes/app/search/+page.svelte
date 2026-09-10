<script lang="ts">
	import { onMount } from 'svelte';
	import { checkedJSON, LatestRequest, publicError, record, rows, text } from '$lib/api/reliable';
	import { formatKobo } from '$lib/money';
 import { saleView } from '$lib/records';

	type Organization = { id: string; legal_name: string; trading_name?: string };
	type Customer = { buyer_user_id: string; buyer_business_id?: string; legal_name?: string; trading_name?: string; phone?: string; email?: string; state?: string };
	type CreditRequest = { id: string; buyer_legal_name?: string; goods_description?: string; invoice_reference?: string; principal_kobo?: number | string; state?: string };
	type Payment = { id: string; payment_id?: string; buyer_legal_name?: string; reference?: string; description?: string; request_id?: string; provider_reference?: string; external_reference?: string; amount_kobo?: number | string; state?: string };

	let organizations = $state<Organization[]>([]);
	let organizationID = $state('');
	let customers = $state<Customer[]>([]);
	let sales = $state<CreditRequest[]>([]);
	let payments = $state<Payment[]>([]);
	let query = $state('');
	let loading = $state(true);
	let error = $state('');

	const needle = $derived(query.trim().toLowerCase());
	const customerResults = $derived(needle ? customers.filter((item) => [item.legal_name, item.trading_name, item.phone, item.email, item.buyer_user_id, item.buyer_business_id].some((value) => String(value ?? '').toLowerCase().includes(needle))).slice(0, 20) : []);
	const saleResults = $derived(needle ? sales.filter((item) => [item.id, item.buyer_legal_name, item.goods_description, item.invoice_reference, item.state].some((value) => String(value ?? '').toLowerCase().includes(needle))).slice(0, 20) : []);
	const paymentResults = $derived(needle ? payments.filter((item) => [item.id, item.payment_id, item.buyer_legal_name, item.reference, item.description, item.request_id, item.provider_reference, item.external_reference, item.state].some((value) => String(value ?? '').toLowerCase().includes(needle))).slice(0, 20) : []);
	const count = $derived(customerResults.length + saleResults.length + paymentResults.length);

 const reads = new LatestRequest();
 function decodeItem<T>(value: unknown, id: string, strings: string[]): T {
  const item = record(value); if (!text(item[id])) throw new Error('Missing record identity');
  for (const field of strings) if (item[field] !== undefined && item[field] !== null) text(item[field]);
  return item as T;
 }
 async function loadBusiness() {
  const request = reads.begin(); loading = true; error = ''; customers = []; sales = []; payments = [];
  try {
   if (!organizationID) return;
   const base = `/api/v1/organizations/${encodeURIComponent(organizationID)}`;
   const [customerData, saleData, paymentData] = await Promise.all([
    checkedJSON(base + '/customers', rows('customers', value => decodeItem<Customer>(value, 'buyer_user_id', ['buyer_business_id','legal_name','trading_name','phone','email','state'])), { signal: request.signal }),
    checkedJSON(base + '/credit-requests', rows('requests', value => { saleView(value); return decodeItem<CreditRequest>(record(value).request, 'id', ['buyer_legal_name','goods_description','invoice_reference','state']); }), { signal: request.signal }),
    checkedJSON(base + '/payments', rows('payments', value => decodeItem<Payment>(value, 'id', ['payment_id','buyer_legal_name','reference','description','request_id','provider_reference','external_reference','state'])), { signal: request.signal })
   ]);
   if (request.current()) { customers = customerData; sales = saleData; payments = paymentData; }
  } catch (cause) { if (request.current()) error = publicError(cause, 'your records'); }
  finally { if (request.current()) loading = false; }
 }
 async function initialize() {
  const request = reads.begin(); loading = true; error = '';
  try {
   const result = await checkedJSON('/api/v1/organizations', rows('organizations', value => decodeItem<Organization>(value, 'id', ['legal_name','trading_name'])), { signal: request.signal });
   if (!request.current()) return;
   organizations = result;
   const requested = new URL(window.location.href).searchParams.get('organization');
   organizationID = result.find(item => item.id === requested)?.id ?? result[0]?.id ?? '';
   await loadBusiness();
  } catch (cause) { if (request.current()) { error = publicError(cause, 'your businesses'); loading = false; } }
 }
 onMount(() => { void initialize(); return () => reads.cancel(); });
</script>

<svelte:head><title>Search — Kredit</title></svelte:head>
<main class="shell workspace search-page">
	<header><p class="eyebrow">Find anything</p><h1>A customer, a sale, or a payment.</h1><p class="lede">Search names, goods, invoice numbers, Kredit numbers and transfer references, all inside your own business.</p></header>
	<div class="search-bar"><label>Business<select bind:value={organizationID} onchange={loadBusiness}>{#each organizations as organization}<option value={organization.id}>{organization.trading_name || organization.legal_name}</option>{/each}</select></label><label class="query">Search<input bind:value={query} type="search" placeholder="Name, invoice, sale or transfer reference" autocomplete="off" /></label></div>
	{#if error}<p class="error" role="alert">{error} <button type="button" onclick={() => organizations.length ? loadBusiness() : initialize()}>Try again</button></p>{:else if loading}<p role="status">Opening your records…</p>{:else if !organizationID}<section class="empty"><strong>No business selected.</strong><p>Create a business account to search its customers, sales and payments.</p></section>{:else if !needle}<section class="empty"><strong>Start typing.</strong><p>You only ever search records your own account is allowed to see.</p></section>{:else if count === 0}<section class="empty"><strong>Nothing matched that.</strong><p>Try a customer name, an invoice number, the goods, a sale number or a transfer reference.</p></section>{:else}
		<p class="count">{count} match{count === 1 ? '' : 'es'} shown</p>
		{#if customerResults.length}<section><h2>Customers</h2><div class="results">{#each customerResults as customer}<a href={`/app/customers/${encodeURIComponent(customer.buyer_user_id)}?organization=${encodeURIComponent(organizationID)}`}><div><strong>{customer.trading_name || customer.legal_name || 'Customer'}</strong><small>{customer.legal_name && customer.trading_name ? customer.legal_name : customer.email || customer.phone || customer.buyer_user_id}</small></div><span>Customer →</span></a>{/each}</div></section>{/if}
		{#if saleResults.length}<section><h2>Sales</h2><div class="results">{#each saleResults as sale}<a href={`/app/credit/${encodeURIComponent(sale.id)}?organization=${encodeURIComponent(organizationID)}`}><div><strong>{sale.buyer_legal_name || 'Credit sale'}</strong><small>{sale.goods_description || sale.invoice_reference || sale.id}</small></div><div class="money"><b>{formatKobo(sale.principal_kobo)}</b><span>{sale.state || 'Sale'} →</span></div></a>{/each}</div></section>{/if}
		{#if paymentResults.length}<section><h2>Payments</h2><div class="results">{#each paymentResults as payment}<a href={`/app/payments?organization=${organizationID}&q=${encodeURIComponent(payment.reference || payment.provider_reference || payment.external_reference || payment.id)}`}><div><strong>{payment.reference || payment.provider_reference || payment.external_reference || payment.id}</strong><small>{payment.request_id ? `Sale ${payment.request_id}` : 'Payment record'}</small></div><div class="money"><b>{formatKobo(payment.amount_kobo)}</b><span>{payment.state || 'Payment'} →</span></div></a>{/each}</div></section>{/if}
	{/if}
</main>

<style>
	.search-page header{max-width:55rem;padding:2.5rem 0 2rem;border-bottom:3px solid #17181b}.search-page h1{margin:.5rem 0;font-family:var(--font-serif);font-size:clamp(3rem,7vw,5.5rem);font-weight:500;line-height:.92;letter-spacing:-.055em}.search-bar{display:grid;grid-template-columns:minmax(12rem,20rem) 1fr;gap:1rem;margin:2rem 0;padding:1rem;background:#17181b}.search-bar label{display:grid;gap:.35rem;color:#fff;font-size:.78rem;font-weight:800}.search-bar input,.search-bar select{box-sizing:border-box;width:100%;min-height:3rem;padding:.75rem;border:0;background:#fff;color:#17181b;font:inherit}.empty{padding:2rem;border:1px solid var(--color-border);background:#fffdf8}.empty strong{font-family:var(--font-serif);font-size:1.8rem;font-weight:500}.empty p{color:var(--color-muted)}.count{font-weight:800}.search-page section>h2{margin:2rem 0 .75rem;font-family:var(--font-serif);font-size:1.7rem;font-weight:500}.results{border-top:1px solid var(--color-border)}.results a{display:flex;justify-content:space-between;gap:1.5rem;padding:1rem .25rem;border-bottom:1px solid var(--color-border);color:inherit;text-decoration:none}.results a:hover{background:#fffdf8}.results div{display:grid;gap:.3rem}.results small{color:var(--color-muted)}.results span{color:#2738d6;font-size:.78rem;font-weight:800;text-transform:capitalize}.money{text-align:right}.money b{font-family:var(--font-serif);font-size:1.2rem;font-weight:500}.error{color:#b42318}@media(max-width:680px){.search-bar{grid-template-columns:1fr}.results a{align-items:flex-start;flex-direction:column}.money{text-align:left}}
</style>
