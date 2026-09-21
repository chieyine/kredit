const PRIVATE_PAGE = /^\/(?:account|start|signin|workspace|personal|admin|agents|c|pay|receipt|secure|recover|buyer-invitations)(?:\/|$)/;
const PUBLIC_CACHE = 'public, max-age=0, s-maxage=300, stale-while-revalidate=86400';
const TARGETED_CACHE_HEADERS = [
	'cdn-cache-control',
	'vercel-cdn-cache-control',
	'cloudflare-cdn-cache-control',
	'surrogate-control'
] as const;

// Apply after resolving the page: an individual route must not make a private
// response cacheable. Targeted CDN directives can take precedence over the
// ordinary Cache-Control header, so replace those directives as well.
// Infrastructure must still respect origin headers; this cannot purge old data
// or override an administrator's force-cache rule.
export function applyPageCachePolicy(
	pathname: string,
	response: Response,
	method: string,
	hasSessionCookie = false
): void {
	const privateResponse = PRIVATE_PAGE.test(pathname)
		|| hasSessionCookie
		|| response.status >= 400
		|| (method !== 'GET' && method !== 'HEAD')
		|| response.headers.has('set-cookie');
	if (privateResponse) {
		response.headers.set('cache-control', 'private, no-store');
		for (const name of TARGETED_CACHE_HEADERS) response.headers.set(name, 'no-store');
		response.headers.delete('expires');
		return;
	}
	if (!response.headers.has('cache-control')) response.headers.set('cache-control', PUBLIC_CACHE);
}
