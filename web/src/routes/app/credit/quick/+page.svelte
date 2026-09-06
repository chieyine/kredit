<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { csrfHeaders, idempotencyKey } from '$lib/api/client';
	import { formatKobo, parseNaira, verbalizeNaira } from '$lib/money';
	import { productLabel } from '$lib/product-language';

	const DRAFT_KEY='kredit.quick-sale.draft.v1';
	type QuickSaleDraft={goods:string;principal:string;dueDate:string;updatedAt:string};
	let organizations:any[]=$state([]), customers:any[]=$state([]), organizationID=$state(''), selectedBuyer=$state('');
	let buyerUserID=$state(''), buyerBusinessID=$state(''), buyerLegalName=$state('');
	let principal=$state(''), goods=$state(''), dueDate=$state(''), busy=$state(false), error=$state('');
	let step=$state(1), createRequestKey=$state(''), draftReady=$state(false), recoveredDraft=$state(false);
	let amountWords=$derived(verbalizeNaira(parseNaira(principal)));
	let selectedCustomer=$derived(customers.find((item)=>item.buyer_user_id===selectedBuyer));
	let customerWarning=$derived(selectedCustomer?.has_overdue_obligations || selectedCustomer?.overdue_count > 0 || selectedCustomer?.has_network_overdue ? `Be careful. This customer has ${selectedCustomer?.overdue_count ?? 'unpaid'} late payment(s) on Kredit right now.` : '');

	function restoreDraft(){
		try{
			const raw=sessionStorage.getItem(DRAFT_KEY);
			if(!raw)return;
			const draft=JSON.parse(raw) as Partial<QuickSaleDraft>;
			if(typeof draft.goods==='string')goods=draft.goods;
			if(typeof draft.principal==='string')principal=draft.principal;
			if(typeof draft.dueDate==='string')dueDate=draft.dueDate;
			recoveredDraft=Boolean(goods||principal||dueDate);
		}catch{sessionStorage.removeItem(DRAFT_KEY)}
	}
	function clearDraft(){
		try{sessionStorage.removeItem(DRAFT_KEY)}catch{/* storage may be unavailable */}
		recoveredDraft=false;
	}
	$effect(()=>{
		if(!draftReady)return;
		try{
			if(!goods&&!principal&&!dueDate){sessionStorage.removeItem(DRAFT_KEY);return;}
			const draft:QuickSaleDraft={goods,principal,dueDate,updatedAt:new Date().toISOString()};
			sessionStorage.setItem(DRAFT_KEY,JSON.stringify(draft));
		}catch{/* the form remains usable when browser storage is unavailable */}
	});

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
		if(step===1&&!buyerUserID){error='Choose the customer first.';return;}
		if(step===2&&(!goods.trim()||parseNaira(principal)<=0)){error='Write what the goods are, and how much they must pay.';return;}
		if(step===3&&!dueDate){error='Choose the day your customer must pay.';return;}
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
			if(!organizationID||!buyerUserID||!buyerBusinessID||!buyerLegalName||!goods||!dueDate||amount<=0)throw new Error('Something is missing. Go back and check the sale.');
			if(!createRequestKey)createRequestKey=idempotencyKey();
			const response=await fetch(`/api/v1/organizations/${organizationID}/credit-requests`,{
				method:'POST',credentials:'include',headers:{'Content-Type':'application/json','Idempotency-Key':createRequestKey,...csrfHeaders()},
				body:JSON.stringify({buyer_user_id:buyerUserID,buyer_business_id:buyerBusinessID,buyer_legal_name:buyerLegalName,principal_kobo:amount,goods_description:goods,invoice_reference:'',invoice_document_hash:'',due_date:dueDate,grace_hours:24,collection_at:collectionDate(),schedule_type:'one_time',schedule_count:2,schedule_cadence:'monthly',month_end_policy:'last_day',custom_schedule_items:[]})
			});
			const body=await response.json().catch(()=>({}));
			if(!response.ok){if(response.status<500&&(body.title??body.code)!=='idempotency_in_progress')createRequestKey='';throw new Error(body.detail??'We could not save this sale. Please try again.');}
			createRequestKey='';
			clearDraft();
			await goto(`/app/credit/${body.request.id}?organization=${organizationID}`);
		}catch(cause){error=cause instanceof Error?cause.message:'We could not save this sale. Please try again.'}finally{busy=false}
	}
	onMount(async()=>{
		const params=new URLSearchParams(window.location.search);
		const hasPrefill=params.has('goods')||params.has('amount');
		if(!hasPrefill)restoreDraft();
		const response=await fetch('/api/v1/organizations',{credentials:'include'});
		if(response.status===401){location.assign('/app');return;}
		if(!response.ok){error='We could not open your business account. Please try again.';draftReady=true;return;}
		organizations=(await response.json()).organizations??[];
		organizationID=organizations[0]?.id??'';
		await loadCustomers();
		selectedBuyer=params.get('customer')??'';
		if(params.has('goods'))goods=params.get('goods')??'';
		if(params.has('amount'))principal=params.get('amount')??'';
		if(selectedBuyer)chooseBuyer();
		if(buyerUserID&&goods&&parseNaira(principal)>0)step=dueDate?4:3;
		else if(buyerUserID)step=2;
		draftReady=true;
	});
