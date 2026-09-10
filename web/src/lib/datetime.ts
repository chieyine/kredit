// The business runs on one clock: Africa/Lagos. A timestamp rendered in the
// reader's own zone made a Lagos evening look like the next morning to anyone
// travelling, and a payment day is a contractual fact, not a local convenience.
const LAGOS = 'Africa/Lagos';

// datetime-local requires wall time, not an ISO timestamp with its zone removed.
export function localDateTime(value: string | Date): string {
	const date = new Date(value);
	const pad = (n: number) => String(n).padStart(2, '0');
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

/** A date a person can read: "12 March 2026". Empty when there is no date. */
export function readableDate(value: string | Date | null | undefined): string {
	if (!value) return '';
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return '';
	return new Intl.DateTimeFormat('en-NG', {
		day: 'numeric',
		month: 'long',
		year: 'numeric',
		timeZone: LAGOS
	}).format(date);
}

/** The same date with the time, for records where the hour matters. */
export function readableDateTime(value: string | Date | null | undefined): string {
	if (!value) return '';
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return '';
	return new Intl.DateTimeFormat('en-NG', {
		day: 'numeric',
		month: 'long',
		year: 'numeric',
		hour: 'numeric',
		minute: '2-digit',
		timeZone: LAGOS
	}).format(date);
}
