import { test, expect } from '@playwright/test';
import { MutationIntent, MutationError } from '../src/lib/api/mutation';
import { LatestRequest, rows, record, readResource, normalizeNigerianPhone, safeNext, clearPrivateBrowserData } from '../src/lib/api/reliable';
import { readDraft, saveDraft } from '../src/lib/sale-drafts';
import { acceptanceMessage, hostedAuthorizationURL, disputeEffectCopy } from '../src/lib/financial-copy';
import { feeDisclosure, feeForKobo, baseFeeForKobo } from '../src/lib/fee-terms';
import { dateLabel, receivables, saleView, workRow } from '../src/lib/records';
import { attentionItems } from '../src/lib/attention';
import { DEMO_SALE, DEMO_BALANCE_KOBO, DEMO_PAID_PERCENT } from '../src/lib/demo-sale';

class MemoryStorage implements Storage {
  data = new Map<string, string>();
  get length() { return this.data.size; }
  clear() { this.data.clear(); }
  getItem(key: string) { return this.data.get(key) ?? null; }
  key(index: number) { return [...this.data.keys()][index] ?? null; }
  removeItem(key: string) { this.data.delete(key); }
  setItem(key: string, value: string) { this.data.set(key, String(value)); }
}
const originalFetch = globalThis.fetch;
const navigatorDescriptor = Object.getOwnPropertyDescriptor(globalThis, 'navigator');
test.beforeEach(() => { Object.defineProperty(globalThis, 'navigator', { configurable: true, value: { onLine: true } }); });
test.afterEach(() => {
  globalThis.fetch = originalFetch;
  if (navigatorDescriptor) Object.defineProperty(globalThis, 'navigator', navigatorDescriptor);
  else Reflect.deleteProperty(globalThis, 'navigator');
});
const json = (value: unknown, status = 200) => new Response(JSON.stringify(value), { status, headers: { 'Content-Type': 'application/json' } });

test('failed, malformed and missing financial responses stay unavailable, not empty', async () => {
  for (const response of [json({ detail: 'unavailable' }, 503), json({}), json({ payments: null }), new Response('<html>Error</html>')]) {
    globalThis.fetch = async () => response;
    const result = await readResource('org-a', '/payments', rows('payments', record));
    expect(result.state).toBe('error');
    expect(result.scope).toBe('org-a');
    expect('data' in result).toBe(false);
  }
  globalThis.fetch = async () => json({ payments: [] });
  expect(await readResource('org-a', '/payments', rows('payments', record))).toMatchObject({ state: 'ready', data: [] });
});

test('changing business or unmounting invalidates an older response', () => {
  const controller = new LatestRequest(), a = controller.begin(), b = controller.begin();
  expect(a.signal.aborted).toBe(true); expect(a.current()).toBe(false); expect(b.current()).toBe(true);
  controller.cancel(); expect(b.current()).toBe(false);
});

test('lost response reuses exactly one mutation identity even after navigation', async () => {
  const store = new MemoryStorage(), keys: string[] = [], bodies: string[] = [];
  globalThis.fetch = async (_url, init) => { keys.push(new Headers(init?.headers).get('Idempotency-Key')!); bodies.push(String(init?.body)); throw new TypeError('connection lost after commit'); };
  const first = new MutationIntent('user-a', '/payments', store);
  await expect(first.run({ amount_kobo: 10049, reference: 'TEST-ONLY' }, record)).rejects.toMatchObject({ outcome: 'unknown' });
  expect(first.unresolved).toBe(true);
  const restored = new MutationIntent('user-a', '/payments', store);
  globalThis.fetch = async (_url, init) => { keys.push(new Headers(init?.headers).get('Idempotency-Key')!); bodies.push(String(init?.body)); return json({ id: 'payment-one' }); };
  expect(await restored.run({ reference: 'TEST-ONLY', amount_kobo: 10049 }, record)).toEqual({ id: 'payment-one' });
  expect(keys[0]).toBe(keys[1]); expect(bodies[0]).toBe(bodies[1]); expect(store.length).toBe(0);
});

test('changed details cannot silently replace an unresolved payment', async () => {
  let calls = 0; globalThis.fetch = async () => { calls++; throw new TypeError('network'); };
  const intent = new MutationIntent('user', '/payments', new MemoryStorage());
  await expect(intent.run({ amount: 100 }, record)).rejects.toMatchObject({ outcome: 'unknown' });
  await expect(intent.run({ amount: 101 }, record)).rejects.toMatchObject({ outcome: 'unknown' });
  expect(calls).toBe(1);
});

