import { record, text } from '$lib/api/reliable';
import type { DistributorRow } from '$lib/distributor-import';

export type ImportBatch = {
  id: string;
  state: 'draft' | 'approved' | 'completed' | 'cancelled';
  row_count: number;
  source_hash: string;
  created_at: string;
  contacts?: DistributorRow[];
  completed_rows: number[];
};

export function decodeImportBatch(value: unknown): ImportBatch {
  const v = record(value);
  const state = text(v.state);
  if (!['draft', 'approved', 'completed', 'cancelled'].includes(state)) throw new Error('Invalid import state.');
  if (!Number.isInteger(v.row_count) || (v.row_count as number) < 1 || (v.row_count as number) > 200) throw new Error('Invalid import size.');
  const size = v.row_count as number;
  const hash = text(v.source_hash);
  const created = text(v.created_at);
  if (!/^[a-f0-9]{64}$/.test(hash) || !Number.isFinite(Date.parse(created))) throw new Error('Invalid import identity.');
  if (!Array.isArray(v.completed_rows) || !v.completed_rows.every(n => Number.isInteger(n) && n >= 1 && n <= size) || new Set(v.completed_rows).size !== v.completed_rows.length) throw new Error('Invalid import progress.');
  if (state === 'completed' && v.completed_rows.length !== size) throw new Error('Incomplete import progress.');
  let contacts: DistributorRow[] | undefined;
  if (v.contacts !== undefined) {
    if (!Array.isArray(v.contacts) || v.contacts.length !== size) throw new Error('Import contacts are unavailable.');
    contacts = v.contacts.map(value => {
      const c = record(value);
      return {
        source_reference: text(c.source_reference), target: text(c.target), target_type: text(c.target_type),
        legal_name: text(c.legal_name), trading_name: typeof c.trading_name === 'string' ? c.trading_name : '',
        business_type: text(c.business_type), business_address: text(c.business_address), industry: text(c.industry)
      };
    });
    if (new Set(contacts.map(c => c.source_reference)).size !== size) throw new Error('Duplicate saved contact identities.');
  }
  return { id: text(v.id), state: state as ImportBatch['state'], row_count: size, source_hash: hash, created_at: created, contacts, completed_rows: v.completed_rows as number[] };
}
