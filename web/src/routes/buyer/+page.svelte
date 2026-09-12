<script lang="ts">
 import RepaymentCustomer from '$lib/components/RepaymentCustomer.svelte';
 import IdentityChecks from '$lib/components/IdentityChecks.svelte';

 import { page } from '$app/state';
 const businessQuery=$derived(page.url.searchParams.get('business_id')?`?business_id=${encodeURIComponent(page.url.searchParams.get('business_id')!)}`:'');
 import { adminPost } from '$lib/admin-client';
 let checkingVerification=$state(false);
 async function checkVerification(){if(checkingVerification)return;checkingVerification=true;error='';try{await adminPost(`/api/v1/buyer/me/verification/refresh${businessQuery}`,{});await load()}catch(cause){error=cause instanceof Error?cause.message:'We could not check verification.'}finally{checkingVerification=false}}
	import { formatKobo, sumKobo } from '$lib/money';
	import { productLabel } from '$lib/product-language';
	import { kobo, saleView } from '$lib/records';
 import { checkedJSON, LatestRequest, record, rows, text, publicError } from '$lib/api/reliable';
	import FeedbackPrompt from '$lib/components/FeedbackPrompt.svelte';

	let portal = $state<any>(null);
	let requests = $state<any[]>([]);
	let error = $state('');
	let balancesUnavailable = $state(false);
	let loading = $state(true);
 let paymentDays=$state<any[]>([]), datesUnavailable=$state(true);
	// A sale whose obligation did not load contributes nothing we can vouch for,
	// so the total is reported as unconfirmed rather than quietly understated.
	const outstanding = $derived(balancesUnavailable ? null : sumKobo(requests.map((item) => item.obligation ? item.obligation.outstanding_kobo : (item.request?.state === 'DRAFT' || item.request?.state === 'SENT' || item.request?.state === 'BUYER_REVIEWING' ? 0 : null))));
	const pending = $derived(requests.filter((item)=>['SENT','BUYER_REVIEWING','PENDING_BUYER_CONFIRMATION'].includes(String(item.request?.state??item.state??'').toUpperCase())));
	const openBalances = $derived(requests.filter((item)=>Number(item.obligation?.outstanding_kobo??0)>0));

 const nextPayment=$derived([...paymentDays].filter(item=>item.next_due_at&&Number(item.outstanding_kobo)>0).sort((a,b)=>Date.parse(a.next_due_at)-Date.parse(b.next_due_at))[0]??null);
 const overdueCount=$derived(paymentDays.filter(item=>item.overdue&&Number(item.outstanding_kobo)>0).length);
 function decodeDays(value:unknown){return rows('obligations',value=>{
  const item=record(value);text(item.obligation_id);kobo(item.outstanding_kobo);kobo(item.next_due_kobo);
  if(typeof item.overdue!=='boolean'||(item.next_due_at!=null&&!Number.isFinite(Date.parse(text(item.next_due_at)))))throw new Error('Invalid payment dates');return item;
 })(value);}

 const reads=new LatestRequest();
 async function load(query=businessQuery){
  const request=reads.begin();loading=true;error='';balancesUnavailable=true;datesUnavailable=true;paymentDays=[];
  try{
   const [account,sales,days]=await Promise.allSettled([
    checkedJSON(`/api/v1/buyer/me${query}`,value=>{const response=record(value),portal=record(response.portal);text(record(portal.person).full_name);text(record(portal.business).legal_name);return {...portal,verification_current:response.verification_current===true};},{signal:request.signal}),
    checkedJSON('/api/v1/buyer/credit-requests',rows('requests',saleView),{signal:request.signal}),
    checkedJSON('/api/v1/buyer/history',decodeDays,{signal:request.signal})
   ]);
   if(!request.current())return;
   if(account.status==='rejected'){portal=null;error=publicError(account.reason,'your account');return;}
   portal=account.value;
   if(days.status==='fulfilled'){paymentDays=days.value;datesUnavailable=false;}else error=publicError(days.reason,'your payment dates');
   if(sales.status==='fulfilled'){requests=sales.value;balancesUnavailable=false;}
   else{requests=[];error=publicError(sales.reason,'your balances');}
  }catch(cause){if(request.current())error=publicError(cause,'your account');}
  finally{if(request.current())loading=false;}
 }
 $effect(()=>{const query=businessQuery;void load(query);return()=>reads.cancel();});

</script>

<svelte:head><title>Customer account — Kredit</title></svelte:head>

