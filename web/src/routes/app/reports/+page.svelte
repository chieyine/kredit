<script lang="ts">
	import { onMount } from 'svelte';
	import { exactKobo, formatKobo, sumKobo, type KoboValue } from '$lib/money';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';
 import { checkedJSON, LatestRequest, record, rows, publicError } from '$lib/api/reliable';
 import { organization, kobo, paymentRow } from '$lib/records';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import ShareActions from '$lib/components/ShareActions.svelte';
	type Organization = { id: string; legal_name: string; trading_name?: string };
	type Summary = { obligation_count: number; outstanding_kobo: KoboValue; overdue_kobo: KoboValue; voluntary_paid_kobo: KoboValue; collected_paid_kobo: KoboValue };
	let organizations = $state<Organization[]>([]);
	let organizationID = $state('');
	let summary = $state<Summary | null>(null);
	let buckets = $state<Record<string, KoboValue>>({});
	let fees = $state<{ total_fees_kobo: KoboValue } | null>(null);
	let payments = $state<any[]>([]);
	let sharePeriod = $state('today');
	let loading = $state(true);
	let error = $state('');
	let exportKey = '';
	let exporting = $state(false);
	const money = (value: KoboValue) => formatKobo(value);
	// Money is summed in exact kobo. A figure the API did not return is null, not
	// zero: reporting an unread total as ₦0 would read as a real, settled balance.
	const paidTotal = $derived(sumKobo([summary?.voluntary_paid_kobo, summary?.collected_paid_kobo]));
	const trackedValue = $derived(sumKobo([paidTotal, summary?.outstanding_kobo]));
	// Percentages are presentation, so they are the one place a ratio is taken —
	// from BigInt inputs, and only when both sides are known.
	const percent = (part: KoboValue, whole: KoboValue) => {
		const a = exactKobo(part), b = exactKobo(whole);
		if (a === null || b === null || b <= 0n) return null;
		return Number((a * 100n) / b);
	};
	const receivedRate = $derived(percent(paidTotal, trackedValue));
	const overdueShare = $derived(percent(summary?.overdue_kobo, summary?.outstanding_kobo));
	const recognizedPayments = $derived(payments.filter((payment)=>['recognized','confirmed','paid'].includes(String(payment.state??payment.status??'').toLowerCase())));
	const averagePayment = $derived.by(() => {
		if (!recognizedPayments.length) return null;
		const total = sumKobo(recognizedPayments.map((payment) => payment.amount_kobo));
		const exact = exactKobo(total);
		return exact === null ? null : exact / BigInt(recognizedPayments.length);
	});
 const reportsRequest=new LatestRequest(), businessRequest=new LatestRequest();
 async function load() {
  const request=reportsRequest.begin(), scope=organizationID;
  loading=true;error='';summary=null;buckets={};fees=null;payments=[];
  if(!scope){loading=false;return;}
  const root=`/api/v1/organizations/${encodeURIComponent(scope)}`;
  try{
   const [nextSummary,nextBuckets,nextFees,nextPayments]=await Promise.all([
    checkedJSON(`${root}/reports/receivables`,value=>{const data=record(record(value).summary);if(!Number.isSafeInteger(data.obligation_count)||Number(data.obligation_count)<0)throw new Error('Incomplete count');return {obligation_count:Number(data.obligation_count),outstanding_kobo:kobo(data.outstanding_kobo),overdue_kobo:kobo(data.overdue_kobo),voluntary_paid_kobo:kobo(data.voluntary_paid_kobo),collected_paid_kobo:kobo(data.collected_paid_kobo)};},{signal:request.signal}),
    checkedJSON(`${root}/reports/ageing`,value=>Object.fromEntries(Object.entries(record(record(value).buckets)).map(([key,value])=>[key,kobo(value)])),{signal:request.signal}),
    checkedJSON(`${root}/reports/fees`,value=>({total_fees_kobo:kobo(record(value).total_fees_kobo)}),{signal:request.signal}),
    checkedJSON(`${root}/payments`,rows('payments',paymentRow),{signal:request.signal})
   ]);
   if(!request.current()||organizationID!==scope)return;
   summary=nextSummary;buckets=nextBuckets;fees=nextFees;payments=nextPayments;
  }catch(cause){if(request.current())error=publicError(cause,'your reports');}
  finally{if(request.current())loading=false;}
 }
 async function initialize(){
  const request=businessRequest.begin();loading=true;error='';
  try{
   const next=await checkedJSON('/api/v1/organizations',rows('organizations',organization),{signal:request.signal});
   if(!request.current())return;organizations=next;
   const requested=new URLSearchParams(location.search).get('organization');
   organizationID=next.find(item=>item.id===(requested||organizationID))?.id??next[0]?.id??'';await load();
  }catch(cause){if(request.current()){error=publicError(cause,'your businesses');loading=false;}}
 }
	// The business day starts at midnight in Lagos wherever the phone happens to be.
	function lagosDayStart(daysBack: number): number {
		const parts = new Intl.DateTimeFormat('en-CA', { timeZone: 'Africa/Lagos', year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date());
		return new Date(`${parts}T00:00:00+01:00`).getTime() - daysBack * 86400000;
	}
	const shared = $derived.by(() => {
		const start = lagosDayStart(sharePeriod === 'today' ? 0 : 6);
		const received = recognizedPayments.filter((payment) => {
			const at = Date.parse(payment.paid_at ?? payment.created_at ?? '');
			return Number.isFinite(at) && at >= start && at <= Date.now();
		});
		return { count: received.length, total: sumKobo(received.map((payment) => payment.amount_kobo)) };
	});
	const summaryText = $derived(`Kredit ${sharePeriod === 'today' ? 'today' : 'last 7 days'}: ${shared.count} payment${shared.count === 1 ? '' : 's'} received, ${money(shared.total)} in total. ${money(summary?.outstanding_kobo)} still owed; ${money(summary?.overdue_kobo)} overdue.`);
	function bucketName(bucket:string){return ({current:'Not overdue',not_due:'Not overdue','1_7':'1–7 days overdue','8_30':'8–30 days overdue','31_60':'31–60 days overdue', '61_plus':'61+ days overdue',paid:'Paid'} as Record<string,string>)[bucket.toLowerCase()] ?? bucket.replaceAll('_',' ')}
	async function exportCSV() {
		if (exporting||loading||!summary||!organizationID) return;
		exporting = true;
		exportKey ||= idempotencyKey();
		try {
		const response = await fetch(`/api/v1/organizations/${organizationID}/reports/exports?format=csv`, { method: 'POST', signal: AbortSignal.timeout(20000), credentials: 'include', redirect: 'error', headers: { ...csrfHeaders(), 'Idempotency-Key': exportKey } });
		if (!response.ok) { error = 'We could not create that file. Please try again.'; return; }
		if (!response.headers.get('Content-Type')?.toLowerCase().startsWith('text/csv')) throw new Error('Unexpected export');
		const objectURL = URL.createObjectURL(await response.blob());
		exportKey = '';
		const link = document.createElement('a');
		link.href = objectURL;
		link.download = 'kredit-receivables.csv';
		document.body.append(link);
		link.click();
		link.remove();
		// Revoking immediately can abort the download before the browser reads it.
		setTimeout(() => URL.revokeObjectURL(objectURL), 30_000);
		} catch { error = 'We could not download the report. Please try again.'; } finally {
			exporting = false;
		}
	}
 onMount(()=>{void initialize();return()=>{businessRequest.cancel();reportsRequest.cancel();};});

</script>
<svelte:head><title>Reports — Kredit</title></svelte:head>
<main class="shell workspace reports"><header><p class="eyebrow">Business performance</p><h1>Where all your money is sitting.</h1><p class="lede">What you are owed, what is late, what has come in, how old the debts are and what Kredit has charged you.</p></header>
	<div class="toolbar"><label>Business<select disabled={exporting} bind:value={organizationID} onchange={load}>{#each organizations as organization}<option value={organization.id}>{organization.trading_name || organization.legal_name}</option>{/each}</select></label><button onclick={exportCSV} disabled={!summary || exporting || loading || Boolean(error)}>{exporting ? 'Preparing…' : 'Download CSV'}</button></div>
	{#if error}<p class="error" role="alert">{error}</p><button onclick={initialize}>Try again</button>{:else if loading}<div class="loading" aria-live="polite"><Skeleton rows={4} tall /></div>{:else if !organizations.length}<p>Add your business to see its reports. <a href="/app/overview">Set up your business →</a></p>{:else if summary}
		<section class="stats"><article class="focus"><span>Owed to you now</span><strong>{money(summary.outstanding_kobo)}</strong><small>{summary.obligation_count} open sale{summary.obligation_count===1?'':'s'}</small></article><article><span>Late right now</span><strong>{money(summary.overdue_kobo)}</strong><small>{overdueShare === null ? 'Share unavailable' : `${overdueShare}% of what you are owed`}</small></article><article><span>Received</span><strong>{money(paidTotal)}</strong><small>{receivedRate === null ? 'Share unavailable' : `${receivedRate}% of everything Kredit is following`}</small></article><article><span>Average payment</span><strong>{money(averagePayment)}</strong><small>{recognizedPayments.length} payment{recognizedPayments.length===1?'':'s'} recorded</small></article><article><span>Kredit fees</span><strong>{money(fees?.total_fees_kobo)}</strong><small>What Kredit has charged you, after any reductions</small></article></section>
		<section class="performance"><div><p class="eyebrow">How collection is going</p><h2>{receivedRate === null ? 'This share could not be worked out.' : `${receivedRate}% of the money Kredit is following has reached you.`}</h2><p>This compares money already received with money received plus money still owed. Use it to get a feel for how things are going. It is not an accounting statement.</p></div><div class="meter" aria-label={receivedRate === null ? 'Share unavailable' : `${receivedRate}% received`}><div><span style={`width:${receivedRate ?? 0}%`}></span></div><p><b>{money(paidTotal)}</b> received <span>·</span> <b>{money(summary.outstanding_kobo)}</b> still owed</p></div></section>
		<section class="card ageing"><h2>How old is the money you are owed?</h2><p>The longer a debt sits, the harder it usually gets to collect. Chase the old ones first.</p><dl>{#each Object.entries(buckets) as [bucket, amount]}<div><dt>{bucketName(bucket)}</dt><dd>{money(amount)}</dd></div>{/each}</dl><a href={`/app/overdue?organization=${encodeURIComponent(organizationID)}`}>See who is late →</a></section>
		<section class="card share-card"><div><h2>Send someone a money update</h2><p>Send a short summary to a partner or a staff member, without giving them your whole book.</p><label>Period<select bind:value={sharePeriod}><option value="today">Today</option><option value="week">Last 7 days</option></select></label></div><div><strong>{money(shared.total)}</strong><small>{shared.count} payment{shared.count===1?'':'s'} received</small><ShareActions compact title="Kredit money update" text={summaryText}/></div></section>
	{/if}
</main>
<style>.reports>header{max-width:62rem}.reports h1{max-width:15ch;font-family:var(--font-serif);font-size:clamp(3rem,6vw,5.4rem);font-weight:500;line-height:.92;letter-spacing:-.055em}.toolbar{display:flex;justify-content:space-between;gap:1rem;align-items:end;margin:2rem 0;padding:1rem;flex-wrap:wrap;color:#fff;background:#17181b}.toolbar label{display:grid;gap:.4rem;font-weight:700}.toolbar select,.toolbar button{min-height:3rem;padding:.75rem;border:1px solid #4b4c51;border-radius:0;background:#fff;color:#17181b}.toolbar button{background:#2738d6;color:white;border-color:#2738d6;font-weight:750}.stats{display:grid;grid-template-columns:1.2fr repeat(4,1fr);border-top:1px solid var(--color-border);border-left:1px solid var(--color-border)}.stats article,.card{padding:1.25rem;border:0;border-right:1px solid var(--color-border);border-bottom:1px solid var(--color-border);border-radius:0;background:var(--color-surface)}.stats article.focus{color:#fff;background:#2738d6}.stats span,.stats small{display:block;color:var(--color-muted)}.stats .focus span,.stats .focus small{color:#d8dbff}.stats strong{display:block;margin:.45rem 0;font-family:var(--font-serif);font-size:clamp(1.5rem,2.5vw,2.2rem);font-weight:500}.performance{display:grid;grid-template-columns:1fr 1fr;gap:clamp(2rem,7vw,7rem);align-items:center;margin:2rem 0;padding:2rem;border:1px solid var(--color-border);background:#17181b;color:#fff}.performance h2{max-width:13ch;margin:.5rem 0;font-family:var(--font-serif);font-size:clamp(2rem,4vw,3.5rem);font-weight:500;line-height:.98}.performance>div>p:last-child{color:#b6b6b1;line-height:1.65}.meter>div{height:1rem;background:#44454a}.meter>div span{display:block;height:100%;background:#e85f3d}.meter p{display:flex;flex-wrap:wrap;gap:.45rem;color:#b6b6b1}.meter b{color:#fff}.card{margin-top:1rem;border:1px solid var(--color-border)}.ageing>p{color:var(--color-muted)}.card dl{display:grid;gap:.7rem}.card dl div{display:flex;justify-content:space-between;gap:2rem;border-bottom:1px solid var(--color-border);padding-bottom:.7rem}.card dd{font-weight:700}.ageing>a{display:inline-block;margin-top:1rem;font-weight:750}.share-card{display:grid;grid-template-columns:1fr auto;gap:2rem;align-items:center;margin-top:2rem;background:#ebe7de}.share-card h2,.share-card p{margin:.25rem 0}.share-card label{display:grid;gap:.3rem;width:max-content;margin-top:.8rem;font-weight:750}.share-card select{min-height:2.7rem;padding:.55rem;border:1px solid var(--color-border);background:#fff;font:inherit}.share-card>div:last-child>strong,.share-card>div:last-child>small{display:block}.share-card>div:last-child>strong{font-family:var(--font-serif);font-size:2rem;font-weight:500}.error{color:#b42318}@media(max-width:1050px){.stats{grid-template-columns:1fr 1fr}.stats .focus{grid-column:1/-1}}@media(max-width:700px){.stats,.performance{grid-template-columns:1fr}.stats .focus{grid-column:auto}.toolbar{align-items:stretch;flex-direction:column}.toolbar button{width:100%}.share-card{grid-template-columns:1fr}.card dl div{align-items:flex-start;flex-direction:column;gap:.25rem}}
</style>
