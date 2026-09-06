<script lang="ts">
 import { parseNaira, verbalizeNaira, formatKobo } from '$lib/money';
 import { getContext, onMount, tick } from 'svelte';
 import { goto } from '$app/navigation';
 import { ACCOUNT_CONTEXT, type AccountContext } from '$lib/account-context';
 import { checkedJSON, csrfHeader, LatestRequest, readResource, record, rows, text } from '$lib/api/reliable';
 import { MutationIntent } from '$lib/api/mutation';
 import { customer, organization, dateLabel, timeLabel, type Customer, type Organization } from '$lib/records';
 import { deleteDraft, readDraft, saveDraft as retainDraft } from '$lib/sale-drafts';
 import { collectionBoundary } from '$lib/financial-copy';
 import DocumentUploader from '$lib/components/DocumentUploader.svelte';
 import { productLabel } from '$lib/product-language';
 const account = getContext<AccountContext>(ACCOUNT_CONTEXT);
 let organizations:Organization[]=$state([]), customers:Customer[]=$state([]), organizationID=$state(''), selectedBuyer=$state(''), busy=$state(false), error=$state(''), success=$state('');
 let loading=$state(true), customerLoading=$state(false), customerError=$state('');
 let principal=$state(''), goods=$state(''), dueDate=$state(''), collectionAt=$state(''), graceHours=$state(24), invoiceReference=$state(''), invoiceFile:File|null=$state(null);
 let scheduleType=$state('one_time'), scheduleCount=$state(2), scheduleCadence=$state('monthly'), monthEndPolicy=$state('last_day'), customSchedule=$state('');
 let customerSearch=$state(''), draftReady=$state(false), draftMessage=$state(''), keepDraft=$state(false);
 let savedItems:{name:string;amount:string}[]=$state([]), reviewed=$state(''), canonicalAt=$state('');
 let errorSummary:HTMLParagraphElement=$state()!;
 let reviewHeading:HTMLHeadingElement=$state()!;
 let creation:MutationIntent|null=null;
 const businessReads=new LatestRequest(), customerReads=new LatestRequest();
 const customerKey=(item:Customer)=>`${item.buyer_user_id}:${item.buyer_business_id}`;
 const selectedCustomer=$derived(customers.find(item=>customerKey(item)===selectedBuyer));
 const buyerUserID=$derived(selectedCustomer?.buyer_user_id??'');
 const buyerBusinessID=$derived(selectedCustomer?.buyer_business_id??'');
 const buyerLegalName=$derived(selectedCustomer?.legal_name??'');
 const verbalPrincipal=$derived(verbalizeNaira(parseNaira(principal)));
 const customerWarning=$derived(selectedCustomer?.overdue?'This customer has an overdue record. Review it before offering more credit.':'');
 const visibleCustomers=$derived(customers.filter(item=>`${item.trading_name} ${item.legal_name}`.toLowerCase().includes(customerSearch.trim().toLowerCase())));
 function chooseBuyer(){reviewed='';}
 function showError(value:string){error=value;void tick().then(()=>errorSummary?.focus());}
 function saveDraft(){
  if(!draftReady||!organizationID)return;
  try{
   if(!keepDraft)deleteDraft(account.userID,organizationID,sessionStorage);
   else if(!retainDraft(account.userID,organizationID,{goods,principal,dueDate},sessionStorage))draftMessage='Your draft could not be kept on this device. Keep this page open until you finish.';
  }catch{draftMessage='Draft storage is unavailable on this device.';}
 }
 function clearDraft(){try{deleteDraft(account.userID,organizationID,sessionStorage);}catch{/* Optional storage. */}principal='';goods='';dueDate='';collectionAt='';invoiceReference='';customSchedule='';invoiceFile=null;reviewed='';draftMessage='Draft cleared.';}
 function useSaved(item:{name:string;amount:string}){goods=item.name;principal=item.amount;reviewed='';saveDraft();}
 function practise(){
  goods='10 cartons of cooking oil';principal='100,000';
  const next=new Date(Date.now()+30*86400000);dueDate=new Intl.DateTimeFormat('en-CA',{timeZone:'Africa/Lagos',year:'numeric',month:'2-digit',day:'2-digit'}).format(next);
  collectionAt='';graceHours=24;reviewed='';draftMessage='Sample goods and amount filled in. Check all details before saving a real draft.';saveDraft();
 }
 async function loadCustomers(){
  const scope=organizationID,request=customerReads.begin();customers=[];selectedBuyer='';customerSearch='';customerLoading=true;customerError='';reviewed='';creation=null;invoiceFile=null;
  if(!scope){customerLoading=false;return;}
  const result=await readResource(scope,`/api/v1/organizations/${encodeURIComponent(scope)}/customers`,rows('customers',customer),request.signal,'your customers');
  if(!request.current()||scope!==organizationID)return;
  customerLoading=false;if(result.state==='ready')customers=result.data;else if(result.state==='error')customerError=result.message;
 }
 async function changeBusiness(){
  draftReady=false;keepDraft=false;principal='';goods='';dueDate='';collectionAt='';invoiceReference='';customSchedule='';invoiceFile=null;reviewed='';draftMessage='';
  await loadCustomers();draftReady=true;
 }
 async function load(){
  loading=true;error='';const request=businessReads.begin();
  const result=await readResource(account.userID,'/api/v1/organizations',rows('organizations',organization),request.signal,'your businesses');
  if(!request.current())return;
  loading=false;if(result.state!=='ready'){if(result.state==='error')showError(result.message);return;}
  organizations=result.data;
  const params=new URLSearchParams(location.search);
  organizationID=organizations.find(org=>org.id===params.get('organization'))?.id??organizations[0]?.id??'';
  if(!organizationID)return;
  try{for(let i=localStorage.length-1;i>=0;i--){const key=localStorage.key(i);if(key?.startsWith('kredit:sale-draft:')||key==='kredit:saved-sale-items')localStorage.removeItem(key);}}catch{/* Storage is optional. */}
  await loadCustomers();if(!request.current())return;
  if(params.has('goods'))goods=(params.get('goods')??'').slice(0,5000);
  if(params.has('amount'))principal=(params.get('amount')??'').slice(0,40);
  const matches=customers.filter(item=>item.buyer_user_id===params.get('customer'));
  if(matches.length===1)selectedBuyer=customerKey(matches[0]);
  try{
   const draft=readDraft(account.userID,organizationID,sessionStorage);
   if(draft&&!params.has('goods')&&!params.has('amount')){goods=draft.goods;principal=draft.principal;dueDate=draft.dueDate;keepDraft=true;draftMessage='Goods, amount and date restored for this business. Choose the customer and check invoice and instalment details again.';}
  }catch{/* No draft is required. */}
  draftReady=true;
 }
 onMount(()=>{void load();return()=>{businessReads.cancel();customerReads.cancel();};});
 function customItems(){
  if(scheduleType!=='custom')return [];
  const lines=customSchedule.split('\n').map(line=>line.trim()).filter(Boolean);
  if(!lines.length||lines.length>60)throw new Error('Enter between 1 and 60 instalments.');
  let previous='';const result=lines.map((line,index)=>{
   const match=/^(.+),\s*(\d{4}-\d{2}-\d{2})$/.exec(line);
   if(!match)throw new Error(`Check instalment ${index+1}: enter an amount, a comma, then YYYY-MM-DD.`);
   const amount=parseNaira(match[1].trim()), date=match[2];
   if(amount<=0||dateLabel(date)==='Date unavailable'||(previous&&date<=previous))throw new Error(`Check instalment ${index+1}. Amounts must be positive and dates must be valid and in order.`);
   previous=date;return{amount_kobo:amount,due_date:date};
  });
  if(result.reduce((sum,item)=>sum+BigInt(item.amount_kobo),0n)!==BigInt(parseNaira(principal)))throw new Error('The instalments must add up to the sale amount.');
  if(result[0].due_date!==dueDate)throw new Error('The first instalment must use the agreed first payment date.');
  return result;
 }
 function capture(){
  const amount=parseNaira(principal);
  if(!organizationID||!selectedCustomer||customerLoading||customerError||amount<=0||!goods.trim()||dateLabel(dueDate)==='Date unavailable')throw new Error('Check the customer, goods, amount and first payment date.');
  if(!Number.isInteger(graceHours)||graceHours<0||graceHours>720)throw new Error('Extra hours must be a whole number from 0 to 720.');
  if(scheduleType==='equal'&&(!Number.isInteger(scheduleCount)||scheduleCount<2||scheduleCount>60))throw new Error('Choose between 2 and 60 instalments.');
  return{buyer_user_id:buyerUserID,buyer_business_id:buyerBusinessID,buyer_legal_name:buyerLegalName,buyer_trading_name:selectedCustomer.trading_name,principal_kobo:amount,goods_description:goods.trim(),invoice_reference:invoiceReference.trim(),due_date:dueDate,grace_hours:graceHours,schedule_type:scheduleType,schedule_count:scheduleType==='one_time'?1:Number(scheduleCount),schedule_cadence:scheduleType==='equal'?scheduleCadence:'custom',month_end_policy:monthEndPolicy,custom_schedule_items:customItems(),collection_local:collectionAt||undefined};
 }
 async function uploadInvoice(){
  if(!invoiceFile)return '';
  const file=invoiceFile;
  if(!['application/pdf','image/jpeg','image/png'].includes(file.type)||file.size<=0||file.size>2*1024*1024)throw new Error('Choose a PDF, JPG or PNG invoice up to 2 MB.');
  const bytes=await file.arrayBuffer();
  const hash=Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256',bytes)),b=>b.toString(16).padStart(2,'0')).join('');
  const content=await new Promise<string>((resolve,reject)=>{const reader=new FileReader();reader.onerror=()=>reject(new Error('We could not read the invoice.'));reader.onload=()=>resolve(String(reader.result).split(',')[1]??'');reader.readAsDataURL(file);});
  const operation=new MutationIntent(`${account.userID}:${organizationID}:invoice:${hash}`,`/api/v1/organizations/${encodeURIComponent(organizationID)}/documents`);
  return operation.run({purpose:'credit_invoice',file_name:file.name,content_type:file.type,retention_class:'financial_agreement',content_base64:content},value=>{
   const saved=text(record(record(value).document).sha256);
   if(saved!==hash)throw new Error('The stored invoice did not match the selected file.');
   return saved;
  });
 }
 async function submit(event:SubmitEvent){
  event.preventDefault();if(busy)return;error='';success='';busy=true;
  try{
   const input=capture();const signature=JSON.stringify({...input,invoice:invoiceFile?{name:invoiceFile.name,size:invoiceFile.size,lastModified:invoiceFile.lastModified}:null});
   if(signature!==reviewed){
    const timing=await checkedJSON(`/api/v1/organizations/${encodeURIComponent(organizationID)}/credit-terms/preview`,value=>{
     const row=record(value),instant=text(row.collection_at);
     if(row.due_date!==dueDate||row.grace_hours!==graceHours||row.timezone!=='Africa/Lagos'||row.timing_mode!==(collectionAt?'lagos_explicit':'lagos_end_of_day')||!Number.isFinite(Date.parse(instant)))throw new Error('The payment timing could not be verified.');
     return instant;
    },{method:'POST',headers:{'Content-Type':'application/json',...csrfHeader()},body:JSON.stringify({due_date:dueDate,grace_hours:graceHours,collection_local:collectionAt||undefined})});
    canonicalAt=timing;reviewed=signature;success='Terms checked. Review the exact amount and Nigerian collection time below before saving.';await tick();reviewHeading?.focus();return;
   }
   const invoiceDocumentHash=await uploadInvoice();
   creation??=new MutationIntent(`${account.userID}:${organizationID}`,`/api/v1/organizations/${encodeURIComponent(organizationID)}/credit-requests`);
   const id=await creation.run({...input,invoice_document_hash:invoiceDocumentHash,collection_at:canonicalAt,timing_mode:collectionAt?'lagos_explicit':'lagos_end_of_day'},value=>text(record(record(value).request).id));
   draftReady=false;try{deleteDraft(account.userID,organizationID,sessionStorage);}catch{/* Server result is authoritative. */}
   await goto(`/app/credit/${encodeURIComponent(id)}?organization=${encodeURIComponent(organizationID)}`);
  }catch(cause){showError(cause instanceof Error?cause.message:'We could not confirm the result. Check the sale history before starting again.');}finally{busy=false;}
 }
