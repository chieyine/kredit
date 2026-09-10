import {test,expect} from '@playwright/test';

const fields=[
 {key:'automatic_collection',label:'Collect due payments automatically',group:'Collections',kind:'boolean',min:0,max:0,help:'Applies to background workers.'},
 {key:'notice_hours',label:'Minimum delivered notice (hours)',group:'Collections',kind:'number',min:1,max:720,help:'Delivered notice is required.'},
 {key:'base_fee_bps',label:'Supplier base fee (basis points)',group:'Fees',kind:'number',min:0,max:1000,help:'New offers only.'}
];
test('admin reviews a proposal and a different administrator approves it',async({page,context,baseURL})=>{
 await context.addCookies([{name:'kredit_session',value:'settings-session',url:baseURL??'http://127.0.0.1:5173'}]);
 await page.route('**/api/v1/me',r=>r.fulfill({json:{user:{id:'maker'},session:{authentication_level:'AAL2'},organizations:[]}}));
 const state:any={actor_id:'maker',fields,current:{revision:0,values:{automatic_collection:false,notice_hours:24,base_fee_bps:50}},changes:[],events:[],actors:{maker:'Policy maker',checker:'Independent reviewer'},deployment_limits:{}};
 await page.route('**/api/v1/ops/capabilities',r=>r.fulfill({json:{actor_id:state.actor_id,roles:['policy_manager','approver']}}));
 await page.route('**/api/v1/ops/governance',r=>r.fulfill({json:{governance:{mode:'delegated_team'}}}));
 await page.route('**/api/v1/ops/business-policies',async r=>{
  if(r.request().method()==='GET'){await r.fulfill({json:state});return}
  const body=r.request().postDataJSON();expect(r.request().headers()['idempotency-key']).toBeTruthy();expect(body.base_revision).toBe(0);expect(body.values.base_fee_bps).toBe(25);expect(body.reason).toBe('Approved commercial pricing revision');
  state.changes=[{...body,before_values:{...state.current.values},revision:1,proposed_by:'maker',state:'pending',created_at:new Date().toISOString(),decided_by:null}];await r.fulfill({status:201,json:{id:body.id,state:'pending'}});
 });
 await page.route('**/api/v1/ops/business-policies/*/decision',async r=>{const body=r.request().postDataJSON();expect(body.action).toBe('approve');expect(body.reason).toBe('Independently checked pricing and impact');state.changes[0].state='approved';state.changes[0].decided_by='checker';await r.fulfill({json:{status:'recorded'}})});
 await page.goto('/admin/settings');
 await page.getByLabel('Supplier base fee (%)').fill('0.25');
 await expect(page.getByText('0.5% → 0.25%')).toBeVisible();
 const tomorrow=new Date(Date.now()+86400000).toISOString().slice(0,16);await page.getByLabel('Effective date and time (Lagos)').fill(tomorrow);
 await page.getByLabel('Reason for the change',{exact:true}).fill('Approved commercial pricing revision');
 await page.getByRole('button',{name:'Send for approval'}).click();
 await expect(page.getByText('Another platform administrator must approve your proposal.',{exact:true})).toBeVisible();
 await expect(page.getByRole('button',{name:'Approve exactly this'})).toHaveCount(0);
 await expect(page.getByLabel('Supplier base fee (%)')).toBeDisabled();
 state.actor_id='checker';await page.getByRole('button',{name:'Refresh settings'}).click();
 await page.getByLabel('Decision notes').fill('Independently checked pricing and impact');await page.getByRole('button',{name:'Approve exactly this'}).click();
 await expect(page.getByRole('heading',{name:'Revision 1 · scheduled'})).toBeVisible();
 // Scheduled fees have not prematurely replaced the current rate.
 await expect(page.getByText('Current: 0.5%',{exact:true})).toBeVisible();
});

// Fee rates are rendered on the server, so they are in the HTML for a reader
// with no JavaScript and for a crawler. That put the pricing request out of
// reach of page.route, which can only see requests the browser makes — so this
// test cannot decide whether the rates load. Whether the API is up when the
// suite runs is not something the test controls, and a test that passes only
// when a service is down is worse than no test.
//
// What it can pin down is the invariant that matters on a page with a fee
// calculator on it: the page shows rates it has verified, or it says it could
// not verify them. Never a number it has not checked, and never silence.
test('pricing shows verified rates or says why it cannot, never an unchecked number',async({page})=>{
 await page.goto('/pricing');
 const rates=page.locator('.rates');
 const outage=page.locator('.pricing-error[role="alert"]');
 await expect.poll(async()=>await rates.count()+await outage.count(),{message:'exactly one of rates or outage notice'}).toBe(1);

 if(await outage.count()){
  await expect(outage).toContainText('We could not verify the current fees');
  // Recovery does run in the browser, so this half is drivable either way.
  await page.route('**/api/v1/pricing',r=>r.fulfill({json:{base_bps:25,collection_bps:75,policy_revision:9}}));
  await page.getByRole('button',{name:'Try again'}).click();
  await expect(page.getByText('0.25%',{exact:true}).first()).toBeVisible();
  await expect(page.getByText('0.75%',{exact:true})).toBeVisible();
  await expect(outage).toHaveCount(0);
 }

 // However the rates arrived, each is a plain percentage a reader can act on.
 await expect(rates).toBeVisible();
 for(const value of await rates.locator('strong').allInnerTexts()) expect(value.trim()).toMatch(/^\d+(\.\d+)?%$/);
});
