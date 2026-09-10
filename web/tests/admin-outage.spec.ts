import { expect, test } from '@playwright/test';

const json = (body: unknown) => ({ status: 200, contentType: 'application/json', body: JSON.stringify(body) });
test.beforeEach(async ({ page, context, baseURL }) => {
  await context.addCookies([{ name: 'kredit_session', value: 'audit-session', url: baseURL! }]);
  await page.route('**/api/v1/me', route => route.fulfill(json({ user: { id: 'audit-admin' }, organizations: [] })));
  await page.route('**/api/v1/ops/attention', route => route.fulfill(json({ items: [] })));
});

test('analytics rejects incomplete metrics and recovers without a page crash', async ({ page }) => {
	const errors: string[] = [];
	page.on('pageerror', error => errors.push(error.message));
	const card = {
		generated_at: '2026-09-09T10:00:00Z', from: '2026-09-01T00:00:00Z', to: '2026-09-09T00:00:00Z',
		refresh_mode: 'live query', reconciliation_ok: true,
		kpis: [], drivers: [], guardrails: [], reconciliation: [],
		feedback: { total: 0, yes: 0, partly: 0, no: 0, seller: 0, buyer: 0, clear_percent: 0 }
	};
	let state = 'missing-feedback';
	await page.route('**/api/v1/ops/analytics/scorecard?*', route => route.fulfill(json({ scorecard:
		state === 'missing-feedback' ? { ...card, feedback: null } : state === 'invalid-number'
			? { ...card, kpis: [{ key: 'gross_trade_credit_volume', label: 'Trade', value: '100', unit: 'kobo', definition: 'Recorded', source: 'records' }] }
			: card
	})));
	await page.goto('/admin/analytics');
	await expect(page.getByRole('alert')).toContainText('We could not check the scorecard');
	state = 'invalid-number';
	await page.getByRole('button', { name: 'Try again', exact: true }).click();
	await expect(page.getByRole('alert')).toContainText('We could not check the scorecard');
	state = 'ready';
	await page.getByRole('button', { name: 'Try again', exact: true }).click();
	await expect(page.getByText('No answers yet', { exact: true })).toBeVisible();
	await expect(page.getByRole('alert')).toHaveCount(0);
	expect(errors).toEqual([]);
});

test('admin lists reject malformed entries before rendering', async ({ page }) => {
	const errors: string[] = [];
	page.on('pageerror', error => errors.push(error.message));
	for (const [path, endpoint, key] of [['cases', 'cases?state=', 'cases'], ['disputes', 'disputes?state=', 'disputes'], ['audit', 'audit', 'events']]) {
		await page.route(`**/api/v1/ops/${endpoint}`, route => route.fulfill(json({ [key]: [{ id: 'incomplete' }] })));
		await page.goto(`/admin/${path}`);
		await expect(page.getByRole('alert')).toBeVisible();
	}
	expect(errors).toEqual([]);
});

test('support case update retries keep the original request', async ({ page }) => {
	const endpoint = '/api/v1/ops/cases/support-1';
	const item = { id: 'support-1', state: 'OPEN', subject_type: 'payment', created_at: '2026-09-09T10:00:00Z' };
	const sent: { key: string; body: unknown }[] = [];
	await page.route(`**${endpoint}`, route => {
		if (route.request().method() === 'GET') return route.fulfill(json({ case: item, timeline: [] }));
		sent.push({ key: route.request().headers()['idempotency-key'], body: route.request().postDataJSON() });
		return sent.length === 1 ? route.abort('failed') : route.fulfill(json({ case: item }));
	});
	await page.goto('/admin/cases/support-1');
	await page.getByRole('textbox', { name: 'What happened?' }).fill('The customer provided the requested receipt.');
	const submit = page.getByRole('button', { name: 'Save case update', exact: true });
	await submit.click();
	await expect(page.getByRole('status')).toContainText('We have not confirmed the result');
	await submit.click();
	await expect(page.getByRole('status')).toContainText('Case updated.');
	expect(sent).toHaveLength(2);
	expect(sent[0].key).toBeTruthy();
	expect(sent[1]).toEqual(sent[0]);
});

