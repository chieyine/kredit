import { build, files, version } from '$service-worker';

const shellCache = `kredit-shell-${version}`;
const staticAssets = new Set([...build, ...files]);

self.addEventListener('install', (event) => {
	// Activate immediately without downloading every route in the application.
	// Each asset is cached only after the visitor actually needs it.
	self.skipWaiting();
});

self.addEventListener('activate', (event) => {
	event.waitUntil(
		caches.keys().then((keys) =>
			Promise.all(keys.filter((key) => key.startsWith('kredit-shell-') && key !== shellCache).map((key) => caches.delete(key)))
		)
	);
	self.clients.claim();
});

self.addEventListener('fetch', (event) => {
	const request = event.request;
	const url = new URL(request.url);
	if (request.method !== 'GET' || url.origin !== self.location.origin) return;
	// Financial, identity, invitation, document, and admin responses must never
	// be cached or replayed while offline.
	if (url.pathname.startsWith('/api/') || url.pathname.startsWith('/buyer-invitations/') || url.pathname.startsWith('/admin/') || url.pathname.startsWith('/app/')) return;
	if (request.destination === 'document') return;
	if (!staticAssets.has(url.pathname)) return;
	event.respondWith(
		caches.match(request).then(async (cached) => {
			if (cached) return cached;
			const response = await fetch(request);
			if (response.ok) {
				const cache = await caches.open(shellCache);
				await cache.put(request, response.clone());
			}
			return response;
		})
	);
});
