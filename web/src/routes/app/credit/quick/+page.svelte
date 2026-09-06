<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';
	import { parseNaira, verbalizeNaira } from '$lib/money';
	import { productLabel } from '$lib/product-language';

	let organizations:any[]=$state([]), customers:any[]=$state([]), organizationID=$state(''), selectedBuyer=$state('');
	let buyerUserID=$state(''), buyerBusinessID=$state(''), buyerLegalName=$state('');
	let principal=$state(''), goods=$state(''), dueDate=$state(''), busy=$state(false), error=$state('');
	let step=$state(1), createRequestKey=$state('');
	let amountWords=$derived(verbalizeNaira(parseNaira(principal)));
	let selectedCustomer=$derived(customers.find((item)=>item.buyer_user_id===selectedBuyer));
	let customerWarning=$derived(selectedCustomer?.has_overdue_obligations || selectedCustomer?.overdue_count > 0 || selectedCustomer?.has_network_overdue ? `This customer currently has ${selectedCustomer?.overdue_count ?? 'active'} overdue payment(s) recorded across Kredit.` : '');

	async function loadCustomers(){
		customers=[]; selectedBuyer=''; buyerUserID=''; buyerBusinessID=''; buyerLegalName='';
		if(!organizationID)return;
		const response=await fetch(`/api/v1/organizations/${organizationID}/customers`,{credentials:'include'});
		if(response.ok)customers=(await response.json()).customers??[];
	}
	function chooseBuyer(){
		const customer=customers.find((item)=>item.buyer_user_id===selectedBuyer);
		buyerUserID=customer?.buyer_user_id??'';
		buyerBusinessID=customer?.buyer_business_id??'';
		buyerLegalName=customer?.legal_name??customer?.trading_name??'';
	}
	function next(){
		error='';
		if(step===1&&!buyerUserID){error='Choose a customer before you continue.';return;}
		if(step===2&&(!goods.trim()||parseNaira(principal)<=0)){error='Add the goods and a valid amount before you continue.';return;}
		if(step===3&&!dueDate){error='Choose the day your customer should pay.';return;}
		step=Math.min(4,step+1);
	}
	function back(){error='';step=Math.max(1,step-1)}
	function collectionDate(){
		const date=new Date(`${dueDate}T23:59:00`);
		date.setDate(date.getDate()+1);
		return date.toISOString();
	}
	async function submit(){
		error=''; busy=true;
		try{
			const amount=parseNaira(principal);
			if(!organizationID||!buyerUserID||!buyerBusinessID||!buyerLegalName||!goods||!dueDate||amount<=0)throw new Error('Some sale details are missing. Go back and check the sale.');
			if(!createRequestKey)createRequestKey=idempotencyKey();
			const response=await fetch(`/api/v1/organizations/${organizationID}/credit-requests`,{
				method:'POST',credentials:'include',headers:{'Content-Type':'application/json','Idempotency-Key':createRequestKey,...csrfHeaders()},
				body:JSON.stringify({buyer_user_id:buyerUserID,buyer_business_id:buyerBusinessID,buyer_legal_name:buyerLegalName,principal_kobo:amount,goods_description:goods,invoice_reference:'',invoice_document_hash:'',due_date:dueDate,grace_hours:24,collection_at:collectionDate(),schedule_type:'one_time',schedule_count:2,schedule_cadence:'monthly',month_end_policy:'last_day',custom_schedule_items:[]})
			});
			const body=await response.json().catch(()=>({}));
			if(!response.ok){if(response.status<500&&(body.title??body.code)!=='idempotency_in_progress')createRequestKey='';throw new Error(body.detail??'The sale could not be saved.');}
			createRequestKey='';
			await goto(`/app/credit/${body.request.id}?organization=${organizationID}`);
		}catch(cause){error=cause instanceof Error?cause.message:'The sale could not be saved.'}finally{busy=false}
	}
	onMount(async()=>{
		const response=await fetch('/api/v1/organizations',{credentials:'include'});
		if(response.status===401){location.assign('/app');return;}
		if(!response.ok){error='We could not load your business account.';return;}
		organizations=(await response.json()).organizations??[];
		organizationID=organizations[0]?.id??'';
		await loadCustomers();
		const params=new URLSearchParams(window.location.search);
		selectedBuyer=params.get('customer')??'';
		goods=params.get('goods')??'';
		principal=params.get('amount')??'';
		if(selectedBuyer)chooseBuyer();
		if(buyerUserID&&goods&&parseNaira(principal)>0)step=3;
		else if(buyerUserID)step=2;
	});
</script>

