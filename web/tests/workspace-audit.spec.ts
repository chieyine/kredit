import { expect, test } from '@playwright/test';

const organizations = [{ id: 'org-a', legal_name: 'First business' }, { id: 'org-b', legal_name: 'Second business' }];
test.beforeEach(async ({ page, context, baseURL }) => {
	await context.addCookies([{ name: 'kredit_session', value: 'solo-audit', url: baseURL! }]);
	await page.route('**/api/v1/me', route => route.fulfill({ json: { user: { id: 'user-a', status: 'active' }, session: { authentication_level: 'AAL1' }, organizations } }));
});

test('failed business lookup shows an error and can be retried', async ({ page }) => {
	let attempts = 0;
	await page.route('**/api/v1/organizations', route => {
		attempts++;
		return route.fulfill(attempts === 1 ? { status: 503, json: {} } : { json: { organizations } });
	});
	await page.route('**/api/v1/organizations/*/customers', route => route.fulfill({ json: { customers: [{ id: 'buyer-1', legal_name: 'Ada Stores', outstanding_kobo: 123456 }] } }));
	await page.goto('/app/customers');
	await expect(page.getByRole('alert')).toContainText('We could not open this page');
	await expect(page.getByRole('heading', { name: 'No customers yet' })).toHaveCount(0);
	await page.getByRole('button', { name: 'Try again', exact: true }).click();
	await expect(page.locator('.records')).toContainText('Ada Stores');
	await expect(page.locator('.records')).toContainText('₦1,234.56');
});

test('switching business resets pagination and ignores a late response', async ({ page }) => {
	await page.route('**/api/v1/organizations', route => route.fulfill({ json: { organizations } }));
	await page.route('**/api/v1/organizations/org-a/collections', route => route.fulfill({ json: { collections: Array.from({ length: 10 }, (_, i) => ({ id: `sale-${i}`, buyer_legal_name: `First buyer ${i}`, amount_kobo: 10000 })) } }));
	await page.route('**/api/v1/organizations/org-b/collections', async route => {
		await new Promise(resolve => setTimeout(resolve, 250));
		await route.fulfill({ json: { collections: [{ id: 'other-sale', buyer_legal_name: 'Second buyer', amount_kobo: 20000 }] } });
	});
	await page.goto('/app/collections');
	await page.getByRole('button', { name: 'Next', exact: true }).click();
	await expect(page.locator('.records')).toContainText('First buyer 9');
	await page.getByRole('combobox', { name: 'Business', exact: true }).selectOption('org-b');
	await expect(page.locator('.records')).toContainText('Second buyer');
	await expect(page.locator('.records a')).toHaveAttribute('href', '/app/credit/other-sale?organization=org-b');
	await page.getByRole('combobox', { name: 'Business', exact: true }).selectOption('org-a');
	await expect(page.locator('.records')).toContainText('First buyer 0');
	await page.getByRole('combobox', { name: 'Business', exact: true }).selectOption('org-b');
	await page.getByRole('combobox', { name: 'Business', exact: true }).selectOption('org-a');
	await expect(page.locator('.records')).toContainText('First buyer 0');
	await expect(page.locator('.records')).not.toContainText('Second buyer');
});

test('money owed excludes unaccepted sales without an obligation', async ({ page }) => {
	await page.route('**/api/v1/buyer/credit-requests', route => route.fulfill({ json: { requests: [
		{ request: { id: 'draft', goods_description: 'Pending goods', buyer_legal_name: 'Pending sale' } },
		{ request: { id: 'accepted', buyer_legal_name: 'Accepted sale' }, obligation: { id: 'debt', outstanding_kobo: 75000 } }
	] } }));
	await page.goto('/buyer/obligations');
	await expect(page.locator('.records article')).toHaveCount(1);
	await expect(page.locator('.records')).toContainText('₦750.00');
	await expect(page.locator('.records a')).toHaveAttribute('href', '/buyer/obligations/debt');
});

test('desktop navigation works before JavaScript loads', async ({ browser, baseURL }) => {
	const context = await browser.newContext({ javaScriptEnabled: false, viewport: { width: 1440, height: 900 } });
	const page = await context.newPage();
	await page.goto(baseURL!);
	await expect(page.getByRole('navigation', { name: 'Main navigation' }).getByRole('link', { name: 'For sellers', exact: true })).toBeVisible();
	await page.getByRole('navigation', { name: 'Main navigation' }).getByRole('link', { name: 'For sellers', exact: true }).click();
	await expect(page).toHaveURL(/\/for-suppliers$/);
	await context.close();
});

test('customer history and repeat-sale link preserve the selected business', async ({ page }) => {
	const requested: string[] = [];
	await page.route('**/api/v1/organizations', route => route.fulfill({ json: { organizations } }));
	await page.route('**/api/v1/organizations/*/customers/customer-1/*', route => {
		requested.push(route.request().url());
		return route.fulfill({ json: route.request().url().endsWith('/history') ? { current_active_principal_kobo: 25000, active_obligations: 1, completed_obligations: 0 } : { obligations: [] } });
	});
	await page.goto('/app/customers/customer-1?organization=org-b');
	await expect(page.getByRole('link', { name: 'Make another sale' })).toHaveAttribute('href', '/app/credit/new?customer=customer-1&organization=org-b');
	expect(requested.length).toBe(2);
	expect(requested.every(url => url.includes('/organizations/org-b/'))).toBe(true);
});

test('buyer can recover from a dropped connection while opening an obligation', async ({ page }) => {
	let fail = true;
	await page.route('**/api/v1/buyer/obligations/debt-1', route => fail ? route.abort('failed') : route.fulfill({ json: {
		view: { request: { id: 'sale-1', goods_description: 'Delivered rice' }, obligation: { outstanding_kobo: 10000, payment_status: 'UNPAID' } },
		schedule_items: [], payments: [], payment_claims: []
	} }));
	await page.goto('/buyer/obligations/debt-1');
	await expect(page.getByRole('heading', { name: 'We could not open this sale.' })).toBeVisible();
	fail = false;
	await page.getByRole('button', { name: 'Try again', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'Delivered rice' })).toBeVisible();
});

test('opening a related guide updates the article and structured metadata', async ({ page }) => {
	await page.goto('/blog/how-to-sell-goods-on-credit-in-nigeria');
	const related = page.locator('.related a').first();
	const title = (await related.locator('strong').innerText()).trim();
	const href = await related.getAttribute('href');
	await related.click();
	await expect(page).toHaveURL(new RegExp(`${href}$`));
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(title);
	await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', `https://kredit.com.ng${href}`);
	const schemas = await page.locator('script[type="application/ld+json"]').allTextContents();
	expect(schemas.some(schema => schema.includes(title))).toBe(true);
});
