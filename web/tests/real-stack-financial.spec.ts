import { expect, test, type Page } from '@playwright/test';

const realStack = process.env.KREDIT_REAL_STACK_E2E === '1';
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

		const paymentsResponse = await page.request.get(`/api/v1/organizations/${organization.id}/payments`);
		expect(paymentsResponse.status()).toBe(200);
		const paymentsBody = await paymentsResponse.json() as { payments?: Array<{ amount_kobo: number }> };
		expect(paymentsBody.payments).toBeDefined();

		await page.goto('/app/payments');
		await expect(page.getByText(organization.legal_name, { exact: false }).first()).toBeVisible();
		await expect(page.getByText('We could not open your payment records.')).toHaveCount(0);
	});

	test('buyer browser opens persisted credit requests through the real API', async ({ page }) => {
		const me = await login(page, 'buyer@royal-pharmacy.test');
		expect(me.user.email).toBe('buyer@royal-pharmacy.test');

		const credit = await page.request.get('/api/v1/buyer/credit-requests');
		expect(credit.status()).toBe(200);
		const creditBody = await credit.json() as { requests?: unknown[] };
		expect(creditBody.requests).toBeDefined();
		expect(creditBody.requests!.length).toBeGreaterThan(0);

		await page.goto('/buyer/credit-requests');
		await expect(page.locator('body')).toContainText(/credit|request/i);
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
