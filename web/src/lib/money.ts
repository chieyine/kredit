// Invalid input is negative so both positive-amount and nonnegative-amount
// API validations reject it. Never round a customer's fractional kobo.
export function parseNaira(input: string | number): number {
	const text = String(input).trim();
	if (!/^(?:\d+|\d{1,3}(?:,\d{3})+)(?:\.\d{1,2})?$/.test(text)) return -1;
	const [whole, fraction = ''] = text.replaceAll(',', '').split('.');
	const kobo = BigInt(whole) * 100n + BigInt(fraction.padEnd(2, '0'));
	if (kobo > BigInt(Number.MAX_SAFE_INTEGER)) return -1;
	return Number(kobo);
}

export type KoboValue = number | string | bigint | null | undefined;
export function exactKobo(value: KoboValue): bigint | null {
 if (typeof value === 'bigint') return value;
 if (typeof value === 'number') return Number.isSafeInteger(value) ? BigInt(value) : null;
 if (typeof value === 'string' && /^-?\d+$/.test(value)) return BigInt(value);
 return null;
}
export function sumKobo(values: KoboValue[]): bigint | null {
 let total = 0n;
 for (const value of values) { const amount = exactKobo(value); if (amount === null) return null; total += amount; }
 return total;
}
export function formatKobo(value: KoboValue, currency = 'NGN'): string {
 const amount = exactKobo(value);
 if (amount === null) return 'Amount unavailable';
 const absolute = amount < 0n ? -amount : amount;
 try {
  const whole = new Intl.NumberFormat('en-NG', { style:'currency', currency, minimumFractionDigits:0, maximumFractionDigits:0 }).format(absolute / 100n);
  return `${amount < 0n ? '-' : ''}${whole}.${(absolute % 100n).toString().padStart(2,'0')}`;
 } catch { return 'Amount unavailable'; }
}

const ONES = ['', 'one', 'two', 'three', 'four', 'five', 'six', 'seven', 'eight', 'nine', 'ten', 'eleven', 'twelve', 'thirteen', 'fourteen', 'fifteen', 'sixteen', 'seventeen', 'eighteen', 'nineteen'];
const TENS = ['', '', 'twenty', 'thirty', 'forty', 'fifty', 'sixty', 'seventy', 'eighty', 'ninety'];

function convertUnderThousand(n: number): string {
	let s = '';
	if (n >= 100) {
		s += ONES[Math.floor(n / 100)] + ' hundred';
		n %= 100;
		if (n > 0) s += ' and ';
	}
	if (n >= 20) {
		s += TENS[Math.floor(n / 10)];
		if (n % 10 > 0) s += '-' + ONES[n % 10];
	} else if (n > 0) {
		s += ONES[n];
	}
	return s;
}

export function verbalizeNaira(value: KoboValue): string {
	const amount = exactKobo(value);
	if (amount === null || amount <= 0n) return '';
	const naira = amount / 100n;
	const kobo = Number(amount % 100n);
	if (naira === 0n && kobo > 0) return `${kobo} kobo`;

	const scales = [
		{ val: 1_000_000_000_000_000_000n, name: 'quintillion' },
		{ val: 1_000_000_000_000_000n, name: 'quadrillion' },
		{ val: 1_000_000_000_000n, name: 'trillion' },
		{ val: 1_000_000_000n, name: 'billion' },
		{ val: 1_000_000n, name: 'million' },
		{ val: 1_000n, name: 'thousand' },
		{ val: 1n, name: '' }
	];

	let rem = naira;
	const parts: string[] = [];
	for (const { val, name } of scales) {
		const count = Number(rem / val);
		if (count > 0) {
			const chunk = convertUnderThousand(count);
			parts.push(name ? `${chunk} ${name}` : chunk);
			rem %= val;
		}
	}
	let result = parts.join(', ').replace(/, ([^,]*)$/, parts.length > 1 ? ' and $1' : '$1');
	if (result) {
		result = result.charAt(0).toUpperCase() + result.slice(1) + ' Naira';
		if (kobo > 0) result += ` and ${kobo} kobo`;
	}
	return result;
}

