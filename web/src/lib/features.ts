/**
 * Simple Kredit.
 *
 * The seller works from three tabs (Who owes me, Customers, Money in) and one
 * button (Give goods on credit). The buyer works from the link he is sent.
 * Everything else is switched off here: the menu does not show it and the
 * address sends the person back to the nearest simple screen. No code or data
 * is removed, so any tool can come back by taking it out of this list.
 *
 * Set PUBLIC_KREDIT_FULL_WORKSPACE=1 to show every tool again (for example on
 * a staging copy while checking an old screen).
 */

type Parked = {
	/** The address, or the start of the addresses, being switched off. */
	path: string;
	/** Where the person is sent instead. */
	to: string;
	/** true: only this exact address. false: this address and everything under it. */
	exact?: boolean;
};

export const parkedPages: Parked[] = [
	// Giving credit: one form replaces the quick form and the full form.
	{ path: '/workspace/sales/quick', to: '/workspace/give' },
	{ path: '/workspace/sales/new', to: '/workspace/give' },
	{ path: '/workspace/sales/approvals', to: '/workspace/today' },
	{ path: '/workspace/sales/limits', to: '/workspace/today' },
	{ path: '/workspace/sales', to: '/workspace/today', exact: true },
	{ path: '/workspace/overdue', to: '/workspace/today' },
	{ path: '/workspace/disputes', to: '/workspace/today', exact: true },

	// Customers: one list for businesses and persons.
	{ path: '/workspace/partners', to: '/workspace/partners/customers', exact: true },
	{ path: '/workspace/partners/invitations', to: '/workspace/partners/customers' },
	{ path: '/workspace/partners/import', to: '/workspace/partners/customers' },
	{ path: '/workspace/partners/operations', to: '/workspace/partners/customers' },
	{ path: '/workspace/partners/access', to: '/workspace/partners/customers' },

	// Buying: the buyer keeps his link, What I owe, and each credit's own page.
	{ path: '/workspace/purchases', to: '/workspace/purchases/obligations', exact: true },
	{ path: '/workspace/purchases/permissions', to: '/workspace/purchases/obligations' },
	{ path: '/workspace/purchases/access', to: '/workspace/purchases/obligations' },
	{ path: '/workspace/purchases/trade-lines', to: '/workspace/purchases/obligations' },
	{ path: '/workspace/purchases/amendments', to: '/workspace/purchases/obligations' },
	{ path: '/workspace/purchases/history', to: '/workspace/purchases/obligations' },
	{ path: '/workspace/purchases/disputes', to: '/workspace/purchases/obligations', exact: true },

	// Money in is one screen.
	{ path: '/workspace/money', to: '/workspace/money/received', exact: true },

	// Business tools for later.
	{ path: '/workspace/team', to: '/workspace/settings' },
	{ path: '/workspace/settings/credit-policy', to: '/workspace/settings' },
	{ path: '/workspace/referral', to: '/workspace/today' },
	{ path: '/workspace/reports', to: '/workspace/today' },
	{ path: '/workspace/activity', to: '/workspace/today' },
	{ path: '/workspace/search', to: '/workspace/today' },

	// Public website.
	{ path: '/agents', to: '/' }
];

/** Where a parked address should send the person, or null when the page is in use. */
export function parkedDestination(pathname: string, fullWorkspace = false): string | null {
	if (fullWorkspace) return null;
	const path = pathname.length > 1 ? pathname.replace(/\/+$/, '') : pathname;
	for (const item of parkedPages) {
		if (path === item.path) return item.to;
		if (!item.exact && path.startsWith(item.path + '/')) return item.to;
	}
	return null;
}
