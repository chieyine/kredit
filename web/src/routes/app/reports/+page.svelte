<script lang="ts">
	import { onMount } from 'svelte';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import ShareActions from '$lib/components/ShareActions.svelte';
	type Organization = { id: string; legal_name: string; trading_name?: string };
	type Summary = { obligation_count: number; outstanding_kobo: number; overdue_kobo: number; voluntary_paid_kobo: number; collected_paid_kobo: number };
	let organizations: Organization[] = [], organizationID = '', summary: Summary | null = null;
	let buckets: Record<string, number> = {};
	let fees: { total_fees_kobo: number } | null = null;
	let payments: any[] = [], sharePeriod = 'today';
	let loading = true, error = '';
	const money = (value = 0) => new Intl.NumberFormat('en-NG', { style: 'currency', currency: 'NGN' }).format(value / 100);
	async function load() {
		if (!organizationID) return;
		loading = true; error = '';
		const responses = await Promise.all([...['receivables', 'ageing', 'fees'].map((name) => fetch(`/api/v1/organizations/${organizationID}/reports/${name}`)), fetch(`/api/v1/organizations/${organizationID}/payments`)]);
		if (!responses.every((response) => response.ok)) { error = 'We could not open the reports for this business.'; loading = false; return; }
		summary = (await responses[0].json()).summary; buckets = (await responses[1].json()).buckets; fees = await responses[2].json(); payments = (await responses[3].json()).payments ?? []; loading = false;
	}
	function paymentSummary(){const now=new Date();const start=new Date(now);if(sharePeriod==='today')start.setHours(0,0,0,0);else start.setDate(now.getDate()-6),start.setHours(0,0,0,0);const received=payments.filter((payment)=>payment.state==='recognized'&&new Date(payment.paid_at)>=start);return{count:received.length,total:received.reduce((sum,payment)=>sum+Number(payment.amount_kobo||0),0)}}
	function summaryText(){const value=paymentSummary();return `Kredit ${sharePeriod==='today'?'today':'last 7 days'}: ${value.count} payment(s) received, ${money(value.total)} in total. ${money(summary?.outstanding_kobo)} is still owed.`}
	async function exportCSV() {
		const response = await fetch(`/api/v1/organizations/${organizationID}/reports/exports?format=csv`, { method: 'POST', headers: { ...csrfHeaders(), 'Idempotency-Key': idempotencyKey() } });
		if (!response.ok) { error = 'The export could not be created.'; return; }
		const objectURL = URL.createObjectURL(await response.blob()); const link = document.createElement('a'); link.href = objectURL; link.download = 'kredit-receivables.csv'; link.click(); URL.revokeObjectURL(objectURL);
	}
	onMount(async () => {
		const response = await fetch('/api/v1/organizations');
		if (!response.ok) { error = 'Sign in to see your business reports.'; loading = false; return; }
		organizations = (await response.json()).organizations ?? []; organizationID = organizations[0]?.id ?? ''; await load();
	});
</script>
<svelte:head><title>Reports — Kredit</title></svelte:head>
<main class="shell workspace"><p class="eyebrow">Your reports</p><h1>See where your money is.</h1><p class="lede">Money owed, money late, money paid and Kredit fees.</p>
	<div class="toolbar"><label>Business<select bind:value={organizationID} onchange={load}>{#each organizations as organization}<option value={organization.id}>{organization.trading_name || organization.legal_name}</option>{/each}</select></label><button onclick={exportCSV} disabled={!summary}>Download a copy</button></div>
	{#if error}<p class="error" role="alert">{error}</p>{:else if loading}<div class="loading" aria-live="polite"><Skeleton rows={4} tall /></div>{:else if summary}<section class="card share-card"><div><h2>Send a money update</h2><p>Share a short payment update with your business partner or staff.</p><label>Time<select bind:value={sharePeriod}><option value="today">Today</option><option value="week">Last 7 days</option></select></label></div><div><strong>{money(paymentSummary().total)}</strong><small>{paymentSummary().count} payment(s)</small><ShareActions compact title="Kredit money update" text={summaryText()}/></div></section><section class="stats"><article><span>Money still owed</span><strong>{money(summary.outstanding_kobo)}</strong><small>{summary.obligation_count} sales</small></article><article><span>Money late</span><strong>{money(summary.overdue_kobo)}</strong><small>Extra payment days have ended</small></article><article><span>Money paid</span><strong>{money(summary.voluntary_paid_kobo + summary.collected_paid_kobo)}</strong><small>All payments</small></article><article><span>Kredit fees</span><strong>{money(fees?.total_fees_kobo)}</strong><small>All fees paid</small></article></section><section class="card"><h2>How late is the money?</h2><dl>{#each Object.entries(buckets) as [bucket, amount]}<div><dt>{bucket.replace('_', '–')} days</dt><dd>{money(amount)}</dd></div>{/each}</dl></section>{/if}
</main>
<style>.toolbar{display:flex;justify-content:space-between;gap:1rem;align-items:end;margin:2rem 0;padding:1rem;flex-wrap:wrap;color:#fff;background:#17181b}.toolbar label{display:grid;gap:.4rem;font-weight:700}.toolbar select,.toolbar button{min-height:3rem;padding:.75rem;border:1px solid #4b4c51;border-radius:0;background:#fff;color:#17181b}.toolbar button{background:#2738d6;color:white;border-color:#2738d6;font-weight:750}.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(12rem,1fr));gap:0;border-top:1px solid var(--color-border);border-left:1px solid var(--color-border)}.stats article,.card{padding:1.25rem;border:0;border-right:1px solid var(--color-border);border-bottom:1px solid var(--color-border);border-radius:0;background:var(--color-surface)}.stats span,.stats small{display:block;color:var(--color-muted)}.stats strong{display:block;margin:.35rem 0;font-family:Georgia,'Times New Roman',serif;font-size:1.8rem;font-weight:500}.card{margin-top:1rem;border:1px solid var(--color-border)}.share-card{display:grid;grid-template-columns:1fr auto;gap:2rem;align-items:center;margin-bottom:1rem;background:#ebe7de}.share-card h2,.share-card p{margin:.25rem 0}.share-card label{display:grid;gap:.3rem;width:max-content;margin-top:.8rem;font-weight:750}.share-card select{min-height:2.7rem;padding:.55rem;border:1px solid var(--color-border);background:#fff;font:inherit}.share-card>div:last-child>strong,.share-card>div:last-child>small{display:block}.share-card>div:last-child>strong{font-family:Georgia,'Times New Roman',serif;font-size:2rem;font-weight:500}.card dl{display:grid;gap:.7rem}.card dl div{display:flex;justify-content:space-between;border-bottom:1px solid var(--color-border);padding-bottom:.6rem}.card dd{font-weight:700}.error{color:#b42318}@media(max-width:600px){.stats{grid-template-columns:1fr}.toolbar{align-items:stretch;flex-direction:column}.toolbar button{width:100%}.share-card{grid-template-columns:1fr}}
</style>
