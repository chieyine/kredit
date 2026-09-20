import { expect, test } from '@playwright/test';

test.describe('isolated public API contract', () => {
  test.skip(Boolean(process.env.PLAYWRIGHT_BASE_URL || process.env.API_INTERNAL_URL), 'Uses the explicitly isolated public test fixture, never a live stack.');

  test('initial legal publication and fees render without JavaScript', async ({ browser, baseURL }) => {
    const context = await browser.newContext({ javaScriptEnabled: false, baseURL });
    try {
      const page = await context.newPage();
      expect((await page.goto('/legal/privacy'))?.status()).toBe(200);
      await expect(page.getByRole('heading', { name: 'Privacy notice', exact: true })).toBeVisible();
      expect((await page.goto('/pricing'))?.status()).toBe(200);
      await expect(page.locator('.rates strong')).toHaveText(['0.5%', '0.5%']);
      await expect(page.getByLabel('Fee example')).toContainText('₦2,500.00');
    } finally { await context.close(); }
  });

  for (const [version, status] of [['audit-outage', 503], ['audit-malformed', 503], ['missing', 404]] as const) {
    test(`unverified legal publication ${version} never becomes a successful document`, async ({ page }) => {
      expect((await page.goto(`/legal/terms?version=${version}`))?.status()).toBe(status);
      await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'noindex,nofollow');
      await expect(page.getByRole('heading', { name: 'Terms of service', exact: true })).toHaveCount(0);
      await expect(page.getByText('KREDIT TECHNOLOGIES LIMITED', { exact: true })).toHaveCount(0);
    });
  }
});
