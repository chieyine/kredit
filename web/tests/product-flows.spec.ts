import { expect, test } from '@playwright/test';

test.beforeEach(async ({ page, context, baseURL }) => {
	await context.addCookies([{ name: 'kredit_session', value: 'product-test-session', url: baseURL ?? 'http://127.0.0.1:5173' }]);
	await page.route('**/api/v1/me', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ user: { id: 'product-user', status: 'active', created_at: '2026-01-01T00:00:00Z' }, session: { id: 'product-session', user_id: 'product-user', authentication_level: 'AAL1', created_at: '2026-01-01T00:00:00Z', expires_at: '2027-01-01T00:00:00Z' }, mfa_enrolled: false, organizations: [] }) }));
});

async function mockOperationsCommand(page: import('@playwright/test').Page, applied: Record<string, unknown>[]) {
	await page.route('**/api/v1/ops/commands/preview', async (route) => {
		const input = route.request().postDataJSON();
		await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ command: { ...input, current_version: input.expected_version, impact_preview: { effect: `Safely apply ${input.command_type}`, will_notify: true, audit: 'immutable' } } }) });
	});
	await page.route('**/api/v1/ops/commands', async (route) => {
		const input = route.request().postDataJSON(); applied.push(input);
		await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ command: { id: `command-${applied.length}`, ...input, state: 'APPLIED' } }) });
	});
}

test('operator previews and safely retries one failed job', async ({ page }) => {
	const applied:Record<string,unknown>[]=[];await mockOperationsCommand(page,applied);
	await page.route('**/api/v1/ops/jobs',async(route)=>route.fulfill({status:200,contentType:'application/json',body:JSON.stringify({jobs:[{id:41,kind:'kredit.financial',queue:'critical-financial',state:'retryable',attempt:2,max_attempts:12}]})}));
	await page.goto('/admin/jobs');await page.getByRole('button',{name:'Preview safe retry'}).click();await page.getByLabel('Reason for this action').fill('Provider incident has cleared');await page.getByRole('button',{name:'Preview impact'}).click();await expect(page.getByText('Verified impact preview')).toBeVisible();await page.getByRole('button',{name:'Confirm safe retry'}).click();await expect(page.getByText('Job safely requeued.')).toBeVisible();expect(applied[0]).toMatchObject({command_type:'retry_job',target_id:'41',expected_version:3});
});

test('operator replays a failed webhook without changing its identity',async({page})=>{
	const applied:Record<string,unknown>[]=[];await mockOperationsCommand(page,applied);
	await page.route('**/api/v1/ops/provider-events',async(route)=>route.fulfill({status:200,contentType:'application/json',body:JSON.stringify({events:[{provider:'certified-provider',event_id:'evt-1',event_type:'collection.updated',state:'failed',attempts:1}]})}));
	await page.goto('/admin/provider-events');await page.getByRole('button',{name:'Preview safe replay'}).click();await page.getByLabel('Reason for this action').fill('Verified provider outage ended');await page.getByRole('button',{name:'Preview impact'}).click();await expect(page.getByText('Verified impact preview')).toBeVisible();await page.getByRole('button',{name:'Confirm safe replay'}).click();await expect(page.getByText('Webhook safely requeued.')).toBeVisible();expect(applied[0]).toMatchObject({command_type:'retry_webhook',target_id:'evt-1',expected_version:2});
});

test('operator resolves an unknown provider submission through protected controls',async({page})=>{
	const applied:Record<string,unknown>[]=[];await mockOperationsCommand(page,applied);await page.goto('/admin/controls');await expect(page.locator('form[data-ready="true"]')).toBeVisible();await page.getByLabel('Command').selectOption('resolve_unknown_submission');await page.getByLabel('Target type').fill('collection');await page.getByLabel('Target ID').fill('00000000-0000-0000-0000-000000000101');await page.getByLabel('Structured reason').fill('Provider reconciliation confirms final state');await page.getByRole('button',{name:'Preview impact'}).click();await expect(page.getByText('Safely apply resolve_unknown_submission')).toBeVisible();await page.getByRole('button',{name:'Apply protected command'}).click();expect(applied[0]).toMatchObject({command_type:'resolve_unknown_submission'});
});

