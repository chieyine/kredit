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
// Kredit's business day is Lagos, wherever the operator happens to be.
export function localTime(value:string){
  const at = new Date(value);
  return Number.isFinite(at.getTime())
    ? new Intl.DateTimeFormat('en-NG',{dateStyle:'medium',timeStyle:'short',timeZone:'Africa/Lagos'}).format(at)
    : 'Time unavailable';
}
/** A `datetime-local` value the operator reads as Lagos wall time. */
export function localInput(value:string){
  const at = new Date(value);
  if(!Number.isFinite(at.getTime())) return '';
  const parts = new Intl.DateTimeFormat('en-CA',{timeZone:'Africa/Lagos',year:'numeric',month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit',hour12:false}).formatToParts(at);
  const get = (type:string) => parts.find(part => part.type === type)?.value ?? '';
  return `${get('year')}-${get('month')}-${get('day')}T${get('hour')}:${get('minute')}`;
}
/** The Lagos wall time the operator typed, as an exact instant. */
export function lagosISO(value:string){
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(value)) throw new Error('Enter a valid date and time.');
  const at = new Date(`${value}:00+01:00`);
  if(!Number.isFinite(at.getTime()) || new Date(at.getTime() + 3600000).toISOString().slice(0,16) !== value) throw new Error('Enter a valid date and time.');
  return at.toISOString();
}
