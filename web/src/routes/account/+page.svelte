<script lang="ts">
	import WorkspaceHub from '$lib/components/WorkspaceHub.svelte';
	const groups = [
		{
			title: 'Your account',
			description: 'These settings apply to you whether you buy, sell or use Kredit for personal purchases.',
			links: [
				['Security', '/account/security', 'Manage extra sign-in checks, recovery codes and signed-in devices.'],
				['Notifications', '/account/notifications', 'Choose your channels and quiet hours.'],
				['Supplier reminders', '/account/suppliers', 'Choose which suppliers may send you relationship reminders.'],
				['Messages', '/account/messages', 'Review messages and whether they were delivered.'],
				['Privacy', '/account/privacy', 'Request a copy, correction or deletion of your information.'],
				['Display and data', '/account/display', 'Adjust this device for your connection and reading preferences.'],
				['Recover access', '/recover', 'Use the recovery process if you cannot sign in.']
			]
		}
	] as { title: string; description: string; links: [string, string, string][] }[];
</script>

<WorkspaceHub
	eyebrow="Your Kredit account"
	title="Account settings"
	description="One set of preferences for your Kredit account. Business terms and staff access are managed separately in the workspace."
	{groups}
/>