test('operator previews user suspension and restoration consequences',async({page})=>{
	const applied:Record<string,unknown>[]=[];await mockOperationsCommand(page,applied);await page.goto('/admin/controls');await expect(page.locator('form[data-ready="true"]')).toBeVisible();await page.getByLabel('Target ID').fill('00000000-0000-0000-0000-000000000202');await page.getByLabel('Structured reason').fill('Confirmed account compromise investigation');await page.getByRole('button',{name:'Preview impact'}).click();await page.getByRole('button',{name:'Apply protected command'}).click();await page.getByLabel('Command').selectOption('restore_user');await page.getByLabel('Target ID').fill('00000000-0000-0000-0000-000000000203');await page.getByLabel('Structured reason').fill('Security investigation completed safely');await page.getByRole('button',{name:'Preview impact'}).click();await page.getByRole('button',{name:'Apply protected command'}).click();expect(applied.map(x=>x.command_type)).toEqual(['suspend_user','restore_user']);
});

test('operator places an expiring scoped buyer risk hold',async({page})=>{
	const applied:Record<string,unknown>[]=[];await mockOperationsCommand(page,applied);await page.goto('/admin/controls');await expect(page.locator('form[data-ready="true"]')).toBeVisible();await page.getByLabel('Command').selectOption('place_risk_hold');await page.getByLabel('Target type').fill('buyer');await page.getByLabel('Target ID').fill('00000000-0000-0000-0000-000000000301');await page.getByLabel('Scope').selectOption('collection');await page.getByLabel('Expires').fill('2026-08-30T12:00');await page.getByLabel('Structured reason').fill('Collection anomaly requires compliance review');await page.getByRole('button',{name:'Preview impact'}).click();await expect(page.getByText('User notification',{exact:true})).toBeVisible();await page.getByRole('button',{name:'Apply protected command'}).click();expect(applied[0]).toMatchObject({command_type:'place_risk_hold',target_type:'buyer',scope:'collection'});
});

test('supplier can create exact credit terms with a replay-safe request', async ({ page }) => {
	let submitted: Record<string, unknown> | undefined;
	let idempotency = '';
	await page.route('**/api/v1/organizations', async (route) => route.fulfill({
		status: 200, contentType: 'application/json', body: JSON.stringify({ organizations: [{ id: 'org-1', legal_name: 'Adebayo Supplies', trading_name: 'Adebayo' }] })
	}));
	await page.route('**/api/v1/organizations/org-1/customers', async (route) => route.fulfill({
		status: 200, contentType: 'application/json', body: JSON.stringify({ customers: [{ buyer_user_id: 'buyer-1', buyer_business_id: 'business-1', legal_name: 'Kano Retail Limited', trading_name: 'Kano Retail', state: 'VERIFIED' }] })
	}));
	await page.route('**/api/v1/organizations/org-1/credit-requests', async (route) => {
		submitted = route.request().postDataJSON();
		idempotency = route.request().headers()['idempotency-key'];
		await route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify({ request: { id: 'request-1' } }) });
	});
	await page.route('**/api/v1/organizations/org-1/credit-requests/request-1', async (route) => route.fulfill({ status: 404, contentType: 'application/json', body: '{}' }));
	await page.goto('/app/credit/new');
	await page.getByRole('combobox', { name: 'Customer', exact: true }).selectOption('buyer-1');
	await page.getByLabel('Money to pay (₦)').fill('1,200,000');
	await page.getByLabel('What goods did they take?').fill('Twenty cartons of verified inventory');
	await page.getByLabel('First payment day').fill('2026-09-30');
	await page.getByLabel('Day Kredit may debit if unpaid').fill('2026-10-02T09:00');
	await page.getByRole('button', { name: 'Save this sale' }).click();
	await expect(page).toHaveURL(/\/app\/credit\/request-1\?organization=org-1/);
	expect(idempotency.length).toBeGreaterThanOrEqual(8);
	await expect.poll(() => submitted).toMatchObject({ buyer_user_id: 'buyer-1', buyer_business_id: 'business-1', principal_kobo: 120000000, goods_description: 'Twenty cartons of verified inventory' });
});

test('operations overview presents redacted health counters', async ({ page }) => {
	await page.route('**/api/v1/ops/overview', async (route) => route.fulfill({
		status: 200, contentType: 'application/json', body: JSON.stringify({ role: 'platform_admin', overview: { queued_jobs: 2, failed_jobs: 0, dead_letter_jobs: 0, pending_outbox: 1, failed_outbox: 0, provider_failures: 0, open_cases: 3, open_disputes: 1 } })
	}));
	await page.goto('/admin');
	await expect(page.getByRole('heading', { name: 'Controlled support and compliance access.' })).toBeVisible();
	await expect(page.getByText('Queued work')).toBeVisible();
	await expect(page.getByText('Active role: platform admin')).toBeVisible();
});

