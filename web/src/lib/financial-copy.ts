import { formatKobo, type KoboValue } from './money';
export const creditBoundary = 'You provide the goods. Kredit does not lend money or guarantee repayment.';
export const collectionBoundary = 'Bank debit needs valid permission, the agreed collection time and the applicable payment and dispute checks. A debit can still fail.';
export const submittedDebitCaveat = 'A debit request already sent to the bank may still complete.';
export function disputeEffectCopy(effect: string, amount?: KoboValue): string {
  if (effect === 'FULL_BLOCK') return `New bank debit requests for this sale will be on hold once this report is saved. ${submittedDebitCaveat}`;
  if (effect === 'NO_AUTOMATIC_BLOCK') return `This report does not request an automatic hold. Eligible payments may continue. ${submittedDebitCaveat}`;
  return `Only the amount in question${amount == null ? '' : ` (${formatKobo(amount)})`} will be held once this report is saved. Undisputed amounts may still be collected. ${submittedDebitCaveat}`;
}
export function hostedAuthorizationURL(value: string, provider: string): string | null {
  if (provider !== 'mono-sweep') return null;
  try {
    const url = new URL(value);
    return url.protocol === 'https:' && url.hostname === 'authorise.mono.co' && !url.port && !url.username && !url.password && url.pathname !== '/' ? url.href : null;
  } catch { return null; }
}
export function acceptanceMessage(state: string): string {
  if (state === 'READY_TO_RELEASE') return 'Sale accepted. Bank permission is ready, and the seller can arrange your goods.';
  if (state === 'BUYER_ACCEPTED') return 'Sale accepted. Bank permission must be ready before the seller can release the goods.';
  return 'Your sale has been updated. Check its current status below.';
}
