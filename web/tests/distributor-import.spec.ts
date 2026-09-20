import { expect, test } from '@playwright/test';

const header='source_reference,target,target_type,legal_name,trading_name,business_type,business_address,industry\n';
const row='D001,buyer@example.test,email,Example Distributors,,limited_company,"Market Road, Lagos",food\n';
test.beforeEach(async({page,context,baseURL})=>{
 await context.addCookies([{name:'kredit_session',value:'import-test',url:baseURL!}]);
 let batch:any;
 await page.route('**/api/v1/organizations/manufacturer/distributor-imports',r=>{if(r.request().method()==='GET')return r.fulfill({json:{batches:batch?[batch]:[]}});const input=r.request().postDataJSON();expect(input.source_hash).toMatch(/^[a-f0-9]{64}$/);batch={id:'batch-1',state:'draft',contacts:input.contacts,row_count:input.contacts.length,completed_rows:[],source_hash:input.source_hash,created_at:'2026-09-20T00:00:00Z'};return r.fulfill({status:201,json:batch});});
 await page.route('**/api/v1/organizations/manufacturer/distributor-imports/batch-1',r=>{if(r.request().method()==='POST')batch.state=r.request().postDataJSON().action==='approve'?'approved':'cancelled';return r.fulfill({json:batch});});
 await page.route('**/api/v1/me',r=>r.fulfill({json:{user:{id:'owner'}}}));
 await page.route('**/api/v1/organizations',r=>r.fulfill({json:{organizations:[{id:'manufacturer',legal_name:'Manufacturer',trading_name:''}]}}));
});
test('roster preview requires approval and preserves stable import identity',async({page})=>{
 let submissions=0;
 await page.route('**/api/v1/organizations/manufacturer/distributor-imports/batch-1/rows/*',async r=>{submissions++;expect(r.request().postDataJSON()).toEqual({});await r.fulfill({status:202,json:{invitation_url:'https://kredit.test/buyer-invitations/private',delivery_state:'manual_handoff_required'}})});
 await page.goto('/workspace/partners/import');
 await page.getByLabel('Business contacts CSV').setInputFiles({name:'distributors.csv',mimeType:'text/csv',buffer:Buffer.from(header+row)});
 await expect(page.getByRole('heading',{name:'Review 1 contact'})).toBeVisible();
 const send=page.getByRole('button',{name:'Create and send invitations'});
 await expect(send).toBeDisabled();
 expect(submissions).toBe(0);
 await page.getByRole('button',{name:'Save this roster'}).click();
 await page.getByRole('checkbox').check();await send.click();
 await expect(page.getByText('Share link manually',{exact:true})).toBeVisible();
 await expect(send).toBeDisabled();expect(submissions).toBe(1);
});
test('duplicate contacts fail preview before any invitation is sent',async({page})=>{
 await page.goto('/workspace/partners/import');
 await page.getByLabel('Business contacts CSV').setInputFiles({name:'duplicates.csv',mimeType:'text/csv',buffer:Buffer.from(header+row+row.replace('D001','D002'))});
 await expect(page.getByRole('alert')).toContainText('duplicate contact');
 await expect(page.getByRole('button',{name:'Create and send invitations'})).toHaveCount(0);
});
test('a failed invitation stops the batch before the next contact',async({page})=>{
 let submissions=0;
 await page.route('**/api/v1/organizations/manufacturer/distributor-imports/batch-1/rows/*',r=>{submissions++;return r.fulfill({status:503,json:{detail:'unknown'}})});
 await page.goto('/workspace/partners/import');
 await page.getByLabel('Business contacts CSV').setInputFiles({name:'two.csv',mimeType:'text/csv',buffer:Buffer.from(header+row+row.replace('D001','D002').replace('buyer@','second@'))});
 await page.getByRole('button',{name:'Save this roster'}).click();
 await page.getByRole('checkbox').check();await page.getByRole('button',{name:'Create and send invitations'}).click();
 await expect(page.getByRole('alert')).toContainText('not confirmed');expect(submissions).toBe(1);
});

test('saved roster can be reopened and cancellation survives reload',async({page})=>{
 await page.goto('/workspace/partners/import');
 await page.getByLabel('Business contacts CSV').setInputFiles({name:'saved.csv',mimeType:'text/csv',buffer:Buffer.from(header+row)});
 await page.getByRole('button',{name:'Save this roster'}).click();
 await expect(page.getByRole('button',{name:'Open 1-contact import'})).toBeVisible();
 await page.reload();await page.getByRole('button',{name:'Open 1-contact import'}).click();
 await expect(page.getByRole('heading',{name:'Review 1 contact'})).toBeVisible();
 await page.getByRole('button',{name:'Cancel remaining invitations'}).click();
 await expect(page.getByText('Batch status: cancelled')).toBeVisible();
 await page.reload();await page.getByRole('button',{name:'Open 1-contact import'}).click();
 await expect(page.getByRole('button',{name:'Create and send invitations'})).toBeDisabled();
 await expect(page.getByRole('checkbox')).toBeDisabled();
});

test('resuming a saved batch sends only its remaining contact',async({page})=>{
 const contacts=[{source_reference:'roster:D001',target:'buyer@example.test',target_type:'email',legal_name:'First distributor',trading_name:'',business_type:'limited_company',business_address:'Lagos',industry:'food'},{source_reference:'roster:D002',target:'second@example.test',target_type:'email',legal_name:'Second distributor',trading_name:'',business_type:'limited_company',business_address:'Ibadan',industry:'food'}];
 const saved={id:'resume-batch',state:'approved',contacts,row_count:2,completed_rows:[1],source_hash:'a'.repeat(64),created_at:'2026-09-20T00:00:00Z'};
 await page.route('**/api/v1/organizations/manufacturer/distributor-imports',r=>r.fulfill({json:{batches:[saved]}}));
 await page.route('**/api/v1/organizations/manufacturer/distributor-imports/resume-batch',r=>r.fulfill({json:saved}));
 const sent:string[]=[];
 await page.route('**/api/v1/organizations/manufacturer/distributor-imports/resume-batch/rows/*',r=>{sent.push(r.request().url().split('/').at(-1)!);return r.fulfill({status:202,json:{delivery_state:'manual_handoff_required',invitation_url:'https://kredit.test/buyer-invitations/second'}})});
 await page.goto('/workspace/partners/import');await page.getByRole('button',{name:'Open 2-contact import'}).click();
 await expect(page.getByText('Existing invitation',{exact:true})).toBeVisible();
 await page.getByRole('button',{name:'Create and send invitations'}).click();
 await expect(page.getByText('Share link manually',{exact:true})).toBeVisible();expect(sent).toEqual(['2']);
});
