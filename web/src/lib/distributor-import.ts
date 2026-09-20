import { normalizeNigerianPhone } from '$lib/api/reliable';

export type DistributorRow = {
  source_reference: string; target: string; target_type: string; legal_name: string;
  trading_name: string; business_type: string; business_address: string; industry: string;
};
const columns = ['source_reference','target','target_type','legal_name','trading_name','business_type','business_address','industry'];
export const distributorTemplate = columns.join(',')+'\nDIST-001,distributor@example.test,email,Example Distributors,,limited_company,"12 Market Road, Lagos",packaged food\n';

/** Bounded RFC4180-style parser: escaped quotes, multiline values and CRLF. */
export function parseDistributorCSV(source: string): DistributorRow[] {
  if (source.length > 1_000_000) throw new Error('Use a CSV file smaller than 1 MB.');
  const records: string[][] = []; let row: string[] = [], cell = '', quoted = false, closed = false;
  const input = source.replace(/^\uFEFF/, '');
  function finishCell() { row.push(cell.trim()); cell=''; closed=false; }
  function finishRow() { finishCell(); if(row.some(Boolean)) records.push(row); row=[]; if(records.length>201) throw new Error('Import at most 200 distributors at a time.'); }
  for(let i=0;i<input.length;i++) {
    const c=input[i];
    if(quoted) { if(c==='"') { if(input[i+1]==='"'){cell+='"';i++;}else{quoted=false;closed=true;} }else cell+=c; continue; }
    if(c==='"') { if(cell||closed)throw new Error('A quoted field must start immediately after a comma.'); quoted=true; }
    else if(c===',')finishCell();
    else if(c==='\n'||c==='\r'){if(c==='\r'&&input[i+1]==='\n')i++;finishRow();}
    else {if(closed)throw new Error('Unexpected text after a closing quote.');cell+=c;}
  }
  if(quoted)throw new Error('A quoted field is not closed.');
  if(cell||row.length||closed)finishRow();
  const header=records.shift();
  if(!header||header.join(',')!==columns.join(','))throw new Error('Use the template column names and order.');
  if(!records.length)throw new Error('Add at least one distributor.');
  const refs=new Set<string>(), contacts=new Set<string>();
  return records.map((values,index)=>{
    const line=index+2;
    if(values.length!==columns.length)throw new Error(`Row ${line}: expected ${columns.length} fields.`);
    const entry=Object.fromEntries(columns.map((key,n)=>[key,values[n]])) as DistributorRow;
    if(!/^[A-Za-z0-9][A-Za-z0-9._:-]{0,99}$/.test(entry.source_reference))throw new Error(`Row ${line}: use a stable customer reference, up to 100 letters, numbers, dots, colons or hyphens.`);
    entry.source_reference='roster:'+entry.source_reference;
    if(refs.has(entry.source_reference))throw new Error(`Row ${line}: duplicate customer reference.`);
    refs.add(entry.source_reference);
    entry.target_type=entry.target_type.toLowerCase();
    if(entry.target_type==='phone')entry.target=normalizeNigerianPhone(entry.target);
    else if(entry.target_type==='email'&&/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(entry.target))entry.target=entry.target.toLowerCase();
    else throw new Error(`Row ${line}: enter a valid email or Nigerian phone and matching target type.`);
    const contact=entry.target_type+':'+entry.target;
    if(contacts.has(contact))throw new Error(`Row ${line}: duplicate contact; review the intended business before inviting.`);
    contacts.add(contact);
    if(!entry.legal_name||!entry.business_address||!entry.industry)throw new Error(`Row ${line}: business name, address and industry are required.`);
    if(!['unregistered_business','registered_business','sole_proprietor','limited_company','partnership'].includes(entry.business_type))throw new Error(`Row ${line}: select a business type from the template instructions.`);
    if(values.some(v=>v.length>500))throw new Error(`Row ${line}: a field is longer than 500 characters.`);
    return entry;
  });
}
