import {test,expect} from '@playwright/test';

const fields=[
 {key:'automatic_collection',label:'Collect due payments automatically',group:'Collections',kind:'boolean',min:0,max:0,help:'Applies to background workers.'},
 {key:'notice_hours',label:'Minimum delivered notice (hours)',group:'Collections',kind:'number',min:1,max:720,help:'Delivered notice is required.'},
 {key:'base_fee_bps',label:'Supplier base fee (basis points)',group:'Fees',kind:'number',min:0,max:1000,help:'New offers only.'}
];
test('admin reviews a proposal and a different administrator approves it',async({page,context,baseURL})=>{
 await context.addCookies([{name:'kredit_session',value:'settings-session',url:baseURL??'http://127.0.0.1:5173'}]);
 await page.route('**/api/v1/me',r=>r.fulfill({json:{user:{id:'maker'},session:{authentication_level:'AAL2'},organizations:[]}}));
 const state:any={actor_id:'maker',fields,current:{revision:0,values:{automatic_collection:false,notice_hours:24,base_fee_bps:50}},changes:[],events:[],deployment_limits:{}};
 await page.route('**/api/v1/ops/business-policies',async r=>{
  if(r.request().method()==='GET'){await r.fulfill({json:state});return}
  const body=r.request().postDataJSON();expect(r.request().headers()['idempotency-key']).toBeTruthy();expect(body.base_revision).toBe(0);expect(body.values.base_fee_bps).toBe(25);expect(body.reason).toBe('Approved commercial pricing revision');
  state.changes=[{...body,revision:1,proposed_by:'maker',state:'pending',created_at:new Date().toISOString(),decided_by:null}];await r.fulfill({status:201,json:{id:body.id,state:'pending'}});
 });
 await page.route('**/api/v1/ops/business-policies/*/decision',async r=>{const body=r.request().postDataJSON();expect(body.action).toBe('approve');expect(body.reason).toBe('Independently checked pricing and impact');state.changes[0].state='approved';state.changes[0].decided_by='checker';await r.fulfill({json:{status:'recorded'}})});
 await page.goto('/admin/settings');
 await page.getByLabel('Supplier base fee (%)').fill('0.25');
 await expect(page.getByText('0.5% → 0.25%')).toBeVisible();
 const tomorrow=new Date(Date.now()+86400000).toISOString().slice(0,16);await page.getByLabel('Effective date and time (Lagos)').fill(tomorrow);
 await page.getByLabel('Reason for the change',{exact:true}).fill('Approved commercial pricing revision');
 await page.getByRole('button',{name:'Submit for independent approval'}).click();
 await expect(page.getByText('Another platform administrator must approve your proposal.',{exact:true})).toBeVisible();
 await expect(page.getByRole('button',{name:'Approve exact proposal'})).toHaveCount(0);
 await expect(page.getByLabel('Supplier base fee (%)')).toBeDisabled();
 state.actor_id='checker';await page.getByRole('button',{name:'Refresh settings'}).click();
 await page.getByLabel('Decision notes').fill('Independently checked pricing and impact');await page.getByRole('button',{name:'Approve exact proposal'}).click();
 await expect(page.getByRole('heading',{name:'Revision 1 · scheduled'})).toBeVisible();
 // Scheduled fees have not prematurely replaced the current rate.
 await expect(page.getByText('Current: 0.5%',{exact:true})).toBeVisible();
});

test('pricing follows published policy and fails visibly when pricing is unavailable',async({page})=>{
 await page.route('**/api/v1/pricing',r=>r.fulfill({json:{base_bps:25,collection_bps:75,policy_revision:9}}));
 await page.goto('/pricing');await expect(page.getByText('0.25%',{exact:true}).first()).toBeVisible();await expect(page.getByText('0.75%',{exact:true})).toBeVisible();
 await page.route('**/api/v1/pricing',r=>r.fulfill({status:503,json:{detail:'Pricing unavailable'}}));await page.reload();await expect(page.getByRole('alert')).toContainText('Current rates could not be loaded');await expect(page.getByText('0.25%',{exact:true})).toHaveCount(0);
});
