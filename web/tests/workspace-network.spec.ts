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

test('Today keeps purchasing balances tied to the selected selling workspace', async ({ page }) => {
	await page.goto('/workspace/today?organization=org-b');
	const balances = page.getByRole('region', { name: 'Supplier balances' });
	await expect(balances).toContainText('₦250.00');
	await expect(balances.getByRole('link')).toHaveAttribute('href', '/workspace/purchases?business_id=profile-1');
	await page.getByRole('combobox', { name: 'Business', exact: true }).selectOption('org-a');
	await expect(balances).toContainText('₦100.00');
	await expect(balances).not.toContainText('₦250.00');
	await expect(balances.getByRole('link')).toHaveAttribute('href', '/workspace/purchases?business_id=profile-0');
});

test('an unavailable purchasing profile never appears as a zero balance', async ({ page }) => {
	await page.route('**/api/v1/buyer/businesses', (route) => route.fulfill({ json: { businesses: [] } }));
	await page.goto('/workspace/today');
	const balances = page.getByRole('region', { name: 'Supplier balances' });
	await expect(balances).toContainText('No purchasing profile is available');
	await expect(balances).not.toContainText('₦0.00');
});

test('distributor import preserves the originating business', async ({ page }) => {
	await page.goto('/workspace/partners/import?organization=org-b');
	await expect(page.getByRole('combobox', { name: 'Your business' })).toHaveValue('org-b');
	await expect(page.getByRole('link', { name: 'Track invitations' })).toHaveAttribute(
		'href',
		'/workspace/partners/invitations?organization=org-b'
	);
});
