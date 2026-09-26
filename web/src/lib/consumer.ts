import { parseNaira } from '$lib/money';
import { MutationIntent } from '$lib/api/mutation';
import { checkedJSON, record, type Decoder } from '$lib/api/reliable';
export type Due = { date: string; amount_kobo: number; paid_kobo: number };
export type Terms = {
	item: string;
	quantity: number;
	total_kobo: number;
	deposit_kobo: number;
	deposit_date: string;
	first_date: string;
	count: number;
	cadence: string;
	fulfillment: string;
	threshold_percent: number;
	delivery_days: number;
	stock_reference: string;
	stock_reserved: boolean;
	returns_policy: string;
	bank_name: string;
	account_name: string;
	account_number: string;
	seller_name: string;
	seller_address: string;
	terms_version: string;
	schedule: Due[];
};
export type SaleEvent = {
	id: string;
	action: string;
	amount_kobo: number;
	reference: string;
	related_id: string;
	note: string;
	occurred_at: string;
};
export type Purchase = {
	id: string;
	organization_id: string;
	buyer_user_id: string;
	target_type: string;
	target: string;
	terms: Terms;
	schedule_progress: Due[];
	agreement_hash: string;
	state: string;
	customer_name: string;
	delivery_address: string;
	accepted_at: string | null;
	released_at: string | null;
	received_at: string | null;
	case_state: string;
	version: number;
	events: SaleEvent[];
	paid_kobo: number;
	refunded_kobo: number;
	reduction_kobo: number;
	outstanding_kobo: number;
	refund_due_kobo: number;
	release_eligible: boolean;
	delivery_due_at: string | null;
	role: string;
};
export const money = (n: number) =>
	new Intl.NumberFormat('en-NG', { style: 'currency', currency: 'NGN' }).format(n / 100);
const words: Record<string, string> = {
	offered: 'Waiting for the customer',
	active: 'Being paid',
	received: 'Delivered',
	completed: 'Paid in full',
	cancelled: 'Cancelled',
	declined: 'Declined',
	requested: 'Return requested',
	escalated: 'Sent to Kredit support',
	rejected: 'Return refused',
	approved: 'Return approved'
};
const sentence = (v: string) => v.replaceAll('_', ' ').replace(/^./, (letter) => letter.toUpperCase());
/** A consumer sale's state or return case in plain words. */
export const label = (v: string) => words[v] ?? sentence(v);
// Each form's heading matches the button that opened it.
const actions: Record<string, string> = {
	accept: 'Review and accept',
	decline: 'Decline this offer',
	cancel: 'Cancel',
	release: 'Record dispatch or handover',
	received: 'Confirm you received the goods',
	payment: 'Confirm money received',
	claim: 'Report a payment you made',
	reject_claim: 'Review an unmatched payment',
	reverse_payment: 'Correct a recorded receipt',
	reduce_price: 'Reduce the sale price',
	refund: 'Record a refund paid to the customer',
	request_return: 'Report a problem or ask to return it',
	approve_return: 'Approve the return and full refund',
	reject_return: 'Decline the return',
	escalate: 'Ask Kredit to review the decision'
};
const events: Record<string, string> = {
	accept: 'Accepted',
	decline: 'Declined',
	cancel: 'Cancelled',
	release: 'Goods sent',
	received: 'Goods received',
	payment: 'Payment recorded',
	claim: 'Payment reported',
	reject_claim: 'Reported payment not found',
	reverse_payment: 'Payment reversed',
	reduce_price: 'Price reduced',
	refund: 'Refund recorded',
	request_return: 'Return requested',
	approve_return: 'Return approved',
	reject_return: 'Return refused',
	escalate: 'Sent to Kredit support'
};
/** A recorded step in a consumer sale's history, in the past tense. */
export const eventLabel = (v: string) => events[v] ?? sentence(v);
/** What a button on a consumer sale does, as a heading for its form. */
export const actionLabel = (v: string) => actions[v] ?? sentence(v);
export function kobo(v: string) {
	const amount = parseNaira(v);
	if (amount < 0) throw new Error('Enter a valid naira amount with no more than two decimal places.');
	return amount;
}
export function purchase(v: unknown): Purchase {
	const r = record(v);
	if (
		typeof r.id !== 'string' ||
		typeof r.version !== 'number' ||
		!Array.isArray(r.events) ||
		!Array.isArray(record(r.terms).schedule)
	)
		throw new Error('The purchase response could not be verified. Refresh before acting.');
	return r as unknown as Purchase;
}
export async function read(url: string) {
	return checkedJSON(url, record);
}
// Persist request identities/digests, never purchase or banking evidence.
export class Mutation {
	private intents = new Map<string, MutationIntent>();
	pending: { url: string; body: string; decode: Decoder<unknown> } | null = null;
	async retry(): Promise<unknown> {
		if (!this.pending)
			throw new Error('Re-enter the original details to recover this action, or check its saved record.');
		return this.send(this.pending.url, JSON.parse(this.pending.body), this.pending.decode);
	}
	async send(url: string, body: unknown, decode: Decoder<unknown>): Promise<unknown> {
		const encoded = JSON.stringify(body);
		if (this.pending && (this.pending.url !== url || this.pending.body !== encoded))
			throw new Error('Retry the previous action with its original details before starting another.');
		let intent = this.intents.get(url);
		if (!intent) {
			intent = new MutationIntent('consumer', url);
			this.intents.set(url, intent);
		}
		this.pending = { url, body: encoded, decode };
		try {
			const result = await intent.run(body, decode);
			this.pending = null;
			return result;
		} catch (error) {
			if (!intent.unresolved) this.pending = null;
			throw error;
		}
	}
}
