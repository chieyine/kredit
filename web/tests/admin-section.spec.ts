import { expect, test } from '@playwright/test';

const json = (body: unknown) => ({
	status: 200,
	contentType: 'application/json',
	body: JSON.stringify(body)
});

test.beforeEach(async ({ page, context, baseURL }) => {
	await page.route('**/api/v1/ops/attention', (r) => r.fulfill(json({ items: [] })));
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
						version: 1,
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
	await expect(page.getByRole('heading', { name: 'Operations overview' })).toBeVisible();
	await expect(page.getByRole('navigation', { name: 'Admin account', exact: true })).toBeVisible();

	await page.goto('/admin/users');
	await expect(page.getByRole('heading', { level: 1, name: 'Users', exact: true })).toBeVisible();
	await expect(page.getByText('Ada Okafor')).toBeVisible();

	await page.goto('/admin/organizations');
	await expect(page.getByRole('heading', { level: 1, name: 'Businesses', exact: true })).toBeVisible();
	await expect(page.getByText('Ada Market Store')).toBeVisible();
	await expect(page.getByText('unregistered business', { exact: true })).toBeVisible();

	await page.goto('/admin/money');
	await expect(page.getByRole('heading', { level: 1, name: 'Money', exact: true })).toBeVisible();
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
	// Five daily areas; everything else is one tap away under the menu.
	await expect(mobileNavigation.getByRole('link')).toHaveCount(5);
	await expect(mobileNavigation.getByRole('button')).toHaveCount(0);
	await page
		.getByRole('navigation', { name: 'Admin account', exact: true })
		.getByRole('button', { name: 'Menu', exact: true })
		.click();

	const more = page.getByRole('dialog', { name: 'Admin account menu' });
	await expect(more).toBeVisible();
	await expect(more.getByRole('navigation', { name: 'Account menu pages' }).getByRole('link')).toHaveCount(13);
	await expect(more.getByRole('link', { name: 'Engineering tools' })).toBeVisible();
	await expect(more.getByRole('link', { name: 'Reconciliation' })).toBeVisible();
	await expect(more.getByText('Problems', { exact: true })).toBeVisible();
	await expect(more.getByText('Settings', { exact: true })).toBeVisible();
	await more.getByRole('link', { name: 'Support cases' }).click();
	await expect(page).toHaveURL(/\/admin\/cases$/);
	await expect(page.getByRole('heading', { level: 1, name: 'Support cases', exact: true })).toBeVisible();
	await expect(more).toHaveCount(0);
});

test('an administrator can find a person and give access without copying an ID', async ({ page }) => {
	await page.route('**/api/v1/ops/team', async (route) => route.fulfill(json({ members: [] })));
	await page.route('**/api/v1/ops/users?q=*&limit=10', async (route) =>
		route.fulfill(
			json({ users: [{ id: 'user-1', display_name: 'Ada Okafor', identifier: 'ada@example.com', status: 'active' }] })
		)
	);
	let submitted: Record<string, unknown> | undefined;
	await page.route('**/api/v1/ops/team/user-1/roles', async (route) => {
		submitted = route.request().postDataJSON();
		await route.fulfill(
			json({
				member: {
					assignment_id: 'role-1',
					user_id: 'user-1',
					display_name: 'Ada Okafor',
					identifier: 'ada@example.com',
					role: submitted!.role
				}
			})
		);
	});

	await page.goto('/admin/team');
	await expect(page.getByRole('heading', { level: 1, name: 'Admin team', exact: true })).toBeVisible();
	await page.getByLabel('Find the person').fill('Ada');
	await page.getByRole('button', { name: 'Find user' }).click();
	await page.getByRole('button', { name: /Ada Okafor/ }).click();
	await page.getByLabel('Role').selectOption('dispute_reviewer');
	await page.getByLabel('Why are you giving access?').fill('Ada will review customer disputes.');
	await page.getByRole('button', { name: 'Review this access' }).click();
	expect(submitted).toBeUndefined();
	await page.getByRole('button', { name: 'Grant the reviewed access' }).click();

	await expect(page.getByText('Admin access was granted and recorded.')).toBeVisible();
	expect(submitted).toMatchObject({ role: 'dispute_reviewer', reason: 'Ada will review customer disputes.' });
});
