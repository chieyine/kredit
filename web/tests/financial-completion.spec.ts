import {expect,test} from '@playwright/test';

test('financial review shows unresolved evidence and closes only after records agree',async({page,context,baseURL})=>{
 await context.addCookies([{name:'kredit_session',value:'review-session',url:baseURL??'http://127.0.0.1:5173'}]);
 await page.route('**/api/v1/me',route=>route.fulfill({json:{user:{id:'operator'},session:{authentication_level:'AAL2'},organizations:[]}}));
 const item={id:'review-1',kind:'balance',target_id:'sale-1',expected:'100000',actual:'99000',owner_id:null as string|null,history:[{id:'event-1',action:'DETECTED',reason:'Financial records disagree',occurred_at:'2026-09-01T12:00:00Z'}]};
 let resolved=false,agrees=false;
 await page.route('**/api/v1/ops/financial-reconciliation',route=>route.fulfill({json:{cases:resolved?[]:[item]}}));
 await page.route('**/api/v1/ops/financial-reconciliation/review-1/decision',async route=>{
  expect(route.request().headers()['idempotency-key']).toBeTruthy();
  const body=route.request().postDataJSON();
  if(body.action==='claim'){item.owner_id='operator';await route.fulfill({json:{status:'applied'}})}
  else if(!agrees){await route.fulfill({status:409,json:{detail:'Financial discrepancy remains unresolved'}})}
  else{resolved=true;await route.fulfill({json:{status:'applied'}})}
 });
 await page.goto('/admin/reconciliation');
 await expect(page.getByText('₦1,000.00',{exact:true})).toBeVisible();
 await expect(page.getByRole('button',{name:'Close resolved review'})).toBeDisabled();
 await page.getByLabel('Investigation notes').fill('Checked the supporting payment evidence');
 await page.getByRole('button',{name:'Claim review'}).click();
 await expect(page.getByText('Assigned to a reviewer')).toBeVisible();
 await page.getByRole('button',{name:'Close resolved review'}).click();
 await expect(page.getByRole('alert')).toHaveText('Financial discrepancy remains unresolved');
 agrees=true;
 await page.getByRole('button',{name:'Close resolved review'}).click();
 await expect(page.getByText('No open financial reviews.')).toBeVisible();
});

test('a failed financial read never displays an empty successful payment history',async({page,context,baseURL})=>{
 await context.addCookies([{name:'kredit_session',value:'payments-session',url:baseURL??'http://127.0.0.1:5173'}]);
 await page.route('**/api/v1/me',route=>route.fulfill({json:{user:{id:'seller'},session:{id:'session'},organizations:[]}}));
 await page.route('**/api/v1/organizations',route=>route.fulfill({json:{organizations:[{id:'org',legal_name:'Test Supplier'}]}}));
 await page.route('**/api/v1/organizations/org/payments',route=>route.fulfill({status:503,json:{detail:'Financial data unavailable'}}));
 await page.route('**/api/v1/organizations/org/payment-claims',route=>route.fulfill({json:{payment_claims:[]}}));
 await page.goto('/app/payments');
 await expect(page.getByText('We could not open your payment records.')).toBeVisible();
 await expect(page.getByText('No payment has arrived yet.')).toHaveCount(0);
});