test('pilot scorecard shows definitions, guardrails, filters, and reconciliation state', async ({ page }) => {
	let requestedOrganization = '';
	await page.route('**/api/v1/ops/analytics/scorecard**', async (route) => { requestedOrganization = new URL(route.request().url()).searchParams.get('organization_id') ?? ''; await route.fulfill({
		status: 200, contentType: 'application/json', body: JSON.stringify({ scorecard: {
			generated_at: '2026-08-29T12:00:00Z', refresh_mode: 'live query', reconciliation_ok: true,
			kpis: [{ key:'gross_trade_credit_volume', label:'Gross trade credit activated', value:120000000, unit:'kobo', definition:'Sum of principal for obligations activated in the window.', source:'app.obligations' }],
			drivers: [{ key:'sent_to_acceptance', label:'Sent-to-acceptance conversion', value:75, unit:'percent', definition:'Accepted agreements divided by sent events.', source:'transition evidence' }],
			guardrails: [{ key:'dispute_rate', label:'Dispute rate', value:1.2, unit:'percent', definition:'Obligations with a dispute divided by activated obligations.', source:'app.disputes + app.obligations' }],
			reconciliation: [{ event:'obligation.activated', source_count:4, event_count:4, status:'reconciled' }]
		} })
	}); });
	await page.goto('/admin/analytics');
	await expect(page.getByRole('heading', {name:'Know whether the pilot is healthy.'})).toBeVisible();
	await expect(page.getByText('Gross trade credit activated')).toBeVisible();
	await expect(page.getByText('Dispute rate')).toBeVisible();
	await expect(page.getByText('Reconciled', {exact:true})).toBeVisible();
	await page.getByLabel('Supplier organisation UUID (optional)').fill('00000000-0000-7000-8000-000000000001');
	await page.getByRole('button', {name:'Apply filters'}).click();
	await expect.poll(()=>requestedOrganization).toBe('00000000-0000-7000-8000-000000000001');
});

test('supplier can amend and cancel a draft before immutable terms are sent', async ({ page }) => {
	let amendment: Record<string, unknown> | undefined;
	let cancelled = false;
	const draft = { id: 'request-1', state: 'DRAFT', version: 1, buyer_legal_name: 'Kano Retail Limited', principal_kobo: 120000000, goods_description: 'Initial inventory', due_date: '2026-09-30', collection_at: '2026-10-02T08:00:00Z', grace_hours: 48 };
	await page.route('**/api/v1/organizations', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ organizations: [{ id: 'org-1', legal_name: 'Adebayo Supplies' }] }) }));
	await page.route('**/api/v1/organizations/org-1/credit-requests/request-1', async (route) => {
		if (route.request().method() === 'PATCH') { amendment = route.request().postDataJSON(); draft.version = 2; draft.principal_kobo = Number(amendment?.principal_kobo); }
		await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(route.request().method() === 'PATCH' ? { request: draft } : { request: draft, receipts: [] }) });
	});
	await page.route('**/api/v1/organizations/org-1/credit-requests/request-1/cancel', async (route) => { cancelled = true; draft.state = 'CANCELLED'; await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ request: draft }) }); });
	await page.goto('/app/credit/request-1?organization=org-1');
	await page.getByLabel('Money to pay (₦)').fill('1,250,000');
	await page.getByRole('button', { name: 'Save for later' }).click();
	await expect.poll(() => amendment).toMatchObject({ expected_version: 1, principal_kobo: 125000000 });
	await page.getByRole('button', { name: 'Delete this sale' }).click();
	await expect.poll(() => cancelled).toBe(true);
});

