import { expect, test } from '@playwright/test';
import { isPrivateRoute, isUnlistedRoute, publicSitemapEntries } from '../src/lib/seo';

test('private and directly shared routes stay out of public discovery', () => {
	for (const path of [
		'/account',
		'/account/privacy',
		'/workspace/today',
		'/personal/purchases',
		'/signin',
		'/start',
		'/agents',
		'/pay/token'
	]) {
		expect(isPrivateRoute(path)).toBe(true);
	}
	for (const path of ['/join', '/deck', '/deck/investor']) expect(isUnlistedRoute(path)).toBe(true);
	for (const entry of publicSitemapEntries) {
		expect(isPrivateRoute(entry.path)).toBe(false);
		expect(isUnlistedRoute(entry.path)).toBe(false);
	}
	for (const path of ['/accounting', '/workspace-guide', '/joining', '/deckchairs']) {
		expect(isPrivateRoute(path)).toBe(false);
		expect(isUnlistedRoute(path)).toBe(false);
	}
});

test('investor presentation is not indexable', async ({ page }) => {
	await page.goto('/deck/investor');
	await expect(page.locator('meta[name="robots"]')).toHaveCount(1);
	await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'noindex,nofollow');
});

test('invalid fractional slide references never hide every slide', async ({ page }) => {
	await page.goto('/deck/investor#1.5');
	await expect(page.locator('.slide.current')).toHaveCount(1);
	await expect(page.locator('.slide').first()).toBeVisible();
	await page.evaluate(() => {
		location.hash = '2.5';
	});
	await expect(page.locator('.slide').first()).toBeVisible();
	await page.evaluate(() => {
		location.hash = '2';
	});
	await expect(page.locator('.slide').nth(1)).toBeVisible();
});

test('focused deck controls keep native keyboard activation', async ({ page }) => {
	await page.goto('/deck/investor');
	const next = page.getByRole('button', { name: 'Next slide', exact: true });
	await next.focus();
	await next.press('Space');
	await expect(page.locator('.slide').nth(1)).toBeVisible();
	const previous = page.getByRole('button', { name: 'Previous slide', exact: true });
	await previous.focus();
	await previous.press('Space');
	await expect(page.locator('.slide').first()).toBeVisible();
});
