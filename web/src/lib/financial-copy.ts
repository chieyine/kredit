import { formatKobo, type KoboValue } from './money';
export const creditBoundary = 'You provide the goods. Kredit does not lend money or guarantee repayment.';
export const collectionBoundary =
	'Bank debit needs valid permission, the agreed collection time and the applicable payment and dispute checks. A debit can still fail.';
export const submittedDebitCaveat = 'A debit request already sent to the bank may still complete.';
/**
 * What a problem report does to bank debits. Before saving it describes what
 * will happen; once saved (`saved`), what now applies.
 */
export function disputeEffectCopy(effect: string, amount?: KoboValue, saved = false): string {
	const when = saved ? 'are now' : 'will be';
	if (effect === 'FULL_BLOCK') return `New bank debit requests for this sale ${when} on hold. ${submittedDebitCaveat}`;
	if (effect === 'NO_AUTOMATIC_BLOCK')
		return `This report does not ask for an automatic hold. Eligible payments may continue. ${submittedDebitCaveat}`;
	return `Only the amount in question${amount == null ? '' : ` (${formatKobo(amount)})`} ${saved ? 'is now' : 'will be'} on hold. Undisputed amounts may still be collected. ${submittedDebitCaveat}`;
}
export function hostedAuthorizationURL(value: string, _provider: string): string | null {
	if (/^\/workspace\/purchases\/bank-authorization\/[a-f0-9]{32}$/.test(value)) return value;
	try {
		const url = new URL(value);
		const allowed =
			url.hostname === 'authorise.mono.co' ||
			url.hostname === 'link.paystack.com' ||
			url.hostname === 'checkout.paystack.com';
		return url.protocol === 'https:' && allowed && !url.port && !url.username && !url.password && url.pathname !== '/'
			? url.href
			: null;
	} catch {
		return null;
	}
}
export function acceptanceMessage(state: string): string {
	if (state === 'READY_TO_RELEASE')
		return 'Sale accepted. Bank permission is ready, and the seller can arrange your goods.';
	if (state === 'BUYER_ACCEPTED')
		return 'Sale accepted. Bank permission must be ready before the seller can release the goods.';
	return 'Your sale has been updated. Check its current status below.';
}
