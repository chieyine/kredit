import { env } from '$env/dynamic/private';
import { error } from '@sveltejs/kit';
import { legalPublication } from './legal-publication';

export type LegalConfig = typeof legalPublication;
export function loadLegalConfig(): LegalConfig { return legalPublication; }

export function assertLaunchWebConfig() {
	loadLegalConfig();
	if (env.APP_ENV?.trim().toLowerCase() !== 'production') return;
	const origin = env.ORIGIN?.trim();
	const api = env.API_INTERNAL_URL?.trim();
	if (origin !== 'https://kredit.ng') {
		throw error(503, 'ORIGIN must be https://kredit.ng in production.');
	}
	let apiURL: URL;
	try {
		apiURL = new URL(api ?? '');
	} catch {
		throw error(503, 'API_INTERNAL_URL must be an absolute internal service URL.');
	}
	if (!['http:', 'https:'].includes(apiURL.protocol) || ['localhost', '127.0.0.1', '::1'].includes(apiURL.hostname)) {
		throw error(503, 'API_INTERNAL_URL must point to the production internal API service.');
	}
}
