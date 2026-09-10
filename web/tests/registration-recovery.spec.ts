import {expect,test} from '@playwright/test';
test('admin recovers an uncertain registration and sees a truthful empty state',async({page,context,baseURL})=>{
 await context.addCookies([{name:'kredit_session',value:'registration-fixture',url:baseURL!}]);
 await page.route('**/api/v1/me',route=>route.fulfill({json:{user:{id:'admin'},organizations:[],mfa_enrolled:true}}));
 let done=false;
 await page.route('**/api/v1/ops/customer-registrations',route=>route.fulfill({json:{registrations:done?[]:[{id:'attempt',business_id:'business-reference',created_at:'2026-09-10T08:00:00Z'}]}}));
 await page.route('**/api/v1/ops/customer-registrations/attempt',route=>{
  expect(route.request().postDataJSON()).toMatchObject({action:'not_created',reason:'Provider dashboard confirms this customer was not created.'});done=true;return route.fulfill({json:{resolved:true}});
 });
 await page.setViewportSize({width:390,height:844});
 await page.goto('/admin/customer-registrations');
 await page.getByRole('button',{name:'Review attempt'}).click();
 await page.getByLabel('Provider result').selectOption('not_created');
 await page.getByLabel('Evidence checked and reason').fill('Provider dashboard confirms this customer was not created.');
 await page.getByRole('button',{name:'Save resolution'}).click();
 await expect(page.getByText('Resolution saved.')).toBeVisible();
 await expect(page.getByText('No registrations need recovery.')).toBeVisible();
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBeTruthy();
});
test('registration outage is not presented as an empty queue',async({page,context,baseURL})=>{
 await context.addCookies([{name:'kredit_session',value:'registration-fixture',url:baseURL!}]);
 await page.route('**/api/v1/me',route=>route.fulfill({json:{user:{id:'admin'},organizations:[],mfa_enrolled:true}}));
 await page.route('**/api/v1/ops/customer-registrations',route=>route.fulfill({status:503,json:{detail:'unavailable'}}));
 await page.goto('/admin/customer-registrations');
 await expect(page.getByRole('alert')).toBeVisible();
 await expect(page.getByText('No registrations need recovery.')).toHaveCount(0);
});
