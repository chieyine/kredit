import {test,expect} from '@playwright/test';
test.use({timezoneId:'America/New_York'});
test.beforeEach(async({page,context,baseURL})=>{await context.addCookies([{name:'kredit_session',value:'audit-session',url:baseURL??'http://127.0.0.1:5173'}]);await page.route('**/api/v1/me',r=>r.fulfill({json:{user:{id:'audit-user',status:'active'},session:{authentication_level:'AAL2'},organizations:[]}}))});
test('draft keeps the same instant when loaded in another timezone',async({page})=>{const draft={id:'tz-request',state:'DRAFT',version:1,buyer_legal_name:'Buyer',principal_kobo:10000,goods_description:'Stock',due_date:'2026-10-02',collection_at:'2026-10-02T08:00:00Z',grace_hours:24};let saved:any;await page.route('**/api/v1/organizations/org-tz/credit-requests/tz-request',async route=>{if(route.request().method()==='PATCH'){saved=route.request().postDataJSON();await route.fulfill({json:{request:{...draft,...saved,version:2}}});return}await route.fulfill({json:{request:draft,receipts:[]}})});await page.goto('/app/credit/tz-request?organization=org-tz');await expect(page.getByLabel('Bank debit may be considered from (Nigerian time)')).toHaveValue('2026-10-02T09:00');await page.getByRole('button',{name:'Save for later'}).click();await expect.poll(()=>saved?.collection_at).toBe('2026-10-02T08:00:00.000Z')});
test('invoice button follows the signed download returned by the API',async({page})=>{const request={id:'invoice-request',version:1,state:'SENT',buyer_legal_name:'Buyer',principal_kobo:10000,goods_description:'Stock',due_date:'2026-10-02',collection_at:'2026-10-03T08:00:00Z',grace_hours:24,invoice_document_id:'doc-1'};await page.route('**/api/v1/organizations/org-1/credit-requests/invoice-request',r=>r.fulfill({json:{request,receipts:[]}}));await page.route('**/api/v1/organizations/org-1/documents/doc-1/download',r=>r.fulfill({json:{url:'/signed-invoice.pdf'}}));await page.route('**/signed-invoice.pdf',r=>r.abort());await page.goto('/app/credit/invoice-request?organization=org-1');const opened=page.waitForRequest(r=>r.url().endsWith('/signed-invoice.pdf'));await page.getByRole('button',{name:'Open the invoice'}).click();expect((await opened).url()).toMatch(/signed-invoice\.pdf$/)});
test('invitation changes a rejected key but retains an uncertain key',async({page})=>{const keys:string[]=[];let attempts=0;await page.route('**/api/v1/buyer-invitations/token',r=>r.fulfill({json:{invitation:{proposed_legal_name:'Buyer Ltd',proposed_business_type:'limited',proposed_address:'Lagos',proposed_industry:'retail',expires_at:'2027-01-01'},supplier:{legal_name:'Supplier',trading_name:''}}}));await page.route('**/api/v1/buyer-invitations/token/otp',r=>r.fulfill({json:{challenge_id:'challenge',development_code:'123456'}}));await page.route('**/api/v1/buyer-invitations/token/accept',route=>{keys.push(route.request().headers()['idempotency-key']);attempts++;return route.fulfill({status:attempts===1?422:503,json:{title:attempts===1?'invalid_request':'unavailable',detail:'Try again'}})});await page.goto('/buyer-invitations/token');await page.getByRole('button',{name:'Send me my code'}).click();await page.getByLabel('Full name').fill('Buyer Name');await page.getByLabel('The six-digit code we sent you').fill('123456');for(let i=0;i<3;i++){await page.getByRole('button',{name:'Yes, this is my business'}).click();await expect.poll(()=>keys.length).toBe(i+1)}expect(keys[0]).not.toBe(keys[1]);expect(keys[1]).toBe(keys[2])});

test('invitation can resend a code and clears state when its link changes', async ({page}) => {
 let sends=0;
 for(const token of ['first','second']) {
  await page.route(`**/api/v1/buyer-invitations/${token}`,route=>route.fulfill({json:{invitation:{proposed_legal_name:`${token} buyer`,proposed_business_type:'limited',proposed_address:'Lagos',proposed_industry:'retail',expires_at:'2027-01-01'},supplier:{legal_name:`${token} seller`,trading_name:''}}}));
  await page.route(`**/api/v1/buyer-invitations/${token}/otp`,route=>route.fulfill({json:{challenge_id:`challenge-${++sends}`}}));
 }
 await page.goto('/buyer-invitations/first');
 await page.getByRole('button',{name:'Send me my code'}).click();
 await page.getByLabel('Full name').fill('Original Buyer');
 await page.getByLabel('The six-digit code we sent you').fill('123456');
 await page.getByRole('button',{name:'Send a new code'}).click();
 await expect.poll(()=>sends).toBe(2);
 await expect(page.getByLabel('The six-digit code we sent you')).toHaveValue('');
 await expect(page.getByLabel('Full name')).toHaveValue('Original Buyer');
 await page.evaluate(()=>{const a=document.createElement('a');a.href='/buyer-invitations/second';a.textContent='Open second invitation';document.body.append(a);});
 await page.getByRole('link',{name:'Open second invitation'}).click();
 await expect(page.getByRole('heading',{name:/second seller wants to add you/})).toBeVisible();
 await expect(page.getByRole('button',{name:'Send me my code'})).toBeVisible();
 await expect(page.getByLabel('Full name')).toHaveCount(0);
 await page.getByRole('button',{name:'Send me my code'}).click();
 await expect(page.getByLabel('Full name')).toHaveValue('');
});
