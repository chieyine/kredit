import { csrfHeaders, idempotencyKey } from '$lib/api/client';

import { checkedJSON, record } from '$lib/api/reliable';

// The shared transport owns deadlines, safe errors and response parsing.
// Typed admin projections still validate fields at their point of use.
async function request<T extends object>(path: string, init: RequestInit = {}): Promise<T> {
	return checkedJSON(path, (value) => record(value) as T, init);
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
