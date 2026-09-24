<script lang="ts">
	import WorkspacePage from '$lib/components/WorkspacePage.svelte';
	import { readableDate } from '$lib/datetime';
	type Invitation = {
		name: string;
		state: string;
		source_reference?: string;
		trades?: number;
		zero_balance_obligations?: number;
		expires_at?: string;
	};
</script>

<WorkspacePage
	eyebrow="Business invitations"
	title="Invitation progress"
	description="Follow the latest 500 invitations for this business. Joining Kredit, creating a trade and reducing a balance are separate milestones."
	organizationPath="/distributor-invitations"
	collectionKey="invitations"
	primaryLabel="Import contacts"
	primaryHref="/workspace/partners/import"
	emptyTitle="No invitations yet"
	emptyCopy="Import your roster or invite your first customer to begin."
	rowTitle={(item: Invitation) => item.name}
	rowStatus={(item) => item.state}
	rowDetail={(item) =>
		`${item.source_reference || 'Individual invitation'} · ${item.trades} trade records · ${item.zero_balance_obligations} obligations with no principal outstanding · Invitation expires ${readableDate(item.expires_at)}`}
/>
