import http from 'k6/http';
import { check, sleep } from 'k6';

const base = __ENV.BASE_URL || 'http://127.0.0.1:8080';
const mode = __ENV.PERFORMANCE_MODE || 'smoke';
const profiles = {
  smoke: { vus: 5, duration: '30s' },
  portfolio: { vus: 50, duration: '5m' },
  webhook_burst: { scenarios: { burst: { executor: 'shared-iterations', vus: 200, iterations: 10000, maxDuration: '2m' } } }
};
export const options = { ...(profiles[mode] || profiles.smoke), thresholds: { http_req_failed: ['rate<0.01'], http_req_duration: ['p(95)<1000'] } };

export default function () {
  const response = mode === 'webhook_burst'
    ? http.post(`${base}/api/v1/webhooks/collection/mock-collection`, JSON.stringify({ provider_event_id: `perf-${__VU}-${__ITER}`, provider_collection_id: 'unknown', state: 'pending' }), { headers: { 'Content-Type': 'application/json', 'X-Provider-Signature': 'performance-placeholder' } })
    : http.get(`${base}/api/v1/healthz`);
  check(response, { 'bounded response': (r) => r.status >= 200 && r.status < 500 });
  if (mode !== 'webhook_burst') sleep(0.2);
}
