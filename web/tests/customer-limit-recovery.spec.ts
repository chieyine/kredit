import {test,expect} from '@playwright/test';
const line=(id:string)=>({id,supplier_organization_id:'org',buyer_user_id:'buyer',buyer_business_id:'business',state:'ACTIVE',approved_limit_kobo:100000,current_exposure_kobo:0,reserved_pending_kobo:0,available_limit_kobo:100000,version:4});
test.beforeEach(async({page,context,baseURL})=>{
 await context.addCookies([{name:'kredit_session',value:'limit-review',url:baseURL??'http://127.0.0.1:5173'}]);
 await page.route('**/api/v1/me',r=>r.fulfill({json:{user:{id:'seller'},session:{authentication_level:'AAL2'},organizations:[]}}));
 await page.route('**/api/v1/organizations',r=>r.fulfill({json:{organizations:[{id:'org',legal_name:'Seller'}]}}));
 await page.route('**/api/v1/organizations/org/trade-lines/*/statement',r=>r.fulfill({json:{line:line(r.request().url().split('/').at(-2)!),drawdowns:[]}}));
});
test('uncertain customer-limit changes retry the original amount and version',async({page})=>{
 await page.route('**/api/v1/platform/capabilities',r=>r.fulfill({json:{features:{drawdowns:true}}}));
 const sent:{body:any;key:string}[]=[];
 await page.route('**/api/v1/organizations/org/trade-lines/first',r=>{sent.push({body:r.request().postDataJSON(),key:r.request().headers()['idempotency-key']});return r.fulfill(sent.length===1?{status:503,json:{detail:'Result unknown'}}:{json:{trade_line:{...line('first'),version:5,approved_limit_kobo:80000,available_limit_kobo:80000}}});});
 await page.goto('/app/trade-lines/first?organization=org');
 await page.getByLabel('New limit (₦)').fill('800');
 await page.getByRole('button',{name:'Change limit to ₦800.00'}).click();
 await expect(page.getByRole('button',{name:'Retry the original change'})).toBeVisible();
 await expect(page.getByLabel('New limit (₦)')).toBeDisabled();
 await page.getByRole('button',{name:'Retry the original change'}).click();
 await expect.poll(()=>sent.length).toBe(2);
 expect(sent[1]).toEqual(sent[0]);
 expect(sent[0].body).toEqual({approved_limit_kobo:80000,expected_version:4});
 await expect(page.getByRole('button',{name:'Retry the original change'})).toHaveCount(0);
});
test('customer-limit route changes clear prior input and capability outages remain retryable',async({page})=>{
 let calls=0;
 await page.route('**/api/v1/platform/capabilities',r=>r.fulfill(++calls===1?{status:503,json:{detail:'Unavailable'}}:{json:{features:{drawdowns:true}}}));
 await page.goto('/app/trade-lines/first?organization=org');
 await expect(page.getByRole('button',{name:'Check sale availability again'})).toBeVisible();
 await expect(page.getByText(/Selling from a customer limit is switched off/)).toHaveCount(0);
 await page.getByRole('button',{name:'Check sale availability again'}).click();
 await page.getByLabel('What are they buying?').fill('First limit goods');
 await page.getByLabel('Why are you pausing this limit?').fill('Reason for first limit');
 await page.evaluate(()=>{const a=document.createElement('a');a.href='/app/trade-lines/second?organization=org';a.textContent='Open second limit';document.body.append(a);});
 await page.getByRole('link',{name:'Open second limit'}).click();
 await expect(page.getByLabel('What are they buying?')).toHaveValue('');
 await expect(page.getByLabel('Why are you pausing this limit?')).toHaveValue('');
 await expect(page.getByLabel('New limit (₦)')).toHaveValue('1000.00');
});
