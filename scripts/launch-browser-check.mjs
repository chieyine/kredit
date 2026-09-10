import { chromium } from '../web/node_modules/@playwright/test/index.mjs';
import assert from 'node:assert/strict';
const browser = await chromium.launch({executablePath:process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE || undefined,headless:true});
const page = await browser.newPage({viewport:{width:1440,height:1000}});
const base=process.env.PLAYWRIGHT_BASE_URL || 'http://127.0.0.1:5173';
let sessionRequests=0;page.on('request',r=>{if(r.url().endsWith('/api/v1/me'))sessionRequests++});
try {
 for (const [name,path] of [['home','/'],['pricing','/pricing'],['terms','/legal/terms'],['privacy','/legal/privacy'],['complaints','/legal/complaints'],['signin','/app']]) {
  const response=await page.goto(base+path); assert.equal(response.status(),200,path);
  await page.locator('h1').waitFor();
  const text=await page.locator('body').innerText();
  assert.doesNotMatch(text,/pre-launch|approval pending|Before you read all this|Check session again|current session|Invalid Date/i);
  if(name==='pricing') {assert.match(text,/0.5%/);assert.doesNotMatch(text,/temporarily unavailable/)}
  if(['terms','privacy','complaints'].includes(name)) assert.match(text,/hello@kredit.com.ng/);
  await page.screenshot({path:`docs/launch-readiness/evidence/after-${name}.png`,fullPage:true});
  console.log('PASS rendered',path,'1440x1000');
 }
 assert.equal(sessionRequests,0,'ordinary guests must not request /me');
 for(const width of [320,360,390,430,768,1024,1440]) {
  await page.setViewportSize({width,height:900});await page.goto(base+'/legal/terms');
  assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),`overflow at ${width}`);
  const links=await page.locator('.document nav a').evaluateAll(els=>els.map(a=>a.getAttribute('href')));
  for(const link of links) assert(await page.locator(link).count(),`missing anchor ${link}`);
  if(width===390){await page.screenshot({path:'docs/launch-readiness/evidence/after-terms-mobile-closed.png'});await page.locator('.mobile-contents summary').click();await page.screenshot({path:'docs/launch-readiness/evidence/after-terms-mobile-open.png'});await page.locator('.mobile-contents a').last().click();assert.equal(await page.locator('.mobile-contents').getAttribute('open'),null)}
  console.log('PASS legal reflow and anchors',width);
 }
 await page.goto(base+'/admin/platform-settings');assert.match(page.url(),/\/app\?next=/);console.log('PASS protected guest redirect');
} finally {await browser.close()}
