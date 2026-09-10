import type { PageServerLoad } from './$types';
export const load: PageServerLoad = ({ cookies }) => ({ hasSessionCookie: Boolean(cookies.get('kredit_session')) });
