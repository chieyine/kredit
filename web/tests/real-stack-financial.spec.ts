import { expect, test, type Page } from '@playwright/test';

const realStack = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env?.KREDIT_REAL_STACK_E2E === '1';
test.describe('real-stack financial journeys', () => {
	test.skip(!realStack, 'requires the real Go API and PostgreSQL acceptance database');

	async function login(page: Page, identifier: string) {
		const challenge = await page.request.post('/api/v1/auth/otp/challenges', {
			data: { identifier, channel: 'email', purpose: 'login' }
		});
		expect(challenge.status()).toBe(202);
		const challengeBody = await challenge.json() as { challenge_id: string; development_code?: string };
		expect(challengeBody.challenge_id).toBeTruthy();
		expect(challengeBody.development_code).toMatch(/^\d{6}$/);

		const verified = await page.request.post('/api/v1/auth/otp/verify', {
			data: {
				challenge_id: challengeBody.challenge_id,
				code: challengeBody.development_code,
				device_label: 'phase3-playwright'
			}
		});
		expect(verified.status()).toBe(200);
		const me = await page.request.get('/api/v1/me');
		expect(me.status()).toBe(200);
		return me.json() as Promise<{ user: { id: string; email?: string }; organizations: Array<{ id: string; legal_name: string }> }>;
	}

	test('supplier browser reads the same real payment records as the API', async ({ page }) => {
		const me = await login(page, 'owner@abc-pharmaceuticals.test');
		expect(me.organizations.length).toBeGreaterThan(0);
		const organization = me.organizations[0];
		for (const [endpoint, key] of [['customers','customers'],['credit-requests','requests'],['collections','collections'],['overdue','overdue'],['disputes','disputes'],['payment-claims','payment_claims']] as const) {
			const response=await page.request.get(`/api/v1/organizations/${organization.id}/${endpoint}`);
			expect(response.status(), endpoint).toBe(200);
			expect(Array.isArray((await response.json())[key]), endpoint).toBe(true);
		}

		const paymentsResponse = await page.request.get(`/api/v1/organizations/${organization.id}/payments`);
		expect(paymentsResponse.status()).toBe(200);
		const paymentsBody = await paymentsResponse.json() as { payments?: Array<{ amount_kobo: number }> };
		expect(paymentsBody.payments).toBeDefined();

		await page.goto('/app/payments');
		await expect(page.getByRole('heading', { name: 'Your money, clearly.' })).toBeVisible();
		await expect(page.getByRole('heading', { name: 'Money received.' })).toBeVisible();
		await expect(page.getByText('We could not open your payment records.')).toHaveCount(0);
	});

	test('buyer browser opens persisted credit requests through the real API', async ({ page }) => {
		const me = await login(page, 'buyer@royal-pharmacy.test');
		expect(me.user.email).toBe('buyer@royal-pharmacy.test');

		const credit = await page.request.get('/api/v1/buyer/credit-requests');
		expect(credit.status()).toBe(200);
		const creditBody = await credit.json() as { requests?: Array<{ request: { id: string; state: string; goods_description: string } }> };
		expect(creditBody.requests).toBeDefined();
		expect(creditBody.requests!.length).toBeGreaterThan(0);

		const listRead = page.waitForResponse(response => response.url().endsWith('/api/v1/buyer/credit-requests') && response.request().method() === 'GET');
		await page.goto('/buyer/requests');
		expect((await listRead).status()).toBe(200);
		await expect(page.getByRole('heading', { name: 'Sales waiting for you', exact: true })).toBeVisible();
		const waiting = creditBody.requests!.filter(view => ['SENT', 'BUYER_REVIEWING'].includes(view.request.state));
		if (waiting.length === 0) await expect(page.getByRole('heading', { name: 'No sale is waiting for you' })).toBeVisible();
		for (const view of waiting.slice(0, 20)) await expect(page.locator(`a[href="/buyer/credit-requests/${view.request.id}"]`)).toBeVisible();
		const persisted = creditBody.requests![0].request;
		await page.goto(`/buyer/credit-requests/${persisted.id}`);
		await expect(page.getByText(persisted.goods_description, { exact: true }).first()).toBeVisible();
		await expect(page.getByText(/Service unavailable|We could not open/i)).toHaveCount(0);
	});

	test('frontend proxy and API readiness agree against the same running stack', async ({ page }) => {
		const proxied = await page.request.get('/api/v1/readyz');
		expect(proxied.status()).toBe(200);
		const body = await proxied.json() as { status?: string };
		expect(body.status ?? 'ok').not.toBe('failed');
		await page.goto('/');
		await expect(page.locator('body')).toContainText('Kredit');
	});
});
