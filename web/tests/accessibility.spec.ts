import { expect, test, type Page, type Route } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const organization = { id: 'org-a11y', legal_name: 'Accessible Supplies Limited', trading_name: 'Accessible Supplies' };
const line = { id: 'line-a11y', buyer_user_id: 'buyer-a11y', buyer_business_id: 'business-a11y', approved_limit_kobo: 100_000_000, current_exposure_kobo: 0, reserved_pending_kobo: 25_000_000, available_limit_kobo: 75_000_000, state: 'ACTIVE', version: 2 };
const drawdown = { id: 'drawdown-a11y', principal_kobo: 25_000_000, goods_description: 'Twenty bags of rice', invoice_reference: 'INV-A11Y', due_date: '2026-10-30', collection_at: '2026-10-31T09:00:00Z', grace_hours: 24, agreement_hash: 'accessible-agreement-hash', state: 'GOODS_RELEASED', delivery_method: 'Courier', release_evidence_reference: 'TRACK-A11Y' };
const creditRequest = { id: 'request-a11y', state: 'BUYER_REVIEWING', supplier_legal_name: 'Accessible Supplies Limited', buyer_legal_name: 'Inclusive Retail Limited', buyer_user_id: 'buyer-a11y', buyer_business_id: 'business-a11y', principal_kobo: 50_000_000, goods_description: 'Verified inventory', due_date: '2026-10-30', collection_at: '2026-10-31T09:00:00Z', grace_hours: 24, schedule_type: 'one_time' };
const dispute = { id: 'dispute-a11y', obligation_id: 'obligation-a11y', supplier_organization_id: organization.id, buyer_user_id: 'buyer-a11y', total_disputed_kobo: 10_000_000, remaining_disputed_kobo: 10_000_000, reason: 'Goods quality', explanation: 'The delivered batch did not match the accepted specification.', state: 'OPEN', collection_effect: 'CONTESTED_ONLY', opened_at: '2026-08-29T08:00:00Z' };

async function mockAPI(page: Page) {
	await page.route('**/api/v1/**', async (route: Route) => {
		const path = new URL(route.request().url()).pathname;
		let body: unknown = {};
		let status = 200;
		if (path === '/api/v1/me' && route.request().method() === 'GET') {
			if (new URL(page.url()).pathname === '/app') status = 401;
			else body = { user: { id: 'user-a11y', status: 'active', created_at: '2026-01-01T00:00:00Z' }, session: { id: 'session-a11y', user_id: 'user-a11y', authentication_level: 'AAL1', created_at: '2026-01-01T00:00:00Z', expires_at: '2027-01-01T00:00:00Z' }, mfa_enrolled: false, organizations: [organization] };
		}
		else if (path === '/api/v1/organizations') body = { organizations: [organization] };
		else if (path.endsWith('/onboarding')) body = { profile: { version: 5, kyb_state: 'approved', settlement_state: 'verified', billing_state: 'configured', authorized_representative_name: 'Ada Example', authorized_representative_title: 'Director', terms_version: 'supplier-terms-v1', privacy_version: 'privacy-v1' }, readiness: { state: 'pilot_ready', ready: true, requirements: [{ code: 'business_identity', label: 'Business identity', complete: true, manage_path: '/app/onboarding' }], missing: [] }, permissions: { business: true, consents: true }, current_terms_version: 'supplier-terms-v1', current_privacy_version: 'privacy-v1' };
		else if (path.endsWith('/customers')) body = { customers: [{ id: 'buyer-a11y', buyer_user_id: 'buyer-a11y', buyer_business_id: 'business-a11y', legal_name: 'Inclusive Retail Limited', state: 'verified' }] };
		else if (path.endsWith('/trade-lines/line-a11y/statement')) body = { line, drawdowns: [drawdown] };
		else if (path.endsWith('/trade-lines')) body = { trade_lines: [line] };
		else if (path.endsWith('/payments')) body = { payments: [] };
		else if (path.endsWith('/payment-claims')) body = { payment_claims: [] };
		else if (path === '/api/v1/buyer/credit-requests/request-a11y') body = { request: creditRequest, agreement: { id: 'agreement-a11y', document_hash: 'accessible-agreement-hash' } };
		else if (path === '/api/v1/buyer/credit-requests/request-a11y/payments') body = { payments: [] };
		else if (path.endsWith('/disputes/dispute-a11y')) body = { dispute, evidence: [], decisions: [] };
		else if (path.endsWith('/disputes')) body = { disputes: [dispute] };
		else if (path === '/api/v1/me/privacy-requests') body = { requests: [] };
		else if (path === '/api/v1/ops/commands/preview') body = { command: { current_version: 1, impact_preview: { effect: 'Sensitive actions will be blocked.', will_notify: true, audit: 'Immutable event required.' } } };
		else if (path.startsWith('/api/v1/account-recovery/requests')) body = { message: 'If eligible, recovery instructions have been sent.' };
		await route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
	});
}

