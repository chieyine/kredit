<script lang="ts">
	import WorkspaceHub from '$lib/components/WorkspaceHub.svelte';
	const groups = [
		{
			title: 'Your business',
			description: 'Your business details and payment settings.',
			links: [
				['Business details', '/workspace/onboarding', 'Name, address, CAC number and anything still left to do.'],
				['Where your money goes', '/workspace/settings/settlement', 'The bank account your customers’ payments go to.'],
				['Kredit fees', '/workspace/settings/billing', 'What Kredit charges and how you pay it.'],
				['Default sale terms', '/workspace/settings/credit-policy', 'Your usual payment date and grace period.'],
				['Existing customer limits', '/workspace/sales/limits', 'View earlier credit limits and finish their sales.'],
				['Purchasing setup', '/workspace/purchases', 'Identity checks and bank setup for buying from suppliers.'],
				[
					'Personal sales',
					'/workspace/sales/consumers',
					'Connect your receiving account and review individual purchases.'
				]
			]
		},
		{
			title: 'You',
			description: 'Your sign-in, your phone number and the messages we send you.',
			links: [['My account', '/account', 'Sign-in, security, messages and privacy.']]
		}
	] as { title: string; description: string; links: [string, string, string][] }[];
</script>

<WorkspaceHub title="Settings" description="" eyebrow="" {groups} />
