import http from 'k6/http';
import { check, sleep } from 'k6';

const base = (__ENV.BASE_URL || 'http://127.0.0.1:8080').replace(/\/$/, '');
const mode = __ENV.PERFORMANCE_MODE || 'smoke';
const profiles = {
  smoke: { vus: 5, duration: '30s' },
  portfolio: { vus: 50, duration: '5m' },
  webhook_burst: { scenarios: { burst: { executor: 'shared-iterations', vus: 200, iterations: 10000, maxDuration: '2m' } } }
};
if (!profiles[mode]) throw new Error('Choose smoke, portfolio, or webhook_burst.');
if (mode === 'portfolio' && (!__ENV.ORGANIZATION_ID || !__ENV.SESSION_COOKIE)) {
  throw new Error('Portfolio checks require an isolated business and its authenticated session cookie.');
}
// A provider-authenticated fixture must reference an existing isolated attempt.
// Repeating it measures authenticated webhook deduplication, not unique debit throughput.
const fixture = mode === 'webhook_burst' ? JSON.parse(open(__ENV.WEBHOOK_FIXTURE || 'missing-webhook-fixture.json')) : null;
if (fixture && (!fixture.EventID || !fixture.ExternalReference || !fixture.Signature)) {
  throw new Error('Supply a complete, signed mock collection webhook fixture.');
}
export const options = {
  ...profiles[mode],
  thresholds: { checks: ['rate>0.99'], http_req_failed: ['rate<0.01'], http_req_duration: ['p(95)<1000'] }
};

export default function () {
  let response;
  if (mode === 'webhook_burst') {
    response = http.post(`${base}/api/v1/webhooks/collection/mock-collection`, JSON.stringify(fixture), { headers: { 'Content-Type': 'application/json' } });
    check(response, { 'authenticated webhook accepted': r => r.status === 202 });
  } else if (mode === 'portfolio') {
    response = http.get(`${base}/api/v1/organizations/${encodeURIComponent(__ENV.ORGANIZATION_ID)}/reports/receivables`, { headers: { Cookie: __ENV.SESSION_COOKIE } });
    check(response, { 'portfolio report returned': r => {
      if (r.status !== 200) return false;
      try { const report = r.json(); return !!report && typeof report.summary === 'object' && report.summary !== null; } catch { return false; }
    } });
  } else {
    response = http.get(`${base}/api/v1/healthz`);
    check(response, { 'service healthy': r => r.status === 200 });
  }
  if (mode !== 'webhook_burst') sleep(0.2);
}
