import { redirect } from '@sveltejs/kit';

export function load({ params }: { params: { token: string } }) {
	throw redirect(307, `/buyer-invitations/${encodeURIComponent(params.token)}`);
}
