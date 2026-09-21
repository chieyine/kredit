import { parseNaira } from '$lib/money';
import { MutationIntent } from '$lib/api/mutation';
import { boundedFetch, record, RequestError, type Decoder } from '$lib/api/reliable';
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
export const label = (v: string) => v.replaceAll('_', ' ');
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
	const response = await boundedFetch(url);
	const data = await response.json();
	if (!response.ok) throw new RequestError(data.detail || 'Could not load this purchase.', response.status);
	return data;
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
	async send(url: string, body: unknown, decode: Decoder<unknown> = purchase): Promise<unknown> {
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
