import { expect, test } from '@playwright/test';

test('public homepage renders the product promise', async ({ page }) => {
	await page.goto('/');
	await expect(page.getByRole('heading', { name: /Give goods now/i })).toBeVisible();
	await expect(page.getByRole('link', { name: /Add your first sale/i }).first()).toBeVisible();
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
	await expect(page.getByRole('heading', { name: /See a credit sale/i })).toBeVisible();
	for (const label of ['Send to my customer', 'Accept this sale', 'The goods have left', 'I received the goods', 'Record sample payment']) {
		await page.getByRole('button', { name: label }).click();
	}
	await expect(page.getByRole('heading', { name: /Everyone sees .* left/i })).toBeVisible();
	await expect(page.getByRole('link', { name: /Add my first real sale/i })).toBeVisible();
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
