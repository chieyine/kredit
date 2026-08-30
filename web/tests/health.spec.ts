import { expect, test } from '@playwright/test';

test('public homepage renders the product promise', async ({ page }) => {
	await page.goto('/');
	await expect(page.getByRole('heading', { name: /Give goods now/i })).toBeVisible();
	await expect(page.getByRole('link', { name: /Add your first sale/i }).first()).toBeVisible();
});

test('public product routes expose clear conversion and trust content', async ({ page }) => {
	for (const path of ['/how-it-works', '/for-suppliers', '/for-buyers', '/pricing', '/security', '/faq']) {
		await page.goto(path);
		await expect(page.locator('h1')).toBeVisible();
		await expect(page.getByRole('navigation', { name: 'Main navigation' })).toBeVisible();
	}
});

test('mobile homepage remains navigable', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/');
	await expect(page.getByRole('heading', { name: /Give goods now/i })).toBeVisible();
	await page.locator('summary', { hasText: 'Menu' }).click();
	await expect(page.getByRole('navigation', { name: 'Main navigation' }).getByRole('link', { name: 'Price' })).toBeVisible();
});

test('private pages are excluded from indexing', async ({ page }) => {
	await page.goto('/app/overview');
	await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'noindex,nofollow');
});
