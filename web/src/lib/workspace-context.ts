import { goto } from '$app/navigation';

/** Changing business starts a fresh page; in-flight reads from the previous page are discarded. */
export function chooseWorkspace(organizationID: string) {
  const url = new URL(location.href);
  url.searchParams.set('organization', organizationID);
  url.searchParams.delete('business_id');
  return goto(url.pathname + url.search, { noScroll: true });
}
export function requestedWorkspace<T extends {id:string}>(items: T[]): string {
  const requested = new URLSearchParams(location.search).get('organization');
  if (requested && !items.some(item => item.id === requested)) throw new Error('This business is not available in your account.');
  return requested || items[0]?.id || '';
}
