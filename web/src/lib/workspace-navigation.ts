/**
 * Simple Kredit: three tabs and one button. See $lib/features for the tools
 * that are switched off and where their addresses now lead.
 */
export const workspacePrimary: [string, string, string][] = [
	['Who owes me', '/workspace/today', 'home'],
	['Give goods on credit', '/workspace/give', 'add'],
	['Customers', '/workspace/partners/customers', 'customers'],
	['Money in', '/workspace/money/received', 'payments']
];
export const workspaceMore: [string, string, string][] = [
	['Settings', '/workspace/settings', 'Business'],
	['What I owe suppliers', '/workspace/purchases/obligations', 'Business'],
	['My account', '/account', 'You'],
	['Get help', '/workspace/help', 'You']
];
export const workspaceLinks: [string, string][] = [...workspacePrimary, ...workspaceMore].map(([name, href]) => [
	name,
	href
]);
/** No sub-tabs: each simple screen carries what it needs on the page itself. */
export const workspaceSections: { root: string; label: string; links: [string, string][] }[] = [];

/** Scope belongs in the URL so refreshes and shared workspace links stay explicit. */
export function workspaceHref(href: string, current: URL): string {
	if (!href.startsWith('/workspace')) return href;
	const target = new URL(href, current.origin);
	for (const key of ['organization', ...(target.pathname.startsWith('/workspace/purchases') ? ['business_id'] : [])]) {
		if (
			key === 'business_id' &&
			target.searchParams.has('organization') &&
			target.searchParams.get('organization') !== current.searchParams.get('organization')
		)
			continue;
		if (
			key === 'organization' &&
			target.searchParams.has('business_id') &&
			target.searchParams.get('business_id') !== current.searchParams.get('business_id')
		)
			continue;
		const value = current.searchParams.get(key);
		if (value && !target.searchParams.has(key)) target.searchParams.set(key, value);
	}
	return target.pathname + target.search + target.hash;
}