test('buyer can decline exact terms without creating an obligation', async ({ page }) => {
	let declined = false;
	const request = { id: 'request-2', state: 'BUYER_REVIEWING', supplier_legal_name: 'Adebayo Supplies', principal_kobo: 50000000, currency: 'NGN', goods_description: 'Verified inventory', due_date: '2026-09-30' };
	await page.route('**/api/v1/buyer/credit-requests/request-2', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ request, agreement: { document_hash: 'agreement-hash' } }) }));
	await page.route('**/api/v1/buyer/credit-requests/request-2/payments', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ payments: [] }) }));
	await page.route('**/api/v1/buyer/credit-requests/request-2/decline', async (route) => { declined = true; request.state = 'DECLINED'; await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ request }) }); });
	await page.goto('/buyer/credit-requests/request-2');
	await page.getByRole('button', { name: 'No, I do not agree' }).click();
	await expect(page.getByText('You said no. This sale will not start.')).toBeVisible();
	await expect.poll(() => declined).toBe(true);
});

test('buyer payment claim explains and applies a bounded hold', async ({ page }) => {
	let submitted: Record<string, unknown> | undefined;
	const request = { id: 'request-3', state: 'ACTIVE', supplier_legal_name: 'Adebayo Supplies', principal_kobo: 50000000, currency: 'NGN', goods_description: 'Verified inventory', due_date: '2026-09-30' };
	await page.route('**/api/v1/buyer/credit-requests/request-3', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ request, obligation: { id: 'obligation-3', outstanding_kobo: 50000000 }, agreement: { document_hash: 'agreement-hash' } }) }));
	await page.route('**/api/v1/buyer/credit-requests/request-3/payments', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ payments: [] }) }));
	await page.route('**/api/v1/buyer/credit-requests/request-3/payment-claims', async (route) => { submitted = route.request().postDataJSON(); await route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify({ payment_claim: { id: 'claim-1', state: 'pending' } }) }); });
	await page.goto('/buyer/credit-requests/request-3');
	await page.getByLabel('Money you paid (₦)').fill('125,000');
	await page.getByLabel('Transfer number').fill('BANK-2026-001');
	await page.getByRole('button', { name: 'Tell the seller I paid' }).click();
	await expect(page.getByText('We told the seller. They will check their bank account.')).toBeVisible();
	await expect.poll(() => submitted).toMatchObject({ amount_kobo: 12500000, transfer_reference: 'BANK-2026-001' });
});

test('buyer can cancel an active mandate', async ({ page }) => {
	let cancelled = false;
	await page.route('**/api/v1/buyer/mandates', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ mandates: [{ id: 'mandate-1', provider: 'approved-provider', status: cancelled ? 'CANCELLED' : 'ACTIVE', amount_ceiling_kobo: 50000000 }] }) }));
	await page.route('**/api/v1/buyer/mandates/mandate-1/cancel', async (route) => { cancelled = true; await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ mandate: { id: 'mandate-1', status: 'CANCELLED' } }) }); });
	await page.goto('/buyer/mandates');
	await page.getByRole('button', { name: 'Stop bank debit' }).click();
	await expect(page.getByText('CANCELLED')).toBeVisible();
	await expect.poll(() => cancelled).toBe(true);
});

test('public receipt renders only the approved projection', async ({ page }) => {
	await page.route('**/api/v1/public/receipts/signed-token', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ receipt: { reference: 'payment-1', amount_kobo: 12500000, currency: 'NGN', source_type: 'buyer_payment_claim', state: 'recognized', paid_at: '2026-08-21T10:00:00Z', recognized_at: '2026-08-21T10:05:00Z' } }) }));
	await page.goto('/receipt/signed-token');
	await expect(page.getByRole('heading', { name: 'This money was received.' })).toBeVisible();
	await expect(page.getByText('₦125,000.00')).toBeVisible();
	await expect(page.getByText(/Names and bank details are hidden/)).toBeVisible();
});

