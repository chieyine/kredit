import { checkedJSON, csrfHeader, randomKey, RequestError, UserFacingError, type Decoder } from './reliable';

type StoredIntent = { version: 1; key: string; fingerprint: string; createdAt: string };
export class MutationError extends UserFacingError {
  outcome: 'not_sent' | 'rejected' | 'unknown';
  constructor(message: string, outcome: MutationError['outcome'], readonly recoveryReference = '') { super(recoveryReference ? `${message} Support reference: ${recoveryReference}.` : message); this.name = 'MutationError'; this.outcome = outcome; }
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
  get recoveryReference() { return this.saved?.key ?? ''; }
  get unresolved() { return this.blocked || this.saved !== null; }
  private retain(item: StoredIntent) {
    if (!this.storage) throw new MutationError('Your browser cannot keep this request safe. Enable site storage or use another browser before submitting.', 'not_sent');
    this.storage.setItem(this.storageKey, JSON.stringify(item));
    this.saved = item;
  }
  private clear() {
    try { this.storage?.removeItem(this.storageKey); } catch { this.blocked = true; throw new MutationError('The response arrived, but this browser could not clear its saved request. Check the record before making another change.', 'unknown', this.recoveryReference); }
    this.saved = null; this.bornAt = new Date().toISOString();
  }
  async run<T>(payload: unknown, decode: Decoder<T>, method = 'POST'): Promise<T> {
    if (this.busy) throw new MutationError('This request is still being checked.', 'not_sent');
    if (this.blocked) throw new MutationError('An earlier request could not be verified. Check the record or contact support before submitting another.', 'unknown', this.recoveryReference);
    if (typeof navigator !== 'undefined' && !navigator.onLine) {
      if (this.unresolved) throw new MutationError('You are offline and an earlier request is still unconfirmed. Reconnect before checking its record or retrying the same details.', 'unknown', this.recoveryReference);
      throw new MutationError('You are offline. This request has not been sent.', 'not_sent');
    }
    this.busy = true;
    let sent = false;
    const wasUnresolved = this.saved !== null;
    try {
      const body = payload === undefined ? undefined : JSON.stringify(canonical(payload));
      if (!globalThis.crypto?.subtle) throw new MutationError('Use an updated, secure browser to submit this request.', 'not_sent');
      const fingerprint = Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(`${method}\n${this.url}\n${body ?? ''}`))), byte => byte.toString(16).padStart(2, '0')).join('');
      if (this.saved && this.saved.fingerprint !== fingerprint) throw new MutationError('Your earlier request may already have completed. Check the record or retry with the same details before making a different request.', 'unknown', this.recoveryReference);
      // Do not silently expire unknown requests and submit them under a fresh key.
      if (this.saved && Date.now() - Date.parse(this.saved.createdAt) > 15 * 60 * 1000) throw new MutationError('This earlier request needs a record check. Do not submit it again; open the sale history or contact support.', 'unknown', this.recoveryReference);
      const item: StoredIntent = this.saved ?? { version: 1, key: randomKey(), fingerprint, createdAt: new Date().toISOString() };
      this.retain(item);
      sent = true;
      const result = await checkedJSON(this.url, decode, { method, headers: { ...csrfHeader(), 'Idempotency-Key': item.key, ...(body === undefined ? {} : { 'Content-Type': 'application/json' }) }, body });
      this.clear();
      return result;
    } catch (error) {
      if (error instanceof MutationError) throw error;
      if (sent && error instanceof RequestError && error.status === 409 && ['approval_changed','network_changed', 'purchasing_changed'].includes(error.code)) {
        this.clear();
        throw new MutationError('The record changed. Refresh it before making another change.', 'rejected');
      }
      if (sent && error instanceof RequestError && error.status === 409 && error.code === 'credit_approval_required') {
        this.clear();
        throw new MutationError('Request internal approval for these exact terms. Once approved, you can send the offer.', 'rejected');
      }
      // This response is emitted only before a financial-review write occurs.
      if (sent && error instanceof RequestError && error.status === 409 && error.code === 'financial_difference_unresolved') {
        this.clear();
        throw new MutationError('Financial discrepancy remains unresolved. Correct the underlying records before closing this review.', 'rejected');
      }
      if (sent && !wasUnresolved && error instanceof RequestError && error.code !== 'invalid_response' && [400, 401, 403, 404, 422].includes(error.status)) {
        this.clear();
        const message = error.status === 401 ? 'Your session has ended. Sign in again before continuing.' : error.status === 403 ? (error.code === 'step_up_required' ? 'Confirm your identity with your authenticator code, then try again.' : 'Your account cannot perform this action. Check your permissions.') : 'The request was not accepted. Check the details and try again.';
        throw new MutationError(message, 'rejected');
      }
      throw new MutationError(sent ? 'We have not confirmed the result. Check the record or retry the same request; do not submit it as a new request.' : 'This request has not been sent. Check your browser and connection.', sent ? 'unknown' : 'not_sent', sent ? this.recoveryReference : '');
    } finally { this.busy = false; }
  }
}