test('unknown identities are never silently expired and replaced', async () => {
  const store = new MemoryStorage(); let calls = 0;
  globalThis.fetch = async () => { calls++; throw new Error('unknown'); };
  await expect(new MutationIntent('user', '/payments', store).run({ amount: 100 }, record)).rejects.toThrow();
  const key = store.key(0)!, saved = JSON.parse(store.getItem(key)!); saved.createdAt = new Date(Date.now() - 16 * 60000).toISOString(); store.setItem(key, JSON.stringify(saved));
  await expect(new MutationIntent('user', '/payments', store).run({ amount: 100 }, record)).rejects.toMatchObject({ outcome: 'unknown' });
  expect(calls).toBe(1);
});

test('two fast submissions do not send two mutations', async () => {
  let release: () => void = () => {}; let calls = 0;
  globalThis.fetch = async () => { calls++; await new Promise<void>(resolve => { release = resolve; }); return json({ id: 'one' }); };
  const intent = new MutationIntent('user', '/receipt', new MemoryStorage());
  const first = intent.run({ state: 'confirmed' }, record);
  await expect(intent.run({ state: 'confirmed' }, record)).rejects.toBeInstanceOf(MutationError);
  await expect.poll(() => calls).toBe(1); release(); await first;
});

test('offline, malformed saved identity and unavailable storage fail before a write', async () => {
  let calls = 0; globalThis.fetch = async () => { calls++; return json({}); };
  Object.defineProperty(globalThis, 'navigator', { configurable: true, value: { onLine: false } });
  await expect(new MutationIntent('u', '/payments', new MemoryStorage()).run({}, record)).rejects.toMatchObject({ outcome: 'not_sent' });
  Object.defineProperty(globalThis, 'navigator', { configurable: true, value: { onLine: true } });
  const store = new MemoryStorage(); store.setItem('kredit.intent.v1:u:%2Fpayments', '{broken');
  await expect(new MutationIntent('u', '/payments', store).run({}, record)).rejects.toMatchObject({ outcome: 'unknown' });
  const unavailable = new MemoryStorage(); unavailable.setItem = () => { throw new Error('storage unavailable'); };
  await expect(new MutationIntent('u', '/payments', unavailable).run({}, record)).rejects.toMatchObject({ outcome: 'not_sent' });
  expect(calls).toBe(0);
});

test('malformed success and server errors retain the unresolved identity', async () => {
  for (const response of [new Response('<html>bad</html>'), json({ code: 'idempotency_in_progress' }, 409), json({}, 500)]) {
    const store = new MemoryStorage(); globalThis.fetch = async () => response;
    const intent = new MutationIntent('user', '/payment', store);
    await expect(intent.run({}, record)).rejects.toMatchObject({ outcome: 'unknown' });
    expect(intent.unresolved).toBe(true); expect(store.length).toBe(1);
  }
});

test('request storage contains a digest, not personal or financial form fields', async () => {
  const store = new MemoryStorage(); globalThis.fetch = async () => { throw new Error('unknown'); };
  await expect(new MutationIntent('user-a', '/payment', store).run({ secretEvidence: 'DO-NOT-RETAIN', amount_kobo: 9876543 }, record)).rejects.toThrow();
  const saved = store.getItem(store.key(0)!)!;
  expect(saved).not.toContain('DO-NOT-RETAIN'); expect(saved).not.toContain('9876543');
  expect(Object.keys(JSON.parse(saved)).sort()).toEqual(['createdAt', 'fingerprint', 'key', 'version']);
});

test('drafts are isolated by account and business and expire', () => {
  const store = new MemoryStorage(), now = Date.now(), draft = { goods: '40 cartons', principal: '127,500.49', dueDate: '2026-09-18' };
  expect(saveDraft('u-a', 'org-a', draft, store, now)).toBe(true);
  expect(readDraft('u-a', 'org-a', store, now)).toEqual(draft);
  expect(readDraft('u-b', 'org-a', store, now)).toBeNull();
  expect(readDraft('u-a', 'org-b', store, now)).toBeNull();
  expect(readDraft('u-a', 'org-a', store, now + 12 * 3600000)).toBeNull();
});

test('drafts reject tampering and discard the legacy unscoped record', () => {
  const store = new MemoryStorage();
  store.setItem('kredit.quick-sale.draft.v1', JSON.stringify({ goods: 'old private draft' }));
  saveDraft('a', 'one', { goods: 'test', principal: '100', dueDate: '' }, store);
  const key = store.key(1)!; const value = JSON.parse(store.getItem(key)!); value.userID = 'b'; store.setItem(key, JSON.stringify(value));
  expect(readDraft('a', 'one', store)).toBeNull(); expect(store.getItem('kredit.quick-sale.draft.v1')).toBeNull();
});

