import { expect, test } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const publicRoutes = [
	'/', '/how-it-works', '/for-suppliers', '/for-buyers', '/pricing', '/security',
	'/faq', '/glossary', '/blog', '/blog/sell-on-credit-safely',
	'/blog/why-credit-agreements-fail', '/blog/collections-last-resort',
	'/blog/trade-credit-vs-loan', '/legal/complaints'
];

test('every indexable page has complete, unique search and social metadata', async ({ page }) => {
	test.setTimeout(180_000);
	const titles = new Set<string>();
	const descriptions = new Set<string>();
	for (const path of publicRoutes) {
		const response = await page.goto(path);
		expect(response?.status(), path).toBe(200);
		await expect(page.locator('h1'), `${path} h1`).toHaveCount(1);
		const title = await page.title();
		const description = await page.locator('meta[name="description"]').getAttribute('content');
		expect(title.length, `${path} title length`).toBeGreaterThan(20);
		expect(description?.length ?? 0, `${path} description length`).toBeGreaterThan(70);
		expect(titles.has(title), `${path} unique title`).toBe(false);
		expect(descriptions.has(description ?? ''), `${path} unique description`).toBe(false);
		titles.add(title); descriptions.add(description ?? '');
		await expect(page.locator('meta[name="description"]')).toHaveCount(1);
		await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', `https://kredit.com.ng${path}`);
		await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', /index,follow/);
		await expect(page.locator('meta[property="og:title"]')).toHaveAttribute('content', title);
		await expect(page.locator('meta[property="og:description"]')).toHaveAttribute('content', description ?? '');
		await expect(page.locator('meta[property="og:image"]')).toHaveAttribute('content', 'https://kredit.com.ng/og.png');
		await expect(page.locator('meta[name="twitter:title"]')).toHaveAttribute('content', title);
		const schemas = await page.locator('script[type="application/ld+json"]').allTextContents();
		expect(schemas.length, `${path} structured data`).toBeGreaterThanOrEqual(3);
		for (const schema of schemas) expect(() => JSON.parse(schema), `${path} valid structured data`).not.toThrow();
	}
});

test('responsive public pages avoid horizontal overflow and serious accessibility defects', async ({ page }) => {
	test.setTimeout(180_000);
	await page.setViewportSize({ width: 390, height: 844 });
	for (const path of publicRoutes) {
		await page.goto(path);
		const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
		expect(overflow, `${path} horizontal overflow`).toBeLessThanOrEqual(1);
		const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa', 'wcag22aa']).analyze();
		const blocking = results.violations.filter((violation) => violation.impact === 'serious' || violation.impact === 'critical');
		expect(blocking, `${path}: ${blocking.map((item) => item.id).join(', ')}`).toEqual([]);
	}
});

test('index boundaries, error recovery, sitemap and install assets are safe and complete', async ({ page, request }) => {
	for (const path of ['/app/overview', '/buyer', '/admin', '/recover', '/legal/privacy', '/legal/terms']) {
		const response = await page.goto(path);
		await expect(page.locator('meta[name="robots"]'), path).toHaveAttribute('content', 'noindex,nofollow');
		if (!path.startsWith('/legal/')) expect(response?.headers()['cache-control'], path).toContain('no-store');
	}
	await page.goto('/this-page-does-not-exist');
	await expect(page.getByRole('heading', { name: 'Page not found' })).toBeVisible();
	await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'noindex,nofollow');

	const sitemap = await (await request.get('/sitemap.xml')).text();
	for (const path of publicRoutes) expect(sitemap, `sitemap ${path}`).toContain(`<loc>https://kredit.com.ng${path}</loc>`);
	for (const path of ['/app/', '/buyer/', '/admin/', '/recover', '/legal/privacy', '/legal/terms']) expect(sitemap).not.toContain(`<loc>https://kredit.com.ng${path}`);
	const robots = await (await request.get('/robots.txt')).text();
	for (const path of ['/app/', '/buyer/', '/admin/', '/recover']) expect(robots).toContain(`Disallow: ${path}`);

	const manifestResponse = await request.get('/manifest.webmanifest');
	expect(manifestResponse.ok()).toBe(true);
	const manifest = await manifestResponse.json();
	expect(manifest).toMatchObject({ id: '/', display: 'standalone', lang: 'en-NG' });
	for (const icon of ['/icon-192.png', '/icon-512.png', '/apple-touch-icon.png']) expect((await request.get(icon)).ok(), icon).toBe(true);
});
