import { parseNaira, formatKobo } from '$lib/money';
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
export const money = (n: number) => formatKobo(n);
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
	const terms = record(r.terms);
	const validAmount = (value: unknown) => typeof value === 'number' && Number.isSafeInteger(value) && value >= 0;
	const validSchedule = (value: unknown) =>
		Array.isArray(value) &&
		value.every((entry) => {
			const due = record(entry);
			return (
				typeof due.date === 'string' &&
				/^\d{4}-\d{2}-\d{2}$/.test(due.date) &&
				validAmount(due.amount_kobo) &&
				validAmount(due.paid_kobo)
			);
		});
	const validEvents =
		Array.isArray(r.events) &&
		r.events.every((value) => {
			const event = record(value);
			return (
				['id', 'action', 'reference', 'related_id', 'note', 'occurred_at'].every(
					(key) => typeof event[key] === 'string'
				) && validAmount(event.amount_kobo)
			);
		});
	if (
		typeof r.id !== 'string' ||
		!r.id ||
		typeof r.organization_id !== 'string' ||
		typeof r.target !== 'string' ||
		typeof r.state !== 'string' ||
		typeof r.customer_name !== 'string' ||
		typeof r.case_state !== 'string' ||
		typeof r.release_eligible !== 'boolean' ||
		typeof terms.item !== 'string' ||
		!Number.isSafeInteger(r.version) ||
		!validEvents ||
		!validAmount(terms.total_kobo) ||
		!['paid_kobo', 'refunded_kobo', 'reduction_kobo', 'outstanding_kobo', 'refund_due_kobo'].every((key) =>
			validAmount(r[key])
		) ||
		!validSchedule(terms.schedule) ||
		!validSchedule(r.schedule_progress)
	)
		throw new Error('The purchase response could not be verified. Refresh before acting.');
	return r as unknown as Purchase;
}
export async function read(url: string) {
	return checkedJSON(url, record);
}

/** Totals require every page. Never return a partial list as a complete balance. */
export async function readAllPurchases(url: string, signal?: AbortSignal): Promise<Purchase[]> {
	const deadline = AbortSignal.timeout(20_000);
	const requestSignal = signal ? AbortSignal.any([signal, deadline]) : deadline;
	const items: Purchase[] = [];
	const cursors = new Set<string>();
	const ids = new Set<string>();
	let cursor = '';
	do {
		const path = cursor ? `${url}${url.includes('?') ? '&' : '?'}before=${encodeURIComponent(cursor)}` : url;
		const page = await checkedJSON(
			path,
			(value) => {
				const body = record(value);
				if (!Array.isArray(body.items) || typeof body.next_cursor !== 'string')
					throw new Error('The complete purchase list could not be verified.');
				return { items: body.items.map(purchase), cursor: body.next_cursor };
			},
			{ signal: requestSignal }
		);
		for (const item of page.items) {
			if (ids.has(item.id)) throw new Error('The purchase list changed. Refresh before using its total.');
			ids.add(item.id);
			items.push(item);
		}
		cursor = page.cursor;
		if (cursor && (cursors.has(cursor) || !page.items.length))
			throw new Error('The complete purchase list could not be verified.');
		cursors.add(cursor);
	} while (cursor);
	return items;
}

export function activePurchase(sale: Pick<Purchase, 'state'>): boolean {
	return sale.state === 'active' || sale.state === 'received';
}

export function pendingPurchaseClaims(sale: Purchase): SaleEvent[] {
	const resolved = new Set(
		sale.events.filter((event) => ['payment', 'reject_claim'].includes(event.action)).map((event) => event.related_id)
	);
	return sale.events.filter((event) => event.action === 'claim' && !resolved.has(event.id));
}

/** Price reductions cap what is due even when the original schedule is retained. */
export function purchaseOverdue(
	sale: Pick<Purchase, 'state' | 'schedule_progress' | 'outstanding_kobo'>,
	today: string
): bigint {
	if (!activePurchase(sale)) return 0n;
	const due = sale.schedule_progress
		.filter((item) => item.date < today)
		.reduce((sum, item) => sum + BigInt(Math.max(0, item.amount_kobo - item.paid_kobo)), 0n);
	const outstanding = BigInt(sale.outstanding_kobo);
	return due < outstanding ? due : outstanding;
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
