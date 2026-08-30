<script lang="ts">
	import { page } from '$app/state'; import { onMount } from 'svelte';
	import { productLabel } from '$lib/product-language';
	let history: Record<string, any> | null = null, statement: Record<string, any> | null = null, error = '';
	const money = (value = 0) => new Intl.NumberFormat('en-NG', { style: 'currency', currency: 'NGN' }).format(value / 100);
	onMount(async () => {
		const organizationsResponse = await fetch('/api/v1/organizations'); const organizations = organizationsResponse.ok ? (await organizationsResponse.json()).organizations ?? [] : [];
		for (const organization of organizations) {
			const [historyResponse, statementResponse] = await Promise.all([fetch(`/api/v1/organizations/${organization.id}/customers/${page.params.id}/history`), fetch(`/api/v1/organizations/${organization.id}/customers/${page.params.id}/statement`)]);
			if (historyResponse.ok && statementResponse.ok) { history = await historyResponse.json(); statement = await statementResponse.json(); return; }
		}
		error = 'We could not find this customer in your business.';
	});
</script>
<svelte:head><title>Customer — Kredit</title></svelte:head>
<main class="shell workspace"><a href="/app/customers">← Customers</a><p class="eyebrow">Customer</p><h1>How this customer has paid you.</h1><p class="lede">You see only sales this customer made with your business. Other sellers cannot see your records.</p>
	{#if error}<p class="error" role="alert">{error}</p>{:else if !history}<p>Opening this customer…</p>{:else}<section class="stats"><article><span>Sales still open</span><strong>{history.active_obligations ?? 0}</strong></article><article><span>Money still owed</span><strong>{money(history.current_active_principal_kobo)}</strong></article><article><span>Sales fully paid</span><strong>{history.completed_obligations ?? 0}</strong></article><article><span>Paid on time</span><strong>{Number(history.on_time_percentage ?? 0).toFixed(0)}%</strong></article></section><section class="card"><h2>Sales with you</h2>{#if statement?.obligations?.length}<div class="table">{#each statement.obligations as obligation}<a href={`/app/credit/${obligation.credit_request_id}`}><span>{obligation.buyer_name || 'Credit sale'}</span><strong>{money(obligation.outstanding_kobo)}</strong><small>{productLabel(obligation.payment_status)}</small></a>{/each}</div>{:else}<p>This customer has no active credit sale with you.</p>{/if}</section>{/if}
</main>
<style>.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(11rem,1fr));gap:1rem;margin:2rem 0}.stats article,.card{padding:1.2rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}.stats span{display:block;color:var(--color-muted)}.stats strong{display:block;font-size:1.45rem;margin-top:.4rem}.table{display:grid}.table a{display:grid;grid-template-columns:1fr auto auto;gap:1rem;padding:.9rem 0;border-bottom:1px solid var(--color-border);color:inherit;text-decoration:none}.table small{color:var(--color-muted)}.error{color:#b42318}</style>
