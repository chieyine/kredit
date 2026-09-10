import { expect, test } from '@playwright/test';

test('public homepage renders the product promise', async ({ page }) => {
	await page.goto('/');
	await expect(page.getByRole('heading', { name: /Keep track of every credit sale/i })).toBeVisible();
	// The homepage is read by someone who has no account yet, so its actions say
	// what they do — open an account, or try the sample — rather than promising a
	// sale they cannot record until they have signed in.
	await expect(page.getByRole('link', { name: /Open your account/i }).first()).toBeVisible();
	await expect(page.getByRole('link', { name: /sample sale/i }).first()).toBeVisible();
});

test('public product routes expose clear conversion and trust content', async ({ page }) => {
	for (const path of ['/demo', '/how-it-works', '/for-suppliers', '/for-buyers', '/pricing', '/security', '/faq']) {
		await page.goto(path);
		await expect(page.locator('h1')).toBeVisible();
		await expect(page.getByRole('navigation', { name: 'Main navigation' })).toBeVisible();
	}
});

test('visitor can complete the sample sale without signing in', async ({ page }) => {
	await page.goto('/demo');
	await expect(page.getByRole('heading', { name: /One sale,/i })).toBeVisible();
	for (const label of ['Send it to my customer', 'Yes, I accept this sale', 'Complete sample bank permission', 'The goods have left', 'Yes, I got the goods', 'Enter a sample payment']) {
		await page.getByRole('button', { name: label }).click();
	}
	await expect(page.getByRole('heading', { name: /Both sides see .* left/i })).toBeVisible();
	await expect(page.getByRole('link', { name: /Add my first real sale/i })).toBeVisible();
});

test('mobile homepage remains navigable', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/');
	await expect(page.getByRole('heading', { name: /Keep track of every credit sale/i })).toBeVisible();
	await page.locator('summary', { hasText: 'Menu' }).click();
	await expect(page.getByRole('navigation', { name: 'Main navigation' }).getByRole('link', { name: 'Pricing' })).toBeVisible();
});

test('private pages are excluded from indexing', async ({ page }) => {
	await page.goto('/app/overview');
	await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'noindex,nofollow');
});
