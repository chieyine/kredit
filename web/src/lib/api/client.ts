import { boundedFetch, clearPrivateBrowserData } from './reliable';

export class ApiReadError extends Error {
	constructor(
		message: string,
		public readonly status = 0
	) {
		super(message);
		this.name = 'ApiReadError';
	}
}

// Safe reads may retry brief network/5xx failures. Financial mutations must not
// use this helper: writes rely on their explicit Idempotency-Key contract so a
// timeout can never silently become a duplicate action.
export async function readJSON<T>(
	url: string,
	options: { signal?: AbortSignal; attempts?: number } = {}
): Promise<T> {
	const attempts = Math.max(1, Math.min(options.attempts ?? 2, 3));
	let lastError: unknown;
	for (let attempt = 1; attempt <= attempts; attempt += 1) {
		try {
            const deadline = AbortSignal.timeout(20_000);
            const signal = options.signal ? AbortSignal.any([options.signal, deadline]) : deadline;
			const response = await fetch(url, {
				method: 'GET',
				credentials: 'include',
				signal,
                cache: 'no-store'
			});
			if (response.ok) return (await response.json()) as T;
			const body = await response.json().catch(() => ({}));
			const message = body.detail ?? body.title ?? `Request failed (${response.status}).`;
			if (response.status < 500 || attempt === attempts) throw new ApiReadError(message, response.status);
			lastError = new ApiReadError(message, response.status);
		} catch (error) {
			if (options.signal?.aborted) throw error;
			lastError = error;
			if (error instanceof ApiReadError && error.status > 0 && error.status < 500) throw error;
			if (attempt === attempts) throw error;
		}
		await new Promise((resolve) => setTimeout(resolve, 250 * attempt));
	}
	throw lastError instanceof Error ? lastError : new ApiReadError('The request could not be completed.');
}

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
	const response = await boundedFetch('/api/v1/auth/logout', {
		method: 'POST',
		credentials: 'include',
		headers: csrfHeaders()
	});
	if (!response.ok && response.status !== 401) throw new Error('Sign-out was not confirmed. Your account may still be open. Try again before leaving this device.');
	clearPrivateBrowserData();
	location.assign('/app?signed_out=1');
}
