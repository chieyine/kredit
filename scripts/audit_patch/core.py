def apply(write, replace, transform, remove):
    write('web/src/lib/api/reliable.ts', r'''/** Explicit outcomes for unreliable networks. Financial writes are never retried automatically. */
export type Decoder<T> = (value: unknown) => T;
export type Resource<T> =
  | { state: 'loading'; scope: string }
  | { state: 'ready'; scope: string; data: T; checkedAt: string }
  | { state: 'error'; scope: string; message: string; status: number };

export class RequestError extends Error {
  status: number;
  code: string;
  constructor(message: string, status = 0, code = 'request_unavailable') {
    super(message); this.name = 'RequestError'; this.status = status; this.code = code;
  }
}
export function record(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new RequestError('The response was incomplete.', 0, 'invalid_response');
  return value as Record<string, unknown>;
}
export function text(value: unknown): string {
  if (typeof value !== 'string') throw new RequestError('The response was incomplete.', 0, 'invalid_response');
  return value;
}
export function rows<T>(key: string, decode: Decoder<T>): Decoder<T[]> {
  return value => {
    const list = record(value)[key];
    if (!Array.isArray(list)) throw new RequestError('We could not verify this list.', 0, 'invalid_response');
    return list.map(decode);
  };
}
export function publicError(error: unknown, subject = 'these details'): string {
  if (error instanceof RequestError && error.status === 401) return 'Your session has ended. Sign in again to continue.';
  if (error instanceof RequestError && error.status === 403) return 'Your account does not have permission to open these details.';
  if (error instanceof RequestError && error.status === 429) return 'Too many requests. Wait a moment, then try again.';
  return `We could not check ${subject}. Check your connection and try again.`;
}
export async function boundedFetch(url: string, init: RequestInit = {}, timeoutMs = 20000): Promise<Response> {
  const controller = new AbortController();
  const abort = () => controller.abort();
  if (init.signal?.aborted) abort(); else init.signal?.addEventListener('abort', abort, { once: true });
  const timer = setTimeout(abort, timeoutMs);
  try { return await fetch(url, { ...init, credentials: 'include', cache: 'no-store', signal: controller.signal }); }
  finally { clearTimeout(timer); init.signal?.removeEventListener('abort', abort); }
}
export async function checkedJSON<T>(url: string, decode: Decoder<T>, init: RequestInit = {}): Promise<T> {
  // Keep this deadline alive through body decoding, not merely until headers arrive.
  const controller = new AbortController();
  const abort = () => controller.abort();
  if (init.signal?.aborted) abort(); else init.signal?.addEventListener('abort', abort, { once: true });
  const timer = setTimeout(abort, 20000);
  try {
    const response = await fetch(url, { ...init, credentials: 'include', cache: 'no-store', signal: controller.signal });
    let payload: unknown;
    try { payload = await response.json(); }
    catch { throw new RequestError('We could not verify the response.', response.ok ? 0 : response.status, 'invalid_response'); }
    if (!response.ok) {
      const problem = payload && typeof payload === 'object' && !Array.isArray(payload) ? payload as Record<string, unknown> : {};
      const code = typeof problem.code === 'string' ? problem.code : typeof problem.title === 'string' ? problem.title : 'request_unavailable';
      // Storage/provider diagnostics are not safe public copy.
      throw new RequestError('The request was not confirmed.', response.status, code);
    }
    return decode(payload);
  } finally { clearTimeout(timer); init.signal?.removeEventListener('abort', abort); }
}
export async function readResource<T>(scope: string, url: string, decode: Decoder<T>, signal?: AbortSignal, subject = 'these details'): Promise<Resource<T>> {
  try { return { state: 'ready', scope, data: await checkedJSON(url, decode, { signal }), checkedAt: new Date().toISOString() }; }
  catch (error) { return { state: 'error', scope, message: publicError(error, subject), status: error instanceof RequestError ? error.status : 0 }; }
}
/** A business switch or teardown invalidates every older response. */
export class LatestRequest {
  private controller: AbortController | null = null;
  private generation = 0;
  begin() {
    this.cancel(); const generation = this.generation; const controller = new AbortController(); this.controller = controller;
    return { signal: controller.signal, current: () => generation === this.generation && !controller.signal.aborted };
  }
  cancel() { this.generation += 1; this.controller?.abort(); this.controller = null; }
}
export function randomKey(): string {
  if (!globalThis.crypto?.getRandomValues) throw new RequestError('This browser cannot securely submit the request. Use an updated browser.');
  return `kredit-${Array.from(crypto.getRandomValues(new Uint8Array(16)), byte => byte.toString(16).padStart(2, '0')).join('')}`;
}
export function csrfHeader(): Record<string, string> {
  if (typeof document === 'undefined') return {};
  const token = document.cookie.split('; ').find(cookie => cookie.startsWith('kredit_csrf='))?.split('=')[1];
  return token ? { 'X-CSRF-Token': token } : {};
}
export function clearPrivateBrowserData(): void {
  if (typeof window === 'undefined') return;
  for (const storage of [() => window.sessionStorage, () => window.localStorage]) {
    try {
      const store = storage();
      for (let index = store.length - 1; index >= 0; index -= 1) {
        const key = store.key(index);
        if (key && /^(kredit\.(quick-sale\.|intent\.|bank-return\.|account\.))/.test(key)) store.removeItem(key);
      }
    } catch { /* Server revocation remains authoritative when browser storage is unavailable. */ }
  }
}
export function safeNext(value: string | null, origin: string): string {
  if (!value || !value.startsWith('/') || value.startsWith('//') || /[\\\x00-\x1f]/.test(value)) return '/app/overview';
  try { const parsed = new URL(value, origin); return parsed.origin === origin && parsed.pathname !== '/app' ? parsed.pathname + parsed.search + parsed.hash : '/app/overview'; }
  catch { return '/app/overview'; }
}
export function normalizeNigerianPhone(value: string): string {
  const compact = value.trim().replace(/[\s()-]/g, '');
  if (/^0\d{10}$/.test(compact)) return '+234' + compact.slice(1);
  if (/^234\d{10}$/.test(compact)) return '+' + compact;
  if (/^\+234\d{10}$/.test(compact)) return compact;
  throw new RequestError('Enter an 11-digit Nigerian phone number, or a number starting with +234.', 400, 'invalid_phone');
}
''')
    write('web/src/lib/api/mutation.ts', r'''import { checkedJSON, csrfHeader, randomKey, RequestError, type Decoder } from './reliable';

type StoredIntent = { version: 1; key: string; fingerprint: string; createdAt: string };
export class MutationError extends Error {
  outcome: 'not_sent' | 'rejected' | 'unknown';
  constructor(message: string, outcome: MutationError['outcome']) { super(message); this.name = 'MutationError'; this.outcome = outcome; }
}
function canonical(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(canonical);
  if (value && typeof value === 'object') return Object.fromEntries(Object.entries(value).filter(([, v]) => v !== undefined).sort(([a], [b]) => a.localeCompare(b)).map(([k, v]) => [k, canonical(v)]));
  if (typeof value === 'number' && !Number.isFinite(value)) throw new MutationError('Check the amount before continuing.', 'not_sent');
  return value;
}
/** Persist only a random request identity, timestamp and payload digest; never payment evidence or credentials. */
export class MutationIntent {
  private storage: Storage | null = null;
  private storageKey: string;
  private saved: StoredIntent | null = null;
  private blocked = false;
  private busy = false;
  private bornAt = new Date().toISOString();
  private url: string;
  constructor(scope: string, url: string, storage?: Storage) {
    this.url = url;
    this.storageKey = `kredit.intent.v1:${encodeURIComponent(scope)}:${encodeURIComponent(url)}`;
    try {
      this.storage = storage ?? (typeof window === 'undefined' ? null : window.sessionStorage);
      const raw = this.storage?.getItem(this.storageKey);
      if (raw) {
        const item = JSON.parse(raw) as Partial<StoredIntent>;
        if (item.version !== 1 || typeof item.key !== 'string' || !/^kredit-[0-9a-f]{32}$/.test(item.key) || typeof item.fingerprint !== 'string' || !/^[0-9a-f]{64}$/.test(item.fingerprint) || typeof item.createdAt !== 'string' || !Number.isFinite(Date.parse(item.createdAt))) this.blocked = true;
        else { this.saved = item as StoredIntent; this.bornAt = item.createdAt; }
      }
    } catch { this.blocked = true; }
  }
  get createdAt() { return this.bornAt; }
  get unresolved() { return this.blocked || this.saved !== null; }
  private retain(item: StoredIntent) {
    if (!this.storage) throw new MutationError('Your browser cannot keep this request safe. Enable site storage or use another browser before submitting.', 'not_sent');
    this.storage.setItem(this.storageKey, JSON.stringify(item));
    this.saved = item;
  }
  private clear() {
    try { this.storage?.removeItem(this.storageKey); } catch { /* Replaying a retained key is safer than inventing another. */ }
    this.saved = null; this.bornAt = new Date().toISOString();
  }
  async run<T>(payload: unknown, decode: Decoder<T>, method = 'POST'): Promise<T> {
    if (this.busy) throw new MutationError('This request is still being checked.', 'not_sent');
    if (this.blocked) throw new MutationError('An earlier request could not be verified. Check the record or contact support before submitting another.', 'unknown');
    if (typeof navigator !== 'undefined' && !navigator.onLine) throw new MutationError('You are offline. This request has not been sent.', 'not_sent');
    this.busy = true;
    let sent = false;
    const wasUnresolved = this.saved !== null;
    try {
      const body = payload === undefined ? undefined : JSON.stringify(canonical(payload));
      if (!globalThis.crypto?.subtle) throw new MutationError('Use an updated, secure browser to submit this request.', 'not_sent');
      const fingerprint = Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(`${method}\n${this.url}\n${body ?? ''}`))), byte => byte.toString(16).padStart(2, '0')).join('');
      if (this.saved && this.saved.fingerprint !== fingerprint) throw new MutationError('Your earlier request may already have completed. Check the record or retry with the same details before making a different request.', 'unknown');
      // Do not silently expire unknown requests and submit them under a fresh key.
      if (this.saved && Date.now() - Date.parse(this.saved.createdAt) > 15 * 60 * 1000) throw new MutationError('This earlier request needs a record check. Do not submit it again; open the sale history or contact support.', 'unknown');
      const item: StoredIntent = this.saved ?? { version: 1, key: randomKey(), fingerprint, createdAt: this.bornAt };
      this.retain(item);
      sent = true;
      const result = await checkedJSON(this.url, decode, { method, headers: { ...csrfHeader(), 'Idempotency-Key': item.key, ...(body === undefined ? {} : { 'Content-Type': 'application/json' }) }, body });
      this.clear();
      return result;
    } catch (error) {
      if (error instanceof MutationError) throw error;
      if (sent && !wasUnresolved && error instanceof RequestError && error.code !== 'invalid_response' && [400, 401, 403, 404, 422].includes(error.status)) {
        this.clear();
        const message = error.status === 401 ? 'Your session has ended. Sign in again before continuing.' : error.status === 403 ? 'Your account cannot perform this action. Check your permissions.' : 'The request was not accepted. Check the details and try again.';
        throw new MutationError(message, 'rejected');
      }
      throw new MutationError(sent ? 'We have not confirmed the result. Check the record or retry the same request; do not submit it as a new payment.' : 'This request has not been sent. Check your browser and connection.', sent ? 'unknown' : 'not_sent');
    } finally { this.busy = false; }
  }
}
''')
    write('web/src/lib/account-context.ts', r'''export const ACCOUNT_CONTEXT = 'kredit.account.v1';
export interface AccountContext { readonly userID: string; }
''')
    write('web/src/lib/sale-drafts.ts', r'''export type SaleDraft = { goods: string; principal: string; dueDate: string };
const PREFIX = 'kredit.quick-sale.v2:';
const TTL = 12 * 60 * 60 * 1000;
function key(userID: string, organizationID: string) {
  if (!userID || !organizationID) throw new Error('An authenticated business is required for draft storage.');
  return `${PREFIX}${encodeURIComponent(userID)}:${encodeURIComponent(organizationID)}`;
}
export function readDraft(userID: string, organizationID: string, store: Storage, now = Date.now()): SaleDraft | null {
  try {
    store.removeItem('kredit.quick-sale.draft.v1');
    const storageKey = key(userID, organizationID);
    const raw = store.getItem(storageKey);
    if (!raw) return null;
    const value = JSON.parse(raw);
    if (value.userID !== userID || value.organizationID !== organizationID || !Number.isFinite(value.expiresAt) || value.expiresAt <= now || value.expiresAt > now + TTL || typeof value.goods !== 'string' || value.goods.length > 5000 || typeof value.principal !== 'string' || value.principal.length > 40 || typeof value.dueDate !== 'string' || (value.dueDate && !/^\d{4}-\d{2}-\d{2}$/.test(value.dueDate))) { store.removeItem(storageKey); return null; }
    return { goods: value.goods, principal: value.principal, dueDate: value.dueDate };
  } catch { return null; }
}
export function saveDraft(userID: string, organizationID: string, draft: SaleDraft, store: Storage, now = Date.now()): boolean {
  try { store.setItem(key(userID, organizationID), JSON.stringify({ ...draft, userID, organizationID, expiresAt: now + TTL })); return true; }
  catch { return false; }
}
export function deleteDraft(userID: string, organizationID: string, store: Storage): void {
  try { store.removeItem(key(userID, organizationID)); } catch { /* The form still works without draft storage. */ }
}
''')
    write('web/src/lib/records.ts', r'''import { exactKobo, type KoboValue } from './money';
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
  id: string; state: string; supplier_legal_name: string; buyer_legal_name: string; buyer_user_id: string; buyer_business_id: string;
  principal_kobo: KoboValue; goods_description: string; due_date: string; collection_at: string; grace_hours: number;
  schedule_type: string; schedule_count: number; schedule_cadence: string; fee_terms: FeeTerms | null;
  custom_schedule_items: { amount_kobo: KoboValue; due_date: string }[];
}
export interface SaleView {
  request: SaleRequest;
  agreement: { id: string; document_hash: string } | null;
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
      id: text(row.id), state: text(row.state), supplier_legal_name: optionalText(row.supplier_legal_name), buyer_legal_name: text(row.buyer_legal_name), buyer_user_id: optionalText(row.buyer_user_id), buyer_business_id: optionalText(row.buyer_business_id),
      principal_kobo: kobo(row.principal_kobo), goods_description: optionalText(row.goods_description), due_date: text(row.due_date), collection_at: optionalText(row.collection_at), grace_hours: typeof row.grace_hours === 'number' ? row.grace_hours : 0,
      schedule_type: optionalText(row.schedule_type), schedule_count: typeof row.schedule_count === 'number' ? row.schedule_count : 1, schedule_cadence: optionalText(row.schedule_cadence), fee_terms: validFeeTerms(row.fee_terms) ? row.fee_terms : null,
      custom_schedule_items: Array.isArray(row.custom_schedule_items) ? row.custom_schedule_items.map(value => { const item = record(value); return { amount_kobo: kobo(item.amount_kobo), due_date: text(item.due_date) }; }) : []
    },
    agreement: agreement ? { id: text(agreement.id), document_hash: text(agreement.document_hash) } : null,
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
  return Number.isFinite(date.getTime()) ? new Intl.DateTimeFormat('en-NG', { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'Africa/Lagos' }).format(date) : 'Date unavailable';
}
export function timeLabel(value: string): string {
  const date = new Date(value);
  return Number.isFinite(date.getTime()) ? new Intl.DateTimeFormat('en-NG', { day: 'numeric', month: 'short', year: 'numeric', hour: 'numeric', minute: '2-digit', timeZone: 'Africa/Lagos', timeZoneName: 'short' }).format(date) : 'Time unavailable';
}
''')
    write('web/src/lib/fee-terms.ts', r'''import { exactKobo, type KoboValue } from './money';
export type FeeTerms = { policy_revision: number; base_bps: number; collection_bps: number };
export function validFeeTerms(value: unknown): value is FeeTerms {
  if (!value || typeof value !== 'object') return false;
  const item = value as Partial<FeeTerms>;
  return Number.isInteger(item.base_bps) && Number.isInteger(item.collection_bps) && Number(item.base_bps) >= 0 && Number(item.base_bps) <= 1000 && Number(item.collection_bps) >= 0 && Number(item.collection_bps) <= 1000;
}
export function feeDisclosure(terms?: FeeTerms | null): string {
  if (!validFeeTerms(terms)) return 'Fee terms are unavailable. Please refresh before accepting.';
  return `The seller pays ${terms.base_bps / 100}% when this sale becomes active, plus ${terms.collection_bps / 100}% on any amount Kredit successfully collects after the permitted collection time. These fees are not added to the buyer’s principal.`;
}
export function feeForKobo(value: KoboValue, basisPoints: number): bigint | null {
  const amount = exactKobo(value);
  if (amount === null || amount < 0n || !Number.isInteger(basisPoints) || basisPoints < 0 || basisPoints > 1000) return null;
  return amount * BigInt(basisPoints) / 10000n;
}
''')
    write('web/src/lib/financial-copy.ts', r'''import { formatKobo, type KoboValue } from './money';
export const creditBoundary = 'You provide the goods. Kredit does not lend money or guarantee repayment.';
export const collectionBoundary = 'Bank debit needs valid permission, the agreed collection time and the applicable payment and dispute checks. A debit can still fail.';
export const submittedDebitCaveat = 'A debit request already sent to the bank may still complete.';
export function disputeEffectCopy(effect: string, amount?: KoboValue): string {
  if (effect === 'FULL_BLOCK') return `New bank debit requests for this sale will be on hold once this report is saved. ${submittedDebitCaveat}`;
  if (effect === 'NO_AUTOMATIC_BLOCK') return `This report does not request an automatic hold. Eligible payments may continue. ${submittedDebitCaveat}`;
  return `Only the amount in question${amount == null ? '' : ` (${formatKobo(amount)})`} will be held once this report is saved. Undisputed amounts may still be collected. ${submittedDebitCaveat}`;
}
export function hostedAuthorizationURL(value: string, provider: string): string | null {
  if (provider !== 'mono-sweep') return null;
  try {
    const url = new URL(value);
    return url.protocol === 'https:' && url.hostname === 'authorise.mono.co' && !url.port && !url.username && !url.password && url.pathname !== '/' ? url.href : null;
  } catch { return null; }
}
export function acceptanceMessage(state: string): string {
  if (state === 'READY_TO_RELEASE') return 'Sale accepted. Bank permission is ready, and the seller can arrange your goods.';
  if (state === 'BUYER_ACCEPTED') return 'Sale accepted. Bank permission must be ready before the seller can release the goods.';
  return 'Your sale has been updated. Check its current status below.';
}
''')
    write('internal/credit/calendar.go', r'''package credit

import (
 "errors"
 "time"
)

// CollectionInstant preserves the existing quick-sale Lagos cutoff (23:59)
// while removing the browser's timezone from financially material terms.
// This helper does not change any already accepted agreement.
func CollectionInstant(dueDate string, graceHours int) (time.Time, error) {
 if graceHours < 0 || graceHours > 720 { return time.Time{}, errors.New("grace hours must be between 0 and 720") }
 location, err := time.LoadLocation("Africa/Lagos")
 if err != nil { return time.Time{}, err }
 date, err := time.ParseInLocation("2006-01-02", dueDate, location)
 if err != nil || len(dueDate) != 10 || date.Format("2006-01-02") != dueDate { return time.Time{}, errors.New("a valid payment date is required") }
 cutoff := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 0, 0, location)
 return cutoff.Add(time.Duration(graceHours) * time.Hour).UTC(), nil
}
''')
    write('internal/web/credit_terms_handlers.go', r'''package web

import (
 "net/http"
 "kredit/internal/access"
 "kredit/internal/credit"
)

func (s *Server) previewCreditTerms(w http.ResponseWriter, r *http.Request) {
 organizationID, err := pathID(r, "organizationID")
 if err != nil { writeProblem(w, 400, "invalid_path", "Choose a business first."); return }
 if _, _, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionCreateCredit); !ok { return }
 if !s.requireCSRF(w, r) { return }
 var input struct { DueDate string `json:"due_date"`; GraceHours int `json:"grace_hours"` }
 if err := decodeJSON(w, r, &input); err != nil { writeProblem(w, 400, "invalid_request", "Check the payment date and extra hours."); return }
 instant, err := credit.CollectionInstant(input.DueDate, input.GraceHours)
 if err != nil { writeProblem(w, 422, "credit_terms_invalid", "Check the payment date and extra hours."); return }
 writeJSON(w, 200, map[string]any{"due_date": input.DueDate, "grace_hours": input.GraceHours, "collection_at": instant, "timezone": "Africa/Lagos", "cutoff": "23:59", "timing_mode": "lagos_end_of_day"})
}
''')
    replace('internal/web/server.go', '\ts.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/credit-requests", s.createCreditRequest)', '\ts.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/credit-terms/preview", s.previewCreditTerms)\n\ts.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/credit-requests", s.createCreditRequest)')
    replace('internal/web/credit_handlers.go', 'type creditRequestInput struct {', 'type creditRequestInput struct {\n\tTimingMode string `json:"timing_mode,omitempty"`')
    replace('internal/web/credit_handlers.go', '\torg, exists := s.runtime.Organizations.Get(orgID)\n', '''\tif in.TimingMode != "" {
        if in.TimingMode != "lagos_end_of_day" { writeProblem(w, 422, "credit_terms_invalid", "That payment timing option is not supported."); return }
        canonical, timingErr := credit.CollectionInstant(in.DueDate, in.GraceHours)
        if timingErr != nil || (!in.CollectionAt.IsZero() && !in.CollectionAt.Equal(canonical)) { writeProblem(w, 422, "credit_terms_changed", "Review the payment date again before saving."); return }
        in.CollectionAt = canonical
    }
\torg, exists := s.runtime.Organizations.Get(orgID)
''', 1)
    replace('internal/web/credit_handlers.go', 'writeProblem(w, 503, "payment_unavailable", "Your payment company cannot cancel a request that has already been sent.")', 'writeProblem(w, 503, "payment_unavailable", "Payment recording is temporarily unavailable. No payment has been confirmed.")')
    write('internal/providers/mono/hosted_url.go', r'''package mono

import (
 "errors"
 "net/url"
)

func validateHostedAuthorizationURL(value string) error {
 parsed, err := url.Parse(value)
 if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "authorise.mono.co" || parsed.Port() != "" || parsed.User != nil || parsed.Path == "" || parsed.Path == "/" { return errors.New("unapproved hosted authorization URL") }
 return nil
}
''')
    replace('internal/providers/mono/mono.go', 'if !successfulEnvelope(out.Status) || id == "" || out.Data.MonoURL == "" {', 'if !successfulEnvelope(out.Status) || id == "" || validateHostedAuthorizationURL(out.Data.MonoURL) != nil {')
    def api_update(source):
        marker = '    CreditRequestInput:\n'
        start = source.index(marker)
        pos = source.index('      properties:\n', start) + len('      properties:\n')
        source = source[:pos] + '        timing_mode: {type: string, enum: [lagos_end_of_day], description: Server-validated Africa/Lagos end-of-day collection timing.}\n' + source[pos:]
        endpoint = '''  /organizations/{organizationID}/credit-terms/preview:
    parameters:
      - $ref: '#/components/parameters/OrganizationID'
    post:
      operationId: previewCreditTerms
      summary: Preview server-calculated Nigerian payment timing without creating a sale
      description: Requires organization access and CSRF. Does not move money or change existing agreements.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              additionalProperties: false
              required: [due_date, grace_hours]
              properties:
                due_date: {type: string, format: date}
                grace_hours: {type: integer, minimum: 0, maximum: 720}
      responses:
        '200':
          description: Canonical timing for review before creating a sale.
          content:
            application/json:
              schema:
                type: object
                required: [due_date, grace_hours, collection_at, timezone, cutoff, timing_mode]
                properties:
                  due_date: {type: string, format: date}
                  grace_hours: {type: integer}
                  collection_at: {type: string, format: date-time}
                  timezone: {type: string, const: Africa/Lagos}
                  cutoff: {type: string, const: '23:59'}
                  timing_mode: {type: string, const: lagos_end_of_day}
        '400': {description: Invalid input.}
        '401': {description: Sign-in required.}
        '403': {description: Organization access or CSRF check failed.}
        '422': {description: Invalid date or grace period.}
        '503': {description: Service unavailable.}
'''
        return source.replace('paths:\n', 'paths:\n' + endpoint, 1)
    transform('api/openapi.yaml', api_update)
    replace('web/src/lib/api/client.ts', "import createClient from 'openapi-fetch';", "import createClient from 'openapi-fetch';\nimport { boundedFetch, clearPrivateBrowserData } from './reliable';")
    replace('web/src/lib/api/client.ts', "\tawait fetch('/api/v1/auth/logout', {", "\tconst response = await boundedFetch('/api/v1/auth/logout', {")
    replace('web/src/lib/api/client.ts', "\tlocation.assign('/app');", "\tif (!response.ok && response.status !== 401) throw new Error('Sign-out was not confirmed. Your account may still be open. Try again before leaving this device.');\n\tclearPrivateBrowserData();\n\tlocation.assign('/app?signed_out=1');")
