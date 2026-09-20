<script lang="ts">
 import {onMount} from 'svelte';
 import {checkedJSON,record,text,publicError} from '$lib/api/reliable';
 import {formatKobo} from '$lib/money';
 import {MutationIntent} from '$lib/api/mutation';
 import {actualPaymentTime,positiveNaira} from '$lib/financial-input';
 import VerifyIdentity from './VerifyIdentity.svelte';
 let {organizationID='',admin=false}:{organizationID?:string;admin?:boolean}=$props();
 type Payout={attempt_id:string;organization_id:string;payment_id:string;payment_state:string;collected_kobo:number;paid_kobo:number;returned_kobo:number;outstanding_kobo:number;bank:string;account:string;last4:string;method:string};
 let items=$state<Payout[]>([]),loading=$state(true),busy=$state(false),error=$state(''),message=$state(''),selected=$state<Payout|null>(null);
 let intent:MutationIntent|null=null;
 let unconfirmed=$state(false),recoveryReference=$state('');
 let amount=$state(''),reference=$state(''),at=$state(''),evidence=$state(''),direction=$state('paid');
 function money(v:unknown):number{if(!Number.isSafeInteger(v)||Number(v)<0)throw new Error('Invalid settlement amount');return Number(v)}
 async function load(){loading=true;error='';try{items=await checkedJSON(admin?'/api/v1/ops/seller-settlements':`/api/v1/organizations/${encodeURIComponent(organizationID)}/seller-settlements`,value=>{const body=record(value);if(!Array.isArray(body.settlements))throw new Error('Incomplete settlements');return body.settlements.map(value=>{const p=record(value),route=record(p.route);return{attempt_id:text(p.attempt_id),organization_id:text(p.organization_id),payment_id:text(p.payment_id),payment_state:text(p.payment_state),collected_kobo:money(p.collected_kobo),paid_kobo:money(p.paid_kobo),returned_kobo:money(p.returned_kobo),outstanding_kobo:money(p.outstanding_kobo),bank:text(route.bank_name),account:text(route.account_name),last4:text(route.account_last4),method:text(route.method)}})})}catch(cause){error=publicError(cause,'Settlement records could not be loaded.')}finally{loading=false}}
 function choose(p:Payout){if(busy||unconfirmed)return;selected=p;intent=new MutationIntent('seller-settlement',`/api/v1/ops/seller-settlements/${encodeURIComponent(p.organization_id)}/${encodeURIComponent(p.attempt_id)}`);unconfirmed=intent.unresolved;recoveryReference=intent.recoveryReference;amount='';reference='';at='';evidence='';direction='paid';message=''}
 async function save(){
  if(!selected||!intent||busy)return;
  busy=true;error='';
  try{
   const input={reference:reference.trim(),amount_kobo:positiveNaira(amount),direction,occurred_at:actualPaymentTime(at),evidence:evidence.trim()};
   await intent.run(input,value=>{
    const saved=record(record(value).settlement);
    if(saved.attempt_id!==selected?.attempt_id||saved.organization_id!==selected?.organization_id||!Array.isArray(saved.receipts))throw new Error('Bank evidence confirmation was incomplete.');
    const matched=saved.receipts.map(record).find(receipt=>receipt.reference===input.reference&&receipt.amount_kobo===input.amount_kobo&&receipt.direction===input.direction&&Date.parse(text(receipt.occurred_at))===Date.parse(input.occurred_at));
    if(!matched||!text(matched.id))throw new Error('Bank evidence confirmation was incomplete.');
    return saved;
   });
   unconfirmed=false;recoveryReference='';message='Bank evidence recorded. No new transfer was sent.';selected=null;await load();
  }catch(cause){unconfirmed=intent.unresolved;recoveryReference=intent.recoveryReference;error=cause instanceof Error?cause.message:'Settlement recording was not confirmed.';}
  finally{busy=false}
 }

 onMount(load);
</script>
<section><h2>Seller bank settlements</h2><p>A successful customer debit and a completed seller payout are separate records. The saved bank below belongs to the original collection, even if bank settings have since changed.</p>
{#if message}<p role="status">{message}</p>{/if}{#if error}<p role="alert">{error}</p>{/if}<button disabled={busy||loading} onclick={load}>Refresh settlements</button>
{#if loading}<p>Loading settlements…</p>{:else if !items.length&&!error}<p>No collections with saved settlement destinations yet.</p>{/if}
{#each items as item(item.attempt_id)}<article><h3>{item.account}</h3><p>{item.bank} · ending {item.last4}</p><p>Collected: {formatKobo(item.collected_kobo)} · Paid to seller: {formatKobo(item.paid_kobo)} · Returned: {formatKobo(item.returned_kobo)} · Awaiting settlement: {formatKobo(item.outstanding_kobo)}</p><p>{item.payment_state==='reversed'?'Customer payment reversed — review any bank payout already made.':item.payment_id?'Customer payment recognized.':'Customer payment is not yet recognized.'}</p><p>{item.method==='provider_split'?'Provider split requested; bank receipt still requires confirmation.':'A transfer to the saved seller bank must be completed and reconciled.'}</p>{#if admin&&item.payment_id}<button disabled={busy||unconfirmed} onclick={()=>choose(item)}>Record completed bank transfer</button>{/if}</article>{/each}
{#if selected}<VerifyIdentity/><form onsubmit={(event)=>{event.preventDefault();void save()}}><fieldset disabled={busy}><h3>Record evidence for {selected.account}, ending {selected.last4}</h3><p>This records a transfer that has already happened. It does not send money or reduce the customer’s debt again.</p><label>Direction<select bind:value={direction}><option value="paid">Paid to the seller</option><option value="returned">Returned from the seller’s bank</option></select></label><label>Amount in naira<input bind:value={amount} inputmode="decimal" required/></label><label>Unique bank transfer reference<input bind:value={reference} required maxlength="160"/></label><label>Transfer time (Lagos)<input type="datetime-local" bind:value={at} required/></label><label>Bank evidence and destination checked<textarea bind:value={evidence} required minlength="20" maxlength="2000"></textarea></label><button disabled={busy}>{busy?'Saving…':unconfirmed?'Retry the same bank evidence':'Record verified bank evidence'}</button><button type="button" disabled={busy||unconfirmed} onclick={()=>selected=null}>Cancel</button></fieldset>{#if unconfirmed}<p role="status">The result is unconfirmed. Keep the original details for a retry, or check the settlement history. Do not send another bank transfer. <a href="/contact">Contact support</a> with this reference: <code>{recoveryReference||'Reference unavailable; provide the settlement and approximate time'}</code>.</p>{/if}</form>{/if}</section>
<style>section{margin-block:1.5rem}article,form{padding:1rem;border:1px solid var(--border,var(--color-border));border-radius:.7rem;margin-block:1rem}form,fieldset,label{display:grid;gap:.6rem}input,button,select,textarea{font:inherit;padding:.65rem}p{line-height:1.5}</style>
