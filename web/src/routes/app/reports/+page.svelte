<script lang="ts">
	import { onMount } from 'svelte';
	import { formatKobo, type KoboValue } from '$lib/money';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import ShareActions from '$lib/components/ShareActions.svelte';
	type Organization = { id: string; legal_name: string; trading_name?: string };
	type Summary = { obligation_count: number; outstanding_kobo: number; overdue_kobo: number; voluntary_paid_kobo: number; collected_paid_kobo: number };
	let organizations = $state<Organization[]>([]);
	let organizationID = $state('');
	let summary = $state<Summary | null>(null);
	let buckets = $state<Record<string, number>>({});
	let fees = $state<{ total_fees_kobo: number } | null>(null);
	let payments = $state<any[]>([]);
	let sharePeriod = $state('today');
	let loading = $state(true);
	let error = $state('');
	const money = (value: KoboValue) => formatKobo(value);
	const paidTotal = $derived((summary?.voluntary_paid_kobo ?? 0) + (summary?.collected_paid_kobo ?? 0));
	const trackedValue = $derived(paidTotal + (summary?.outstanding_kobo ?? 0));
	const receivedRate = $derived(trackedValue > 0 ? Math.round((paidTotal / trackedValue) * 100) : 0);
	const overdueShare = $derived((summary?.outstanding_kobo ?? 0) > 0 ? Math.round(((summary?.overdue_kobo ?? 0) / (summary?.outstanding_kobo ?? 1)) * 100) : 0);
	const recognizedPayments = $derived(payments.filter((payment)=>['recognized','confirmed','paid'].includes(String(payment.state??payment.status??'').toLowerCase())));
	const averagePayment = $derived(recognizedPayments.length ? Math.round(recognizedPayments.reduce((sum,payment)=>sum+Number(payment.amount_kobo||0),0)/recognizedPayments.length) : 0);
	async function load() {
		if (!organizationID) return;
		loading = true; error = '';
		const responses = await Promise.all([...['receivables', 'ageing', 'fees'].map((name) => fetch(`/api/v1/organizations/${organizationID}/reports/${name}`)), fetch(`/api/v1/organizations/${organizationID}/payments`)]);
		if (!responses.every((response) => response.ok)) { error = 'We could not open the reports. Please try again.'; loading = false; return; }
		summary = (await responses[0].json()).summary; buckets = (await responses[1].json()).buckets; fees = await responses[2].json(); payments = (await responses[3].json()).payments ?? []; loading = false;
	}
	function paymentSummary(){const now=new Date();const start=new Date(now);if(sharePeriod==='today')start.setHours(0,0,0,0);else start.setDate(now.getDate()-6),start.setHours(0,0,0,0);const received=payments.filter((payment)=>['recognized','confirmed','paid'].includes(String(payment.state??payment.status??'').toLowerCase())&&new Date(payment.paid_at??payment.created_at)>=start);return{count:received.length,total:received.reduce((sum,payment)=>sum+Number(payment.amount_kobo||0),0)}}
	function summaryText(){const value=paymentSummary();return `Kredit ${sharePeriod==='today'?'today':'last 7 days'}: ${value.count} payment(s) received, ${money(value.total)} in total. ${money(summary?.outstanding_kobo)} still owed; ${money(summary?.overdue_kobo)} is overdue.`}
	function bucketName(bucket:string){const normalized=bucket.toLowerCase();if(normalized.includes('current')||normalized.includes('not_due'))return 'Not overdue';if(normalized.includes('60'))return '60+ days overdue';const numbers=bucket.match(/\d+/g);return numbers?.length===2?`${numbers[0]}–${numbers[1]} days overdue`:numbers?.length===1?`${numbers[0]}+ days overdue`:bucket.replaceAll('_',' ')}
	async function exportCSV() {
		const response = await fetch(`/api/v1/organizations/${organizationID}/reports/exports?format=csv`, { method: 'POST', headers: { ...csrfHeaders(), 'Idempotency-Key': idempotencyKey() } });
		if (!response.ok) { error = 'We could not create that file. Please try again.'; return; }
		const objectURL = URL.createObjectURL(await response.blob()); const link = document.createElement('a'); link.href = objectURL; link.download = 'kredit-receivables.csv'; link.click(); URL.revokeObjectURL(objectURL);
	}
	onMount(async () => {
		const response = await fetch('/api/v1/organizations');
		if (!response.ok) { error = 'Sign in to see your reports.'; loading = false; return; }
		organizations = (await response.json()).organizations ?? []; organizationID = organizations[0]?.id ?? ''; await load();
	});
