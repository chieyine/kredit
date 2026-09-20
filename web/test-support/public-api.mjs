import { createServer } from 'node:http';
import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';

const pages = new Set(['home', 'faq', 'pricing', 'contact', 'terms', 'privacy', 'complaints']);

/** A deliberately read-only, stateless public API contract for isolated browser tests. */
export function publicReply(method, path, headers = {}) {
  const url = new URL(path, 'http://127.0.0.1:5174');
  if (method !== 'GET') return [405, { error: 'The fixture never accepts writes.' }];
  if (url.pathname === '/healthz') return [200, { fixture: 'kredit-public-api' }];
  if (headers.cookie || headers.authorization) return [400, { error: 'Public reads must omit credentials.' }];
  if (url.pathname === '/api/v1/pricing') {
    return [200, { policy_revision: 1, base_bps: 50, collection_bps: 50, min_fee_kobo: 0 }];
  }
  const page = /^\/api\/v1\/website\/([a-z]+)$/.exec(url.pathname)?.[1];
  if (page && pages.has(page)) {
    const version = url.searchParams.get('version');
    if (version === 'audit-outage') return [503, { error: 'Synthetic publication outage.' }];
    if (version === 'audit-malformed') return [200, { publication: {} }];
    if (version) return [404, { error: 'No such publication in the fixture.' }];
    // No override means use the repository's initial publication; it is not a new approval.
    return [200, { publication: null }];
  }
  // Missing mocks must fail visibly, never become invented accounts or financial success.
  return [503, { error: 'This endpoint is not implemented by the public test fixture.' }];
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  if (process.env.KREDIT_PUBLIC_API_FIXTURE !== '1' || process.env.APP_ENV?.trim().toLowerCase() === 'production') {
    throw new Error('This loopback-only fixture must be explicitly enabled in a test process.');
  }
  const server = createServer((req, res) => {
    const [status, body] = publicReply(req.method, req.url, req.headers);
    res.writeHead(status, { 'content-type': 'application/json', 'cache-control': 'no-store' });
    res.end(JSON.stringify(body));
  });
  server.listen(5174, '127.0.0.1');
  for (const signal of ['SIGINT', 'SIGTERM']) process.once(signal, () => server.close());
}
