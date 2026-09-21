/** Explicit business scope for purchasing list/report reads. Detail routes retain their own IDs. */
export function buyerEndpoint(path: string): string {
	if (typeof location === 'undefined') return path;
	const current = new URL(location.href),
		target = new URL(path, current.origin);
	for (const key of ['business_id', 'organization']) {
		const value = current.searchParams.get(key);
		if (value) target.searchParams.set(key, value);
	}
	return target.pathname + target.search;
}
