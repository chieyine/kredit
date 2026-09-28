import { boundedFetch, clearPrivateBrowserData, csrfHeader } from './reliable';

export function csrfHeaders(): HeadersInit {
	return csrfHeader();
}

export function idempotencyKey(): string {
	if (typeof crypto === 'undefined') throw new Error('Secure browser randomness is unavailable.');
	if (typeof crypto.randomUUID === 'function') return crypto.randomUUID();
	const bytes = crypto.getRandomValues(new Uint8Array(16));
	return `idem-${Array.from(bytes, (value) => value.toString(16).padStart(2, '0')).join('')}`;
}

export async function signOut(): Promise<void> {
	const response = await boundedFetch('/api/v1/auth/logout', {
		method: 'POST',
		credentials: 'include',
		headers: csrfHeaders()
	});
	if (!response.ok && response.status !== 401)
		throw new Error(
			'Sign-out was not confirmed. Your account may still be open. Try again before leaving this device.'
		);
	clearPrivateBrowserData();
	location.assign('/signin?signed_out=1');
}
