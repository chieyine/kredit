import type { PageServerLoad } from './$types';
import { rows } from '$lib/api/reliable';
import {
	organization,
	paymentRow,
	receivables,
	saleView,
	workRow,
	type Organization,
	type PaymentRow,
	type Receivables,
	type SaleView,
	type WorkRow
} from '$lib/records';

/** Server-side prefetch for the supplier dashboard.
 *
 * The page used to render an empty shell, hydrate, fetch the business list,
 * then fetch seven resources - two sequential round trips from the browser
 * before the first number appears. On a congested mobile link, which is the
 * normal case for the traders this is built for, that is seconds of blank
 * screen. Running the same reads here turns them into in-datacentre calls
 * streamed into the first response.
 *
 * This is deliberately additive. Anything unexpected - no session, an API
 * error, a business the account cannot see - returns null and the existing
 * client path runs exactly as before. The page must never fail because its
 * prefetch did.
 */

export type Prefetched = {
	organizationID: string;
	businesses: Organization[];
	sales: SaleView[];
	payments: PaymentRow[];
	overdue: WorkRow[];
	claims: WorkRow[];
	disputes: WorkRow[];
	due: WorkRow[];
	summary: Receivables;
	checkedAt: string;
};

export const load: PageServerLoad = async ({ fetch, url, depends }) => {
	// Re-run when the client switches business via ?organization=.
	depends('workspace:today');

	const read = async <T>(path: string, decode: (value: unknown) => T): Promise<T> => {
		const response = await fetch(path);
		if (!response.ok) throw new Error(`${path} responded ${response.status}`);
		return decode(await response.json());
	};

	try {
		const businesses = await read('/api/v1/organizations', rows('organizations', organization));
		const requested = url.searchParams.get('organization');
		// Mirror requestedWorkspace(): an explicit business the account cannot
		// see is an error the client must show, not something to paper over.
		if (requested && !businesses.some((item) => item.id === requested)) return { prefetched: null };
		const organizationID = requested || businesses[0]?.id || '';
		if (!organizationID) return { prefetched: null };

		const root = `/api/v1/organizations/${encodeURIComponent(organizationID)}`;
		const [sales, payments, overdue, claims, disputes, summary, due] = await Promise.all([
			read(`${root}/credit-requests`, rows('requests', saleView)),
			read(`${root}/payments`, rows('payments', paymentRow)),
			read(`${root}/overdue`, rows('overdue', workRow)),
			read(`${root}/payment-claims`, rows('payment_claims', workRow)),
			read(`${root}/disputes`, rows('disputes', workRow)),
			read(`${root}/reports/receivables`, receivables),
			read(`${root}/due`, rows('due', workRow))
		]);

		const prefetched: Prefetched = {
			organizationID,
			businesses,
			sales,
			payments,
			overdue,
			claims,
			disputes,
			due,
			summary,
			checkedAt: new Date().toISOString()
		};
		return { prefetched };
	} catch {
		// Fall back to the client path rather than failing the page.
		return { prefetched: null };
	}
};
