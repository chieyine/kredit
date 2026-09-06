"""Align existing behavioural regressions with the reviewed interface without dropping scenarios."""
from pathlib import Path
import hashlib
root=Path.cwd()
expected={
'web/tests/product-flows.spec.ts':'194be53cc2f554c84b9ef52dab0eba25a137b4c01e29533b274831cc917e8fe6',
'web/tests/accessibility.spec.ts':'a705a1b7eccbbab4eece18f1c606c42b65041bf65b5f7d502c7269a138d000cc',
'web/tests/product-quality.spec.ts':'5ee8c6d8b758bbfe31cada12c389a16ff7a84563370b6b0be527a04d8944e11f',
'web/tests/financial-completion.spec.ts':'6a1f3d092a4f9e6e8d0f6da5cb5ff9cf856fdac5575c9a7928fe9477a96abae6'}
sources={}
for path,digest in expected.items():
 data=(root/path).read_bytes();assert hashlib.sha256(data).hexdigest()==digest,path;sources[path]=data.decode()
def sub(path,old,new):
 assert old in sources[path],(path,old);sources[path]=sources[path].replace(old,new)
p='web/tests/product-flows.spec.ts'
sub(p,".selectOption('buyer-1');", ".selectOption('buyer-1:business-1');")
sub(p,"getByLabel('Money to pay (₦)').fill('1,200,000')", "getByLabel('Sale amount (₦)').fill('1,200,000')")
sub(p,"getByLabel('What goods did they take?')", "getByLabel('What goods are they taking?')")
sub(p,"getByLabel('First payment day')", "getByLabel('First payment date')")
sub(p,"getByLabel('Day Kredit may debit if unpaid')", "getByLabel('Optional later collection time (Nigerian time)')")
sub(p,"\tawait page.goto('/app/credit/new');", "\tawait page.route('**/api/v1/organizations/org-1/credit-terms/preview', route => route.fulfill({json:{due_date:'2026-09-30',grace_hours:24,collection_at:'2026-10-02T08:00:00Z',timezone:'Africa/Lagos',cutoff:'23:59',timing_mode:'lagos_explicit'}}));\n\tawait page.goto('/app/credit/new');")
sub(p,"await page.getByRole('button', { name: 'Save this sale' }).click();", "await page.getByRole('button', { name: 'Check terms', exact:true }).click();\n\texpect(submitted).toBeUndefined();\n\tawait page.getByRole('button', { name: 'Save draft sale', exact:true }).click();")
sub(p,"supplier_legal_name: 'Adebayo Supplies', principal_kobo:", "supplier_legal_name: 'Adebayo Supplies', buyer_legal_name: 'Kano Retail Limited', principal_kobo:")
sub(p,"agreement: { document_hash: 'agreement-hash' }", "agreement: { id:'agreement-1', document_hash: 'a'.repeat(64) }")
sub(p,"{ name: 'No, I do not agree' }", "{ name: 'Decline sale', exact:true }")
sub(p,"'You said no. This sale will not start.'", "'You declined this sale.'")
sub(p,"await page.getByLabel('Money you paid (₦)').fill", "await page.getByText('Already paid by bank transfer?', {exact:true}).click();\n\tawait page.getByLabel('Amount transferred (₦)').fill")
sub(p,"getByLabel('Transfer number')", "getByLabel('Transfer reference')")
sub(p,"{ name: 'Tell the seller I have paid' }", "{ name: 'Report my transfer', exact:true }")
sub(p,"'We told the seller. They will check their bank account.'", "'Transfer reported. The seller must confirm receipt before your balance changes.'")
p='web/tests/accessibility.spec.ts'
sub(p,".selectOption('buyer-a11y');", ".selectOption('buyer-a11y:business-a11y');")
sub(p,"getByLabel('Money to pay (₦)')", "getByLabel('Sale amount (₦)')")
sub(p,"getByLabel('What goods did they take?')", "getByLabel('What goods are they taking?')")
sub(p,"getByLabel('First payment day')", "getByLabel('First payment date')")
sub(p,"getByLabel('Day Kredit may debit if unpaid')", "getByLabel('Optional later collection time (Nigerian time)')")
sub(p,"{ name: 'Save this sale' }", "{ name: 'Check terms', exact:true }")
sub(p,"'Complete every required field'", "'Check the customer, goods, amount and first payment date.'")
sub(p,"schedule_type: 'one_time' };", "schedule_type: 'one_time', fee_terms:{policy_revision:1,base_bps:50,collection_bps:50} };")
sub(p,"document_hash: 'accessible-agreement-hash'", "document_hash: 'a'.repeat(64)")
p='web/tests/product-quality.spec.ts'
sub(p,"/Trust comes from the record/", "/Both sides can.*check the details/")
sub(p,"'Delivery evidence stays with the sale.'", "'The delivery proof stays with the sale.'")
sub(p,"'Every recorded payment changes the balance.'", "'Confirmed payments update the balance.'")
sub(p,"/full sale journey/i", "/See how a sale works/i")
sub(p,"/security and privacy controls/i", "/See how we keep it safe/i")
start=sources[p].index("test('default sale creation redirects")
end=sources[p].index("test('every indexable page",start)
sources[p]=sources[p][:start]+'''test('both sale-creation entry points preserve authentication and the intended destination', async ({ request }) => {
 for (const path of ['/app/credit/quick?customer=u1&goods=Rice&amount=100000','/app/credit/new?advanced=1']) {
  const response=await request.get(path,{maxRedirects:0});
  expect(response.status()).toBe(303);
  const location=new URL(response.headers().location,'http://127.0.0.1:5173');
  expect(location.pathname).toBe('/app');
  expect(location.searchParams.get('next')).toBe(path);
  expect(response.headers()['cache-control']).toContain('no-store');
 }
});

'''+sources[p][end:]
p='web/tests/financial-completion.spec.ts'
sub(p,"'We could not open your payment records.'", "'We could not open your payments.'")
for path,content in sources.items():(root/path).write_text(content)
(root/'.tmp').mkdir(exist_ok=True)
(root/'.tmp/regression-paths.txt').write_text(''.join(path+'\n' for path in sources))
print('Updated four existing test suites; retained failure, authorization, money and focus assertions.')
