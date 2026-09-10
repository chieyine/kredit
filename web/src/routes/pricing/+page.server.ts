import { loadWebsiteCopy } from '$lib/server/website-content';
import { validFeeTerms, type FeeTerms } from '$lib/fee-terms';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, setHeaders }) => {
  // Prices are public policy. Never forward session state or cache superseded rates.
  setHeaders({ 'cache-control': 'no-store' });
  const copy = await loadWebsiteCopy(fetch, 'pricing');
  try {
    const response = await fetch('/api/v1/pricing', { credentials: 'omit', signal: AbortSignal.timeout(5000) });
    if (!response.ok) throw new Error('Pricing unavailable');
    const rates: unknown = await response.json();
    if (!validFeeTerms(rates)) throw new Error('Invalid pricing');
    return { rates: rates as FeeTerms, copy };
  } catch { return { rates: null, copy }; }
};