test('supplier reserves exact drawdown terms and releases only after buyer confirmation', async ({ page }) => {
	test.setTimeout(60_000);
	await page.setViewportSize({ width: 390, height: 844 });
	let reserved: Record<string, unknown> | undefined;
	let released: Record<string, unknown> | undefined;
	const line = { id: 'line-1', approved_limit_kobo: 100000000, current_exposure_kobo: 0, reserved_pending_kobo: 25000000, available_limit_kobo: 75000000, state: 'ACTIVE', version: 2 };
	const confirmed = { id: 'drawdown-confirmed', trade_line_id: 'line-1', principal_kobo: 25000000, goods_description: 'Twenty bags of rice', invoice_reference: 'INV-25', due_date: '2026-09-30', collection_at: '2026-10-01T09:00:00Z', grace_hours: 24, agreement_hash: 'hash-confirmed', state: 'BUYER_CONFIRMED' };
	await page.route('**/api/v1/organizations', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ organizations: [{ id: 'org-1', legal_name: 'Adebayo Supplies' }] }) }));
	await page.route('**/api/v1/organizations/org-1/trade-lines/line-1/statement', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ line, drawdowns: [confirmed] }) }));
	await page.route('**/api/v1/organizations/org-1/trade-lines/line-1/drawdowns', async (route) => { reserved = route.request().postDataJSON(); await route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify({ drawdown: { id: 'new-drawdown' } }) }); });
	await page.route('**/api/v1/organizations/org-1/trade-lines/line-1/drawdowns/drawdown-confirmed/release', async (route) => { released = route.request().postDataJSON(); confirmed.state = 'GOODS_RELEASED'; await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ drawdown: confirmed, trade_line: line }) }); });
	await page.goto('/app/trade-lines/line-1');
	await page.getByLabel('Money to pay (₦)').fill('100,000');
	await page.getByLabel('What are they buying?').fill('Ten cartons of oil');
	await page.getByLabel('Pay before').fill('2026-10-30');
	await page.getByLabel('Bank debit may start after').fill('2026-10-31T09:00');
	await page.getByRole('button', { name: 'Add ₦100,000.00 sale' }).click();
	await expect.poll(() => reserved).toMatchObject({ principal_kobo: 10000000, goods_description: 'Ten cartons of oil', due_date: '2026-10-30' });
	await page.getByLabel('How will they get the goods?').fill('Courier');
	await page.getByLabel('Delivery or receipt number').fill('TRACK-100');
	await page.getByRole('button', { name: 'The goods have left' }).click();
	await expect.poll(() => released).toMatchObject({ delivery_method: 'Courier', evidence_reference: 'TRACK-100' });
});

test('buyer confirms the exact hash and no-issue receipt activates the drawdown once', async ({ page }) => {
	let confirmation: Record<string, unknown> | undefined;
	let receipt: Record<string, unknown> | undefined;
	const line = { id: 'line-1', available_limit_kobo: 75000000, current_exposure_kobo: 0, reserved_pending_kobo: 25000000 };
	const drawdown = { id: 'drawdown-1', trade_line_id: 'line-1', principal_kobo: 25000000, goods_description: 'Twenty bags of rice', invoice_reference: 'INV-25', due_date: '2026-09-30', collection_at: '2026-10-01T09:00:00Z', grace_hours: 24, agreement_hash: 'immutable-hash-1', state: 'PENDING_BUYER_CONFIRMATION', delivery_method: 'Courier', release_evidence_reference: 'TRACK-25', obligation_id: '' };
	await page.route('**/api/v1/buyer/trade-lines', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ trade_lines: [line] }) }));
	await page.route('**/api/v1/buyer/trade-lines/line-1/statement', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ line, drawdowns: [drawdown] }) }));
	await page.route('**/api/v1/buyer/trade-lines/line-1/drawdowns/drawdown-1/confirm', async (route) => { confirmation = route.request().postDataJSON(); drawdown.state = 'GOODS_RELEASED'; await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ drawdown, trade_line: line }) }); });
	await page.route('**/api/v1/buyer/trade-lines/line-1/drawdowns/drawdown-1/receipt', async (route) => { receipt = route.request().postDataJSON(); drawdown.state = 'ACTIVATED'; drawdown.obligation_id = 'obligation-1'; await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ drawdown, trade_line: line }) }); });
	await page.goto('/buyer/trade-lines');
	await page.getByText('Technical record').click();
	await expect(page.getByText('immutable-hash-1')).toBeVisible();
	await page.getByRole('button', { name: 'Yes to this ₦250,000.00 sale' }).click();
	await expect.poll(() => confirmation).toEqual({ agreement_hash: 'immutable-hash-1' });
	await page.getByRole('button', { name: 'Yes, I got the goods' }).click();
	await expect.poll(() => receipt).toEqual({ state: 'no_issue' });
	await expect(page.getByRole('link', { name: 'Open payment details →' })).toBeVisible();
});

