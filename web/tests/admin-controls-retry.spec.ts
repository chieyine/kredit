import {expect,test} from '@playwright/test';

test('unknown protected command survives reload and cannot be replaced with changed details',async({page,context,baseURL})=>{
 await context.addCookies([{name:'kredit_session',value:'synthetic-admin-session',url:baseURL!}]);
 await page.route('**/api/v1/me',route=>route.fulfill({json:{user:{id:'admin'},organizations:[],mfa_enrolled:true}}));
 await page.route('**/api/v1/ops/commands/preview',route=>route.fulfill({json:{command:{...route.request().postDataJSON(),current_version:1,impact_preview:{effect:'Suspend this synthetic user',will_notify:true,audit:'immutable'}}}}));
 const attempts:{key:string,body:unknown}[]=[];
 await page.route('**/api/v1/ops/commands',async route=>{
  attempts.push({key:route.request().headers()['idempotency-key'],body:route.request().postDataJSON()});
  if(attempts.length===1){await route.abort('failed');return;}
  await route.fulfill({json:{command:{...route.request().postDataJSON(),id:'saved-command',state:'APPLIED'}}});
 });
 const fill=async(target:string)=>{
  await expect(page.locator('form[data-ready="true"]')).toBeVisible();
  await page.getByLabel('Target ID').fill(target);
  await page.getByLabel('Structured reason').fill('Confirmed synthetic account review');
  await page.getByRole('button',{name:'Preview impact',exact:true}).click();
  await page.getByRole('button',{name:'Apply this change',exact:true}).click();
 };
 const first='00000000-0000-0000-0000-000000000202';
 await page.goto('/admin/controls');await fill(first);
 await expect(page.getByText(/We have not confirmed the result/)).toBeVisible();
 await fill('00000000-0000-0000-0000-000000000203');
 await expect(page.getByText(/Your earlier request may already have completed/)).toBeVisible();
 expect(attempts).toHaveLength(1);
 await page.reload();await fill(first);
 await expect(page.getByText('Done. Reference saved-command')).toBeVisible();
 expect(attempts).toHaveLength(2);expect(attempts[1]).toEqual(attempts[0]);
});

test('document recovery uses the version returned by the protected preview', async ({page,context,baseURL}) => {
 await context.addCookies([{name:'kredit_session',value:'synthetic-admin-session',url:baseURL!}]);
 await page.route('**/api/v1/me',route=>route.fulfill({json:{user:{id:'admin'},organizations:[],mfa_enrolled:true}}));
 await page.route('**/api/v1/ops/commands/preview',route=>route.fulfill({json:{command:{...route.request().postDataJSON(),current_version:7,impact_preview:{effect:'Queue a fresh scan; file stays blocked.',will_notify:false,audit:'immutable'}}}}));
 let submitted:any;
 await page.route('**/api/v1/ops/commands',route=>{submitted=route.request().postDataJSON();return route.fulfill({json:{command:{...submitted,id:'scan-review',state:'APPLIED'}}});});
 await page.goto('/admin/controls?target_type=document&target_id=scan-document');
 await expect(page.getByRole('combobox',{name:'Action',exact:true})).toHaveValue('retry_document_scan');
 await page.getByLabel('Structured reason').fill('Scanner restored after an outage');
 await page.getByRole('button',{name:'Preview impact',exact:true}).click();
 await expect(page.getByText('Queue a fresh scan; file stays blocked.')).toBeVisible();
 await page.getByRole('button',{name:'Apply this change',exact:true}).click();
 await expect(page.getByText('Done. Reference scan-review')).toBeVisible();
 expect(submitted).toMatchObject({command_type:'retry_document_scan',target_type:'document',target_id:'scan-document',expected_version:7});
});
