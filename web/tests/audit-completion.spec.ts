import { test, expect, type Page, type BrowserContext } from '@playwright/test';
import { readDraft, saveDraft } from '../src/lib/sale-drafts';

async function signedIn(page: Page, context: BrowserContext, baseURL?: string) {
  await context.addCookies([{ name: 'kredit_session', value: 'synthetic-completion-session', url: baseURL ?? 'http://127.0.0.1:5173' }]);
  await page.route('**/api/v1/**', route => {
    const path = new URL(route.request().url()).pathname;
    if (path === '/api/v1/me') return route.fulfill({ json: { user: { id: 'u1' }, session: { id: 'session-1' } } });
    if (path === '/api/v1/organizations') return route.fulfill({ json: { organizations: [{ id: 'org-a', legal_name: 'Example supplier A' }, { id: 'org-b', legal_name: 'Example supplier B' }] } });
    if (path.endsWith('/customers')) return route.fulfill({ json: { customers: [{ buyer_user_id: 'buyer-1', buyer_business_id: 'business-1', legal_name: 'Example buyer', trading_name: 'Example buyer', state: 'verified' }] } });
    if (path.endsWith('/reports/receivables')) return route.fulfill({ json: { summary: { obligation_count: 0, outstanding_kobo: 0, overdue_kobo: 0 } } });
    if (path.endsWith('/credit-requests')) return route.fulfill({ json: { requests: [] } });
    for (const [suffix, key] of [['/payments', 'payments'], ['/payment-claims', 'payment_claims'], ['/overdue', 'overdue'], ['/disputes', 'disputes']]) {
      if (path.endsWith(suffix)) return route.fulfill({ json: { [key]: [] } });
    }
    return route.fulfill({ status: 404, json: { code: 'not_found' } });
  });
}
async function draftCount(page: Page) {
  return page.evaluate(() => Object.keys(sessionStorage).filter(key => key.startsWith('kredit.quick-sale.')).length);
}
for (const [kind, path, goodsLabel] of [
  ['quick', '/app/credit/quick?organization=org-a', 'Goods and quantity'],
  ['advanced', '/app/credit/new?advanced=1&organization=org-a', 'What goods are they taking?']
]) {
  test(`${kind} draft storage requires consent, restores only opted-in fields and can be revoked`, async ({ page, context, baseURL }) => {
    await signedIn(page, context, baseURL);
    await page.goto(path);
    const consent = page.getByRole('checkbox', { name: /Keep .*on this device/ });
    const chooseCustomer = async () => {
      await page.getByRole('combobox', { name: 'Customer', exact: true }).selectOption('buyer-1:business-1');
      if (kind === 'quick') await page.getByRole('button', { name: 'Continue', exact: true }).click();
    };
    await expect(consent).not.toBeChecked();
    await chooseCustomer();
    await page.getByRole('textbox', { name: goodsLabel, exact: true }).fill('Synthetic oil cartons');
    await page.getByRole('textbox', { name: 'Sale amount (₦)', exact: true }).fill('127,500.49');
    await expect.poll(() => draftCount(page)).toBe(0);
    await consent.check();
    await expect.poll(() => draftCount(page)).toBe(1);
    const saved = await page.evaluate(() => JSON.parse(sessionStorage.getItem('kredit.quick-sale.v3:u1:org-a')!));
    expect(saved).toMatchObject({ userID: 'u1', organizationID: 'org-a', goods: 'Synthetic oil cartons', principal: '127,500.49' });
    expect(Object.keys(saved).sort()).toEqual(['dueDate', 'expiresAt', 'goods', 'organizationID', 'principal', 'userID']);
    await page.reload();
    await expect(consent).toBeChecked();
    await expect(page.getByRole('combobox', { name: 'Customer', exact: true })).toHaveValue('');
    await chooseCustomer();
    await expect(page.getByRole('textbox', { name: goodsLabel, exact: true })).toHaveValue('Synthetic oil cartons');
    await consent.uncheck();
    await expect.poll(() => draftCount(page)).toBe(0);
    await page.reload();
    await expect(consent).not.toBeChecked();
    await chooseCustomer();
    await expect(page.getByRole('textbox', { name: goodsLabel, exact: true })).toHaveValue('');
  });
}