for (const [path, endpoint, key, empty] of [
  ['jobs', 'jobs', 'jobs', 'No jobs found'],
  ['privacy', 'privacy-requests', 'requests', 'No privacy requests'],
  ['recovery', 'account-recovery?state=PENDING_REVIEW', 'requests', 'Nothing waiting for review'],
  ['provider-events', 'provider-events', 'events', 'No provider events found']
]) {
  test(`${path}: an outage or malformed response cannot become an empty queue`, async ({ page }) => {
    let state = 'outage';
    await page.route(`**/api/v1/ops/${endpoint}`, route => state === 'outage'
      ? route.fulfill({ status: 503, contentType: 'application/json', body: '{}' })
      : state === 'malformed' ? route.fulfill(json({})) : route.fulfill(json({ [key]: [] })));
    await page.goto(`/admin/${path}`);
    await expect(page.getByRole('alert')).toContainText('We could not check');
    await expect(page.getByRole('heading', { name: empty, exact: true })).toHaveCount(0);
    state = 'malformed';
    await page.getByRole('button', { name: 'Try again', exact: true }).click();
    await expect(page.getByRole('alert')).toContainText('We could not check');
    await expect(page.getByRole('heading', { name: empty, exact: true })).toHaveCount(0);
    state = 'ready';
    await page.getByRole('button', { name: 'Try again', exact: true }).click();
    await expect(page.getByRole('heading', { name: empty, exact: true })).toBeVisible();
    await expect(page.getByRole('alert')).toHaveCount(0);
  });
}

test('unavailable owner settings never claim solo-owner approval is enabled', async ({ page }) => {
  await page.route('**/api/v1/ops/capabilities', route => route.fulfill(json({ roles: ['platform_owner'] })));
  await page.route('**/api/v1/ops/platform-settings', route => route.fulfill({ status: 503, contentType: 'application/json', body: '{}' }));
  await page.goto('/admin/platform-settings');
  await expect(page.getByText('Approval rules unavailable', { exact: true })).toBeVisible();
  await expect(page.getByText('You run Kredit alone', { exact: true })).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Change this', exact: true })).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Try again', exact: true })).toBeVisible();
});

