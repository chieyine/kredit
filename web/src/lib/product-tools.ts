import { browser } from '$app/environment';

export function readLocal<T>(key: string, fallback: T): T {
	if (!browser) return fallback;
	try {
		const value = localStorage.getItem(key);
		return value ? (JSON.parse(value) as T) : fallback;
	} catch {
		return fallback;
	}
}

export function writeLocal(key: string, value: unknown): boolean {
	if (!browser) return false;
	try {
		localStorage.setItem(key, JSON.stringify(value));
		return true;
	} catch {
		return false;
	}
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
