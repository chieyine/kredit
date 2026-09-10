<script lang="ts">
	import WorkspacePage from '$lib/components/WorkspacePage.svelte';
</script>

<WorkspacePage
	eyebrow="Your customers"
	title="Customers"
	description="Everyone who takes goods from you and pays later. Open one to see what they owe and how they have paid."
	organizationPath="/customers"
	collectionKey="customers"
	primaryLabel="Add a customer"
	primaryHref="/app/customers/new"
	emptyTitle="No customers yet"
	emptyCopy="Add a customer before you record a sale."
	searchPlaceholder="Customer name"
	rowTitle={(c) => c.trading_name || c.legal_name || 'Customer'}
	rowDetail={(c) =>
		c.trading_name && c.legal_name && c.trading_name !== c.legal_name ? c.legal_name : ''}
	rowStatus={(c) => c.state ?? ''}
	rowAmount={(c) => c.outstanding_kobo ?? null}
	rowAmountLabel="Owed to you"
	rowHref={(c, organization) =>
		`/app/customers/${encodeURIComponent(c.buyer_user_id)}?organization=${encodeURIComponent(organization)}`}
/>
