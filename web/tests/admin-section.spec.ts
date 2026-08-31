import { expect, test } from '@playwright/test';

const json = (body: unknown) => ({
	status: 200,
	contentType: 'application/json',
	body: JSON.stringify(body)
});

test.beforeEach(async ({ page, context, baseURL }) => {
	await context.addCookies([
		{
			name: 'kredit_session',
			value: 'admin-session',
			url: baseURL ?? 'http://127.0.0.1:5173'
		}
	]);
	await page.route('**/api/v1/me', async (route) =>
		route.fulfill(json({ user: { id: 'admin-1' }, session: { id: 'session-1' }, organizations: [] }))
	);
});

test('the admin section exposes the main platform work without dead screens', async ({ page }) => {
	await page.route('**/api/v1/ops/overview', async (route) =>
		route.fulfill(
			json({
				role: 'platform_admin',
				overview: {
					queued_jobs: 2,
					failed_jobs: 0,
					open_cases: 1,
					open_disputes: 1
				}
			})
		)
	);
	await page.route('**/api/v1/ops/users?*', async (route) =>
		route.fulfill(
			json({
				users: [
					{
						id: 'user-1',
						display_name: 'Ada Okafor',
						identifier: 'ada@example.com',
						status: 'active',
						organization_count: 1,
						created_at: '2026-08-20T09:00:00Z'
					}
				]
			})
		)
	);
	await page.route('**/api/v1/ops/organizations?*', async (route) =>
		route.fulfill(
			json({
				organizations: [
					{
						id: 'org-1',
						legal_name: 'Ada Market Store',
						status: 'verified',
						business_type: 'unregistered_business',
						industry: 'Food supplies',
						outstanding_kobo: 15000000,
						open_sales: 2,
						member_count: 1,
						version: 1,
						created_at: '2026-08-20T09:00:00Z'
					}
				]
			})
		)
	);
	await page.route('**/api/v1/ops/money?*', async (route) =>
		route.fulfill(
			json({
				summary: {
					received_kobo: 20000000,
					reversed_kobo: 0,
					collection_requested_kobo: 10000000,
					collection_succeeded_kobo: 8000000,
					outstanding_kobo: 15000000
				},
				activity: []
			})
		)
	);

	await page.goto('/admin');
	await expect(page.getByRole('heading', { name: 'Run the whole platform from one place.' })).toBeVisible();
	await expect(page.getByRole('navigation', { name: 'Admin account' })).toBeVisible();

	await page.goto('/admin/users');
	await expect(page.getByRole('heading', { name: 'Every Kredit user.' })).toBeVisible();
	await expect(page.getByText('Ada Okafor')).toBeVisible();

	await page.goto('/admin/organizations');
	await expect(page.getByRole('heading', { name: 'Every business on Kredit.' })).toBeVisible();
	await expect(page.getByText('Ada Market Store')).toBeVisible();
	await expect(page.getByText('unregistered business', { exact: true })).toBeVisible();

	await page.goto('/admin/money');
	await expect(page.getByRole('heading', { name: 'The platform money position.' })).toBeVisible();
	await expect(page.getByText('₦200,000.00')).toBeVisible();
});

test('mobile admin navigation stays small and closes after a page is chosen', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.route('**/api/v1/ops/overview', async (route) =>
		route.fulfill(json({ role: 'platform_admin', overview: {} }))
	);
	await page.route('**/api/v1/ops/cases?*', async (route) => route.fulfill(json({ cases: [] })));

	await page.goto('/admin');
	const mobileNavigation = page.getByLabel('Admin account main pages');
	await expect(mobileNavigation).toBeVisible();
	await expect(mobileNavigation.getByRole('link')).toHaveCount(4);
	await mobileNavigation.getByRole('button', { name: 'More' }).click();

	const more = page.getByRole('dialog', { name: 'More pages' });
	await expect(more).toBeVisible();
	await expect(more.getByRole('navigation', { name: 'More account pages' }).getByRole('link')).toHaveCount(12);
	await expect(more.getByText('Customer support', { exact: true })).toBeVisible();
	await expect(more.getByText('Access and control', { exact: true })).toBeVisible();
	await more.getByRole('link', { name: 'Support cases' }).click();
	await expect(page).toHaveURL(/\/admin\/cases$/);
	await expect(page.getByRole('heading', { name: 'Help people reach an answer.' })).toBeVisible();
	await expect(more).toHaveCount(0);
});

test('an administrator can find a person and give access without copying an ID', async ({ page }) => {
	await page.route('**/api/v1/ops/team', async (route) => route.fulfill(json({ members: [] })));
	await page.route('**/api/v1/ops/users?q=*&limit=10', async (route) =>
		route.fulfill(json({ users: [{ id: 'user-1', display_name: 'Ada Okafor', identifier: 'ada@example.com', status: 'active' }] }))
	);
	let submitted: Record<string, unknown> | undefined;
	await page.route('**/api/v1/ops/team/user-1/roles', async (route) => {
		submitted = route.request().postDataJSON();
		await route.fulfill(json({ member: { assignment_id: 'role-1' } }));
	});

	await page.goto('/admin/team');
	await expect(page.getByRole('heading', { name: 'Who can run Kredit.' })).toBeVisible();
	await page.getByLabel('Find the person').fill('Ada');
	await page.getByRole('button', { name: 'Find user' }).click();
	await page.getByRole('button', { name: /Ada Okafor/ }).click();
	await page.getByLabel('Role').selectOption('dispute_reviewer');
	await page.getByLabel('Why are you giving access?').fill('Ada will review customer disputes.');
	await page.getByRole('button', { name: 'Give this access' }).click();

	await expect(page.getByText('Admin access was granted and recorded.')).toBeVisible();
	expect(submitted).toMatchObject({ role: 'dispute_reviewer', reason: 'Ada will review customer disputes.' });
});
