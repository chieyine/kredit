import {csrfHeaders,idempotencyKey} from '$lib/api/client';

function fail(response: Response, data: any, fallback: string): Error {
  if (response.status === 401 && data?.title === 'mfa_invalid') return new Error('That authenticator code did not work. Check the six digits and try again.');
  if (response.status === 503 && data?.title === 'recovery_codes_unavailable') return new Error('Backup codes could not be created. Refresh your safety settings and make a new set.');
  if (response.status === 401) return new Error('Your session has ended. Sign in again.');
  if (response.status === 403 && data?.title === 'step_up_required') return new Error('Confirm your identity with your authenticator code, then try again.');
  if (response.status === 403) return new Error('Your account does not have permission for this.');
  if (response.status === 409) return new Error('Somebody changed this first. Open it again and check.');
  if (response.status >= 500) return new Error('Kredit could not complete that. Try again in a moment.');
  return new Error(typeof data?.detail === 'string' && data.detail ? data.detail : fallback);
}
// Keep the deadline active until the body has been read. A successful status
// with an unreadable body is an unknown outcome, never a confirmed change.
async function request(path: string, init: RequestInit = {}) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 20_000);
  const writing = init.method && init.method !== 'GET';
  try {
    const response = await fetch(path, { ...init, credentials: 'include', cache: 'no-store', signal: init.signal ? AbortSignal.any([init.signal, controller.signal]) : controller.signal });
    let data: any;
    try { data = response.status === 204 ? {} : await response.json(); }
    catch {
      if (!response.ok) throw fail(response, {}, 'We could not open this.');
      throw new Error(writing
        ? 'We could not confirm the result. Check the record before trying again.'
        : 'The response was incomplete. Try again.');
    }
    if (!response.ok) throw fail(response, data, writing ? 'That change was not saved.' : 'We could not open this.');
    if (!data || typeof data !== 'object' || Array.isArray(data)) throw new Error('The response was incomplete. Check the record before trying again.');
    return data;
  } catch (error) {
    if (controller.signal.aborted || error instanceof TypeError) throw new Error(writing
      ? 'We could not confirm the result. Check the record before trying again.'
      : 'We could not open this. Check your connection and try again.');
    throw error;
  } finally { clearTimeout(timer); }
}
export async function adminGet(path: string, signal?: AbortSignal) { return request(path, {signal}); }
export async function adminPost(path: string, payload: unknown, method: 'POST' | 'PATCH' | 'DELETE' | 'PUT' = 'POST', requestKey = idempotencyKey()) {
  return request(path, { method, headers: { 'Content-Type': 'application/json', 'Idempotency-Key': requestKey, ...csrfHeaders() }, body: JSON.stringify(payload) });
}
export { localTime, localInput, lagosISO } from '$lib/business-time';
