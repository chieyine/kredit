import {test,expect,type Page} from '@playwright/test';
const org='00000000-0000-4000-8000-000000000071',profile='00000000-0000-4000-8000-000000000072',staff='00000000-0000-4000-8000-000000000073';
async function setup(page:Page,owner=true){
 const state={can_manage:owner,grants:[{active:true,user_id:staff,name:'Amina',actions:['read'],ceiling_kobo:0,expires_at:'2027-01-01T00:00:00Z',version:1}]};
 await page.route('**/api/v1/me',r=>r.fulfill({json:{user:{id:'owner'}}}));
 await page.route('**/api/v1/organizations',r=>r.fulfill({json:{organizations:[{id:org,legal_name:'City Distributors'}]}}));
 await page.route('**/api/v1/buyer/businesses',r=>r.fulfill({json:{businesses:[{id:profile,workspace_id:org,legal_name:'City Distributors'}]}}));
 await page.route('**/purchasing-authority',r=>r.fulfill({json:state}));return state;
}
test.beforeEach(async({context,baseURL})=>{await context.addCookies([{name:'kredit_session',value:'staff-fixture',url:baseURL!}]);});
test('owner grants an exact kobo ceiling with read access and expiry',async({page})=>{
 const state=await setup(page);let saved=false;
 await page.route(`**/purchasing-authority/${staff}`,r=>{const b=r.request().postDataJSON();expect(r.request().method()).toBe('PUT');expect(b.ceiling_kobo).toBe(12500050);expect(b.version).toBe(1);expect(b.actions).toEqual(['read','accept']);expect(b.expires_at).toBe('2027-01-01T22:59:59.000Z');state.grants[0]={...state.grants[0],...b,active:b.actions.length>0,version:2};saved=true;return r.fulfill({json:state.grants[0]})});
 await page.goto(`/workspace/purchases/permissions?organization=${org}`);await page.getByLabel('Accept verified purchase terms').check();await page.getByLabel('Maximum per accepted purchase (₦)').fill('125000.50');await page.getByRole('button',{name:'Save permissions for Amina'}).click();await expect(page.getByText('Purchasing permissions saved.')).toBeVisible();expect(saved).toBe(true);await page.evaluate(()=>window.scrollTo(0,0));await page.screenshot({path:'../../identity-completion/staff-permissions-desktop.png',fullPage:true});await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBeTruthy();await page.screenshot({path:'../../identity-completion/staff-permissions-mobile.png',fullPage:true});
});
test('staff can inspect but cannot assign their own purchasing authority',async({page})=>{
 await setup(page,false);await page.goto(`/workspace/purchases/permissions?organization=${org}`);await expect(page.getByRole('heading',{name:'Amina'})).toBeVisible();await expect(page.getByLabel('Accept verified purchase terms')).toBeDisabled();await expect(page.getByRole('button',{name:'Save permissions for Amina'})).toHaveCount(0);
});
test('removal sends no purchasing actions and keeps the exact saved version',async({page})=>{
 const state=await setup(page);await page.route(`**/purchasing-authority/${staff}`,r=>{const b=r.request().postDataJSON();expect(b.actions).toEqual([]);expect(b.version).toBe(1);state.grants[0]={...state.grants[0],...b,active:b.actions.length>0,version:2};return r.fulfill({json:state.grants[0]})});await page.goto(`/workspace/purchases/permissions?organization=${org}`);await page.getByLabel('Maximum per accepted purchase (₦)').fill('invalid unsaved amount');await page.getByLabel('Access expiry').fill('');await page.getByRole('button',{name:'Remove delegated access'}).click();await expect(page.getByText('Purchasing access removed.')).toBeVisible();await expect(page.getByText('No current delegated purchasing access')).toBeVisible();
});
test('staff enrollment binds current notices to the chosen buying business',async({page})=>{
 await setup(page,false);let posted=false;
 await page.route('**/api/v1/identity/checks',r=>r.fulfill({json:{cases:[],consent_version:'identity-current'}}));
 await page.route('**/api/v1/buyer/me/purchasing-access',r=>{if(r.request().method()==='POST'){expect(r.request().postDataJSON()).toEqual({business_id:profile,full_name:'Amina Bello',consents_accepted:true,terms_version:'terms-current',privacy_version:'privacy-current',identity_notice_version:'identity-current'});posted=true;return r.fulfill({json:{portal:{}}})}return r.fulfill({json:{businesses:[{id:profile,workspace_id:org,legal_name:'City Distributors'}],legal_versions:{terms_version:'terms-current',privacy_version:'privacy-current'},identity_notice:'Your own verification is required.',identity_notice_version:'identity-current'}})});
 await page.goto(`/workspace/purchases/access?organization=${org}`);await page.getByLabel('Your full name').fill('Amina Bello');await page.getByRole('checkbox').check();await page.getByRole('button',{name:'Set up my purchasing profile'}).click();await expect(page.getByText('Your profile is ready. Start or refresh verification below.')).toBeVisible();expect(posted).toBe(true);
});
