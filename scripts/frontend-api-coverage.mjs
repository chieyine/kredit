#!/usr/bin/env node

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { spawnSync } from 'node:child_process';

const root = resolve(import.meta.dirname, '..');
const server = readFileSync(resolve(root, 'internal/web/server.go'), 'utf8');
let listing = spawnSync('rg', ['--files', 'web/src', '-g', '!web/src/lib/api/generated/**'], { cwd: root, encoding: 'utf8' });
if (listing.status !== 0) {
	listing = spawnSync('git', ['ls-files', '--cached', '--others', '--exclude-standard', 'web/src/**', ':(exclude)web/src/lib/api/generated/**'], {
		cwd: root,
		encoding: 'utf8'
	});
}
if (listing.status !== 0) throw new Error(listing.stderr || 'Could not list frontend files.');
const frontend = listing.stdout.trim().split('\n').filter(Boolean).map((file) => readFileSync(resolve(root, file), 'utf8')).join('\n');

const normalise = (path) => path.split('?')[0].replace(/\$\{[^}]+\}|\{[^}]+\}/g, '{id}').replace(/\/$/, '');
const routes = [...server.matchAll(/HandleFunc\("([A-Z]+) ([^"]+)"/g)].map((match) => ({ method: match[1], path: normalise(match[2]) }));
const expandedFrontend = frontend
	.replace(/\$\{[^}]*['"]\?[^}]*\}|\$\{[^}]*(?:query|Query)[^}]*\}/g, '')
	.replace(/\$\{[^}]+\}/g, '{id}');
const directPaths = new Set([...expandedFrontend.matchAll(/\/api\/v1\/[A-Za-z0-9_./?&={}:~-]*/g)].map((match) => normalise(match[0])));
for (const match of frontend.matchAll(/api\.(?:GET|POST|PUT|PATCH|DELETE)\(\s*['"]([^'"]+)/g)) directPaths.add(normalise(`/api/v1${match[1]}`));

// These routes are deliberately not separate product screens. The reason is
// kept beside the exception so a new backend route cannot silently disappear.
const noSeparateScreen = new Map([
 ["POST /api/v1/organizations/{id}/drawdowns/{id}/approval", "API-only drawdown review request; no shipped browser controller"],
 ["POST /api/v1/organizations/{id}/credit-requests/{id}/line-items", "API-only item setup; deliveries screen reads but does not create line items"],
 ["POST /api/v1/organizations/{id}/credit-requests/{id}/receipts", "legacy API alias for buyer-owned receipt evidence; not a supplier UI action"],
 ["POST /api/v1/buyer/credit-requests/{id}/shipments/{id}/receipt", "API-only item-level buyer receipt; existing buyer UI confirms aggregate receipt only"],
 ["POST /api/v1/organizations/{id}/credit-notes/{id}/approve", "API-only independent approval; no shipped reviewer UI"],
 ["GET /api/v1/organizations/{id}/terms-imports", "API-only proposed-terms staging; financial application and browser workflow remain deferred"],
 ["POST /api/v1/organizations/{id}/terms-imports", "API-only proposed-terms staging; financial application and browser workflow remain deferred"],
 ["GET /api/v1/organizations/{id}/terms-imports/{id}", "API-only terms review; not the separate distributor-contact import UI"],
 ["POST /api/v1/organizations/{id}/terms-imports/{id}", "API-only independent terms review; does not apply opening balances"],
 ["POST /api/v1/organizations/{id}/erp/reconcile", "API-only comparison report; browser controller and broader accounting verification remain pending"],
	['POST /api/v1/webhooks/paystack/{id}', 'provider-to-server callback; no browser action'],
 ['POST /api/v1/webhooks/bank/{id}', 'provider-to-server callback; no browser action'],
 ['GET /healthz', 'service health check'],
	['GET /readyz', 'service readiness check'],
	['POST /api/v1/webhooks/collection/{id}', 'provider-to-server webhook'],
	['POST /api/v1/webhooks/messaging/whatsapp', 'provider-to-server webhook'],
	['GET /api/v1/ops/metrics/prometheus', 'machine-readable monitoring feed'],
	['POST /api/v1/organizations/{id}/documents/upload-slot', 'used behind the document uploader'],
	['POST /api/v1/organizations/{id}/documents/{id}/complete', 'used behind the document uploader'],
	['GET /api/v1/organizations/{id}/credit-requests/{id}/reconciliation', 'internal repair and diagnostic action'],
	['GET /api/v1/buyer/credit-requests/{id}/agreement', 'included in the full buyer sale response'],
	['GET /api/v1/buyer/credit-requests/{id}/payments', 'included on the buyer obligation screen'],
	['GET /api/v1/buyer/credit-requests/{id}/schedule', 'included on the buyer obligation screen'],
	['POST /webhooks/mono', 'provider-to-server webhook'],
	['POST /api/v1/webhooks/mono', 'provider-to-server webhook'],
	['POST /api/v1/webhooks/notifications/{id}', 'provider-to-server webhook'],
	['POST /api/v1/buyer/businesses/{id}/repayment-customer', 'provider customer setup behind authorization session'],
	['GET /api/v1/organizations/{id}', 'the organisation list already returns each full business record']
]);

// Some screens build endpoint paths from a shared base or action name. Static
// string matching cannot reconstruct those paths, so each one is tied to the
// component that exposes it. Removing that component makes this check fail.
const coveredThroughComponent = new Map([
 ["GET /api/v1/organizations/{id}/credit-requests/{id}/deliveries", "web/src/routes/workspace/sales/[id]/deliveries/+page.svelte"],
 ["POST /api/v1/organizations/{id}/credit-requests/{id}/shipments", "web/src/routes/workspace/sales/[id]/deliveries/+page.svelte"],
 ["POST /api/v1/organizations/{id}/credit-requests/{id}/credit-notes", "web/src/routes/workspace/sales/[id]/deliveries/+page.svelte"],
 ["GET /api/v1/organizations/{id}/reports/enterprise", "web/src/routes/workspace/reports/+page.svelte"],
 ['GET /api/v1/organizations/{id}/branch-access', 'web/src/routes/workspace/partners/access/+page.svelte'],
 ['PUT /api/v1/organizations/{id}/branch-access/{id}', 'web/src/routes/workspace/partners/access/+page.svelte'],
 ['GET /api/v1/organizations/{id}/purchasing-authority', 'web/src/routes/workspace/purchases/permissions/+page.svelte'],
 ['PUT /api/v1/organizations/{id}/purchasing-authority/{id}', 'web/src/routes/workspace/purchases/permissions/+page.svelte'],
 ['GET /api/v1/organizations/{id}/network-operations', 'web/src/routes/workspace/partners/operations/+page.svelte'],
 ['PUT /api/v1/organizations/{id}/branches/{id}', 'web/src/routes/workspace/partners/operations/+page.svelte'],
 ['PUT /api/v1/organizations/{id}/customers/{id}/assignment', 'web/src/routes/workspace/partners/operations/+page.svelte'],
 ['PUT /api/v1/organizations/{id}/credit-approvals/reviewers/{id}', 'web/src/routes/workspace/sales/approvals/+page.svelte'],
	['GET /api/v1/organizations/{id}/credit-approvals', 'web/src/routes/workspace/sales/approvals/+page.svelte'],
	['PUT /api/v1/organizations/{id}/credit-approvals/policy', 'web/src/routes/workspace/sales/approvals/+page.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/approval', 'web/src/routes/workspace/sales/approvals/+page.svelte'],
	['POST /api/v1/organizations/{id}/credit-approvals/{id}', 'web/src/routes/workspace/sales/approvals/+page.svelte'],
	['GET /api/v1/organizations/{id}/distributor-imports/{id}', 'web/src/routes/workspace/partners/import/+page.svelte'],
	['POST /api/v1/organizations/{id}/distributor-imports/{id}', 'web/src/routes/workspace/partners/import/+page.svelte'],
	['POST /api/v1/buyer-invitations/{id}/otp', 'web/src/routes/buyer-invitations/[token]/+page.svelte'],
	['POST /api/v1/buyer-invitations/{id}/accept', 'web/src/routes/buyer-invitations/[token]/+page.svelte'],
	['POST /api/v1/buyer/mandates/{id}/refresh', 'web/src/routes/workspace/purchases/mandates/+page.svelte'],
 ['GET /api/v1/dsa/code/{id}', 'web/src/routes/join/+page.svelte'],
 ['GET /api/v1/organizations/{id}/distributor-invitations', 'web/src/routes/workspace/partners/invitations/+page.svelte'],
 ['GET /api/v1/organizations/{id}/payments', 'web/src/routes/workspace/money/received/+page.svelte'],
	['GET /api/v1/organizations/{id}/customers/{id}/history', 'web/src/routes/workspace/partners/customers/[id]/+page.svelte'],
	['GET /api/v1/organizations/{id}/customers/{id}/statement', 'web/src/routes/workspace/partners/customers/[id]/+page.svelte'],
	['POST /api/v1/buyer/credit-requests/{id}/mandate', 'web/src/routes/workspace/purchases/orders/[requestID]/+page.svelte'],
	['POST /api/v1/buyer/credit-requests/{id}/accept', 'web/src/routes/workspace/purchases/orders/[requestID]/+page.svelte'],
	['POST /api/v1/buyer/credit-requests/{id}/decline', 'web/src/routes/workspace/purchases/orders/[requestID]/+page.svelte'],
	['POST /api/v1/buyer/credit-requests/{id}/receipt', 'web/src/routes/workspace/purchases/orders/[requestID]/+page.svelte'],
	['POST /api/v1/buyer/credit-requests/{id}/payment-claims', 'web/src/routes/workspace/purchases/orders/[requestID]/+page.svelte'],
	['POST /api/v1/buyer/credit-requests/{id}/payment-link', 'web/src/routes/workspace/purchases/orders/[requestID]/+page.svelte'],
	['POST /api/v1/buyer/credit-requests/{id}/disputes', 'web/src/routes/workspace/purchases/orders/[requestID]/+page.svelte'],
	['GET /api/v1/organizations/{id}/payment-claims', 'web/src/routes/workspace/money/received/+page.svelte'],
	['POST /api/v1/ops/privacy-requests/{id}/decide', 'web/src/routes/admin/privacy/+page.svelte'],
	['POST /api/v1/ops/privacy-requests/{id}/complete', 'web/src/routes/admin/privacy/+page.svelte'],
	['GET /api/v1/organizations/{id}/audit-events', 'web/src/routes/workspace/activity/+page.svelte'],
	['POST /api/v1/organizations/{id}/onboarding/contacts/challenges', 'web/src/routes/workspace/onboarding/+page.svelte'],
	['POST /api/v1/organizations/{id}/onboarding/contacts/verify', 'web/src/routes/workspace/onboarding/+page.svelte'],
	['PATCH /api/v1/organizations/{id}/onboarding/representative', 'web/src/routes/workspace/onboarding/+page.svelte'],
	['POST /api/v1/organizations/{id}/onboarding/kyb', 'web/src/routes/workspace/onboarding/+page.svelte'],
	['POST /api/v1/organizations/{id}/onboarding/kyb/reconcile', 'web/src/routes/workspace/onboarding/+page.svelte'],
	['POST /api/v1/organizations/{id}/onboarding/consents', 'web/src/routes/workspace/onboarding/+page.svelte'],
	['POST /api/v1/buyer/mandates/{id}/cancel', 'web/src/routes/workspace/purchases/mandates/+page.svelte'],
	['POST /api/v1/buyer/mandates/{id}/restore', 'web/src/routes/workspace/purchases/mandates/+page.svelte'],
	['POST /api/v1/buyer/disputes/{id}/evidence', 'web/src/lib/components/DisputeDetail.svelte'],
	['GET /api/v1/organizations/{id}/collections', 'web/src/routes/workspace/money/collections/+page.svelte'],
	['GET /api/v1/organizations/{id}/overdue', 'web/src/routes/workspace/overdue/+page.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/send', 'web/src/routes/workspace/sales/[id]/+page.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/cancel', 'web/src/routes/workspace/sales/[id]/+page.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/release', 'web/src/routes/workspace/sales/[id]/+page.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/payments', 'web/src/routes/workspace/sales/[id]/+page.svelte'],
	['GET /api/v1/organizations/{id}/credit-requests/{id}/payments', 'web/src/routes/workspace/sales/[id]/+page.svelte'],
	['GET /api/v1/organizations/{id}/credit-requests/{id}/payment-claims', 'web/src/routes/workspace/sales/[id]/+page.svelte'],
	['GET /api/v1/organizations/{id}/credit-requests/{id}/schedule', 'web/src/routes/workspace/sales/[id]/+page.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/schedule', 'web/src/routes/workspace/sales/[id]/+page.svelte'],
	['GET /api/v1/organizations/{id}/credit-requests/{id}/collection/eligibility', 'web/src/routes/workspace/sales/[id]/+page.svelte'],
	['GET /api/v1/organizations/{id}/credit-requests/{id}/collections', 'web/src/routes/workspace/sales/[id]/+page.svelte'],
	['POST /api/v1/organizations/{id}/collections/{id}/retry', 'web/src/routes/workspace/sales/[id]/+page.svelte'],
	['POST /api/v1/organizations/{id}/collections/{id}/reconcile', 'web/src/routes/workspace/sales/[id]/+page.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/disputes', 'web/src/routes/workspace/sales/[id]/+page.svelte'],
	['GET /api/v1/organizations/{id}/disputes', 'web/src/routes/workspace/disputes/+page.svelte'],
	['POST /api/v1/organizations/{id}/disputes/{id}/evidence', 'web/src/lib/components/DisputeDetail.svelte'],
	['POST /api/v1/organizations/{id}/disputes/{id}/decide', 'web/src/lib/components/DisputeDetail.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/write-off', 'web/src/routes/workspace/sales/[id]/+page.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/fee-waiver', 'web/src/routes/workspace/sales/[id]/+page.svelte'],
	['GET /api/v1/organizations/{id}/operations', 'web/src/routes/workspace/activity/+page.svelte'],
	['GET /api/v1/organizations/{id}/reports/receivables', 'web/src/routes/workspace/reports/+page.svelte'],
	['GET /api/v1/organizations/{id}/provider-status', 'web/src/routes/workspace/activity/+page.svelte'],
	['GET /api/v1/organizations/{id}/readiness', 'web/src/routes/workspace/activity/+page.svelte'],
	['GET /api/v1/ops/metrics', 'web/src/routes/admin/diagnostics/+page.svelte'],
	['GET /api/v1/organizations/{id}/reports/ageing', 'web/src/routes/workspace/reports/+page.svelte'],
	['GET /api/v1/organizations/{id}/reports/fees', 'web/src/routes/workspace/reports/+page.svelte'],
	['GET /api/v1/organizations/{id}/corrections', 'web/src/routes/workspace/activity/+page.svelte'],
	['POST /api/v1/buyer/disputes/{id}/documents', 'web/src/lib/components/DisputeDetail.svelte'],
	['GET /api/v1/buyer/disputes/{id}/documents/{id}', 'web/src/lib/components/DisputeDetail.svelte'],
	['GET /api/v1/buyer/disputes/{id}/documents/{id}/download', 'web/src/lib/components/DisputeDetail.svelte'],
	['POST /api/v1/organizations/{id}/disputes/{id}/documents', 'web/src/lib/components/DisputeDetail.svelte'],
	['GET /api/v1/organizations/{id}/disputes/{id}/documents/{id}', 'web/src/lib/components/DisputeDetail.svelte'],
	['GET /api/v1/organizations/{id}/disputes/{id}/documents/{id}/download', 'web/src/lib/components/DisputeDetail.svelte'],
	['GET /api/v1/ops/disputes/{id}/documents/{id}', 'web/src/routes/admin/disputes/[id]/+page.svelte'],
	['GET /api/v1/buyer/credit-requests/{id}/invoice', 'web/src/routes/workspace/purchases/orders/[requestID]/+page.svelte'],
	['GET /api/v1/organizations/{id}/due', 'web/src/routes/workspace/today/+page.svelte']
]);

for (const [route, file] of coveredThroughComponent) {
	if (!listing.stdout.split('\n').includes(file)) throw new Error(`${route} points to missing frontend component ${file}`);
}

// These paths are assembled from local helper/base variables. Check their
// actual request expressions as well as the screen mapping above.
const dynamicBindings = [
 ['web/src/routes/workspace/sales/[id]/deliveries/+page.svelte', ['checkedJSON(`${base}/deliveries`', '`${base}/shipments`', '`${base}/credit-notes`']],
 ['web/src/routes/workspace/reports/+page.svelte', ['checkedJSON(`${root}/reports/enterprise`']],
 ['web/src/routes/workspace/partners/access/+page.svelte', ['checkedJSON(`${base()}/branch-access`','`${base()}/branch-access/${encodeURIComponent(s.user_id)}`']],
 ['web/src/routes/workspace/partners/operations/+page.svelte', ['checkedJSON(`${base()}/network-operations`','save(`/branches/${encodeURIComponent(b.id)}`','save(`/customers/${encodeURIComponent(p.business_id)}/assignment`']],
	['web/src/routes/workspace/sales/approvals/+page.svelte', ['checkedJSON(`${base()}/credit-approvals`', "change('/credit-approvals/policy'", 'change(`/credit-requests/${draft!.id}/approval`', 'change(`/credit-approvals/${item.id}`', 'change(`/credit-approvals/reviewers/${encodeURIComponent(reviewer.user_id)}`']],
	['web/src/routes/workspace/partners/import/+page.svelte', ['checkedJSON(`${endpoint()}/${encodeURIComponent(id)}`', "new MutationIntent('cancel-import:' + batch.id, `${endpoint()}/${batch.id}`)"]],
	['web/src/routes/buyer-invitations/[token]/+page.svelte', ["checkedJSON(invitationPath()+'/otp'", "{method:'POST'}", "new MutationIntent('accept-buyer-invitation',invitationPath()+'/accept')", 'acceptance.run(']],
	['web/src/routes/workspace/money/received/+page.svelte', ['checkedJSON(`${root}/payments`']],
	['web/src/routes/workspace/partners/customers/[id]/+page.svelte', ['checkedJSON(`${base}/history${query}`', 'checkedJSON(`${base}/statement${query}`']]
];
for (const [file, bindings] of dynamicBindings) {
	const source = readFileSync(resolve(root, file), 'utf8');
	for (const binding of bindings) if (!source.includes(binding)) throw new Error(`Dynamic API controller is missing in ${file}: ${binding}`);
}

// Dynamic financial actions must retain their actual controller binding, not
// merely the page filename. Browser regression tests exercise the transitions.
const buyerController = readFileSync(resolve(root, 'web/src/routes/workspace/purchases/orders/[requestID]/+page.svelte'), 'utf8');
for (const action of ['mandate', 'accept', 'decline', 'receipt', 'payment-claims', 'payment-link', 'disputes']) {
 if (!buyerController.includes(`perform('${action}'`)) throw new Error(`Buyer ${action} controller is missing`);
}
const missing = routes.filter(({ method, path }) => {
	if (directPaths.has(path)) return false;
	const key = `${method} ${path}`;
	return !noSeparateScreen.has(key) && !coveredThroughComponent.has(key);
});

if (missing.length) {
	for (const route of missing) process.stderr.write(`${route.method} ${route.path}\n`);
	process.stderr.write(`Frontend API coverage failed: ${missing.length} route(s) have no screen or recorded server-only reason.\n`);
	process.exit(1);
}
process.stdout.write(`Frontend API coverage passed for ${routes.length} backend routes (${noSeparateScreen.size} intentionally have no separate screen).\n`);
