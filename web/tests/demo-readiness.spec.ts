import { expect, test } from '@playwright/test';

test('sample sale waits for its controls to become interactive', async ({ page }) => {
  let release!: () => void;
  const assets = new Promise<void>(resolve => { release = resolve; });
  await page.route(/\.js(?:\?|$)/, async route => { await assets; await route.continue(); });
  try {
    await page.goto('/demo', { waitUntil: 'commit' });
    const send = page.getByRole('button', { name: 'Send it to my customer' });
    await expect(send).toBeVisible();
    await expect(send).toBeDisabled();
    release();
    await expect(send).toBeEnabled();
    await send.click();
    await expect(page.getByRole('button', { name: 'Yes, I accept this sale' })).toBeVisible();
  } finally { release(); }
});
