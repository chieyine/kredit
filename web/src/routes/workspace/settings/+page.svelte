<script lang="ts">
	import WorkspaceHub from '$lib/components/WorkspaceHub.svelte';
	const groups = [
		{
			title: 'Your business',
			description: 'Set these once. Only the owner signs in for now.',
			links: [
				['Business details', '/workspace/onboarding', 'Name, address, CAC number and anything still left to do.'],
				['Where your money goes', '/workspace/settings/settlement', 'The bank account your customers’ payments go to.'],
				['Kredit fees', '/workspace/settings/billing', 'What Kredit charges and how you pay it.']
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
