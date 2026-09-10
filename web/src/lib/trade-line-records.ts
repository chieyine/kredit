import { record, text } from './api/reliable';
import { kobo } from './records';
import { validFeeTerms } from './fee-terms';

export function tradeLine(value: unknown): Record<string, any> {
 const row = record(value);
 for (const key of ['id','supplier_organization_id','buyer_user_id','buyer_business_id','state']) if (!text(row[key])) throw new Error('Incomplete customer limit');
 for (const key of ['approved_limit_kobo','current_exposure_kobo','reserved_pending_kobo','available_limit_kobo']) kobo(row[key]);
 if (!Number.isSafeInteger(row.version) || Number(row.version) < 1) throw new Error('Incomplete limit version');
 return row;
}
export function drawdown(value: unknown): Record<string, any> {
 const row = record(value);
 for (const key of ['id','trade_line_id','state','goods_description','due_date','collection_at','agreement_hash']) text(row[key]);
 kobo(row.principal_kobo);
 if(row.legal_versions != null){const versions=record(row.legal_versions);if(!text(versions.terms_version)||!text(versions.privacy_version))throw new Error('Incomplete legal references');}
 if (row.fee_terms != null && !validFeeTerms(row.fee_terms)) throw new Error('Incomplete fee terms');
 if (!Number.isSafeInteger(row.grace_hours) || Number(row.grace_hours) < 0) throw new Error('Incomplete grace period');
 return row;
}
export function tradeStatement(value: unknown): { line: Record<string, any>; drawdowns: Record<string, any>[] } {
 const result = record(value), line = tradeLine(result.line);
 if (!Array.isArray(result.drawdowns)) throw new Error('Incomplete sales list');
 const drawdowns = result.drawdowns.map(drawdown);
 if (drawdowns.some(item => item.trade_line_id !== line.id)) throw new Error('Sale belongs to another customer limit');
 return { line, drawdowns };
}
