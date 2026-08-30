import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = { vus: 5, duration: '15s', thresholds: { http_req_failed: ['rate<0.01'], http_req_duration: ['p(95)<900'] } };
export default function () {
  const base = __ENV.BASE_URL || 'http://localhost:8080';
  const response = http.get(`${base}/api/v1/healthz`);
  check(response, { 'health is 200': (r) => r.status === 200 });
  sleep(1);
}
