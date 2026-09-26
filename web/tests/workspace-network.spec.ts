import { expect, test } from '@playwright/test';

const organizations = [
	{ id: 'org-a', legal_name: 'First business' },
	{ id: 'org-b', legal_name: 'Second business' }
];
const profiles = organizations.map((org, i) => ({
	id: `profile-${i}`,
	workspace_id: org.id,
	legal_name: org.legal_name
}));
const sale = (business: string, amount: number) => ({
	request: {
		id: `sale-${business}`,
		state: 'ACTIVE',
		buyer_business_id: business,
		buyer_legal_name: 'Distributor',
		principal_kobo: amount,
		due_date: '2026-10-01'
	},
	obligation: { id: `debt-${business}`, outstanding_kobo: amount }
});

test.beforeEach(async ({ page, context, baseURL }) => {
	await context.addCookies([{ name: 'kredit_session', value: 'network-test', url: baseURL! }]);
	await page.route('**/api/v1/**', (route) => {
		const path = new URL(route.request().url()).pathname;
		if (path === '/api/v1/me') return route.fulfill({ json: { user: { id: 'owner' }, organizations } });
		if (path === '/api/v1/organizations') return route.fulfill({ json: { organizations } });
		if (path === '/api/v1/buyer/businesses') return route.fulfill({ json: { businesses: profiles } });
		if (path === '/api/v1/buyer/credit-requests')
			return route.fulfill({ json: { requests: [sale('profile-0', 10000), sale('profile-1', 25000)] } });
		if (path.endsWith('/reports/receivables'))
			return route.fulfill({ json: { summary: { obligation_count: 0, outstanding_kobo: 0, overdue_kobo: 0 } } });
		const key = (
			{
				'distributor-imports': 'batches',
				'credit-requests': 'requests',
				payments: 'payments',
				overdue: 'overdue',
				'payment-claims': 'payment_claims',
				disputes: 'disputes',
				due: 'due'
			} as Record<string, string>
		)[path.split('/').at(-1)!];
		return route.fulfill({ json: key ? { [key]: [] } : {} });
	});
});

test('Who owes me carries the selected business into every tab', async ({ page }) => {
	await page.goto('/workspace/today?organization=org-b');
	await expect(page.getByRole('heading', { name: 'Who owes me', exact: true })).toBeVisible();
	const tabs = page.getByRole('navigation', { name: 'Your business', exact: true });
	await expect(tabs.getByRole('link', { name: 'Customers', exact: true })).toHaveAttribute(
		'href',
		'/workspace/partners/customers?organization=org-b'
	);
	await expect(tabs.getByRole('link', { name: 'Give goods on credit', exact: true })).toHaveAttribute(
		'href',
		'/workspace/give?organization=org-b'
	);
	// Supplier balances moved off the seller's home to What I owe.
	await expect(page.getByRole('region', { name: 'Supplier balances' })).toHaveCount(0);
});

test('a switched-off tool sends the person back and keeps the business', async ({ page }) => {
	await page.goto('/workspace/partners/import?organization=org-b');
	await expect(page).toHaveURL(/\/workspace\/partners\/customers\?organization=org-b$/);
	await page.goto('/workspace/sales/quick?organization=org-b&goods=Rice');
	await expect(page).toHaveURL(/\/workspace\/give\?organization=org-b&goods=Rice$/);
});
