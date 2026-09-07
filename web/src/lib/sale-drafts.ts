export type SaleDraft = { goods: string; principal: string; dueDate: string };
// v2 could have been saved without explicit consent. Never restore it as an opt-in.
const PREFIX = 'kredit.quick-sale.v3:';
const TTL = 12 * 60 * 60 * 1000;
function key(userID: string, organizationID: string) {
  if (!userID || !organizationID) throw new Error('An authenticated business is required for draft storage.');
  return `${PREFIX}${encodeURIComponent(userID)}:${encodeURIComponent(organizationID)}`;
}
export function readDraft(userID: string, organizationID: string, store: Storage, now = Date.now()): SaleDraft | null {
  try {
    store.removeItem('kredit.quick-sale.draft.v1');
    const storageKey = key(userID, organizationID);
    store.removeItem(`kredit.quick-sale.v2:${encodeURIComponent(userID)}:${encodeURIComponent(organizationID)}`);
    const raw = store.getItem(storageKey);
    if (!raw) return null;
    const value = JSON.parse(raw);
    if (value.userID !== userID || value.organizationID !== organizationID || !Number.isFinite(value.expiresAt) || value.expiresAt <= now || value.expiresAt > now + TTL || typeof value.goods !== 'string' || value.goods.length > 5000 || typeof value.principal !== 'string' || value.principal.length > 40 || typeof value.dueDate !== 'string' || (value.dueDate && !/^\d{4}-\d{2}-\d{2}$/.test(value.dueDate))) { store.removeItem(storageKey); return null; }
    return { goods: value.goods, principal: value.principal, dueDate: value.dueDate };
  } catch { return null; }
}
export function saveDraft(userID: string, organizationID: string, draft: SaleDraft, store: Storage, now = Date.now()): boolean {
  try { store.setItem(key(userID, organizationID), JSON.stringify({ goods: draft.goods, principal: draft.principal, dueDate: draft.dueDate, userID, organizationID, expiresAt: now + TTL })); return true; }
  catch { return false; }
}
export function deleteDraft(userID: string, organizationID: string, store: Storage): void {
  try { store.removeItem(key(userID, organizationID)); } catch { /* The form still works without draft storage. */ }
}
