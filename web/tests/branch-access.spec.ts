import {test,expect,type Page} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
const org='00000000-0000-4000-8000-000000000001',staff='00000000-0000-4000-8000-000000000002',branch='00000000-0000-4000-8000-000000000003';
async function setup(page:Page,owner=true){
 const state={can_manage:owner,scopes:[{user_id:staff,name:'Ada',mode:'all',branch_ids:[] as string[],version:0,active:true}]};
 await page.route('**/api/v1/me',r=>r.fulfill({json:{user:{id:'owner'}}}));
 await page.route('**/api/v1/organizations',r=>r.fulfill({json:{organizations:[{id:org,legal_name:'Factory'}]}}));
 await page.route('**/network-operations',r=>r.fulfill({json:{branches:[{id:branch,name:'Mainland',active:true}]}}));
 await page.route('**/branch-access',r=>r.fulfill({json:state}));return state;
}
test.beforeEach(async({context,baseURL})=>{await context.addCookies([{name:'kredit_session',value:'branch-fixture',url:baseURL!}]);});
test('owner saves an explicit branch boundary and can remove all branch access',async({page},testInfo)=>{
 const state=await setup(page);const bodies:unknown[]=[];
 await page.route('**/branch-access/*',r=>{const body=r.request().postDataJSON();bodies.push(body);expect(r.request().method()).toBe('PUT');Object.assign(state.scopes[0],body,{version:body.version+1});return r.fulfill({json:state.scopes[0]});});
 await page.goto(`/workspace/partners/access?organization=${org}`);
 await page.getByLabel('Access for Ada').selectOption('branches');await page.getByLabel('Mainland',{exact:true}).check();await page.getByRole('button',{name:'Save access for Ada'}).click();await expect(page.getByText('Team access saved.')).toBeVisible();
 expect(bodies[0]).toEqual({mode:'branches',branch_ids:[branch],version:0});
 await page.getByLabel('Mainland',{exact:true}).uncheck();await expect(page.getByText('No branches selected: this member cannot access business customer sales.')).toBeVisible();await page.getByRole('button',{name:'Save access for Ada'}).click();await expect.poll(()=>bodies.length).toBe(2);expect(bodies[1]).toEqual({mode:'branches',branch_ids:[],version:1});
 await page.setViewportSize({width:390,height:844});await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
 await expect(page.getByRole('button',{name:'Save access for Ada'})).toBeEnabled();await page.evaluate(()=>window.scrollTo(0,0));
 expect((await new AxeBuilder({page}).analyze()).violations).toEqual([]);
 await page.screenshot({path:testInfo.outputPath('branch-access-mobile.png'),fullPage:true});
});
test('staff see their scope without owner controls',async({page})=>{
 const state=await setup(page,false);state.scopes[0].mode='branches';state.scopes[0].branch_ids=[branch];
 await page.goto(`/workspace/partners/access?organization=${org}`);await expect(page.getByLabel('Access for Ada')).toBeDisabled();await expect(page.getByRole('button',{name:'Save access for Ada'})).toHaveCount(0);
});
test('changed membership requires a new owner decision',async({page})=>{
 const state=await setup(page);state.scopes[0].active=false;
 await page.goto(`/workspace/partners/access?organization=${org}`);await expect(page.getByText('Membership has changed.',{exact:false})).toBeVisible();
});
