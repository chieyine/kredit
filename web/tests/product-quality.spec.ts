import { expect, test } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const publicRoutes = [
	'/', '/demo', '/how-it-works', '/for-suppliers', '/for-buyers', '/pricing', '/security',
	'/faq', '/glossary', '/blog', '/legal/complaints'
];

test('public navigation is clear, complete and closes after a mobile choice', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/');
	const menu = page.locator('.site-menu-disclosure');
	await menu.locator('summary').click();
	await expect(menu).toHaveAttribute('open', '');
	await menu.getByRole('link', { name: 'For sellers' }).click();
	await expect(page).toHaveURL(/\/for-suppliers$/);
	await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
	await expect(page.locator('.site-menu-disclosure')).not.toHaveAttribute('open', '');
	const footer = page.locator('footer.site-footer');
	await expect(footer.getByRole('link', { name: 'For customers' })).toBeVisible();
	await expect(footer.getByRole('link', { name: 'How we keep it safe' })).toBeVisible();
});

test('homepage distinguishes sample records and explains confirmed payments', async ({ page }) => {
 await page.goto('/');
 await expect(page.locator('.hero-product')).toContainText('Example sale');
 await expect(page.locator('.hero-product')).toContainText('not a real account');
 await expect(page.locator('.hero-product [aria-hidden="true"] a')).toHaveCount(0);
 await page.getByRole('tab', { name: /The money/ }).click();
 await expect(page.getByRole('tabpanel')).toContainText('Confirmed payments reduce the balance');
 await expect(page.getByRole('tabpanel').getByRole('link', { name: /Explore the payment record/ })).toHaveAttribute('href', '/demo');
});

test('both sale-creation entry points preserve authentication and the intended destination', async ({ request }) => {
 for (const path of ['/app/credit/quick?customer=u1&goods=Rice&amount=100000','/app/credit/new?advanced=1']) {
  const response=await request.get(path,{maxRedirects:0});
  expect(response.status()).toBe(303);
  const location=new URL(response.headers().location,'http://127.0.0.1:5173');
  expect(location.pathname).toBe('/app');
  expect(location.searchParams.get('next')).toBe(path);
  expect(response.headers()['cache-control']).toContain('no-store');
 }
});

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
	test.setTimeout(300_000);
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
	// Account areas are never indexable, whatever else is true.
	for (const path of ['/app/overview', '/buyer', '/admin', '/recover']) {
		const response = await page.goto(path);
		await expect(page.locator('meta[name="robots"]'), path).toHaveAttribute('content', 'noindex,nofollow');
		expect(response?.headers()['cache-control'], path).toContain('no-store');
	}

	// The legal documents are different: they become public once the published
	// versions are approved and in effect, because people and regulators have to
	// be able to find them. What must never happen is the two halves disagreeing
	// — a page inviting crawlers that robots.txt shuts out, or the reverse. So
	// this asserts they agree, in whichever state the deployment is in.
	const robotsBody = await (await request.get('/robots.txt')).text();
	for (const path of ['/legal/privacy', '/legal/terms']) {
		await page.goto(path);
		const blockedByRobots = robotsBody.includes(`Disallow: ${path}`);
		await expect(page.locator('meta[name="robots"]'), path).toHaveAttribute(
			'content',
			blockedByRobots ? 'noindex,nofollow' : /^index,follow/
		);
	}
	await page.goto('/this-page-does-not-exist');
	await expect(page.getByRole('heading', { name: 'This page is not here' })).toBeVisible();
	await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'noindex,nofollow');

	const sitemap = await (await request.get('/sitemap.xml')).text();
	for (const path of publicRoutes) expect(sitemap, `sitemap ${path}`).toContain(`<loc>https://kredit.com.ng${path}</loc>`);
	for (const path of ['/app/', '/buyer/', '/admin/', '/recover']) expect(sitemap).not.toContain(`<loc>https://kredit.com.ng${path}`);
	const robots = await (await request.get('/robots.txt')).text();
	for (const path of ['/app', '/buyer', '/admin', '/recover']) expect(robots).toContain(`Disallow: ${path}`);
	// The third place publication state shows up. A legal document that robots.txt
	// shuts out must not be advertised in the sitemap, and one that is published
	// must be — the meta tag, robots.txt and the sitemap are one decision, and a
	// deployment where they disagree is the failure worth catching.
	for (const path of ['/legal/privacy', '/legal/terms']) {
		const blockedByRobots = robots.includes(`Disallow: ${path}`);
		const listed = sitemap.includes(`<loc>https://kredit.com.ng${path}</loc>`);
		expect(listed, `${path} sitemap listing must match robots.txt`).toBe(!blockedByRobots);
	}

	const manifestResponse = await request.get('/manifest.webmanifest');
	expect(manifestResponse.ok()).toBe(true);
	const manifest = await manifestResponse.json();
	expect(manifest).toMatchObject({ id: '/', display: 'standalone', lang: 'en-NG' });
	for (const icon of ['/icon-192.png', '/icon-512.png', '/apple-touch-icon.png']) expect((await request.get(icon)).ok(), icon).toBe(true);
});
