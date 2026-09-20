import { expect, test } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const organizations = [{id:'org-a',legal_name:'Factory Limited'}, {id:'org-b',legal_name:'Distributor Limited'}];
const businesses = organizations.map((org,i)=>({id:`profile-${i}`,workspace_id:org.id,legal_name:org.legal_name}));
async function account(page:any,context:any,baseURL:string){
 await context.addCookies([{name:'kredit_session',value:'unified-test',url:baseURL}]);
 await page.route('**/api/v1/**',(route:any)=>{
  const path=new URL(route.request().url()).pathname;
  let json:any={};
  if(path==='/api/v1/me')json={user:{id:'owner'},organizations};
  else if(path==='/api/v1/organizations')json={organizations};
  else if(path==='/api/v1/buyer/businesses')json={businesses};
  else if(path==='/api/v1/buyer/purchases')json={items:[]};
  else if(path.endsWith('/credit-requests'))json={requests:[]};
  else if(path.endsWith('/reports/receivables'))json={summary:{obligation_count:0,outstanding_kobo:0,overdue_kobo:0}};
  else if(path.endsWith('/onboarding'))json={readiness:{ready:true,requirements:[],missing:[]}};
  else {const key=({'payments':'payments','overdue':'overdue','payment-claims':'payment_claims','disputes':'disputes','due':'due'} as any)[path.split('/').at(-1)!];if(key)json={[key]:[]};}
  return route.fulfill({json});
 });
}

test('public network reads clearly on mobile and the consumer demo completes',async({page})=>{
 test.setTimeout(120000);
 await page.setViewportSize({width:390,height:844});
 for(const path of ['/','/manufacturers','/distributors','/retailers','/consumers','/how-it-works']){
  await page.goto(path);await expect(page.locator('h1')).toBeVisible();
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),path).toBeTruthy();
  if(path==='/'){
   await page.screenshot({path:'../.tmp/redesign-flow/home-mobile.png',fullPage:true});
   await page.setViewportSize({width:1440,height:1000});
   await page.screenshot({path:'../.tmp/redesign-flow/home-desktop.png',fullPage:true});
   await page.setViewportSize({width:390,height:844});
  }
 }
 await page.getByRole('link',{name:'Consumers',exact:true}).first().click();
 await expect(page).toHaveURL(/\/consumers$/);
 await page.locator('main').getByRole('link',{name:'Explore the demo',exact:true}).click();
 await expect(page).toHaveURL(/\/demo\/consumer$/);
 for(const name of ['Accept sample terms','Complete sample payments','Confirm sample delivery'])await page.getByRole('button',{name:new RegExp(name)}).click();
 await expect(page.getByRole('status')).toContainText('Sample complete');
 const audit=await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa']).analyze();
 expect(audit.violations.filter(v=>['serious','critical'].includes(v.impact||''))).toEqual([]);
 await page.evaluate(()=>{(document.activeElement as HTMLElement)?.blur();window.scrollTo(0,0)});
 await page.screenshot({path:'../.tmp/redesign-flow/consumer-mobile.png',fullPage:true});
});
test('one account opens personal purchases without business registration',async({page,context,baseURL})=>{
 await account(page,context,baseURL!);await page.goto('/start');
 await expect(page.getByRole('heading',{name:'Where would you like to go?'})).toBeVisible();
 await page.getByRole('link',{name:/Individual consumers/}).click();
 await expect(page.getByRole('heading',{name:'Your purchases',exact:true})).toBeVisible();
 await expect(page.getByText('No matching purchases yet.')).toBeVisible();
 await page.getByRole('link',{name:'Account settings',exact:true}).first().click();
 await expect(page).toHaveURL(/\/account$/);
 await expect(page.getByRole('heading',{name:'Account settings',exact:true})).toBeVisible();
});
test('workspace navigation keeps the selected business across both sides of trade',async({page,context,baseURL})=>{
 await account(page,context,baseURL!);await page.goto('/workspace/today?organization=org-b');
 await expect(page.getByRole('heading',{name:'Distributor Limited',exact:true})).toBeVisible();
 await page.getByRole('link',{name:'Purchases',exact:true}).first().click();
 await expect(page.getByRole('combobox',{name:'Purchasing business'})).toHaveValue('profile-1');
 await expect(page).toHaveURL(/business_id=profile-1/);
 await page.getByRole('link',{name:'Sales',exact:true}).first().click();
 await expect(page).toHaveURL(/\/workspace\/sales\?organization=org-b$/);
 await expect(page.getByRole('combobox',{name:'Business',exact:true})).toHaveValue('org-b');
 await page.goto('/workspace/partners?organization=org-b');
 await expect(page.getByRole('heading',{name:'Partners',exact:true})).toBeVisible();
 await page.screenshot({path:'../.tmp/redesign-flow/partners-desktop.png',fullPage:true});
 await page.setViewportSize({width:390,height:844});
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBeTruthy();
 await page.screenshot({path:'../.tmp/redesign-flow/partners-mobile.png',fullPage:true});
});
test('a foreign purchasing business fails closed instead of showing another business',async({page,context,baseURL})=>{
 await account(page,context,baseURL!);await page.goto('/workspace/purchases?organization=org-a&business_id=profile-1');
 await expect(page.getByRole('alert')).toContainText('not available');
 await expect(page.getByRole('heading',{name:'Supplier purchases'})).toHaveCount(0);
});
test('signed-out private entry preserves the intended destination',async({page})=>{
 for(const path of ['/start','/account','/workspace/today','/personal/purchases']){
  await page.goto(path);await expect(page).toHaveURL(new RegExp('/signin\\?next='+encodeURIComponent(path)));
 }
});

test('a direct purchase link resolves the actual business and rejects a conflicting context',async({page,context,baseURL})=>{
 await account(page,context,baseURL!);
 await page.route('**/api/v1/buyer/obligations/debt-2',route=>route.fulfill({json:{
  view:{request:{id:'sale-2',buyer_business_id:'profile-1',goods_description:'Goods for Distributor Limited'},obligation:{outstanding_kobo:10000,payment_status:'UNPAID'}},
  schedule_items:[],collection_notices:[],payments:[],payment_claims:[]
 }}));
 await page.goto('/workspace/purchases/obligations/debt-2');
 await expect(page.getByRole('combobox',{name:'Purchasing business'})).toHaveValue('profile-1');
 await expect(page.getByRole('heading',{name:'Goods for Distributor Limited'})).toBeVisible();
 await expect(page).toHaveURL(/business_id=profile-1&organization=org-b/);
 await page.goto('/workspace/purchases/obligations/debt-2?business_id=profile-0');
 await expect(page.getByRole('alert')).toBeVisible();
 await expect(page.getByRole('heading',{name:'Goods for Distributor Limited'})).toHaveCount(0);
});
