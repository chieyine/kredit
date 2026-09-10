import {expect,test} from '@playwright/test';

test('privacy approval acknowledges retained records without reporting an uncertain failure',async({page,context,baseURL})=>{
 await context.addCookies([{name:'kredit_session',value:'synthetic-owner-session',url:baseURL!}]);
 await page.route('**/api/v1/me',route=>route.fulfill({json:{user:{id:'owner'},organizations:[],mfa_enrolled:true}}));
 let item={id:'privacy-request',request_type:'DELETION',state:'IN_REVIEW',version:1,due_at:'2026-10-01T09:00:00Z',details:'Remove optional profile information',decided_by:''};
 await page.route('**/api/v1/ops/privacy-requests',route=>route.fulfill({json:{requests:[item]}}));
 await page.route('**/api/v1/ops/privacy-requests/privacy-request/decide',async route=>{
  const body=route.request().postDataJSON();expect(body.decision).toBe('APPROVED');expect(body.expected_version).toBe(1);
  item={...item,state:'PARTIALLY_APPROVED',version:2,decided_by:'owner'};
  await route.fulfill({json:{request:{...item,legal_hold_applies:true,retention_outcome:'Required records retained'}}});
 });
 await page.goto('/admin/privacy');
 await page.getByRole('button',{name:'Approve',exact:true}).click();
 const dialog=page.getByRole('dialog');
 await dialog.getByLabel('Why are you doing this?').fill('Retain required records and restrict optional use');
 await dialog.getByRole('button',{name:'Record decision'}).click();
 await expect(dialog).not.toBeVisible();
 await expect(page.getByRole('status')).toContainText('Partially approved');
 await expect(page.getByRole('button',{name:'Finish this request'})).toBeVisible();
});