test('buyer receipt issue opens a case without activating an obligation', async ({ page }) => {
	let receipt: Record<string, unknown> | undefined;
	const line = { id: 'line-issue', available_limit_kobo: 50000000, current_exposure_kobo: 0, reserved_pending_kobo: 10000000 };
	const drawdown = { id: 'drawdown-issue', principal_kobo: 10000000, goods_description: 'Damaged cartons', invoice_reference: 'INV-ISSUE', due_date: '2026-09-30', collection_at: '2026-10-01T09:00:00Z', grace_hours: 24, agreement_hash: 'issue-hash', state: 'GOODS_RELEASED', delivery_method: 'Courier', release_evidence_reference: 'TRACK-ISSUE', obligation_id: '' };
	await page.route('**/api/v1/buyer/trade-lines', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ trade_lines: [line] }) }));
	await page.route('**/api/v1/buyer/trade-lines/line-issue/statement', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ line, drawdowns: [drawdown] }) }));
	await page.route('**/api/v1/buyer/trade-lines/line-issue/drawdowns/drawdown-issue/receipt', async (route) => { receipt = route.request().postDataJSON(); drawdown.state = 'RECEIPT_ISSUE_REPORTED'; await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ drawdown: { ...drawdown, receipt_dispute_id: 'dispute-1' }, trade_line: line }) }); });
	await page.goto('/buyer/trade-lines');
	await page.getByLabel('What is wrong?').fill('Four cartons arrived damaged');
	await page.getByRole('button', { name: 'Report the problem' }).click();
	await expect.poll(() => receipt).toEqual({ state: 'issue_reported', issue_reason: 'Four cartons arrived damaged' });
	expect(drawdown.obligation_id).toBe('');
});

test('pilot-ready owner reviews mobile readiness and invites a sales user', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	let invited: Record<string, unknown> | undefined;
	await page.route('**/api/v1/organizations', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ organizations: [{ id: 'org-ready', legal_name: 'Fresh Foods Ltd' }] }) }));
	await page.route('**/api/v1/organizations/org-ready/onboarding', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ profile: { organization_id: 'org-ready', version: 12, kyb_state: 'approved', settlement_state: 'verified', billing_state: 'configured', terms_version: 'supplier-terms-v1', privacy_version: 'privacy-v1' }, readiness: { state: 'pilot_ready', ready: true, requirements: ['business_identity','email_verified','phone_verified','kyb_approved','settlement_verified','billing_configured','credit_policy','current_consents','owner_mfa','finance_mfa'].map((code) => ({ code, label: code.replaceAll('_',' '), complete: true, manage_path: '/app/onboarding' })), missing: [] }, permissions: { business: true, settlement: true, billing: true, credit_policy: true, consents: true }, current_terms_version: 'supplier-terms-v1', current_privacy_version: 'privacy-v1' }) }));
	await page.route('**/api/v1/organizations/org-ready/members', async (route) => {
		if (route.request().method() === 'POST') { invited = route.request().postDataJSON(); await route.fulfill({ status: 202, contentType: 'application/json', body: JSON.stringify({ membership: { user_id: 'sales-user', role: 'sales', status: 'invited' } }) }); return; }
		await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ members: [{ user_id: 'owner-user', role: 'owner', status: 'active' }] }) });
	});
	await page.goto('/app/onboarding');
	await expect(page.getByRole('heading', { name: 'Ready to use' })).toBeVisible();
	await expect(page.getByText('10/10')).toBeVisible();
	await page.getByRole('link', { name: 'Protect staff who handle money →' }).click();
	await expect(page).toHaveURL(/\/app\/team$/);
	await page.getByLabel('Email or phone').fill('sales@fresh-foods.test');
	await page.getByRole('button', { name: 'Send invite' }).click();
	await expect(page.getByText('Invite sent.')).toBeVisible();
	await expect.poll(() => invited).toEqual({ target: 'sales@fresh-foods.test', channel: 'email', role: 'sales' });
});

test('incomplete supplier sees precise recovery steps before financial activity', async ({ page }) => {
	await page.route('**/api/v1/organizations', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ organizations: [{ id: 'org-incomplete', legal_name: 'Starting Supplier Ltd' }] }) }));
	const missing = [
		{ code: 'phone_verified', label: 'Owner phone verified', complete: false, manage_path: '/app/onboarding' },
		{ code: 'kyb_approved', label: 'Business verification approved', complete: false, manage_path: '/app/onboarding' },
		{ code: 'settlement_verified', label: 'Settlement destination verified', complete: false, manage_path: '/app/settings/settlement' }
	];
	await page.route('**/api/v1/organizations/org-incomplete/onboarding', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ profile: { organization_id: 'org-incomplete', version: 2, kyb_state: 'not_started', settlement_state: 'not_started', billing_state: 'configured' }, readiness: { state: 'incomplete', ready: false, requirements: missing, missing }, permissions: { business: false, settlement: false, billing: false, credit_policy: false, consents: false }, current_terms_version: 'supplier-terms-v1', current_privacy_version: 'privacy-v1' }) }));
	await page.goto('/app/onboarding');
	await expect(page.getByText('3 step(s) left.')).toBeVisible();
	await expect(page.getByText('Add the bank account for your money')).toBeVisible();
	await expect(page.getByRole('link', { name: 'Bank account for your money →' })).toBeVisible();
});

