import type { SaleView } from './records';
export type NextStep = { actor: string; title: string; detail: string; action: string };
// Guidance is based on recorded evidence. It never authorizes a financial action.
export function saleNextStep(view: SaleView): NextStep {
 const state=view.request.state;
 if(view.obligation && String(view.obligation.outstanding_kobo)==='0') return {actor:'No action needed',title:'The sale balance is paid',detail:'Keep the agreement and payment records. Seller settlement is recorded separately.',action:'Review payments'};
 const steps:Record<string,NextStep>={
  DRAFT:{actor:'Seller',title:'Check and send the sale',detail:'Confirm the goods, amount, fees and payment dates. Save your changes before sending.',action:'Review draft'},
  SENT:{actor:'Customer',title:'Read the agreement and decide',detail:'The customer needs to accept or decline the exact sale sent by the seller.',action:'Review agreement'},
  BUYER_REVIEWING:{actor:'Customer',title:'Read the agreement and decide',detail:'Acceptance records the sale agreement. Bank-debit permission is a separate step.',action:'Review agreement'},
  VERIFICATION_PENDING:{actor:'Customer',title:'Complete the account checks',detail:'Identity, business and authority checks must be current before acceptance.',action:'Review account checks'},
  BUYER_ACCEPTED:{actor:'Customer / bank',title:'Finish bank-debit permission',detail:'The sale is accepted. Complete the secure bank step, then check its status before goods are released.',action:'Review bank permission'},
  READY_TO_RELEASE:{actor:'Seller',title:'Arrange and record delivery',detail:'The customer has accepted and bank permission is ready. Record when the goods actually leave.',action:'Review delivery'},
  GOODS_RELEASED:{actor:'Customer',title:'Confirm the goods arrived',detail:'Confirm receipt or report a delivery problem. The dispatch record alone does not prove receipt.',action:'Review receipt'},
  RECEIPT_CONFIRMATION_PENDING:{actor:'Customer',title:'Confirm the goods arrived',detail:'Confirm receipt or report a problem. Any automatic recognition requires its own recorded delivery and waiting-period evidence.',action:'Review receipt'},
  ACTIVE:{actor:'Customer / seller',title:'Follow the current payment schedule',detail:'Pay the amount due. The seller confirms reported transfers; a bank request stays pending until its outcome is known.',action:'Review payments'},
  CANCELLED:{actor:'No action needed',title:'This sale was cancelled',detail:'Its history remains available. A new sale requires a new agreement.',action:'Review history'},
  DECLINED:{actor:'Seller',title:'The customer declined this sale',detail:'Discuss any correction with the customer before creating a new offer.',action:'Review history'}
 };
 return steps[state]??{actor:'Support',title:'Check the current sale record',detail:'This status needs review before taking another action.',action:'Review history'};
}
