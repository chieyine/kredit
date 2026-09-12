import { formatKobo } from './money';
import type { SaleView, WorkRow } from './records';
export interface AttentionItem { id: string; rank: number; title: string; detail: string; href: string; action: string; }
/** This queue is never sliced before counting; disputes cannot disappear behind drafts. */
export function attentionItems(organizationID: string, sales: SaleView[], claims: WorkRow[], overdue: WorkRow[], disputes: WorkRow[], due: WorkRow[] = []): AttentionItem[] {
  const items: AttentionItem[] = [];
  const query = `?organization=${encodeURIComponent(organizationID)}`;
  const saleHref = (id: string) => `/app/credit/${encodeURIComponent(id)}${query}`;
  const customerName = (item: WorkRow) => item.buyer_legal_name || sales.find(sale => (item.credit_request_id && sale.request.id === item.credit_request_id) || (item.obligation_id && sale.obligation?.id === item.obligation_id) || (item.buyer_user_id && sale.request.buyer_user_id === item.buyer_user_id))?.request.buyer_legal_name || 'A customer';
  for (const item of disputes.filter(item => ['OPEN', 'UNDER_REVIEW', 'PARTIALLY_RESOLVED', 'PENDING'].includes(item.state.toUpperCase()))) items.push({ id: `dispute-${item.id}`, rank: 0, title: 'A sale needs review', detail: item.reason || 'Check the reported problem and the amount on hold.', href: `/app/disputes/${encodeURIComponent(item.id)}${query}`, action: 'Review problem' });
  for (const item of claims.filter(item => item.state.toUpperCase() === 'PENDING')) items.push({ id: `claim-${item.id}`, rank: 1, title: `${customerName(item)} reported a payment`, detail: `${formatKobo(item.amount_kobo)} · Check your bank account before confirming.`, href: item.credit_request_id ? saleHref(item.credit_request_id) : `/app/payments${query}`, action: 'Check payment' });
  for (const item of overdue) items.push({ id: `late-${item.id}`, rank: 2, title: `${customerName(item)} has an overdue payment`, detail: `${formatKobo(item.amount_kobo)} remains unpaid.`, href: saleHref(item.credit_request_id || item.id), action: 'Open sale' });
  for(const item of due){
    if(overdue.some(old=>(old.credit_request_id||old.id)===(item.credit_request_id||item.id)))continue;
    items.push({id:`due-${item.id}`,rank:3,title:`Payment ${item.state==='DUE'?'due':'due within seven days'} from ${customerName(item)}`,detail:`${formatKobo(item.amount_kobo)} on the current schedule. Check reported transfers before taking action.`,href:saleHref(item.credit_request_id||item.id),action:'Review payment dates'});
  }
  for (const { request } of sales) {
    if(['GOODS_RELEASED','RECEIPT_CONFIRMATION_PENDING'].includes(request.state))items.push({id:`receipt-${request.id}`,rank:3,title:`Waiting for ${request.buyer_legal_name} to confirm delivery`,detail:'Check the delivery record and any reported problem. Dispatch is not proof of receipt.',href:saleHref(request.id),action:'Review receipt'});
    if (request.state === 'READY_TO_RELEASE') items.push({ id: `release-${request.id}`, rank: 3, title: `Arrange goods for ${request.buyer_legal_name}`, detail: 'The agreement is accepted and bank permission is ready.', href: saleHref(request.id), action: 'Review delivery' });
    if (request.state === 'BUYER_ACCEPTED') items.push({ id: `bank-${request.id}`, rank: 4, title: `${request.buyer_legal_name} accepted the sale`, detail: 'Bank permission still needs confirmation. Do not release the goods yet.', href: saleHref(request.id), action: 'Check status' });
    if (['SENT', 'BUYER_REVIEWING'].includes(request.state)) items.push({ id: `waiting-${request.id}`, rank: 5, title: `Waiting for ${request.buyer_legal_name}`, detail: 'The customer has not accepted this sale yet.', href: saleHref(request.id), action: 'Open sale' });
    if (request.state === 'DRAFT') items.push({ id: `draft-${request.id}`, rank: 6, title: 'Draft sale', detail: `Not yet sent to ${request.buyer_legal_name}.`, href: saleHref(request.id), action: 'Continue draft' });
  }
  return items.sort((a, b) => a.rank - b.rank || a.id.localeCompare(b.id));
}
