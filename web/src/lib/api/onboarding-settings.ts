import { adminGet } from '$lib/admin-client';
import { record, text } from './reliable';

/**
 * A business's onboarding profile. `version` is verified because every save
 * sends it back as the expected version; the named text fields are the ones
 * the setup screens read, and the rest stay available as unknown values.
 */
export type SettingsProfile = {
	version: number;
	authorized_representative_name?: string;
	authorized_representative_title?: string;
	kyb_provider_reference?: string;
	kyb_state?: string;
	terms_version?: string;
	privacy_version?: string;
	[key: string]: unknown;
};
const profileText = [
	'authorized_representative_name',
	'authorized_representative_title',
	'kyb_provider_reference',
	'kyb_state',
	'terms_version',
	'privacy_version'
];

export function settingsProfile(value: unknown): SettingsProfile {
	const profile = record(value);
	if (!Number.isSafeInteger(profile.version) || Number(profile.version) < 0)
		throw new Error('We could not confirm the current settings. Refresh this page before saving.');
	for (const key of profileText)
		if (profile[key] != null && typeof profile[key] !== 'string')
			throw new Error('We could not confirm the current settings. Refresh this page before saving.');
	return { ...profile, version: Number(profile.version) } as SettingsProfile;
}

export async function loadOnboardingSettings(
	selectedOrganization = new URLSearchParams(window.location.search).get('organization')
) {
	const organizations = await adminGet('/api/v1/organizations');
	if (!Array.isArray(organizations.organizations)) throw new Error('We could not load your businesses. Try again.');
	if (!organizations.organizations.length)
		throw new Error('Finish your business setup before changing these settings.');
	const available: Record<string, unknown>[] = organizations.organizations.map(record);
	const selected = selectedOrganization
		? available.find((organization) => text(organization.id) === selectedOrganization)
		: available[0];
	if (!selected) throw new Error('You do not have access to the selected business.');
	const orgID = text(selected.id);
	const organizationName = text(selected.name ?? '') || text(selected.legal_name ?? '') || 'Your business';
	if (!orgID) throw new Error('We could not confirm your business. Try again.');
	const result = await adminGet(`/api/v1/organizations/${encodeURIComponent(orgID)}/onboarding`);
	return { orgID, organizationName, profile: settingsProfile(result.profile), permissions: record(result.permissions) };
}
