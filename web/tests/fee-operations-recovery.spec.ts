import { expect, test } from '@playwright/test';

const feeURL = '**/api/v1/organizations/org-a/fee-operations';
const emptyMessage = 'No fee collections have been recorded.';

test.beforeEach(async ({ page, context, baseURL }) => {
  await context.addCookies([{ name: 'kredit_session', value: 'audit-session', url: baseURL! }]);
  await page.route('**/api/v1/me', route => route.fulfill({ json: { user: { id: 'audit-user' }, organizations: [], mfa_enrolled: true } }));
  await page.route('**/api/v1/organizations', route => route.fulfill({ json: { organizations: [{ id: 'org-a' }] } }));
  await page.route('**/api/v1/organizations/org-a/onboarding', route => route.fulfill({ json: { profile: { version: 1 }, permissions: { billing: true } } }));
  await page.route('**/api/v1/organizations/org-a/fee-invoices', route => route.fulfill({ json: { invoices: [] } }));
});

test('a fee read in progress is not a verified empty history', async ({ page }) => {
  let release!: () => void;
  const gate = new Promise<void>(resolve => { release = resolve; });
  await page.route(feeURL, async route => {
    await gate;
    await route.fulfill({ json: { banks: [], debits: [] } });
  });
  try {
    await page.goto('/workspace/settings/billing');
    const section = page.getByRole('region', { name: 'Fee collections and bank receipts' });
    await expect(section.getByRole('status')).toContainText('Loading fee movements');
    await expect(section.getByText(emptyMessage, { exact: true })).toHaveCount(0);
    await expect(section.getByRole('button', { name: 'Refresh fee movements' })).toBeDisabled();
    release();
    await expect(section.getByText(emptyMessage, { exact: true })).toBeVisible();
    await expect(section.getByRole('status')).toHaveCount(0);
  } finally { release(); }
});

for (const failure of ['unavailable', 'malformed'] as const) {
  test(`a ${failure} fee read remains an error until a valid retry completes`, async ({ page }) => {
    let ready = false;
    await page.route(feeURL, route => {
      if (ready) return route.fulfill({ json: { banks: [], debits: [] } });
      return failure === 'unavailable'
        ? route.fulfill({ status: 503, json: {} })
        : route.fulfill({ json: { banks: [], debits: [{ id: 'debit-1', invoice_id: 'invoice-1', provider: 'synthetic', amount_kobo: 100, state: 'pending', review_required: 'false' }] } });
    });
    await page.goto('/workspace/settings/billing');
    const section = page.getByRole('region', { name: 'Fee collections and bank receipts' });
    await expect(section.getByRole('alert')).toBeVisible();
    await expect(section.getByText(emptyMessage, { exact: true })).toHaveCount(0);
    ready = true;
    await section.getByRole('button', { name: 'Refresh fee movements' }).click();
    await expect(section.getByText(emptyMessage, { exact: true })).toBeVisible();
    await expect(section.getByRole('alert')).toHaveCount(0);
    await expect(section.getByRole('button', { name: 'Refresh fee movements' })).toBeEnabled();
  });
}

test('a refresh outage invalidates previously displayed fee records', async ({ page }) => {
  let available = true;
  await page.route(feeURL, route => available
    ? route.fulfill({ json: { banks: [{ provider: 'synthetic-bank', allocated_kobo: 100, received_kobo: 0, returned_kobo: 0, outstanding_kobo: 100 }], debits: [] } })
    : route.fulfill({ status: 503, json: {} }));
  await page.goto('/workspace/settings/billing');
  const section = page.getByRole('region', { name: 'Fee collections and bank receipts' });
  await expect(section.getByRole('heading', { name: 'synthetic-bank', exact: true })).toBeVisible();
  available = false;
  await section.getByRole('button', { name: 'Refresh fee movements' }).click();
  await expect(section.getByRole('alert')).toBeVisible();
  await expect(section.getByRole('heading', { name: 'synthetic-bank', exact: true })).toHaveCount(0);
  await expect(section.getByText(emptyMessage, { exact: true })).toHaveCount(0);
});
