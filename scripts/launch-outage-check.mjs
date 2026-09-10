import {chromium} from '../web/node_modules/@playwright/test/index.mjs';
import assert from 'node:assert/strict';
const browser=await chromium.launch({executablePath:process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE || undefined,headless:true});
const base=process.env.PLAYWRIGHT_BASE_URL || 'http://127.0.0.1:3001';
const page=await browser.newPage({viewport:{width:1440,height:1000}});
try {
 for (const path of ['/','/legal/terms','/legal/privacy','/security','/blog','/app']) {
  const response=await page.goto(base+path);assert.equal(response.status(),200);
  assert.doesNotMatch(await page.locator('body').innerText(),/current session|Check session again|approval pending/i);
  console.log('PASS API unavailable, public route',path);
 }
 await page.getByLabel('Phone number',{exact:true}).fill('08012345678');
 await page.getByRole('button',{name:'Send me a code'}).click();
 await page.getByRole('alert').waitFor();
 assert.match(await page.getByRole('alert').innerText(),/could not confirm/);
 assert.equal(await page.getByLabel('Six-digit code',{exact:true}).count(),0);
 console.log('PASS failed sign-in does not invent delivery');
 await page.goto(base+'/pricing');
 assert.match(await page.locator('body').innerText(),/Pricing is temporarily unavailable/);
 assert.equal(await page.locator('.rates').count(),0);
 await page.screenshot({path:'docs/launch-readiness/evidence/pricing-source-outage.png'});
 console.log('PASS missing pricing source does not fabricate rates');
 const response=await page.request.get(base+'/api/v1/me');assert.equal(response.status(),503);
 console.log('PASS protected API outage remains unavailable');
} finally {await browser.close()}
