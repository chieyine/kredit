import { expect, test } from '@playwright/test';
import { websiteDefaults } from '../src/lib/website-editor-content';

for(const selected of ['home','terms','contact','guide-how-to-sell-goods-on-credit-in-nigeria'] as const) test(`owner saves, reviews and publishes ${selected} without interpreting editor text as HTML`, async ({page,context,baseURL})=>{
 await context.addCookies([{name:'kredit_session',value:'synthetic-owner-session',url:baseURL!}]);
 await page.route('**/api/v1/me',route=>route.fulfill({json:{user:{id:'owner'},organizations:[],mfa_enrolled:true}}));
 let version=0;let content:any=null;const actions:string[]=[];
 await page.route('**/api/v1/ops/platform-settings/history?*',route=>route.fulfill({json:{history:[]}}));
 await page.route('**/api/v1/ops/website/*',async route=>{
  if(!route.request().url().endsWith(`/${selected}`)){await route.fulfill({json:{version:0,content:null}});return;}
  if(route.request().method()==='GET'){await route.fulfill({json:{version,content}});return;}
  const body=route.request().postDataJSON();expect(body.expected_version).toBe(version);expect(route.request().headers()['idempotency-key']).toBeTruthy();
  actions.push(body.action);version++;
  if(body.action==='save')content={draft:body.copy};
  else {expect(body.copy).toEqual(content.draft);content.published={copy:body.copy,version,published_at:'2026-09-09T09:00:00Z',content_hash:'a'.repeat(64),...(selected==='terms'?{document_version:`legal-terms-v${version}`,effective_date:'2026-09-09'}:{})};}
  await route.fulfill({json:{version,content}});
 });
 await page.goto('/admin/website');
 await expect(page.getByLabel('Title',{exact:true})).toHaveValue(websiteDefaults.home.title);
 if(selected!=='home'){await page.getByRole('combobox',{name:'Page',exact:true}).selectOption(selected);await expect(page.getByLabel('Title',{exact:true})).toHaveValue(websiteDefaults[selected].title);if(selected!=='contact')await expect(page.getByLabel('Section text').first()).toHaveValue(websiteDefaults[selected].sections[0].body);}
 await expect(page.getByRole('button',{name:'Preview saved copy'})).toBeDisabled();
 await page.getByLabel('Title',{exact:true}).fill('<img src=x onerror=alert(1)>');
 await page.getByLabel('Reason for this change').fill('Clarify the first headline');
 await page.getByRole('button',{name:'Save draft',exact:true}).click();
 await expect(page.getByRole('status')).toContainText('Draft saved');
 expect(actions).toEqual(['save']);expect(content.published).toBeUndefined();
 await page.getByRole('button',{name:'Preview saved copy'}).click();
 const preview=page.getByRole('region',{name:'Saved copy preview'});
 await expect(preview.getByRole('heading',{level:2})).toContainText('<img src=x onerror=alert(1)>');
 await expect(preview.locator('img')).toHaveCount(0);
 await expect(page.getByRole('button',{name:'Publish reviewed copy'})).toBeDisabled();
 await page.getByLabel('Reason for this change').fill('Publish the reviewed headline');
 await page.getByRole('button',{name:'Publish reviewed copy'}).click();
 await expect(page.getByRole('status')).toContainText('Published');expect(actions).toEqual(['save','publish']);
 if(selected==='terms')await expect(page.getByRole('link',{name:/Read published document/})).toHaveAttribute('href','/legal/terms?version=legal-terms-v2');
});