test('switching business does not transfer draft contents or draft consent', async ({ page, context, baseURL }) => {
  await signedIn(page, context, baseURL);
  await page.goto('/app/credit/quick?organization=org-a');
  const consent = page.getByRole('checkbox', { name: /Keep this draft/ });
  await page.getByRole('combobox', { name: 'Customer', exact: true }).selectOption('buyer-1:business-1');
  await page.getByRole('button', { name: 'Continue', exact: true }).click();
  await page.getByRole('textbox', { name: 'Goods and quantity' }).fill('Private draft for A');
  await consent.check();
  await expect.poll(() => draftCount(page)).toBe(1);
  await page.getByRole('button', { name: 'Back', exact: true }).click();
  await page.getByRole('combobox', { name: 'Your business', exact: true }).selectOption('org-b');
  await expect(consent).not.toBeChecked();
  await page.getByRole('combobox', { name: 'Customer', exact: true }).selectOption('buyer-1:business-1');
  await page.getByRole('button', { name: 'Continue', exact: true }).click();
  await expect(page.getByRole('textbox', { name: 'Goods and quantity' })).toHaveValue('');
  await expect.poll(() => page.evaluate(() => sessionStorage.getItem('kredit.quick-sale.v3:u1:org-b'))).toBeNull();
});

test('legacy default-on drafts are discarded and runtime-only fields are never persisted', () => {
  const values = new Map<string, string>();
  const storage: Storage = { get length() { return values.size; }, key: i => [...values.keys()][i] ?? null, clear: () => values.clear(), getItem: k => values.get(k) ?? null, setItem: (k, v) => { values.set(k, v); }, removeItem: k => { values.delete(k); } };
  const draft = { goods: 'Synthetic stock', principal: '100.49', dueDate: '', customer: 'DO-NOT-RETAIN', invoice: 'DO-NOT-RETAIN' };
  storage.setItem('kredit.quick-sale.v2:u1:org-a', JSON.stringify({ ...draft, userID: 'u1', organizationID: 'org-a', expiresAt: Date.now() + 10000 }));
  expect(readDraft('u1', 'org-a', storage)).toBeNull();
  expect(storage.length).toBe(0);
  expect(saveDraft('u1', 'org-a', draft, storage)).toBe(true);
  expect(storage.getItem(storage.key(0)!)).not.toContain('DO-NOT-RETAIN');
});

test('feedback recovers from a dropped response and success survives unavailable storage', async ({ page, context, baseURL }) => {
  await signedIn(page, context, baseURL);
  await page.addInitScript(() => { Object.defineProperty(window, 'localStorage', { get() { throw new Error('Storage unavailable in this synthetic test'); } }); });
  const keys: string[] = [];
  await page.route('**/api/v1/me/product-feedback', route => {
    keys.push(route.request().headers()['idempotency-key']);
    return keys.length === 1 ? route.abort('failed') : route.fulfill({ status: 201, json: { feedback: { answer: 'yes' } } });
  });
  await page.goto('/app/overview');
  const yes = page.getByRole('button', { name: 'Yes', exact: true });
  await yes.click();
  await expect(page.getByRole('status').filter({ hasText: 'We could not confirm your answer' })).toBeVisible();
  await expect(yes).toBeEnabled();
  await page.getByRole('button', { name: 'No', exact: true }).click();
  await expect(page.getByRole('status').filter({ hasText: 'Retry the same answer' })).toBeVisible();
  expect(keys).toHaveLength(1);
  await yes.click();
  await expect(page.getByText('Thank you. Your answer helps us improve Kredit.')).toBeVisible();
  await expect(yes).toBeDisabled();
  expect(keys).toHaveLength(2); expect(keys[0]).toBe(keys[1]);
});

test('guide filters preserve user input and no-match recovery after hydration', async ({ page }) => {
  await page.goto('/blog');
  await page.getByLabel('Search guides').fill('fake bank alert');
  await expect(page.locator('.post-row')).toHaveCount(1);
  await page.getByRole('combobox', { name: 'Topic', exact: true }).selectOption('Industry guides');
  await expect(page.getByRole('heading', { name: 'Nothing matches that.' })).toBeVisible();
  await page.getByLabel('Search guides').fill('');
  await expect(page.getByRole('combobox', { name: 'Topic', exact: true })).toHaveValue('Industry guides');
  await expect(page.locator('.post-row')).toHaveCount(1);
  await page.getByLabel('Search guides').fill('not-a-guide-title-987654321');
  await page.getByRole('button', { name: 'Show me every guide' }).click();
  await expect(page.locator('.post-row')).toHaveCount(12);
});

async function saleDetail(page: Page) {
  await page.route('**/organizations/org-a/credit-requests/retry-sale', route => route.fulfill({ json: {
    request: { id: 'retry-sale', version: 1, state: 'ACTIVE', buyer_legal_name: 'Example buyer', principal_kobo: 100049, goods_description: 'Synthetic stock', due_date: '2026-10-01', collection_at: '2026-10-02T12:00:00Z', grace_hours: 24 },
    obligation: { id: 'obl-1', outstanding_kobo: 100049 }
  } }));
  await page.route('**/credit-requests/retry-sale/schedule', route => route.fulfill({ json: { schedule: { id: 'schedule-1' }, items: [] } }));
  await page.route('**/credit-requests/retry-sale/collection/eligibility', route => route.fulfill({ json: { eligible: false, reasons: [] } }));
  await page.route('**/credit-requests/retry-sale/collections', route => route.fulfill({ json: { attempts: [] } }));
}

