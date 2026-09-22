import { expect, test } from '@playwright/test';
import { applyPageCachePolicy, isAccountPage } from '../src/lib/server/page-cache';

const targetedHeaders = [
	'cdn-cache-control',
	'vercel-cdn-cache-control',
	'cloudflare-cdn-cache-control',
	'surrogate-control'
];

function cacheableResponse(status = 200): Response {
	const headers = new Headers({
		'cache-control': 'public, max-age=600, s-maxage=86400',
		expires: 'Thu, 01 Jan 2099 00:00:00 GMT',
		'content-type': 'text/plain',
		'x-content-type-options': 'nosniff'
	});
	for (const name of targetedHeaders) headers.set(name, 'public, max-age=86400');
	return new Response('synthetic response', { status, headers });
}

function expectPrivate(response: Response): void {
	expect(response.headers.get('cache-control')).toBe('private, no-store');
	for (const name of targetedHeaders) expect(response.headers.get(name), name).toBe('no-store');
	expect(response.headers.has('expires')).toBe(false);
}

test('private pages override conflicting browser and CDN cache directives', () => {
	for (const root of [
		'account',
		'start',
		'signin',
		'workspace',
		'personal',
		'admin',
		'agents',
		'c',
		'pay',
		'receipt',
		'secure',
		'recover',
		'buyer-invitations'
	]) {
		for (const path of [
			`/${root}`,
			`/${root}/synthetic`,
			`/%${root.charCodeAt(0).toString(16)}${root.slice(1)}/synthetic`
		]) {
			const response = cacheableResponse();
			applyPageCachePolicy(path, response, 'GET');
			expectPrivate(response);
		}
	}
});

test('account gate recognizes encoded paths without matching public prefix lookalikes', () => {
	for (const path of [
		'/workspace',
		'/workspace/today',
		'/%77orkspace/today',
		'/%61dmin/website',
		'/account/messages'
	]) {
		expect(isAccountPage(path), path).toBe(true);
	}
	for (const path of ['/workspace-guide', '/accounting', '/pricing', '/signin', '/%']) {
		expect(isAccountPage(path), path).toBe(false);
	}
});

test('errors, unsafe methods, session responses and malformed paths cannot be cached', () => {
	for (const status of [400, 404, 410, 429, 500, 503]) {
		const response = cacheableResponse(status);
		applyPageCachePolicy('/pricing', response, 'GET');
		expectPrivate(response);
		expect(response.status).toBe(status);
	}
	for (const method of ['POST', 'PUT', 'PATCH', 'DELETE']) {
		const response = cacheableResponse();
		applyPageCachePolicy('/contact', response, method);
		expectPrivate(response);
	}
	const session = cacheableResponse();
	applyPageCachePolicy('/pricing', session, 'GET', true);
	expectPrivate(session);
	const cookie = cacheableResponse();
	cookie.headers.set('set-cookie', 'synthetic=example; HttpOnly; Secure');
	applyPageCachePolicy('/', cookie, 'GET');
	expectPrivate(cookie);
	expect(cookie.headers.get('set-cookie')).toBe('synthetic=example; HttpOnly; Secure');
	const malformed = cacheableResponse();
	applyPageCachePolicy('/%invalid', malformed, 'GET');
	expectPrivate(malformed);
});

test('anonymous public reads retain caching and response contents are unchanged', async () => {
	for (const method of ['GET', 'HEAD']) {
		const response = cacheableResponse();
		applyPageCachePolicy('/pricing', response, method);
		expect(response.headers.get('cache-control')).toBe('public, max-age=600, s-maxage=86400');
		for (const name of targetedHeaders) expect(response.headers.get(name)).toBe('public, max-age=86400');
	}
	const defaults = new Response('public content');
	applyPageCachePolicy('/', defaults, 'GET');
	expect(defaults.headers.get('cache-control')).toBe('public, max-age=0, s-maxage=300, stale-while-revalidate=86400');
	const sensitive = cacheableResponse();
	applyPageCachePolicy('/workspace', sensitive, 'GET');
	expect(sensitive.status).toBe(200);
	expect(sensitive.headers.get('content-type')).toBe('text/plain');
	expect(sensitive.headers.get('x-content-type-options')).toBe('nosniff');
	expect(await sensitive.text()).toBe('synthetic response');
});

test('rendered sign-in and missing pages carry no-store directives', async ({ request }) => {
	for (const [path, status] of [
		['/signin', 200],
		['/audit-cache-page-does-not-exist', 404]
	] as const) {
		const response = await request.get(path);
		expect(response.status()).toBe(status);
		expect(response.headers()['cache-control']).toBe('private, no-store');
		for (const name of targetedHeaders) expect(response.headers()[name], name).toBe('no-store');
	}
});

test('encoded account URLs still require the sign-in navigation gate', async ({ request }) => {
	const response = await request.get('/%77orkspace/today', { maxRedirects: 0 });
	expect(response.status()).toBe(303);
	expect(response.headers().location).toMatch(/^\/signin\?next=/);
	expect(response.headers()['cache-control']).toBe('private, no-store');
});
