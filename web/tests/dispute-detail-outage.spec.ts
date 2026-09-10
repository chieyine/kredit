import { test, expect } from '@playwright/test';

for (const area of ['app', 'buyer']) {
  test(`${area} dispute evidence retries preserve the submitted request`, async ({ page, context, baseURL }) => {
    await context.addCookies([{ name: 'kredit_session', value: 'synthetic-dispute-session', url: baseURL! }]);
    await page.route('**/api/v1/me', route => route.fulfill({ json: { user: { id: 'dispute-user' } } }));
    const endpoint = area === 'buyer' ? '/api/v1/buyer/disputes/problem-1' : '/api/v1/organizations/org-a/disputes/problem-1';
    await page.route(`**${endpoint}`, route => route.fulfill({ json: {
      dispute: { id: 'problem-1', state: 'OPEN', total_disputed_kobo: 10000, remaining_disputed_kobo: 10000, collection_effect: 'CONTESTED_ONLY', reason: 'Missing cartons', opened_at: '2026-09-09T10:00:00Z' },
      evidence: [], decisions: []
    } }));
    const sent: { key: string; body: unknown }[] = [];
    await page.route(`**${endpoint}/evidence`, route => {
      sent.push({ key: route.request().headers()['idempotency-key'], body: route.request().postDataJSON() });
      return sent.length === 1 ? route.abort('failed') : route.fulfill({ status: 201, json: { evidence: { id: 'evidence-1' } } });
    });
    await page.goto(`/${area}/disputes/problem-1?organization=org-a`);
    await page.getByRole('textbox', { name: 'What else should we know?' }).fill('Two cartons were missing from the delivery.');
    const submit = page.getByRole('button', { name: 'Add this information', exact: true });
    await submit.click();
    await expect(page.getByRole('alert')).toContainText('We have not confirmed the result');
    await expect(submit).toBeEnabled();
    await submit.click();
    await expect(page.getByRole('status').filter({ hasText: 'Information added.' })).toBeVisible();
    expect(sent).toHaveLength(2);
    expect(sent[0].key).toBeTruthy();
    expect(sent[1]).toEqual(sent[0]);
  });
}

test('admin dispute decision keeps the same identity after an unknown result', async ({ page, context, baseURL }) => {
  await context.addCookies([{ name: 'kredit_session', value: 'synthetic-admin-session', url: baseURL! }]);
  await page.route('**/api/v1/me', route => route.fulfill({ json: { user: { id: 'reviewer' } } }));
  const endpoint = '/api/v1/ops/disputes/problem-1';
  await page.route(`**${endpoint}`, route => route.fulfill({ json: {
    dispute: { id: 'problem-1', state: 'OPEN', total_disputed_kobo: 10000, remaining_disputed_kobo: 10000, collection_effect: 'CONTESTED_ONLY', reason: 'Missing cartons' },
    evidence: [], decisions: []
  } }));
  const sent: { key: string; body: unknown }[] = [];
  await page.route(`**${endpoint}/decide`, route => {
    sent.push({ key: route.request().headers()['idempotency-key'], body: route.request().postDataJSON() });
    return sent.length === 1 ? route.abort('failed') : route.fulfill({ json: { dispute: { id: 'problem-1' }, decision: { id: 'decision-1' } } });
  });
  await page.goto('/admin/disputes/problem-1');
  await page.getByRole('textbox', { name: 'Reason', exact: true }).fill('The buyer and supplier evidence confirms this amount.');
  const submit = page.getByRole('button', { name: 'Record this decision', exact: true });
  await submit.click();
  await expect(page.getByRole('alert')).toContainText('We have not confirmed the result');
  await expect(submit).toBeEnabled();
  await submit.click();
  await expect(page.getByRole('status')).toContainText('Decision recorded.');
  expect(sent).toHaveLength(2);
  expect(sent[0].key).toBeTruthy();
  expect(sent[1]).toEqual(sent[0]);
});
