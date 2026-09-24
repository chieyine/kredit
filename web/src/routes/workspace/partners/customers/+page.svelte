<script lang="ts">
	import type { KoboValue } from '$lib/money';
	import WorkspacePage from '$lib/components/WorkspacePage.svelte';
	type CustomerRow = {
		buyer_business_id: string;
		legal_name?: string;
		trading_name?: string;
		state?: string;
		outstanding_kobo?: KoboValue;
	};
</script>

<WorkspacePage
	eyebrow="Your customers"
	title="Customers"
	description="Every business you sell to on credit. Open one to see the relationship to review its credit sales and payment history."
	organizationPath="/customers"
	collectionKey="customers"
	primaryLabel="Add a customer"
	primaryHref="/workspace/partners/customers/new"
	emptyTitle="No customers yet"
	emptyCopy="Add a customer before you record a sale."
	searchPlaceholder="Customer name"
	rowTitle={(c: CustomerRow) => c.trading_name || c.legal_name || 'Customer'}
	rowDetail={(c) => (c.trading_name && c.legal_name && c.trading_name !== c.legal_name ? c.legal_name : '')}
	rowStatus={(c) => c.state ?? ''}
	rowAmount={(c) => c.outstanding_kobo ?? null}
	rowAmountLabel="Owed to you"
	rowHref={(c, organization) =>
		`/workspace/partners/customers/${encodeURIComponent(c.buyer_business_id)}?organization=${encodeURIComponent(organization)}`}
/>
