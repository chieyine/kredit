import { SITE_URL } from '$lib/seo';
import { loadLegalConfig } from '$lib/server/legal-config';

const privatePaths = ['/app', '/buyer', '/admin', '/c', '/pay', '/receipt', '/secure', '/recover', '/buyer-invitations'];

export function GET() {
	const legalActive = loadLegalConfig().active;
	const disallowed = legalActive ? privatePaths : [...privatePaths, '/legal/privacy', '/legal/terms'];
	const body = [
		'User-agent: *',
		'Allow: /',
		...disallowed.map((path) => `Disallow: ${path}`),
		`Sitemap: ${SITE_URL}/sitemap.xml`,
		''
	].join('\n');
	return new Response(body, {
		headers: { 'Content-Type': 'text/plain; charset=utf-8', 'Cache-Control': 'public, max-age=3600' }
	});
}
