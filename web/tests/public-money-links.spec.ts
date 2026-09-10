import { expect, test } from '@playwright/test';

test('a reversed receipt cannot be mistaken for an active payment', async ({ page }) => {
  await page.route('**/api/v1/public/receipts/audit', route => route.fulfill({ json: { receipt: {
    reference: 'payment-audit', amount_kobo: 150000, state: 'reversed', paid_at: '2026-09-01T12:00:00Z'
  } } }));
  await page.goto('/receipt/audit');
  await expect(page.getByRole('heading', { name: 'This payment was reversed.' })).toBeVisible();
  await expect(page.getByText('This money was received.', { exact: true })).toHaveCount(0);
  await expect(page.getByRole('status')).toContainText('no longer reduces the balance');
  await expect(page.getByRole('link', { name: 'Email Kredit support' })).toHaveAttribute('href', /^mailto:hello@kredit\.com\.ng/);
});

test('public receipt failures recover and missing amounts are never accepted', async ({ page }) => {
  let ready = false;
  await page.route('**/api/v1/public/receipts/audit', route => route.fulfill({ json: { receipt: ready
    ? { reference: 'payment-audit', amount_kobo: 150000, state: 'recognized', paid_at: '2026-09-01T12:00:00Z' }
    : { reference: 'payment-audit', state: 'recognized', paid_at: '2026-09-01T12:00:00Z' } } }));
  await page.goto('/receipt/audit');
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(page.getByText('This money was received.', { exact: true })).toHaveCount(0);
  ready = true;
  await page.getByRole('button', { name: 'Try again' }).click();
  await expect(page.getByRole('heading', { name: 'This money was received.' })).toBeVisible();
});

test('a paid-off payment link offers no further payment action', async ({ page }) => {
  await page.route('**/api/v1/public/payment-intents/audit', route => route.fulfill({ json: { payment_intent: {
    reference: 'sale-audit', supplier_name: 'Test seller', description: 'Goods', amount_kobo: 0, provider_action: 'Sign in to review.'
  } } }));
  await page.goto('/pay/audit');
  await expect(page.getByRole('heading', { name: 'There is nothing left to pay.' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Continue to pay' })).toHaveCount(0);
});
