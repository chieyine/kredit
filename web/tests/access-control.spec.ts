import { expect, test } from '@playwright/test';

test('logged-out visitors are redirected before protected account pages render', async ({ page }) => {
	await page.goto('/app/payments');
	await expect(page).toHaveURL(/\/app\?next=%2Fapp%2Fpayments$/);
	await expect(page.getByRole('heading', { name: 'Welcome back.' })).toBeVisible();
	await expect(page.getByRole('navigation', { name: 'Main navigation' })).toBeVisible();
	await expect(page.locator('footer.site-footer')).toBeVisible();
	await expect(page.locator('footer.site-footer').getByText('How it works')).toBeVisible();
	await expect(page.getByRole('navigation', { name: 'Seller account', exact: true })).toHaveCount(0);
	await expect(page.getByRole('heading', { name: 'Your money, clearly.' })).toHaveCount(0);
});

test('every account area requires a session while private-token pages remain public', async ({ request }) => {
	for (const path of ['/app/overview', '/app/payments', '/buyer', '/admin']) {
		const response = await request.get(path, { maxRedirects: 0 });
		expect(response.status(), path).toBe(303);
		expect(response.headers().location, path).toContain('/app?next=');
	}
	const removedSupplierRoute = await request.get('/supplier/organizations/example/reports', { maxRedirects: 0 });
	expect(removedSupplierRoute.status()).toBe(404);
	for (const path of ['/pay/example', '/receipt/example', '/secure?token=example', '/recover', '/buyer-invitations/example']) {
		const response = await request.get(path, { maxRedirects: 0 });
		expect(response.status(), path).toBe(200);
	}
	const shortInvitation = await request.get('/c/example', { maxRedirects: 0 });
	expect(shortInvitation.status()).toBe(307);
	expect(shortInvitation.headers().location).toBe('/buyer-invitations/example');
});

test('an invalid saved session shows only the account check before sign-in', async ({ page, context, baseURL }) => {
	await context.addCookies([{ name: 'kredit_session', value: 'expired-session', url: baseURL ?? 'http://127.0.0.1:5173' }]);
	await page.route('**/api/v1/me', async (route) => {
		await new Promise((resolve) => setTimeout(resolve, 500));
		await route.fulfill({ status: 401, contentType: 'application/problem+json', body: JSON.stringify({ detail: 'Authentication required.' }) });
	});
	await page.goto('/app/payments');
	await expect(page.getByRole('heading', { name: 'Checking your account…' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Your money, clearly.' })).toHaveCount(0);
	await expect(page.getByRole('navigation', { name: 'Seller account', exact: true })).toHaveCount(0);
	await expect(page).toHaveURL(/\/app\?next=%2Fapp%2Fpayments$/);
});

test('an expired session cannot flash the next page during an account navigation', async ({ page, context, baseURL }) => {
	await context.addCookies([{ name: 'kredit_session', value: 'expires-during-use', url: baseURL ?? 'http://127.0.0.1:5173' }]);
	let checks = 0;
	await page.route('**/api/v1/me', async (route) => {
		checks++;
		if (checks === 1) {
			await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ user: { id: 'seller-1' }, session: { id: 'session-1' }, organizations: [] }) });
			return;
		}
		await new Promise((resolve) => setTimeout(resolve, 500));
		await route.fulfill({ status: 401, contentType: 'application/problem+json', body: JSON.stringify({ detail: 'Session expired.' }) });
	});
	await page.goto('/app/settings');
	await expect(page.getByRole('navigation', { name: 'Seller account', exact: true })).toBeVisible();
	await expect(page.getByRole('navigation', { name: 'Seller account', exact: true }).getByRole('button', { name: 'Menu', exact: true })).toBeVisible();
	await page.getByRole('link', { name: 'Payments', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'Your money, clearly.' })).toHaveCount(0);
	await expect(page).toHaveURL(/\/app\?next=%2Fapp%2Fpayments$/);
	await expect(page.getByRole('heading', { name: 'Welcome back.' })).toBeVisible();
	await expect(page.locator('footer.site-footer')).toBeVisible();
});

test('the secure payment link is public but never shows the seller account', async ({ page }) => {
	await page.route('**/api/v1/public/payment-intents/example', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ payment_intent: { supplier_name: 'Adebayo Supplies', description: 'Twenty bags of rice', amount_kobo: 25000000, payment_status: 'ready', provider_action: 'Continue to your approved payment provider.' } }) }));
	await page.goto('/pay/example');
	await expect(page.getByRole('heading', { name: 'Check the amount before you pay.' })).toBeVisible();
	await expect(page.getByText('₦250,000.00')).toBeVisible();
	await expect(page.getByRole('navigation', { name: 'Seller account', exact: true })).toHaveCount(0);
});

test('the signed-in payments page prioritizes money and items needing an answer', async ({ page, context, baseURL }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await context.addCookies([{ name: 'kredit_session', value: 'payments-session', url: baseURL ?? 'http://127.0.0.1:5173' }]);
	await page.route('**/api/v1/me', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ user: { id: 'seller-1' }, session: { id: 'session-1' }, organizations: [] }) }));
	await page.route('**/api/v1/organizations', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ organizations: [{ id: 'org-1', legal_name: 'Adebayo Supplies' }] }) }));
	await page.route('**/api/v1/organizations/org-1/payments', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ payments: [{ id: 'sale-1', buyer_legal_name: 'Kano Retail', description: 'Twenty bags of rice', amount_kobo: 40000000, source_type: 'integrated_voluntary', state: 'recognized', paid_at: '2026-08-29T09:00:00Z', reference: 'PAY-100' }] }) }));
	await page.route('**/api/v1/organizations/org-1/payment-claims', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ payment_claims: [{ id: 'claim-1', amount_kobo: 10000000, transfer_reference: 'TRF-100', state: 'pending', paid_at: '2026-08-30T09:00:00Z', hold_expires_at: '2026-08-31T09:00:00Z' }] }) }));
	await page.goto('/app/payments');
	await expect(page.getByRole('heading', { name: 'Your money, clearly.' })).toBeVisible();
	await expect(page.getByRole('navigation', { name: 'Seller account', exact: true })).toBeVisible();
	const headerMenu = page.getByRole('navigation', { name: 'Seller account', exact: true }).getByRole('button', { name: 'Menu', exact: true });
	await expect(headerMenu).toBeVisible();
	const bottomNavigation = page.getByLabel('Seller account main pages');
	await expect(bottomNavigation.getByRole('link')).toHaveCount(4);
	await expect(bottomNavigation.getByRole('button')).toHaveCount(0);
	await expect(page.locator('footer.site-footer')).toHaveCount(0);
	await expect(page.getByLabel('Payment summary').getByText('₦400,000.00')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Yes, I got the money' })).toBeVisible();
	await expect(page.getByText('Kano Retail')).toBeVisible();
	await headerMenu.click();
	const moreMenu = page.getByRole('dialog', { name: 'Seller account menu' });
	await expect(moreMenu).toBeVisible();
	await expect(moreMenu.getByRole('navigation', { name: 'Account menu pages' }).getByRole('link')).toHaveCount(9);
	await expect(moreMenu.getByText('Sales and money', { exact: true })).toBeVisible();
	await expect(moreMenu.getByText('Account and help', { exact: true })).toBeVisible();
	await expect(moreMenu.getByRole('link', { name: /Settings/ })).toBeVisible();
	await expect(moreMenu.getByRole('button', { name: 'Find a page' })).toHaveCount(0);
});
