import { expect, test } from '@playwright/test';

test('guide library exposes 100 researched topics with search and categories',async({page})=>{
	await page.goto('/blog');
	await expect(page.getByRole('heading',{name:'Find the answer you need.'})).toBeVisible();
	await expect(page.getByText('100 helpful guides')).toBeVisible();
	await page.getByLabel('Search guides').fill('fake bank alert');
	await expect(page.getByRole('link',{name:/protect your business from fake bank alerts/i})).toBeVisible();
	await page.getByLabel('Topic').selectOption('Industry guides');
	await page.getByLabel('Search guides').fill('');
	await expect(page.getByText('10 helpful guides')).toBeVisible();
});

test('long guide renders complete SEO, useful content and source links',async({page})=>{
	await page.goto('/blog/how-to-sell-goods-on-credit-in-nigeria');
	await expect(page).toHaveTitle('How to sell goods on credit in Nigeria');
	await expect(page.locator('meta[name="description"]')).toHaveAttribute('content',/simple Nigerian guide/i);
	await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href','https://kredit.com.ng/blog/how-to-sell-goods-on-credit-in-nigeria');
	await expect(page.getByText(/\d,[\d]{3} words/)).toBeVisible();
	await expect(page.getByRole('heading',{name:'What to do step by step'})).toBeVisible();
	await expect(page.getByRole('heading',{name:'Frequently asked questions'})).toBeVisible();
	await expect(page.getByRole('heading',{name:'Official sources used for this guide'})).toBeVisible();
	await expect(page.getByRole('complementary',{name:'Related guide'})).toHaveCount(2);
	await expect(page.getByRole('navigation',{name:'Useful Kredit pages'}).getByRole('link')).toHaveCount(4);
	await expect(page.getByRole('navigation',{name:'Breadcrumb'}).getByRole('link',{name:'Credit sales'})).toHaveAttribute('href','/blog/topic/credit-sales');
	await expect(page.locator('script[type="application/ld+json"]')).toHaveCount(5);
});

test('topic hubs give every article a crawlable route into its guide cluster',async({page,request})=>{
	await page.goto('/blog/topic/customer-checks');
	await expect(page).toHaveTitle('Customer check guides for Nigerian businesses — Kredit');
	await expect(page.getByRole('heading',{name:'Customer check guides'})).toBeVisible();
	await expect(page.locator('.topic-list').getByRole('link')).toHaveCount(10);
	await expect(page.getByRole('link',{name:/12 questions to ask before giving business credit/i})).toBeVisible();
	await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href','https://kredit.com.ng/blog/topic/customer-checks');
	const sitemap=await (await request.get('/sitemap.xml')).text();
	expect(sitemap).toContain('<loc>https://kredit.com.ng/blog/topic/customer-checks</loc>');
});

test('sitemap and RSS publish the guide library for discovery',async({request})=>{
	const sitemap=await request.get('/sitemap.xml');expect(sitemap.ok()).toBeTruthy();const sitemapText=await sitemap.text();
	expect((sitemapText.match(/<url>/g)??[]).length).toBeGreaterThanOrEqual(114);expect(sitemapText).toContain('/blog/how-to-sell-goods-on-credit-in-nigeria');
	const rss=await request.get('/blog/rss.xml');expect(rss.ok()).toBeTruthy();const rssText=await rss.text();expect(rssText).toContain('<rss version="2.0"');expect(rssText).toContain('rel="self"');expect(rssText).toContain('<lastBuildDate>');
});

test('privacy notice gives a complete, readable account of information use and rights', async ({ page }) => {
	await page.goto('/legal/privacy');
	await expect(page).toHaveTitle('Privacy notice — Kredit');
	await expect(page.getByRole('heading', { name: 'Your information belongs to you.' })).toBeVisible();
	await expect(page.getByText('Complete pre-launch draft — legal approval pending')).toBeVisible();
	await expect(page.getByRole('heading', { name: '2. Information we collect' })).toBeVisible();
	await expect(page.getByRole('heading', { name: '6. Your rights and choices' })).toBeVisible();
	await expect(page.getByRole('heading', { name: '7. How we protect information' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Nigeria Data Protection Commission' })).toBeVisible();
	await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'noindex,nofollow');
});

test('terms explain the complete sale, payment and complaint journey', async ({ page }) => {
	await page.goto('/legal/terms');
	await expect(page).toHaveTitle('Terms of service — Kredit');
	await expect(page.getByRole('heading', { name: 'Clear rules for using Kredit.' })).toBeVisible();
	await expect(page.getByRole('heading', { name: '3. Making a credit sale' })).toBeVisible();
	await expect(page.getByRole('heading', { name: '5. Payments, balances and Kredit fees' })).toBeVisible();
	await expect(page.getByRole('heading', { name: '6. Bank-debit permission and late payment' })).toBeVisible();
	await expect(page.getByRole('heading', { name: '10. Help, complaints and regulators' })).toBeVisible();
	await expect(page.getByText('Kredit is not a bank, wallet, credit bureau, insurance company, marketplace or debt buyer.')).toBeVisible();
	await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'noindex,nofollow');
});

test('detailed legal content stays readable on a small phone', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	for (const path of ['/legal/privacy', '/legal/terms']) {
		await page.goto(path);
		const sizes = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, screen: window.innerWidth }));
		expect(sizes.page, `${path} must not scroll sideways`).toBeLessThanOrEqual(sizes.screen);
		await expect(page.getByRole('navigation', { name: /contents/i })).toBeVisible();
		await expect(page.locator('.document-actions')).toBeVisible();
	}
});

test('approved production details activate both legal documents', async ({ page }) => {
	const environment = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env;
	test.skip(environment?.TEST_ACTIVE_LEGAL !== '1', 'runs against the active-legal configuration');
	for (const path of ['/legal/privacy', '/legal/terms']) {
		await page.goto(path);
		await expect(page.getByText('Kredit Launch Test Limited').first()).toBeVisible();
		await expect(page.getByText(/Effective 1 September 2026/).first()).toBeVisible();
		await expect(page.getByText('Complete pre-launch draft — legal approval pending')).toHaveCount(0);
		await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'index,follow,max-image-preview:large,max-snippet:-1');
	}
	const sitemap = await (await page.request.get('/sitemap.xml')).text();
	expect(sitemap).toContain('<loc>https://kredit.com.ng/legal/privacy</loc>');
	expect(sitemap).toContain('<loc>https://kredit.com.ng/legal/terms</loc>');
	const robots = await (await page.request.get('/robots.txt')).text();
	expect(robots).not.toContain('Disallow: /legal/privacy');
	expect(robots).not.toContain('Disallow: /legal/terms');
});
