<script lang="ts">
	import WorkspaceHub from '$lib/components/WorkspaceHub.svelte';
	const groups = [
		{
			title: 'Money coming in',
			description: 'Review customer payments and collection attempts before treating them as settled funds.',
			links: [
				[
					'Payments received',
					'/workspace/money/received',
					'Check confirmed payments and open the sale they belong to.'
				],
				[
					'Bank collections',
					'/workspace/money/collections',
					'Authorised attempts, the ones that failed, and anything still waiting on confirmation.'
				],
				['Overdue sales', '/workspace/overdue', 'Follow balances that have passed their agreed payment dates.']
			]
		},
		{
			title: 'Money going out',
			description: 'Supplier obligations remain separate from customer receivables.',
			links: [
				[
					'Supplier balances',
					'/workspace/purchases/obligations',
					'Open what your business owes and review the payment schedule.'
				],
				[
					'Reported supplier payments',
					'/workspace/purchases/payments',
					'Follow transfers you reported and whether the supplier confirmed them.'
				],
				['Kredit fees', '/workspace/settings/billing', 'Review platform fees and how your business pays them.']
			]
		},
		{
			title: 'Reconcile and configure',
			description: 'Keep the receiving account and your financial records up to date.',
			links: [
				[
					'Receiving account',
					'/workspace/settings/settlement',
					'Review the bank account used for your business’s receipts.'
				],
				['Reports', '/workspace/reports', 'Inspect statements and financial summaries for the selected business.'],
				['Business activity', '/workspace/activity', 'Trace important changes and who made them.']
			]
		}
	] as { title: string; description: string; links: [string, string, string][] }[];
</script>

<WorkspaceHub
	title="Money"
	description="Keep incoming payments, supplier obligations and platform fees clear. A reported payment is not a confirmed payment."
	{groups}
/>
