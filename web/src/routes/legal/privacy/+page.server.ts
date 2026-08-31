import { loadLegalConfig } from '$lib/server/legal-config';

export function load() {
	return { legal: loadLegalConfig() };
}
