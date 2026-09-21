import { UserFacingError } from './api/reliable';
import { parseNaira } from './money';
import { lagosISO } from './business-time';

/** A confirmed receipt always names the actual transfer time, never UI time. */
export function actualPaymentTime(value: string): string {
	let instant: string;
	try {
		instant = lagosISO(value);
	} catch {
		throw new UserFacingError('Enter a valid payment date and time in Nigerian time.');
	}
	if (Date.parse(instant) > Date.now())
		throw new UserFacingError('Enter the actual payment date and time in Nigerian time. It cannot be in the future.');
	return instant;
}

export function positiveNaira(value: string, maximum?: number): number {
	const amount = parseNaira(value);
	if (amount <= 0) throw new UserFacingError('Enter a positive naira amount with no more than two decimal places.');
	if (maximum !== undefined && (!Number.isSafeInteger(maximum) || maximum < 0 || amount > maximum))
		throw new UserFacingError('Enter an amount within the available balance.');
	return amount;
}
