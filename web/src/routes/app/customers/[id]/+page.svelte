<script lang="ts">
	import { page } from '$app/state'; import { onMount } from 'svelte';
	import { formatKobo } from '$lib/money';
	import { productLabel } from '$lib/product-language';
	import ShareActions from '$lib/components/ShareActions.svelte';
	import { readLocal, writeLocal } from '$lib/product-tools';
	let history: Record<string, any> | null = null, statement: Record<string, any> | null = null, error = '', organizationID = '', note = '', noteMessage = '';
	const money = formatKobo;
	async function loadCustomer() {
		error = '';
		history = null;
		statement = null;
		try {
			const response = await fetch('/api/v1/organizations');
			if (!response.ok) throw new Error('We could not load your businesses. Please try again.');
			const organizations = (await response.json()).organizations ?? [];
			const selected = page.url.searchParams.get('organization');
			const candidates = selected ? organizations.filter((org: any) => org.id === selected) : organizations;
			for (const organization of candidates) {
				const [historyResponse, statementResponse] = await Promise.all([
					fetch(`/api/v1/organizations/${organization.id}/customers/${page.params.id}/history`),
					fetch(`/api/v1/organizations/${organization.id}/customers/${page.params.id}/statement`)
				]);
				if (historyResponse.status === 404 || statementResponse.status === 404) continue;
				if (!historyResponse.ok || !statementResponse.ok) throw new Error('We could not load this customer. Please try again.');
				const [loadedHistory, loadedStatement] = await Promise.all([historyResponse.json(), statementResponse.json()]);
				organizationID = organization.id;
				history = loadedHistory;
				statement = loadedStatement;
				note = readLocal(`kredit:customer-note:${organization.id}:${page.params.id}`, '');
				return;
			}
			throw new Error('We could not find this customer in your business.');
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'We could not load this customer. Check your connection and try again.';
		}
	}
	onMount(() => { void loadCustomer(); });
	function saveNote(){writeLocal(`kredit:customer-note:${organizationID}:${page.params.id}`,note.trim());noteMessage='Note saved on this phone.';setTimeout(()=>noteMessage='',1800)}
</script>
<svelte:head><title>Customer — Kredit</title></svelte:head>
<main class="shell workspace"><a href="/app/customers">← Customers</a><p class="eyebrow">Customer</p><h1>How this customer has paid you.</h1><p class="lede">You see only sales this customer made with your business. Other sellers cannot see your records.</p>
	{#if error}<p class="error" role="alert">{error} <button type="button" onclick={loadCustomer}>Try again</button></p>{:else if !history}<p>Opening this customer…</p>{:else}<div class="customer-actions"><a class="primary-link" href={`/app/credit/new?customer=${encodeURIComponent(page.params.id ?? '')}&organization=${encodeURIComponent(organizationID)}`}>Make another sale</a><button type="button" onclick={()=>window.print()}>Print this page</button></div><ShareActions title="Kredit customer statement" text={`Kredit statement: ${money(history.current_active_principal_kobo)} is still owed across ${history.active_obligations ?? 0} open sale(s). ${history.completed_obligations ?? 0} sale(s) fully paid.`}/><section class="stats"><article><span>Sales still open</span><strong>{history.active_obligations ?? 0}</strong></article><article><span>Money still owed</span><strong>{money(history.current_active_principal_kobo)}</strong></article><article><span>Sales fully paid</span><strong>{history.completed_obligations ?? 0}</strong></article><article><span>Paid on time</span><strong>{Number(history.on_time_percentage ?? 0).toFixed(0)}%</strong></article></section><section class="card"><h2>Private note</h2><p>Use this for a delivery direction, usual order or reminder. This note stays only on this phone.</p><label>Note about this customer<textarea bind:value={note} rows="3" placeholder="For example: delivers to the second shop on Mondays"></textarea></label><button type="button" onclick={saveNote}>Save note</button>{#if noteMessage}<small role="status">{noteMessage}</small>{/if}</section><section class="card"><h2>Sales with you</h2>{#if statement?.obligations?.length}<div class="table">{#each statement.obligations as obligation}<a href={`/app/credit/${obligation.credit_request_id}?organization=${organizationID}`}><span>{obligation.buyer_name || 'Credit sale'}</span><strong>{money(obligation.outstanding_kobo)}</strong><small>{productLabel(obligation.payment_status)}</small></a>{/each}</div>{:else}<p>This customer has no active credit sale with you.</p>{/if}</section>{/if}
</main>
<style>.customer-actions{display:flex;flex-wrap:wrap;gap:.7rem;margin-top:1.25rem}.customer-actions a,.customer-actions button,.card button{padding:.7rem .9rem;border:1px solid var(--color-foreground);background:var(--color-surface);color:inherit;font:inherit;font-weight:750;text-decoration:none}.customer-actions .primary-link{border-color:var(--color-primary);background:var(--color-primary);color:#fff}.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(11rem,1fr));gap:1rem;margin:2rem 0}.stats article,.card{padding:1.2rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}.card{margin-top:1rem}.card label{display:grid;gap:.4rem;font-weight:750}.card textarea{box-sizing:border-box;width:100%;margin:.4rem 0;padding:.75rem;border:1px solid var(--color-border);font:inherit}.stats span{display:block;color:var(--color-muted)}.stats strong{display:block;font-size:1.45rem;margin-top:.4rem}.table{display:grid}.table a{display:grid;grid-template-columns:1fr auto auto;gap:1rem;padding:.9rem 0;border-bottom:1px solid var(--color-border);color:inherit;text-decoration:none}.table small{color:var(--color-muted)}.error{color:#b42318}@media print{.customer-actions,:global(.share-actions){display:none!important}}@media(max-width:600px){.table a{grid-template-columns:1fr auto}.table small{grid-column:1/-1}}</style>
