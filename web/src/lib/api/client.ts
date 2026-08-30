import createClient from 'openapi-fetch';
import type { paths } from './generated/schema';

export const api = createClient<paths>({
	baseUrl: '/api/v1',
	credentials: 'include'
});

export function csrfToken(): string {
	if (typeof document === 'undefined') return '';
	return (
		document.cookie
			.split('; ')
			.find((cookie) => cookie.startsWith('kredit_csrf='))
			?.split('=')[1] ?? ''
	);
}

export function csrfHeaders(): HeadersInit {
	const token = csrfToken();
	return token ? { 'X-CSRF-Token': token } : {};
}

export function idempotencyKey(): string {
	if (typeof crypto === 'undefined') throw new Error('Secure browser randomness is unavailable.');
	if (typeof crypto.randomUUID === 'function') return crypto.randomUUID();
	const bytes = crypto.getRandomValues(new Uint8Array(16));
	return `idem-${Array.from(bytes, (value) => value.toString(16).padStart(2, '0')).join('')}`;
}

export async function signOut(): Promise<void> {
	await fetch('/api/v1/auth/logout', {
		method: 'POST',
		credentials: 'include',
		headers: csrfHeaders()
	});
	location.assign('/app');
}
