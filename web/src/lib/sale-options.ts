import { MutationIntent } from '$lib/api/mutation';
import { record, text, UserFacingError } from '$lib/api/reliable';
import { parseNaira } from '$lib/money';
import { dateLabel } from '$lib/records';

export const defaultSaleOptions = () => ({
	invoiceReference: '',
	invoiceFile: null as File | null,
	scheduleType: 'one_time',
	count: 2,
	cadence: 'monthly',
	custom: ''
});
export type SaleOptions = ReturnType<typeof defaultSaleOptions>;
export function scheduleTerms(options: SaleOptions, principal: number, firstDate: string) {
	let custom: { amount_kobo: number; due_date: string }[] = [];
	if (options.scheduleType === 'equal' && (!Number.isInteger(options.count) || options.count < 2 || options.count > 60))
		throw new UserFacingError('Choose between 2 and 60 payments.');
	if (options.scheduleType === 'custom') {
		let previous = '';
		custom = options.custom
			.split('\n')
			.map((line) => line.trim())
			.filter(Boolean)
			.map((line, index) => {
				const match = /^(.+),\s*(\d{4}-\d{2}-\d{2})$/.exec(line);
				if (!match) throw new UserFacingError(`Check payment ${index + 1}: enter amount, YYYY-MM-DD.`);
				const amount = parseNaira(match[1]),
					date = match[2];
				if (amount <= 0 || dateLabel(date) === 'Date unavailable' || (previous && date <= previous))
					throw new UserFacingError('Payments need positive amounts and valid dates in order.');
				previous = date;
				return { amount_kobo: amount, due_date: date };
			});
		if (
			custom.length < 2 ||
			custom.length > 60 ||
			custom[0].due_date !== firstDate ||
			custom.reduce((sum, item) => sum + BigInt(item.amount_kobo), 0n) !== BigInt(principal)
		)
			throw new UserFacingError('Use 2–60 payments adding up to the sale amount, starting on the first payment date.');
	}
	return {
		schedule_type: options.scheduleType,
		schedule_count: options.scheduleType === 'one_time' ? 1 : options.count,
		schedule_cadence: options.scheduleType === 'equal' ? options.cadence : 'custom',
		month_end_policy: 'last_day',
		custom_schedule_items: custom
	};
}

/** A confirmed upload is reused when a later draft save needs recovery. */
export class InvoiceUpload {
	private saved = new Map<string, { id: string; hash: string }>();
	async upload(user: string, organization: string, file: File | null) {
		if (!file) return { id: '', hash: '' };
		if (
			!['application/pdf', 'image/jpeg', 'image/png'].includes(file.type) ||
			file.size <= 0 ||
			file.size > 2 * 1024 * 1024
		)
			throw new UserFacingError('Choose a PDF, JPG or PNG invoice up to 2 MB.');
		const hash = Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256', await file.arrayBuffer())), (b) =>
			b.toString(16).padStart(2, '0')
		).join('');
		const key = `${user}:${organization}:${hash}`;
		const previous = this.saved.get(key);
		if (previous) return previous;
		const content = await new Promise<string>((resolve, reject) => {
			const reader = new FileReader();
			reader.onerror = () => reject(new UserFacingError('We could not read the invoice.'));
			reader.onload = () => resolve(String(reader.result).split(',')[1] ?? '');
			reader.readAsDataURL(file);
		});
		const operation = new MutationIntent(
			`${user}:${organization}:invoice:${hash}`,
			`/api/v1/organizations/${encodeURIComponent(organization)}/documents`
		);
		const result = await operation.run(
			{
				purpose: 'credit_invoice',
				file_name: file.name,
				content_type: file.type,
				retention_class: 'financial_agreement',
				content_base64: content
			},
			(value) => {
				const document = record(record(value).document),
					id = text(document.id);
				if (!id || text(document.sha256) !== hash) throw new Error('The saved invoice could not be verified.');
				return { id, hash };
			}
		);
		this.saved.set(key, result);
		return result;
	}
}
