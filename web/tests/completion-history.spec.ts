import { expect, test } from '@playwright/test';
import { PARKED, PARKED_REASON } from './parked';
test.beforeEach(async ({ page, context, baseURL }) => {
	await context.addCookies([{ name: 'kredit_session', value: 'history-fixture', url: baseURL! }]);
	await page.route('**/api/v1/buyer/businesses', (route) =>
		route.fulfill({
			json: { businesses: [{ id: 'business-1', workspace_id: 'buyer-workspace', legal_name: 'Buyer business' }] }
		})
	);
	await page.route('**/api/v1/me', (r) =>
		r.fulfill({ json: { user: { id: 'buyer' }, session: { authentication_level: 'AAL2' }, organizations: [] } })
	);
});
test('buyer sees the approved correction note beside the original financial history', async ({ page }) => {
	test.skip(PARKED, PARKED_REASON);
	await page.route('**/api/v1/buyer/history*', (r) =>
		r.fulfill({
			json: {
				active_obligations: 0,
				completed_obligations: 0,
				dispute_count: 0,
				on_time_count: 0,
				obligations: [],
				corrections: [
					{
						id: 'correction',
						subject_id: 'sale',
						reason: 'Check the delivery description',
						state: 'APPROVED',
						decisions: [
							{
								id: 'decision',
								reason: 'Confirmed that delivery was by pickup.',
								outcome: 'APPROVED',
								decided_at: '2026-09-09T09:00:00Z'
							}
						]
					}
				]
			}
		})
	);
	await page.goto('/workspace/purchases/history');
	await expect(page.getByRole('heading', { name: 'Your correction requests' })).toBeVisible();
	await expect(page.getByText('Approved correction note', { exact: true })).toBeVisible();
	await expect(page.getByText('Confirmed that delivery was by pickup.')).toBeVisible();
	await expect(page.getByText(/Any change to money owed is recorded separately/)).toBeVisible();
});
test('privacy completion explains what was done and what was retained', async ({ page }) => {
	await page.route('**/api/v1/me/privacy-requests', (r) =>
		r.fulfill({
			json: {
				requests: [
					{
						id: 'privacy',
						request_type: 'DELETION',
						state: 'COMPLETED',
						due_at: '2026-10-09T09:00:00Z',
						version: 3,
						completion_reason: 'Removed the optional contact list entry.',
						retention_outcome: 'Existing financial records retained for the reviewed retention period.'
					}
				]
			}
		})
	);
	await page.goto('/account/privacy');
	await expect(page.getByText(/Removed the optional contact list entry/)).toBeVisible();
	await expect(page.getByText(/Existing financial records retained/)).toBeVisible();
});
