import { expect, test, type Page } from '@playwright/test';
import { PARKED, PARKED_REASON } from './parked';

test.skip(PARKED, PARKED_REASON);

type Role = 'owner' | 'finance';
async function setup(page: Page, role: Role) {
	const user = role === 'owner' ? 'maker' : 'reviewer';
	const state = {
		role,
		user_id: user,
		reviewers: [
			{
				user_id: 'reviewer',
				name: 'Finance reviewer',
				role: 'finance',
				ceiling_kobo: null as number | null,
				version: 0
			}
		],
		controls: { enabled: true, threshold_kobo: 500000, version: 3 },
		approvals: [
			{
				id: 'approval-1',
				request_id: 'draft-1',
				requested_by: 'maker',
				customer_name: 'City Distributors',
				reason: '',
				state: 'pending',
				stale: false,
				proposal: {
					principal_kobo: 750000,
					goods_description: 'Ten cartons',
					due_date: '2026-10-01',
					created_by: 'maker'
				}
			}
		]
	};
	await page.route('**/api/v1/me', (r) => r.fulfill({ json: { user: { id: user } } }));
	await page.route('**/api/v1/organizations', (r) =>
		r.fulfill({ json: { organizations: [{ id: 'manufacturer', legal_name: 'Factory' }] } })
	);
	await page.route('**/api/v1/organizations/manufacturer/credit-approvals', (r) => r.fulfill({ json: state }));
	return state;
}
test.beforeEach(async ({ context, baseURL }) => {
	await context.addCookies([{ name: 'kredit_session', value: 'approval-fixture', url: baseURL! }]);
});
test('the maker sees the rule but cannot approve their own request', async ({ page }) => {
	await setup(page, 'owner');
	await page.goto('/workspace/sales/approvals?organization=manufacturer');
	await expect(page.getByRole('heading', { name: 'City Distributors · ₦7,500.00' })).toBeVisible();
	await expect(page.getByText('A different authorized reviewer must decide this request.')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Approve exact terms' })).toHaveCount(0);
});
test('an independent reviewer must explain the decision and sees the recorded result', async ({ page }) => {
	const state = await setup(page, 'finance');
	let count = 0;
	await page.route('**/api/v1/organizations/manufacturer/credit-approvals/approval-1', (r) => {
		count++;
		expect(r.request().postDataJSON()).toEqual({ decision: 'approved', reason: 'Checked the order and due date' });
		state.approvals[0].state = 'approved';
		state.approvals[0].reason = 'Checked the order and due date';
		return r.fulfill({ json: { id: 'approval-1', status: 'recorded' } });
	});
	await page.goto('/workspace/sales/approvals?organization=manufacturer');
	const approve = page.getByRole('button', { name: 'Approve exact terms' });
	await expect(approve).toBeDisabled();
	await page.getByLabel('Reason for your decision').fill('Checked the order and due date');
	await approve.click();
	await expect(page.getByText('Approval recorded. The seller can now send these exact terms.')).toBeVisible();
	await expect(approve).toHaveCount(0);
	expect(count).toBe(1);
	await expect(page.getByRole('button', { name: 'Save approval rule' })).toHaveCount(0);
});
test('changed draft terms cannot be approved from the old record', async ({ page }) => {
	const state = await setup(page, 'finance');
	state.approvals[0].stale = true;
	await page.goto('/workspace/sales/approvals?organization=manufacturer');
	await expect(page.getByText('This record no longer matches an editable draft.')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Approve exact terms' })).toHaveCount(0);
});
test('owner threshold updates use exact kobo and the loaded version', async ({ page }) => {
	const state = await setup(page, 'owner');
	await page.route('**/api/v1/organizations/manufacturer/credit-approvals/policy', (r) => {
		expect(r.request().method()).toBe('PUT');
		expect(r.request().postDataJSON()).toEqual({ enabled: true, threshold_kobo: 5000050, version: 3 });
		state.controls = { enabled: true, threshold_kobo: 5000050, version: 4 };
		return r.fulfill({ json: state.controls });
	});
	await page.goto('/workspace/sales/approvals?organization=manufacturer');
	await page.getByLabel('Approval required above (₦)').fill('50,000.50');
	await page.getByRole('button', { name: 'Save approval rule' }).click();
	await expect(page.getByText('Approval rule saved.')).toBeVisible();
	await expect(page.getByLabel('Approval required above (₦)')).toHaveValue('50000.50');
});

test('a reviewer cannot approve an offer above their own ceiling', async ({ page }) => {
	const state = await setup(page, 'finance');
	state.reviewers[0].ceiling_kobo = 500000;
	await page.goto('/workspace/sales/approvals?organization=manufacturer');
	await page.getByLabel('Reason for your decision').fill('Reviewed all terms');
	await expect(page.getByRole('button', { name: 'Approve exact terms' })).toBeDisabled();
	await expect(
		page.getByText('This offer exceeds your approval limit. Ask another authorized reviewer.')
	).toBeVisible();
});
test('an owner sets a reviewer ceiling with exact money and version', async ({ page }) => {
	const state = await setup(page, 'owner');
	await page.route('**/credit-approvals/reviewers/reviewer', (r) => {
		expect(r.request().postDataJSON()).toEqual({ ceiling_kobo: 2500050, version: 0 });
		state.reviewers[0].ceiling_kobo = 2500050;
		state.reviewers[0].version = 1;
		return r.fulfill({ json: state.reviewers[0] });
	});
	await page.goto('/workspace/sales/approvals?organization=manufacturer');
	await page.getByLabel('Approval limit for Finance reviewer (₦)').fill('25,000.50');
	await page.getByRole('button', { name: 'Save limit for Finance reviewer' }).click();
	await expect(page.getByText('Reviewer limit saved.')).toBeVisible();
});
