import { test, expect, type Page, type BrowserContext, type Route } from '@playwright/test';
const send=(route:Route,body:unknown,status=200)=>route.fulfill({status,contentType:'application/json',body:JSON.stringify(body)});
async function prepare(page:Page,context:BrowserContext,baseURL?:string){
 await context.addCookies([{name:'kredit_session',value:'synthetic-audit',url:baseURL??'http://127.0.0.1:5173'},{name:'kredit_csrf',value:'synthetic-csrf',url:baseURL??'http://127.0.0.1:5173'}]);
 await page.route('**/api/v1/**',route=>{
  const path=new URL(route.request().url()).pathname;
  if(path==='/api/v1/me')return send(route,{user:{id:'user-1'},session:{authentication_level:'aal2'},organizations:[{id:'org-a',legal_name:'Kora Wholesale'}]});
  if(path==='/api/v1/organizations')return send(route,{organizations:[{id:'org-a',legal_name:'Kora Wholesale',trading_name:''}]});
  if(path.endsWith('/customers'))return send(route,{customers:[{buyer_user_id:'buyer-1',buyer_business_id:'business-1',legal_name:'Amina Stores',trading_name:'Amina Stores',state:'verified'}]});
  if(path.endsWith('/credit-terms/preview')){const input=route.request().postDataJSON();return send(route,{due_date:input.due_date,grace_hours:input.grace_hours,collection_at:'2026-09-19T22:59:00Z',timezone:'Africa/Lagos',cutoff:'23:59',timing_mode:input.collection_local?'lagos_explicit':'lagos_end_of_day'});}
  return send(route,{},404);
 });
 await page.goto('/app/credit/new?advanced=1&organization=org-a');
 await page.getByRole('combobox',{name:'Customer',exact:true}).selectOption('buyer-1:business-1');
 await page.getByRole('textbox',{name:'Sale amount (₦)'}).fill('127,500.49');
 await page.getByRole('textbox',{name:'What goods are they taking?'}).fill('40 cartons of cooking oil');
 await page.getByLabel('First payment date').fill('2026-09-18');
}
test('invoice upload retries preserve one operation before sale creation',async({page,context,baseURL})=>{
 await prepare(page,context,baseURL);
 // Use a real, repository-owned PNG as synthetic upload bytes. No bank or personal data.
 const hash=await page.evaluate(async()=>{
  const response=await fetch('/icon-192.png');if(!response.ok)throw new Error('Synthetic fixture unavailable');
  return Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256',await response.arrayBuffer())),byte=>byte.toString(16).padStart(2,'0')).join('');
 });
 const keys:string[]=[];const creations:Record<string,unknown>[]=[];
 await page.route('**/api/v1/organizations/org-a/documents',route=>{
  keys.push(route.request().headers()['idempotency-key']);
  if(keys.length===1)return route.abort('failed');
  return send(route,{document:{sha256:hash}},201);
 });
 await page.route('**/api/v1/organizations/org-a/credit-requests',route=>{creations.push(route.request().postDataJSON());return send(route,{request:{id:'created'}},201);});
 await page.locator('input[type=file]').setInputFiles('static/icon-192.png');
 await page.getByRole('button',{name:'Check terms',exact:true}).click();
 await page.getByRole('button',{name:'Save draft sale',exact:true}).click();
 await expect(page.getByRole('alert')).toContainText('not confirmed');
 expect(creations).toHaveLength(0);
 await page.getByRole('button',{name:'Save draft sale',exact:true}).click();
 await expect.poll(()=>creations.length).toBe(1);
 expect(keys).toHaveLength(2);expect(keys[0]).toBeTruthy();expect(keys[1]).toBe(keys[0]);
 expect(creations[0]).toMatchObject({invoice_document_hash:hash,principal_kobo:12750049});
});
test('custom instalments must reconcile exactly before any write',async({page,context,baseURL})=>{
 await prepare(page,context,baseURL);let writes=0;
 await page.route('**/api/v1/organizations/org-a/credit-requests',route=>{writes++;return send(route,{},500);});
 await page.getByLabel('How will they pay?').selectOption('custom');
 await page.getByRole('textbox',{name:'Write out each payment'}).fill('60,000.00, 2026-09-18\n60,000.00, 2026-10-18');
 await page.getByRole('button',{name:'Check terms',exact:true}).click();
 await expect(page.getByRole('alert')).toContainText('must add up');
 expect(writes).toBe(0);
});
