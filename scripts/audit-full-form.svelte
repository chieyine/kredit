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
