import { expect, test } from '@playwright/test';

test('public security page uses service language instead of an internal launch checklist', async ({ page }) => {
	const response = await page.goto('/security');
	expect(response?.ok()).toBeTruthy();
	await expect(page.getByRole('heading', { name: 'How Kredit protects you.' })).toBeVisible();
	const content = await page.locator('main.trust-page').innerText();
	expect(content).not.toMatch(/pre[- ]launch|before we (?:go live|launch)|coming soon|outside reviewers check us before|tested before launch/i);
	expect(content).not.toMatch(/we never debit twice|nobody signs in as you|certified by Mono|independently certified/i);
	await expect(page.getByRole('heading', { name: 'Permission before bank collection' })).toBeVisible();
	await expect(page.getByText(/a debit already submitted may still finish/)).toBeVisible();
	await expect(page.getByRole('link', { name: 'Report a problem' })).toHaveAttribute('href', '/legal/complaints');
});

test('public security content remains readable on a small phone', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/security');
	await expect(page.getByRole('heading', { name: 'How Kredit protects you.' })).toBeVisible();
	const width = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, viewport: window.innerWidth }));
	expect(width.page).toBeLessThanOrEqual(width.viewport);
	await expect(page.getByRole('heading', { name: 'Check before retrying.' })).toBeVisible();
});