async function expectNoSeriousViolations(page: Page, label: string) {
	const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa', 'wcag22aa']).analyze();
	const blocking = results.violations.filter((violation) => violation.impact === 'serious' || violation.impact === 'critical');
	expect(blocking, `${label}: ${blocking.map((item) => `${item.id} (${item.nodes.length})`).join(', ')}`).toEqual([]);
}

test.beforeEach(async ({ page, context, baseURL }) => {
	await context.addCookies([{ name: 'kredit_session', value: 'accessibility-session', url: baseURL ?? 'http://127.0.0.1:5173' }]);
	await mockAPI(page);
});

for (const journey of [
	['login', '/app'],
	['supplier onboarding', '/app/onboarding'],
	['credit creation', '/app/credit/new'],
	['buyer acceptance', '/buyer/credit-requests/request-a11y'],
	['goods release', '/app/trade-lines/line-a11y'],
	['goods receipt and drawdown', '/buyer/trade-lines'],
	['payments', '/app/payments'],
	['supplier disputes', '/app/disputes/dispute-a11y?organization=org-a11y'],
	['settings', '/app/settings'],
	['account recovery', '/recover'],
	['privacy', '/app/settings/privacy']
] as const) {
	test(`${journey[0]} has no serious or critical WCAG violations`, async ({ page }) => {
		await page.goto(journey[1]);
		if (journey[1] !== '/app') await expect(page.locator('.account-gate')).toHaveCount(0);
		await expect(page.locator('h1')).toBeVisible();
		await expectNoSeriousViolations(page, journey[0]);
	});
}

test('operations command surface has no serious or critical WCAG violations', async ({ page }) => {
	await page.goto('/admin/controls');
	await expect(page.getByRole('button', { name: 'Preview impact' })).toBeVisible();
	await expectNoSeriousViolations(page, 'operations command surface');
});

test('keyboard, focus, reflow, reduced motion, and touch-target safeguards remain active', async ({ page }) => {
	await page.emulateMedia({ reducedMotion: 'reduce' });
	await page.goto('/app/overview');
	await expect(page.locator('.account-gate')).toHaveCount(0);
	await page.keyboard.press('Tab');
	await expect(page.getByRole('link', { name: 'Skip to content' })).toBeFocused();
	await page.keyboard.press('Enter');
	await expect(page.locator('#main-content')).toBeFocused();
	const search = page.getByRole('button', { name: /Search/ });
	await search.click();
	await expect(page.getByRole('searchbox', { name: 'Search pages' })).toBeFocused();
	await page.keyboard.press('Shift+Tab');
	expect(await page.locator('dialog').evaluate((element) => element.contains(document.activeElement))).toBe(true);
	await page.keyboard.press('Escape');
	await expect(search).toBeFocused();
	// A 640 CSS-pixel viewport is the layout viewport produced by 200% browser zoom
	// on the required 1280px desktop baseline.
	await page.setViewportSize({ width: 640, height: 800 });
	const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
	expect(overflow).toBeLessThanOrEqual(1);
	const tooSmall = await page.locator('button:visible, a:visible, input:visible, select:visible, textarea:visible').evaluateAll((elements) => elements.filter((element) => { const box = element.getBoundingClientRect(); return box.width > 0 && box.height > 0 && box.height < 24; }).map((element) => element.outerHTML));
	expect(tooSmall).toEqual([]);
});

test('credit validation focuses a linked error summary', async ({ page }) => {
	await page.goto('/app/credit/new');
	await page.getByRole('combobox', { name: 'Customer', exact: true }).selectOption('buyer-a11y');
	await page.getByLabel('Money to pay (₦)').fill('0');
	await page.getByLabel('What goods did they take?').fill('Inventory');
	await page.getByLabel('First payment day').fill('2026-10-30');
	await page.getByLabel('Day Kredit may debit if unpaid').fill('2026-10-31T09:00');
	await page.getByRole('button', { name: 'Save this sale' }).click();
	const summary = page.getByRole('alert');
	await expect(summary).toContainText('Complete every required field');
	await expect(summary).toBeFocused();
});

test('offline mode is announced and financial actions remain unqueued', async ({ page, context }) => {
	await page.goto('/app/settings');
	await expect(page.locator('.palette-trigger')).toHaveAttribute('data-ready', 'true');
	await context.setOffline(true);
	await expect(page.getByText('You are offline. Financial actions are not submitted or queued until you reconnect.')).toBeVisible();
	await context.setOffline(false);
});
