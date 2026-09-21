import type { RequestHandler } from './$types';

// Node-process liveness only. API/database readiness is checked separately.
export const prerender = false;

export const GET: RequestHandler = () => new Response('ok', {
	status: 200,
	headers: {
		'content-type': 'text/plain; charset=utf-8',
		'cache-control': 'no-store'
	}
});
