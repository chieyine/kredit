"""One-time reviewed source refinement; fixed paths and expected-source assertions."""
from pathlib import Path
import subprocess
r=Path.cwd()
changed=[]
def save(p,s):
 (r/p).write_text(s);changed.append(p)
def edit(p,old,new):
 s=(r/p).read_text();assert old in s,p+': expected source not found';save(p,s.replace(old,new))
edit('internal/web/business_policy_handlers.go', ', "min_fee_kobo": 0', '')
p='web/tests/audit-product-journeys.spec.ts';s=(r/p).read_text();save(p,s.replace("'**/buyer/", "'**/api/v1/buyer/"))
edit('web/src/lib/components/PortalNav.svelte', "if (event.key === 'Escape' && moreOpen) closeMenus(true);", '''if (!moreOpen || !dialog?.open) return;
        if (event.key === 'Escape') { event.preventDefault(); closeMenus(true); return; }
        if (event.key !== 'Tab') return;
        const controls = Array.from(dialog.querySelectorAll<HTMLElement>('a[href],button:not(:disabled),input:not(:disabled),select:not(:disabled),textarea:not(:disabled),[tabindex]:not([tabindex="-1"])')).filter(node => node.tabIndex >= 0 && node.getClientRects().length > 0);
        const first = controls[0], last = controls.at(-1);
        if (!first || !last) { event.preventDefault(); dialog.focus(); return; }
        if (event.shiftKey && (document.activeElement === first || !dialog.contains(document.activeElement))) { event.preventDefault(); last.focus(); }
        else if (!event.shiftKey && (document.activeElement === last || !dialog.contains(document.activeElement))) { event.preventDefault(); first.focus(); }''')
p='web/src/product-ui.css';save(p,(r/p).read_text()+"\n:root body .product-route :is(main,section,article,fieldset,input,select,textarea,button) { box-sizing:border-box; min-width:0; }\n:root body .product-route :is(dd,code,.amount,.outstanding) { overflow-wrap:anywhere; }\n")
edit('web/src/lib/api/reliable.ts', "key === 'kredit:saved-sale-items' ||", "key === 'kredit:saved-sale-items' || key.startsWith('kredit:sale-draft:') ||")
p='internal/credit/calendar.go';save(p,(r/p).read_text()+'''
// ExplicitCollectionInstant interprets an advanced-form datetime as Nigerian
// business time. No URL or browser timezone can change its meaning.
func ExplicitCollectionInstant(local string) (time.Time, error) {
 location, err := time.LoadLocation("Africa/Lagos")
 if err != nil { return time.Time{}, err }
 date, err := time.ParseInLocation("2006-01-02T15:04", local, location)
 if err != nil || len(local) != 16 || date.Format("2006-01-02T15:04") != local { return time.Time{}, errors.New("a valid Nigerian collection date and time is required") }
 return date.UTC(), nil
}
''')
edit('internal/web/credit_terms_handlers.go','GraceHours int    `json:"grace_hours"`','GraceHours int    `json:"grace_hours"`\n CollectionLocal string `json:"collection_local,omitempty"`')
edit('internal/web/credit_terms_handlers.go','instant, err := credit.CollectionInstant(input.DueDate, input.GraceHours)', '''instant, err := credit.CollectionInstant(input.DueDate, input.GraceHours)
 mode := "lagos_end_of_day"
 if err == nil && input.CollectionLocal != "" {
  earliest := instant
  instant, err = credit.ExplicitCollectionInstant(input.CollectionLocal)
  mode = "lagos_explicit"
  if err == nil && instant.Before(earliest) { writeProblem(w, 422, "credit_terms_invalid", "Bank collection must be after the agreed payment day and extra hours."); return }
 }''')
edit('internal/web/credit_terms_handlers.go','"timing_mode": "lagos_end_of_day"','"timing_mode": mode')
edit('internal/web/credit_handlers.go','TimingMode          string    `json:"timing_mode,omitempty"`','TimingMode          string    `json:"timing_mode,omitempty"`\n CollectionLocal string `json:"collection_local,omitempty"`')
edit('internal/web/credit_handlers.go','if in.TimingMode != "lagos_end_of_day" {','if in.TimingMode != "lagos_end_of_day" && in.TimingMode != "lagos_explicit" {')
edit('internal/web/credit_handlers.go','canonical, timingErr := credit.CollectionInstant(in.DueDate, in.GraceHours)','''canonical, timingErr := credit.CollectionInstant(in.DueDate, in.GraceHours)
  if timingErr == nil && in.TimingMode == "lagos_explicit" {
   earliest := canonical
   canonical, timingErr = credit.ExplicitCollectionInstant(in.CollectionLocal)
   if timingErr == nil && canonical.Before(earliest) { writeProblem(w, 422, "credit_terms_invalid", "Bank collection must be after the agreed payment day and extra hours."); return }
  }''')
