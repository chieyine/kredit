import { checkedJSON, record } from '$lib/api/reliable';

export function paymentDateAfter(days: number): string {
	const parts = new Intl.DateTimeFormat('en-CA', {
		timeZone: 'Africa/Lagos',
		year: 'numeric',
		month: '2-digit',
		day: '2-digit'
	}).formatToParts(new Date());
	const part = (name: string) => Number(parts.find((item) => item.type === name)?.value);
	const date = new Date(Date.UTC(part('year'), part('month') - 1, part('day') + days));
	return date.toISOString().slice(0, 10);
}

export async function loadSaleDefaults(organizationID: string, signal: AbortSignal) {
	return checkedJSON(
		`/api/v1/organizations/${encodeURIComponent(organizationID)}/onboarding`,
		(value) => {
			const profile = record(record(value).profile);
			const configured =
				typeof profile.default_credit_policy_updated_at === 'string' &&
				!profile.default_credit_policy_updated_at.startsWith('0001-');
			const paymentDays = configured ? profile.default_payment_days : 30;
			// A saved zero is omitted by Go's JSON encoder and is still a real preference.
			const graceHours = configured ? (profile.default_grace_hours ?? 0) : 24;
			if (
				typeof paymentDays !== 'number' ||
				!Number.isInteger(paymentDays) ||
				paymentDays < 1 ||
				paymentDays > 365 ||
				typeof graceHours !== 'number' ||
				!Number.isInteger(graceHours) ||
				graceHours < 0 ||
				graceHours > 720
			)
				throw new Error('Your saved payment settings could not be confirmed. Check your credit policy settings.');
			return { paymentDays, graceHours, dueDate: paymentDateAfter(paymentDays) };
		},
		{ signal }
	);
}
