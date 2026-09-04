import { env } from '$env/dynamic/private';
import type { Handle } from '@sveltejs/kit';
import { assertLaunchWebConfig } from '$lib/server/legal-config';

assertLaunchWebConfig();

// Keep browser cookies first-party while the API remains an internal service.
// The same-origin proxy is used by the production Node adapter; Vite's proxy
// serves the equivalent role during local development.

// Applied to every HTML response the Node adapter serves; the Go API sets the
// equivalent headers for /api traffic.
const SECURITY_HEADERS: Record<string, string> = {
	'x-content-type-options': 'nosniff',
	'x-frame-options': 'DENY',
	'referrer-policy': 'no-referrer',
	'permissions-policy': 'camera=(), microphone=(), geolocation=()',
	'cross-origin-opener-policy': 'same-origin',
	'cross-origin-resource-policy': 'same-origin',
	// SvelteKit hydration data is embedded as an inline module script, so
	// 'unsafe-inline' is required for script/style until nonce-based CSP lands.
	'content-security-policy':
		"default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'",
	// Browsers ignore HSTS over plain HTTP, so it is safe to always send.
	'strict-transport-security': 'max-age=31536000; includeSubDomains'
};

export const handle: Handle = async ({ event, resolve }) => {
	if (!event.url.pathname.startsWith('/api/')) {
		const protectedAccountRoute = /^(?:\/app\/.+|\/buyer(?:\/|$)|\/admin(?:\/|$))/.test(event.url.pathname);
		if (protectedAccountRoute && !event.cookies.get('kredit_session')) {
			const next = `${event.url.pathname}${event.url.search}`;
			return new Response(null, {
				status: 303,
				headers: {
					location: `/app?next=${encodeURIComponent(next)}`,
					'cache-control': 'private, no-store'
				}
			});
		}
		const response = await resolve(event);
		for (const [name, value] of Object.entries(SECURITY_HEADERS)) {
			response.headers.set(name, value);
		}
		const privateRoute = /^\/(app|buyer|admin|c|pay|receipt|secure|recover|buyer-invitations)(\/|$)/.test(event.url.pathname);
		response.headers.set('cache-control', privateRoute
			? 'private, no-store'
			: 'public, max-age=0, s-maxage=300, stale-while-revalidate=86400');
		return response;
	}
	const upstream = env.API_INTERNAL_URL ?? 'http://localhost:8080';
	const target = new URL(event.url.pathname + event.url.search, upstream);
	const headers = new Headers(event.request.headers);
	headers.delete('host');
	headers.delete('connection');
	// The Go API trusts forwarded-client-address headers when the peer is the
	// private proxy address, which this hop is. Anything the browser supplied
	// must therefore be discarded and replaced with the address this server
	// actually observed, or a caller could rotate its rate-limit and OTP-throttle
	// identity at will by setting the header itself.
	for (const spoofable of ['x-forwarded-for', 'x-real-ip', 'cf-connecting-ip', 'true-client-ip', 'forwarded']) {
		headers.delete(spoofable);
	}
	try {
		const clientAddress = event.getClientAddress();
		if (clientAddress) headers.set('x-forwarded-for', clientAddress);
	} catch {
		// No observable client address (some adapters during prerender); leave the
		// header absent so the API falls back to the connection address.
	}
	const method = event.request.method;
	try {
		let body: Uint8Array<ArrayBuffer> | undefined;
		if (method !== 'GET' && method !== 'HEAD' && event.request.body) {
			const reader = event.request.body.getReader();
			const chunks: Uint8Array[] = [];
			let size = 0;
			while (true) {
				const { value, done } = await reader.read();
				if (done) break;
				size += value.byteLength;
				if (size > (event.url.pathname.endsWith('/documents') ? 3 : 2) * 1024 * 1024) {
					await reader.cancel();
					return new Response(JSON.stringify({ title: 'Request too large', status: 413 }), {
						status: 413, headers: { 'content-type': 'application/problem+json', 'cache-control': 'no-store' }
					});
				}
				chunks.push(value);
			}
			body = new Uint8Array(size);
			let offset = 0;
			for (const chunk of chunks) { body.set(chunk, offset); offset += chunk.byteLength; }
		}
		return await fetch(target, { method, headers, body, redirect: 'manual', signal: AbortSignal.timeout(30_000) });
	} catch {
		return new Response(JSON.stringify({ type: 'about:blank', title: 'Service unavailable', status: 503, detail: 'The API is temporarily unavailable.' }), {
			status: 503,
			headers: { 'content-type': 'application/problem+json', 'cache-control': 'no-store' }
		});
	}
};