</script>

<svelte:head><title>Add a sale — Kredit</title></svelte:head>
<main class="shell quick-sale">
	<header class="page-head">
		<div><p class="eyebrow">New sale</p><h1>This takes about a minute.</h1><p class="lede">Four things: who is taking it, what they are taking, how much and when they pay. We keep the rest of the record for you.</p></div>
		<a href="/app/credit/new?advanced=1">Paying in parts, or adding an invoice? Use the full form →</a>
	</header>

	{#if recoveredDraft}<div class="draft-note" role="status"><span><strong>We kept what you typed last time.</strong> We do not save which customer you picked, so choose them again.</span><button type="button" onclick={()=>{goods='';principal='';dueDate='';clearDraft()}}>Start fresh</button></div>{/if}
	<nav class="steps" aria-label="Sale steps">
		{#each ['Customer','Goods & money','Payment day','Check it'] as label,index}
			<button class:active={step===index+1} class:done={step>index+1} type="button" onclick={()=>{if(index+1<step)step=index+1}}><span>{step>index+1?'✓':index+1}</span>{label}</button>
		{/each}
	</nav>

	{#if error}<p class="error" role="alert">{error}</p>{/if}
	<section class="sale-card">
		{#if step===1}
			<div class="stage-copy"><p class="eyebrow">01 — Customer</p><h2>Who is taking the goods?</h2><p>Pick the person taking the goods. They will get this sale and have to agree to it before anything moves.</p></div>
			<div class="fields">
				{#if organizations.length>1}<label>Your business<select bind:value={organizationID} onchange={loadCustomers}>{#each organizations as org}<option value={org.id}>{org.trading_name||org.legal_name}</option>{/each}</select></label>{/if}
				{#if customers.length}<label>Customer<select bind:value={selectedBuyer} onchange={chooseBuyer}><option value="">Choose a customer</option>{#each customers as customer}<option value={customer.buyer_user_id}>{customer.trading_name||customer.legal_name}</option>{/each}</select></label>{:else}<div class="empty-inline"><strong>You have not added a customer yet.</strong><p>Add them first. They get a private link and confirm their own details.</p><a class="primary" href="/app/customers/new">Add a customer</a></div>{/if}
				{#if selectedCustomer}<div class="customer-card"><span>This customer</span><strong>{selectedCustomer.trading_name||selectedCustomer.legal_name}</strong><small>{productLabel(selectedCustomer.state??selectedCustomer.status,'Customer added')}</small>{#if customerWarning}<p class="warning">⚠ {customerWarning}</p>{/if}</div>{/if}
			</div>
		{:else if step===2}
			<div class="stage-copy"><p class="eyebrow">02 — Goods & money</p><h2>What are they taking?</h2><p>Write it clearly enough that six months from now, nobody can argue about what this sale covered.</p></div>
			<div class="fields"><label>What goods are they taking?<textarea bind:value={goods} rows="5" placeholder="For example: 40 cartons of 5L cooking oil"></textarea></label><label>How much must they pay? (₦)<input bind:value={principal} inputmode="decimal" placeholder="1,200,000" />{#if amountWords}<small class="amount-words">{amountWords}</small>{/if}</label></div>
		{:else if step===3}
			<div class="stage-copy"><p class="eyebrow">03 — Payment day</p><h2>When must they pay?</h2><p>Pick the day the two of you agreed. We give them 24 extra hours after that before any bank debit can even be tried.</p></div>
			<div class="fields"><label>Day they must pay<input type="date" bind:value={dueDate} /></label><div class="trust-note"><strong>What Kredit will never do</strong><p>Kredit will not touch your customer's bank before that day plus the extra hours. Even then, it only happens if they gave permission and there is no open problem.</p></div></div>
		{:else}
			<div class="stage-copy"><p class="eyebrow">04 — Check it</p><h2>Read it once more before you save.</h2><p>Nothing has reached your customer yet. Save it first, then open it and send it to them.</p></div>
			<div class="review-card"><dl><div><dt>Customer</dt><dd>{buyerLegalName}</dd></div><div><dt>Goods</dt><dd>{goods}</dd></div><div><dt>Money to pay</dt><dd>{formatKobo(parseNaira(principal))}</dd></div><div><dt>Pay by</dt><dd>{new Date(`${dueDate}T12:00:00`).toLocaleDateString('en-NG',{day:'numeric',month:'long',year:'numeric'})}</dd></div></dl><div class="trust-strip"><span>✓ Your customer reads it before agreeing</span><span>✓ These details stay on record</span><span>✓ Every payment brings the balance down</span></div></div>
		{/if}
	</section>

	<footer class="actions"><div>{#if step>1}<button class="secondary" type="button" onclick={back}>← Back</button>{/if}</div>{#if step<4}<button class="primary" type="button" onclick={next}>Next →</button>{:else}<button class="primary" type="button" onclick={submit} disabled={busy}>{busy?'Saving…':'Save this sale →'}</button>{/if}</footer>
</main>

<style>
	.quick-sale{max-width:72rem;padding-bottom:6rem}.page-head{display:grid;grid-template-columns:1fr auto;gap:3rem;align-items:end;padding:3rem 0 2rem;border-bottom:3px solid #17181b}.page-head h1{max-width:11ch;margin:.5rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(3.2rem,7vw,6rem);font-weight:500;line-height:.9;letter-spacing:-.06em}.page-head>a{max-width:18rem;color:#2738d6;font-size:.82rem;font-weight:750;line-height:1.5}.draft-note{display:flex;justify-content:space-between;gap:1rem;align-items:center;margin:1.25rem 0 0;padding:.8rem 1rem;border-left:4px solid #2738d6;background:#eef0ff;font-size:.8rem;line-height:1.5}.draft-note button{border:0;background:transparent;color:#2738d6;font:inherit;font-weight:800;text-decoration:underline;cursor:pointer;white-space:nowrap}.steps{display:grid;grid-template-columns:repeat(4,1fr);margin:1.5rem 0 2rem;border-top:1px solid #cfc9be;border-left:1px solid #cfc9be}.steps button{display:flex;align-items:center;gap:.65rem;min-height:3.3rem;padding:.7rem;border:0;border-right:1px solid #cfc9be;border-bottom:1px solid #cfc9be;background:#fffdf8;color:#6a6c67;text-align:left;font:inherit;font-size:.78rem;font-weight:750}.steps button span{display:grid;place-items:center;width:1.5rem;height:1.5rem;border:1px solid #aaa69e;border-radius:50%;font-size:.68rem}.steps button.active{background:#17181b;color:#fff}.steps button.active span{border-color:#e85f3d;background:#e85f3d;color:#fff}.steps button.done{color:#2738d6}.steps button.done span{border-color:#2738d6;background:#2738d6;color:#fff}.sale-card{display:grid;grid-template-columns:.75fr 1.25fr;gap:clamp(2rem,6vw,6rem);min-height:27rem;padding:clamp(1.5rem,5vw,3.5rem);border:1px solid #cfc9be;background:#fffdf8;box-shadow:12px 12px 0 #2738d6}.stage-copy h2{max-width:11ch;margin:.5rem 0 1rem;font-family:Georgia,'Times New Roman',serif;font-size:clamp(2.2rem,4vw,3.7rem);font-weight:500;line-height:.95;letter-spacing:-.045em}.stage-copy>p:last-child{max-width:30rem;color:#6a6c67;line-height:1.7}.fields{display:grid;align-content:start;gap:1.2rem}.fields label{display:grid;gap:.45rem;font-weight:750}.fields input,.fields select,.fields textarea{box-sizing:border-box;width:100%;min-height:3rem;padding:.8rem;border:1px solid #aaa69e;border-radius:0;background:#fff;font:inherit}.fields textarea{resize:vertical}.amount-words{color:#2738d6;line-height:1.5}.customer-card,.trust-note,.empty-inline{padding:1rem;border-left:4px solid #2738d6;background:#eef0ff}.customer-card{display:grid;gap:.35rem}.customer-card span,.customer-card small{color:#636863;font-size:.72rem}.warning{margin:.5rem 0 0;padding:.65rem;background:#fff5dd;color:#7a4a00;font-size:.8rem;line-height:1.5}.trust-note{border-left-color:#16794e;background:#ecf8f1}.trust-note p,.empty-inline p{margin:.4rem 0;color:#626762;line-height:1.6}.review-card{display:grid;gap:1.5rem}.review-card dl{margin:0;border-top:1px solid #cfc9be}.review-card dl div{display:grid;grid-template-columns:8rem 1fr;gap:1rem;padding:1rem 0;border-bottom:1px solid #cfc9be}.review-card dt{color:#686a66;font-size:.75rem;font-weight:750}.review-card dd{margin:0;font-weight:700;overflow-wrap:anywhere}.trust-strip{display:grid;gap:.45rem;padding:1rem;color:#fff;background:#17181b;font-size:.78rem}.actions{display:flex;justify-content:space-between;gap:1rem;margin-top:2rem}.actions button{min-height:3rem;padding:.75rem 1.1rem;border-radius:0;font:inherit;font-weight:800}.secondary{border:1px solid #aaa69e;background:#fff}.error{margin:1rem 0;padding:1rem;border-left:4px solid #b42318;background:#fff0ed;color:#8a1c14;font-weight:700}@media(max-width:760px){.page-head,.sale-card{grid-template-columns:1fr}.page-head{gap:1rem}.draft-note{align-items:flex-start;flex-direction:column}.steps{grid-template-columns:1fr 1fr}.steps button{min-height:2.9rem}.sale-card{min-height:auto;box-shadow:7px 7px 0 #2738d6}.actions{position:sticky;bottom:0;z-index:5;margin-inline:-1rem;padding:1rem;background:rgb(245 242 234 / .96);border-top:1px solid #cfc9be}.actions button{min-width:8rem}.review-card dl div{grid-template-columns:1fr;gap:.25rem}}
</style>
