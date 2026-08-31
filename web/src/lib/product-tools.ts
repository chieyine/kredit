import { browser } from '$app/environment';

export type SavedSaleItem = { name: string; amount: string; usedAt: string };

export function readLocal<T>(key: string, fallback: T): T {
	if (!browser) return fallback;
	try {
		const value = localStorage.getItem(key);
		return value ? (JSON.parse(value) as T) : fallback;
	} catch {
		return fallback;
	}
}

export function writeLocal(key: string, value: unknown) {
	if (!browser) return;
	try { localStorage.setItem(key, JSON.stringify(value)); } catch { /* Storage is optional. */ }
}

export function removeLocal(key: string) {
	if (!browser) return;
	try { localStorage.removeItem(key); } catch { /* Storage is optional. */ }
}

export function rememberSaleItem(name: string, amount: string) {
	const cleanName = name.trim();
	const cleanAmount = amount.trim();
	if (!cleanName || !cleanAmount) return;
	const current = readLocal<SavedSaleItem[]>('kredit:saved-sale-items', []);
	const next = [{ name: cleanName, amount: cleanAmount, usedAt: new Date().toISOString() }, ...current.filter((item) => item.name.toLowerCase() !== cleanName.toLowerCase())].slice(0, 12);
	writeLocal('kredit:saved-sale-items', next);
}

export function whatsappURL(message: string) {
	return `https://wa.me/?text=${encodeURIComponent(message.trim())}`;
}

export async function shareOrCopy(title: string, text: string, url = '') {
	const shareData = { title, text, ...(url ? { url } : {}) };
	if (browser && navigator.share) {
		await navigator.share(shareData);
		return 'shared';
	}
	const combined = [text, url].filter(Boolean).join('\n');
	if (browser && navigator.clipboard) {
		await navigator.clipboard.writeText(combined);
		return 'copied';
	}
	return 'unavailable';
}

export function setLowData(enabled: boolean) {
	if (!browser) return;
	document.documentElement.classList.toggle('low-data', enabled);
	writeLocal('kredit:low-data', enabled);
}

export function applySavedDisplayChoice() {
	setLowData(readLocal('kredit:low-data', false));
}
