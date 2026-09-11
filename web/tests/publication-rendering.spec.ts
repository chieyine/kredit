import {expect,test} from '@playwright/test';
import {createServer,type Server} from 'node:http';
import {websiteDefaults} from '../src/lib/website-editor-content';
let api:Server;
const slug='how-to-sell-goods-on-credit-in-nigeria';
const copy=structuredClone(websiteDefaults[`guide-${slug}`]);
copy.title='Reviewed guide from the owner editor';copy.guide!.description='A newly published guide description.';copy.guide!.category='Payments';
const contact=structuredClone(websiteDefaults.contact);contact.contact!.support_email='helpdesk@example.test';
const publication=(copy:any)=>({copy,version:2,published_at:'2026-09-09T09:00:00Z',content_hash:'a'.repeat(64)});
test.beforeAll(async()=>{
 test.skip(!process.env.EDITORIAL_FIXTURE_PORT,'requires local preview configured to the isolated editorial API fixture');
 api=createServer((req,res)=>{res.setHeader('Content-Type','application/json');if(req.url==='/api/v1/website-guides')res.end(JSON.stringify({publications:{[slug]:publication(copy)}}));else if(req.url==='/api/v1/website/contact')res.end(JSON.stringify({publication:publication(contact)}));else{res.statusCode=404;res.end('{}')}});
 await new Promise<void>((resolve,reject)=>{api.once('error',reject);api.listen(Number(process.env.EDITORIAL_FIXTURE_PORT),'127.0.0.1',resolve)});
});
test.afterAll(async()=>{if(api)await new Promise<void>((resolve,reject)=>api.close(err=>err?reject(err):resolve()))});
test('published guide reaches its page, listings, topic, feed and search metadata',async({page,request})=>{
 await page.goto(`/blog/${slug}`);await expect(page.getByRole('heading',{level:1})).toHaveText(copy.title);await expect(page).toHaveTitle(copy.title);await expect(page.locator('meta[name="description"]')).toHaveAttribute('content',copy.guide!.description);
 await page.goto('/blog');await expect(page.getByRole('heading',{name:copy.title}).first()).toBeVisible();
 await page.goto('/blog/topic/payments');await expect(page.getByRole('heading',{name:copy.title})).toBeVisible();
 expect(await (await request.get('/blog/rss.xml')).text()).toContain(copy.title);
 expect(await (await request.get('/sitemap.xml')).text()).toContain(`<loc>https://kredit.ng/blog/${slug}</loc><lastmod>2026-09-09</lastmod>`);
});
test('published contact details render safe email links and responsive content',async({page})=>{
 await page.setViewportSize({width:390,height:844});await page.goto('/contact');
 await expect(page.getByRole('link',{name:'helpdesk@example.test'})).toHaveAttribute('href','mailto:helpdesk@example.test');
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
});
