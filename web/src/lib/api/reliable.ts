/** Explicit outcomes for unreliable networks. Financial writes are never retried automatically. */
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
        if (key && (key === 'kredit:saved-sale-items' || /^(kredit\.(quick-sale\.|intent\.|bank-return\.|account\.))/.test(key))) store.removeItem(key);
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
