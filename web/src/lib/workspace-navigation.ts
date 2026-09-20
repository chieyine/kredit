/** One workspace, five everyday destinations. Specialist tools live inside each area. */
export const workspacePrimary: [string, string, string][] = [
  ['Today', '/workspace/today', 'home'],
  ['Sales', '/workspace/sales', 'sales'],
  ['Purchases', '/workspace/purchases', 'owe'],
  ['Partners', '/workspace/partners', 'customers'],
  ['Money', '/workspace/money', 'payments']
];
export const workspaceMore: [string, string, string][] = [
  ['Find a record', '/workspace/search', 'Your business'],
  ['Reports', '/workspace/reports', 'Your business'],
  ['Team', '/workspace/team', 'Your business'],
  ['Business setup', '/workspace/onboarding', 'Your business'],
  ['Settings', '/workspace/settings', 'Your business'],
  ['Messages', '/account/messages', 'Your account'],
  ['Personal purchases', '/personal/purchases', 'Your account'],
  ['Get help', '/workspace/help', 'Your account']
];
export const workspaceLinks: [string, string][] = [...workspacePrimary, ...workspaceMore].map(([name, href]) => [name, href]);
export const workspaceSections: { root: string; label: string; links: [string, string][] }[] = [
  { root: '/workspace/sales', label: 'Sales tools', links: [['Business sales','/workspace/sales'],['Credit approvals','/workspace/sales/approvals'],['Consumer sales','/workspace/sales/consumers'],['Customer limits','/workspace/sales/limits'],['Overdue','/workspace/overdue'],['Disputes','/workspace/disputes']] },
  { root: '/workspace/purchases', label: 'Purchasing tools', links: [['Overview','/workspace/purchases'],['Team permissions','/workspace/purchases/permissions'],['Staff setup','/workspace/purchases/access'],['Offers','/workspace/purchases/orders'],['Balances','/workspace/purchases/obligations'],['Buying limits','/workspace/purchases/trade-lines'],['Payments','/workspace/purchases/payments'],['Bank permissions','/workspace/purchases/mandates'],['History','/workspace/purchases/history'],['Disputes','/workspace/purchases/disputes'],['Payment changes','/workspace/purchases/amendments'],['Preferences','/account']] },
  { root: '/workspace/partners', label: 'Partner tools', links: [['Overview','/workspace/partners'],['Customers','/workspace/partners/customers'],['Invitations','/workspace/partners/invitations'],['Import contacts','/workspace/partners/import'],['Branches & managers','/workspace/partners/operations'],['Branch access','/workspace/partners/access']] },
  { root: '/workspace/money', label: 'Money tools', links: [['Overview','/workspace/money'],['Received','/workspace/money/received'],['Collections','/workspace/money/collections'],['Receiving account','/workspace/settings/settlement'],['Kredit fees','/workspace/settings/billing']] }
];

/** Scope belongs in the URL so refreshes and shared workspace links stay explicit. */
export function workspaceHref(href: string, current: URL): string {
  if (!href.startsWith('/workspace')) return href;
  const target = new URL(href, current.origin);
  for (const key of ['organization', ...(target.pathname.startsWith('/workspace/purchases') ? ['business_id'] : [])]) {
    if (key === 'business_id' && target.searchParams.has('organization') && target.searchParams.get('organization') !== current.searchParams.get('organization')) continue;
    if (key === 'organization' && target.searchParams.has('business_id') && target.searchParams.get('business_id') !== current.searchParams.get('business_id')) continue;
    const value = current.searchParams.get(key);
    if (value && !target.searchParams.has(key)) target.searchParams.set(key, value);
  }
  return target.pathname + target.search + target.hash;
}
