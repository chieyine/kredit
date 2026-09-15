import { boundedFetch, record, RequestError } from '$lib/api/reliable';
export type Due={date:string;amount_kobo:number;paid_kobo:number};
export type Terms={item:string;quantity:number;total_kobo:number;deposit_kobo:number;deposit_date:string;first_date:string;count:number;cadence:string;fulfillment:string;threshold_percent:number;delivery_days:number;stock_reference:string;stock_reserved:boolean;returns_policy:string;bank_name:string;account_name:string;account_number:string;seller_name:string;seller_address:string;terms_version:string;schedule:Due[]};
export type SaleEvent={id:string;action:string;amount_kobo:number;reference:string;related_id:string;note:string;occurred_at:string};
export type Purchase={id:string;organization_id:string;buyer_user_id:string;target_type:string;target:string;terms:Terms;schedule_progress:Due[];agreement_hash:string;state:string;customer_name:string;delivery_address:string;accepted_at:string|null;released_at:string|null;received_at:string|null;case_state:string;version:number;events:SaleEvent[];paid_kobo:number;refunded_kobo:number;reduction_kobo:number;outstanding_kobo:number;refund_due_kobo:number;release_eligible:boolean;delivery_due_at:string|null;role:string};
export const money=(n:number)=>new Intl.NumberFormat('en-NG',{style:'currency',currency:'NGN'}).format(n/100);
export const label=(v:string)=>v.replaceAll('_',' ');
export function kobo(v:string){if(!/^\d+(\.\d{1,2})?$/.test(v.trim()))throw new Error('Enter a valid naira amount with at most two decimal places.');const [whole,fraction='']=v.trim().split('.');const n=Number(whole)*100+Number(fraction.padEnd(2,'0'));if(!Number.isSafeInteger(n))throw new Error('Amount is too large.');return n;}
export function purchase(v:unknown):Purchase{const r=record(v);if(typeof r.id!=='string'||typeof r.version!=='number'||!Array.isArray(r.events)||!Array.isArray(record(r.terms).schedule))throw new Error('The purchase response could not be verified. Refresh before acting.');return r as unknown as Purchase;}
export async function read(url:string){const response=await boundedFetch(url);const data=await response.json();if(!response.ok)throw new RequestError(data.detail||'Could not load this purchase.',response.status);return data;}
// Keep a failed request's key and exact body until its outcome is recovered.
export class Mutation {
 async retry(){if(!this.pending)throw new Error("No unconfirmed action to retry.");return this.send(this.pending.url,JSON.parse(this.pending.body));}
 pending:{url:string;body:string;key:string}|null=null;
 async send(url:string,body:unknown){const encoded=JSON.stringify(body);if(this.pending&&(this.pending.url!==url||this.pending.body!==encoded))throw new Error('Retry the previous action or refresh its record before starting another.');this.pending??={url,body:encoded,key:crypto.randomUUID()};const csrf=document.cookie.split('; ').find(v=>v.startsWith('kredit_csrf='))?.slice('kredit_csrf='.length)||'';
 const response=await boundedFetch(url,{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':decodeURIComponent(csrf),'Idempotency-Key':this.pending.key},body:encoded});const data=await response.json();if(!response.ok){if(response.status<500)this.pending=null;throw new RequestError(data.detail||'The action was not confirmed.',response.status);}this.pending=null;return data;
 }
}
