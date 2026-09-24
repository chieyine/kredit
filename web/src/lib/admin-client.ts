import { csrfHeaders, idempotencyKey } from '$lib/api/client';

type ProblemBody = { title?: unknown; detail?: unknown };

function fail(response: Response, data: ProblemBody, fallback: string): Error {
	if (response.status === 401 && data?.title === 'mfa_invalid')
		return new Error('That authenticator code did not work. Check the six digits and try again.');
	if (response.status === 503 && data?.title === 'recovery_codes_unavailable')
		return new Error('Backup codes could not be created. Refresh your safety settings and make a new set.');
	if (response.status === 401) return new Error('Your session has ended. Sign in again.');
	if (response.status === 403 && data?.title === 'step_up_required')
		return new Error('Confirm your identity with your authenticator code, then try again.');
	if (response.status === 403) return new Error('Your account does not have permission for this.');
	if (response.status === 409) return new Error('Somebody changed this first. Open it again and check.');
	if (response.status >= 500) return new Error('Kredit could not complete that. Try again in a moment.');
	return new Error(typeof data?.detail === 'string' && data.detail ? data.detail : fallback);
}
// Keep the deadline active until the body has been read. A successful status
// with an unreadable body is an unknown outcome, never a confirmed change.
// The type parameter documents the server contract the caller relies on; it is
// not a runtime check. Callers that act on money or state must still validate.
async function request<T extends object>(path: string, init: RequestInit = {}): Promise<T> {
	const controller = new AbortController();
	const timer = setTimeout(() => controller.abort(), 20_000);
	const writing = init.method && init.method !== 'GET';
	try {
		const response = await fetch(path, {
			...init,
			credentials: 'include',
			cache: 'no-store',
			signal: init.signal ? AbortSignal.any([init.signal, controller.signal]) : controller.signal
		});
		let data: unknown;
		try {
			data = response.status === 204 ? {} : await response.json();
		} catch {
			if (!response.ok) throw fail(response, {}, 'We could not open this.');
			throw new Error(
				writing
					? 'We could not confirm the result. Check the record before trying again.'
					: 'The response was incomplete. Try again.'
			);
		}
		if (!response.ok)
			throw fail(
				response,
				(data ?? {}) as ProblemBody,
				writing ? 'That change was not saved.' : 'We could not open this.'
			);
		if (!data || typeof data !== 'object' || Array.isArray(data))
			throw new Error('The response was incomplete. Check the record before trying again.');
		return data as T;
	} catch (error) {
		if (controller.signal.aborted || error instanceof TypeError)
			throw new Error(
				writing
					? 'We could not confirm the result. Check the record before trying again.'
					: 'We could not open this. Check your connection and try again.'
			);
		throw error;
	} finally {
		clearTimeout(timer);
	}
}
export type AdminResponse = Record<string, unknown>;

export type GovernanceMode = 'solo_owner' | 'delegated_team';

/** The approval rule from a response carrying `governance`, or null when it is missing or unknown. */
export function governanceMode(body: AdminResponse): GovernanceMode | null {
	const governance = body.governance;
	if (!governance || typeof governance !== 'object') return null;
	const mode = (governance as Record<string, unknown>).mode;
	return mode === 'solo_owner' || mode === 'delegated_team' ? mode : null;
}

/** The caller's admin roles from /api/v1/ops/capabilities; throws when the list cannot be verified. */
export function adminRoles(body: AdminResponse): string[] {
	const roles = body.roles;
	if (!Array.isArray(roles) || roles.some((role) => typeof role !== 'string'))
		throw new Error('Admin permissions could not be verified.');
	return roles as string[];
}

/** A thrown value's message when it is a real Error with one, otherwise the fallback. */
export function errorMessage(cause: unknown, fallback: string): string {
	return cause instanceof Error && cause.message ? cause.message : fallback;
}

/** A verified operations command preview: what the command will do before anyone applies it. */
export interface CommandPreview {
	current_version: number;
	impact_preview: { effect: string; will_notify: boolean; audit: string };
}

/** Reads and verifies the preview returned by /api/v1/ops/commands/preview. */
export function commandPreview(body: AdminResponse): CommandPreview {
	const command = body.command;
	if (!command || typeof command !== 'object') throw new Error('The impact preview could not be verified.');
	const { current_version: version, impact_preview: impact } = command as Record<string, unknown>;
	if (!impact || typeof impact !== 'object') throw new Error('The impact preview could not be verified.');
	const { effect, will_notify: notify, audit } = impact as Record<string, unknown>;
	if (typeof effect !== 'string') throw new Error('The impact preview could not be verified.');
	return {
		current_version: typeof version === 'number' ? version : 0,
		impact_preview: { effect, will_notify: notify === true, audit: typeof audit === 'string' ? audit : '' }
	};
}

export async function adminGet<T extends object = AdminResponse>(path: string, signal?: AbortSignal) {
	return request<T>(path, { signal });
}
export async function adminPost<T extends object = AdminResponse>(
	path: string,
	payload: unknown,
	method: 'POST' | 'PATCH' | 'DELETE' | 'PUT' = 'POST',
	requestKey = idempotencyKey()
) {
	return request<T>(path, {
		method,
		headers: { 'Content-Type': 'application/json', 'Idempotency-Key': requestKey, ...csrfHeaders() },
		body: JSON.stringify(payload)
	});
}
export { localTime, localInput, lagosISO } from '$lib/business-time';
