import { test, expect, type Page, type Route, type BrowserContext } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const orgs = [{ id: 'org-a', legal_name: 'Kora Wholesale', trading_name: 'Kora Wholesale' }, { id: 'org-b', legal_name: 'Borno Supplies', trading_name: 'Borno Supplies' }];
const sale = (id = 'sale-1', state = 'BUYER_REVIEWING') => ({
 request: { id, state, supplier_legal_name: 'Kora Wholesale', buyer_legal_name: 'Amina Stores', buyer_user_id: 'buyer-1', buyer_business_id: 'business-1', principal_kobo: 12750049, goods_description: '40 cartons of cooking oil', due_date: '2026-09-18', collection_at: '2026-09-19T22:59:00Z', grace_hours: 24, schedule_type: 'one_time', fee_terms: { policy_revision: 1, base_bps: 50, collection_bps: 50 } },
 agreement: { id: 'agreement-1', document_hash: 'a'.repeat(64) }, mandate: null as unknown, obligation: null as unknown
});
const send = (route: Route, body: unknown, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
async function signedIn(page: Page, context: BrowserContext, baseURL?: string) {
 await context.addCookies([{ name: 'kredit_session', value: 'synthetic-audit-session', url: baseURL ?? 'http://127.0.0.1:5173' }, { name: 'kredit_csrf', value: 'synthetic-csrf', url: baseURL ?? 'http://127.0.0.1:5173' }]);
 await page.route('**/api/v1/**', async route => {
  const path = new URL(route.request().url()).pathname;
  if (path === '/api/v1/me') return send(route, { user: { id: 'user-1' }, session: { id: 'session-1' } });
  if (path === '/api/v1/organizations') return send(route, { organizations: orgs });
  if (path === '/api/v1/pricing') return send(route, { policy_revision: 1, base_bps: 50, collection_bps: 50, min_fee_kobo: 0 });
  if (path.endsWith('/credit-requests')) return send(route, { requests: [sale('draft-1', 'DRAFT')] });
  if (path.endsWith('/customers')) return send(route, { customers: [{ buyer_user_id: 'buyer-1', buyer_business_id: 'business-1', legal_name: 'Amina Stores', trading_name: 'Amina Stores', state: 'verified' }] });
  if (path.endsWith('/payments')) return send(route, { payments: [] });
  if (path.endsWith('/payment-claims')) return send(route, { payment_claims: [] });
  if (path.endsWith('/overdue')) return send(route, { overdue: [] });
  if (path.endsWith('/disputes')) return send(route, { disputes: [] });
  if (path.endsWith('/reports/receivables')) return send(route, { summary: { obligation_count: 1, outstanding_kobo: 12750049, overdue_kobo: 0 } });
  if (path === '/api/v1/buyer/credit-requests/sale-1') return send(route, sale());
  return send(route, { code: 'not_found' }, 404);
 });
}

test('unavailable balance, payments and disputes never produce an all-clear', async ({ page, context, baseURL }) => {
 await signedIn(page, context, baseURL);
 for (const suffix of ['reports/receivables', 'payments', 'disputes']) await page.route(`**/organizations/org-a/${suffix}`, route => send(route, { code: 'financial_data_unavailable' }, 503));
 await page.goto('/app/overview');
 await expect(page.getByText('Balance unavailable', { exact: true })).toBeVisible();
 await expect(page.getByText('Reported problems unavailable', { exact: true })).toBeVisible();
 await expect(page.getByText('Nothing needs an action right now.')).toHaveCount(0);
 await expect(page.getByText('No overdue balance in this checked summary.')).toHaveCount(0);
 await expect(page.getByLabel('What you are owed').getByText('₦0.00', { exact: true })).toHaveCount(0);
});

test('switching businesses cannot show an earlier balance under the new name', async ({ page, context, baseURL }) => {
 await signedIn(page, context, baseURL);
 let finishA: (() => void) | undefined;
 await page.route('**/organizations/org-a/reports/receivables', async route => { await new Promise<void>(resolve => { finishA = resolve; }); await send(route, { summary: { obligation_count: 1, outstanding_kobo: 12345600, overdue_kobo: 0 } }).catch(() => {}); });
 await page.route('**/organizations/org-b/reports/receivables', route => send(route, { summary: { obligation_count: 2, outstanding_kobo: 9876549, overdue_kobo: 0 } }));
 await page.goto('/app/overview');
 await page.getByRole('combobox', { name: 'Business', exact: true }).selectOption('org-b');
 await expect(page.getByRole('heading', { name: 'Borno Supplies' })).toBeVisible();
 await expect(page.getByLabel('What you are owed')).toContainText('₦98,765.49');
 finishA?.();
 await expect(page.getByLabel('What you are owed')).not.toContainText('₦123,456.00');
});

test('accepting a sale does not automatically grant bank permission or announce release readiness', async ({ page, context, baseURL }) => {
 await signedIn(page, context, baseURL);
 let state = 'BUYER_REVIEWING'; let mandateCalls = 0;
 await page.route('**/api/v1/buyer/credit-requests/sale-1', route => send(route, sale('sale-1', state)));
 await page.route('**/api/v1/buyer/credit-requests/sale-1/accept', route => { state = 'BUYER_ACCEPTED'; return send(route, sale('sale-1', state)); });
 await page.route('**/api/v1/buyer/credit-requests/sale-1/mandate', route => { mandateCalls++; return send(route, { code: 'unexpected' }, 500); });
 await page.goto('/buyer/credit-requests/sale-1');
 await page.getByRole('button', { name: /Accept sale for/ }).click();
 await expect(page.getByText('Bank permission must be ready before the seller can release the goods.')).toBeVisible();
 await expect(page.getByRole('button', { name: 'Set up bank permission' })).toBeVisible();
 expect(mandateCalls).toBe(0);
 await expect(page.getByText('The seller can now arrange the goods.')).toHaveCount(0);
});

test('pending hosted permission stays pending after return parameters', async ({ page, context, baseURL }) => {
 await signedIn(page, context, baseURL);
 const current = sale('sale-1', 'BUYER_ACCEPTED'); current.mandate = { id: 'mandate-1', provider_id: 'provider-1', provider: 'mono-sweep', status: 'PENDING', authorization_url: 'https://authorise.mono.co/synthetic-example' };
 await page.route('**/api/v1/buyer/credit-requests/sale-1', route => send(route, current));
 await page.goto('/buyer/credit-requests/sale-1?success=true&status=approved');
 await expect(page.getByRole('link', { name: /Continue securely with Mono/ })).toHaveAttribute('href', 'https://authorise.mono.co/synthetic-example');
 await expect(page.getByText('Permission active', { exact: true })).toHaveCount(0);
 await page.getByRole('button', { name: 'Check permission status' }).click();
 await expect(page.getByRole('link', { name: /Continue securely with Mono/ })).toBeVisible();
});

test('untrusted hosted permission links are never rendered', async ({ page, context, baseURL }) => {
 await signedIn(page, context, baseURL);
 const current = sale('sale-1', 'BUYER_ACCEPTED'); current.mandate = { id: 'mandate-1', provider_id: 'provider-1', provider: 'mono-sweep', status: 'PENDING', authorization_url: 'https://bank-permission.evil.test/steal' };
 await page.route('**/api/v1/buyer/credit-requests/sale-1', route => send(route, current));
 await page.goto('/buyer/credit-requests/sale-1');
 await expect(page.locator('a[href*="evil.test"]')).toHaveCount(0);
 await expect(page.getByText(/The provider link is not available/)).toBeVisible();
});

test('failed OTP request leaves the form recoverable', async ({ page }) => {
 await page.route('**/api/v1/me', route => send(route, { code: 'authentication_required' }, 401));
 await page.route('**/api/v1/auth/otp/challenges', route => route.abort('failed'));
 await page.goto('/app');
 await page.getByRole('textbox', { name: 'Phone number' }).fill('08031234567');
 await page.getByRole('button', { name: 'Send me a code' }).click();
 await expect(page.getByRole('alert')).toContainText('We could not confirm that a code was sent');
 await expect(page.getByRole('button', { name: 'Send me a code' })).toBeEnabled();
});

test('OTP has an expiry and resend countdown without exposing the full target', async ({ page }) => {
 await page.route('**/api/v1/me', route => send(route, { code: 'authentication_required' }, 401));
 await page.route('**/api/v1/auth/otp/challenges', route => send(route, { challenge_id: 'challenge-1', expires_at: new Date(Date.now() + 300000).toISOString(), channel: 'sms' }, 202));
 await page.goto('/app'); await page.getByRole('textbox', { name: 'Phone number' }).fill('08031234567'); await page.getByRole('button', { name: 'Send me a code' }).click();
 await expect(page.getByRole('textbox', { name: 'Six-digit code' })).toBeVisible();
 await expect(page.getByRole('button', { name: /Resend in/ })).toBeDisabled();
 await expect(page.getByText(/This code expires in/)).toBeVisible();
 await expect(page.locator('form')).not.toContainText('+2348031234567');
});

test('failed sign-out is not presented as successful logout', async ({ page, context, baseURL }) => {
 await signedIn(page, context, baseURL);
 await page.route('**/api/v1/auth/logout', route => send(route, { code: 'service_unavailable' }, 503));
 await page.goto('/app/settings');
 await page.getByRole('button', { name: 'Sign out', exact: true }).click();
 await expect(page.getByRole('alert')).toContainText('Sign-out was not confirmed');
 await expect(page).toHaveURL(/\/app\/settings$/);
 expect((await context.cookies()).some(cookie => cookie.name === 'kredit_session')).toBe(true);
});

test('quick sale uses the server timing preview and exact kobo', async ({ page, context, baseURL }) => {
 await signedIn(page, context, baseURL);
 await page.route('**/organizations/org-a/credit-terms/preview', route => send(route, { due_date: '2026-09-18', grace_hours: 24, collection_at: '2026-09-19T22:59:00Z', timezone: 'Africa/Lagos', cutoff: '23:59', timing_mode: 'lagos_end_of_day' }));
 let body: Record<string, unknown> | undefined;
 await page.route('**/organizations/org-a/credit-requests', async route => { if (route.request().method() === 'POST') { body = route.request().postDataJSON(); return send(route, { request: { id: 'created-sale' } }, 201); } return send(route, { requests: [] }); });
 await page.goto('/app/credit/quick?organization=org-a');
 await page.getByRole('combobox', { name: 'Customer', exact: true }).selectOption('buyer-1:business-1');
 await page.getByRole('button', { name: 'Continue', exact: true }).click();
 await page.getByRole('textbox', { name: 'Goods and quantity' }).fill('40 cartons of cooking oil');
 await page.getByRole('textbox', { name: 'Sale amount (₦)' }).fill('127,500.49');
 await page.getByRole('button', { name: 'Continue', exact: true }).click();
 await page.getByLabel('Agreed payment date').fill('2026-09-18');
 await page.getByRole('button', { name: 'Continue', exact: true }).click();
 await expect(page.getByRole('heading', { name: 'Check your sale' })).toBeFocused();
 await expect(page.locator('.sale-summary')).toContainText('₦127,500.49');
 await page.getByRole('button', { name: 'Save draft sale' }).click();
 await expect.poll(() => body).toBeTruthy();
 expect(body).toMatchObject({ principal_kobo: 12750049, collection_at: '2026-09-19T22:59:00Z', timing_mode: 'lagos_end_of_day', schedule_count: 1 });
});

test('exact typed pricing supports voluntary and partial repayment examples', async ({ page }) => {
 await page.route('**/api/v1/pricing', route => send(route, { policy_revision: 1, base_bps: 50, collection_bps: 50 }));
 await page.goto('/pricing');
 await page.getByRole('textbox', { name: 'Sale amount (₦)', exact: true }).fill('127,500.49');
 const example = page.getByLabel('Fee example');
 await expect(example).toContainText('₦127,500.49'); await expect(example).toContainText('₦637.50');
 await page.getByRole('textbox', { name: 'Amount Kredit successfully collects (₦)' }).fill('40,000.25');
 await expect(example).toContainText('₦200.00'); await expect(example).toContainText('₦837.50');
});

test('payment decisions require a bank check and preserve their request identity', async ({ page, context, baseURL }) => {
 await signedIn(page, context, baseURL);
 const claim = { id: 'claim-1', state: 'pending', amount_kobo: 10049, transfer_reference: 'TEST-TRANSFER-1', paid_at: '2026-09-06T12:00:00Z', hold_expires_at: '2026-09-07T12:00:00Z' };
 await page.route('**/organizations/org-a/payment-claims', route => send(route, { payment_claims: [claim] }));
 const keys: string[] = [];
 await page.route('**/payment-claims/claim-1/decide', route => { keys.push(route.request().headers()['idempotency-key']); return keys.length === 1 ? route.abort('failed') : send(route, { payment_claim: { ...claim, state: 'confirmed' } }); });
 await page.goto('/app/payments'); await page.getByRole('button', { name: 'Yes, I got the money' }).click();
 const dialog = page.getByRole('dialog', { name: 'Confirm money received' });
 await expect(dialog).toContainText('₦100.49'); await expect(dialog.getByRole('button', { name: 'Confirm received' })).toBeDisabled();
 await dialog.getByRole('checkbox').check(); await dialog.getByRole('button', { name: 'Confirm received' }).click();
 await expect(dialog.getByRole('alert')).toContainText('We have not confirmed the result');
 await dialog.getByRole('button', { name: 'Confirm received' }).click();
 await expect(dialog).toHaveCount(0); expect(keys).toHaveLength(2); expect(keys[0]).toBe(keys[1]);
});

test('native account menu traps focus and restores it on Escape', async ({ page, context, baseURL }) => {
 await signedIn(page, context, baseURL); await page.setViewportSize({ width: 390, height: 844 });
 await page.goto('/app/settings');
 const trigger = page.getByRole('navigation', { name: 'Seller account', exact: true }).getByRole('button', { name: 'Menu', exact: true });
 await trigger.click(); const dialog = page.getByRole('dialog', { name: 'Seller account menu' });
 await expect(dialog).toBeVisible();
 for (let i = 0; i < 20; i++) { await page.keyboard.press('Tab'); expect(await dialog.evaluate(node => node.contains(document.activeElement))).toBe(true); }
 await page.keyboard.press('Escape'); await expect(trigger).toBeFocused();
});

for (const [label, path] of [['home', '/'], ['pricing', '/pricing'], ['supplier', '/app/overview'], ['quick-sale', '/app/credit/quick'], ['full-sale', '/app/credit/new'], ['buyer', '/buyer/credit-requests/sale-1']] as const) {
 test(`visual and accessibility evidence: ${label}`, async ({ page, context, baseURL }, testInfo) => {
  await signedIn(page, context, baseURL);
  await page.emulateMedia({ reducedMotion: 'reduce' });
  for (const width of [390, 1440]) {
   await page.setViewportSize({ width, height: width === 390 ? 844 : 1000 }); await page.goto(path);
   await expect(page.locator('h1')).toBeVisible();
   if (path.startsWith('/app/') || path.startsWith('/buyer/')) await expect(page.locator('.account-gate')).toHaveCount(0);
   await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
   const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa', 'wcag22aa']).analyze();
   expect(results.violations.filter(item => item.impact === 'serious' || item.impact === 'critical'), JSON.stringify(results.violations.map(item => ({ id: item.id, nodes: item.nodes.map(node => node.target) })))).toEqual([]);
   await page.screenshot({ path: testInfo.outputPath(`${label}-${width}.png`), fullPage: true });
  }
 });
}


test('full sale keeps business identity and server-reviewed timing on the invoice path', async ({ page, context, baseURL }) => {
 await signedIn(page, context, baseURL);
 await page.route('**/api/v1/organizations/org-a/customers', route => send(route, { customers: [
  { buyer_user_id:'buyer-1', buyer_business_id:'business-1', legal_name:'First shop', trading_name:'First shop', state:'verified' },
  { buyer_user_id:'buyer-1', buyer_business_id:'business-2', legal_name:'Second shop', trading_name:'Second shop', state:'verified' }
 ] }));
 await page.route('**/api/v1/organizations/org-a/credit-terms/preview', route => send(route, { due_date:'2026-09-18', grace_hours:24, collection_at:'2026-09-19T22:59:00Z', timezone:'Africa/Lagos', cutoff:'23:59', timing_mode:'lagos_end_of_day' }));
 const saved: Record<string, unknown>[] = [];
 await page.route('**/api/v1/organizations/org-a/credit-requests', route => { if(route.request().method()==='POST'){saved.push(route.request().postDataJSON());return send(route,{request:{id:'created-sale'}},201);}return send(route,{requests:[]}); });
 await page.goto('/app/credit/new?organization=org-a');
 await page.getByRole('combobox',{name:'Customer',exact:true}).selectOption('buyer-1:business-2');
 await page.getByRole('textbox',{name:'Sale amount (₦)'}).fill('127,500.49');
 await page.getByRole('textbox',{name:'What goods are they taking?'}).fill('40 cartons of cooking oil');
 await page.getByLabel('First payment date').fill('2026-09-18');
 await page.getByRole('button',{name:'Check terms',exact:true}).click();
 await expect(page.getByRole('button',{name:'Save draft sale',exact:true})).toBeEnabled();
 expect(saved).toHaveLength(0);
 await expect(page.locator('.review')).toContainText('₦127,500.49');
 await page.getByRole('button',{name:'Save draft sale',exact:true}).click();
 await expect.poll(()=>saved.length).toBe(1);
 expect(saved[0]).toMatchObject({buyer_user_id:'buyer-1',buyer_business_id:'business-2',principal_kobo:12750049,collection_at:'2026-09-19T22:59:00Z',timing_mode:'lagos_end_of_day'});
});

test('full sale never presents an unavailable customer list as empty', async ({page,context,baseURL}) => {
 await signedIn(page,context,baseURL);
 await page.route('**/api/v1/organizations/org-a/customers', route=>send(route,{code:'financial_data_unavailable'},503));
 await page.goto('/app/credit/new');
 await expect(page.getByRole('alert')).toContainText('Customer list unavailable');
 await expect(page.getByText('You have not added a customer yet.',{exact:true})).toHaveCount(0);
 await expect(page.getByRole('button',{name:'Check terms',exact:true})).toBeDisabled();
});
