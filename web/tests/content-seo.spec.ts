import { expect, test } from '@playwright/test';

test('guide library exposes individually written guides with search and categories',async({page})=>{
	await page.goto('/blog');
	await expect(page.getByRole('heading',{name:'What do you need to know?'})).toBeVisible();
	await expect(page.getByText('12 helpful guides')).toBeVisible();
	await page.getByLabel('Search guides').fill('fake bank alert');
	await expect(page.getByRole('link',{name:/protect your business from fake bank alerts/i})).toBeVisible();
	await page.getByLabel('Topic').selectOption('Industry guides');
	await page.getByLabel('Search guides').fill('');
	await expect(page.getByText('1 helpful guide')).toBeVisible();
});

test('guide renders useful content without invented publication or research claims',async({page})=>{
	await page.goto('/blog/how-to-sell-goods-on-credit-in-nigeria');
	await expect(page).toHaveTitle('How to sell goods on credit in Nigeria');
	await expect(page.locator('meta[name="description"]')).toHaveAttribute('content',/Before goods leave your shop/i);
	await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href','https://kredit.com.ng/blog/how-to-sell-goods-on-credit-in-nigeria');
	await expect(page.locator('meta[property="article:published_time"]')).toHaveCount(0);
	await expect(page.locator('.guide > section').first().getByRole('heading')).toBeVisible();
	await expect(page.getByRole('heading',{name:'Frequently asked questions'})).toBeVisible();
	await expect(page.getByRole('heading',{name:'Official sources used for this guide'})).toHaveCount(0);
	await expect(page.getByRole('complementary',{name:'Related guide'})).toHaveCount(2);
	await expect(page.getByRole('navigation',{name:'Useful Kredit pages'}).getByRole('link')).toHaveCount(4);
	await expect(page.getByRole('navigation',{name:'Breadcrumb'}).getByRole('link',{name:'Credit sales'})).toHaveAttribute('href','/blog/topic/credit-sales');
	await expect(page.locator('script[type="application/ld+json"]')).toHaveCount(5);
});

test('topic hubs give every article a crawlable route into its guide cluster',async({page,request})=>{
	await page.goto('/blog/topic/customer-checks');
	await expect(page).toHaveTitle('Customer check guides for Nigerian businesses — Kredit');
	await expect(page.getByRole('heading',{name:'Customer check guides'})).toBeVisible();
	await expect(page.locator('.topic-list').getByRole('link')).toHaveCount(1);
	await expect(page.getByRole('link',{name:/12 questions to ask before giving business credit/i})).toBeVisible();
	await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href','https://kredit.com.ng/blog/topic/customer-checks');
	const sitemap=await (await request.get('/sitemap.xml')).text();
	expect(sitemap).toContain('<loc>https://kredit.com.ng/blog/topic/customer-checks</loc>');
});

test('sitemap and RSS publish the guide library for discovery',async({request})=>{
	const sitemap=await request.get('/sitemap.xml');expect(sitemap.ok()).toBeTruthy();const sitemapText=await sitemap.text();
	expect(sitemapText).not.toContain('/blog/cash-sale-vs-credit-sale');expect(sitemapText).toContain('/blog/how-to-sell-goods-on-credit-in-nigeria');
	const rss=await request.get('/blog/rss.xml');expect(rss.ok()).toBeTruthy();const rssText=await rss.text();expect(rssText).toContain('<rss version="2.0"');expect(rssText).toContain('rel="self"');expect(rssText).toContain('<lastBuildDate>');expect(rssText).not.toContain('<pubDate>');
});

test('privacy notice gives a complete, readable account of information use and rights', async ({ page }) => {
	await page.goto('/legal/privacy');
	await expect(page).toHaveTitle('Privacy notice — Kredit');
	await expect(page.getByRole('heading', { name: 'Privacy notice', exact: true })).toBeVisible();
	await expect(page.getByText('Complete pre-launch draft — legal approval pending')).toHaveCount(0);
	await expect(page.getByRole('heading', { name: 'What we keep about you' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Your rights and choices' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'How we protect information' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Nigeria Data Protection Commission' })).toBeVisible();
	await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'index,follow,max-image-preview:large,max-snippet:-1');
});

test('terms explain the complete sale, payment and complaint journey', async ({ page }) => {
	await page.goto('/legal/terms');
	await expect(page).toHaveTitle('Terms of service — Kredit');
	await expect(page.getByRole('heading', { name: 'Terms of service', exact: true })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Making a credit sale' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Payments, balances and Kredit fees' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Bank-debit permission and late payment' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Help, complaints and regulators' })).toBeVisible();
	await expect(page.getByText('Kredit is not a bank, wallet, credit bureau, insurance company, marketplace or debt buyer.')).toBeVisible();
	await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'index,follow,max-image-preview:large,max-snippet:-1');
});

test('detailed legal content stays readable on a small phone', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	for (const path of ['/legal/privacy', '/legal/terms']) {
		await page.goto(path);
		const sizes = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, screen: window.innerWidth }));
		expect(sizes.page, `${path} must not scroll sideways`).toBeLessThanOrEqual(sizes.screen);
		await page.locator('.mobile-contents summary').click();
 await expect(page.getByRole('navigation', { name: 'Document sections' })).toBeVisible();
		await expect(page.locator('.document-actions')).toBeVisible();
	}
});

test('approved production details activate both legal documents', async ({ page }) => {
	for (const path of ['/legal/privacy', '/legal/terms']) {
		await page.goto(path);
		await expect(page.getByText('KREDIT TECHNOLOGIES LIMITED').first()).toBeVisible();
		await expect(page.getByText(/Effective 7 September 2026/).first()).toBeVisible();
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

test('FAQ matches activation fees and conditional collections', async ({ page }) => {
 await page.goto('/faq');
 await expect(page.getByText(/seller owes the agreed base fee when the accepted sale becomes active/)).toBeVisible();
 await expect(page.getByText(/A debit can fail; repayment is not guaranteed/)).toBeVisible();
 await expect(page.getByRole('link', { name: 'what it costs', exact: true })).toHaveAttribute('href', '/pricing');
});
