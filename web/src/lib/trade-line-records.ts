import { record, text } from './api/reliable';
import { kobo } from './records';
import { validFeeTerms, type FeeTerms } from './fee-terms';
import type { KoboValue } from './money';

/** A customer's approved credit limit, as the trade-line endpoints return it. */
export interface TradeLine {
	id: string;
	supplier_organization_id: string;
	buyer_user_id: string;
	buyer_business_id: string;
	approved_limit_kobo: KoboValue;
	current_exposure_kobo: KoboValue;
	reserved_pending_kobo: KoboValue;
	available_limit_kobo: KoboValue;
	state: string;
	version: number;
	cadence?: string;
	default_grace_hours?: number;
	start_at?: string;
	end_at?: string;
	mandate_active?: boolean;
	suspension_reason?: string;
}

/** One sale drawn against a trade line. Optional fields appear as the sale progresses. */
export interface Drawdown {
	id: string;
	trade_line_id: string;
	state: string;
	goods_description: string;
	due_date: string;
	collection_at: string;
	agreement_hash: string;
	principal_kobo: KoboValue;
	grace_hours: number;
	legal_versions?: { terms_version: string; privacy_version: string };
	fee_terms?: FeeTerms;
	outstanding_kobo?: KoboValue;
	invoice_reference?: string;
	obligation_id?: string;
	delivery_method?: string;
	release_notes?: string;
	release_evidence_reference?: string;
	release_actor_id?: string;
	released_at?: string;
	receipt_state?: string;
	receipt_issue_reason?: string;
	receipt_dispute_id?: string;
	receipt_at?: string;
	activated_at?: string;
	created_at?: string;
}

export function tradeLine(value: unknown): TradeLine {
	const row = record(value);
	for (const key of ['id', 'supplier_organization_id', 'buyer_user_id', 'buyer_business_id', 'state'])
		if (!text(row[key])) throw new Error('Incomplete customer limit');
	for (const key of ['approved_limit_kobo', 'current_exposure_kobo', 'reserved_pending_kobo', 'available_limit_kobo'])
		kobo(row[key]);
	if (!Number.isSafeInteger(row.version) || Number(row.version) < 1) throw new Error('Incomplete limit version');
	return row as unknown as TradeLine;
}
export function drawdown(value: unknown): Drawdown {
	const row = record(value);
	for (const key of [
		'id',
		'trade_line_id',
		'state',
		'goods_description',
		'due_date',
		'collection_at',
		'agreement_hash'
	])
		text(row[key]);
	kobo(row.principal_kobo);
	if (row.legal_versions != null) {
		const versions = record(row.legal_versions);
		if (!text(versions.terms_version) || !text(versions.privacy_version))
			throw new Error('Incomplete legal references');
	}
	if (row.fee_terms != null && !validFeeTerms(row.fee_terms)) throw new Error('Incomplete fee terms');
	if (!Number.isSafeInteger(row.grace_hours) || Number(row.grace_hours) < 0) throw new Error('Incomplete grace period');
	return row as unknown as Drawdown;
}
export interface TradeStatement {
	line: TradeLine;
	drawdowns: Drawdown[];
}
export function tradeStatement(value: unknown): TradeStatement {
	const result = record(value),
		line = tradeLine(result.line);
	if (!Array.isArray(result.drawdowns)) throw new Error('Incomplete sales list');
	const drawdowns = result.drawdowns.map(drawdown);
	if (drawdowns.some((item) => item.trade_line_id !== line.id))
		throw new Error('Sale belongs to another customer limit');
	return { line, drawdowns };
}
