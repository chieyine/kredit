import { adminGet } from '$lib/admin-client';
import { record, text } from './reliable';

export function settingsProfile(value: unknown) {
  const profile = record(value);
  if (!Number.isSafeInteger(profile.version) || Number(profile.version) < 0) throw new Error('We could not confirm the current settings. Refresh this page before saving.');
  return profile;
}

export async function loadOnboardingSettings(selectedOrganization = new URLSearchParams(window.location.search).get('organization')) {
  const organizations = await adminGet('/api/v1/organizations');
  if (!Array.isArray(organizations.organizations)) throw new Error('We could not load your businesses. Try again.');
  if (!organizations.organizations.length) throw new Error('Finish your business setup before changing these settings.');
  const available: Record<string, unknown>[] = organizations.organizations.map(record);
  const selected = selectedOrganization ? available.find((organization) => text(organization.id) === selectedOrganization) : available[0];
  if (!selected) throw new Error('You do not have access to the selected business.');
  const orgID = text(selected.id);
  const organizationName = text(selected.name) || text(selected.legal_name) || 'Your business';
  if (!orgID) throw new Error('We could not confirm your business. Try again.');
  const result = await adminGet(`/api/v1/organizations/${encodeURIComponent(orgID)}/onboarding`);
  return { orgID, organizationName, profile: settingsProfile(result.profile), permissions: record(result.permissions) };
}