</script>

<svelte:head><title>Add a sale — Kredit</title></svelte:head>
<main class="shell workspace form-page"><div class="page-heading"><div><p class="eyebrow">Add a sale</p><h1>Add a credit sale</h1><p class="lede">Add invoice details or instalments. Your customer reviews the exact terms before accepting.</p></div><span class="draft-badge">Not sent yet</span></div>
{#if draftMessage}<p class="draft-message" role="status">{draftMessage} <button type="button" onclick={clearDraft}>Clear</button></p>{/if}{#if error}<p bind:this={errorSummary} class="error" role="alert" tabindex="-1"><strong>Please check:</strong> {error}</p>{/if}{#if success}<p class="success" role="status">{success}</p>{/if}
{#if loading}<p role="status">Checking your businesses…</p>{:else if !organizations.length}<p>Add or reopen your business from <a href="/app/overview">Home</a> before creating a sale.</p>{/if}<div class="practice"><div><strong>First time here?</strong><p>Let us fill it with sample details so you can see how it works. Nothing reaches anybody until you save and send it yourself.</p></div><button type="button" onclick={practise}>Show me with samples</button></div>
<form onsubmit={submit} oninput={() => { reviewed=""; saveDraft(); }} onchange={saveDraft}><fieldset disabled={busy} style="border:0;padding:0;margin:0;min-width:0"><section class="form-card"><b>01</b><div><h2>Who is taking the goods?</h2><div class="form-grid"><label>Your business<select bind:value={organizationID} onchange={changeBusiness} required>{#each organizations as org}<option value={org.id}>{org.trading_name||org.legal_name}</option>{/each}</select></label>{#if customerLoading}<p role="status">Checking your customers…</p>{:else if customerError}<div role="alert"><strong>Customer list unavailable</strong><p>{customerError}</p><button type="button" onclick={loadCustomers}>Try again</button></div>{:else if customers.length}<label>Search your customers<input bind:value={customerSearch} type="search" placeholder="Type their name" autocomplete="off" /></label><label>Customer<select bind:value={selectedBuyer} onchange={chooseBuyer} required><option value="">Choose a customer</option>{#each visibleCustomers as customer}<option value={customerKey(customer)}>{customer.trading_name||customer.legal_name}</option>{/each}</select></label><label>Customer name<input value={buyerLegalName} readonly /></label><label>Customer verification<input value={productLabel(selectedCustomer?.state, 'Choose a customer')} readonly /></label>{#if customerWarning}<div class="network-overdue-warning" role="alert"><strong>Overdue record:</strong> {customerWarning}</div>{/if}{:else}<div class="no-customers"><strong>You have not added a customer yet.</strong><p>Add them first. They get a private link and confirm their own details.</p><a class="primary" href="/app/customers/new">Add a customer</a></div>{/if}</div></div></section><section class="form-card"><b>02</b><div><h2>What are they taking, and for how much?</h2>{#if savedItems.length}<div class="saved-items"><strong>Goods you sold before</strong>{#each savedItems.slice(0,4) as item}<button type="button" onclick={()=>useSaved(item)}>{item.name} · ₦{item.amount}</button>{/each}</div>{/if}<div class="form-grid"><label>Sale amount (₦)<input bind:value={principal} inputmode="decimal" placeholder="1,200,000" required />{#if verbalPrincipal}<small class="verbal-naira" aria-live="polite">{verbalPrincipal}</small>{/if}</label><label>What goods are they taking?<textarea bind:value={goods} rows="4" required></textarea></label><label>Invoice number <small>if you have one</small><input bind:value={invoiceReference} maxlength="120" /></label><div><DocumentUploader accept="application/pdf,image/jpeg,image/png" disabled={busy} onselect={(file)=>{invoiceFile=file;reviewed="";}} />{#if invoiceFile}<small>{invoiceFile.name}</small>{/if}</div><label>First payment date<input type="date" bind:value={dueDate} required /></label><label>Optional later collection time (Nigerian time)<input type="datetime-local" bind:value={collectionAt} /><small>Leave blank to use the end of the payment day plus the extra hours. This time is always interpreted in Africa/Lagos, wherever your device is.</small></label><label>Extra hours you are giving them<input type="number" min="0" max="720" bind:value={graceHours} /></label><label>How will they pay?<select bind:value={scheduleType}><option value="one_time">All at once</option><option value="equal">The same amount, many times</option><option value="custom">Different amounts</option></select></label>{#if scheduleType==='equal'}<label>How many payments?<input type="number" min="2" max="60" bind:value={scheduleCount} required /></label><label>How often?<select bind:value={scheduleCadence}><option value="weekly">Every week</option><option value="fortnightly">Every two weeks</option><option value="monthly">Every month</option></select></label>{#if scheduleCadence==='monthly'}<label>If that day does not exist in a month<select bind:value={monthEndPolicy}><option value="last_day">Use the last day of that month</option><option value="cap">Use the nearest day</option></select></label>{/if}{:else if scheduleType==='custom'}<label class="wide">Write out each payment<textarea bind:value={customSchedule} rows="5" placeholder={'600000, 2026-09-30\n600000, 2026-10-30'} required></textarea><small>One payment on each line: the amount, then the date. They must add up to the total money owed.</small></label>{/if}</div></div></section><section class="form-card review"><b>03</b><div><h2 bind:this={reviewHeading} tabindex="-1">Review this sale</h2><p><strong>{buyerLegalName||'Choose a customer'}</strong> will see that they took <strong>{goods||'the goods'}</strong> and must pay <strong>{formatKobo(parseNaira(principal))}</strong>{dueDate?` by ${dateLabel(dueDate)}`:''}.</p>{#if reviewed}<p><strong>Bank collection may be considered from:</strong> {timeLabel(canonicalAt)}</p>{:else}<p>Check the terms to see the server-verified collection time.</p>{/if}<p>{collectionBoundary}</p><p>Saving creates a draft, not a payment or bank permission.</p></div></section><div class="form-actions"><a href="/app/overview">Go back</a><button class="primary" disabled={busy||!buyerUserID}>{busy?'Checking…':reviewed?'Save draft sale':'Check terms'}</button></div></fieldset></form><label class="draft-choice"><input type="checkbox" bind:checked={keepDraft} onchange={saveDraft} /> Keep goods, amount and date on this device for up to 12 hours</label><p>Leave this off on a shared device. Customer, invoice and instalment details are not stored in a browser draft.</p></main>
<style>.draft-choice{display:flex;align-items:center;gap:.7rem;margin-top:1.5rem;font-size:.9rem}.draft-choice input{min-height:1.2rem}.form-grid>label{min-width:0}.form-card{min-width:0}.form-card>div{min-width:0}.verbal-naira{color:#2738d6;font-weight:600;font-size:.85rem;margin-top:.25rem}.network-overdue-warning{grid-column:span 2;padding:.85rem 1rem;background:#fff8e6;border-left:4px solid #b7791f;color:#744210;font-size:.9rem;font-weight:600;margin-top:.75rem}.form-page{max-width:68rem}.page-heading{display:flex;justify-content:space-between;gap:2rem;padding-bottom:2rem;border-bottom:3px solid #17181b}.page-heading h1{font-family:Georgia,'Times New Roman',serif;font-size:clamp(3rem,6vw,5.2rem);font-weight:500;line-height:.92;letter-spacing:-.055em;margin:.5rem 0}.draft-badge{height:max-content;padding:.55rem .8rem;color:#17181b;background:#e85f3d;font-weight:750}.draft-message,.practice{display:flex;align-items:center;justify-content:space-between;gap:1rem;padding:1rem;background:#eef1ff;border-left:4px solid #2738d6}.draft-message button,.practice button{padding:.6rem .8rem;border:1px solid #2738d6;background:#fff;font-weight:750}.practice{margin-top:1rem;background:#fffdf8;border:1px solid var(--color-border)}.practice p{margin:.25rem 0;color:var(--color-muted)}.form-card{display:grid;grid-template-columns:3rem 1fr;gap:1.5rem;margin:1.5rem 0;padding:clamp(1.25rem,4vw,2.3rem);border:1px solid var(--color-border);background:#fffdf8;box-shadow:8px 8px 0 #ded8cc}.form-card>b{display:grid;place-items:center;width:2.6rem;height:2.6rem;background:#2738d6;color:#fff}.form-card h2{margin:0;font-family:Georgia,'Times New Roman',serif;font-size:1.8rem;font-weight:500}.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1.15rem;margin-top:1.5rem}.form-grid label{display:grid;gap:.45rem;font-weight:700}.form-grid input,.form-grid select,.form-grid textarea{box-sizing:border-box;width:100%;padding:.8rem;border:1px solid #aaa69e;border-radius:0;background:#fff;font:inherit}.form-grid input:focus,.form-grid select:focus,.form-grid textarea:focus{border-color:#2738d6}.form-grid input[readonly]{background:var(--color-surface-muted)}.form-grid textarea{min-width:0}.no-customers,.wide{grid-column:span 2}.no-customers{padding:1.4rem;color:#fff;background:#17181b}.no-customers p{color:#c5c4bf}.saved-items{display:flex;flex-wrap:wrap;gap:.5rem;margin:1rem 0;padding:.8rem;background:var(--color-surface-muted)}.saved-items strong{flex-basis:100%}.saved-items button{padding:.55rem .7rem;border:1px solid var(--color-border);background:#fff;font:inherit}.review p{font-size:1.05rem;line-height:1.6}.form-actions{display:flex;justify-content:flex-end;align-items:center;gap:1rem;padding-top:1rem}.success{padding:.8rem;background:#e6f7ed;color:var(--color-positive)}@media(max-width:720px){.page-heading{display:block}.draft-badge{display:inline-flex;margin-top:1rem}.practice{align-items:start;flex-direction:column}.form-card{grid-template-columns:1fr;padding:1.25rem;box-shadow:5px 5px 0 #ded8cc}.form-grid{grid-template-columns:1fr}.form-grid textarea,.no-customers,.wide{grid-column:auto}}</style>
