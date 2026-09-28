import { checkedJSON, record, rows, text, RequestError } from '$lib/api/reliable';
import type { Due } from '$lib/consumer';

export type PurchaseSummary = {
	id: string;
	organization_id: string;
	target: string;
	customer_name: string;
	state: string;
	case_state: string;
	terms: { item: string; total_kobo: number };
	schedule_progress: Due[];
	paid_kobo: number;
	refunded_kobo: number;
	outstanding_kobo: number;
	refund_due_kobo: number;
	release_eligible: boolean;
	pending_claim_count: number;
};
function amount(value: unknown): number {
	if (typeof value !== 'number' || !Number.isSafeInteger(value) || value < 0)
		throw new RequestError('The purchase balance could not be verified.', 0, 'invalid_response');
	return value;
}
export function purchaseSummary(value: unknown): PurchaseSummary {
	const row = record(value),
		terms = record(row.terms);
	if (typeof row.release_eligible !== 'boolean' || !Array.isArray(row.schedule_progress))
		throw new RequestError('The purchase summary was incomplete.', 0, 'invalid_response');
	return {
		id: text(row.id),
		organization_id: text(row.organization_id),
		target: text(row.target),
		customer_name: text(row.customer_name),
		state: text(row.state),
		case_state: text(row.case_state),
		terms: { item: text(terms.item), total_kobo: amount(terms.total_kobo) },
		schedule_progress: row.schedule_progress.map((value) => {
			const due = record(value),
				date = text(due.date);
			if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) throw new RequestError('The payment date was incomplete.');
			return { date, amount_kobo: amount(due.amount_kobo), paid_kobo: amount(due.paid_kobo) };
		}),
		paid_kobo: amount(row.paid_kobo),
		refunded_kobo: amount(row.refunded_kobo),
		outstanding_kobo: amount(row.outstanding_kobo),
		refund_due_kobo: amount(row.refund_due_kobo),
		release_eligible: row.release_eligible,
		pending_claim_count: amount(row.pending_claim_count)
	};
}
export function readPurchaseSummaries(organizationID: string, signal?: AbortSignal) {
	return checkedJSON(
		`/api/v1/organizations/${encodeURIComponent(organizationID)}/consumer-sales/summary`,
		rows('items', purchaseSummary),
		{ signal }
	);
}
