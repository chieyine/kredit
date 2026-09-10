import { expect, test } from '@playwright/test';

test.beforeEach(async ({ page, context, baseURL }) => {
  await context.addCookies([{ name: 'kredit_session', value: 'audit-session', url: baseURL! }]);
  await page.route('**/api/v1/me', route => route.fulfill({ json: { user: { id: 'audit-user' }, organizations: [], mfa_enrolled: true } }));
});

test('failed permission reads never claim reminders are stopped', async ({ page }) => {
  let ready = false;
  await page.route('**/api/v1/buyer/credit-requests', route => route.fulfill({ json: { requests: [{ request: { supplier_organization_id: 'seller-a', supplier_legal_name: 'Audit seller' } }] } }));
  await page.route('**/api/v1/buyer/relationships/consents', route => ready
    ? route.fulfill({ json: { suppliers:[{id:'seller-a',legal_name:'Audit seller',trading_name:''}],consents: [{ id: 'consent-a', supplier_organization_id: 'seller-a', consent_type: 'payment_reminders', granted: true, created_at: '2026-09-01T12:00:00Z' }] } })
    : route.fulfill({ status: 503, json: {} }));
  await page.goto('/buyer/permissions');
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(page.getByText('Reminders stopped', { exact: true })).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Allow payment reminders' })).toHaveCount(0);
  ready = true;
  await page.getByRole('button', { name: 'Try again' }).click();
  await expect(page.getByText('Reminders allowed', { exact: true })).toBeVisible();
});

test('an interrupted backup-code rotation releases the button and reports uncertainty', async ({ page }) => {
  await page.route('**/api/v1/me/recovery-codes/regenerate', route => route.abort('failed'));
  await page.goto('/app/settings/security');
  await expect(page.getByRole('heading', { name: 'Extra sign-in safety is on' })).toBeVisible();
  await page.getByRole('button', { name: 'Make new backup codes' }).click();
  await expect(page.getByRole('status')).toContainText('We could not confirm the result');
  await expect(page.getByRole('button', { name: 'Make new backup codes' })).toBeEnabled();
  await expect(page.getByRole('heading', { name: 'Write these backup codes down now' })).toHaveCount(0);
});

test('admin team list failure is recoverable without claiming no administrators exist', async ({ page }) => {
  let ready = false;
  await page.route('**/api/v1/ops/team', route => ready ? route.fulfill({ json: { members: [] } }) : route.abort('failed'));
  await page.goto('/admin/team');
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(page.getByText('No active admin roles were found.')).toHaveCount(0);
  ready = true;
  await page.getByRole('button', { name: 'Try again' }).click();
  await expect(page.getByText('No active admin roles were found.')).toBeVisible();
});


test('multiple collection attempts for one sale render without duplicate row keys', async ({ page }) => {
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.route('**/api/v1/organizations', route => route.fulfill({ json: { organizations: [{ id: 'org-a', legal_name: 'Audit seller' }] } }));
  await page.route('**/api/v1/organizations/org-a/collections', route => route.fulfill({ json: { collections: [
    { id: 'same-sale', attempt_id: 'attempt-a', buyer_legal_name: 'Test customer', amount_kobo: 100000, state: 'FAILED', created_at: '2026-09-01T10:00:00Z' },
    { id: 'same-sale', attempt_id: 'attempt-b', buyer_legal_name: 'Test customer', amount_kobo: 100000, state: 'PENDING', created_at: '2026-09-02T10:00:00Z' }
  ] } }));
  await page.goto('/app/collections');
  await expect(page.locator('a.record')).toHaveCount(2);
  expect(errors).toEqual([]);
});

test('message choices cannot be overwritten with defaults after a failed read', async ({ page }) => {
  let ready = false;
  await page.route('**/api/v1/me/notification-preferences', route => ready ? route.fulfill({ json: { preferences: { preferred_channel: 'email', fallback_channel: 'sms', payment_reminders_enabled: false, product_updates_enabled: false, quiet_start_hour: 22, quiet_end_hour: 7, timezone: 'Africa/Lagos', version: 4 } } }) : route.fulfill({ status: 503, json: {} }));
  await page.goto('/app/settings/notifications');
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Save my choices' })).toHaveCount(0);
  ready = true;
  await page.getByRole('button', { name: 'Try again' }).click();
  await expect(page.getByLabel('Try this first')).toHaveValue('email');
  await expect(page.getByLabel('Remind me about payments')).not.toBeChecked();
  await expect(page.getByRole('button', { name: 'Save my choices' })).toBeEnabled();
});

test('interrupted account recovery can be retried without a stuck button', async ({ page }) => {
  await page.route('**/api/v1/account-recovery/requests', route => route.abort('failed'));
  await page.goto('/recover');
  await page.getByLabel('Your email or phone number').fill('owner@example.test');
  await page.getByRole('button', { name: 'Start', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('We could not confirm the result');
  await expect(page.getByRole('button', { name: 'Start', exact: true })).toBeEnabled();
  await expect(page.getByRole('status')).toHaveCount(0);
});

for (const path of ['billing', 'settlement', 'credit-policy']) {
  test(`${path} settings explain an outage and recover before allowing edits`, async ({ page }) => {
    let ready = false;
    await page.route('**/api/v1/organizations', route => ready ? route.fulfill({ json: { organizations: [{id:'org-a'}] } }) : route.abort('failed'));
    await page.route('**/api/v1/organizations/org-a/onboarding', route => route.fulfill({json:{profile:{version:1},permissions:{billing:true,settlement:true,credit_policy:true}}}));
    await page.goto(`/app/settings/${path}`);
    await expect(page.getByRole('alert')).toBeVisible();
    await expect(page.getByText(/only the owner/)).toHaveCount(0);
    ready = true;
    await page.getByRole('button',{name:'Try again'}).click();
    await expect(page.getByRole('alert')).toHaveCount(0);
    await expect(page.getByRole('button',{name:/Save/})).toBeVisible();
  });
}

test('privacy request outage does not claim an empty history',async({page})=>{
 await page.route('**/api/v1/me/privacy-requests',route=>route.abort('failed'));
 await page.goto('/app/settings/privacy');
 await expect(page.getByRole('alert')).toBeVisible();
 await expect(page.getByText('You have not sent any request.')).toHaveCount(0);
 await expect(page.getByRole('button',{name:'Send my request'})).toBeDisabled();
});

test('buyer can manage reminders for a seller before any sale activates',async({page})=>{
 let granted=false;
 await page.route('**/api/v1/buyer/relationships/consents',async route=>{
  const supplier={id:'limit-only-seller',legal_name:'Limit supplier',trading_name:''};
  if(route.request().method()==='POST'){
   const body=route.request().postDataJSON();expect(body.supplier_organization_id).toBe(supplier.id);expect(body.granted).toBe(true);expect(body.evidence_hash).toMatch(/^[a-f0-9]{64}$/);
   granted=true;await route.fulfill({json:{consent:{id:'new-consent',supplier_organization_id:supplier.id,consent_type:'payment_reminders',granted,created_at:'2026-09-09T12:00:00Z'}}});return;
  }
  await route.fulfill({json:{suppliers:[supplier],consents:granted?[{id:'new-consent',supplier_organization_id:supplier.id,consent_type:'payment_reminders',granted,created_at:'2026-09-09T12:00:00Z'}]:[]}});
 });
 await page.goto('/buyer/permissions');await expect(page.getByText('Limit supplier',{exact:true})).toBeVisible();
 await page.getByRole('button',{name:'Allow payment reminders'}).click();
 await expect(page.getByRole('button',{name:'Stop optional reminders'})).toBeVisible();
});
