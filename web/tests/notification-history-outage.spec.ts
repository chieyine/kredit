import { test, expect } from '@playwright/test';

for (const area of ['app', 'buyer']) {
  test(`${area} message history recovers from a dropped response without claiming an empty history`, async ({ page, context, baseURL }) => {
    await context.addCookies([{ name: 'kredit_session', value: 'synthetic-history-session', url: baseURL! }]);
    await page.route('**/api/v1/me', route => route.fulfill({ json: { user: { id: 'history-user' }, session: { id: 'session' } } }));
    let ready = false;
    await page.route('**/api/v1/me/notifications', route => ready ? route.fulfill({ json: { notifications: [{
      id: 'message-1', channel: 'email', template: 'PaymentRecorded', state: 'sent', body: 'Synthetic payment notice',
      sent_at: '0001-01-01T00:00:00Z', scheduled_at: '2026-09-09T12:00:00Z'
    }] } }) : route.abort('failed'));
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(`/${area}/notifications`);
    await expect(page.getByRole('alert')).toContainText('We could not check your message history');
    await expect(page.getByText('No messages yet', { exact: true })).toHaveCount(0);
    ready = true;
    await page.getByRole('button', { name: 'Try again', exact: true }).click();
    await expect(page.getByText('Synthetic payment notice')).toBeVisible();
    await expect(page.locator('.list time')).toContainText('2026');
    await expect(page.locator('.explain')).toContainText('does not confirm');
    expect(await page.locator('html').evaluate(element => element.scrollWidth <= element.clientWidth)).toBe(true);
  });
}

test('malformed message history is an error rather than an empty list', async ({ page, context, baseURL }) => {
  await context.addCookies([{ name: 'kredit_session', value: 'synthetic-history-session', url: baseURL! }]);
  await page.route('**/api/v1/me', route => route.fulfill({ json: { user: { id: 'history-user' } } }));
  await page.route('**/api/v1/me/notifications', route => route.fulfill({ json: {} }));
  await page.goto('/buyer/notifications');
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(page.getByText('No messages yet', { exact: true })).toHaveCount(0);
});
