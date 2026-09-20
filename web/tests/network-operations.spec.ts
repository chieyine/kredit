import {test,expect,type Page} from '@playwright/test';
const org='00000000-0000-4000-8000-000000000001',branch='00000000-0000-4000-8000-000000000002',customer='00000000-0000-4000-8000-000000000003',manager='00000000-0000-4000-8000-000000000004';
async function setup(page:Page,manage=true){
 const state={can_manage:manage,branches:[{id:branch,name:'Mainland',territory:'Lagos',active:true,version:1}],partners:[{business_id:customer,name:'City Distributors',branch_id:'',manager_id:'',manager_active:false,version:0}],managers:[{id:manager,name:'Ada'}]};
 await page.route('**/api/v1/me',r=>r.fulfill({json:{user:{id:'owner'}}}));
 await page.route('**/api/v1/organizations',r=>r.fulfill({json:{organizations:[{id:org,legal_name:'Factory'}]}}));
 await page.route('**/network-operations',r=>r.fulfill({json:state}));return state;
}
test.beforeEach(async({context,baseURL})=>{await context.addCookies([{name:'kredit_session',value:'network-fixture',url:baseURL!}]);});
test('an administrator assigns only the selected business customer',async({page})=>{
 const state=await setup(page);
 await page.route(`**/organizations/${org}/customers/${customer}/assignment`,r=>{expect(r.request().method()).toBe('PUT');expect(r.request().postDataJSON()).toEqual({branch_id:branch,manager_id:manager,version:0});state.partners[0]={...state.partners[0],branch_id:branch,manager_id:manager,manager_active:true,version:1};return r.fulfill({json:state.partners[0]});});
 await page.goto(`/workspace/partners/operations?organization=${org}`);
 await page.getByRole('combobox',{name:'Branch for City Distributors',exact:true}).selectOption(branch);await page.getByRole('combobox',{name:'Account manager for City Distributors',exact:true}).selectOption(manager);
 await page.getByRole('button',{name:'Save assignment for City Distributors'}).click();await expect(page.getByText('Partner assignment saved.')).toBeVisible();await expect(page.getByRole('combobox',{name:'Branch for City Distributors',exact:true})).toHaveValue(branch);
});
test('read-only members cannot change branches or assignments',async({page})=>{
 await setup(page,false);await page.goto(`/workspace/partners/operations?organization=${org}`);
 await expect(page.getByRole('heading',{name:'Customer assignments'})).toBeVisible();await expect(page.getByRole('button',{name:'Add branch'})).toHaveCount(0);await expect(page.getByRole('button',{name:'Save assignment for City Distributors'})).toHaveCount(0);await expect(page.getByRole('combobox',{name:'Branch for City Distributors',exact:true})).toBeDisabled();
});
test('removed managers and closed branches are clearly flagged',async({page})=>{
 const state=await setup(page);state.branches[0].active=false;state.partners[0].branch_id=branch;state.partners[0].manager_id='removed';
 await page.goto(`/workspace/partners/operations?organization=${org}`);
 await expect(page.getByText('The assigned manager is no longer available. Choose a current team member.')).toBeVisible();await expect(page.getByText('This branch is closed to new assignments. Choose an open branch before saving changes.')).toBeVisible();
});
test('new branches use a stable UUID and exact versioned body',async({page})=>{
 const state=await setup(page);let savedID='';
 await page.route('**/branches/*',r=>{const body=r.request().postDataJSON();expect(body).toEqual({name:'Island',territory:'Lagos Island',active:true,version:0});savedID=r.request().url().split('/').at(-1)!;expect(savedID).toMatch(/^[0-9a-f-]{36}$/);state.branches.push({id:savedID,name:body.name,territory:body.territory,active:true,version:1});return r.fulfill({json:state.branches.at(-1)});});
 await page.goto(`/workspace/partners/operations?organization=${org}`);await page.getByLabel('Branch name',{exact:true}).fill('Island');await page.getByLabel('Territory',{exact:true}).fill('Lagos Island');await page.getByRole('button',{name:'Add branch'}).click();await expect(page.getByText('Branch saved.')).toBeVisible();await expect(page.getByRole('heading',{name:'Island',exact:true})).toBeVisible();
});