<svelte:head><title>Add a credit sale — Kredit</title></svelte:head>
<main class="shell quick-sale">
	<header class="page-head">
		<div><p class="eyebrow">New credit sale</p><h1>Four things.<br />Then send it.</h1><p class="lede">Choose the customer, describe the goods, enter the amount and set the payment day. Kredit handles the rest of the record.</p></div>
		<a href="/app/credit/new?advanced=1">Need instalments or invoice details? Use the full form →</a>
	</header>

	<nav class="steps" aria-label="Sale steps">
		{#each ['Customer','Goods & amount','Payment day','Review'] as label,index}
			<button class:active={step===index+1} class:done={step>index+1} type="button" onclick={()=>{if(index+1<step)step=index+1}}><span>{step>index+1?'✓':index+1}</span>{label}</button>
		{/each}
	</nav>

	{#if error}<p class="error" role="alert">{error}</p>{/if}
	<section class="sale-card">
		{#if step===1}
			<div class="stage-copy"><p class="eyebrow">01 — Customer</p><h2>Who is taking the goods?</h2><p>Choose the customer who will receive and approve this sale.</p></div>
			<div class="fields">
				{#if organizations.length>1}<label>Your business<select bind:value={organizationID} onchange={loadCustomers}>{#each organizations as org}<option value={org.id}>{org.trading_name||org.legal_name}</option>{/each}</select></label>{/if}
				{#if customers.length}<label>Customer<select bind:value={selectedBuyer} onchange={chooseBuyer}><option value="">Choose a customer</option>{#each customers as customer}<option value={customer.buyer_user_id}>{customer.trading_name||customer.legal_name}</option>{/each}</select></label>{:else}<div class="empty-inline"><strong>No customers yet.</strong><p>Add the customer first. They will receive a private link to check their details.</p><a class="primary" href="/app/customers/new">Add customer</a></div>{/if}
				{#if selectedCustomer}<div class="customer-card"><span>Customer status</span><strong>{selectedCustomer.trading_name||selectedCustomer.legal_name}</strong><small>{productLabel(selectedCustomer.state??selectedCustomer.status,'Customer added')}</small>{#if customerWarning}<p class="warning">⚠ {customerWarning}</p>{/if}</div>{/if}
			</div>
		{:else if step===2}
			<div class="stage-copy"><p class="eyebrow">02 — Goods & amount</p><h2>What are you giving them?</h2><p>Keep this description clear enough that both sides know exactly what the sale covers.</p></div>
			<div class="fields"><label>Goods<textarea bind:value={goods} rows="5" placeholder="For example: 40 cartons of 5L cooking oil"></textarea></label><label>Amount to pay (₦)<input bind:value={principal} inputmode="decimal" placeholder="1,200,000" />{#if amountWords}<small class="amount-words">{amountWords}</small>{/if}</label></div>
		{:else if step===3}
			<div class="stage-copy"><p class="eyebrow">03 — Payment day</p><h2>When should they pay?</h2><p>Choose the agreed payment date. The quick flow uses a 24-hour grace period before any permitted collection attempt.</p></div>
			<div class="fields"><label>Payment due date<input type="date" bind:value={dueDate} /></label><div class="trust-note"><strong>What Kredit will not do</strong><p>Kredit will not start a bank debit before the agreed date and grace period. Bank debit also requires valid authorization and the applicable payment/dispute checks.</p></div></div>
		{:else}
			<div class="stage-copy"><p class="eyebrow">04 — Review</p><h2>Make sure both sides will see the same sale.</h2><p>Nothing is sent to the customer until you save the sale and open it to send for approval.</p></div>
			<div class="review-card"><dl><div><dt>Customer</dt><dd>{buyerLegalName}</dd></div><div><dt>Goods</dt><dd>{goods}</dd></div><div><dt>Amount</dt><dd>{new Intl.NumberFormat('en-NG',{style:'currency',currency:'NGN'}).format(parseNaira(principal)/100)}</dd></div><div><dt>Pay by</dt><dd>{new Date(`${dueDate}T12:00:00`).toLocaleDateString('en-NG',{day:'numeric',month:'long',year:'numeric'})}</dd></div></dl><div class="trust-strip"><span>✓ Customer reviews before accepting</span><span>✓ Sale details stay on record</span><span>✓ Payments reduce the balance</span></div></div>
		{/if}
	</section>

	<footer class="actions"><div>{#if step>1}<button class="secondary" type="button" onclick={back}>← Back</button>{/if}</div>{#if step<4}<button class="primary" type="button" onclick={next}>Continue →</button>{:else}<button class="primary" type="button" onclick={submit} disabled={busy}>{busy?'Saving sale…':'Save sale and continue →'}</button>{/if}</footer>
</main>

<style>
	.quick-sale{max-width:72rem;padding-bottom:6rem}.page-head{display:grid;grid-template-columns:1fr auto;gap:3rem;align-items:end;padding:3rem 0 2rem;border-bottom:3px solid #17181b}.page-head h1{max-width:11ch;margin:.5rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(3.2rem,7vw,6rem);font-weight:500;line-height:.9;letter-spacing:-.06em}.page-head>a{max-width:18rem;color:#2738d6;font-size:.82rem;font-weight:750;line-height:1.5}.steps{display:grid;grid-template-columns:repeat(4,1fr);margin:1.5rem 0 2rem;border-top:1px solid #cfc9be;border-left:1px solid #cfc9be}.steps button{display:flex;align-items:center;gap:.65rem;min-height:3.3rem;padding:.7rem;border:0;border-right:1px solid #cfc9be;border-bottom:1px solid #cfc9be;background:#fffdf8;color:#6a6c67;text-align:left;font:inherit;font-size:.78rem;font-weight:750}.steps button span{display:grid;place-items:center;width:1.5rem;height:1.5rem;border:1px solid #aaa69e;border-radius:50%;font-size:.68rem}.steps button.active{background:#17181b;color:#fff}.steps button.active span{border-color:#e85f3d;background:#e85f3d;color:#fff}.steps button.done{color:#2738d6}.steps button.done span{border-color:#2738d6;background:#2738d6;color:#fff}.sale-card{display:grid;grid-template-columns:.75fr 1.25fr;gap:clamp(2rem,6vw,6rem);min-height:27rem;padding:clamp(1.5rem,5vw,3.5rem);border:1px solid #cfc9be;background:#fffdf8;box-shadow:12px 12px 0 #2738d6}.stage-copy h2{max-width:11ch;margin:.5rem 0 1rem;font-family:Georgia,'Times New Roman',serif;font-size:clamp(2.2rem,4vw,3.7rem);font-weight:500;line-height:.95;letter-spacing:-.045em}.stage-copy>p:last-child{max-width:30rem;color:#6a6c67;line-height:1.7}.fields{display:grid;align-content:start;gap:1.2rem}.fields label{display:grid;gap:.45rem;font-weight:750}.fields input,.fields select,.fields textarea{box-sizing:border-box;width:100%;min-height:3rem;padding:.8rem;border:1px solid #aaa69e;border-radius:0;background:#fff;font:inherit}.fields textarea{resize:vertical}.amount-words{color:#2738d6;line-height:1.5}.customer-card,.trust-note,.empty-inline{padding:1rem;border-left:4px solid #2738d6;background:#eef0ff}.customer-card{display:grid;gap:.35rem}.customer-card span,.customer-card small{color:#636863;font-size:.72rem}.warning{margin:.5rem 0 0;padding:.65rem;background:#fff5dd;color:#7a4a00;font-size:.8rem;line-height:1.5}.trust-note{border-left-color:#16794e;background:#ecf8f1}.trust-note p,.empty-inline p{margin:.4rem 0;color:#626762;line-height:1.6}.review-card{display:grid;gap:1.5rem}.review-card dl{margin:0;border-top:1px solid #cfc9be}.review-card dl div{display:grid;grid-template-columns:8rem 1fr;gap:1rem;padding:1rem 0;border-bottom:1px solid #cfc9be}.review-card dt{color:#686a66;font-size:.75rem;font-weight:750}.review-card dd{margin:0;font-weight:700;overflow-wrap:anywhere}.trust-strip{display:grid;gap:.45rem;padding:1rem;color:#fff;background:#17181b;font-size:.78rem}.actions{display:flex;justify-content:space-between;gap:1rem;margin-top:2rem}.actions button{min-height:3rem;padding:.75rem 1.1rem;border-radius:0;font:inherit;font-weight:800}.secondary{border:1px solid #aaa69e;background:#fff}.error{margin:1rem 0;padding:1rem;border-left:4px solid #b42318;background:#fff0ed;color:#8a1c14;font-weight:700}@media(max-width:760px){.page-head,.sale-card{grid-template-columns:1fr}.page-head{gap:1rem}.steps{grid-template-columns:1fr 1fr}.steps button{min-height:2.9rem}.sale-card{min-height:auto;box-shadow:7px 7px 0 #2738d6}.actions{position:sticky;bottom:0;z-index:5;margin-inline:-1rem;padding:1rem;background:rgb(245 242 234 / .96);border-top:1px solid #cfc9be}.actions button{min-width:8rem}.review-card dl div{grid-template-columns:1fr;gap:.25rem}}
</style>