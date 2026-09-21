import { strict as assert } from 'node:assert';
import { test } from 'node:test';
import { publicReply } from './public-api.mjs';

test('public fixture supplies only known read contracts', () => {
  assert.deepEqual(publicReply('GET', '/api/v1/website/terms'), [200, { publication: null }]);
  assert.equal(publicReply('GET', '/api/v1/pricing')[1].base_bps, 50);
  assert.equal(publicReply('POST', '/api/v1/pricing')[0], 405);
  for (const path of ['/api/v1/me', '/api/v1/payments', '/api/v1/website/unknown']) {
    assert.equal(publicReply('GET', path)[0], 503);
  }
});
test('public fixture rejects credential forwarding and models publication failures', () => {
  for (const headers of [{ cookie: 'synthetic-session' }, { authorization: 'synthetic' }]) {
    assert.equal(publicReply('GET', '/api/v1/website/privacy', headers)[0], 400);
  }
  assert.equal(publicReply('GET', '/api/v1/website/terms?version=audit-outage')[0], 503);
  assert.equal(publicReply('GET', '/api/v1/website/terms?version=missing')[0], 404);
  assert.deepEqual(publicReply('GET', '/api/v1/website/terms?version=audit-malformed')[1], { publication: {} });
});
