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
const expandedFrontend = frontend.replace(/\$\{[^}]+\}/g, '{id}');
const directPaths = new Set([...expandedFrontend.matchAll(/\/api\/v1\/[A-Za-z0-9_./?&={}:~-]*/g)].map((match) => normalise(match[0])));
for (const match of frontend.matchAll(/api\.(?:GET|POST|PUT|PATCH|DELETE)\(\s*['"]([^'"]+)/g)) directPaths.add(normalise(`/api/v1${match[1]}`));

// These routes are deliberately not separate product screens. The reason is
// kept beside the exception so a new backend route cannot silently disappear.
const noSeparateScreen = new Map([
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
	['POST /api/v1/ops/privacy-requests/{id}/decide', 'web/src/routes/admin/privacy/+page.svelte'],
	['POST /api/v1/ops/privacy-requests/{id}/complete', 'web/src/routes/admin/privacy/+page.svelte'],
	['GET /api/v1/organizations/{id}/audit-events', 'web/src/routes/app/activity/+page.svelte'],
	['POST /api/v1/organizations/{id}/onboarding/contacts/challenges', 'web/src/routes/app/onboarding/+page.svelte'],
	['POST /api/v1/organizations/{id}/onboarding/contacts/verify', 'web/src/routes/app/onboarding/+page.svelte'],
	['PATCH /api/v1/organizations/{id}/onboarding/representative', 'web/src/routes/app/onboarding/+page.svelte'],
	['POST /api/v1/organizations/{id}/onboarding/kyb', 'web/src/routes/app/onboarding/+page.svelte'],
	['POST /api/v1/organizations/{id}/onboarding/kyb/reconcile', 'web/src/routes/app/onboarding/+page.svelte'],
	['POST /api/v1/organizations/{id}/onboarding/consents', 'web/src/routes/app/onboarding/+page.svelte'],
	['POST /api/v1/buyer/mandates/{id}/cancel', 'web/src/routes/buyer/mandates/+page.svelte'],
	['POST /api/v1/buyer/mandates/{id}/restore', 'web/src/routes/buyer/mandates/+page.svelte'],
	['POST /api/v1/buyer/disputes/{id}/evidence', 'web/src/lib/components/DisputeDetail.svelte'],
	['GET /api/v1/organizations/{id}/collections', 'web/src/routes/app/collections/+page.svelte'],
	['GET /api/v1/organizations/{id}/overdue', 'web/src/routes/app/overdue/+page.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/send', 'web/src/routes/app/credit/[id]/+page.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/cancel', 'web/src/routes/app/credit/[id]/+page.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/release', 'web/src/routes/app/credit/[id]/+page.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/payments', 'web/src/routes/app/credit/[id]/+page.svelte'],
	['GET /api/v1/organizations/{id}/credit-requests/{id}/payments', 'web/src/routes/app/credit/[id]/+page.svelte'],
	['GET /api/v1/organizations/{id}/credit-requests/{id}/payment-claims', 'web/src/routes/app/credit/[id]/+page.svelte'],
	['GET /api/v1/organizations/{id}/credit-requests/{id}/schedule', 'web/src/routes/app/credit/[id]/+page.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/schedule', 'web/src/routes/app/credit/[id]/+page.svelte'],
	['GET /api/v1/organizations/{id}/credit-requests/{id}/collection/eligibility', 'web/src/routes/app/credit/[id]/+page.svelte'],
	['GET /api/v1/organizations/{id}/credit-requests/{id}/collections', 'web/src/routes/app/credit/[id]/+page.svelte'],
	['POST /api/v1/organizations/{id}/collections/{id}/retry', 'web/src/routes/app/credit/[id]/+page.svelte'],
	['POST /api/v1/organizations/{id}/collections/{id}/reconcile', 'web/src/routes/app/credit/[id]/+page.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/disputes', 'web/src/routes/app/credit/[id]/+page.svelte'],
	['GET /api/v1/organizations/{id}/disputes', 'web/src/routes/app/disputes/+page.svelte'],
	['POST /api/v1/organizations/{id}/disputes/{id}/evidence', 'web/src/lib/components/DisputeDetail.svelte'],
	['POST /api/v1/organizations/{id}/disputes/{id}/decide', 'web/src/lib/components/DisputeDetail.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/write-off', 'web/src/routes/app/credit/[id]/+page.svelte'],
	['POST /api/v1/organizations/{id}/credit-requests/{id}/fee-waiver', 'web/src/routes/app/credit/[id]/+page.svelte'],
	['GET /api/v1/organizations/{id}/operations', 'web/src/routes/app/activity/+page.svelte'],
	['GET /api/v1/organizations/{id}/reports/receivables', 'web/src/routes/app/reports/+page.svelte'],
	['GET /api/v1/organizations/{id}/provider-status', 'web/src/routes/app/activity/+page.svelte'],
	['GET /api/v1/organizations/{id}/readiness', 'web/src/routes/app/activity/+page.svelte'],
	['GET /api/v1/ops/metrics', 'web/src/routes/admin/diagnostics/+page.svelte'],
	['GET /api/v1/organizations/{id}/reports/ageing', 'web/src/routes/app/reports/+page.svelte'],
	['GET /api/v1/organizations/{id}/reports/fees', 'web/src/routes/app/reports/+page.svelte'],
	['GET /api/v1/organizations/{id}/corrections', 'web/src/routes/app/activity/+page.svelte']
]);

for (const [route, file] of coveredThroughComponent) {
	if (!listing.stdout.split('\n').includes(file)) throw new Error(`${route} points to missing frontend component ${file}`);
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
