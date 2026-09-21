import { expect, test } from '@playwright/test';

test('web liveness responds without a session and cannot be cached', async ({ request }) => {
	const response = await request.get('/healthz', { maxRedirects: 0 });
	expect(response.status()).toBe(200);
	expect(await response.text()).toBe('ok');
	expect(response.headers()['cache-control']).toContain('no-store');
	expect(response.headers()['content-type']).toContain('text/plain');
	expect(response.headers()['set-cookie']).toBeUndefined();
});