</script>
<svelte:head><title>Reports — Kredit</title></svelte:head>
<main class="shell workspace reports"><header><p class="eyebrow">Business performance</p><h1>Where all your money is sitting.</h1><p class="lede">What you are owed, what is late, what has come in, how old the debts are and what Kredit has charged you.</p></header>
	<div class="toolbar"><label>Business<select bind:value={organizationID} onchange={load}>{#each organizations as organization}<option value={organization.id}>{organization.trading_name || organization.legal_name}</option>{/each}</select></label><button onclick={exportCSV} disabled={!summary}>Download CSV</button></div>
	{#if error}<p class="error" role="alert">{error}</p>{:else if loading}<div class="loading" aria-live="polite"><Skeleton rows={4} tall /></div>{:else if summary}
		<section class="stats"><article class="focus"><span>Owed to you now</span><strong>{money(summary.outstanding_kobo)}</strong><small>{summary.obligation_count} open sale{summary.obligation_count===1?'':'s'}</small></article><article><span>Late right now</span><strong>{money(summary.overdue_kobo)}</strong><small>{overdueShare}% of what you are owed</small></article><article><span>Received</span><strong>{money(paidTotal)}</strong><small>{receivedRate}% of everything Kredit is following</small></article><article><span>Average payment</span><strong>{money(averagePayment)}</strong><small>{recognizedPayments.length} payment saved{recognizedPayments.length===1?'':'s'}</small></article><article><span>Kredit fees</span><strong>{money(fees?.total_fees_kobo)}</strong><small>What Kredit has charged you, after any reductions</small></article></section>
		<section class="performance"><div><p class="eyebrow">How collection is going</p><h2>{receivedRate}% of the money Kredit is following has reached you.</h2><p>This compares money already received with money received plus money still owed. Use it to get a feel for how things are going. It is not an accounting statement.</p></div><div class="meter" aria-label={`${receivedRate}% received`}><div><span style={`width:${receivedRate}%`}></span></div><p><b>{money(paidTotal)}</b> received <span>·</span> <b>{money(summary.outstanding_kobo)}</b> still owed</p></div></section>
		<section class="card ageing"><h2>How old is the money you are owed?</h2><p>The longer a debt sits, the harder it usually gets to collect. Chase the old ones first.</p><dl>{#each Object.entries(buckets) as [bucket, amount]}<div><dt>{bucketName(bucket)}</dt><dd>{money(amount)}</dd></div>{/each}</dl><a href="/app/overdue">See who is late →</a></section>
		<section class="card share-card"><div><h2>Send someone a money update</h2><p>Send a short summary to a partner or a staff member, without giving them your whole book.</p><label>Period<select bind:value={sharePeriod}><option value="today">Today</option><option value="week">Last 7 days</option></select></label></div><div><strong>{money(paymentSummary().total)}</strong><small>{paymentSummary().count} payment{paymentSummary().count===1?'':'s'} received</small><ShareActions compact title="Kredit money update" text={summaryText()}/></div></section>
	{/if}
</main>
<style>.reports>header{max-width:62rem}.reports h1{max-width:15ch;font-family:Georgia,'Times New Roman',serif;font-size:clamp(3rem,6vw,5.4rem);font-weight:500;line-height:.92;letter-spacing:-.055em}.toolbar{display:flex;justify-content:space-between;gap:1rem;align-items:end;margin:2rem 0;padding:1rem;flex-wrap:wrap;color:#fff;background:#17181b}.toolbar label{display:grid;gap:.4rem;font-weight:700}.toolbar select,.toolbar button{min-height:3rem;padding:.75rem;border:1px solid #4b4c51;border-radius:0;background:#fff;color:#17181b}.toolbar button{background:#2738d6;color:white;border-color:#2738d6;font-weight:750}.stats{display:grid;grid-template-columns:1.2fr repeat(4,1fr);border-top:1px solid var(--color-border);border-left:1px solid var(--color-border)}.stats article,.card{padding:1.25rem;border:0;border-right:1px solid var(--color-border);border-bottom:1px solid var(--color-border);border-radius:0;background:var(--color-surface)}.stats article.focus{color:#fff;background:#2738d6}.stats span,.stats small{display:block;color:var(--color-muted)}.stats .focus span,.stats .focus small{color:#d8dbff}.stats strong{display:block;margin:.45rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(1.5rem,2.5vw,2.2rem);font-weight:500}.performance{display:grid;grid-template-columns:1fr 1fr;gap:clamp(2rem,7vw,7rem);align-items:center;margin:2rem 0;padding:2rem;border:1px solid var(--color-border);background:#17181b;color:#fff}.performance h2{max-width:13ch;margin:.5rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(2rem,4vw,3.5rem);font-weight:500;line-height:.98}.performance>div>p:last-child{color:#b6b6b1;line-height:1.65}.meter>div{height:1rem;background:#44454a}.meter>div span{display:block;height:100%;background:#e85f3d}.meter p{display:flex;flex-wrap:wrap;gap:.45rem;color:#b6b6b1}.meter b{color:#fff}.card{margin-top:1rem;border:1px solid var(--color-border)}.ageing>p{color:var(--color-muted)}.card dl{display:grid;gap:.7rem}.card dl div{display:flex;justify-content:space-between;gap:2rem;border-bottom:1px solid var(--color-border);padding-bottom:.7rem}.card dd{font-weight:700}.ageing>a{display:inline-block;margin-top:1rem;font-weight:750}.share-card{display:grid;grid-template-columns:1fr auto;gap:2rem;align-items:center;margin-top:2rem;background:#ebe7de}.share-card h2,.share-card p{margin:.25rem 0}.share-card label{display:grid;gap:.3rem;width:max-content;margin-top:.8rem;font-weight:750}.share-card select{min-height:2.7rem;padding:.55rem;border:1px solid var(--color-border);background:#fff;font:inherit}.share-card>div:last-child>strong,.share-card>div:last-child>small{display:block}.share-card>div:last-child>strong{font-family:Georgia,'Times New Roman',serif;font-size:2rem;font-weight:500}.error{color:#b42318}@media(max-width:1050px){.stats{grid-template-columns:1fr 1fr}.stats .focus{grid-column:1/-1}}@media(max-width:700px){.stats,.performance{grid-template-columns:1fr}.stats .focus{grid-column:auto}.toolbar{align-items:stretch;flex-direction:column}.toolbar button{width:100%}.share-card{grid-template-columns:1fr}.card dl div{align-items:flex-start;flex-direction:column;gap:.25rem}}
</style>
