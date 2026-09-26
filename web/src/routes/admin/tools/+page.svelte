<script lang="ts">
	import WorkspaceHub from '$lib/components/WorkspaceHub.svelte';
	// Every admin tool still works; these are the ones you open now and then, not daily.
	const groups = [
		{
			title: 'Jobs, providers and messages',
			description: 'Background work and what outside services sent back.',
			links: [
				['Background jobs', '/admin/jobs', 'Work in progress, retries and stopped jobs.'],
				['Provider events', '/admin/provider-events', 'What the payment and identity providers reported.'],
				['Provider work', '/admin/provider-work', 'Requests waiting on a provider.'],
				['Message recovery', '/admin/message-submissions', 'Messages that did not go out.'],
				['Mono integration', '/admin/mono', 'Bank-debit provider setup and status.'],
				['System diagnostics', '/admin/diagnostics', 'Health of the running system.']
			]
		},
		{
			title: 'Recovery',
			description: 'Put right a registration, bank permission or account that got stuck.',
			links: [
				['Mandate recovery', '/admin/mandate-authorizations', 'Bank permissions that did not finish.'],
				['Account recovery', '/admin/recovery', 'People who cannot sign in.'],
				['Privacy requests', '/admin/privacy', 'Copy, correction and deletion requests.']
			]
		},
		{
			title: 'Records and controls',
			description: 'History, evidence and the switches that protect money.',
			links: [
				['Change history', '/admin/history', 'Who changed what, and when.'],
				['Audit trail', '/admin/audit', 'The full record of sensitive actions.'],
				['Find a reference', '/admin/search', 'Look up any record by its reference.'],
				['Analytics evidence', '/admin/analytics', 'Product numbers and where they come from.'],
				['Fee billing', '/admin/billing', 'Kredit fee invoices and receipts.'],
				['Protected controls', '/admin/controls', 'Switches that stop or allow money movement.'],
				['Business settings', '/admin/settings', 'Rules applied to every business.'],
				['Launch setup', '/admin/setup', 'Checks before going live.'],
				['Website content', '/admin/website', 'Edit the public website text.'],
				['Field agents', '/admin/agents', 'Introductions programme (switched off for now).']
			]
		}
	] as { title: string; description: string; links: [string, string, string][] }[];
</script>

<WorkspaceHub eyebrow="Admin" title="Engineering tools" description="" {groups} />