p='api/openapi.yaml';s=(r/p).read_text().replace("grace_hours: {type: integer, minimum: 0, maximum: 720}", "grace_hours: {type: integer, minimum: 0, maximum: 720}\n                collection_local: {type: string, description: Optional Nigerian local datetime in YYYY-MM-DDTHH:mm format.}",1);s=s.replace('timing_mode: {type: string, const: lagos_end_of_day}', 'timing_mode: {type: string, enum: [lagos_end_of_day, lagos_explicit]}');s=s.replace('timing_mode: {type: string, enum: [lagos_end_of_day], description: Server-validated Africa/Lagos end-of-day collection timing.}', 'timing_mode: {type: string, enum: [lagos_end_of_day, lagos_explicit], description: Server-validated Nigerian collection timing.}\n        collection_local: {type: string, description: Nigerian local datetime for lagos_explicit mode.}');save(p,s)
p='internal/credit/calendar_test.go';save(p,(r/p).read_text()+'''
func TestExplicitCollectionInstantRejectsNormalizedDates(t *testing.T) {
 for _, input := range []string{"2026-02-30T12:00", "2026-09-18T24:00", "2026-09-18T12:00Z", "2026-09-18T12:00+01:00", ""} {
  if _, err := ExplicitCollectionInstant(input); err == nil { t.Errorf("accepted invalid input %q", input) }
 }
 value, err := ExplicitCollectionInstant("2026-09-19T23:59")
 if err != nil || value.Format(time.RFC3339) != "2026-09-19T22:59:00Z" { t.Fatalf("incorrect Nigerian time: %v %v", value, err) }
}
''')
p='web/src/routes/app/credit/new/+page.svelte';s=(r/p).read_text();s=(r/'scripts/audit-full-form.svelte').read_text()+s.split('</script>',1)[1]
s=s.replace('<h1>Who took the goods?</h1>','<h1>Add a credit sale</h1>').replace('Who took it, what they took, how much and when they pay. Your customer sees exactly what you type here before they agree to it.','Add invoice details or instalments. Your customer reviews the exact terms before accepting.')
s=s.replace('<form onsubmit={submit} oninput={saveDraft} onchange={saveDraft}>','<form onsubmit={submit} oninput={() => { reviewed=""; saveDraft(); }} onchange={saveDraft}><fieldset disabled={busy} style="border:0;padding:0;margin:0;min-width:0">')
s=s.replace('</form></main>','</fieldset></form><label class="draft-choice"><input type="checkbox" bind:checked={keepDraft} onchange={saveDraft} /> Keep goods, amount and date on this device for up to 12 hours</label><p>Leave this off on a shared device. Customer, invoice and instalment details are not stored in a browser draft.</p></main>')
s=s.replace('onchange={loadCustomers} required','onchange={changeBusiness} required')
s=s.replace('{#if customers.length}<label>Search your customers', '{#if customerLoading}<p role="status">Checking your customers…</p>{:else if customerError}<div role="alert"><strong>Customer list unavailable</strong><p>{customerError}</p><button type="button" onclick={loadCustomers}>Try again</button></div>{:else if customers.length}<label>Search your customers')
s=s.replace('value={customer.buyer_user_id}', 'value={customerKey(customer)}').replace("customers.find((item)=>item.buyer_user_id===selectedBuyer)?.state",'selectedCustomer?.state')
s=s.replace('<strong>⚠️ Think twice:</strong>', '<strong>Overdue record:</strong>')
s=s.replace('onselect={(file)=>invoiceFile=file}', 'onselect={(file)=>{invoiceFile=file;reviewed="";}}')
s=s.replace('If still unpaid, Kredit may debit their bank from<input type="datetime-local" bind:value={collectionAt} required />','Optional later collection time (Nigerian time)<input type="datetime-local" bind:value={collectionAt} /><small>Leave blank to use the end of the payment day plus the extra hours. This time is always interpreted in Africa/Lagos, wherever your device is.</small>')
s=s.replace('<h2>Read it once more</h2>', '<h2 bind:this={reviewHeading} tabindex="-1">Review this sale</h2>')
s=s.replace("<strong>₦{principal||'0'}</strong>","<strong>{formatKobo(parseNaira(principal))}</strong>")
s=s.replace("new Date(`${dueDate}T12:00:00`).toLocaleDateString('en-NG')",'dateLabel(dueDate)')
s=s.replace('<p>We will not touch their bank before the day and time you set above.</p>','{#if reviewed}<p><strong>Bank collection may be considered from:</strong> {timeLabel(canonicalAt)}</p>{:else}<p>Check the terms to see the server-verified collection time.</p>{/if}<p>{collectionBoundary}</p><p>Saving creates a draft, not a payment or bank permission.</p>')
s=s.replace("{busy?'Saving…':'Save this sale'}", "{busy?'Checking…':reviewed?'Save draft sale':'Check terms'}")
s=s.replace('<div class="practice">','{#if loading}<p role="status">Checking your businesses…</p>{:else if !organizations.length}<p>Add or reopen your business from <a href="/app/overview">Home</a> before creating a sale.</p>{/if}<div class="practice">')
s=s.replace('.form-grid textarea,.no-customers,.wide{grid-column:span 2}', '.form-grid textarea{min-width:0}.no-customers,.wide{grid-column:span 2}')
s=s.replace('<label>Have we checked them?', '<label>Customer verification').replace('How much must they pay?', 'Sale amount').replace('Day they must pay','First payment date')
s=s.replace('<style>', '<style>.draft-choice{display:flex;align-items:center;gap:.7rem;margin-top:1.5rem;font-size:.9rem}.draft-choice input{min-height:1.2rem}.form-grid>label{min-width:0}.form-card{min-width:0}.form-card>div{min-width:0}')
s=s.replace('let savedItems:{name:string;amount:string}[]=$state([])', 'let savedItems:{name:string;amount:string}[]=$state([])')
save(p,s)
# Bind evidence to tests that execute the revised full form, not a placeholder assertion.
p='web/tests/audit-product-journeys.spec.ts';s=(r/p).read_text();s=s.replace("['quick-sale', '/app/credit/quick']", "['quick-sale', '/app/credit/quick'], ['full-sale', '/app/credit/new']")
s+='''

test('full sale keeps business identity and server-reviewed timing on the invoice path', async ({ page, context, baseURL }) => {
 await signedIn(page, context, baseURL);
 await page.route('**/api/v1/organizations/org-a/customers', route => send(route, { customers: [
  { buyer_user_id:'buyer-1', buyer_business_id:'business-1', legal_name:'First shop', trading_name:'First shop', state:'verified' },
  { buyer_user_id:'buyer-1', buyer_business_id:'business-2', legal_name:'Second shop', trading_name:'Second shop', state:'verified' }
 ] }));
 await page.route('**/api/v1/organizations/org-a/credit-terms/preview', route => send(route, { due_date:'2026-09-18', grace_hours:24, collection_at:'2026-09-19T22:59:00Z', timezone:'Africa/Lagos', cutoff:'23:59', timing_mode:'lagos_end_of_day' }));
 const saved: Record<string, unknown>[] = [];
 await page.route('**/api/v1/organizations/org-a/credit-requests', route => { if(route.request().method()==='POST'){saved.push(route.request().postDataJSON());return send(route,{request:{id:'created-sale'}},201);}return send(route,{requests:[]}); });
 await page.goto('/app/credit/new?organization=org-a');
 await page.getByRole('combobox',{name:'Customer',exact:true}).selectOption('buyer-1:business-2');
 await page.getByRole('textbox',{name:'Sale amount (₦)'}).fill('127,500.49');
 await page.getByRole('textbox',{name:'What goods are they taking?'}).fill('40 cartons of cooking oil');
 await page.getByLabel('First payment date').fill('2026-09-18');
 await page.getByRole('button',{name:'Check terms',exact:true}).click();
 await expect(page.getByRole('button',{name:'Save draft sale',exact:true})).toBeEnabled();
 expect(saved).toHaveLength(0);
 await expect(page.locator('.review')).toContainText('₦127,500.49');
 await page.getByRole('button',{name:'Save draft sale',exact:true}).click();
 await expect.poll(()=>saved.length).toBe(1);
 expect(saved[0]).toMatchObject({buyer_user_id:'buyer-1',buyer_business_id:'business-2',principal_kobo:12750049,collection_at:'2026-09-19T22:59:00Z',timing_mode:'lagos_end_of_day'});
});

test('full sale never presents an unavailable customer list as empty', async ({page,context,baseURL}) => {
 await signedIn(page,context,baseURL);
 await page.route('**/api/v1/organizations/org-a/customers', route=>send(route,{code:'financial_data_unavailable'},503));
 await page.goto('/app/credit/new');
 await expect(page.getByRole('alert')).toContainText('Customer list unavailable');
 await expect(page.getByText('You have not added a customer yet.',{exact:true})).toHaveCount(0);
 await expect(page.getByRole('button',{name:'Check terms',exact:true})).toBeDisabled();
});
'''
save(p,s)
subprocess.run(['gofmt','-w',*sorted(set(p for p in changed if p.endswith('.go')))],check=True)
save('docs/testing/audit-2026-09-06-refinement.md', '''# Audit refinement

The advanced invoice/instalment form now shares authenticated business scoping,
exact kobo, durable invoice/create operation identities, customer-fetch errors,
server-verified Lagos timing and an explicit review step. Legacy unscoped browser
records are removed rather than migrated between people. Browser tests target API
URLs only; they cannot accidentally replace application HTML with mock JSON.
Native account menus also wrap Tab/Shift-Tab across the browser focus boundary.
The public pricing API retains its established three-field contract.

No existing accepted agreement is rewritten. No live collection gate is enabled.
Test outcomes must be read from the checks on the final commit; this note itself
is not evidence that a test passed. External gates #5 and #7 remain open.
''')
Path('.tmp').mkdir(exist_ok=True)
Path('.tmp/finalize-paths.txt').write_text(''.join(p+'\n' for p in sorted(set(changed))))
print('Refinement source materialised with fixed-path assertions.')