test('connector configuration sends an atomic replacement and clears credentials after closing', async ({ page }) => {
  const setting = { key: 'integrations.notifications.sms', category: 'integrations', description: 'SMS delivery connector', is_secret: true, value: '', version: 0 };
  let saved: any;
  await page.route('**/api/v1/ops/capabilities', route => route.fulfill(json({ roles: ['platform_owner'] })));
  await page.route('**/api/v1/ops/platform-settings', route => {
    if (route.request().method() === 'POST') { saved = route.request().postDataJSON(); return route.fulfill(json({ setting: { ...setting, version: 1 } })); }
    return route.fulfill(json({ settings: [setting], governance: { mode: 'solo_owner' } }));
  });
  await page.goto('/admin/platform-settings');
  await page.getByRole('button', { name: 'Change', exact: true }).click();
  await page.getByLabel('Connector HTTPS address').fill('https://connector.example/send');
  await page.getByLabel('Connector access token').fill('private-connector-token');
  await page.getByLabel('Why are you making this change?').fill('Connect SMS for launch');
  await page.getByRole('button', { name: 'Save this change', exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  expect(saved.expected_version).toBe(0);
  expect(JSON.parse(saved.value)).toEqual({ enabled: true, endpoint: 'https://connector.example/send', token: 'private-connector-token' });
  await page.getByRole('button', { name: 'Change', exact: true }).click();
  await expect(page.getByLabel('Connector access token')).toHaveValue('');
});

test('platform setting save requires confirmation and reuses the original request', async ({ page }) => {
	const setting = { key: 'features.sample', category: 'features', description: 'Sample feature', value: false, version: 2 };
	const sent: { key: string; body: unknown }[] = [];
	await page.route('**/api/v1/ops/capabilities', route => route.fulfill(json({ roles: ['platform_owner'] })));
	await page.route('**/api/v1/ops/platform-settings', route => {
		if (route.request().method() === 'GET') return route.fulfill(json({ settings: [setting], governance: { mode: 'solo_owner' } }));
		sent.push({ key: route.request().headers()['idempotency-key'], body: route.request().postDataJSON() });
		return route.fulfill(json(sent.length === 1 ? {} : { setting: { ...setting, value: true, version: 3 } }));
	});
	await page.goto('/admin/platform-settings');
	await page.getByRole('button', { name: 'Change', exact: true }).click();
	const dialog = page.getByRole('dialog');
	await dialog.getByRole('combobox').selectOption({ label: 'On' });
	await dialog.getByLabel('Why are you making this change?').fill('Enable this feature after launch review');
	await dialog.getByRole('button', { name: 'Save this change', exact: true }).click();
	await expect(dialog.getByRole('alert')).toContainText('We have not confirmed the result');
	await dialog.getByRole('button', { name: 'Save this change', exact: true }).click();
	await expect(dialog).toHaveCount(0);
	expect(sent).toHaveLength(2);
	expect(sent[1]).toEqual(sent[0]);
});

test('editing a protected action invalidates its impact preview', async ({ page }) => {
  await page.route('**/api/v1/ops/commands/preview', route => route.fulfill(json({ command: { current_version: 1, impact_preview: { effect: 'Suspend this user', will_notify: true, audit: 'Required' } } })));
  await page.goto('/admin/controls?target_type=user&target_id=first-user');
  await page.getByLabel('Structured reason').fill('Investigated account abuse');
  await page.getByRole('button', { name: 'Preview impact', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Apply this change', exact: true })).toBeVisible();
  await page.getByLabel('Target ID', { exact: true }).fill('different-user');
  await expect(page.getByRole('button', { name: 'Apply this change', exact: true })).toHaveCount(0);
});

test('business and suspended-account links select the matching protected action', async ({ page }) => {
	await page.goto('/admin/controls?target_type=organization&target_id=business&organization_id=business&version=7&status=active');
	await expect(page.getByRole('combobox', { name: 'Action', exact: true })).toHaveValue('suspend_organization');
	await expect(page.getByLabel('Current version')).toHaveValue('7');
	await page.goto('/admin/controls?target_type=organization&target_id=business&status=suspended');
	await expect(page.getByRole('combobox', { name: 'Action', exact: true })).toHaveValue('restore_organization');
	await page.goto('/admin/controls?target_type=user&target_id=customer&status=suspended');
	await expect(page.getByRole('combobox', { name: 'Action', exact: true })).toHaveValue('restore_user');
});

test('user directory opens the requested account and carries its current version into controls', async ({ page }) => {
	let requested = '';
	await page.route('**/api/v1/ops/users?*', route => {
		requested = new URL(route.request().url()).searchParams.get('q') || '';
		return route.fulfill(json({ users: [{ id: 'customer', display_name: 'Test customer', identifier: 'test@example.test', status: 'suspended', version: 9, organization_count: 1 }] }));
	});
	await page.goto('/admin/users?q=customer');
	await expect(page.getByRole('link', { name: 'Manage →', exact: true })).toBeVisible();
	expect(requested).toBe('customer');
	await page.getByRole('link', { name: 'Manage →', exact: true }).click();
	await expect(page.getByRole('combobox', { name: 'Action', exact: true })).toHaveValue('restore_user');
	await expect(page.getByLabel('Current version')).toHaveValue('9');
});

for (const [path, endpoint, empty] of [
  ['disputes', 'disputes?state=', 'No disputes in this view'],
  ['reconciliation', 'financial-reconciliation', 'No open financial reviews.'],
  ['audit', 'audit', 'No entries yet']
]) {
  test(`${path}: malformed data is not an empty result`, async ({ page }) => {
    await page.route(`**/api/v1/ops/${endpoint}`, route => route.fulfill(json({})));
    await page.goto(`/admin/${path}`);
    await expect(page.getByRole('alert')).toBeVisible();
    await expect(page.getByText(empty, { exact: true })).toHaveCount(0);
  });
}

test('Mono settings are editable without revealing credentials and clearly require a restart', async ({ page }) => {
  let saved: any;
  const setting: any = {
    key: 'integrations.runtime.mono', category: 'integrations', description: 'Mono bank collections',
    is_secret: true, value: '••••••••', version: 3, requires_restart: true,
    connection_state: 'applied_unverified', applied_version: 3,
    connection_fields: [
      { key: 'MonoSweepEnabled', label: 'Enable new Mono collections', kind: 'boolean' },
      { key: 'MonoSecretKey', label: 'Mono secret key', kind: 'password' },
      { key: 'MonoRedirectURL', label: 'Customer return HTTPS address', kind: 'url' }
    ], connection_values: { MonoSweepEnabled: true, MonoRedirectURL: 'https://app.kredit.test/return' }
  };
  await page.route('**/api/v1/ops/capabilities', route => route.fulfill(json({ roles: ['platform_owner'] })));
  await page.route('**/api/v1/ops/platform-settings', route => {
    if (route.request().method() === 'POST') {
      saved = route.request().postDataJSON(); setting.version = 4; setting.connection_state = 'restart_required';
      return route.fulfill(json({ setting }));
    }
    return route.fulfill(json({ settings: [setting], governance: { mode: 'solo_owner' } }));
  });
  await page.goto('/admin/platform-settings');
  await page.getByRole('button', { name: 'Change', exact: true }).click();
  await expect(page.getByLabel('Mono secret key', { exact: true })).toHaveValue('');
  await expect(page.getByLabel('Customer return HTTPS address')).toHaveValue('https://app.kredit.test/return');
  await page.getByLabel('Enable new Mono collections').selectOption({ label: 'Disabled' });
  await page.getByLabel('Why are you making this change?').fill('Pause new collections while retaining reconciliation');
  await page.getByRole('button', { name: 'Save this change', exact: true }).click();
  await expect(page.getByText('Saved · restart API and worker', { exact: true })).toBeVisible();
  expect(saved.expected_version).toBe(3);
  expect(saved.clear_credentials).toBe(false);
  expect(JSON.parse(saved.value)).toEqual({ MonoSweepEnabled: false, MonoSecretKey: '', MonoRedirectURL: 'https://app.kredit.test/return' });
});
