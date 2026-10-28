/**
 * Simple Kredit: three daily views and one sale-entry button.
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