test('user changes notification routing, quiet hours, and optional categories', async ({ page }) => {
	let saved: Record<string, unknown> | undefined;
	let preferences = { preferred_channel: 'whatsapp', fallback_channel: 'email', payment_reminders_enabled: true, product_updates_enabled: false, quiet_start_hour: 22, quiet_end_hour: 7, timezone: 'Africa/Lagos', version: 1 };
	await page.route('**/api/v1/me/notification-preferences', async (route) => {
		if (route.request().method() === 'PUT') { saved = route.request().postDataJSON(); preferences = { ...preferences, ...saved, version: 2 }; await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ preferences }) }); return; }
		await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ preferences, required_groups: ['SECURITY_REQUIRED', 'TRANSACTIONAL_REQUIRED'] }) });
	});
	await page.goto('/app/settings/notifications');
	await expect(page.getByText(/Security alerts and financial receipts remain on/)).toBeVisible();
	await page.getByLabel('Preferred channel').selectOption('email');
	await page.getByLabel('Fallback channel').selectOption('sms');
	await page.getByLabel('Payment reminders').uncheck();
	await page.getByLabel('Product updates').check();
	await page.getByLabel('Quiet time starts').fill('21');
	await page.getByRole('button', { name: 'Save preferences' }).click();
	await expect(page.getByText(/Future optional messages/)).toBeVisible();
	await expect.poll(() => saved).toMatchObject({ preferred_channel: 'email', fallback_channel: 'sms', payment_reminders_enabled: false, product_updates_enabled: true, quiet_start_hour: 21, expected_version: 1 });
});

test('identity-bound privacy request is submitted and remains trackable', async ({ page }) => {
	let submitted: Record<string, unknown> | undefined;
	let requests: any[] = [];
	await page.route('**/api/v1/me/privacy-requests', async (route) => {
		if (route.request().method() === 'POST') { submitted = route.request().postDataJSON(); requests = [{ id: 'privacy-1', request_type: 'PORTABILITY', state: 'IN_REVIEW', due_at: '2026-09-28T00:00:00Z' }]; await route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify({ request: requests[0] }) }); return; }
		await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ requests }) });
	});
	await page.goto('/app/settings/privacy');
	await page.getByLabel('What should we do?').selectOption('PORTABILITY');
	await page.getByLabel('Tell us more').fill('Provide my account information in a portable format');
	await page.getByRole('button', { name: 'Send my request' }).click();
	await expect(page.getByText('Download my information')).toBeVisible();
	await expect(page.getByText('In review')).toBeVisible();
	await expect.poll(() => submitted).toEqual({ request_type: 'PORTABILITY', details: 'Provide my account information in a portable format' });
});

test('recovery start is enumeration-safe and explains independent proof', async ({ page }) => {
	let submitted: Record<string, unknown> | undefined;
	await page.route('**/api/v1/account-recovery/requests', async (route) => { submitted = route.request().postDataJSON(); await route.fulfill({ status: 202, contentType: 'application/json', body: JSON.stringify({ message: 'If the account is eligible, recovery instructions have been sent.' }) }); });
	await page.goto('/recover');
	await expect(page.getByText(/ask for more than your phone number/)).toBeVisible();
	await page.getByLabel('Your email or phone').fill('someone@example.test');
	await page.getByRole('button', { name: 'Help me sign in' }).click();
	await expect(page.getByText('If the account is eligible, recovery instructions have been sent.')).toBeVisible();
	await expect.poll(() => submitted).toEqual({ identifier: 'someone@example.test', channel: 'email' });
});

