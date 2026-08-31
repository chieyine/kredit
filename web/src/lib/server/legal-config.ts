import { env } from '$env/dynamic/private';
import { error } from '@sveltejs/kit';

export type LegalConfig = {
	active: boolean;
	entityName: string;
	serviceAddress: string;
	legalEmail: string;
	privacyEmail: string;
	effectiveDate: string;
	termsVersion: string;
	privacyVersion: string;
};

const requiredProductionValues = [
	'LEGAL_ENTITY_NAME',
	'LEGAL_SERVICE_ADDRESS',
	'LEGAL_CONTACT_EMAIL',
	'PRIVACY_CONTACT_EMAIL',
	'LEGAL_EFFECTIVE_DATE',
	'TERMS_VERSION',
	'PRIVACY_VERSION'
] as const;

function validEmail(value: string) {
	return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
}

function validDate(value: string) {
	if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
	const [year, month, day] = value.split('-').map(Number);
	const date = new Date(Date.UTC(year, month - 1, day));
	return date.getUTCFullYear() === year && date.getUTCMonth() === month - 1 && date.getUTCDate() === day;
}

export function loadLegalConfig(): LegalConfig {
	const production = env.APP_ENV?.trim().toLowerCase() === 'production';
	const active = env.LEGAL_DOCUMENTS_ACTIVE?.trim().toLowerCase() === 'true';
	const missing = requiredProductionValues.filter((name) => !env[name]?.trim());

	if (production && !active) {
		throw error(503, 'LEGAL_DOCUMENTS_ACTIVE must be true before production can start.');
	}
	if (production && missing.length) {
		throw error(503, `Missing public legal configuration: ${missing.join(', ')}`);
	}
	if (active && missing.length) {
		throw error(503, `Legal documents cannot be activated without: ${missing.join(', ')}`);
	}
	if (active && !validEmail(env.LEGAL_CONTACT_EMAIL ?? '')) {
		throw error(503, 'LEGAL_CONTACT_EMAIL must be a valid email address.');
	}
	if (active && !validEmail(env.PRIVACY_CONTACT_EMAIL ?? '')) {
		throw error(503, 'PRIVACY_CONTACT_EMAIL must be a valid email address.');
	}
	if (active && !validDate(env.LEGAL_EFFECTIVE_DATE ?? '')) {
		throw error(503, 'LEGAL_EFFECTIVE_DATE must use YYYY-MM-DD.');
	}
	if (production && env.TERMS_VERSION !== 'supplier-terms-v1') {
		throw error(503, 'TERMS_VERSION must match the supplier onboarding version.');
	}
	if (production && env.PRIVACY_VERSION !== 'privacy-v1') {
		throw error(503, 'PRIVACY_VERSION must match the supplier onboarding version.');
	}

	return {
		active,
		entityName: env.LEGAL_ENTITY_NAME?.trim() || 'Kredit operating company pending approval',
		serviceAddress: env.LEGAL_SERVICE_ADDRESS?.trim() || 'Service address pending approval',
		legalEmail: env.LEGAL_CONTACT_EMAIL?.trim() || '',
		privacyEmail: env.PRIVACY_CONTACT_EMAIL?.trim() || '',
		effectiveDate: env.LEGAL_EFFECTIVE_DATE?.trim() || '',
		termsVersion: env.TERMS_VERSION?.trim() || 'supplier-terms-v1',
		privacyVersion: env.PRIVACY_VERSION?.trim() || 'privacy-v1'
	};
}

export function assertLaunchWebConfig() {
	loadLegalConfig();
	if (env.APP_ENV?.trim().toLowerCase() !== 'production') return;
	const origin = env.ORIGIN?.trim();
	const api = env.API_INTERNAL_URL?.trim();
	if (origin !== 'https://kredit.com.ng') {
		throw error(503, 'ORIGIN must be https://kredit.com.ng in production.');
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
