import { exactKobo, type KoboValue } from './money';
import { record, text, RequestError } from './api/reliable';
import { validFeeTerms, type FeeTerms } from './fee-terms';

const optionalText = (value: unknown) => typeof value === 'string' ? value : '';
export function kobo(value: unknown): KoboValue {
  if ((typeof value !== 'number' && typeof value !== 'string' && typeof value !== 'bigint') || exactKobo(value) === null || exactKobo(value)! < 0n) throw new RequestError('The amount could not be verified.', 0, 'invalid_response');
  return value;
}
export interface Organization { id: string; legal_name: string; trading_name: string; }
export function organization(value: unknown): Organization { const row = record(value); return { id: text(row.id), legal_name: text(row.legal_name), trading_name: optionalText(row.trading_name) }; }
export interface Customer { buyer_user_id: string; buyer_business_id: string; legal_name: string; trading_name: string; state: string; overdue: boolean; }
export function customer(value: unknown): Customer {
  const row = record(value);
  return { buyer_user_id: text(row.buyer_user_id), buyer_business_id: text(row.buyer_business_id), legal_name: text(row.legal_name), trading_name: optionalText(row.trading_name), state: optionalText(row.state ?? row.status), overdue: row.has_overdue_obligations === true || (typeof row.overdue_count === 'number' && row.overdue_count > 0) || row.has_network_overdue === true };
}
export interface SaleRequest {
  system_acceptance_id?: string; id: string; state: string; supplier_legal_name: string; buyer_legal_name: string; buyer_user_id: string; buyer_business_id: string;
  principal_kobo: KoboValue; goods_description: string; due_date: string; collection_at: string; grace_hours: number;
  schedule_type: string; schedule_count: number; schedule_cadence: string; fee_terms: FeeTerms | null;
  custom_schedule_items: { amount_kobo: KoboValue; due_date: string }[];
}
export interface SaleView {
  request: SaleRequest;
  agreement: { id: string; document_hash: string; terms_version: string; privacy_version: string } | null;
  mandate: { id: string; provider_id: string; provider: string; status: string; authorization_url: string } | null;
  obligation: { id: string; outstanding_kobo: KoboValue } | null;
}
export function saleView(value: unknown): SaleView {
  const view = record(value); const row = record(view.request);
  const agreement = view.agreement ? record(view.agreement) : null;
  const mandate = view.mandate ? record(view.mandate) : null;
  const obligation = view.obligation ? record(view.obligation) : null;
  return {
    request: {
      system_acceptance_id: optionalText(row.system_acceptance_id), id: text(row.id), state: text(row.state), supplier_legal_name: optionalText(row.supplier_legal_name), buyer_legal_name: text(row.buyer_legal_name), buyer_user_id: optionalText(row.buyer_user_id), buyer_business_id: optionalText(row.buyer_business_id),
      principal_kobo: kobo(row.principal_kobo), goods_description: optionalText(row.goods_description), due_date: text(row.due_date), collection_at: optionalText(row.collection_at), grace_hours: typeof row.grace_hours === 'number' ? row.grace_hours : 0,
      schedule_type: optionalText(row.schedule_type), schedule_count: typeof row.schedule_count === 'number' ? row.schedule_count : 1, schedule_cadence: optionalText(row.schedule_cadence), fee_terms: validFeeTerms(row.fee_terms) ? row.fee_terms : null,
      custom_schedule_items: Array.isArray(row.custom_schedule_items) ? row.custom_schedule_items.map(value => { const item = record(value); return { amount_kobo: kobo(item.amount_kobo), due_date: text(item.due_date) }; }) : []
    },
    agreement: agreement ? { id: text(agreement.id), document_hash: text(agreement.document_hash), terms_version: optionalText(agreement.terms_version), privacy_version: optionalText(agreement.privacy_version) } : null,
    mandate: mandate ? { id: optionalText(mandate.id), provider_id: text(mandate.provider_id), provider: optionalText(mandate.provider), status: text(mandate.status), authorization_url: optionalText(mandate.authorization_url) } : null,
    obligation: obligation ? { id: text(obligation.id), outstanding_kobo: kobo(obligation.outstanding_kobo) } : null
  };
}
export interface PaymentRow { id: string; amount_kobo: KoboValue; state: string; source_type: string; paid_at: string; }
export function paymentRow(value: unknown): PaymentRow { const row = record(value); return { id: text(row.id), amount_kobo: kobo(row.amount_kobo), state: text(row.state), source_type: optionalText(row.source_type), paid_at: optionalText(row.paid_at) }; }
export interface WorkRow { id: string; state: string; buyer_legal_name: string; buyer_user_id: string; obligation_id: string; credit_request_id: string; amount_kobo: KoboValue; reason: string; description: string; }
export function workRow(value: unknown): WorkRow {
  const row = record(value); const amount = row.amount_kobo ?? row.outstanding_kobo ?? row.disputed_amount_kobo;
  return { id: text(row.id), state: optionalText(row.state), buyer_legal_name: optionalText(row.buyer_legal_name), buyer_user_id: optionalText(row.buyer_user_id), obligation_id: optionalText(row.obligation_id), credit_request_id: optionalText(row.credit_request_id), amount_kobo: amount === undefined ? null : kobo(amount), reason: optionalText(row.reason), description: optionalText(row.description) };
}
export interface Receivables { obligation_count: number; outstanding_kobo: KoboValue; overdue_kobo: KoboValue; }
export function receivables(value: unknown): Receivables {
  const row = record(record(value).summary);
  if (!Number.isSafeInteger(row.obligation_count) || Number(row.obligation_count) < 0) throw new RequestError('The summary could not be verified.', 0, 'invalid_response');
  return { obligation_count: Number(row.obligation_count), outstanding_kobo: kobo(row.outstanding_kobo), overdue_kobo: kobo(row.overdue_kobo) };
}
export function dateLabel(value: string): string {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return 'Date unavailable';
  const date = new Date(value + 'T12:00:00+01:00');
  return Number.isFinite(date.getTime()) && date.toISOString().slice(0, 10) === value ? new Intl.DateTimeFormat('en-NG', { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'Africa/Lagos' }).format(date) : 'Date unavailable';
}
export function timeLabel(value: string): string {
  const date = new Date(value);
  return Number.isFinite(date.getTime()) ? new Intl.DateTimeFormat('en-NG', { day: 'numeric', month: 'short', year: 'numeric', hour: 'numeric', minute: '2-digit', timeZone: 'Africa/Lagos', timeZoneName: 'short' }).format(date) : 'Time unavailable';
}