<main class="shell buyer-home">
	{#if portal}
		<header class="buyer-head"><div><p class="eyebrow">Your Kredit account</p><h1>Here is what you owe.</h1><p>Signed in as <strong>{portal.person.full_name}</strong> for {portal.business.legal_name}.</p></div>{#if pending.length}<a class="primary" href="/buyer/requests">Read {pending.length} sale{pending.length===1?'':'s'} waiting for you</a>{/if}</header>
		{#if !portal.verification_current}<section><h2>Your account checks need attention</h2><p>Your details are saved. Identity, business and authority checks must be current before you can accept a sale. If a check has expired, contact support to renew it.</p><button disabled={checkingVerification||loading} onclick={checkVerification}>{checkingVerification?'Checking…':'Check verification status'}</button></section>{/if}
        {#if error}<p role="alert">{error}</p><button onclick={()=>void load()} disabled={loading}>Try again</button>{/if}{#if loading}<p role="status">Opening your balances…</p>{:else}
			<section class="owe-hero" aria-label="What you owe">
				<div class="owed"><span>You owe</span><strong>{outstanding === null ? 'Not confirmed' : formatKobo(outstanding)}</strong><small>{outstanding === null ? 'We could not check every sale. Open them one by one.' : `Across ${openBalances.length} sale${openBalances.length === 1 ? '' : 's'}`}</small></div>
				{#if outstanding===null||datesUnavailable}<p class="due-strip calm">Payment dates could not be confirmed.</p>{:else if overdueCount}<a class="due-strip late" href="/buyer/obligations"><span>{overdueCount} payment day{overdueCount===1?' has':'s have'} passed</span><em>See what is late →</em></a>{:else if nextPayment}<a class="due-strip" href="/buyer/obligations"><span>Next: {formatKobo(nextPayment.next_due_kobo)}</span><em>by {new Date(nextPayment.next_due_at).toLocaleDateString('en-NG',{timeZone:'Africa/Lagos'})} →</em></a>{:else if outstanding>0n}<a class="due-strip" href="/buyer/obligations">Open your sales to check payment days →</a>{:else}<p class="due-strip calm">Nothing is due right now.</p>{/if}
			</section>
			<section class="next-actions">
				<div><p class="eyebrow">What needs you</p><h2>{pending.length ? 'Read the sale before you say yes.' : overdueCount ? 'A payment day has passed.' : outstanding === null ? 'We could not confirm your total.' : outstanding > 0n ? 'See your payment schedule.' : 'You owe nothing right now.'}</h2></div>
				<div class="action-copy">{#if pending.length}<p>A seller has sent you a sale. Read what they wrote: the goods, the money, the day. Nobody can force you into it. You decide.</p><a href="/buyer/requests">Read the sale →</a>{:else if overdueCount}<p>Open your balances and see what is late, what you have already paid and what is still left.</p><a href="/buyer/obligations">See what is late →</a>{:else if outstanding === null}<p>We could not confirm one or more of your balances. Open each sale instead of trusting the total above.</p><a href="/buyer/obligations">Check my balances →</a>{:else if outstanding > 0n}<p>You can open your balances any time and see what you agreed to, what you have paid and what is left.</p><a href="/buyer/obligations">See my balances →</a>{:else}<p>If a seller sends you a new sale, it will show up right here.</p><a href="/buyer/history">See how I have paid before →</a>{/if}</div>
			</section>
		{/if}
		<FeedbackPrompt area="buyer" />
	{:else if error}
		<p class="eyebrow">Your Kredit account</p>
		<h1>We could not open your account.</h1>
		<p class="error" role="alert">{error}</p>
		<button onclick={()=>void load()} disabled={loading}>Try again</button><p><a href="/app?next=%2Fbuyer">Sign in to your account →</a></p><p class="help">Look for the link in the message your seller sent you. If it will not open, just ask them for a fresh one. And never give your sign-in code to anybody, not even to somebody who says they are from Kredit.</p>
	{:else}
		<p class="eyebrow">Your Kredit account</p>
		<h1>Opening your account…</h1>
	{/if}
</main>

<style>.owe-hero{margin:1.5rem 0}.owe-hero .owed{background:#17181b;color:#fff;padding:1.6rem 1.4rem 1.5rem}.owe-hero .owed span{display:block;font-size:.7rem;font-weight:800;letter-spacing:.1em;text-transform:uppercase;color:#b9b8b3}.owe-hero .owed strong{display:block;font-family:var(--font-serif);font-size:clamp(2.6rem,7vw,4rem);font-weight:500;line-height:1;margin:.5rem 0 .4rem;letter-spacing:-.03em;font-variant-numeric:tabular-nums}.owe-hero .owed small{color:#b9b8b3;font-size:.82rem}.due-strip{display:flex;align-items:center;justify-content:space-between;gap:1rem;margin:0;padding:.85rem 1.4rem;background:#eef0ff;color:#2738d6;font-weight:750;font-size:.9rem;text-decoration:none}.due-strip em{font-style:normal;font-variant-numeric:tabular-nums}.due-strip.late{background:var(--color-overdue);color:#fff}.due-strip.calm{color:#5f645f;background:#e8e3d9}.buyer-home{padding-bottom:6rem}.eyebrow{color:#2738d6;font-weight:800;text-transform:uppercase;letter-spacing:.1em;font-size:.72rem}.buyer-head{display:flex;justify-content:space-between;align-items:end;gap:2rem;padding:3rem 0 2rem;border-bottom:3px solid #17181b}.buyer-head h1,.buyer-home>h1{max-width:13ch;margin:.5rem 0;font-family:var(--font-serif);font-size:clamp(3rem,7vw,5.8rem);font-weight:500;line-height:.92;letter-spacing:-.06em}.buyer-head p{color:#656862}.next-actions{display:grid;grid-template-columns:1fr 1fr;gap:clamp(2rem,8vw,8rem);margin:3rem 0;padding:2rem 0;border-top:1px solid #cfc9be;border-bottom:1px solid #cfc9be}.next-actions h2{max-width:12ch;margin:.5rem 0;font-family:var(--font-serif);font-size:clamp(2.2rem,4vw,3.8rem);font-weight:500;line-height:.96;letter-spacing:-.04em}.action-copy{padding-top:.5rem}.action-copy p{max-width:36rem;color:#626762;line-height:1.7}.action-copy a{color:#2738d6;font-weight:800}.error{color:#b42318}.help{max-width:34rem;color:var(--color-muted);line-height:1.65}@media(max-width:900px){}@media(max-width:720px){.buyer-head{display:block}.buyer-head .primary{margin-top:1rem}.next-actions{grid-template-columns:1fr}.next-actions{gap:1rem}}
</style>

<IdentityChecks />
{#if portal?.business?.id}<RepaymentCustomer businessID={portal.business.id} businessName={portal.business.legal_name} />{/if}