test('sale-detail payment retry preserves its key, amount and paid-at timestamp after reload', async ({ page, context, baseURL }) => {
  await signedIn(page, context, baseURL); await saleDetail(page);
  const keys: string[] = [], bodies: unknown[] = [];
  await page.route('**/credit-requests/retry-sale/payments', route => {
    if (route.request().method() === 'GET') return route.fulfill({ json: { payments: [] } });
    keys.push(route.request().headers()['idempotency-key']); bodies.push(route.request().postDataJSON());
    return keys.length === 1 ? route.abort('failed') : route.fulfill({ status: 201, json: { payment: { id: 'payment-1' } } });
  });
  const enter = async () => {
    await page.getByLabel('How much did you receive? (₦)').fill('100.49');
    await page.getByLabel('Transfer or POS number').fill('SYNTHETIC-TRANSFER');
    await page.getByRole('button', { name: 'Save this payment', exact: true }).click();
  };
  await page.goto('/app/credit/retry-sale?organization=org-a'); await enter();
  await expect(page.getByRole('alert')).toContainText('We have not confirmed the result');
  await expect(page.getByLabel('How much did you receive? (₦)')).toHaveValue('100.49');
  await expect(page.getByRole('button', { name: 'Save this payment', exact: true })).toBeEnabled();
  await page.reload(); await enter();
  await expect(page.getByText('₦100.49 payment saved.', { exact: true })).toBeVisible();
  expect(keys).toHaveLength(2); expect(keys[0]).toBe(keys[1]); expect(bodies[0]).toEqual(bodies[1]);
  expect(bodies[0]).toMatchObject({ amount_kobo: 10049, provider_reference: 'SYNTHETIC-TRANSFER' });
});

test('sale-detail failed payment and schedule reads never become empty successful histories', async ({ page, context, baseURL }) => {
  await signedIn(page, context, baseURL); await saleDetail(page);
  for (const path of ['payments', 'schedule', 'payment-claims', 'collections', 'collection/eligibility']) {
    await page.route(`**/credit-requests/retry-sale/${path}`, route => route.fulfill({ status: 503, json: { code: 'unavailable' } }));
  }
  await page.goto('/app/credit/retry-sale?organization=org-a');
  await expect(page.getByRole('alert').filter({ hasText: 'payment history' })).toBeVisible();
  await expect(page.getByText('No payment yet.', { exact: true })).toHaveCount(0);
  await page.getByText('Payment days', { exact: true }).click();
  await expect(page.getByRole('alert').filter({ hasText: 'payment dates' })).toBeVisible();
  await expect(page.getByText('No scheduled payments are present in this checked record.')).toHaveCount(0);
  await page.getByText('Ask their bank for the money', { exact: true }).click();
  await expect(page.getByRole('alert').filter({ hasText: 'bank debit eligibility' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Ask the bank now' })).toHaveCount(0);
});

test('sale-detail transfer confirmation cannot bypass the bank-check dialog', async ({ page, context, baseURL }) => {
  await signedIn(page, context, baseURL); await saleDetail(page);
  const claim = { id: 'claim-1', state: 'pending', amount_kobo: 10049, transfer_reference: 'SYNTHETIC-CLAIM', paid_at: '2026-09-07T12:00:00Z' };
  await page.route('**/credit-requests/retry-sale/payment-claims', route => route.fulfill({ json: { payment_claims: [claim] } }));
  let submitted = 0;
  await page.route('**/payment-claims/claim-1/decide', route => { submitted++; return route.fulfill({ json: { payment_claim: { ...claim, state: 'confirmed' } } }); });
  await page.goto('/app/credit/retry-sale?organization=org-a');
  await page.getByLabel('Why are you accepting or rejecting this?').fill('Synthetic bank verification');
  await page.getByRole('button', { name: 'Yes, this money reached me' }).click();
  const dialog = page.getByRole('dialog', { name: 'Confirm money received' });
  await expect(dialog.getByRole('button', { name: 'Confirm received' })).toBeDisabled(); expect(submitted).toBe(0);
  await dialog.getByRole('checkbox').check(); await dialog.getByRole('button', { name: 'Confirm received' }).click();
  await expect(dialog).toHaveCount(0); expect(submitted).toBe(1);
});
