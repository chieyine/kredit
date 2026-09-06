<script lang="ts">
	import { onMount } from 'svelte';
	import { readJSON } from '$lib/api/client';
	import { formatKobo } from '$lib/money';

	type Organization = { id: string; legal_name: string; trading_name?: string };
	type Customer = { buyer_user_id: string; buyer_business_id?: string; legal_name?: string; trading_name?: string; phone?: string; email?: string; state?: string };
	type CreditRequest = { id: string; buyer_legal_name?: string; goods_description?: string; invoice_reference?: string; principal_kobo?: number | string; state?: string };
	type Payment = { id: string; request_id?: string; provider_reference?: string; external_reference?: string; amount_kobo?: number | string; state?: string };

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
	const paymentResults = $derived(needle ? payments.filter((item) => [item.id, item.request_id, item.provider_reference, item.external_reference, item.state].some((value) => String(value ?? '').toLowerCase().includes(needle))).slice(0, 20) : []);
	const count = $derived(customerResults.length + saleResults.length + paymentResults.length);

	async function loadBusiness() {
		if (!organizationID) return;
		loading = true;
		error = '';
		try {
			const [customerData, saleData, paymentData] = await Promise.all([
				readJSON<{ customers?: Customer[] }>(`/api/v1/organizations/${organizationID}/customers`),
				readJSON<{ requests?: CreditRequest[] }>(`/api/v1/organizations/${organizationID}/credit-requests`),
				readJSON<{ payments?: Payment[] }>(`/api/v1/organizations/${organizationID}/payments`)
			]);
			customers = customerData.customers ?? [];
			sales = saleData.requests ?? [];
			payments = paymentData.payments ?? [];
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Search data could not be loaded.';
		} finally {
			loading = false;
		}
	}

	onMount(async () => {
		try {
			const body = await readJSON<{ organizations?: Organization[] }>('/api/v1/organizations');
			organizations = body.organizations ?? [];
			organizationID = organizations[0]?.id ?? '';
			await loadBusiness();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Search could not be opened.';
			loading = false;
		}
	});
</script>

<svelte:head><title>Search — Kredit</title></svelte:head>
<main class="shell workspace search-page">
	<header><p class="eyebrow">Find anything</p><h1>Customer, sale or payment.</h1><p class="lede">Search names, goods, invoice numbers, Kredit references and payment references inside your business.</p></header>
	<div class="search-bar"><label>Business<select bind:value={organizationID} onchange={loadBusiness}>{#each organizations as organization}<option value={organization.id}>{organization.trading_name || organization.legal_name}</option>{/each}</select></label><label class="query">Search<input bind:value={query} type="search" placeholder="Name, invoice, sale or payment reference" autocomplete="off" autofocus /></label></div>
	{#if error}<p class="error" role="alert">{error}</p>{:else if loading}<p>Loading your business records…</p>{:else if !needle}<section class="empty"><strong>Start typing.</strong><p>Only records your seller account is authorized to see are searched.</p></section>{:else if count === 0}<section class="empty"><strong>No match found.</strong><p>Try a customer name, invoice number, goods description, sale ID or payment reference.</p></section>{:else}
		<p class="count">{count} match{count === 1 ? '' : 'es'} shown</p>
		{#if customerResults.length}<section><h2>Customers</h2><div class="results">{#each customerResults as customer}<a href={`/app/customers/${customer.buyer_user_id}?organization=${organizationID}`}><div><strong>{customer.trading_name || customer.legal_name || 'Customer'}</strong><small>{customer.legal_name && customer.trading_name ? customer.legal_name : customer.email || customer.phone || customer.buyer_user_id}</small></div><span>Customer →</span></a>{/each}</div></section>{/if}
		{#if saleResults.length}<section><h2>Sales</h2><div class="results">{#each saleResults as sale}<a href={`/app/credit/${sale.id}?organization=${organizationID}`}><div><strong>{sale.buyer_legal_name || 'Credit sale'}</strong><small>{sale.goods_description || sale.invoice_reference || sale.id}</small></div><div class="money"><b>{formatKobo(sale.principal_kobo)}</b><span>{sale.state || 'Sale'} →</span></div></a>{/each}</div></section>{/if}
		{#if paymentResults.length}<section><h2>Payments</h2><div class="results">{#each paymentResults as payment}<a href={`/app/payments?organization=${organizationID}&q=${encodeURIComponent(payment.provider_reference || payment.external_reference || payment.id)}`}><div><strong>{payment.provider_reference || payment.external_reference || payment.id}</strong><small>{payment.request_id ? `Sale ${payment.request_id}` : 'Payment record'}</small></div><div class="money"><b>{formatKobo(payment.amount_kobo)}</b><span>{payment.state || 'Payment'} →</span></div></a>{/each}</div></section>{/if}
	{/if}
</main>

<style>
	.search-page header{max-width:55rem;padding:2.5rem 0 2rem;border-bottom:3px solid #17181b}.search-page h1{margin:.5rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(3rem,7vw,5.5rem);font-weight:500;line-height:.92;letter-spacing:-.055em}.search-bar{display:grid;grid-template-columns:minmax(12rem,20rem) 1fr;gap:1rem;margin:2rem 0;padding:1rem;background:#17181b}.search-bar label{display:grid;gap:.35rem;color:#fff;font-size:.78rem;font-weight:800}.search-bar input,.search-bar select{box-sizing:border-box;width:100%;min-height:3rem;padding:.75rem;border:0;background:#fff;color:#17181b;font:inherit}.empty{padding:2rem;border:1px solid var(--color-border);background:#fffdf8}.empty strong{font-family:Georgia,serif;font-size:1.8rem;font-weight:500}.empty p{color:var(--color-muted)}.count{font-weight:800}.search-page section>h2{margin:2rem 0 .75rem;font-family:Georgia,serif;font-size:1.7rem;font-weight:500}.results{border-top:1px solid var(--color-border)}.results a{display:flex;justify-content:space-between;gap:1.5rem;padding:1rem .25rem;border-bottom:1px solid var(--color-border);color:inherit;text-decoration:none}.results a:hover{background:#fffdf8}.results div{display:grid;gap:.3rem}.results small{color:var(--color-muted)}.results span{color:#2738d6;font-size:.78rem;font-weight:800;text-transform:capitalize}.money{text-align:right}.money b{font-family:Georgia,serif;font-size:1.2rem;font-weight:500}.error{color:#b42318}@media(max-width:680px){.search-bar{grid-template-columns:1fr}.results a{align-items:flex-start;flex-direction:column}.money{text-align:left}}
</style>