test('owner changes a role, suspends access, and restores it from the team interface', async ({ page }) => {
	const changes: Record<string, unknown>[] = [];
	let member = { user_id: 'team-user', role: 'sales', status: 'active' };
	await page.route('**/api/v1/organizations', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ organizations: [{ id: 'org-team', legal_name: 'Team Supplier Ltd' }] }) }));
	await page.route('**/api/v1/organizations/org-team/members', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ members: [{ user_id: 'owner-user', role: 'owner', status: 'active' }, member] }) }));
	await page.route('**/api/v1/organizations/org-team/members/team-user', async (route) => { const change = route.request().postDataJSON(); changes.push(change); member = { ...member, ...change }; await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ membership: member }) }); });
	await page.goto('/app/team');
	await page.locator('.controls select').selectOption('finance');
	await page.getByRole('button', { name: 'Save change' }).click();
	await page.getByRole('button', { name: 'Stop access' }).click();
	await expect(page.getByRole('button', { name: 'Allow access again' })).toBeVisible();
	await page.getByRole('button', { name: 'Allow access again' }).click();
	await expect.poll(() => changes).toEqual([{ role: 'finance' }, { status: 'suspended' }, { status: 'active' }]);
});

test('supplier creates a mandate-backed trade line and sees provider validation safely', async ({ page }) => {
	let submitted: Record<string, unknown> | undefined;
	let attempts = 0;
	await page.route('**/api/v1/organizations', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ organizations: [{ id: 'org-line', legal_name: 'Line Supplier Ltd' }] }) }));
	await page.route('**/api/v1/organizations/org-line/customers', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ customers: [{ buyer_user_id: 'buyer-line', buyer_business_id: 'business-line', legal_name: 'Repeat Buyer Ltd' }] }) }));
	await page.route('**/api/v1/organizations/org-line/trade-lines', async (route) => { if (route.request().method() === 'GET') { await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ trade_lines: [] }) }); return; } submitted = route.request().postDataJSON(); attempts++; if (attempts === 1) { await route.fulfill({ status: 422, contentType: 'application/problem+json', body: JSON.stringify({ detail: 'The mandate provider could not verify this reference.' }) }); return; } await route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify({ trade_line: { id: 'line-created' } }) }); });
	await page.goto('/app/trade-lines');
	await page.locator('.card form select').first().selectOption('buyer-line');
	await page.getByLabel('Bank debit setup number').fill('mandate-approved');
	await page.getByLabel('Most they can owe (₦)').fill('1,000,000');
	await page.getByLabel('Start day').fill('2026-09-01T09:00');
	await page.getByLabel('End day').fill('2027-09-01T09:00');
	await page.getByRole('button', { name: 'Give a ₦1,000,000.00 limit' }).click();
	await expect(page.getByText('The mandate provider could not verify this reference.')).toBeVisible();
	await page.getByRole('button', { name: 'Give a ₦1,000,000.00 limit' }).click();
	await expect(page.getByText('Your customer can now owe up to ₦1,000,000.00.')).toBeVisible();
	await expect.poll(() => submitted).toMatchObject({ buyer_user_id: 'buyer-line', buyer_business_id: 'business-line', mandate_id: 'mandate-approved', approved_limit_kobo: 100000000 });
});

test('buyer attaches evidence to an existing dispute and sees the status timeline', async ({ page }) => {
	let submitted: Record<string, unknown> | undefined;
	const dispute = { id: 'dispute-evidence', total_disputed_kobo: 5000000, remaining_disputed_kobo: 5000000, reason: 'Damaged cartons', explanation: 'Five cartons were damaged.', state: 'OPEN', collection_effect: 'CONTESTED_ONLY', opened_at: '2026-08-29T08:00:00Z' };
	let evidence: any[] = [];
	await page.route('**/api/v1/buyer/disputes/dispute-evidence', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ dispute, evidence, decisions: [] }) }));
	await page.route('**/api/v1/buyer/disputes/dispute-evidence/evidence', async (route) => { submitted = route.request().postDataJSON(); evidence = [{ statement: submitted?.statement, submitted_at: '2026-08-29T09:00:00Z' }]; await route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify({ evidence: evidence[0] }) }); });
	await page.goto('/buyer/disputes/dispute-evidence');
	await page.getByLabel('What should we know?').fill('Courier inspection confirms the damaged seals.');
	await page.getByRole('button', { name: 'Add this information' }).click();
	await expect(page.getByText('Courier inspection confirms the damaged seals.')).toBeVisible();
	await expect.poll(() => submitted).toEqual({ document_id: '', statement: 'Courier inspection confirms the damaged seals.' });
});
