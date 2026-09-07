import { exactKobo, formatKobo, type KoboValue } from './money';
export type FeeTerms = { policy_revision: number; base_bps: number; collection_bps: number; min_fee_kobo?: KoboValue };
export function validFeeTerms(value: unknown): value is FeeTerms {
  if (!value || typeof value !== 'object') return false;
  const item = value as Partial<FeeTerms>;
  if (item.min_fee_kobo !== undefined && (exactKobo(item.min_fee_kobo) === null || exactKobo(item.min_fee_kobo)! < 0n)) return false;
  return Number.isInteger(item.base_bps) && Number.isInteger(item.collection_bps) && Number(item.base_bps) >= 0 && Number(item.base_bps) <= 1000 && Number(item.collection_bps) >= 0 && Number(item.collection_bps) <= 1000;
}
export function feeDisclosure(terms?: FeeTerms | null): string {
  if (!validFeeTerms(terms)) return 'Fee terms are unavailable. Please refresh before accepting.';
  const floor = exactKobo(terms.min_fee_kobo ?? 0);
  return `The seller pays ${terms.base_bps / 100}% when this sale becomes active${floor && floor > 0n ? `, with a minimum of ${formatKobo(floor)} capped at the sale amount` : ''}, plus ${terms.collection_bps / 100}% on any amount Kredit successfully collects after the permitted collection time. These fees are not added to the buyer’s principal.`;
}
export function feeForKobo(value: KoboValue, basisPoints: number): bigint | null {
  const amount = exactKobo(value);
  if (amount === null || amount < 0n || !Number.isInteger(basisPoints) || basisPoints < 0 || basisPoints > 1000) return null;
  return amount * BigInt(basisPoints) / 10000n;
}

export function baseFeeForKobo(value: KoboValue, terms: FeeTerms): bigint | null {
  if (!validFeeTerms(terms)) return null;
  const amount = exactKobo(value), percentage = feeForKobo(value, terms.base_bps), floor = exactKobo(terms.min_fee_kobo ?? 0);
  if (amount === null || amount < 0n || percentage === null || floor === null) return null;
  const required = percentage > floor ? percentage : floor;
  return amount < required ? amount : required;
}