for (const phone of ['0803 123 4567', '2348031234567', '+234 (803) 123-4567']) test(`normalises Nigerian phone ${phone}`, () => { expect(normalizeNigerianPhone(phone)).toBe('+2348031234567'); });
for (const phone of ['123', '+12025551234', '080312345678', '80letters']) test(`rejects invalid Nigerian phone ${phone}`, () => { expect(() => normalizeNigerianPhone(phone)).toThrow(); });

test('post-login destinations stay on the same origin', () => {
  for (const path of ['//evil.test', '/\\evil.test', '/app', 'https://evil.test', '/\u0000evil']) expect(safeNext(path, 'https://kredit.com.ng')).toBe('/app/overview');
  expect(safeNext('/buyer/credit-requests/abc?review=1', 'https://kredit.com.ng')).toBe('/buyer/credit-requests/abc?review=1');
});

test('hosted permission only links to the approved HTTPS provider origin', () => {
  expect(hostedAuthorizationURL('https://authorise.mono.co/1234', 'mono-sweep')).toBe('https://authorise.mono.co/1234');
  for (const url of ['javascript:alert(1)', 'http://authorise.mono.co/a', 'https://authorise.mono.co.evil.test/a', 'https://evil.test/a', 'https://user@authorise.mono.co/a', 'https://authorise.mono.co:8443/a', 'https://authorise.mono.co/']) expect(hostedAuthorizationURL(url, 'mono-sweep')).toBeNull();
});

test('acceptance and active bank permission are not treated as the same state', () => {
  expect(acceptanceMessage('BUYER_ACCEPTED')).not.toContain('can arrange');
  expect(acceptanceMessage('BUYER_ACCEPTED')).toContain('Bank permission must be ready');
  expect(acceptanceMessage('READY_TO_RELEASE')).toContain('can arrange');
});

test('partial disputes never promise a blanket debit stop', () => {
  expect(disputeEffectCopy('CONTESTED_ONLY', 10049)).toContain('₦100.49');
  expect(disputeEffectCopy('CONTESTED_ONLY')).toContain('Undisputed amounts may still be collected');
  expect(disputeEffectCopy('FULL_BLOCK')).toContain('already sent to the bank may still complete');
  expect(disputeEffectCopy('NO_AUTOMATIC_BLOCK')).toContain('does not request an automatic hold');
});

test('prices retain exact kobo, fee floors and caps', () => {
  expect(feeForKobo(12750049, 50)).toBe(63750n);
  expect(baseFeeForKobo(10049, { policy_revision: 1, base_bps: 50, collection_bps: 50, min_fee_kobo: 1000 })).toBe(1000n);
  expect(baseFeeForKobo(100, { policy_revision: 1, base_bps: 50, collection_bps: 50, min_fee_kobo: 1000 })).toBe(100n);
  expect(feeForKobo('9007199254740993', 50)).toBe(45035996273704n);
  expect(feeDisclosure(undefined)).toContain('unavailable');
  expect(feeForKobo(-1, 50)).toBeNull(); expect(feeForKobo(100, 1001)).toBeNull();
});

test('financial summaries reject missing, negative and unsafe amounts', () => {
  for (const summary of [{}, { obligation_count: 1, outstanding_kobo: -1, overdue_kobo: 0 }, { obligation_count: 1, outstanding_kobo: Number.MAX_SAFE_INTEGER + 1, overdue_kobo: 0 }]) expect(() => receivables({ summary })).toThrow();
  expect(receivables({ summary: { obligation_count: 1, outstanding_kobo: 10049, overdue_kobo: 0 } }).outstanding_kobo).toBe(10049);
});

test('the entire attention queue counts disputes before ordinary drafts', () => {
  const sales = Array.from({ length: 12 }, (_, index) => saleView({ request: { id: `draft-${index}`, state: 'DRAFT', buyer_legal_name: 'Amina Stores', principal_kobo: 10049, due_date: '2026-09-18' } }));
  const dispute = workRow({ id: 'dispute-1', state: 'OPEN', reason: 'Five cartons missing', amount_kobo: 10049 });
  const items = attentionItems('org-a', sales, [], [], [dispute]);
  expect(items).toHaveLength(13); expect(items[0].id).toBe('dispute-dispute-1');
  expect(items[0].href).toContain('organization=org-a');
});

test('date labels reject invalid calendar dates', () => {
  expect(dateLabel('2026-02-31')).toBe('Date unavailable');
  expect(dateLabel('2026-09-18')).toContain('2026');
});


test('every sample uses one internally consistent sale, not invented activity', () => {
  expect(DEMO_SALE.principalKobo - DEMO_SALE.paidKobo).toBe(DEMO_BALANCE_KOBO);
  expect(DEMO_BALANCE_KOBO).toBe(80000000);
  expect(DEMO_PAID_PERCENT).toBeCloseTo(33.3333333);
  expect(DEMO_SALE.reference).toContain('EXAMPLE');
});
